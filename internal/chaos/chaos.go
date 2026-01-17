package chaos

import (
	"context"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	"chaos-kvs/internal/cluster"
	"chaos-kvs/internal/events"
	"chaos-kvs/internal/logger"
	"chaos-kvs/internal/node"
)

// AttackType は障害の種類を表す
type AttackType int

const (
	AttackKill AttackType = iota
	AttackSuspend
	AttackDelay
)

func (a AttackType) String() string {
	switch a {
	case AttackKill:
		return "kill"
	case AttackSuspend:
		return "suspend"
	case AttackDelay:
		return "delay"
	default:
		return "unknown"
	}
}

// Config はChaosMonkeyの設定
type Config struct {
	Interval      time.Duration // 攻撃間隔
	TargetCount   int           // 同時攻撃対象数
	AttackTypes   []AttackType  // 有効な攻撃タイプ
	DelayDuration time.Duration // Delay攻撃時の遅延時間
	SuspendTime   time.Duration // Suspend攻撃の継続時間（0で手動Resume）
}

// DefaultConfig はデフォルト設定を返す
func DefaultConfig() Config {
	return Config{
		Interval:      5 * time.Second,
		TargetCount:   1,
		AttackTypes:   []AttackType{AttackKill, AttackSuspend, AttackDelay},
		DelayDuration: 100 * time.Millisecond,
		SuspendTime:   3 * time.Second,
	}
}

// Stats はカオス攻撃の統計情報
type Stats struct {
	TotalAttacks uint64            `json:"total_attacks"`
	ByType       map[string]uint64 `json:"attacks_by_type"`
}

// Monkey はカオスエンジニアリングを実行する
type Monkey struct {
	config   Config
	cluster  *cluster.Cluster
	eventBus *events.Bus

	running atomic.Bool
	ctx     context.Context
	cancel  context.CancelFunc
	wg      sync.WaitGroup

	mu           sync.RWMutex
	attackCount  uint64
	attackByType map[AttackType]uint64
	lastAttack   time.Time
	suspendedIDs map[string]time.Time
}

// New は新しいChaosMonkeyを作成する
func New(c *cluster.Cluster, config Config) *Monkey {
	return &Monkey{
		config:       config,
		cluster:      c,
		suspendedIDs: make(map[string]time.Time),
		attackByType: make(map[AttackType]uint64),
	}
}

// SetEventBus はイベントバスを設定する
func (m *Monkey) SetEventBus(bus *events.Bus) {
	m.eventBus = bus
}

// publishEvent はイベントを発行する
func (m *Monkey) publishEvent(event events.Event) {
	if m.eventBus != nil {
		m.eventBus.Publish(event)
	}
}

// Start はカオス注入を開始する
func (m *Monkey) Start(ctx context.Context) {
	if m.running.Swap(true) {
		return
	}

	m.ctx, m.cancel = context.WithCancel(ctx)

	m.wg.Add(1)
	go m.attackLoop()

	if m.config.SuspendTime > 0 {
		m.wg.Add(1)
		go m.resumeLoop()
	}

	logger.Info("", "ChaosMonkey started (interval: %v, targets: %d)",
		m.config.Interval, m.config.TargetCount)
}

// Stop はカオス注入を停止する
func (m *Monkey) Stop() {
	if !m.running.Swap(false) {
		return
	}

	m.cancel()
	m.wg.Wait()

	// 残っているsuspendedノードをresumeする
	m.resumeAll()

	logger.Info("", "ChaosMonkey stopped (total attacks: %d)", m.attackCount)
}

// attackLoop は定期的に攻撃を実行する
func (m *Monkey) attackLoop() {
	defer m.wg.Done()

	ticker := time.NewTicker(m.config.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-m.ctx.Done():
			return
		case <-ticker.C:
			m.attack()
		}
	}
}

// resumeLoop はsuspendされたノードを自動的にresumeする
func (m *Monkey) resumeLoop() {
	defer m.wg.Done()

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-m.ctx.Done():
			return
		case <-ticker.C:
			m.checkAndResume()
		}
	}
}

// attack は攻撃を実行する
func (m *Monkey) attack() {
	targets := m.selectTargets()
	if len(targets) == 0 {
		return
	}

	attackType := m.selectAttackType()

	for _, n := range targets {
		m.executeAttack(n, attackType)
	}

	m.mu.Lock()
	m.attackCount++
	m.lastAttack = time.Now()
	m.mu.Unlock()
}

// selectTargets は攻撃対象のノードを選択する
func (m *Monkey) selectTargets() []*node.Node {
	nodes := m.cluster.Nodes()
	if len(nodes) == 0 {
		return nil
	}

	// 稼働中のノードのみを対象とする
	running := make([]*node.Node, 0)
	for _, n := range nodes {
		if n.Status() == node.StatusRunning {
			running = append(running, n)
		}
	}

	if len(running) == 0 {
		return nil
	}

	// ターゲット数を調整
	count := m.config.TargetCount
	if count > len(running) {
		count = len(running)
	}

	// ランダムに選択
	rand.Shuffle(len(running), func(i, j int) {
		running[i], running[j] = running[j], running[i]
	})

	return running[:count]
}

// selectAttackType は攻撃タイプをランダムに選択する
func (m *Monkey) selectAttackType() AttackType {
	if len(m.config.AttackTypes) == 0 {
		return AttackKill
	}
	return m.config.AttackTypes[rand.Intn(len(m.config.AttackTypes))]
}

// executeAttack は指定された攻撃を実行する
func (m *Monkey) executeAttack(n *node.Node, attackType AttackType) {
	switch attackType {
	case AttackKill:
		m.attackKill(n)
	case AttackSuspend:
		m.attackSuspend(n)
	case AttackDelay:
		m.attackDelay(n)
	}
}

// attackKill はノードを強制停止する
func (m *Monkey) attackKill(n *node.Node) {
	if err := n.Stop(); err != nil {
		logger.Warn("", "ChaosMonkey: failed to kill node %s: %v", n.ID(), err)
		return
	}
	logger.Warn("", "ChaosMonkey: killed node %s", n.ID())
	m.publishEvent(events.NewChaosAttackEvent(n.ID(), events.AttackTypeKill))

	m.mu.Lock()
	m.attackByType[AttackKill]++
	m.mu.Unlock()
}

// attackSuspend はノードを一時停止する
func (m *Monkey) attackSuspend(n *node.Node) {
	if err := n.Suspend(); err != nil {
		logger.Warn("", "ChaosMonkey: failed to suspend node %s: %v", n.ID(), err)
		return
	}

	m.mu.Lock()
	m.suspendedIDs[n.ID()] = time.Now()
	m.attackByType[AttackSuspend]++
	m.mu.Unlock()

	logger.Warn("", "ChaosMonkey: suspended node %s", n.ID())
	m.publishEvent(events.NewChaosAttackEvent(n.ID(), events.AttackTypeSuspend))
}

// attackDelay はノードに遅延を注入する
func (m *Monkey) attackDelay(n *node.Node) {
	n.SetDelay(m.config.DelayDuration)
	logger.Warn("", "ChaosMonkey: injected %v delay to node %s", m.config.DelayDuration, n.ID())
	m.publishEvent(events.NewChaosAttackEventWithDelay(n.ID(), m.config.DelayDuration))

	m.mu.Lock()
	m.attackByType[AttackDelay]++
	m.mu.Unlock()
}

// checkAndResume はsuspend時間が経過したノードをresumeする
func (m *Monkey) checkAndResume() {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	for nodeID, suspendTime := range m.suspendedIDs {
		if now.Sub(suspendTime) >= m.config.SuspendTime {
			if n, exists := m.cluster.GetNode(nodeID); exists {
				if err := n.Resume(); err == nil {
					logger.Info("", "ChaosMonkey: auto-resumed node %s", nodeID)
					m.publishEvent(events.NewChaosResumeEvent(nodeID))
				}
			}
			delete(m.suspendedIDs, nodeID)
		}
	}
}

// resumeAll は全てのsuspendedノードをresumeする
func (m *Monkey) resumeAll() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for nodeID := range m.suspendedIDs {
		if n, exists := m.cluster.GetNode(nodeID); exists {
			if err := n.Resume(); err == nil {
				logger.Info("", "ChaosMonkey: resumed node %s on shutdown", nodeID)
			}
		}
	}
	m.suspendedIDs = make(map[string]time.Time)
}

// IsRunning は実行中かどうかを返す
func (m *Monkey) IsRunning() bool {
	return m.running.Load()
}

// AttackCount は攻撃回数を返す
func (m *Monkey) AttackCount() uint64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.attackCount
}

// SetConfig は設定を更新する
func (m *Monkey) SetConfig(config Config) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.config = config
}

// Stats は攻撃統計を返す
func (m *Monkey) Stats() Stats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	byType := make(map[string]uint64)
	for t, count := range m.attackByType {
		byType[t.String()] = count
	}

	return Stats{
		TotalAttacks: m.attackCount,
		ByType:       byType,
	}
}
