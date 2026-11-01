package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/motive-ride/motive-ride/internal/domain"
)

type RouteRepository struct {
	db *DB
}

func NewRouteRepository(db *DB) *RouteRepository {
	return &RouteRepository{db: db}
}

// CreateRoute inserts a route and all its stops in a single transaction.
func (r *RouteRepository) CreateRoute(ctx context.Context, route *domain.Route, stops []domain.RouteStop) error {
	tx, err := r.db.Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	const routeQ = `
		INSERT INTO routes
		    (vehicle_id, shift_id, company_id, route_date, direction,
		     total_distance_km, total_duration_min, escort_required, escort_id)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
		RETURNING id, created_at, updated_at`

	err = tx.QueryRow(ctx, routeQ,
		route.VehicleID, route.ShiftID, route.CompanyID,
		route.RouteDate, route.Direction,
		route.TotalDistanceKm, route.TotalDurationMin,
		route.EscortRequired, nilIfEmpty(route.EscortID),
	).Scan(&route.ID, &route.CreatedAt, &route.UpdatedAt)
	if err != nil {
		return fmt.Errorf("insert route: %w", err)
	}

	_, err = tx.CopyFrom(
		ctx,
		pgx.Identifier{"route_stops"},
		[]string{"route_id", "employee_id", "sequence_order", "stop_lat", "stop_lng", "estimated_arrival"},
		pgx.CopyFromSlice(len(stops), func(i int) ([]interface{}, error) {
			s := stops[i]
			return []interface{}{
				route.ID, s.EmployeeID, s.SequenceOrder, s.StopLat, s.StopLng, s.EstimatedArrival,
			}, nil
		}),
	)
	if err != nil {
		return fmt.Errorf("insert route stops: %w", err)
	}

	return tx.Commit(ctx)
}

// GetRouteWithStops retrieves a route and all its ordered stops.
func (r *RouteRepository) GetRouteWithStops(ctx context.Context, routeID string) (*domain.Route, error) {
	const routeQ = `
		SELECT id, vehicle_id, shift_id, company_id, route_date, direction, status,
		       total_distance_km, total_duration_min, escort_required,
		       COALESCE(escort_id::text, '') AS escort_id,
		       created_at, updated_at
		FROM routes WHERE id = $1`

	rows, err := r.db.Pool.Query(ctx, routeQ, routeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	route, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[domain.Route])
	if err != nil {
		return nil, fmt.Errorf("get route %s: %w", routeID, err)
	}

	const stopsQ = `
		SELECT id, route_id, employee_id, sequence_order, stop_lat, stop_lng,
		       estimated_arrival, actual_arrival, boarded
		FROM route_stops
		WHERE route_id = $1
		ORDER BY sequence_order ASC`

	stopRows, err := r.db.Pool.Query(ctx, stopsQ, routeID)
	if err != nil {
		return nil, fmt.Errorf("get route stops: %w", err)
	}
	defer stopRows.Close()

	route.Stops, err = pgx.CollectRows(stopRows, pgx.RowToStructByName[domain.RouteStop])
	if err != nil {
		return nil, err
	}

	return &route, nil
}

// ListByDate returns all routes for a given date and optionally a company.
func (r *RouteRepository) ListByDate(ctx context.Context, date time.Time, companyID string) ([]domain.Route, error) {
	q := `
		SELECT id, vehicle_id, shift_id, company_id, route_date, direction, status,
		       total_distance_km, total_duration_min, escort_required,
		       COALESCE(escort_id::text, '') AS escort_id,
		       created_at, updated_at
		FROM routes
		WHERE route_date = $1`

	args := []interface{}{date}
	if companyID != "" {
		q += " AND company_id = $2"
		args = append(args, companyID)
	}
	q += " ORDER BY company_id, shift_id, direction"

	rows, err := r.db.Pool.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("list routes by date: %w", err)
	}
	defer rows.Close()

	return pgx.CollectRows(rows, pgx.RowToStructByName[domain.Route])
}

// MarkStopArrived records the actual arrival time for a driver check-in.
func (r *RouteRepository) MarkStopArrived(ctx context.Context, stopID string) error {
	const q = `
		UPDATE route_stops
		SET actual_arrival = NOW(), boarded = TRUE
		WHERE id = $1`
	_, err := r.db.Pool.Exec(ctx, q, stopID)
	return err
}

func nilIfEmpty(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}
