package routing

import (
	"fmt"
	"sort"

	"github.com/motive-ride/motive-ride/internal/domain"
)

// Assigner pairs each cluster with the smallest available vehicle that fits all passengers.
type Assigner struct{}

func NewAssigner() *Assigner {
	return &Assigner{}
}

// Assign performs best-fit vehicle assignment for all clusters.
// Vehicles are consumed in ascending capacity order so that the smallest
// suitable vehicle is always chosen first.
func (a *Assigner) Assign(
	clusters []Cluster,
	vehicles []domain.Vehicle,
	escortRequired bool,
) ([]AssignedCluster, error) {
	// Work on a copy so we can mark vehicles as used
	available := make([]domain.Vehicle, len(vehicles))
	copy(available, vehicles)

	// Sort ascending by capacity for best-fit
	sort.Slice(available, func(i, j int) bool {
		return available[i].Capacity < available[j].Capacity
	})

	usedIdx := make([]bool, len(available))
	result := make([]AssignedCluster, 0, len(clusters))

	for _, cl := range clusters {
		needed := len(cl.Employees)
		needsEscort := escortRequired && hasFemalEmployee(cl.Employees)

		// Reserve one seat for escort if required
		effectiveNeed := needed
		if needsEscort {
			effectiveNeed++
		}

		vehicleIdx := -1
		for i, v := range available {
			if !usedIdx[i] && v.Capacity >= effectiveNeed {
				vehicleIdx = i
				break
			}
		}

		if vehicleIdx == -1 {
			return nil, fmt.Errorf("no available vehicle for cluster of %d passengers (escort=%v)", needed, needsEscort)
		}

		usedIdx[vehicleIdx] = true
		result = append(result, AssignedCluster{
			Cluster:        cl,
			Vehicle:        available[vehicleIdx],
			EscortRequired: needsEscort,
		})
	}

	return result, nil
}

func hasFemalEmployee(employees []domain.Employee) bool {
	for _, e := range employees {
		if e.IsFemale() {
			return true
		}
	}
	return false
}
