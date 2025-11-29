// Package bucket provides batching and concurrent processing
// Extracted and adapted from gPRO (ext/pkg/bucket/bucket.go)
package bucket

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// ProcessFunc processes a batch of items
type ProcessFunc[T any] func(ctx context.Context, items []T) error

// Config configures bucket batching and worker behavior
type Config struct {
	BatchSize int           // Number of items per batch
	Timeout   time.Duration // Max time to wait before flushing partial batch
	WorkerNum int           // Number of parallel workers
}

// Bucket batches items and processes them with multiple workers
type Bucket[T any] struct {
	cfg      Config
	consumer chan T
}

// New creates a new bucket with the given configuration
func New[T any](cfg *Config) (*Bucket[T], error) {
	if cfg.BatchSize <= 0 {
		cfg.BatchSize = 100
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 5 * time.Second
	}
	if cfg.WorkerNum <= 0 {
		cfg.WorkerNum = 1
	}

	return &Bucket[T]{
		cfg:      *cfg,
		consumer: make(chan T, cfg.BatchSize),
	}, nil
}

// Consume adds an item to the bucket for processing
func (b *Bucket[T]) Consume(item T) {
	b.consumer <- item
}

// Close signals that no more items will be added
func (b *Bucket[T]) Close() {
	close(b.consumer)
}

// Run starts processing items with multiple workers
// Each worker accumulates items into batches and calls processFunc
// Batches are flushed when:
// - Batch size is reached
// - Timeout occurs
// - Channel is closed
// - Context is cancelled
func (b *Bucket[T]) Run(ctx context.Context, processFunc ProcessFunc[T]) error {
	procCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	errCh := make(chan error, b.cfg.WorkerNum)
	var wg sync.WaitGroup

	// Start workers
	for i := 0; i < b.cfg.WorkerNum; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			if err := b.worker(procCtx, processFunc); err != nil {
				select {
				case errCh <- fmt.Errorf("worker %d: %w", workerID, err):
				default:
				}
				cancel() // Cancel other workers on error
			}
		}(i)
	}

	// Wait for all workers
	wg.Wait()
	close(errCh)

	// Check for errors
	for err := range errCh {
		return err
	}

	return nil
}

// worker processes items in batches
func (b *Bucket[T]) worker(ctx context.Context, processFunc ProcessFunc[T]) error {
	ticker := time.NewTicker(b.cfg.Timeout)
	defer ticker.Stop()

	queue := make([]T, 0, b.cfg.BatchSize)

	flush := func() error {
		if len(queue) > 0 {
			if err := processFunc(ctx, queue); err != nil {
				return err
			}
			queue = queue[:0] // Reset queue
		}
		return nil
	}

	for {
		select {
		case <-ctx.Done():
			// Flush remaining items on context cancellation
			return flush()

		case <-ticker.C:
			// Timeout: flush partial batch
			if err := flush(); err != nil {
				return err
			}

		case item, ok := <-b.consumer:
			if !ok {
				// Channel closed: flush remaining items
				return flush()
			}

			queue = append(queue, item)

			// Flush when batch size is reached
			if len(queue) >= b.cfg.BatchSize {
				if err := flush(); err != nil {
					return err
				}
			}
		}
	}
}
