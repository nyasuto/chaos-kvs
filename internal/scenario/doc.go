// Package scenario は統合シナリオ実行機能を提供する。
//
// シナリオエンジンはChaosMonkey、RecoveryManager、Clientを
// 連携させて統合テストを実行する。
//
// # 機能
//
// - シナリオ定義と実行
// - 定義済みプリセットシナリオ
// - 実行結果のレポート生成
//
// # プリセットシナリオ
//
// - basic: カオスなしの基本負荷テスト
// - resilience: ノードkillと復旧のテスト
// - latency: レイテンシ注入テスト
// - stress: 高負荷ストレステスト
// - quick: 短時間の動作確認
//
// # 使用例
//
//	config := scenario.ResilienceScenario()
//	engine := scenario.New(config)
//	result, err := engine.Run(ctx)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Println(result.Report())
package scenario
