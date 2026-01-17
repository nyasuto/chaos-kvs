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

	// デモ: 単一ノードの起動
	node := NewNode("node-1")
	if err := node.Start(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "failed to start node: %v\n", err)
		os.Exit(1)
	}

	// デモ: データの読み書き
	_ = node.Set("hello", []byte("world"))
	if value, ok := node.Get("hello"); ok {
		fmt.Printf("[INFO] Get 'hello' = '%s'\n", string(value))
	}

	fmt.Printf("[INFO] Node %s is running with %d keys\n", node.ID, node.Size())

	// シグナル待機
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	fmt.Println("[INFO] Press Ctrl+C to stop...")
	<-sigCh

	fmt.Println("\n[INFO] Shutting down...")
	_ = node.Stop()
	fmt.Println("[INFO] Goodbye!")
}
