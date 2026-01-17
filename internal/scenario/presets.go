package scenario

import (
	"time"

	"chaos-kvs/internal/chaos"
)

// BasicScenario は基本的なシナリオ設定を返す
// カオス注入なし、純粋な負荷テスト
func BasicScenario() Config {
	return Config{
		Name:           "basic",
		Description:    "Basic load test without chaos injection",
		Duration:       10 * time.Second,
		NodeCount:      3,
		ClientWorkers:  10,
		WriteRatio:     0.5,
		EnableChaos:    false,
		EnableRecovery: false,
	}
}

// ResilienceScenario は耐障害性テストシナリオを返す
// Kill攻撃のみ、復旧あり
func ResilienceScenario() Config {
	return Config{
		Name:           "resilience",
		Description:    "Resilience test with node kills and recovery",
		Duration:       15 * time.Second,
		NodeCount:      5,
		ClientWorkers:  10,
		WriteRatio:     0.5,
		EnableChaos:    true,
		ChaosInterval:  3 * time.Second,
		ChaosTargets:   1,
		AttackTypes:    []chaos.AttackType{chaos.AttackKill},
		EnableRecovery: true,
		RecoveryDelay:  1 * time.Second,
		MaxRetries:     3,
	}
}

// LatencyScenario はレイテンシ注入シナリオを返す
// Delay攻撃のみ
func LatencyScenario() Config {
	return Config{
		Name:           "latency",
		Description:    "Latency injection test",
		Duration:       10 * time.Second,
		NodeCount:      3,
		ClientWorkers:  10,
		WriteRatio:     0.5,
		EnableChaos:    true,
		ChaosInterval:  2 * time.Second,
		ChaosTargets:   1,
		AttackTypes:    []chaos.AttackType{chaos.AttackDelay},
		EnableRecovery: true,
		RecoveryDelay:  500 * time.Millisecond,
		MaxRetries:     0,
	}
}

// StressScenario は高負荷シナリオを返す
// 多数のワーカー、複数の攻撃タイプ
func StressScenario() Config {
	return Config{
		Name:           "stress",
		Description:    "High load stress test with multiple attack types",
		Duration:       20 * time.Second,
		NodeCount:      7,
		ClientWorkers:  50,
		WriteRatio:     0.3,
		EnableChaos:    true,
		ChaosInterval:  2 * time.Second,
		ChaosTargets:   2,
		AttackTypes:    []chaos.AttackType{chaos.AttackKill, chaos.AttackSuspend, chaos.AttackDelay},
		EnableRecovery: true,
		RecoveryDelay:  500 * time.Millisecond,
		MaxRetries:     5,
	}
}

// QuickScenario はクイックテスト用シナリオを返す
// 短時間での動作確認用
func QuickScenario() Config {
	return Config{
		Name:           "quick",
		Description:    "Quick test for verification",
		Duration:       5 * time.Second,
		NodeCount:      3,
		ClientWorkers:  5,
		WriteRatio:     0.5,
		EnableChaos:    true,
		ChaosInterval:  1 * time.Second,
		ChaosTargets:   1,
		AttackTypes:    []chaos.AttackType{chaos.AttackSuspend},
		EnableRecovery: true,
		RecoveryDelay:  500 * time.Millisecond,
		MaxRetries:     2,
	}
}

// GetPreset は名前からプリセットシナリオを取得する
func GetPreset(name string) (Config, bool) {
	presets := map[string]func() Config{
		"basic":      BasicScenario,
		"resilience": ResilienceScenario,
		"latency":    LatencyScenario,
		"stress":     StressScenario,
		"quick":      QuickScenario,
	}

	if fn, ok := presets[name]; ok {
		return fn(), true
	}
	return Config{}, false
}

// ListPresets は利用可能なプリセット名を返す
func ListPresets() []string {
	return []string{"basic", "resilience", "latency", "stress", "quick"}
}
