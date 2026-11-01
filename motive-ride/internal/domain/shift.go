package domain

import "time"

type ShiftType string

const (
	ShiftMorning   ShiftType = "MORNING"
	ShiftAfternoon ShiftType = "AFTERNOON"
	ShiftEvening   ShiftType = "EVENING"
	ShiftNight     ShiftType = "NIGHT"
)

type Shift struct {
	ID        string    `db:"id" json:"id"`
	CompanyID string    `db:"company_id" json:"company_id"`
	Name      string    `db:"name" json:"name"`
	StartTime string    `db:"start_time" json:"start_time"` // HH:MM
	EndTime   string    `db:"end_time" json:"end_time"`     // HH:MM
	ShiftType ShiftType `db:"shift_type" json:"shift_type"`
	Active    bool      `db:"active" json:"active"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}

func (s Shift) IsNight() bool {
	return s.ShiftType == ShiftNight
}
