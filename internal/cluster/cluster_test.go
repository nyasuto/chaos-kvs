package cluster

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"chaos-kvs/internal/node"
)

func TestNewCluster(t *testing.T) {
	c := New()

	if c.Size() != 0 {
		t.Errorf("expected size 0, got %d", c.Size())
	}
}

func TestClusterAddRemoveNode(t *testing.T) {
	c := New()
	n := node.New("test-node-1")

	// Add
	if err := c.AddNode(n); err != nil {
		t.Errorf("failed to add node: %v", err)
	}

	if c.Size() != 1 {
		t.Errorf("expected size 1, got %d", c.Size())
	}

	// Add duplicate should fail
	if err := c.AddNode(n); err == nil {
		t.Error("expected error when adding duplicate node")
	}

	// Get
	retrieved, ok := c.GetNode("test-node-1")
	if !ok {
		t.Error("expected to find node")
	}
	if retrieved.ID() != n.ID() {
		t.Errorf("expected ID %s, got %s", n.ID(), retrieved.ID())
	}

	// Remove
	if err := c.RemoveNode("test-node-1"); err != nil {
		t.Errorf("failed to remove node: %v", err)
	}

	if c.Size() != 0 {
		t.Errorf("expected size 0, got %d", c.Size())
	}

	// Remove non-existent should fail
	if err := c.RemoveNode("test-node-1"); err == nil {
		t.Error("expected error when removing non-existent node")
	}
}

func TestClusterStartStopAll(t *testing.T) {
	c := New()
	ctx := context.Background()

	_ = c.CreateNodes(5, "node")

	// Start all
	if err := c.StartAll(ctx); err != nil {
		t.Errorf("failed to start all: %v", err)
	}

	if c.RunningCount() != 5 {
		t.Errorf("expected 5 running, got %d", c.RunningCount())
	}

	if c.StoppedCount() != 0 {
		t.Errorf("expected 0 stopped, got %d", c.StoppedCount())
	}

	// Stop all
	if err := c.StopAll(); err != nil {
		t.Errorf("failed to stop all: %v", err)
	}

	if c.RunningCount() != 0 {
		t.Errorf("expected 0 running, got %d", c.RunningCount())
	}

	if c.StoppedCount() != 5 {
		t.Errorf("expected 5 stopped, got %d", c.StoppedCount())
	}
}

func TestClusterNodes(t *testing.T) {
	c := New()

	_ = c.CreateNodes(3, "node")

	nodes := c.Nodes()
	if len(nodes) != 3 {
		t.Errorf("expected 3 nodes, got %d", len(nodes))
	}
}

func TestClusterCreateNodes(t *testing.T) {
	c := New()

	if err := c.CreateNodes(10, "test"); err != nil {
		t.Errorf("failed to create nodes: %v", err)
	}

	if c.Size() != 10 {
		t.Errorf("expected size 10, got %d", c.Size())
	}

	// Verify nodes exist
	for i := 1; i <= 10; i++ {
		nodeID := fmt.Sprintf("test-%d", i)
		if _, ok := c.GetNode(nodeID); !ok {
			t.Errorf("expected node %s to exist", nodeID)
		}
	}
}

func TestClusterConcurrentAccess(t *testing.T) {
	c := New()
	ctx := context.Background()

	_ = c.CreateNodes(10, "node")
	_ = c.StartAll(ctx)

	var wg sync.WaitGroup

	// Concurrent reads
	for range 50 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = c.Nodes()
			_ = c.Size()
			_ = c.RunningCount()
		}()
	}

	wg.Wait()
	_ = c.StopAll()
}
