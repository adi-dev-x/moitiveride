package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/motive-ride/motive-ride/internal/domain"
)

type VehicleRepository struct {
	db *DB
}

func NewVehicleRepository(db *DB) *VehicleRepository {
	return &VehicleRepository{db: db}
}

// ListAvailable returns all vehicles with AVAILABLE status, sorted by capacity ascending.
func (r *VehicleRepository) ListAvailable(ctx context.Context) ([]domain.Vehicle, error) {
	const q = `
		SELECT id, registration, vehicle_type, capacity,
		       driver_id, status, current_lat, current_lng, created_at, updated_at
		FROM vehicles
		WHERE status = 'AVAILABLE'
		ORDER BY capacity ASC`

	rows, err := r.db.Pool.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("list available vehicles: %w", err)
	}
	defer rows.Close()

	return pgx.CollectRows(rows, pgx.RowToStructByName[domain.Vehicle])
}

// GetByID returns a single vehicle.
func (r *VehicleRepository) GetByID(ctx context.Context, id string) (*domain.Vehicle, error) {
	const q = `
		SELECT id, registration, vehicle_type, capacity,
		       driver_id, status, current_lat, current_lng, created_at, updated_at
		FROM vehicles WHERE id = $1`

	rows, err := r.db.Pool.Query(ctx, q, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	v, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[domain.Vehicle])
	if err != nil {
		return nil, fmt.Errorf("get vehicle %s: %w", id, err)
	}
	return &v, nil
}

// Create inserts a new vehicle.
func (r *VehicleRepository) Create(ctx context.Context, v *domain.Vehicle) error {
	const q = `
		INSERT INTO vehicles (registration, vehicle_type, capacity, driver_id, status)
		VALUES ($1,$2,$3,$4,'AVAILABLE')
		RETURNING id, created_at, updated_at`

	return r.db.Pool.QueryRow(ctx, q,
		v.Registration, v.VehicleType, v.Capacity, v.DriverID,
	).Scan(&v.ID, &v.CreatedAt, &v.UpdatedAt)
}

// UpdateStatus changes vehicle status (e.g. AVAILABLE → ON_ROUTE).
func (r *VehicleRepository) UpdateStatus(ctx context.Context, id string, status domain.VehicleStatus) error {
	const q = `UPDATE vehicles SET status = $1, updated_at = NOW() WHERE id = $2`
	_, err := r.db.Pool.Exec(ctx, q, status, id)
	return err
}

// SmallestFit returns the available vehicle with the smallest capacity >= minCapacity.
func (r *VehicleRepository) SmallestFit(ctx context.Context, minCapacity int) (*domain.Vehicle, error) {
	const q = `
		SELECT id, registration, vehicle_type, capacity,
		       driver_id, status, current_lat, current_lng, created_at, updated_at
		FROM vehicles
		WHERE status = 'AVAILABLE' AND capacity >= $1
		ORDER BY capacity ASC
		LIMIT 1`

	rows, err := r.db.Pool.Query(ctx, q, minCapacity)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	v, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[domain.Vehicle])
	if err != nil {
		return nil, fmt.Errorf("no vehicle with capacity >= %d: %w", minCapacity, err)
	}
	return &v, nil
}
