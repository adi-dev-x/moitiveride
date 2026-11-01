package routing

import (
	"context"
	"time"
)

// TimeMatrixProvider builds an N×N travel-time matrix (minutes) for a set of
// geographic points. Implementations can be offline (Haversine + static
// traffic factors) or real (Google Maps Distance Matrix API with Redis cache).
type TimeMatrixProvider interface {
	BuildTimeMatrix(ctx context.Context, points []Point, departure time.Time) ([][]float64, error)
}

// ────────────────────────────────────────────────────────────────────────────
// Offline implementation
// Used when no Google Maps API key is configured, or as fallback on API error.
// ────────────────────────────────────────────────────────────────────────────

// trafficFactors maps hour-of-day → speed multiplier modelling Bangalore patterns.
var trafficFactors = map[int]float64{
	0: 0.95, 1: 0.95, 2: 0.95, 3: 0.95, 4: 0.95, 5: 0.90,
	6: 0.70, 7: 0.55, 8: 0.55, 9: 0.60, // morning peak
	10: 0.80, 11: 0.85, 12: 0.85, 13: 0.90,
	14: 0.88, 15: 0.85, 16: 0.75,
	17: 0.55, 18: 0.50, 19: 0.55, 20: 0.70, // evening peak
	21: 0.80, 22: 0.85, 23: 0.90,
}

const avgSpeedKmh = 40.0

// OfflineProvider satisfies TimeMatrixProvider using Haversine distance and
// static Bangalore traffic factors. Zero external dependencies.
type OfflineProvider struct{}

func NewOfflineProvider() *OfflineProvider { return &OfflineProvider{} }

func (o *OfflineProvider) BuildTimeMatrix(ctx context.Context, points []Point, departure time.Time) ([][]float64, error) {
	n := len(points)
	m := make([][]float64, n)
	for i := range m {
		m[i] = make([]float64, n)
		for j := range m[i] {
			if i != j {
				distKm := haversineKm(points[i].Lat, points[i].Lng, points[j].Lat, points[j].Lng)
				m[i][j] = o.travelTimeMin(distKm, departure)
			}
		}
	}
	return m, nil
}

func (o *OfflineProvider) travelTimeMin(distanceKm float64, departure time.Time) float64 {
	factor, ok := trafficFactors[departure.Hour()]
	if !ok {
		factor = 0.85
	}
	return (distanceKm / (avgSpeedKmh * factor)) * 60
}

// TravelTimeMin is exposed so the optimizer can compute per-leg times
// without rebuilding the full matrix.
func (o *OfflineProvider) TravelTimeMin(distanceKm float64, departure time.Time) float64 {
	return o.travelTimeMin(distanceKm, departure)
}
