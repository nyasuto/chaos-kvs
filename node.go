package main

import (
	"context"
	"fmt"
	"sync"
)

// NodeStatus はノードの状態を表す
type NodeStatus int

const (
	NodeStatusStopped NodeStatus = iota
	NodeStatusRunning
	NodeStatusSuspended
)

func (s NodeStatus) String() string {
	switch s {
	case NodeStatusStopped:
		return "stopped"
	case NodeStatusRunning:
		return "running"
	case NodeStatusSuspended:
		return "suspended"
	default:
		return "unknown"
	}
}

// Node はインメモリKVSの単一ノードを表す
type Node struct {
	ID     string
	status NodeStatus

	mu   sync.RWMutex
	data map[string][]byte

	ctx    context.Context
	cancel context.CancelFunc
}

// NewNode は新しいノードを作成する
func NewNode(id string) *Node {
	return &Node{
		ID:     id,
		status: NodeStatusStopped,
		data:   make(map[string][]byte),
	}
}

// Start はノードを起動する
func (n *Node) Start(ctx context.Context) error {
	n.mu.Lock()
	defer n.mu.Unlock()

	if n.status == NodeStatusRunning {
		return fmt.Errorf("node %s is already running", n.ID)
	}

	n.ctx, n.cancel = context.WithCancel(ctx)
	n.status = NodeStatusRunning

	fmt.Printf("[INFO] Node %s started\n", n.ID)
	return nil
}

// Stop はノードを停止する
func (n *Node) Stop() error {
	n.mu.Lock()
	defer n.mu.Unlock()

	if n.status == NodeStatusStopped {
		return fmt.Errorf("node %s is already stopped", n.ID)
	}

	if n.cancel != nil {
		n.cancel()
	}
	n.status = NodeStatusStopped

	fmt.Printf("[INFO] Node %s stopped\n", n.ID)
	return nil
}

// Status はノードの現在のステータスを返す
func (n *Node) Status() NodeStatus {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.status
}

// Get はキーに対応する値を取得する
func (n *Node) Get(key string) ([]byte, bool) {
	n.mu.RLock()
	defer n.mu.RUnlock()

	if n.status != NodeStatusRunning {
		return nil, false
	}

	value, exists := n.data[key]
	return value, exists
}

// Set はキーに値を設定する
func (n *Node) Set(key string, value []byte) error {
	n.mu.Lock()
	defer n.mu.Unlock()

	if n.status != NodeStatusRunning {
		return fmt.Errorf("node %s is not running", n.ID)
	}

	n.data[key] = value
	return nil
}

// Delete はキーを削除する
func (n *Node) Delete(key string) error {
	n.mu.Lock()
	defer n.mu.Unlock()

	if n.status != NodeStatusRunning {
		return fmt.Errorf("node %s is not running", n.ID)
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
