package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"chaos-kvs/internal/chaos"
	"chaos-kvs/internal/scenario"

	"gopkg.in/yaml.v3"
)

// FileConfig は設定ファイルの構造
type FileConfig struct {
	Scenario ScenarioConfig `yaml:"scenario" json:"scenario"`
}

// ScenarioConfig はシナリオ設定
type ScenarioConfig struct {
	Name        string `yaml:"name" json:"name"`
	Description string `yaml:"description" json:"description"`
	Duration    string `yaml:"duration" json:"duration"`
	NodeCount   int    `yaml:"node_count" json:"node_count"`

	Client   ClientConfig   `yaml:"client" json:"client"`
	Chaos    ChaosConfig    `yaml:"chaos" json:"chaos"`
	Recovery RecoveryConfig `yaml:"recovery" json:"recovery"`
}

// ClientConfig はクライアント設定
type ClientConfig struct {
	Workers    int     `yaml:"workers" json:"workers"`
	WriteRatio float64 `yaml:"write_ratio" json:"write_ratio"`
}

// ChaosConfig はカオス設定
type ChaosConfig struct {
	Enabled     bool     `yaml:"enabled" json:"enabled"`
	Interval    string   `yaml:"interval" json:"interval"`
	Targets     int      `yaml:"targets" json:"targets"`
	AttackTypes []string `yaml:"attack_types" json:"attack_types"`
	SuspendTime string   `yaml:"suspend_time" json:"suspend_time"`
	DelayAmount string   `yaml:"delay_amount" json:"delay_amount"`
}

// RecoveryConfig は復旧設定
type RecoveryConfig struct {
	Enabled    bool   `yaml:"enabled" json:"enabled"`
	Delay      string `yaml:"delay" json:"delay"`
	MaxRetries int    `yaml:"max_retries" json:"max_retries"`
}

// LoadFile は設定ファイルを読み込む
func LoadFile(path string) (*FileConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config FileConfig
	ext := strings.ToLower(filepath.Ext(path))

	switch ext {
	case ".yaml", ".yml":
		if err := yaml.Unmarshal(data, &config); err != nil {
			return nil, fmt.Errorf("failed to parse YAML: %w", err)
		}
	case ".json":
		if err := json.Unmarshal(data, &config); err != nil {
			return nil, fmt.Errorf("failed to parse JSON: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported config format: %s", ext)
	}

	return &config, nil
}

// ToScenarioConfig はFileConfigをscenario.Configに変換する
func (f *FileConfig) ToScenarioConfig() (scenario.Config, error) {
	sc := f.Scenario

	// デフォルト値の設定
	config := scenario.DefaultConfig()

	if sc.Name != "" {
		config.Name = sc.Name
	}
	if sc.Description != "" {
		config.Description = sc.Description
	}
	if sc.Duration != "" {
		d, err := time.ParseDuration(sc.Duration)
		if err != nil {
			return config, fmt.Errorf("invalid duration: %w", err)
		}
		config.Duration = d
	}
	if sc.NodeCount > 0 {
		config.NodeCount = sc.NodeCount
	}

	// Client設定
	if sc.Client.Workers > 0 {
		config.ClientWorkers = sc.Client.Workers
	}
	if sc.Client.WriteRatio > 0 {
		config.WriteRatio = sc.Client.WriteRatio
	}

	// Chaos設定
	config.EnableChaos = sc.Chaos.Enabled
	if sc.Chaos.Interval != "" {
		d, err := time.ParseDuration(sc.Chaos.Interval)
		if err != nil {
			return config, fmt.Errorf("invalid chaos interval: %w", err)
		}
		config.ChaosInterval = d
	}
	if sc.Chaos.Targets > 0 {
		config.ChaosTargets = sc.Chaos.Targets
	}
	if len(sc.Chaos.AttackTypes) > 0 {
		attacks, err := parseAttackTypes(sc.Chaos.AttackTypes)
		if err != nil {
			return config, err
		}
		config.AttackTypes = attacks
	}

	// Recovery設定
	config.EnableRecovery = sc.Recovery.Enabled
	if sc.Recovery.Delay != "" {
		d, err := time.ParseDuration(sc.Recovery.Delay)
		if err != nil {
			return config, fmt.Errorf("invalid recovery delay: %w", err)
		}
		config.RecoveryDelay = d
	}
	if sc.Recovery.MaxRetries > 0 {
		config.MaxRetries = sc.Recovery.MaxRetries
	}

	return config, nil
}

// parseAttackTypes は文字列の攻撃タイプをパースする
func parseAttackTypes(types []string) ([]chaos.AttackType, error) {
	var attacks []chaos.AttackType

	for _, t := range types {
		switch strings.ToLower(t) {
		case "kill":
			attacks = append(attacks, chaos.AttackKill)
		case "suspend":
			attacks = append(attacks, chaos.AttackSuspend)
		case "delay":
			attacks = append(attacks, chaos.AttackDelay)
		default:
			return nil, fmt.Errorf("unknown attack type: %s", t)
		}
	}

	return attacks, nil
}

// Validate は設定を検証する
func (f *FileConfig) Validate() error {
	sc := f.Scenario

	if sc.NodeCount < 0 {
		return fmt.Errorf("node_count must be non-negative")
	}

	if sc.Client.Workers < 0 {
		return fmt.Errorf("client.workers must be non-negative")
	}

	if sc.Client.WriteRatio < 0 || sc.Client.WriteRatio > 1 {
		return fmt.Errorf("client.write_ratio must be between 0 and 1")
	}

	if sc.Chaos.Targets < 0 {
		return fmt.Errorf("chaos.targets must be non-negative")
	}

	if sc.Recovery.MaxRetries < 0 {
		return fmt.Errorf("recovery.max_retries must be non-negative")
	}

	return nil
}
