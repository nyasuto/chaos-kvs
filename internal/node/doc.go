// Package node provides an in-memory key-value store node implementation.
//
// Each Node represents a single KVS instance that can store and retrieve
// key-value pairs. Nodes are thread-safe and support concurrent access.
//
// # Basic Usage
//
//	n := node.New("node-1")
//	if err := n.Start(ctx); err != nil {
//	    log.Fatal(err)
//	}
//	defer n.Stop()
//
//	// Store a value
//	if err := n.Set("key", []byte("value")); err != nil {
//	    log.Fatal(err)
//	}
//
//	// Retrieve a value
//	if value, ok := n.Get("key"); ok {
//	    fmt.Println(string(value))
//	}
//
// # Node Lifecycle
//
// A Node must be started before it can accept read/write operations.
// The lifecycle is: Stopped -> Running -> Stopped.
//
// # Thread Safety
//
// All operations on a Node are protected by a RWMutex, allowing concurrent
// reads while serializing writes.
package node
