package worker

import (
	"context"
	"runtime"
	"sync/atomic"
	"testing"
	"time"
)

func TestNewWorkerPool(t *testing.T) {
	pool := NewPool(4)
	if pool.NumWorkers() != 4 {
		t.Errorf("expected 4 workers, got %d", pool.NumWorkers())
	}

	// Zero should default to CPU count
	pool2 := NewPool(0)
	if pool2.NumWorkers() != runtime.NumCPU() {
		t.Errorf("expected %d workers, got %d", runtime.NumCPU(), pool2.NumWorkers())
	}
}

func TestWorkerPoolStartStop(t *testing.T) {
	pool := NewPool(2)
	ctx := context.Background()

	pool.Start(ctx)
	// Double start should be no-op
	pool.Start(ctx)

	pool.Stop()
	// Double stop should be no-op
	pool.Stop()
}

func TestWorkerPoolSubmit(t *testing.T) {
	pool := NewPool(2)
	ctx := context.Background()
	pool.Start(ctx)
	defer pool.Stop()

	var counter atomic.Int32
	done := make(chan struct{})

	for range 10 {
		pool.Submit(func() {
			counter.Add(1)
		})
	}

	go func() {
		for counter.Load() < 10 {
			time.Sleep(time.Millisecond)
		}
		close(done)
	}()

	select {
	case <-done:
		// Success
	case <-time.After(time.Second):
		t.Error("timeout waiting for jobs to complete")
	}

	if counter.Load() != 10 {
		t.Errorf("expected 10 jobs completed, got %d", counter.Load())
	}
}

func TestWorkerPoolSubmitAfterStop(t *testing.T) {
	pool := NewPool(2)
	ctx := context.Background()
	pool.Start(ctx)
	pool.Stop()

	// Submit after stop should return false
	result := pool.Submit(func() {})
	if result {
		t.Error("expected Submit to return false after stop")
	}
}

func TestWorkerPoolQueueSize(t *testing.T) {
	pool := NewPool(1)
	ctx := context.Background()

	// Start but with a blocking job
	pool.Start(ctx)
	defer pool.Stop()

	// Queue should be empty initially
	if pool.QueueSize() != 0 {
		t.Errorf("expected queue size 0, got %d", pool.QueueSize())
	}
}
