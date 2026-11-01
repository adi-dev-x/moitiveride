package maps

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/motive-ride/motive-ride/internal/cache"
	"github.com/motive-ride/motive-ride/internal/routing"
)

const (
	distanceMatrixURL = "https://maps.googleapis.com/maps/api/distancematrix/json"
	cachePrefix       = "timematrix:v1:"
	cacheTTL          = 6 * time.Hour
	// Google Distance Matrix API limit per request
	maxPointsPerBatch = 25
)

// GoogleProvider calls the Google Maps Distance Matrix API and caches results
// in Redis. Falls back to the offline Haversine provider on any API error so
// routing is never blocked by a Maps outage.
type GoogleProvider struct {
	apiKey   string
	cache    *cache.Client
	offline  *routing.OfflineProvider
	httpCli  *http.Client
}

func NewGoogleProvider(apiKey string, cache *cache.Client) *GoogleProvider {
	return &GoogleProvider{
		apiKey:  apiKey,
		cache:   cache,
		offline: routing.NewOfflineProvider(),
		httpCli: &http.Client{Timeout: 10 * time.Second},
	}
}

// BuildTimeMatrix returns an N×N matrix of travel times in minutes.
// Cache hit  → return immediately (no API call).
// Cache miss → call Google Maps API → cache the result → return.
// API error  → fall back to offline Haversine + log warning.
func (g *GoogleProvider) BuildTimeMatrix(ctx context.Context, points []routing.Point, departure time.Time) ([][]float64, error) {
	cacheKey := buildCacheKey(points, departure)

	// ── Cache hit ─────────────────────────────────────────────────────────
	var cached [][]float64
	hit, err := g.cache.GetJSON(ctx, cacheKey, &cached)
	if err != nil {
		slog.Warn("cache get failed, proceeding to API", "error", err)
	}
	if hit {
		slog.Debug("time matrix cache hit", "key", cacheKey)
		return cached, nil
	}

	// ── Google Maps API ───────────────────────────────────────────────────
	matrix, err := g.fetchFromAPI(ctx, points, departure)
	if err != nil {
		slog.Warn("google maps API failed, using offline fallback",
			"error", err,
			"points", len(points),
		)
		return g.offline.BuildTimeMatrix(ctx, points, departure)
	}

	// ── Cache the result ──────────────────────────────────────────────────
	if err := g.cache.SetJSON(ctx, cacheKey, matrix, cacheTTL); err != nil {
		slog.Warn("cache set failed", "error", err) // non-fatal
	}

	return matrix, nil
}

// fetchFromAPI calls the Distance Matrix API in batches if N > maxPointsPerBatch.
// Each batch covers one "row block" of the matrix.
func (g *GoogleProvider) fetchFromAPI(ctx context.Context, points []routing.Point, departure time.Time) ([][]float64, error) {
	n := len(points)
	matrix := make([][]float64, n)
	for i := range matrix {
		matrix[i] = make([]float64, n)
	}

	coordStrings := make([]string, n)
	for i, p := range points {
		coordStrings[i] = fmt.Sprintf("%.6f,%.6f", p.Lat, p.Lng)
	}

	// Ensure departure_time is in the future (API requirement for traffic data).
	depTime := departure
	if depTime.Before(time.Now()) {
		depTime = time.Now().Add(5 * time.Minute)
	}

	// Batch: process maxPointsPerBatch origins at a time.
	for originStart := 0; originStart < n; originStart += maxPointsPerBatch {
		originEnd := min(originStart+maxPointsPerBatch, n)
		origins := coordStrings[originStart:originEnd]

		for destStart := 0; destStart < n; destStart += maxPointsPerBatch {
			destEnd := min(destStart+maxPointsPerBatch, n)
			destinations := coordStrings[destStart:destEnd]

			rows, err := g.callAPI(ctx, origins, destinations, depTime)
			if err != nil {
				return nil, err
			}

			// Fill the corresponding block of the matrix.
			for ri, row := range rows {
				for ci, elem := range row.Elements {
					i := originStart + ri
					j := destStart + ci
					if i == j {
						continue
					}
					matrix[i][j] = elem.travelMinutes()
				}
			}
		}
	}

	return matrix, nil
}

// callAPI makes one Distance Matrix API request and returns the rows.
func (g *GoogleProvider) callAPI(
	ctx context.Context,
	origins, destinations []string,
	departure time.Time,
) ([]apiRow, error) {
	params := url.Values{}
	params.Set("origins", strings.Join(origins, "|"))
	params.Set("destinations", strings.Join(destinations, "|"))
	params.Set("departure_time", fmt.Sprintf("%d", departure.Unix()))
	params.Set("traffic_model", "best_guess")
	params.Set("key", g.apiKey)

	reqURL := distanceMatrixURL + "?" + params.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}

	resp, err := g.httpCli.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http do: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}

	var apiResp apiResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	if apiResp.Status != "OK" {
		return nil, fmt.Errorf("maps API status: %s — %s", apiResp.Status, apiResp.ErrorMessage)
	}

	return apiResp.Rows, nil
}

// ── Response types ────────────────────────────────────────────────────────────

type apiResponse struct {
	Status       string   `json:"status"`
	ErrorMessage string   `json:"error_message"`
	Rows         []apiRow `json:"rows"`
}

type apiRow struct {
	Elements []apiElement `json:"elements"`
}

type apiElement struct {
	Status            string      `json:"status"`
	Duration          apiDuration `json:"duration"`
	DurationInTraffic apiDuration `json:"duration_in_traffic"`
	Distance          apiDistance `json:"distance"`
}

type apiDuration struct {
	Value int `json:"value"` // seconds
}

type apiDistance struct {
	Value int `json:"value"` // metres
}

// travelMinutes returns the best available travel time in minutes.
// Prefers duration_in_traffic (live traffic) over duration (free-flow).
// Falls back to 0 if the element has a non-OK status (e.g. ZERO_RESULTS).
func (e apiElement) travelMinutes() float64 {
	if e.Status != "OK" {
		return 0
	}
	if e.DurationInTraffic.Value > 0 {
		return float64(e.DurationInTraffic.Value) / 60
	}
	return float64(e.Duration.Value) / 60
}

// ── Cache key ─────────────────────────────────────────────────────────────────

// buildCacheKey produces a stable cache key from the set of coordinates and
// the departure hour. The key is independent of point order so two route
// planning runs with the same employees (different sequence) share the cache.
func buildCacheKey(points []routing.Point, departure time.Time) string {
	coords := make([]string, len(points))
	for i, p := range points {
		coords[i] = fmt.Sprintf("%.5f,%.5f", p.Lat, p.Lng)
	}
	sort.Strings(coords) // order-independent

	raw := fmt.Sprintf("%s|hour=%d", strings.Join(coords, "|"), departure.Hour())
	hash := sha256.Sum256([]byte(raw))
	return fmt.Sprintf("%s%x", cachePrefix, hash[:8]) // 16-char hex suffix is enough
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
