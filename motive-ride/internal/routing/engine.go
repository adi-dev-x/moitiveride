package routing

import (
	"context"
	"fmt"
	"log/slog"
	"runtime"
	"sync"
	"time"

	"github.com/motive-ride/motive-ride/internal/domain"
)

// Engine is the top-level orchestrator of the routing pipeline.
// Stages: preprocess → partition → cluster → assign → optimize → validate.
type Engine struct {
	preprocessor *Preprocessor
	partitioner  *Partitioner
	clusterer    *Clusterer
	assigner     *Assigner
	optimizer    *Optimizer
	validator    *Validator

	depotLat    float64
	depotLng    float64
	workerCount int
}

// NewEngine wires the pipeline with the supplied TimeMatrixProvider.
// Pass a GoogleProvider for production; an OfflineProvider for dev/test.
func NewEngine(depotLat, depotLng float64, traffic TimeMatrixProvider) *Engine {
	workers := runtime.NumCPU() * 2
	if workers > 16 {
		workers = 16
	}

	return &Engine{
		preprocessor: NewPreprocessor(),
		partitioner:  NewPartitioner(),
		clusterer:    NewClusterer(20),
		assigner:     NewAssigner(),
		optimizer:    NewOptimizer(traffic),
		validator:    NewValidator(),
		depotLat:     depotLat,
		depotLng:     depotLng,
		workerCount:  workers,
	}
}

// Generate runs the full routing pipeline for the given request.
// All Company × Shift partitions run concurrently up to workerCount goroutines.
func (e *Engine) Generate(ctx context.Context, req GenerateRequest) (*GenerateResult, error) {
	start := time.Now()

	if err := e.preprocessor.Validate(req.Employees); err != nil {
		return nil, fmt.Errorf("preprocessing: %w", err)
	}

	partitions := e.partitioner.Partition(req.Employees)

	type partResult struct {
		routes []OptimizedRoute
		err    *PartitionError
	}

	resultCh := make(chan partResult, len(partitions))
	sem := make(chan struct{}, e.workerCount)

	var wg sync.WaitGroup
	for key, employees := range partitions {
		wg.Add(1)
		go func(k PartitionKey, emps []domain.Employee) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			routes, err := e.processPartition(ctx, k, emps, req)
			if err != nil {
				resultCh <- partResult{err: &PartitionError{Key: k, Error: err}}
				return
			}
			resultCh <- partResult{routes: routes}
		}(key, employees)
	}

	go func() {
		wg.Wait()
		close(resultCh)
	}()

	var allRoutes []OptimizedRoute
	var allErrors []PartitionError
	for res := range resultCh {
		if res.err != nil {
			allErrors = append(allErrors, *res.err)
		} else {
			allRoutes = append(allRoutes, res.routes...)
		}
	}

	if violations := e.validator.Validate(allRoutes); len(violations) > 0 {
		for _, v := range violations {
			allErrors = append(allErrors, PartitionError{Error: fmt.Errorf("%s", v.Error())})
		}
	}

	return &GenerateResult{
		Routes:     allRoutes,
		Errors:     allErrors,
		DurationMs: time.Since(start).Milliseconds(),
	}, nil
}

func (e *Engine) processPartition(
	ctx context.Context,
	key PartitionKey,
	employees []domain.Employee,
	req GenerateRequest,
) ([]OptimizedRoute, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	escortRequired := e.preprocessor.RequiresEscort(req.Shift, req.Direction, employees)

	clusters := e.clusterer.Cluster(employees)

	assigned, err := e.assigner.Assign(clusters, req.Vehicles, escortRequired)
	if err != nil {
		return nil, fmt.Errorf("partition %s:%s assign: %w", key.CompanyID, key.ShiftID, err)
	}

	routes := make([]OptimizedRoute, 0, len(assigned))
	for _, ac := range assigned {
		r, err := e.optimizer.Optimize(ctx, ac, e.depotLat, e.depotLng, req.RouteDate, req.Direction)
		if err != nil {
			slog.Warn("cluster optimisation failed, skipping",
				"company", key.CompanyID,
				"shift", key.ShiftID,
				"employees", len(ac.Cluster.Employees),
				"error", err,
			)
			continue
		}

		e.validator.CheckLastFemaleDropPolicy(r, req.Direction)
		routes = append(routes, *r)
	}

	return routes, nil
}
