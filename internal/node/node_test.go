package node

import (
	"context"
	"sync"
	"testing"
)

func TestNewNode(t *testing.T) {
	n := New("test-node-1")

	if n.ID() != "test-node-1" {
		t.Errorf("expected ID 'test-node-1', got '%s'", n.ID())
	}

	if n.Status() != StatusStopped {
		t.Errorf("expected status Stopped, got %v", n.Status())
	}
}

func TestNodeStartStop(t *testing.T) {
	n := New("test-node-1")
	ctx := context.Background()

	// Start
	if err := n.Start(ctx); err != nil {
		t.Errorf("failed to start node: %v", err)
	}

	if n.Status() != StatusRunning {
		t.Errorf("expected status Running, got %v", n.Status())
	}

	// Double start should fail
	if err := n.Start(ctx); err == nil {
		t.Error("expected error when starting already running node")
	}

	// Stop
	if err := n.Stop(); err != nil {
		t.Errorf("failed to stop node: %v", err)
	}

	if n.Status() != StatusStopped {
		t.Errorf("expected status Stopped, got %v", n.Status())
	}

	// Double stop should fail
	if err := n.Stop(); err == nil {
		t.Error("expected error when stopping already stopped node")
	}
}

func TestNodeGetSet(t *testing.T) {
	n := New("test-node-1")
	ctx := context.Background()

	// Set before start should fail
	if err := n.Set("key1", []byte("value1")); err == nil {
		t.Error("expected error when setting on stopped node")
	}

	// Get before start should return false
	if _, ok := n.Get("key1"); ok {
		t.Error("expected Get to return false on stopped node")
	}

	// Start node
	_ = n.Start(ctx)

	// Set
	if err := n.Set("key1", []byte("value1")); err != nil {
		t.Errorf("failed to set: %v", err)
	}

	// Get
	value, ok := n.Get("key1")
	if !ok {
		t.Error("expected Get to return true")
	}
	if string(value) != "value1" {
		t.Errorf("expected 'value1', got '%s'", string(value))
	}

	// Get non-existent key
	if _, ok := n.Get("nonexistent"); ok {
		t.Error("expected Get to return false for non-existent key")
	}
}

func TestNodeDelete(t *testing.T) {
	n := New("test-node-1")
	ctx := context.Background()
	_ = n.Start(ctx)

	_ = n.Set("key1", []byte("value1"))

	if err := n.Delete("key1"); err != nil {
		t.Errorf("failed to delete: %v", err)
	}

	if _, ok := n.Get("key1"); ok {
		t.Error("expected key to be deleted")
	}
}

func TestNodeKeys(t *testing.T) {
	n := New("test-node-1")
	ctx := context.Background()
	_ = n.Start(ctx)

	_ = n.Set("key1", []byte("value1"))
	_ = n.Set("key2", []byte("value2"))
	_ = n.Set("key3", []byte("value3"))

	keys := n.Keys()
	if len(keys) != 3 {
		t.Errorf("expected 3 keys, got %d", len(keys))
	}
}

func TestNodeSize(t *testing.T) {
	n := New("test-node-1")
	ctx := context.Background()
	_ = n.Start(ctx)

	if n.Size() != 0 {
		t.Errorf("expected size 0, got %d", n.Size())
	}

	_ = n.Set("key1", []byte("value1"))
	_ = n.Set("key2", []byte("value2"))

	if n.Size() != 2 {
		t.Errorf("expected size 2, got %d", n.Size())
	}
}

func TestNodeConcurrentAccess(t *testing.T) {
	n := New("test-node-1")
	ctx := context.Background()
	_ = n.Start(ctx)

	var wg sync.WaitGroup
	numGoroutines := 100

	// Concurrent writes
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			key := string(rune('a' + i%26))
			_ = n.Set(key, []byte("value"))
		}(i)
	}

	// Concurrent reads
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			key := string(rune('a' + i%26))
			n.Get(key)
		}(i)
	}

	wg.Wait()
}
