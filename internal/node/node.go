package node

import (
	"context"
	"fmt"
	"sync"
	"time"

	"chaos-kvs/internal/logger"
)

// Store はKVSの基本操作を定義するインターフェース
type Store interface {
	Get(key string) ([]byte, bool)
	Set(key string, value []byte) error
	Delete(key string) error
	Keys() []string
	Size() int
}

// Ensure Node implements Store
var _ Store = (*Node)(nil)

// Status はノードの状態を表す
type Status int

const (
	StatusStopped Status = iota
	StatusRunning
	StatusSuspended
)

func (s Status) String() string {
	switch s {
	case StatusStopped:
		return "stopped"
	case StatusRunning:
		return "running"
	case StatusSuspended:
		return "suspended"
	default:
		return "unknown"
	}
}

// Node はインメモリKVSの単一ノードを表す
type Node struct {
	id     string
	status Status
	delay  time.Duration

	mu   sync.RWMutex
	data map[string][]byte

	ctx    context.Context
	cancel context.CancelFunc
}

// New は新しいノードを作成する
func New(id string) *Node {
	return &Node{
		id:     id,
		status: StatusStopped,
		data:   make(map[string][]byte),
	}
}

// ID はノードIDを返す
func (n *Node) ID() string {
	return n.id
}

// Start はノードを起動する
func (n *Node) Start(ctx context.Context) error {
	n.mu.Lock()
	defer n.mu.Unlock()

	if n.status == StatusRunning {
		return fmt.Errorf("node %s is already running", n.id)
	}

	n.ctx, n.cancel = context.WithCancel(ctx)
	n.status = StatusRunning

	logger.Info(n.id, "Node started")
	return nil
}

// Stop はノードを停止する
func (n *Node) Stop() error {
	n.mu.Lock()
	defer n.mu.Unlock()

	if n.status == StatusStopped {
		return fmt.Errorf("node %s is already stopped", n.id)
	}

	if n.cancel != nil {
		n.cancel()
	}
	n.status = StatusStopped

	logger.Info(n.id, "Node stopped")
	return nil
}

// Status はノードの現在のステータスを返す
func (n *Node) Status() Status {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.status
}

// Suspend はノードを一時停止する
func (n *Node) Suspend() error {
	n.mu.Lock()
	defer n.mu.Unlock()

	if n.status != StatusRunning {
		return fmt.Errorf("node %s is not running", n.id)
	}

	n.status = StatusSuspended
	logger.Info(n.id, "Node suspended")
	return nil
}

// Resume は一時停止中のノードを再開する
func (n *Node) Resume() error {
	n.mu.Lock()
	defer n.mu.Unlock()

	if n.status != StatusSuspended {
		return fmt.Errorf("node %s is not suspended", n.id)
	}

	n.status = StatusRunning
	logger.Info(n.id, "Node resumed")
	return nil
}

// SetDelay はレスポンス遅延を設定する
func (n *Node) SetDelay(d time.Duration) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.delay = d
	if d > 0 {
		logger.Info(n.id, "Delay set to %v", d)
	} else {
		logger.Info(n.id, "Delay cleared")
	}
}

// Delay は現在の遅延設定を返す
func (n *Node) Delay() time.Duration {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.delay
}

// applyDelay は設定された遅延を適用する
func (n *Node) applyDelay() {
	if d := n.Delay(); d > 0 {
		time.Sleep(d)
	}
}

// Get はキーに対応する値を取得する
func (n *Node) Get(key string) ([]byte, bool) {
	n.applyDelay()

	n.mu.RLock()
	defer n.mu.RUnlock()

	if n.status != StatusRunning {
		return nil, false
	}

	value, exists := n.data[key]
	return value, exists
}

// Set はキーに値を設定する
func (n *Node) Set(key string, value []byte) error {
	n.applyDelay()

	n.mu.Lock()
	defer n.mu.Unlock()

	if n.status != StatusRunning {
		return fmt.Errorf("node %s is not running", n.id)
	}

	n.data[key] = value
	return nil
}

// Delete はキーを削除する
func (n *Node) Delete(key string) error {
	n.mu.Lock()
	defer n.mu.Unlock()

	if n.status != StatusRunning {
		return fmt.Errorf("node %s is not running", n.id)
	}

	delete(n.data, key)
	return nil
}

// Keys は全てのキーを返す
func (n *Node) Keys() []string {
	n.mu.RLock()
	defer n.mu.RUnlock()

	keys := make([]string, 0, len(n.data))
	for k := range n.data {
		keys = append(keys, k)
	}
	return keys
}

// Size はデータストアのサイズを返す
func (n *Node) Size() int {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return len(n.data)
}
