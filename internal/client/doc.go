// Package client provides a load generator for stress testing the cluster.
//
// The Client generates read/write traffic against a cluster at a configurable
// rate and ratio. It collects metrics about the generated load.
//
// # Basic Usage
//
//	c := cluster.New()
//	c.CreateNodes(5, "node")
//	c.StartAll(ctx)
//
//	config := client.DefaultConfig()
//	config.WriteRatio = 0.3 // 30% writes, 70% reads
//	cl := client.New(c, config)
//
//	// Run for a duration
//	snap := cl.RunFor(ctx, 10*time.Second)
//	fmt.Printf("Total: %d, RPS: %.2f\n", snap.TotalRequests, snap.RPS)
//
//	// Or run a fixed number of requests
//	snap := cl.RunRequests(ctx, 10000)
//
// # Configuration
//
// The Config struct allows tuning:
//   - NumWorkers: parallel workers (0 = CPU count)
//   - WriteRatio: fraction of write operations (0.0 to 1.0)
//   - KeyRange: key space size
//   - ValueSize: size of values in bytes
//   - RequestsLimit: max requests (0 = unlimited)
package client
