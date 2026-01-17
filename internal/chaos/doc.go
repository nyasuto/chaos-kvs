// Package chaos はカオスエンジニアリング機能を提供する。
//
// ChaosMonkeyはクラスタ内のノードに対して様々な障害を注入し、
// システムの耐障害性をテストするために使用される。
//
// # 障害タイプ
//
// - Kill: ノードを強制停止
// - Suspend: ノードを一時停止（リクエストを受け付けなくなる）
// - Delay: ノードのレスポンスに遅延を注入
//
// # 使用例
//
//	config := chaos.DefaultConfig()
//	config.Interval = 3 * time.Second
//	config.TargetCount = 2
//
//	monkey := chaos.New(cluster, config)
//	monkey.Start(ctx)
//	defer monkey.Stop()
package chaos
