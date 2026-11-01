package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/motive-ride/motive-ride/internal/domain"
)

type EmployeeRepository struct {
	db *DB
}

func NewEmployeeRepository(db *DB) *EmployeeRepository {
	return &EmployeeRepository{db: db}
}

// ListByShift returns all active employees for a given company and shift.
func (r *EmployeeRepository) ListByShift(ctx context.Context, companyID, shiftID string) ([]domain.Employee, error) {
	const q = `
		SELECT id, company_id, shift_id, employee_code, name, gender,
		       home_lat, home_lng, boarding_point_id, drop_point_id,
		       pickup_window_start::text, pickup_window_end::text,
		       drop_window_start::text,  drop_window_end::text,
		       active, created_at, updated_at
		FROM employees
		WHERE company_id = $1 AND shift_id = $2 AND active = TRUE
		ORDER BY id`

	rows, err := r.db.Pool.Query(ctx, q, companyID, shiftID)
	if err != nil {
		return nil, fmt.Errorf("list employees by shift: %w", err)
	}
	defer rows.Close()

	return pgx.CollectRows(rows, pgx.RowToStructByName[domain.Employee])
}

// ListActiveByCompany returns all active employees for a company (all shifts).
func (r *EmployeeRepository) ListActiveByCompany(ctx context.Context, companyID string) ([]domain.Employee, error) {
	const q = `
		SELECT id, company_id, shift_id, employee_code, name, gender,
		       home_lat, home_lng, boarding_point_id, drop_point_id,
		       pickup_window_start::text, pickup_window_end::text,
		       drop_window_start::text,  drop_window_end::text,
		       active, created_at, updated_at
		FROM employees
		WHERE company_id = $1 AND active = TRUE
		ORDER BY shift_id, id`

	rows, err := r.db.Pool.Query(ctx, q, companyID)
	if err != nil {
		return nil, fmt.Errorf("list employees by company: %w", err)
	}
	defer rows.Close()

	return pgx.CollectRows(rows, pgx.RowToStructByName[domain.Employee])
}

// GetByID returns a single employee.
func (r *EmployeeRepository) GetByID(ctx context.Context, id string) (*domain.Employee, error) {
	const q = `
		SELECT id, company_id, shift_id, employee_code, name, gender,
		       home_lat, home_lng, boarding_point_id, drop_point_id,
		       pickup_window_start::text, pickup_window_end::text,
		       drop_window_start::text,  drop_window_end::text,
		       active, created_at, updated_at
		FROM employees WHERE id = $1`

	rows, err := r.db.Pool.Query(ctx, q, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	emp, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[domain.Employee])
	if err != nil {
		return nil, fmt.Errorf("get employee %s: %w", id, err)
	}
	return &emp, nil
}

// Create inserts a new employee record.
func (r *EmployeeRepository) Create(ctx context.Context, e *domain.Employee) error {
	const q = `
		INSERT INTO employees
		    (company_id, shift_id, employee_code, name, gender,
		     home_lat, home_lng, boarding_point_id, drop_point_id,
		     pickup_window_start, pickup_window_end,
		     drop_window_start, drop_window_end)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)
		RETURNING id, created_at, updated_at`

	return r.db.Pool.QueryRow(ctx, q,
		e.CompanyID, e.ShiftID, e.EmployeeCode, e.Name, e.Gender,
		e.HomeLat, e.HomeLng, e.BoardingPointID, e.DropPointID,
		e.PickupWindowStart, e.PickupWindowEnd,
		e.DropWindowStart, e.DropWindowEnd,
	).Scan(&e.ID, &e.CreatedAt, &e.UpdatedAt)
}

// BulkCreate inserts multiple employees in a single COPY operation for performance.
func (r *EmployeeRepository) BulkCreate(ctx context.Context, employees []domain.Employee) error {
	_, err := r.db.Pool.CopyFrom(
		ctx,
		pgx.Identifier{"employees"},
		[]string{
			"company_id", "shift_id", "employee_code", "name", "gender",
			"home_lat", "home_lng", "boarding_point_id", "drop_point_id",
			"pickup_window_start", "pickup_window_end",
			"drop_window_start", "drop_window_end",
		},
		pgx.CopyFromSlice(len(employees), func(i int) ([]interface{}, error) {
			e := employees[i]
			return []interface{}{
				e.CompanyID, e.ShiftID, e.EmployeeCode, e.Name, e.Gender,
				e.HomeLat, e.HomeLng, e.BoardingPointID, e.DropPointID,
				e.PickupWindowStart, e.PickupWindowEnd,
				e.DropWindowStart, e.DropWindowEnd,
			}, nil
		}),
	)
	return err
}

// FindNearby returns employees within radiusKm of the given point (uses PostGIS).
func (r *EmployeeRepository) FindNearby(ctx context.Context, lat, lng, radiusKm float64, companyID, shiftID string) ([]domain.Employee, error) {
	const q = `
		SELECT id, company_id, shift_id, employee_code, name, gender,
		       home_lat, home_lng, boarding_point_id, drop_point_id,
		       pickup_window_start::text, pickup_window_end::text,
		       drop_window_start::text,  drop_window_end::text,
		       active, created_at, updated_at
		FROM employees
		WHERE company_id = $1 AND shift_id = $2 AND active = TRUE
		  AND ST_DWithin(
		        home_geom::geography,
		        ST_SetSRID(ST_MakePoint($4, $3), 4326)::geography,
		        $5 * 1000
		      )
		ORDER BY home_geom <-> ST_SetSRID(ST_MakePoint($4, $3), 4326)`

	rows, err := r.db.Pool.Query(ctx, q, companyID, shiftID, lat, lng, radiusKm)
	if err != nil {
		return nil, fmt.Errorf("find nearby employees: %w", err)
	}
	defer rows.Close()

	return pgx.CollectRows(rows, pgx.RowToStructByName[domain.Employee])
}
