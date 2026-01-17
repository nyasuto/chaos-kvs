package main

import (
	"context"
	"fmt"
	"sync"
)

// Cluster は複数のノードを管理する
type Cluster struct {
	mu    sync.RWMutex
	nodes map[string]*Node
	ctx   context.Context
}

// NewCluster は新しいクラスタを作成する
func NewCluster() *Cluster {
	return &Cluster{
		nodes: make(map[string]*Node),
	}
}

// AddNode はクラスタにノードを追加する
func (c *Cluster) AddNode(node *Node) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, exists := c.nodes[node.ID]; exists {
		return fmt.Errorf("node %s already exists in cluster", node.ID)
	}

	c.nodes[node.ID] = node
	LogInfo("", "Node %s added to cluster", node.ID)
	return nil
}

// RemoveNode はクラスタからノードを削除する
func (c *Cluster) RemoveNode(nodeID string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	node, exists := c.nodes[nodeID]
	if !exists {
		return fmt.Errorf("node %s not found in cluster", nodeID)
	}

	if node.Status() == NodeStatusRunning {
		_ = node.Stop()
	}

	delete(c.nodes, nodeID)
	LogInfo("", "Node %s removed from cluster", nodeID)
	return nil
}

// GetNode はノードIDでノードを取得する
func (c *Cluster) GetNode(nodeID string) (*Node, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	node, exists := c.nodes[nodeID]
	return node, exists
}

// Nodes は全てのノードを返す
func (c *Cluster) Nodes() []*Node {
	c.mu.RLock()
	defer c.mu.RUnlock()

	nodes := make([]*Node, 0, len(c.nodes))
	for _, node := range c.nodes {
		nodes = append(nodes, node)
	}
	return nodes
}

// StartAll は全てのノードを起動する
func (c *Cluster) StartAll(ctx context.Context) error {
	c.mu.Lock()
	c.ctx = ctx
	nodes := make([]*Node, 0, len(c.nodes))
	for _, node := range c.nodes {
		nodes = append(nodes, node)
	}
	c.mu.Unlock()

	LogInfo("", "Starting all nodes in cluster (count: %d)", len(nodes))

	var wg sync.WaitGroup
	errCh := make(chan error, len(nodes))

	for _, node := range nodes {
		wg.Add(1)
		go func(n *Node) {
			defer wg.Done()
			if err := n.Start(ctx); err != nil {
				errCh <- err
			}
		}(node)
	}

	wg.Wait()
	close(errCh)

	var errs []error
	for err := range errCh {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		LogError("", "Failed to start %d nodes", len(errs))
		return fmt.Errorf("failed to start %d nodes", len(errs))
	}

	LogInfo("", "All nodes started successfully")
	return nil
}

// StopAll は全てのノードを停止する
func (c *Cluster) StopAll() error {
	c.mu.RLock()
	nodes := make([]*Node, 0, len(c.nodes))
	for _, node := range c.nodes {
		nodes = append(nodes, node)
	}
	c.mu.RUnlock()

	LogInfo("", "Stopping all nodes in cluster (count: %d)", len(nodes))

	var wg sync.WaitGroup
	errCh := make(chan error, len(nodes))

	for _, node := range nodes {
		wg.Add(1)
		go func(n *Node) {
			defer wg.Done()
			if err := n.Stop(); err != nil {
				errCh <- err
			}
		}(node)
	}

	wg.Wait()
	close(errCh)

	var errs []error
	for err := range errCh {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		LogWarn("", "Failed to stop %d nodes (may already be stopped)", len(errs))
	}

	LogInfo("", "All nodes stopped")
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
	for _, node := range c.nodes {
		if node.Status() == NodeStatusRunning {
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
	for _, node := range c.nodes {
		if node.Status() == NodeStatusStopped {
			count++
		}
	}
	return count
}

// CreateNodes は指定された数のノードを作成してクラスタに追加する
func (c *Cluster) CreateNodes(count int, prefix string) error {
	LogInfo("", "Creating %d nodes with prefix '%s'", count, prefix)

	for i := range count {
		nodeID := fmt.Sprintf("%s-%d", prefix, i+1)
		node := NewNode(nodeID)
		if err := c.AddNode(node); err != nil {
			return err
		}
	}

	LogInfo("", "Created %d nodes successfully", count)
	return nil
}
