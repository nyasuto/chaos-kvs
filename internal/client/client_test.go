package client

import (
	"context"
	"testing"
	"time"

	"chaos-kvs/internal/cluster"
)

func TestDefaultClientConfig(t *testing.T) {
	config := DefaultConfig()

	if config.WriteRatio != 0.5 {
		t.Errorf("expected WriteRatio 0.5, got %f", config.WriteRatio)
	}
	if config.KeyRange != 10000 {
		t.Errorf("expected KeyRange 10000, got %d", config.KeyRange)
	}
}

func TestNewClient(t *testing.T) {
	c := cluster.New()
	config := DefaultConfig()
	client := New(c, config)

	if client.IsRunning() {
		t.Error("expected client to not be running initially")
	}
}

func TestClientStartStop(t *testing.T) {
	c := cluster.New()
	_ = c.CreateNodes(3, "node")
	ctx := context.Background()
	_ = c.StartAll(ctx)
	defer func() { _ = c.StopAll() }()

	config := DefaultConfig()
	client := New(c, config)

	client.Start(ctx)
	if !client.IsRunning() {
		t.Error("expected client to be running after Start")
	}

	// Give it time to run some requests
	time.Sleep(50 * time.Millisecond)

	client.Stop()
	if client.IsRunning() {
		t.Error("expected client to not be running after Stop")
	}

	// Should have some metrics
	if client.Metrics().TotalRequests() == 0 {
		t.Error("expected some requests to be recorded")
	}
}

func TestClientRunFor(t *testing.T) {
	c := cluster.New()
	_ = c.CreateNodes(3, "node")
	ctx := context.Background()
	_ = c.StartAll(ctx)
	defer func() { _ = c.StopAll() }()

	config := DefaultConfig()
	client := New(c, config)

	snapshot := client.RunFor(ctx, 100*time.Millisecond)

	if snapshot.TotalRequests == 0 {
		t.Error("expected some requests")
	}
	if snapshot.Elapsed < 100*time.Millisecond {
		t.Errorf("expected at least 100ms elapsed, got %v", snapshot.Elapsed)
	}
}

func TestClientRunRequests(t *testing.T) {
	c := cluster.New()
	_ = c.CreateNodes(3, "node")
	ctx := context.Background()
	_ = c.StartAll(ctx)
	defer func() { _ = c.StopAll() }()

	config := DefaultConfig()
	config.RequestsLimit = 100
	client := New(c, config)

	snapshot := client.RunRequests(ctx, 100)

	// Should have at least 100 requests (may have more due to race)
	if snapshot.TotalRequests < 100 {
		t.Errorf("expected at least 100 requests, got %d", snapshot.TotalRequests)
	}
}

func TestClientWithNoNodes(t *testing.T) {
	c := cluster.New()
	config := DefaultConfig()
	client := New(c, config)

	ctx := context.Background()
	client.Start(ctx)

	time.Sleep(50 * time.Millisecond)
	client.Stop()

	// Should have 0 requests since there are no nodes
	if client.Metrics().TotalRequests() != 0 {
		t.Errorf("expected 0 requests with no nodes, got %d", client.Metrics().TotalRequests())
	}
}
