package metrics

import (
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

// Metrics はリクエストのメトリクスを収集する
type Metrics struct {
	totalRequests   atomic.Uint64
	successRequests atomic.Uint64
	failedRequests  atomic.Uint64
	totalLatencyNs  atomic.Uint64

	mu                sync.RWMutex
	startTime         time.Time
	lastResetTime     time.Time
	windowRequests    uint64
	latencies         []time.Duration
	maxLatencySamples int
}

// New は新しいメトリクスを作成する
func New() *Metrics {
	now := time.Now()
	return &Metrics{
		startTime:         now,
		lastResetTime:     now,
		latencies:         make([]time.Duration, 0, 1000),
		maxLatencySamples: 1000,
	}
}

// RecordSuccess は成功したリクエストを記録する
func (m *Metrics) RecordSuccess(latency time.Duration) {
	m.totalRequests.Add(1)
	m.successRequests.Add(1)
	m.totalLatencyNs.Add(uint64(latency.Nanoseconds()))

	m.mu.Lock()
	m.windowRequests++
	if len(m.latencies) < m.maxLatencySamples {
		m.latencies = append(m.latencies, latency)
	}
	m.mu.Unlock()
}

// RecordFailure は失敗したリクエストを記録する
func (m *Metrics) RecordFailure(latency time.Duration) {
	m.totalRequests.Add(1)
	m.failedRequests.Add(1)
	m.totalLatencyNs.Add(uint64(latency.Nanoseconds()))

	m.mu.Lock()
	m.windowRequests++
	m.mu.Unlock()
}

// TotalRequests は総リクエスト数を返す
func (m *Metrics) TotalRequests() uint64 {
	return m.totalRequests.Load()
}

// SuccessRequests は成功リクエスト数を返す
func (m *Metrics) SuccessRequests() uint64 {
	return m.successRequests.Load()
}

// FailedRequests は失敗リクエスト数を返す
func (m *Metrics) FailedRequests() uint64 {
	return m.failedRequests.Load()
}

// RPS は現在のRequests Per Secondを返す
func (m *Metrics) RPS() float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()

	elapsed := time.Since(m.lastResetTime).Seconds()
	if elapsed == 0 {
		return 0
	}
	return float64(m.windowRequests) / elapsed
}

// OverallRPS は開始からの平均RPSを返す
func (m *Metrics) OverallRPS() float64 {
	elapsed := time.Since(m.startTime).Seconds()
	if elapsed == 0 {
		return 0
	}
	return float64(m.totalRequests.Load()) / elapsed
}

// AverageLatency は平均レイテンシを返す
func (m *Metrics) AverageLatency() time.Duration {
	total := m.totalRequests.Load()
	if total == 0 {
		return 0
	}
	avgNs := m.totalLatencyNs.Load() / total
	return time.Duration(avgNs)
}

// P99Latency はP99レイテンシを返す（サンプルベース）
func (m *Metrics) P99Latency() time.Duration {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if len(m.latencies) == 0 {
		return 0
	}

	// コピーしてソート（標準ライブラリ使用）
	sorted := make([]time.Duration, len(m.latencies))
	copy(sorted, m.latencies)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i] < sorted[j]
	})

	idx := int(float64(len(sorted)) * 0.99)
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}
	return sorted[idx]
}

// ErrorRate はエラー率を返す（0.0〜1.0）
func (m *Metrics) ErrorRate() float64 {
	total := m.totalRequests.Load()
	if total == 0 {
		return 0
	}
	return float64(m.failedRequests.Load()) / float64(total)
}

// Reset はウィンドウメトリクスをリセットする
func (m *Metrics) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.windowRequests = 0
	m.lastResetTime = time.Now()
	m.latencies = m.latencies[:0]
}

// Snapshot はメトリクスのスナップショット
type Snapshot struct {
	TotalRequests   uint64
	SuccessRequests uint64
	FailedRequests  uint64
	RPS             float64
	OverallRPS      float64
	AverageLatency  time.Duration
	P99Latency      time.Duration
	ErrorRate       float64
	Elapsed         time.Duration
}

// Snapshot は現在のメトリクスのスナップショットを返す
func (m *Metrics) Snapshot() Snapshot {
	return Snapshot{
		TotalRequests:   m.TotalRequests(),
		SuccessRequests: m.SuccessRequests(),
		FailedRequests:  m.FailedRequests(),
		RPS:             m.RPS(),
		OverallRPS:      m.OverallRPS(),
		AverageLatency:  m.AverageLatency(),
		P99Latency:      m.P99Latency(),
		ErrorRate:       m.ErrorRate(),
		Elapsed:         time.Since(m.startTime),
	}
}
