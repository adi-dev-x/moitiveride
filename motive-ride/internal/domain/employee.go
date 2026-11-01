package domain

import "time"

type Gender string

const (
	GenderMale   Gender = "MALE"
	GenderFemale Gender = "FEMALE"
	GenderOther  Gender = "OTHER"
)

type Employee struct {
	ID                 string    `db:"id" json:"id"`
	CompanyID          string    `db:"company_id" json:"company_id"`
	ShiftID            string    `db:"shift_id" json:"shift_id"`
	EmployeeCode       string    `db:"employee_code" json:"employee_code"`
	Name               string    `db:"name" json:"name"`
	Gender             Gender    `db:"gender" json:"gender"`
	HomeLat            float64   `db:"home_lat" json:"home_lat"`
	HomeLng            float64   `db:"home_lng" json:"home_lng"`
	BoardingPointID    string    `db:"boarding_point_id" json:"boarding_point_id"`
	DropPointID        string    `db:"drop_point_id" json:"drop_point_id"`
	PickupWindowStart  string    `db:"pickup_window_start" json:"pickup_window_start"` // HH:MM
	PickupWindowEnd    string    `db:"pickup_window_end" json:"pickup_window_end"`     // HH:MM
	DropWindowStart    string    `db:"drop_window_start" json:"drop_window_start"`
	DropWindowEnd      string    `db:"drop_window_end" json:"drop_window_end"`
	Active             bool      `db:"active" json:"active"`
	CreatedAt          time.Time `db:"created_at" json:"created_at"`
	UpdatedAt          time.Time `db:"updated_at" json:"updated_at"`

	// Joined fields (not stored)
	Company *Company `db:"-" json:"company,omitempty"`
	Shift   *Shift   `db:"-" json:"shift,omitempty"`
}

func (e Employee) IsFemale() bool {
	return e.Gender == GenderFemale
}

// PartitionKey returns the key used to group employees into independent routing sub-problems.
func (e Employee) PartitionKey() string {
	return e.CompanyID + ":" + e.ShiftID
}
