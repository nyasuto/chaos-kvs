// Package cluster provides multi-node cluster management.
package cluster

import (
	"context"
	"fmt"
	"sync"

	"chaos-kvs/internal/logger"
	"chaos-kvs/internal/node"
)

// Manager はクラスタ管理の基本操作を定義するインターフェース
type Manager interface {
	AddNode(n *node.Node) error
	RemoveNode(nodeID string) error
	GetNode(nodeID string) (*node.Node, bool)
	Nodes() []*node.Node
	StartAll(ctx context.Context) error
	StopAll() error
	Size() int
	RunningCount() int
}

// Ensure Cluster implements Manager
var _ Manager = (*Cluster)(nil)

// Cluster は複数のノードを管理する
type Cluster struct {
	mu    sync.RWMutex
	nodes map[string]*node.Node
	ctx   context.Context
}

// New は新しいクラスタを作成する
func New() *Cluster {
	return &Cluster{
		nodes: make(map[string]*node.Node),
	}
}

// AddNode はクラスタにノードを追加する
func (c *Cluster) AddNode(n *node.Node) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, exists := c.nodes[n.ID()]; exists {
		return fmt.Errorf("node %s already exists in cluster", n.ID())
	}

	c.nodes[n.ID()] = n
	logger.Info("", "Node %s added to cluster", n.ID())
	return nil
}

// RemoveNode はクラスタからノードを削除する
func (c *Cluster) RemoveNode(nodeID string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	n, exists := c.nodes[nodeID]
	if !exists {
		return fmt.Errorf("node %s not found in cluster", nodeID)
	}

	if n.Status() == node.StatusRunning {
		_ = n.Stop()
	}

	delete(c.nodes, nodeID)
	logger.Info("", "Node %s removed from cluster", nodeID)
	return nil
}

// GetNode はノードIDでノードを取得する
func (c *Cluster) GetNode(nodeID string) (*node.Node, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	n, exists := c.nodes[nodeID]
	return n, exists
}

// Nodes は全てのノードを返す
func (c *Cluster) Nodes() []*node.Node {
	c.mu.RLock()
	defer c.mu.RUnlock()

	nodes := make([]*node.Node, 0, len(c.nodes))
	for _, n := range c.nodes {
		nodes = append(nodes, n)
	}
	return nodes
}

// StartAll は全てのノードを起動する
func (c *Cluster) StartAll(ctx context.Context) error {
	c.mu.Lock()
	c.ctx = ctx
	nodes := make([]*node.Node, 0, len(c.nodes))
	for _, n := range c.nodes {
		nodes = append(nodes, n)
	}
	c.mu.Unlock()

	logger.Info("", "Starting all nodes in cluster (count: %d)", len(nodes))

	var wg sync.WaitGroup
	errCh := make(chan error, len(nodes))

	for _, n := range nodes {
		wg.Add(1)
		go func(n *node.Node) {
			defer wg.Done()
			if err := n.Start(ctx); err != nil {
				errCh <- err
			}
		}(n)
	}

	wg.Wait()
	close(errCh)

	var errs []error
	for err := range errCh {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		logger.Error("", "Failed to start %d nodes", len(errs))
		return fmt.Errorf("failed to start %d nodes", len(errs))
	}

	logger.Info("", "All nodes started successfully")
	return nil
}

// StopAll は全てのノードを停止する
func (c *Cluster) StopAll() error {
	c.mu.RLock()
	nodes := make([]*node.Node, 0, len(c.nodes))
	for _, n := range c.nodes {
		nodes = append(nodes, n)
	}
	c.mu.RUnlock()

	logger.Info("", "Stopping all nodes in cluster (count: %d)", len(nodes))

	var wg sync.WaitGroup
	errCh := make(chan error, len(nodes))

	for _, n := range nodes {
		wg.Add(1)
		go func(n *node.Node) {
			defer wg.Done()
			if err := n.Stop(); err != nil {
				errCh <- err
			}
		}(n)
	}

	wg.Wait()
	close(errCh)

	var errs []error
	for err := range errCh {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		logger.Warn("", "Failed to stop %d nodes (may already be stopped)", len(errs))
	}

	logger.Info("", "All nodes stopped")
	return nil
}

// Size はクラスタ内のノード数を返す
func (c *Cluster) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.nodes)
}

// RunningCount は実行中のノード数を返す
func (c *Cluster) RunningCount() int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	count := 0
	for _, n := range c.nodes {
		if n.Status() == node.StatusRunning {
			count++
		}
	}
	return count
}

// StoppedCount は停止中のノード数を返す
func (c *Cluster) StoppedCount() int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	count := 0
	for _, n := range c.nodes {
		if n.Status() == node.StatusStopped {
			count++
		}
	}
	return count
}

// CreateNodes は指定された数のノードを作成してクラスタに追加する
func (c *Cluster) CreateNodes(count int, prefix string) error {
	logger.Info("", "Creating %d nodes with prefix '%s'", count, prefix)

	for i := range count {
		nodeID := fmt.Sprintf("%s-%d", prefix, i+1)
		n := node.New(nodeID)
		if err := c.AddNode(n); err != nil {
			return err
		}
	}

	logger.Info("", "Created %d nodes successfully", count)
	return nil
}
