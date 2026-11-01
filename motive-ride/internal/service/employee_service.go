package service

import (
	"context"
	"fmt"

	"github.com/motive-ride/motive-ride/internal/domain"
	"github.com/motive-ride/motive-ride/internal/repository"
)

type EmployeeService struct {
	repo *repository.EmployeeRepository
}

func NewEmployeeService(repo *repository.EmployeeRepository) *EmployeeService {
	return &EmployeeService{repo: repo}
}

func (s *EmployeeService) Create(ctx context.Context, e *domain.Employee) error {
	if e.HomeLat == 0 || e.HomeLng == 0 {
		return fmt.Errorf("home coordinates are required")
	}
	return s.repo.Create(ctx, e)
}

func (s *EmployeeService) BulkImport(ctx context.Context, employees []domain.Employee) error {
	if len(employees) == 0 {
		return fmt.Errorf("empty employee list")
	}
	return s.repo.BulkCreate(ctx, employees)
}

func (s *EmployeeService) GetByID(ctx context.Context, id string) (*domain.Employee, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *EmployeeService) ListByShift(ctx context.Context, companyID, shiftID string) ([]domain.Employee, error) {
	return s.repo.ListByShift(ctx, companyID, shiftID)
}
