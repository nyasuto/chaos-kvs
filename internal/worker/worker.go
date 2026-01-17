package worker

import (
	"context"
	"runtime"
	"sync"
	"sync/atomic"

	"chaos-kvs/internal/logger"
)

// Job はワーカーが実行するジョブを表す
type Job func()

// PoolConfig はワーカープールの設定
type PoolConfig struct {
	NumWorkers  int // ワーカー数（0でCPU数）
	QueueFactor int // キューサイズ = NumWorkers * QueueFactor
}

// DefaultPoolConfig はデフォルト設定を返す
func DefaultPoolConfig() PoolConfig {
	return PoolConfig{
		NumWorkers:  0,   // CPU数
		QueueFactor: 100, // デフォルト倍率
	}
}

// Pool はゴルーチンのプールを管理する
type Pool struct {
	numWorkers int
	jobs       chan Job
	wg         sync.WaitGroup
	ctx        context.Context
	cancel     context.CancelFunc
	started    bool
	stopping   atomic.Bool
	mu         sync.Mutex
}

// NewPool は新しいワーカープールを作成する
// numWorkers が 0 の場合は CPU 数を使用
func NewPool(numWorkers int) *Pool {
	config := DefaultPoolConfig()
	config.NumWorkers = numWorkers
	return NewPoolWithConfig(config)
}

// NewPoolWithConfig は設定を指定してワーカープールを作成する
func NewPoolWithConfig(config PoolConfig) *Pool {
	numWorkers := config.NumWorkers
	if numWorkers <= 0 {
		numWorkers = runtime.NumCPU()
	}
	queueFactor := config.QueueFactor
	if queueFactor <= 0 {
		queueFactor = 100
	}
	return &Pool{
		numWorkers: numWorkers,
		jobs:       make(chan Job, numWorkers*queueFactor),
	}
}

// Start はワーカープールを起動する
func (p *Pool) Start(ctx context.Context) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.started {
		return
	}

	p.ctx, p.cancel = context.WithCancel(ctx)
	p.started = true

	for i := range p.numWorkers {
		p.wg.Add(1)
		go p.worker(i)
	}

	logger.Info("", "WorkerPool started with %d workers", p.numWorkers)
}

// worker は個々のワーカーゴルーチン
func (p *Pool) worker(_ int) {
	defer p.wg.Done()

	for {
		select {
		case <-p.ctx.Done():
			return
		case job, ok := <-p.jobs:
			if !ok {
				return
			}
			job()
		}
	}
}

// Submit はジョブをプールに送信する
func (p *Pool) Submit(job Job) (submitted bool) {
	if p.stopping.Load() {
		return false
	}

	defer func() {
		if r := recover(); r != nil {
			logger.Warn("", "Submit failed due to panic (channel may be closed): %v", r)
			submitted = false
		}
	}()

	// 先にコンテキストをチェック
	select {
	case <-p.ctx.Done():
		return false
	default:
	}

	select {
	case <-p.ctx.Done():
		return false
	case p.jobs <- job:
		return true
	}
}

// SubmitWait はジョブを送信し、キューに空きがなければブロックする
func (p *Pool) SubmitWait(job Job) bool {
	if p.stopping.Load() {
		return false
	}

	select {
	case <-p.ctx.Done():
		return false
	default:
	}

	select {
	case <-p.ctx.Done():
		return false
	case p.jobs <- job:
		return true
	}
}

// Stop はワーカープールを停止する
func (p *Pool) Stop() {
	p.mu.Lock()
	if !p.started {
		p.mu.Unlock()
		return
	}
	p.mu.Unlock()

	p.stopping.Store(true)
	p.cancel()
	p.wg.Wait()
	close(p.jobs)

	p.mu.Lock()
	p.started = false
	p.stopping.Store(false)
	p.mu.Unlock()

	logger.Info("", "WorkerPool stopped")
}

// NumWorkers はワーカー数を返す
func (p *Pool) NumWorkers() int {
	return p.numWorkers
}

// QueueSize は現在のキューサイズを返す
func (p *Pool) QueueSize() int {
	return len(p.jobs)
}
