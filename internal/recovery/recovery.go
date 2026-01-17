package recovery

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"chaos-kvs/internal/cluster"
	"chaos-kvs/internal/events"
	"chaos-kvs/internal/logger"
	"chaos-kvs/internal/node"
)

// Config はRecoveryManagerの設定
type Config struct {
	HealthCheckInterval time.Duration // ヘルスチェック間隔
	RecoveryDelay       time.Duration // 復旧までの待機時間
	MaxRetries          int           // 最大リトライ回数（0で無制限）
	AutoRestart         bool          // 停止ノードの自動再起動
	AutoResume          bool          // 一時停止ノードの自動再開
	ClearDelay          bool          // 遅延設定のクリア
}

// DefaultConfig はデフォルト設定を返す
func DefaultConfig() Config {
	return Config{
		HealthCheckInterval: 1 * time.Second,
		RecoveryDelay:       2 * time.Second,
		MaxRetries:          3,
		AutoRestart:         true,
		AutoResume:          true,
		ClearDelay:          true,
	}
}

// NodeState はノードの状態追跡
type NodeState struct {
	LastSeen    time.Time
	FailedAt    time.Time
	RetryCount  int
	IsRecovered bool
}

// Stats は復旧統計
type Stats struct {
	TotalRecoveries   uint64
	SuccessRecoveries uint64
	FailedRecoveries  uint64
	CurrentlyFailed   int
}

// Manager は障害からの復旧を管理する
type Manager struct {
	config   Config
	cluster  *cluster.Cluster
	eventBus *events.Bus

	running atomic.Bool
	ctx     context.Context
	cancel  context.CancelFunc
	wg      sync.WaitGroup

	mu         sync.RWMutex
	nodeStates map[string]*NodeState
	stats      Stats
}

// New は新しいRecoveryManagerを作成する
func New(c *cluster.Cluster, config Config) *Manager {
	return &Manager{
		config:     config,
		cluster:    c,
		nodeStates: make(map[string]*NodeState),
	}
}

// SetEventBus はイベントバスを設定する
func (m *Manager) SetEventBus(bus *events.Bus) {
	m.eventBus = bus
}

// publishEvent はイベントを発行する
func (m *Manager) publishEvent(event events.Event) {
	if m.eventBus != nil {
		m.eventBus.Publish(event)
	}
}

// Start は復旧マネージャーを開始する
func (m *Manager) Start(ctx context.Context) {
	if m.running.Swap(true) {
		return
	}

	m.ctx, m.cancel = context.WithCancel(ctx)

	m.wg.Add(1)
	go m.healthCheckLoop()

	logger.Info("", "RecoveryManager started (interval: %v, delay: %v)",
		m.config.HealthCheckInterval, m.config.RecoveryDelay)
}

// Stop は復旧マネージャーを停止する
func (m *Manager) Stop() {
	if !m.running.Swap(false) {
		return
	}

	m.cancel()
	m.wg.Wait()

	m.mu.RLock()
	stats := m.stats
	m.mu.RUnlock()

	logger.Info("", "RecoveryManager stopped (recoveries: %d success, %d failed)",
		stats.SuccessRecoveries, stats.FailedRecoveries)
}

// healthCheckLoop は定期的にヘルスチェックを実行する
func (m *Manager) healthCheckLoop() {
	defer m.wg.Done()

	ticker := time.NewTicker(m.config.HealthCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-m.ctx.Done():
			return
		case <-ticker.C:
			m.checkAndRecover()
		}
	}
}

// checkAndRecover は全ノードをチェックし、必要に応じて復旧する
func (m *Manager) checkAndRecover() {
	nodes := m.cluster.Nodes()
	now := time.Now()

	for _, n := range nodes {
		m.checkNode(n, now)
	}
}

// checkNode は個々のノードをチェックする
func (m *Manager) checkNode(n *node.Node, now time.Time) {
	nodeID := n.ID()
	status := n.Status()

	m.mu.Lock()
	state, exists := m.nodeStates[nodeID]
	if !exists {
		state = &NodeState{LastSeen: now}
		m.nodeStates[nodeID] = state
	}
	m.mu.Unlock()

	switch status {
	case node.StatusRunning:
		m.handleRunningNode(n, state, now)
	case node.StatusStopped:
		m.handleStoppedNode(n, state, now)
	case node.StatusSuspended:
		m.handleSuspendedNode(n, state, now)
	}
}

// handleRunningNode は稼働中のノードを処理する
func (m *Manager) handleRunningNode(n *node.Node, state *NodeState, now time.Time) {
	m.mu.Lock()

	// 遅延クリア
	if m.config.ClearDelay && n.Delay() > 0 {
		n.SetDelay(0)
		logger.Info("", "RecoveryManager: cleared delay on node %s", n.ID())
	}

	// 復旧完了を記録
	shouldPublish := false
	if !state.IsRecovered && state.RetryCount > 0 {
		state.IsRecovered = true
		m.stats.SuccessRecoveries++
		shouldPublish = true
		logger.Info("", "RecoveryManager: node %s recovered successfully", n.ID())
	}

	state.LastSeen = now
	state.RetryCount = 0
	state.IsRecovered = false
	m.mu.Unlock()

	if shouldPublish {
		m.publishEvent(events.NewRecoverySuccessEvent(n.ID()))
	}
}

// handleStoppedNode は停止したノードを処理する
func (m *Manager) handleStoppedNode(n *node.Node, state *NodeState, now time.Time) {
	if !m.config.AutoRestart {
		return
	}

	m.mu.Lock()

	// 初回検出
	if state.FailedAt.IsZero() {
		state.FailedAt = now
		m.stats.CurrentlyFailed++
		m.mu.Unlock()
		logger.Warn("", "RecoveryManager: detected stopped node %s", n.ID())
		return
	}

	// 復旧待機時間チェック
	if now.Sub(state.FailedAt) < m.config.RecoveryDelay {
		m.mu.Unlock()
		return
	}

	// リトライ上限チェック
	if m.config.MaxRetries > 0 && state.RetryCount >= m.config.MaxRetries {
		m.mu.Unlock()
		return
	}

	state.RetryCount++
	state.FailedAt = now
	m.stats.TotalRecoveries++
	retryCount := state.RetryCount
	m.mu.Unlock()

	m.publishEvent(events.NewRecoveryStartEvent(n.ID(), retryCount))

	// 再起動を試みる
	if err := n.Start(m.ctx); err != nil {
		m.mu.Lock()
		m.stats.FailedRecoveries++
		m.mu.Unlock()
		logger.Error("", "RecoveryManager: failed to restart node %s: %v", n.ID(), err)
		m.publishEvent(events.NewRecoveryFailedEvent(n.ID(), err))
		return
	}

	m.mu.Lock()
	m.stats.CurrentlyFailed--
	state.FailedAt = time.Time{}
	m.mu.Unlock()

	logger.Info("", "RecoveryManager: restarted node %s (attempt %d)", n.ID(), retryCount)
}

// handleSuspendedNode は一時停止中のノードを処理する
func (m *Manager) handleSuspendedNode(n *node.Node, state *NodeState, now time.Time) {
	if !m.config.AutoResume {
		return
	}

	m.mu.Lock()

	// 初回検出
	if state.FailedAt.IsZero() {
		state.FailedAt = now
		m.mu.Unlock()
		logger.Warn("", "RecoveryManager: detected suspended node %s", n.ID())
		return
	}

	// 復旧待機時間チェック
	if now.Sub(state.FailedAt) < m.config.RecoveryDelay {
		m.mu.Unlock()
		return
	}

	state.RetryCount++
	state.FailedAt = time.Time{}
	m.stats.TotalRecoveries++
	retryCount := state.RetryCount
	m.mu.Unlock()

	m.publishEvent(events.NewRecoveryStartEvent(n.ID(), retryCount))

	// 再開を試みる
	if err := n.Resume(); err != nil {
		m.mu.Lock()
		m.stats.FailedRecoveries++
		m.mu.Unlock()
		logger.Error("", "RecoveryManager: failed to resume node %s: %v", n.ID(), err)
		m.publishEvent(events.NewRecoveryFailedEvent(n.ID(), err))
		return
	}

	m.mu.Lock()
	m.stats.SuccessRecoveries++
	m.mu.Unlock()

	logger.Info("", "RecoveryManager: resumed node %s", n.ID())
	m.publishEvent(events.NewRecoverySuccessEvent(n.ID()))
}

// IsRunning は実行中かどうかを返す
func (m *Manager) IsRunning() bool {
	return m.running.Load()
}

// Stats は復旧統計を返す
func (m *Manager) Stats() Stats {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.stats
}

// SetConfig は設定を更新する
func (m *Manager) SetConfig(config Config) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.config = config
}

// ResetStats は統計をリセットする
func (m *Manager) ResetStats() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.stats = Stats{}
}
