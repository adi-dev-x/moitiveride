package domain

import "time"

type RouteDirection string
type RouteStatus string

const (
	DirectionPickup RouteDirection = "PICKUP"
	DirectionDrop   RouteDirection = "DROP"

	RouteStatusPlanned   RouteStatus = "PLANNED"
	RouteStatusActive    RouteStatus = "ACTIVE"
	RouteStatusCompleted RouteStatus = "COMPLETED"
	RouteStatusCancelled RouteStatus = "CANCELLED"
)

type Route struct {
	ID              string         `db:"id" json:"id"`
	VehicleID       string         `db:"vehicle_id" json:"vehicle_id"`
	ShiftID         string         `db:"shift_id" json:"shift_id"`
	CompanyID       string         `db:"company_id" json:"company_id"`
	RouteDate       time.Time      `db:"route_date" json:"route_date"`
	Direction       RouteDirection `db:"direction" json:"direction"`
	Status          RouteStatus    `db:"status" json:"status"`
	TotalDistanceKm float64        `db:"total_distance_km" json:"total_distance_km"`
	TotalDurationMin int           `db:"total_duration_min" json:"total_duration_min"`
	EscortRequired  bool           `db:"escort_required" json:"escort_required"`
	EscortID        string         `db:"escort_id" json:"escort_id,omitempty"`
	CreatedAt       time.Time      `db:"created_at" json:"created_at"`
	UpdatedAt       time.Time      `db:"updated_at" json:"updated_at"`

	// Joined
	Stops   []RouteStop `db:"-" json:"stops,omitempty"`
	Vehicle *Vehicle    `db:"-" json:"vehicle,omitempty"`
}

type RouteStop struct {
	ID                string     `db:"id" json:"id"`
	RouteID           string     `db:"route_id" json:"route_id"`
	EmployeeID        string     `db:"employee_id" json:"employee_id"`
	SequenceOrder     int        `db:"sequence_order" json:"sequence_order"`
	StopLat           float64    `db:"stop_lat" json:"stop_lat"`
	StopLng           float64    `db:"stop_lng" json:"stop_lng"`
	EstimatedArrival  *time.Time `db:"estimated_arrival" json:"estimated_arrival"`
	ActualArrival     *time.Time `db:"actual_arrival" json:"actual_arrival,omitempty"`
	Boarded           bool       `db:"boarded" json:"boarded"`

	// Joined
	Employee *Employee `db:"-" json:"employee,omitempty"`
}

// RouteManifest is the denormalized output document sent to drivers / admin.
type RouteManifest struct {
	Route     Route       `json:"route"`
	Vehicle   Vehicle     `json:"vehicle"`
	Stops     []RouteStop `json:"stops"`
	Employees []Employee  `json:"employees"`
}
