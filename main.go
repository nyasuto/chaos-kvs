package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	fmt.Println("ChaosKVS - High-Concurrency In-Memory KVS Simulator")
	fmt.Println("====================================================")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// クラスタの作成
	cluster := NewCluster()

	// 5つのノードを作成
	if err := cluster.CreateNodes(5, "node"); err != nil {
		LogError("", "Failed to create nodes: %v", err)
		os.Exit(1)
	}

	// 全ノードを起動
	if err := cluster.StartAll(ctx); err != nil {
		LogError("", "Failed to start cluster: %v", err)
		os.Exit(1)
	}

	// デモ: データの読み書き
	if node, ok := cluster.GetNode("node-1"); ok {
		_ = node.Set("hello", []byte("world"))
		if value, exists := node.Get("hello"); exists {
			LogInfo(node.ID, "Get 'hello' = '%s'", string(value))
		}
	}

	LogInfo("", "Cluster running: %d nodes, %d running",
		cluster.Size(), cluster.RunningCount())

	// シグナル待機
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	fmt.Println("\nPress Ctrl+C to stop...")
	<-sigCh

	fmt.Println()
	LogInfo("", "Shutting down...")
	_ = cluster.StopAll()
	LogInfo("", "Goodbye!")
}
