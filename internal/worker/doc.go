// Package worker provides a goroutine pool for concurrent job execution.
//
// The Pool manages a fixed number of worker goroutines that process jobs
// from a shared queue. It supports graceful shutdown and context cancellation.
//
// # Basic Usage
//
//	pool := worker.NewPool(4) // 4 workers
//	pool.Start(ctx)
//	defer pool.Stop()
//
//	// Submit jobs
//	for i := 0; i < 100; i++ {
//	    pool.Submit(func() {
//	        // do work
//	    })
//	}
//
// # Configuration
//
// Use NewPoolWithConfig for custom settings:
//
//	config := worker.PoolConfig{
//	    NumWorkers:  8,
//	    QueueFactor: 200, // Queue size = 8 * 200 = 1600
//	}
//	pool := worker.NewPoolWithConfig(config)
//
// # Graceful Shutdown
//
// Stop() waits for all in-flight jobs to complete before returning.
// The context passed to Start() can be used to cancel waiting jobs.
package worker
