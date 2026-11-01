package domain

import "time"

type VehicleStatus string

const (
	VehicleAvailable   VehicleStatus = "AVAILABLE"
	VehicleOnRoute     VehicleStatus = "ON_ROUTE"
	VehicleMaintenance VehicleStatus = "MAINTENANCE"
)

type Vehicle struct {
	ID           string        `db:"id" json:"id"`
	Registration string        `db:"registration" json:"registration"`
	VehicleType  string        `db:"vehicle_type" json:"vehicle_type"`
	Capacity     int           `db:"capacity" json:"capacity"`
	DriverID     string        `db:"driver_id" json:"driver_id"`
	Status       VehicleStatus `db:"status" json:"status"`
	CurrentLat   float64       `db:"current_lat" json:"current_lat"`
	CurrentLng   float64       `db:"current_lng" json:"current_lng"`
	CreatedAt    time.Time     `db:"created_at" json:"created_at"`
	UpdatedAt    time.Time     `db:"updated_at" json:"updated_at"`
}

func (v Vehicle) IsAvailable() bool {
	return v.Status == VehicleAvailable
}

// StandardCapacities defines the vehicle sizes in ascending order for best-fit assignment.
var StandardCapacities = []int{4, 6, 7, 12, 17, 20}
