// Package main is the entry point for ChaosKVS.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"chaos-kvs/internal/api"
	"chaos-kvs/internal/config"
	"chaos-kvs/internal/logger"
	"chaos-kvs/internal/scenario"
)

var (
	version = "dev"
)

func main() {
	// フラグ定義
	var (
		configFile     = flag.String("config", "", "設定ファイルパス (YAML/JSON)")
		presetName     = flag.String("preset", "", "プリセットシナリオ名 (basic, resilience, latency, stress, quick)")
		duration       = flag.Duration("duration", 0, "シナリオ実行時間 (例: 10s, 1m)")
		nodes          = flag.Int("nodes", 0, "ノード数")
		workers        = flag.Int("workers", 0, "クライアントワーカー数")
		enableChaos    = flag.Bool("chaos", true, "カオス注入を有効化")
		enableRecovery = flag.Bool("recovery", true, "自動復旧を有効化")
		listPresets    = flag.Bool("list-presets", false, "利用可能なプリセットを表示")
		showVersion    = flag.Bool("version", false, "バージョンを表示")
		serverMode     = flag.Bool("server", false, "Web UI サーバーモードで起動")
		serverAddr     = flag.String("addr", ":8080", "サーバーアドレス (例: :8080, 0.0.0.0:3000)")
	)

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, `ChaosKVS - High-Concurrency In-Memory KVS Simulator

Usage:
  chaos-kvs [options]

Options:
`)
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, `
Examples:
  # プリセットシナリオを実行
  chaos-kvs --preset quick

  # 設定ファイルから実行
  chaos-kvs --config scenario.yaml

  # フラグでカスタマイズ
  chaos-kvs --preset basic --duration 30s --nodes 10

  # プリセット一覧を表示
  chaos-kvs --list-presets

  # Web UIサーバーモードで起動
  chaos-kvs --server

  # カスタムアドレスでサーバー起動
  chaos-kvs --server --addr :3000
`)
	}

	flag.Parse()

	// バージョン表示
	if *showVersion {
		fmt.Printf("chaos-kvs version %s\n", version)
		return
	}

	// プリセット一覧表示
	if *listPresets {
		printPresets()
		return
	}

	// Web UIサーバーモード
	if *serverMode {
		if err := runServer(*serverAddr); err != nil {
			logger.Error("", "サーバーエラー: %v", err)
			os.Exit(1)
		}
		return
	}

	// シナリオ設定の決定
	scenarioConfig, err := buildScenarioConfig(
		*configFile, *presetName, *duration, *nodes, *workers, *enableChaos, *enableRecovery,
	)
	if err != nil {
		logger.Error("", "設定エラー: %v", err)
		os.Exit(1)
	}

	// シナリオ実行
	if err := runScenario(scenarioConfig); err != nil {
		logger.Error("", "シナリオ実行エラー: %v", err)
		os.Exit(1)
	}
}

// buildScenarioConfig はシナリオ設定を構築する
func buildScenarioConfig(
	configFile, presetName string,
	duration time.Duration, nodes, workers int,
	enableChaos, enableRecovery bool,
) (scenario.Config, error) {
	var cfg scenario.Config

	// 1. 設定ファイルから読み込み
	if configFile != "" {
		fileConfig, err := config.LoadFile(configFile)
		if err != nil {
			return cfg, fmt.Errorf("設定ファイル読み込みエラー: %w", err)
		}
		if err := fileConfig.Validate(); err != nil {
			return cfg, fmt.Errorf("設定検証エラー: %w", err)
		}
		cfg, err = fileConfig.ToScenarioConfig()
		if err != nil {
			return cfg, fmt.Errorf("設定変換エラー: %w", err)
		}
	} else if presetName != "" {
		// 2. プリセットから読み込み
		preset, ok := scenario.GetPreset(presetName)
		if !ok {
			return cfg, fmt.Errorf("不明なプリセット: %s (利用可能: %v)", presetName, scenario.ListPresets())
		}
		cfg = preset
	} else {
		// 3. デフォルト（quickシナリオ）
		cfg = scenario.QuickScenario()
	}

	// フラグでオーバーライド
	if duration > 0 {
		cfg.Duration = duration
	}
	if nodes > 0 {
		cfg.NodeCount = nodes
	}
	if workers > 0 {
		cfg.ClientWorkers = workers
	}

	// フラグが明示的に指定された場合のみオーバーライド
	cfg.EnableChaos = enableChaos
	cfg.EnableRecovery = enableRecovery

	return cfg, nil
}

// runScenario はシナリオを実行する
func runScenario(cfg scenario.Config) error {
	fmt.Println("ChaosKVS - High-Concurrency In-Memory KVS Simulator")
	fmt.Println("====================================================")
	fmt.Printf("Scenario: %s\n", cfg.Name)
	fmt.Printf("Duration: %v\n", cfg.Duration)
	fmt.Printf("Nodes: %d, Workers: %d\n", cfg.NodeCount, cfg.ClientWorkers)
	fmt.Printf("Chaos: %v, Recovery: %v\n", cfg.EnableChaos, cfg.EnableRecovery)
	fmt.Println("====================================================")
	fmt.Println()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// シグナルハンドリング
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigCh
		fmt.Println("\n中断シグナルを受信、シナリオを終了中...")
		cancel()
	}()

	// シナリオ実行
	engine := scenario.New(cfg)
	result, err := engine.Run(ctx)
	if err != nil {
		return err
	}

	// レポート出力
	fmt.Println(result.Report())

	return nil
}

// printPresets は利用可能なプリセットを表示する
func printPresets() {
	fmt.Println("利用可能なプリセットシナリオ:")
	fmt.Println()

	presets := []struct {
		name string
		desc string
	}{
		{"basic", "カオスなしの基本負荷テスト"},
		{"resilience", "ノードkillと復旧のテスト"},
		{"latency", "レイテンシ注入テスト"},
		{"stress", "高負荷ストレステスト"},
		{"quick", "短時間の動作確認（デフォルト）"},
	}

	for _, p := range presets {
		fmt.Printf("  %-12s %s\n", p.name, p.desc)
	}

	fmt.Println()
	fmt.Println("使用例: chaos-kvs --preset quick")
}

// runServer はWeb UIサーバーを起動する
func runServer(addr string) error {
	fmt.Println("ChaosKVS - Web UI Server")
	fmt.Println("========================")
	fmt.Printf("Starting server on http://%s\n", addr)
	fmt.Println("Press Ctrl+C to stop")
	fmt.Println()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// シグナルハンドリング
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigCh
		fmt.Println("\n中断シグナルを受信、サーバーを終了中...")
		cancel()
	}()

	server := api.NewServer(addr)
	return server.Start(ctx)
}
