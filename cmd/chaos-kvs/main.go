// Package main is the entry point for ChaosKVS.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"chaos-kvs/internal/cluster"
	"chaos-kvs/internal/logger"
)

func main() {
	fmt.Println("ChaosKVS - High-Concurrency In-Memory KVS Simulator")
	fmt.Println("====================================================")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// クラスタの作成
	c := cluster.New()

	// 5つのノードを作成
	if err := c.CreateNodes(5, "node"); err != nil {
		logger.Error("", "Failed to create nodes: %v", err)
		os.Exit(1)
	}

	// 全ノードを起動
	if err := c.StartAll(ctx); err != nil {
		logger.Error("", "Failed to start cluster: %v", err)
		os.Exit(1)
	}

	// デモ: データの読み書き
	if node, ok := c.GetNode("node-1"); ok {
		_ = node.Set("hello", []byte("world"))
		if value, exists := node.Get("hello"); exists {
			logger.Info(node.ID(), "Get 'hello' = '%s'", string(value))
		}
	}

	logger.Info("", "Cluster running: %d nodes, %d running",
		c.Size(), c.RunningCount())

	// シグナル待機
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	fmt.Println("\nPress Ctrl+C to stop...")
	<-sigCh

	fmt.Println()
	logger.Info("", "Shutting down...")
	_ = c.StopAll()
	logger.Info("", "Goodbye!")
}
