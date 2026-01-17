package chaos

import (
	"context"
	"testing"
	"time"

	"chaos-kvs/internal/cluster"
	"chaos-kvs/internal/node"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config.Interval != 5*time.Second {
		t.Errorf("expected interval 5s, got %v", config.Interval)
	}
	if config.TargetCount != 1 {
		t.Errorf("expected target count 1, got %d", config.TargetCount)
	}
	if len(config.AttackTypes) != 3 {
		t.Errorf("expected 3 attack types, got %d", len(config.AttackTypes))
	}
}

func TestAttackTypeString(t *testing.T) {
	tests := []struct {
		attack   AttackType
		expected string
	}{
		{AttackKill, "kill"},
		{AttackSuspend, "suspend"},
		{AttackDelay, "delay"},
		{AttackType(99), "unknown"},
	}

	for _, tt := range tests {
		if got := tt.attack.String(); got != tt.expected {
			t.Errorf("AttackType(%d).String() = %s, want %s", tt.attack, got, tt.expected)
		}
	}
}

func TestNewMonkey(t *testing.T) {
	c := cluster.New()
	config := DefaultConfig()

	monkey := New(c, config)

	if monkey == nil {
		t.Fatal("expected non-nil monkey")
	}
	if monkey.IsRunning() {
		t.Error("expected monkey to not be running initially")
	}
}

func TestMonkeyStartStop(t *testing.T) {
	c := cluster.New()
	_ = c.CreateNodes(3, "node")
	_ = c.StartAll(context.Background())
	defer func() { _ = c.StopAll() }()

	config := DefaultConfig()
	config.Interval = 100 * time.Millisecond

	monkey := New(c, config)

	ctx := context.Background()
	monkey.Start(ctx)

	if !monkey.IsRunning() {
		t.Error("expected monkey to be running after Start")
	}

	// 少し待ってから停止
	time.Sleep(50 * time.Millisecond)

	monkey.Stop()

	if monkey.IsRunning() {
		t.Error("expected monkey to not be running after Stop")
	}
}

func TestMonkeyAttackKill(t *testing.T) {
	c := cluster.New()
	_ = c.CreateNodes(3, "node")
	_ = c.StartAll(context.Background())
	defer func() { _ = c.StopAll() }()

	config := DefaultConfig()
	config.Interval = 50 * time.Millisecond
	config.TargetCount = 1
	config.AttackTypes = []AttackType{AttackKill}

	monkey := New(c, config)

	ctx := context.Background()
	monkey.Start(ctx)

	// 攻撃が発生するまで待つ
	time.Sleep(100 * time.Millisecond)

	monkey.Stop()

	// 少なくとも1つのノードがStoppedになっているはず
	stoppedCount := 0
	for _, n := range c.Nodes() {
		if n.Status() == node.StatusStopped {
			stoppedCount++
		}
	}

	if stoppedCount == 0 {
		t.Error("expected at least one node to be killed")
	}
}

func TestMonkeyAttackSuspend(t *testing.T) {
	c := cluster.New()
	_ = c.CreateNodes(3, "node")
	_ = c.StartAll(context.Background())
	defer func() { _ = c.StopAll() }()

	config := DefaultConfig()
	config.Interval = 50 * time.Millisecond
	config.TargetCount = 1
	config.AttackTypes = []AttackType{AttackSuspend}
	config.SuspendTime = 200 * time.Millisecond

	monkey := New(c, config)

	ctx := context.Background()
	monkey.Start(ctx)

	// 攻撃が発生するまで待つ
	time.Sleep(100 * time.Millisecond)

	// 少なくとも1つのノードがSuspendedになっているはず
	suspendedCount := 0
	for _, n := range c.Nodes() {
		if n.Status() == node.StatusSuspended {
			suspendedCount++
		}
	}

	if suspendedCount == 0 {
		t.Error("expected at least one node to be suspended")
	}

	// 自動復旧を待つ
	time.Sleep(300 * time.Millisecond)

	monkey.Stop()

	// 停止後、すべてのsuspendedノードがresumeされているはず
	for _, n := range c.Nodes() {
		if n.Status() == node.StatusSuspended {
			t.Error("expected all suspended nodes to be resumed after Stop")
		}
	}
}

func TestMonkeyAttackDelay(t *testing.T) {
	c := cluster.New()
	_ = c.CreateNodes(1, "node")
	_ = c.StartAll(context.Background())
	defer func() { _ = c.StopAll() }()

	config := DefaultConfig()
	config.Interval = 50 * time.Millisecond
	config.TargetCount = 1
	config.AttackTypes = []AttackType{AttackDelay}
	config.DelayDuration = 50 * time.Millisecond

	monkey := New(c, config)

	ctx := context.Background()
	monkey.Start(ctx)

	// 攻撃が発生するまで待つ
	time.Sleep(100 * time.Millisecond)

	monkey.Stop()

	// ノードに遅延が設定されているはず
	nodes := c.Nodes()
	if len(nodes) > 0 {
		if nodes[0].Delay() != config.DelayDuration {
			t.Errorf("expected delay %v, got %v", config.DelayDuration, nodes[0].Delay())
		}
	}
}

func TestMonkeyAttackCount(t *testing.T) {
	c := cluster.New()
	_ = c.CreateNodes(3, "node")
	_ = c.StartAll(context.Background())
	defer func() { _ = c.StopAll() }()

	config := DefaultConfig()
	config.Interval = 30 * time.Millisecond
	config.TargetCount = 1
	config.AttackTypes = []AttackType{AttackDelay} // Delayはノードを停止しない

	monkey := New(c, config)

	ctx := context.Background()
	monkey.Start(ctx)

	// 複数回の攻撃を待つ
	time.Sleep(100 * time.Millisecond)

	monkey.Stop()

	if monkey.AttackCount() == 0 {
		t.Error("expected at least one attack to be executed")
	}
}

func TestMonkeyNoTargets(t *testing.T) {
	c := cluster.New()
	// ノードを追加しない

	config := DefaultConfig()
	config.Interval = 50 * time.Millisecond

	monkey := New(c, config)

	ctx := context.Background()
	monkey.Start(ctx)

	time.Sleep(100 * time.Millisecond)

	monkey.Stop()

	// ターゲットがないので攻撃は0回
	if monkey.AttackCount() != 0 {
		t.Errorf("expected 0 attacks with no targets, got %d", monkey.AttackCount())
	}
}

func TestMonkeySetConfig(t *testing.T) {
	c := cluster.New()
	config := DefaultConfig()

	monkey := New(c, config)

	newConfig := Config{
		Interval:    1 * time.Second,
		TargetCount: 5,
	}
	monkey.SetConfig(newConfig)

	if monkey.config.TargetCount != 5 {
		t.Errorf("expected target count 5, got %d", monkey.config.TargetCount)
	}
}
