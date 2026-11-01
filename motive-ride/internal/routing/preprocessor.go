package routing

import (
	"fmt"

	"github.com/motive-ride/motive-ride/internal/domain"
)

// Bangalore bounding box
const (
	bangaloreSouthLat = 12.7342
	bangaloreNorthLat = 13.1435
	bangaloreWestLng  = 77.3791
	bangaloreEastLng  = 77.8106
)

// ValidationError holds all issues found during preprocessing.
type ValidationError struct {
	Errors []string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("%d validation errors: %v", len(e.Errors), e.Errors)
}

func (e *ValidationError) IsEmpty() bool {
	return len(e.Errors) == 0
}

// Preprocessor validates and enriches employee data before routing.
type Preprocessor struct{}

func NewPreprocessor() *Preprocessor {
	return &Preprocessor{}
}

// Validate checks all employees and returns a ValidationError if any fail.
func (p *Preprocessor) Validate(employees []domain.Employee) error {
	ve := &ValidationError{}
	seen := make(map[string]bool)

	for _, e := range employees {
		if seen[e.ID] {
			ve.Errors = append(ve.Errors, fmt.Sprintf("duplicate employee ID: %s", e.ID))
			continue
		}
		seen[e.ID] = true

		if err := p.validateCoordinates(e); err != nil {
			ve.Errors = append(ve.Errors, err.Error())
		}
	}

	if !ve.IsEmpty() {
		return ve
	}
	return nil
}

func (p *Preprocessor) validateCoordinates(e domain.Employee) error {
	if e.HomeLat < bangaloreSouthLat || e.HomeLat > bangaloreNorthLat {
		return fmt.Errorf("employee %s: latitude %.6f outside Bangalore bounds", e.ID, e.HomeLat)
	}
	if e.HomeLng < bangaloreWestLng || e.HomeLng > bangaloreEastLng {
		return fmt.Errorf("employee %s: longitude %.6f outside Bangalore bounds", e.ID, e.HomeLng)
	}
	return nil
}

// FlagNightShiftFemales marks employees that trigger the escort requirement.
// Returns true if any female employees exist in a night-shift drop route.
func (p *Preprocessor) RequiresEscort(shift domain.Shift, direction domain.RouteDirection, employees []domain.Employee) bool {
	if !shift.IsNight() || direction != domain.DirectionDrop {
		return false
	}
	for _, e := range employees {
		if e.IsFemale() {
			return true
		}
	}
	return false
}
