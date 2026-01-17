package recovery

import (
	"context"
	"testing"
	"time"

	"chaos-kvs/internal/cluster"
	"chaos-kvs/internal/node"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config.HealthCheckInterval != 1*time.Second {
		t.Errorf("expected interval 1s, got %v", config.HealthCheckInterval)
	}
	if config.RecoveryDelay != 2*time.Second {
		t.Errorf("expected recovery delay 2s, got %v", config.RecoveryDelay)
	}
	if config.MaxRetries != 3 {
		t.Errorf("expected max retries 3, got %d", config.MaxRetries)
	}
	if !config.AutoRestart {
		t.Error("expected auto restart to be true")
	}
	if !config.AutoResume {
		t.Error("expected auto resume to be true")
	}
}

func TestNewManager(t *testing.T) {
	c := cluster.New()
	config := DefaultConfig()

	manager := New(c, config)

	if manager == nil {
		t.Fatal("expected non-nil manager")
	}
	if manager.IsRunning() {
		t.Error("expected manager to not be running initially")
	}
}

func TestManagerStartStop(t *testing.T) {
	c := cluster.New()
	_ = c.CreateNodes(3, "node")
	_ = c.StartAll(context.Background())
	defer func() { _ = c.StopAll() }()

	config := DefaultConfig()
	config.HealthCheckInterval = 50 * time.Millisecond

	manager := New(c, config)

	ctx := context.Background()
	manager.Start(ctx)

	if !manager.IsRunning() {
		t.Error("expected manager to be running after Start")
	}

	// 少し待ってから停止
	time.Sleep(100 * time.Millisecond)

	manager.Stop()

	if manager.IsRunning() {
		t.Error("expected manager to not be running after Stop")
	}
}

func TestManagerAutoRestart(t *testing.T) {
	c := cluster.New()
	_ = c.CreateNodes(1, "node")
	_ = c.StartAll(context.Background())
	defer func() { _ = c.StopAll() }()

	config := DefaultConfig()
	config.HealthCheckInterval = 50 * time.Millisecond
	config.RecoveryDelay = 100 * time.Millisecond
	config.AutoRestart = true

	manager := New(c, config)

	ctx := context.Background()
	manager.Start(ctx)
	defer manager.Stop()

	// ノードを停止
	nodes := c.Nodes()
	if len(nodes) == 0 {
		t.Fatal("expected at least one node")
	}
	_ = nodes[0].Stop()

	if nodes[0].Status() != node.StatusStopped {
		t.Error("expected node to be stopped")
	}

	// 自動復旧を待つ
	time.Sleep(300 * time.Millisecond)

	if nodes[0].Status() != node.StatusRunning {
		t.Errorf("expected node to be running after recovery, got %v", nodes[0].Status())
	}
}

func TestManagerAutoResume(t *testing.T) {
	c := cluster.New()
	_ = c.CreateNodes(1, "node")
	_ = c.StartAll(context.Background())
	defer func() { _ = c.StopAll() }()

	config := DefaultConfig()
	config.HealthCheckInterval = 50 * time.Millisecond
	config.RecoveryDelay = 100 * time.Millisecond
	config.AutoResume = true

	manager := New(c, config)

	ctx := context.Background()
	manager.Start(ctx)
	defer manager.Stop()

	// ノードをsuspend
	nodes := c.Nodes()
	if len(nodes) == 0 {
		t.Fatal("expected at least one node")
	}
	_ = nodes[0].Suspend()

	if nodes[0].Status() != node.StatusSuspended {
		t.Error("expected node to be suspended")
	}

	// 自動復旧を待つ
	time.Sleep(300 * time.Millisecond)

	if nodes[0].Status() != node.StatusRunning {
		t.Errorf("expected node to be running after recovery, got %v", nodes[0].Status())
	}
}

func TestManagerClearDelay(t *testing.T) {
	c := cluster.New()
	_ = c.CreateNodes(1, "node")
	_ = c.StartAll(context.Background())
	defer func() { _ = c.StopAll() }()

	config := DefaultConfig()
	config.HealthCheckInterval = 50 * time.Millisecond
	config.ClearDelay = true

	manager := New(c, config)

	ctx := context.Background()
	manager.Start(ctx)
	defer manager.Stop()

	// ノードに遅延を設定
	nodes := c.Nodes()
	if len(nodes) == 0 {
		t.Fatal("expected at least one node")
	}
	nodes[0].SetDelay(100 * time.Millisecond)

	if nodes[0].Delay() != 100*time.Millisecond {
		t.Error("expected delay to be set")
	}

	// 遅延クリアを待つ
	time.Sleep(150 * time.Millisecond)

	if nodes[0].Delay() != 0 {
		t.Error("expected delay to be cleared")
	}
}

func TestManagerMaxRetries(t *testing.T) {
	c := cluster.New()
	_ = c.CreateNodes(1, "node")
	_ = c.StartAll(context.Background())
	defer func() { _ = c.StopAll() }()

	config := DefaultConfig()
	config.HealthCheckInterval = 30 * time.Millisecond
	config.RecoveryDelay = 50 * time.Millisecond
	config.MaxRetries = 2
	config.AutoRestart = true

	manager := New(c, config)

	ctx := context.Background()
	manager.Start(ctx)
	defer manager.Stop()

	nodes := c.Nodes()
	if len(nodes) == 0 {
		t.Fatal("expected at least one node")
	}

	// ノードを停止（再起動されても再度停止される場合をシミュレート）
	_ = nodes[0].Stop()

	// 複数回のリトライを待つ
	time.Sleep(400 * time.Millisecond)

	stats := manager.Stats()
	if stats.TotalRecoveries == 0 {
		t.Error("expected at least one recovery attempt")
	}
}

func TestManagerStats(t *testing.T) {
	c := cluster.New()
	_ = c.CreateNodes(1, "node")
	_ = c.StartAll(context.Background())
	defer func() { _ = c.StopAll() }()

	config := DefaultConfig()
	config.HealthCheckInterval = 50 * time.Millisecond
	config.RecoveryDelay = 100 * time.Millisecond

	manager := New(c, config)

	ctx := context.Background()
	manager.Start(ctx)
	defer manager.Stop()

	// ノードをsuspend
	nodes := c.Nodes()
	_ = nodes[0].Suspend()

	// 復旧を待つ
	time.Sleep(300 * time.Millisecond)

	stats := manager.Stats()
	if stats.TotalRecoveries == 0 {
		t.Error("expected total recoveries > 0")
	}
}

func TestManagerSetConfig(t *testing.T) {
	c := cluster.New()
	config := DefaultConfig()

	manager := New(c, config)

	newConfig := Config{
		HealthCheckInterval: 5 * time.Second,
		MaxRetries:          10,
	}
	manager.SetConfig(newConfig)

	if manager.config.MaxRetries != 10 {
		t.Errorf("expected max retries 10, got %d", manager.config.MaxRetries)
	}
}

func TestManagerResetStats(t *testing.T) {
	c := cluster.New()
	config := DefaultConfig()

	manager := New(c, config)

	// 手動で統計を設定
	manager.mu.Lock()
	manager.stats.TotalRecoveries = 10
	manager.stats.SuccessRecoveries = 8
	manager.mu.Unlock()

	manager.ResetStats()

	stats := manager.Stats()
	if stats.TotalRecoveries != 0 {
		t.Error("expected stats to be reset")
	}
}

func TestManagerDisabledAutoRestart(t *testing.T) {
	c := cluster.New()
	_ = c.CreateNodes(1, "node")
	_ = c.StartAll(context.Background())
	defer func() { _ = c.StopAll() }()

	config := DefaultConfig()
	config.HealthCheckInterval = 50 * time.Millisecond
	config.RecoveryDelay = 50 * time.Millisecond
	config.AutoRestart = false

	manager := New(c, config)

	ctx := context.Background()
	manager.Start(ctx)
	defer manager.Stop()

	// ノードを停止
	nodes := c.Nodes()
	_ = nodes[0].Stop()

	// 待機しても復旧しないはず
	time.Sleep(200 * time.Millisecond)

	if nodes[0].Status() != node.StatusStopped {
		t.Error("expected node to remain stopped when AutoRestart is disabled")
	}
}

func TestManagerDisabledAutoResume(t *testing.T) {
	c := cluster.New()
	_ = c.CreateNodes(1, "node")
	_ = c.StartAll(context.Background())
	defer func() { _ = c.StopAll() }()

	config := DefaultConfig()
	config.HealthCheckInterval = 50 * time.Millisecond
	config.RecoveryDelay = 50 * time.Millisecond
	config.AutoResume = false

	manager := New(c, config)

	ctx := context.Background()
	manager.Start(ctx)
	defer manager.Stop()

	// ノードをsuspend
	nodes := c.Nodes()
	_ = nodes[0].Suspend()

	// 待機しても復旧しないはず
	time.Sleep(200 * time.Millisecond)

	if nodes[0].Status() != node.StatusSuspended {
		t.Error("expected node to remain suspended when AutoResume is disabled")
	}
}
