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

func TestWorkerPoolContextCancel(t *testing.T) {
	pool := NewPool(2)
	ctx, cancel := context.WithCancel(context.Background())
	pool.Start(ctx)

	var counter atomic.Int32
	blocker := make(chan struct{})

	// Submit a blocking job
	pool.Submit(func() {
		<-blocker
		counter.Add(1)
	})

	// Cancel context
	cancel()

	// Unblock the job
	close(blocker)

	// Give time for workers to stop
	time.Sleep(50 * time.Millisecond)

	// Submit after context cancel should fail
	result := pool.Submit(func() {
		counter.Add(1)
	})
	if result {
		t.Error("expected Submit to return false after context cancel")
	}

	pool.Stop()
}

func TestWorkerPoolSubmitWait(t *testing.T) {
	pool := NewPool(2)
	ctx := context.Background()
	pool.Start(ctx)
	defer pool.Stop()

	var counter atomic.Int32

	// Submit jobs using SubmitWait
	for range 5 {
		result := pool.SubmitWait(func() {
			counter.Add(1)
		})
		if !result {
			t.Error("expected SubmitWait to return true")
		}
	}

	// Wait for jobs to complete
	time.Sleep(50 * time.Millisecond)

	if counter.Load() != 5 {
		t.Errorf("expected 5 jobs completed, got %d", counter.Load())
	}
}

func TestWorkerPoolSubmitWaitAfterCancel(t *testing.T) {
	pool := NewPool(2)
	ctx, cancel := context.WithCancel(context.Background())
	pool.Start(ctx)

	cancel()

	// SubmitWait after cancel should return false
	result := pool.SubmitWait(func() {})
	if result {
		t.Error("expected SubmitWait to return false after cancel")
	}

	pool.Stop()
}

func TestWorkerPoolNegativeWorkers(t *testing.T) {
	// Negative workers should default to CPU count
	pool := NewPool(-5)
	if pool.NumWorkers() != runtime.NumCPU() {
		t.Errorf("expected %d workers for negative input, got %d", runtime.NumCPU(), pool.NumWorkers())
	}
}

func TestWorkerPoolConcurrentSubmit(t *testing.T) {
	pool := NewPool(4)
	ctx := context.Background()
	pool.Start(ctx)
	defer pool.Stop()

	var counter atomic.Int32
	const numGoroutines = 10
	const jobsPerGoroutine = 100

	var wg atomic.Int32
	wg.Store(numGoroutines)

	for range numGoroutines {
		go func() {
			for range jobsPerGoroutine {
				pool.Submit(func() {
					counter.Add(1)
				})
			}
			wg.Add(-1)
		}()
	}

	// Wait for all goroutines to finish submitting
	for wg.Load() > 0 {
		time.Sleep(time.Millisecond)
	}

	// Wait for all jobs to complete
	time.Sleep(100 * time.Millisecond)

	expected := int32(numGoroutines * jobsPerGoroutine)
	if counter.Load() != expected {
		t.Errorf("expected %d jobs completed, got %d", expected, counter.Load())
	}
}
