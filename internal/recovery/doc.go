// Package recovery はノード障害からの自動復旧機能を提供する。
//
// RecoveryManagerはクラスタ内のノードを監視し、
// 障害が発生した場合に自動的に復旧を試みる。
//
// # 機能
//
// - ヘルスチェック: 定期的にノードの状態を監視
// - 自動再起動: 停止したノードを自動的に再起動
// - 自動再開: 一時停止中のノードを自動的に再開
// - 遅延クリア: 復旧したノードの遅延設定をクリア
//
// # 使用例
//
//	config := recovery.DefaultConfig()
//	config.HealthCheckInterval = 1 * time.Second
//	config.MaxRetries = 3
//
//	manager := recovery.New(cluster, config)
//	manager.Start(ctx)
//	defer manager.Stop()
package recovery
