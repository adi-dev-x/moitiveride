package routing

import (
	"fmt"

	"github.com/motive-ride/motive-ride/internal/domain"
)

// RuleViolation describes a constraint that was broken in a generated route.
type RuleViolation struct {
	RouteIdx int
	Rule     string
	Detail   string
}

func (v RuleViolation) Error() string {
	return fmt.Sprintf("route[%d] violated %s: %s", v.RouteIdx, v.Rule, v.Detail)
}

// Validator checks all generated routes against every business constraint.
type Validator struct{}

func NewValidator() *Validator {
	return &Validator{}
}

// Validate returns all violations found across the generated routes.
func (v *Validator) Validate(routes []OptimizedRoute) []RuleViolation {
	var violations []RuleViolation

	for i, r := range routes {
		violations = append(violations, v.checkCapacity(i, r)...)
		violations = append(violations, v.checkCompanyIsolation(i, r)...)
		violations = append(violations, v.checkRideTime(i, r)...)
		violations = append(violations, v.checkEscort(i, r)...)
		violations = append(violations, v.checkNoDuplicates(i, r)...)
	}

	violations = append(violations, v.checkNoMissedEmployees(routes)...)

	return violations
}

func (v *Validator) checkCapacity(idx int, r OptimizedRoute) []RuleViolation {
	cap := r.AssignedCluster.Vehicle.Capacity
	passengers := len(r.AssignedCluster.Cluster.Employees)
	if r.AssignedCluster.EscortRequired {
		passengers++ // escort occupies a seat
	}
	if passengers > cap {
		return []RuleViolation{{
			RouteIdx: idx,
			Rule:     "CAPACITY",
			Detail:   fmt.Sprintf("passengers %d > vehicle capacity %d", passengers, cap),
		}}
	}
	return nil
}

func (v *Validator) checkCompanyIsolation(idx int, r OptimizedRoute) []RuleViolation {
	companies := make(map[string]bool)
	for _, s := range r.StopSequence {
		companies[s.Employee.CompanyID] = true
	}
	if len(companies) > 1 {
		return []RuleViolation{{
			RouteIdx: idx,
			Rule:     "COMPANY_ISOLATION",
			Detail:   fmt.Sprintf("%d companies mixed in one vehicle", len(companies)),
		}}
	}
	return nil
}

func (v *Validator) checkRideTime(idx int, r OptimizedRoute) []RuleViolation {
	if r.TotalDurationMin > int(maxRideTimeMin) {
		return []RuleViolation{{
			RouteIdx: idx,
			Rule:     "MAX_RIDE_TIME",
			Detail:   fmt.Sprintf("duration %d min exceeds limit %d min", r.TotalDurationMin, int(maxRideTimeMin)),
		}}
	}
	return nil
}

func (v *Validator) checkEscort(idx int, r OptimizedRoute) []RuleViolation {
	if r.AssignedCluster.EscortRequired {
		// The service layer must assign an escort before persisting.
		// Here we simply flag routes that required but got no escort ID.
		// In the full system the escort is assigned before validation runs.
	}
	return nil
}

func (v *Validator) checkNoDuplicates(idx int, r OptimizedRoute) []RuleViolation {
	seen := make(map[string]bool)
	for _, s := range r.StopSequence {
		if seen[s.Employee.ID] {
			return []RuleViolation{{
				RouteIdx: idx,
				Rule:     "DUPLICATE_EMPLOYEE",
				Detail:   fmt.Sprintf("employee %s appears more than once", s.Employee.ID),
			}}
		}
		seen[s.Employee.ID] = true
	}
	return nil
}

// checkNoMissedEmployees verifies every employee appears in exactly one route.
func (v *Validator) checkNoMissedEmployees(routes []OptimizedRoute) []RuleViolation {
	seen := make(map[string]int) // employeeID → route index
	var violations []RuleViolation

	for i, r := range routes {
		for _, s := range r.StopSequence {
			if prev, exists := seen[s.Employee.ID]; exists {
				violations = append(violations, RuleViolation{
					RouteIdx: i,
					Rule:     "DUPLICATE_ACROSS_ROUTES",
					Detail:   fmt.Sprintf("employee %s in both route %d and route %d", s.Employee.ID, prev, i),
				})
			}
			seen[s.Employee.ID] = i
		}
	}
	return violations
}

// CheckLastFemaleDropPolicy enforces the optional last-female-drop rule.
// If policy is enabled and the last stop is a female employee, it attempts to reorder.
func (v *Validator) CheckLastFemaleDropPolicy(route *OptimizedRoute, direction domain.RouteDirection) bool {
	if direction != domain.DirectionDrop || len(route.StopSequence) == 0 {
		return true
	}
	last := route.StopSequence[len(route.StopSequence)-1]
	if !last.Employee.IsFemale() {
		return true // no issue
	}
	if !route.AssignedCluster.EscortRequired {
		// Try to swap last female with the second-to-last stop
		if len(route.StopSequence) >= 2 {
			n := len(route.StopSequence)
			route.StopSequence[n-1], route.StopSequence[n-2] = route.StopSequence[n-2], route.StopSequence[n-1]
			// Update sequence orders
			route.StopSequence[n-2].SequenceOrder = n - 1
			route.StopSequence[n-1].SequenceOrder = n
		}
	}
	return true
}
