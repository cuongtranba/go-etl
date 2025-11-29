// Package etl provides a generic Extract-Transform-Load framework
// Extracted and adapted from gPRO (ext/pkg/etl/etl.go)
package etl

import (
	"context"
	"fmt"

	"github.com/cuong/go-etl/pkg/bucket"
)

// ETLProcessor defines the interface for ETL operations
// E: Extract type (source data)
// T: Transform/Load type (destination data)
type ETLProcessor[E, T any] interface {
	// Extract data from source, returns channel of extracted items
	Extract(ctx context.Context) (<-chan Payload[E], error)

	// Transform single extracted item to load format
	Transform(ctx context.Context, e E) T

	// Load batch of transformed items to destination
	Load(ctx context.Context, data []T) error

	// PreProcess runs before extraction (setup, migrations, etc.)
	PreProcess(ctx context.Context) error

	// PostProcess runs after load completes (cleanup, sync, etc.)
	PostProcess(ctx context.Context) error
}

// Payload wraps extracted data with error handling
type Payload[E any] struct {
	Data E
	Err  error
}

// ETL orchestrates the extract-transform-load process
type ETL[E, T any] struct {
	processor ETLProcessor[E, T]
}

// NewETL creates a new ETL instance with the given processor
func NewETL[E, T any](processor ETLProcessor[E, T]) *ETL[E, T] {
	return &ETL[E, T]{
		processor: processor,
	}
}

// Run executes the complete ETL pipeline:
// 1. PreProcess
// 2. Extract -> Bucket (batching) -> Transform -> Load
// 3. PostProcess
func (e *ETL[E, T]) Run(ctx context.Context, bucketCfg *bucket.Config) error {
	// Pre-processing (setup, migrations, etc.)
	if err := e.processor.PreProcess(ctx); err != nil {
		return fmt.Errorf("failed to pre-process: %w", err)
	}

	// Create bucket for batching
	b, err := bucket.New[E](bucketCfg)
	if err != nil {
		return fmt.Errorf("failed to create bucket: %w", err)
	}

	// Extract data
	extractor, err := e.processor.Extract(ctx)
	if err != nil {
		return fmt.Errorf("failed to extract: %w", err)
	}

	// Feed extractor into bucket
	go func() {
		for {
			select {
			case <-ctx.Done():
				b.Close()
				return
			case payload, ok := <-extractor:
				if !ok {
					b.Close()
					return
				}
				if payload.Err != nil {
					fmt.Printf("ERROR: Failed to extract: %v\n", payload.Err)
					b.Close()
					return
				}
				b.Consume(payload.Data)
			}
		}
	}()

	// Process batches: Transform -> Load
	err = b.Run(ctx, func(ctx context.Context, items []E) error {
		// Transform each item
		transformed := make([]T, 0, len(items))
		for _, item := range items {
			t := e.processor.Transform(ctx, item)
			transformed = append(transformed, t)
		}

		// Load batch
		return e.processor.Load(ctx, transformed)
	})

	if err != nil {
		return fmt.Errorf("failed to run ETL: %w", err)
	}

	// Post-processing (cleanup, sync tracking, etc.)
	if err := e.processor.PostProcess(ctx); err != nil {
		return fmt.Errorf("failed to post-process: %w", err)
	}

	return nil
}

// PreProcess calls the processor's pre-process hook
func (e *ETL[E, T]) PreProcess(ctx context.Context) error {
	return e.processor.PreProcess(ctx)
}

// PostProcess calls the processor's post-process hook
func (e *ETL[E, T]) PostProcess(ctx context.Context) error {
	return e.processor.PostProcess(ctx)
}
