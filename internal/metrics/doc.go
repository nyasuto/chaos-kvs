// Package metrics provides request metrics collection and reporting.
//
// Metrics collects statistics about request latency, success/failure rates,
// and throughput (RPS). It is thread-safe and optimized for high-concurrency
// scenarios.
//
// # Basic Usage
//
//	m := metrics.New()
//
//	// Record requests
//	start := time.Now()
//	// ... do work ...
//	m.RecordSuccess(time.Since(start))
//
//	// Get statistics
//	fmt.Printf("Total: %d, RPS: %.2f, P99: %v\n",
//	    m.TotalRequests(), m.RPS(), m.P99Latency())
//
//	// Get a snapshot
//	snap := m.Snapshot()
//
// # Configuration
//
// Use NewWithConfig for custom settings:
//
//	config := metrics.Config{
//	    MaxLatencySamples: 5000, // More samples for P99 accuracy
//	}
//	m := metrics.NewWithConfig(config)
//
// # Thread Safety
//
// All operations use atomic counters and are safe for concurrent access.
package metrics
