package main

import (
	"context"
	"fmt"
	"sync"
	"testing"
)

func TestNewCluster(t *testing.T) {
	cluster := NewCluster()

	if cluster.Size() != 0 {
		t.Errorf("expected size 0, got %d", cluster.Size())
	}
}

func TestClusterAddRemoveNode(t *testing.T) {
	cluster := NewCluster()
	node := NewNode("test-node-1")

	// Add
	if err := cluster.AddNode(node); err != nil {
		t.Errorf("failed to add node: %v", err)
	}

	if cluster.Size() != 1 {
		t.Errorf("expected size 1, got %d", cluster.Size())
	}

	// Add duplicate should fail
	if err := cluster.AddNode(node); err == nil {
		t.Error("expected error when adding duplicate node")
	}

	// Get
	retrieved, ok := cluster.GetNode("test-node-1")
	if !ok {
		t.Error("expected to find node")
	}
	if retrieved.ID != node.ID {
		t.Errorf("expected ID %s, got %s", node.ID, retrieved.ID)
	}

	// Remove
	if err := cluster.RemoveNode("test-node-1"); err != nil {
		t.Errorf("failed to remove node: %v", err)
	}

	if cluster.Size() != 0 {
		t.Errorf("expected size 0, got %d", cluster.Size())
	}

	// Remove non-existent should fail
	if err := cluster.RemoveNode("test-node-1"); err == nil {
		t.Error("expected error when removing non-existent node")
	}
}

func TestClusterStartStopAll(t *testing.T) {
	cluster := NewCluster()
	ctx := context.Background()

	_ = cluster.CreateNodes(5, "node")

	// Start all
	if err := cluster.StartAll(ctx); err != nil {
		t.Errorf("failed to start all: %v", err)
	}

	if cluster.RunningCount() != 5 {
		t.Errorf("expected 5 running, got %d", cluster.RunningCount())
	}

	if cluster.StoppedCount() != 0 {
		t.Errorf("expected 0 stopped, got %d", cluster.StoppedCount())
	}

	// Stop all
	if err := cluster.StopAll(); err != nil {
		t.Errorf("failed to stop all: %v", err)
	}

	if cluster.RunningCount() != 0 {
		t.Errorf("expected 0 running, got %d", cluster.RunningCount())
	}

	if cluster.StoppedCount() != 5 {
		t.Errorf("expected 5 stopped, got %d", cluster.StoppedCount())
	}
}

func TestClusterNodes(t *testing.T) {
	cluster := NewCluster()

	_ = cluster.CreateNodes(3, "node")

	nodes := cluster.Nodes()
	if len(nodes) != 3 {
		t.Errorf("expected 3 nodes, got %d", len(nodes))
	}
}

func TestClusterCreateNodes(t *testing.T) {
	cluster := NewCluster()

	if err := cluster.CreateNodes(10, "test"); err != nil {
		t.Errorf("failed to create nodes: %v", err)
	}

	if cluster.Size() != 10 {
		t.Errorf("expected size 10, got %d", cluster.Size())
	}

	// Verify nodes exist
	for i := 1; i <= 10; i++ {
		nodeID := fmt.Sprintf("test-%d", i)
		if _, ok := cluster.GetNode(nodeID); !ok {
			t.Errorf("expected node %s to exist", nodeID)
		}
	}
}

func TestClusterConcurrentAccess(t *testing.T) {
	cluster := NewCluster()
	ctx := context.Background()

	_ = cluster.CreateNodes(10, "node")
	_ = cluster.StartAll(ctx)

	var wg sync.WaitGroup

	// Concurrent reads
	for range 50 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = cluster.Nodes()
			_ = cluster.Size()
			_ = cluster.RunningCount()
		}()
	}

	wg.Wait()
	_ = cluster.StopAll()
}
