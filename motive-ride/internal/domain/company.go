package domain

import (
	"time"
)

type Company struct {
	ID           string    `db:"id" json:"id"`
	Name         string    `db:"name" json:"name"`
	ContactEmail string    `db:"contact_email" json:"contact_email"`
	Active       bool      `db:"active" json:"active"`
	CreatedAt    time.Time `db:"created_at" json:"created_at"`
	UpdatedAt    time.Time `db:"updated_at" json:"updated_at"`
}
