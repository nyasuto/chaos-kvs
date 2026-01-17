// Package cluster provides multi-node cluster management.
//
// A Cluster manages multiple Node instances, providing operations for
// batch starting/stopping and node discovery.
//
// # Basic Usage
//
//	c := cluster.New()
//
//	// Create and add nodes
//	if err := c.CreateNodes(5, "node"); err != nil {
//	    log.Fatal(err)
//	}
//
//	// Start all nodes
//	if err := c.StartAll(ctx); err != nil {
//	    log.Fatal(err)
//	}
//	defer c.StopAll()
//
//	// Access individual nodes
//	if n, ok := c.GetNode("node-1"); ok {
//	    n.Set("key", []byte("value"))
//	}
//
// # Thread Safety
//
// All cluster operations are thread-safe and can be called concurrently.
// Node starting and stopping is performed in parallel for efficiency.
package cluster
