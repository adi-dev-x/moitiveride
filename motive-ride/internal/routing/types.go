package routing

import (
	"time"

	"github.com/motive-ride/motive-ride/internal/domain"
)

// PartitionKey uniquely identifies one independent routing sub-problem.
type PartitionKey struct {
	CompanyID string
	ShiftID   string
}

// Partition holds the input data for one Company × Shift sub-problem.
type Partition struct {
	Key       PartitionKey
	Employees []domain.Employee
	Shift     domain.Shift
}

// Cluster is a group of employees that will share one vehicle.
type Cluster struct {
	Employees []domain.Employee
}

// AssignedCluster pairs a cluster with its assigned vehicle.
type AssignedCluster struct {
	Cluster        Cluster
	Vehicle        domain.Vehicle
	EscortRequired bool
}

// OptimizedRoute is the output of the VRPTW solver for one vehicle.
type OptimizedRoute struct {
	AssignedCluster AssignedCluster
	StopSequence    []RouteStop
	TotalDistanceKm float64
	TotalDurationMin int
}

// RouteStop is an ordered pickup/drop event on a route.
type RouteStop struct {
	Employee         domain.Employee
	EstimatedArrival time.Time
	SequenceOrder    int
}

// GenerateRequest is the input to the routing engine.
type GenerateRequest struct {
	CompanyID  string
	ShiftID    string
	RouteDate  time.Time
	Direction  domain.RouteDirection
	Employees  []domain.Employee
	Vehicles   []domain.Vehicle
	Shift      domain.Shift
}

// GenerateResult is the complete output from one routing run.
type GenerateResult struct {
	Routes    []OptimizedRoute
	Errors    []PartitionError
	DurationMs int64
}

// PartitionError captures a per-partition failure without stopping other partitions.
type PartitionError struct {
	Key   PartitionKey
	Error error
}
