// Package client provides a load generator for stress testing the cluster.
package client

import (
	"context"
	cryptorand "crypto/rand"
	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	"chaos-kvs/internal/cluster"
	"chaos-kvs/internal/logger"
	"chaos-kvs/internal/metrics"
	"chaos-kvs/internal/node"
	"chaos-kvs/internal/worker"
)

// Config はClientの設定
type Config struct {
	NumWorkers    int     // ワーカー数（0でCPU数）
	WriteRatio    float64 // Write比率（0.0〜1.0）
	KeyRange      int     // キーの範囲（0〜KeyRange-1）
	ValueSize     int     // 値のサイズ（バイト）
	RequestsLimit uint64  // リクエスト上限（0で無制限）
}

// DefaultConfig はデフォルト設定を返す
func DefaultConfig() Config {
	return Config{
		NumWorkers:    0,   // CPU数
		WriteRatio:    0.5, // 50% Write
		KeyRange:      10000,
		ValueSize:     100,
		RequestsLimit: 0,
	}
}

// Client は負荷生成器
type Client struct {
	config  Config
	cluster *cluster.Cluster
	pool    *worker.Pool
	metrics *metrics.Metrics

	running atomic.Bool
	ctx     context.Context
	cancel  context.CancelFunc
	wg      sync.WaitGroup
}

// New は新しいClientを作成する
func New(c *cluster.Cluster, config Config) *Client {
	return &Client{
		config:  config,
		cluster: c,
		pool:    worker.NewPool(config.NumWorkers),
		metrics: metrics.New(),
	}
}

// Start は負荷生成を開始する
func (c *Client) Start(ctx context.Context) {
	if c.running.Swap(true) {
		return // Already running
	}

	c.ctx, c.cancel = context.WithCancel(ctx)
	c.pool.Start(c.ctx)

	logger.Info("", "Client started (workers: %d, write_ratio: %.1f%%)",
		c.pool.NumWorkers(), c.config.WriteRatio*100)

	// リクエスト生成ループ
	c.wg.Add(1)
	go c.generateRequests()
}

// generateRequests はリクエストを生成し続ける
func (c *Client) generateRequests() {
	defer c.wg.Done()

	nodes := c.cluster.Nodes()
	if len(nodes) == 0 {
		logger.Error("", "No nodes available in cluster")
		return
	}

	for {
		select {
		case <-c.ctx.Done():
			return
		default:
		}

		// リクエスト上限チェック
		if c.config.RequestsLimit > 0 && c.metrics.TotalRequests() >= c.config.RequestsLimit {
			return
		}

		// ジョブを生成
		n := nodes[rand.Intn(len(nodes))]
		key := fmt.Sprintf("key-%d", rand.Intn(c.config.KeyRange))
		isWrite := rand.Float64() < c.config.WriteRatio

		job := c.createJob(n, key, isWrite)
		if !c.pool.Submit(job) {
			return
		}
	}
}

// createJob はリクエストジョブを作成する
func (c *Client) createJob(n *node.Node, key string, isWrite bool) worker.Job {
	return func() {
		start := time.Now()
		var err error

		if isWrite {
			value := make([]byte, c.config.ValueSize)
			_, _ = cryptorand.Read(value)
			err = n.Set(key, value)
		} else {
			_, _ = n.Get(key)
		}

		latency := time.Since(start)
		if err != nil {
			c.metrics.RecordFailure(latency)
		} else {
			c.metrics.RecordSuccess(latency)
		}
	}
}

// Stop は負荷生成を停止する
func (c *Client) Stop() {
	if !c.running.Swap(false) {
		return // Not running
	}

	c.cancel()
	c.pool.Stop()
	c.wg.Wait()

	logger.Info("", "Client stopped")
}

// Metrics はメトリクスを返す
func (c *Client) Metrics() *metrics.Metrics {
	return c.metrics
}

// IsRunning は実行中かどうかを返す
func (c *Client) IsRunning() bool {
	return c.running.Load()
}

// RunFor は指定時間だけ負荷生成を実行する
func (c *Client) RunFor(ctx context.Context, duration time.Duration) *metrics.Snapshot {
	c.Start(ctx)

	select {
	case <-ctx.Done():
	case <-time.After(duration):
	}

	c.Stop()

	snapshot := c.metrics.Snapshot()
	return &snapshot
}

// RunRequests は指定数のリクエストを実行する
func (c *Client) RunRequests(ctx context.Context, count uint64) *metrics.Snapshot {
	c.config.RequestsLimit = count
	c.Start(ctx)
	c.wg.Wait()
	c.Stop()

	snapshot := c.metrics.Snapshot()
	return &snapshot
}
