package routing

import "github.com/motive-ride/motive-ride/internal/domain"

// Partitioner splits employees into Company × Shift sub-problems.
// This is the single most important scalability optimisation: it converts one
// 20,000-employee problem into hundreds of small independent ones that run in parallel.
type Partitioner struct{}

func NewPartitioner() *Partitioner {
	return &Partitioner{}
}

// Partition groups employees by their PartitionKey (CompanyID + ShiftID).
func (p *Partitioner) Partition(employees []domain.Employee) map[PartitionKey][]domain.Employee {
	groups := make(map[PartitionKey][]domain.Employee)
	for _, e := range employees {
		key := PartitionKey{CompanyID: e.CompanyID, ShiftID: e.ShiftID}
		groups[key] = append(groups[key], e)
	}
	return groups
}
