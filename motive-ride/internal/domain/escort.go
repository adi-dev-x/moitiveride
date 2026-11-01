package domain

import "time"

type EscortStatus string

const (
	EscortAvailable EscortStatus = "AVAILABLE"
	EscortAssigned  EscortStatus = "ASSIGNED"
)

type Escort struct {
	ID        string       `db:"id" json:"id"`
	Name      string       `db:"name" json:"name"`
	Phone     string       `db:"phone" json:"phone"`
	Status    EscortStatus `db:"status" json:"status"`
	CreatedAt time.Time    `db:"created_at" json:"created_at"`
}
