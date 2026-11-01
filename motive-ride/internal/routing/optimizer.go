package routing

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/motive-ride/motive-ride/internal/domain"
)

const maxRideTimeMin = 90.0

// Optimizer computes the optimal pickup/drop sequence for one vehicle cluster.
// It uses a Nearest Neighbour greedy heuristic as a warm start (fast, ~60% of
// optimal). The time matrix is supplied by a TimeMatrixProvider so the same
// optimizer works with offline Haversine distances or live Google Maps data.
type Optimizer struct {
	traffic TimeMatrixProvider
	offline *OfflineProvider // used for per-leg time after matrix is built
}

func NewOptimizer(traffic TimeMatrixProvider) *Optimizer {
	return &Optimizer{
		traffic: traffic,
		offline: NewOfflineProvider(),
	}
}

// Optimize determines the best stop sequence for an assigned cluster.
// departure is when the vehicle leaves the depot (office / first pickup time).
func (o *Optimizer) Optimize(
	ctx context.Context,
	ac AssignedCluster,
	depotLat, depotLng float64,
	departure time.Time,
	direction domain.RouteDirection,
) (*OptimizedRoute, error) {
	employees := ac.Cluster.Employees
	if len(employees) == 0 {
		return nil, fmt.Errorf("empty cluster")
	}

	// Build N×N time matrix using the configured provider (Google Maps or offline).
	// The depot point is appended at the end so all inter-point times are covered.
	points := toPoints(employees)
	depotIdx := len(points)
	points = append(points, Point{Lat: depotLat, Lng: depotLng, Idx: depotIdx})

	timeMatrix, err := o.traffic.BuildTimeMatrix(ctx, points, departure)
	if err != nil {
		return nil, fmt.Errorf("build time matrix: %w", err)
	}

	sequence := o.nearestNeighbour(timeMatrix, depotIdx, len(employees))
	stops, totalDist, totalTime := o.buildStops(sequence, employees, points, departure)

	if err := o.validateRideTime(stops, departure); err != nil {
		return nil, err
	}

	return &OptimizedRoute{
		AssignedCluster:  ac,
		StopSequence:     stops,
		TotalDistanceKm:  totalDist,
		TotalDurationMin: int(math.Round(totalTime)),
	}, nil
}

// nearestNeighbour builds a greedy sequence starting from the depot.
// It always picks the next stop with the minimum travel time from the current position.
// Using the time matrix (not raw distance) means traffic-weighted road times drive
// ordering, not straight-line kilometres.
func (o *Optimizer) nearestNeighbour(timeMatrix [][]float64, depotIdx, numEmployees int) []int {
	visited := make([]bool, numEmployees)
	sequence := make([]int, 0, numEmployees)
	current := depotIdx

	for len(sequence) < numEmployees {
		bestIdx := -1
		bestTime := math.MaxFloat64

		for i := 0; i < numEmployees; i++ {
			if visited[i] {
				continue
			}
			t := timeMatrix[current][i]
			if t < bestTime {
				bestTime = t
				bestIdx = i
			}
		}

		visited[bestIdx] = true
		sequence = append(sequence, bestIdx)
		current = bestIdx
	}

	return sequence
}

func (o *Optimizer) buildStops(
	sequence []int,
	employees []domain.Employee,
	points []Point,
	departure time.Time,
) ([]RouteStop, float64, float64) {
	depotPoint := points[len(points)-1]
	stops := make([]RouteStop, 0, len(sequence))
	totalDist := 0.0
	totalTime := 0.0
	current := departure
	prevLat, prevLng := depotPoint.Lat, depotPoint.Lng

	for i, idx := range sequence {
		emp := employees[idx]
		dist := haversineKm(prevLat, prevLng, emp.HomeLat, emp.HomeLng)
		// Use the offline provider for per-leg time so buildStops never fails.
		// The ordering was already decided by the traffic-accurate time matrix.
		travelMin := o.offline.TravelTimeMin(dist, current)

		current = current.Add(time.Duration(travelMin*60) * time.Second)
		totalDist += dist
		totalTime += travelMin

		stops = append(stops, RouteStop{
			Employee:         emp,
			EstimatedArrival: current,
			SequenceOrder:    i + 1,
		})

		const dwellMin = 2
		current = current.Add(dwellMin * time.Minute)
		totalTime += dwellMin
		prevLat, prevLng = emp.HomeLat, emp.HomeLng
	}

	return stops, totalDist, totalTime
}

func (o *Optimizer) validateRideTime(stops []RouteStop, departure time.Time) error {
	if len(stops) == 0 {
		return nil
	}
	firstPickup := stops[0].EstimatedArrival
	lastStop := stops[len(stops)-1].EstimatedArrival
	totalRide := lastStop.Sub(firstPickup).Minutes()

	if totalRide > maxRideTimeMin {
		return fmt.Errorf("route exceeds max ride time: %.0f min (limit %d)", totalRide, int(maxRideTimeMin))
	}
	return nil
}
