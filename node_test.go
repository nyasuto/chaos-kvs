package main

import (
	"context"
	"sync"
	"testing"
)

func TestNewNode(t *testing.T) {
	node := NewNode("test-node-1")

	if node.ID != "test-node-1" {
		t.Errorf("expected ID 'test-node-1', got '%s'", node.ID)
	}

	if node.Status() != NodeStatusStopped {
		t.Errorf("expected status Stopped, got %v", node.Status())
	}
}

func TestNodeStartStop(t *testing.T) {
	node := NewNode("test-node-1")
	ctx := context.Background()

	// Start
	if err := node.Start(ctx); err != nil {
		t.Errorf("failed to start node: %v", err)
	}

	if node.Status() != NodeStatusRunning {
		t.Errorf("expected status Running, got %v", node.Status())
	}

	// Double start should fail
	if err := node.Start(ctx); err == nil {
		t.Error("expected error when starting already running node")
	}

	// Stop
	if err := node.Stop(); err != nil {
		t.Errorf("failed to stop node: %v", err)
	}

	if node.Status() != NodeStatusStopped {
		t.Errorf("expected status Stopped, got %v", node.Status())
	}

	// Double stop should fail
	if err := node.Stop(); err == nil {
		t.Error("expected error when stopping already stopped node")
	}
}

func TestNodeGetSet(t *testing.T) {
	node := NewNode("test-node-1")
	ctx := context.Background()

	// Set before start should fail
	if err := node.Set("key1", []byte("value1")); err == nil {
		t.Error("expected error when setting on stopped node")
	}

	// Get before start should return false
	if _, ok := node.Get("key1"); ok {
		t.Error("expected Get to return false on stopped node")
	}

	// Start node
	_ = node.Start(ctx)

	// Set
	if err := node.Set("key1", []byte("value1")); err != nil {
		t.Errorf("failed to set: %v", err)
	}

	// Get
	value, ok := node.Get("key1")
	if !ok {
		t.Error("expected Get to return true")
	}
	if string(value) != "value1" {
		t.Errorf("expected 'value1', got '%s'", string(value))
	}

	// Get non-existent key
	if _, ok := node.Get("nonexistent"); ok {
		t.Error("expected Get to return false for non-existent key")
	}
}

func TestNodeDelete(t *testing.T) {
	node := NewNode("test-node-1")
	ctx := context.Background()
	_ = node.Start(ctx)

	_ = node.Set("key1", []byte("value1"))

	if err := node.Delete("key1"); err != nil {
		t.Errorf("failed to delete: %v", err)
	}

	if _, ok := node.Get("key1"); ok {
		t.Error("expected key to be deleted")
	}
}

func TestNodeKeys(t *testing.T) {
	node := NewNode("test-node-1")
	ctx := context.Background()
	_ = node.Start(ctx)

	_ = node.Set("key1", []byte("value1"))
	_ = node.Set("key2", []byte("value2"))
	_ = node.Set("key3", []byte("value3"))

	keys := node.Keys()
	if len(keys) != 3 {
		t.Errorf("expected 3 keys, got %d", len(keys))
	}
}

func TestNodeSize(t *testing.T) {
	node := NewNode("test-node-1")
	ctx := context.Background()
	_ = node.Start(ctx)

	if node.Size() != 0 {
		t.Errorf("expected size 0, got %d", node.Size())
	}

	_ = node.Set("key1", []byte("value1"))
	_ = node.Set("key2", []byte("value2"))

	if node.Size() != 2 {
		t.Errorf("expected size 2, got %d", node.Size())
	}
}

func TestNodeConcurrentAccess(t *testing.T) {
	node := NewNode("test-node-1")
	ctx := context.Background()
	_ = node.Start(ctx)

	var wg sync.WaitGroup
	numGoroutines := 100

	// Concurrent writes
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			key := string(rune('a' + i%26))
			_ = node.Set(key, []byte("value"))
		}(i)
	}

	// Concurrent reads
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			key := string(rune('a' + i%26))
			node.Get(key)
		}(i)
	}

	wg.Wait()
}
