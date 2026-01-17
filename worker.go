package main

import (
	"context"
	"runtime"
	"sync"
)

// Job はワーカーが実行するジョブを表す
type Job func()

// WorkerPool はゴルーチンのプールを管理する
type WorkerPool struct {
	numWorkers int
	jobs       chan Job
	wg         sync.WaitGroup
	ctx        context.Context
	cancel     context.CancelFunc
	started    bool
	mu         sync.Mutex
}

// NewWorkerPool は新しいワーカープールを作成する
// numWorkers が 0 の場合は CPU 数を使用
func NewWorkerPool(numWorkers int) *WorkerPool {
	if numWorkers <= 0 {
		numWorkers = runtime.NumCPU()
	}
	return &WorkerPool{
		numWorkers: numWorkers,
		jobs:       make(chan Job, numWorkers*100),
	}
}

// Start はワーカープールを起動する
func (p *WorkerPool) Start(ctx context.Context) {
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

	LogInfo("", "WorkerPool started with %d workers", p.numWorkers)
}

// worker は個々のワーカーゴルーチン
func (p *WorkerPool) worker(_ int) {
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
func (p *WorkerPool) Submit(job Job) (submitted bool) {
	defer func() {
		if recover() != nil {
			submitted = false
		}
	}()

	select {
	case <-p.ctx.Done():
		return false
	case p.jobs <- job:
		return true
	}
}

// SubmitWait はジョブを送信し、キューに空きがなければブロックする
func (p *WorkerPool) SubmitWait(job Job) bool {
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
func (p *WorkerPool) Stop() {
	p.mu.Lock()
	if !p.started {
		p.mu.Unlock()
		return
	}
	p.mu.Unlock()

	p.cancel()
	close(p.jobs)
	p.wg.Wait()

	p.mu.Lock()
	p.started = false
	p.mu.Unlock()

	LogInfo("", "WorkerPool stopped")
}

// NumWorkers はワーカー数を返す
func (p *WorkerPool) NumWorkers() int {
	return p.numWorkers
}

// QueueSize は現在のキューサイズを返す
func (p *WorkerPool) QueueSize() int {
	return len(p.jobs)
}
