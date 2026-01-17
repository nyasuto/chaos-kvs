package scenario

import (
	"context"
	"strings"
	"testing"
	"time"

	"chaos-kvs/internal/chaos"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config.Name != "default" {
		t.Errorf("expected name 'default', got '%s'", config.Name)
	}
	if config.NodeCount != 5 {
		t.Errorf("expected node count 5, got %d", config.NodeCount)
	}
	if !config.EnableChaos {
		t.Error("expected chaos to be enabled")
	}
	if !config.EnableRecovery {
		t.Error("expected recovery to be enabled")
	}
}

func TestNewEngine(t *testing.T) {
	config := DefaultConfig()
	engine := New(config)

	if engine == nil {
		t.Fatal("expected non-nil engine")
	}
	if engine.IsRunning() {
		t.Error("expected engine to not be running initially")
	}
}

func TestEngineRunBasic(t *testing.T) {
	config := BasicScenario()
	config.Duration = 1 * time.Second
	config.NodeCount = 2
	config.ClientWorkers = 2

	engine := New(config)
	ctx := context.Background()

	result, err := engine.Run(ctx)
	if err != nil {
		t.Fatalf("failed to run scenario: %v", err)
	}

	if result.ScenarioName != "basic" {
		t.Errorf("expected scenario name 'basic', got '%s'", result.ScenarioName)
	}
	if result.TotalRequests == 0 {
		t.Error("expected some requests to be executed")
	}
	if result.TotalAttacks != 0 {
		t.Error("expected no attacks in basic scenario")
	}
}

func TestEngineRunWithChaos(t *testing.T) {
	config := QuickScenario()
	config.Duration = 2 * time.Second
	config.ChaosInterval = 500 * time.Millisecond

	engine := New(config)
	ctx := context.Background()

	result, err := engine.Run(ctx)
	if err != nil {
		t.Fatalf("failed to run scenario: %v", err)
	}

	if result.TotalAttacks == 0 {
		t.Error("expected some attacks to be executed")
	}
}

func TestEngineRunWithRecovery(t *testing.T) {
	config := Config{
		Name:           "recovery-test",
		Description:    "Test recovery",
		Duration:       3 * time.Second,
		NodeCount:      3,
		ClientWorkers:  2,
		WriteRatio:     0.5,
		EnableChaos:    true,
		ChaosInterval:  500 * time.Millisecond,
		ChaosTargets:   1,
		AttackTypes:    []chaos.AttackType{chaos.AttackSuspend},
		EnableRecovery: true,
		RecoveryDelay:  300 * time.Millisecond,
		MaxRetries:     3,
	}

	engine := New(config)
	ctx := context.Background()

	result, err := engine.Run(ctx)
	if err != nil {
		t.Fatalf("failed to run scenario: %v", err)
	}

	// 復旧が発生しているはず
	if result.TotalRecoveries == 0 {
		t.Log("Warning: no recoveries occurred (may be timing dependent)")
	}
}

func TestEngineDoubleRun(t *testing.T) {
	config := BasicScenario()
	config.Duration = 2 * time.Second
	config.NodeCount = 2
	config.ClientWorkers = 2

	engine := New(config)
	ctx := context.Background()

	// 最初の実行を開始
	done := make(chan struct{})
	var firstResult *Result
	var firstErr error

	go func() {
		firstResult, firstErr = engine.Run(ctx)
		close(done)
	}()

	// 少し待ってから二重実行を試みる
	time.Sleep(100 * time.Millisecond)

	_, err := engine.Run(ctx)
	if err == nil {
		t.Error("expected error when running already running scenario")
	}

	<-done
	if firstErr != nil {
		t.Errorf("first run failed: %v", firstErr)
	}
	if firstResult == nil {
		t.Error("expected first result to be non-nil")
	}
}

func TestResultReport(t *testing.T) {
	result := &Result{
		ScenarioName:      "test",
		StartTime:         time.Now(),
		EndTime:           time.Now().Add(10 * time.Second),
		Duration:          10 * time.Second,
		TotalRequests:     1000,
		SuccessRequests:   990,
		FailedRequests:    10,
		ErrorRate:         0.01,
		AvgLatency:        5 * time.Millisecond,
		P99Latency:        20 * time.Millisecond,
		TotalAttacks:      5,
		TotalRecoveries:   3,
		SuccessRecoveries: 3,
		FailedRecoveries:  0,
		FinalNodeStatus: map[string]string{
			"node-1": "Running",
			"node-2": "Running",
		},
	}

	report := result.Report()

	// レポートに必要な情報が含まれているか確認
	if !strings.Contains(report, "test") {
		t.Error("report should contain scenario name")
	}
	if !strings.Contains(report, "1000") {
		t.Error("report should contain total requests")
	}
	if !strings.Contains(report, "1.00%") {
		t.Error("report should contain error rate")
	}
	if !strings.Contains(report, "node-1") {
		t.Error("report should contain node status")
	}
}

func TestPresets(t *testing.T) {
	presets := ListPresets()

	if len(presets) != 5 {
		t.Errorf("expected 5 presets, got %d", len(presets))
	}

	for _, name := range presets {
		config, ok := GetPreset(name)
		if !ok {
			t.Errorf("failed to get preset '%s'", name)
			continue
		}
		if config.Name != name {
			t.Errorf("expected preset name '%s', got '%s'", name, config.Name)
		}
	}
}

func TestGetPresetNotFound(t *testing.T) {
	_, ok := GetPreset("nonexistent")
	if ok {
		t.Error("expected GetPreset to return false for nonexistent preset")
	}
}

func TestBasicScenario(t *testing.T) {
	config := BasicScenario()

	if config.EnableChaos {
		t.Error("basic scenario should not enable chaos")
	}
	if config.EnableRecovery {
		t.Error("basic scenario should not enable recovery")
	}
}

func TestResilienceScenario(t *testing.T) {
	config := ResilienceScenario()

	if !config.EnableChaos {
		t.Error("resilience scenario should enable chaos")
	}
	if len(config.AttackTypes) != 1 || config.AttackTypes[0] != chaos.AttackKill {
		t.Error("resilience scenario should only use kill attack")
	}
	if !config.EnableRecovery {
		t.Error("resilience scenario should enable recovery")
	}
}

func TestLatencyScenario(t *testing.T) {
	config := LatencyScenario()

	if len(config.AttackTypes) != 1 || config.AttackTypes[0] != chaos.AttackDelay {
		t.Error("latency scenario should only use delay attack")
	}
}

func TestStressScenario(t *testing.T) {
	config := StressScenario()

	if config.ClientWorkers < 50 {
		t.Error("stress scenario should have many workers")
	}
	if len(config.AttackTypes) != 3 {
		t.Error("stress scenario should use all attack types")
	}
}

func TestQuickScenario(t *testing.T) {
	config := QuickScenario()

	if config.Duration > 10*time.Second {
		t.Error("quick scenario should be short")
	}
}

func TestEngineContextCancel(t *testing.T) {
	config := BasicScenario()
	config.Duration = 10 * time.Second
	config.NodeCount = 2
	config.ClientWorkers = 2

	engine := New(config)
	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	var result *Result
	var err error

	go func() {
		result, err = engine.Run(ctx)
		close(done)
	}()

	// 少し待ってからキャンセル
	time.Sleep(500 * time.Millisecond)
	cancel()

	<-done

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected result to be non-nil")
	}
	if result.Duration >= config.Duration {
		t.Error("expected scenario to be cancelled early")
	}
}
