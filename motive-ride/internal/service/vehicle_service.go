package service

import (
	"context"
	"fmt"

	"github.com/motive-ride/motive-ride/internal/domain"
	"github.com/motive-ride/motive-ride/internal/repository"
)

type VehicleService struct {
	repo *repository.VehicleRepository
}

func NewVehicleService(repo *repository.VehicleRepository) *VehicleService {
	return &VehicleService{repo: repo}
}

func (s *VehicleService) Create(ctx context.Context, v *domain.Vehicle) error {
	if v.Capacity <= 0 {
		return fmt.Errorf("vehicle capacity must be positive")
	}
	return s.repo.Create(ctx, v)
}

func (s *VehicleService) ListAvailable(ctx context.Context) ([]domain.Vehicle, error) {
	return s.repo.ListAvailable(ctx)
}

func (s *VehicleService) GetByID(ctx context.Context, id string) (*domain.Vehicle, error) {
	return s.repo.GetByID(ctx, id)
}
