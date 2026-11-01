package service

import (
	"context"
	"fmt"
	"time"

	"github.com/motive-ride/motive-ride/internal/domain"
	"github.com/motive-ride/motive-ride/internal/repository"
	"github.com/motive-ride/motive-ride/internal/routing"
)

// RoutingService orchestrates route generation and persists results.
type RoutingService struct {
	employeeRepo *repository.EmployeeRepository
	vehicleRepo  *repository.VehicleRepository
	routeRepo    *repository.RouteRepository
	engine       *routing.Engine
}

func NewRoutingService(
	emp *repository.EmployeeRepository,
	veh *repository.VehicleRepository,
	rte *repository.RouteRepository,
	engine *routing.Engine,
) *RoutingService {
	return &RoutingService{
		employeeRepo: emp,
		vehicleRepo:  veh,
		routeRepo:    rte,
		engine:       engine,
	}
}

// GenerateRouteRequest is the API-level request to trigger route generation.
type GenerateRouteRequest struct {
	CompanyID string
	ShiftID   string
	RouteDate time.Time
	Direction domain.RouteDirection
	Shift     domain.Shift
}

// GenerateRoutes runs the full routing pipeline and saves results to the database.
func (s *RoutingService) GenerateRoutes(ctx context.Context, req GenerateRouteRequest) (*routing.GenerateResult, error) {
	// Load all active employees for this company+shift
	employees, err := s.employeeRepo.ListByShift(ctx, req.CompanyID, req.ShiftID)
	if err != nil {
		return nil, fmt.Errorf("load employees: %w", err)
	}
	if len(employees) == 0 {
		return nil, fmt.Errorf("no active employees found for company %s shift %s", req.CompanyID, req.ShiftID)
	}

	// Load available vehicles
	vehicles, err := s.vehicleRepo.ListAvailable(ctx)
	if err != nil {
		return nil, fmt.Errorf("load vehicles: %w", err)
	}
	if len(vehicles) == 0 {
		return nil, fmt.Errorf("no vehicles available")
	}

	// Run routing engine
	result, err := s.engine.Generate(ctx, routing.GenerateRequest{
		CompanyID: req.CompanyID,
		ShiftID:   req.ShiftID,
		RouteDate: req.RouteDate,
		Direction: req.Direction,
		Employees: employees,
		Vehicles:  vehicles,
		Shift:     req.Shift,
	})
	if err != nil {
		return nil, fmt.Errorf("routing engine: %w", err)
	}

	// Persist all generated routes
	if err := s.persistRoutes(ctx, result.Routes, req); err != nil {
		return nil, fmt.Errorf("persist routes: %w", err)
	}

	return result, nil
}

func (s *RoutingService) persistRoutes(ctx context.Context, routes []routing.OptimizedRoute, req GenerateRouteRequest) error {
	for _, r := range routes {
		route := domain.Route{
			VehicleID:        r.AssignedCluster.Vehicle.ID,
			ShiftID:          req.ShiftID,
			CompanyID:        req.CompanyID,
			RouteDate:        req.RouteDate,
			Direction:        req.Direction,
			Status:           domain.RouteStatusPlanned,
			TotalDistanceKm:  r.TotalDistanceKm,
			TotalDurationMin: r.TotalDurationMin,
			EscortRequired:   r.AssignedCluster.EscortRequired,
		}

		stops := make([]domain.RouteStop, len(r.StopSequence))
		for i, s := range r.StopSequence {
			t := s.EstimatedArrival
			stops[i] = domain.RouteStop{
				EmployeeID:       s.Employee.ID,
				SequenceOrder:    s.SequenceOrder,
				StopLat:          s.Employee.HomeLat,
				StopLng:          s.Employee.HomeLng,
				EstimatedArrival: &t,
			}
		}

		if err := s.routeRepo.CreateRoute(ctx, &route, stops); err != nil {
			return fmt.Errorf("save route for vehicle %s: %w", r.AssignedCluster.Vehicle.ID, err)
		}

		// Mark vehicle as on-route
		_ = s.vehicleRepo.UpdateStatus(ctx, r.AssignedCluster.Vehicle.ID, domain.VehicleOnRoute)
	}
	return nil
}

// GetRouteManifest returns the full route with all stop details.
func (s *RoutingService) GetRouteManifest(ctx context.Context, routeID string) (*domain.RouteManifest, error) {
	route, err := s.routeRepo.GetRouteWithStops(ctx, routeID)
	if err != nil {
		return nil, err
	}

	vehicle, err := s.vehicleRepo.GetByID(ctx, route.VehicleID)
	if err != nil {
		return nil, err
	}

	employees := make([]domain.Employee, 0, len(route.Stops))
	for _, stop := range route.Stops {
		emp, err := s.employeeRepo.GetByID(ctx, stop.EmployeeID)
		if err != nil {
			continue
		}
		employees = append(employees, *emp)
	}

	return &domain.RouteManifest{
		Route:     *route,
		Vehicle:   *vehicle,
		Stops:     route.Stops,
		Employees: employees,
	}, nil
}
