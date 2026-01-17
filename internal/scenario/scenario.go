package scenario

import (
	"context"
	"fmt"
	"sync"
	"time"

	"chaos-kvs/internal/chaos"
	"chaos-kvs/internal/client"
	"chaos-kvs/internal/cluster"
	"chaos-kvs/internal/events"
	"chaos-kvs/internal/logger"
	"chaos-kvs/internal/metrics"
	"chaos-kvs/internal/recovery"
)

// Config はシナリオの設定
type Config struct {
	Name        string        // シナリオ名
	Description string        // 説明
	Duration    time.Duration // 実行時間
	NodeCount   int           // ノード数

	// クライアント設定
	ClientWorkers int     // ワーカー数
	WriteRatio    float64 // 書き込み比率

	// カオス設定
	EnableChaos   bool               // カオス注入を有効化
	ChaosInterval time.Duration      // 攻撃間隔
	ChaosTargets  int                // 同時攻撃対象数
	AttackTypes   []chaos.AttackType // 有効な攻撃タイプ

	// 復旧設定
	EnableRecovery bool          // 復旧を有効化
	RecoveryDelay  time.Duration // 復旧までの待機時間
	MaxRetries     int           // 最大リトライ回数
}

// DefaultConfig はデフォルト設定を返す
func DefaultConfig() Config {
	return Config{
		Name:           "default",
		Description:    "Default scenario",
		Duration:       10 * time.Second,
		NodeCount:      5,
		ClientWorkers:  10,
		WriteRatio:     0.5,
		EnableChaos:    true,
		ChaosInterval:  2 * time.Second,
		ChaosTargets:   1,
		AttackTypes:    []chaos.AttackType{chaos.AttackKill, chaos.AttackSuspend, chaos.AttackDelay},
		EnableRecovery: true,
		RecoveryDelay:  1 * time.Second,
		MaxRetries:     3,
	}
}

// Result はシナリオ実行結果
type Result struct {
	ScenarioName string
	StartTime    time.Time
	EndTime      time.Time
	Duration     time.Duration

	// メトリクス
	TotalRequests   uint64
	SuccessRequests uint64
	FailedRequests  uint64
	ErrorRate       float64
	AvgLatency      time.Duration
	P99Latency      time.Duration

	// カオス統計
	TotalAttacks uint64

	// 復旧統計
	TotalRecoveries   uint64
	SuccessRecoveries uint64
	FailedRecoveries  uint64

	// ノード状態
	FinalNodeStatus map[string]string
}

// Engine はシナリオ実行エンジン
type Engine struct {
	config   Config
	eventBus *events.Bus

	cluster  *cluster.Cluster
	client   *client.Client
	monkey   *chaos.Monkey
	recovery *recovery.Manager

	mu      sync.RWMutex
	running bool
}

// New は新しいEngineを作成する
func New(config Config) *Engine {
	return &Engine{
		config: config,
	}
}

// SetEventBus はイベントバスを設定する
func (e *Engine) SetEventBus(bus *events.Bus) {
	e.eventBus = bus
}

// Run はシナリオを実行する
func (e *Engine) Run(ctx context.Context) (*Result, error) {
	e.mu.Lock()
	if e.running {
		e.mu.Unlock()
		return nil, fmt.Errorf("scenario is already running")
	}
	e.running = true
	e.mu.Unlock()

	defer func() {
		e.mu.Lock()
		e.running = false
		e.mu.Unlock()
	}()

	logger.Info("", "=== Scenario '%s' started ===", e.config.Name)
	logger.Info("", "Description: %s", e.config.Description)

	result := &Result{
		ScenarioName: e.config.Name,
		StartTime:    time.Now(),
	}

	// セットアップ
	if err := e.setup(ctx); err != nil {
		return nil, fmt.Errorf("setup failed: %w", err)
	}
	defer e.teardown()

	// シナリオ実行
	scenarioCtx, cancel := context.WithTimeout(ctx, e.config.Duration)
	defer cancel()

	e.runScenario(scenarioCtx)

	// 結果収集
	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)
	e.collectResults(result)

	logger.Info("", "=== Scenario '%s' completed ===", e.config.Name)

	return result, nil
}

// setup はシナリオ実行前のセットアップ
func (e *Engine) setup(ctx context.Context) error {
	// クラスタ作成
	e.cluster = cluster.New()
	if err := e.cluster.CreateNodes(e.config.NodeCount, "node"); err != nil {
		return fmt.Errorf("failed to create nodes: %w", err)
	}
	if err := e.cluster.StartAll(ctx); err != nil {
		return fmt.Errorf("failed to start nodes: %w", err)
	}

	// クライアント
	clientConfig := client.DefaultConfig()
	clientConfig.NumWorkers = e.config.ClientWorkers
	clientConfig.WriteRatio = e.config.WriteRatio
	e.client = client.New(e.cluster, clientConfig)

	// カオスモンキー
	if e.config.EnableChaos {
		chaosConfig := chaos.DefaultConfig()
		chaosConfig.Interval = e.config.ChaosInterval
		chaosConfig.TargetCount = e.config.ChaosTargets
		chaosConfig.AttackTypes = e.config.AttackTypes
		e.monkey = chaos.New(e.cluster, chaosConfig)
		if e.eventBus != nil {
			e.monkey.SetEventBus(e.eventBus)
		}
	}

	// 復旧マネージャー
	if e.config.EnableRecovery {
		recoveryConfig := recovery.DefaultConfig()
		recoveryConfig.RecoveryDelay = e.config.RecoveryDelay
		recoveryConfig.MaxRetries = e.config.MaxRetries
		e.recovery = recovery.New(e.cluster, recoveryConfig)
		if e.eventBus != nil {
			e.recovery.SetEventBus(e.eventBus)
		}
	}

	return nil
}

// teardown はシナリオ実行後のクリーンアップ
func (e *Engine) teardown() {
	if e.client != nil {
		e.client.Stop()
	}
	if e.monkey != nil {
		e.monkey.Stop()
	}
	if e.recovery != nil {
		e.recovery.Stop()
	}
	if e.cluster != nil {
		_ = e.cluster.StopAll()
	}
}

// runScenario はシナリオのメイン処理
func (e *Engine) runScenario(ctx context.Context) {
	// クライアント開始
	e.client.Start(ctx)

	// カオス開始
	if e.monkey != nil {
		e.monkey.Start(ctx)
	}

	// 復旧開始
	if e.recovery != nil {
		e.recovery.Start(ctx)
	}

	// 終了まで待機
	<-ctx.Done()

	logger.Info("", "Scenario duration completed, stopping components...")
}

// collectResults は結果を収集する
func (e *Engine) collectResults(result *Result) {
	// メトリクススナップショット
	snapshot := e.client.Metrics().Snapshot()
	result.TotalRequests = snapshot.TotalRequests
	result.SuccessRequests = snapshot.SuccessRequests
	result.FailedRequests = snapshot.FailedRequests
	result.ErrorRate = snapshot.ErrorRate
	result.AvgLatency = snapshot.AverageLatency
	result.P99Latency = snapshot.P99Latency

	// カオス統計
	if e.monkey != nil {
		result.TotalAttacks = e.monkey.AttackCount()
	}

	// 復旧統計
	if e.recovery != nil {
		stats := e.recovery.Stats()
		result.TotalRecoveries = stats.TotalRecoveries
		result.SuccessRecoveries = stats.SuccessRecoveries
		result.FailedRecoveries = stats.FailedRecoveries
	}

	// ノード状態
	result.FinalNodeStatus = make(map[string]string)
	for _, n := range e.cluster.Nodes() {
		result.FinalNodeStatus[n.ID()] = n.Status().String()
	}
}

// Report は結果をフォーマットして返す
func (r *Result) Report() string {
	report := fmt.Sprintf(`
================================================================================
                         SCENARIO REPORT: %s
================================================================================

EXECUTION SUMMARY
-----------------
  Start Time:     %s
  End Time:       %s
  Duration:       %v

TRAFFIC METRICS
---------------
  Total Requests:   %d
  Success:          %d
  Failed:           %d
  Error Rate:       %.2f%%
  Avg Latency:      %v
  P99 Latency:      %v

CHAOS STATISTICS
----------------
  Total Attacks:    %d

RECOVERY STATISTICS
-------------------
  Total Recoveries:   %d
  Successful:         %d
  Failed:             %d

FINAL NODE STATUS
-----------------
`,
		r.ScenarioName,
		r.StartTime.Format("2006-01-02 15:04:05"),
		r.EndTime.Format("2006-01-02 15:04:05"),
		r.Duration.Round(time.Millisecond),
		r.TotalRequests,
		r.SuccessRequests,
		r.FailedRequests,
		r.ErrorRate*100,
		r.AvgLatency.Round(time.Microsecond),
		r.P99Latency.Round(time.Microsecond),
		r.TotalAttacks,
		r.TotalRecoveries,
		r.SuccessRecoveries,
		r.FailedRecoveries,
	)

	for nodeID, status := range r.FinalNodeStatus {
		report += fmt.Sprintf("  %-20s %s\n", nodeID+":", status)
	}

	report += "\n================================================================================"

	return report
}

// IsRunning は実行中かどうかを返す
func (e *Engine) IsRunning() bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.running
}

// ChaosStats はカオス統計を返す
func (e *Engine) ChaosStats() *chaos.Stats {
	e.mu.RLock()
	defer e.mu.RUnlock()
	if e.monkey == nil {
		return nil
	}
	stats := e.monkey.Stats()
	return &stats
}

// RecoveryStats は復旧統計を返す
func (e *Engine) RecoveryStats() *recovery.Stats {
	e.mu.RLock()
	defer e.mu.RUnlock()
	if e.recovery == nil {
		return nil
	}
	stats := e.recovery.Stats()
	return &stats
}

// Metrics はクライアントメトリクスを返す
func (e *Engine) Metrics() *metrics.Snapshot {
	e.mu.RLock()
	defer e.mu.RUnlock()
	if e.client == nil {
		return nil
	}
	snapshot := e.client.Metrics().Snapshot()
	return &snapshot
}

// Cluster はクラスタを返す
func (e *Engine) Cluster() *cluster.Cluster {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.cluster
}
