// Package etl provides ETL pipeline management
// Manager pattern adapted from gPRO and Rust etl-rust
package etl

import (
	"context"
	"fmt"
	"sync"

	"github.com/cuong/go-etl/pkg/bucket"
)

// ETLRunner interface for objects that can be run as ETL pipelines
type ETLRunner interface {
	Name() string
	Run(ctx context.Context, cfg *bucket.Config) error
}

// Config configures the manager's behavior
type Config struct {
	WorkerNum int // Maximum number of concurrent pipelines
}

// Manager manages and runs multiple ETL pipelines concurrently
type Manager struct {
	pipelines    []ETLRunner
	cfg          Config
	bucketConfig *bucket.Config
}

// NewManager creates a new ETL manager
func NewManager(cfg *Config, bucketConfig *bucket.Config) *Manager {
	if cfg.WorkerNum <= 0 {
		cfg.WorkerNum = 4
	}

	return &Manager{
		pipelines:    make([]ETLRunner, 0),
		cfg:          *cfg,
		bucketConfig: bucketConfig,
	}
}

// AddPipeline adds an ETL pipeline to the manager
// Use AddPipelineGeneric function instead
func (m *Manager) addPipelineInternal(runner ETLRunner) {
	m.pipelines = append(m.pipelines, runner)
}

// AddRunner adds a custom ETL runner to the manager
func (m *Manager) AddRunner(runner ETLRunner) {
	m.pipelines = append(m.pipelines, runner)
}

// AddPipelineGeneric adds an ETL pipeline with type parameters
// E: Extract type, T: Transform/Load type
func AddPipelineGeneric[E, T any](m *Manager, processor ETLProcessor[E, T], name string) {
	adapter := &pipelineAdapter[E, T]{
		etl:  NewETL(processor),
		name: name,
	}
	m.addPipelineInternal(adapter)
}

// RunAll executes all pipelines concurrently with semaphore-limited parallelism
// Inspired by Rust's ETLPipelineManager with semaphore + channel pattern
func (m *Manager) RunAll(ctx context.Context) error {
	if len(m.pipelines) == 0 {
		return fmt.Errorf("no pipelines registered")
	}

	// Semaphore to limit concurrent pipeline execution
	sem := make(chan struct{}, m.cfg.WorkerNum)

	// Channel to collect results
	results := make(chan error, len(m.pipelines))

	var wg sync.WaitGroup

	// Launch all pipelines
	for _, pipeline := range m.pipelines {
		wg.Add(1)

		go func(p ETLRunner) {
			defer wg.Done()

			// Acquire semaphore slot
			sem <- struct{}{}
			defer func() { <-sem }()

			// Run pipeline
			if err := p.Run(ctx, m.bucketConfig); err != nil {
				results <- fmt.Errorf("pipeline %s failed: %w", p.Name(), err)
			} else {
				results <- nil
			}
		}(pipeline)
	}

	// Wait for all pipelines to complete
	wg.Wait()
	close(results)

	// Collect and return first error if any
	for err := range results {
		if err != nil {
			return err
		}
	}

	return nil
}

// pipelineAdapter adapts ETL[E,T] to ETLRunner interface
type pipelineAdapter[E, T any] struct {
	etl  *ETL[E, T]
	name string
}

func (a *pipelineAdapter[E, T]) Name() string {
	return a.name
}

func (a *pipelineAdapter[E, T]) Run(ctx context.Context, cfg *bucket.Config) error {
	// Run pre-process
	if err := a.etl.PreProcess(ctx); err != nil {
		return fmt.Errorf("pre-process failed: %w", err)
	}

	// Run main ETL
	if err := a.etl.Run(ctx, cfg); err != nil {
		return fmt.Errorf("ETL run failed: %w", err)
	}

	// Run post-process
	if err := a.etl.PostProcess(ctx); err != nil {
		return fmt.Errorf("post-process failed: %w", err)
	}

	return nil
}
