package config

import (
	"os"
	"path/filepath"
	"testing"

	"chaos-kvs/internal/chaos"
)

func TestLoadFileYAML(t *testing.T) {
	content := `
scenario:
  name: test-scenario
  description: Test scenario
  duration: 10s
  node_count: 5
  client:
    workers: 10
    write_ratio: 0.5
  chaos:
    enabled: true
    interval: 2s
    targets: 1
    attack_types:
      - kill
      - suspend
  recovery:
    enabled: true
    delay: 1s
    max_retries: 3
`
	tmpFile := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}

	cfg, err := LoadFile(tmpFile)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if cfg.Scenario.Name != "test-scenario" {
		t.Errorf("expected name 'test-scenario', got '%s'", cfg.Scenario.Name)
	}
	if cfg.Scenario.NodeCount != 5 {
		t.Errorf("expected node_count 5, got %d", cfg.Scenario.NodeCount)
	}
	if !cfg.Scenario.Chaos.Enabled {
		t.Error("expected chaos to be enabled")
	}
}

func TestLoadFileJSON(t *testing.T) {
	content := `{
  "scenario": {
    "name": "json-test",
    "duration": "5s",
    "node_count": 3,
    "client": {
      "workers": 5
    },
    "chaos": {
      "enabled": false
    },
    "recovery": {
      "enabled": true
    }
  }
}`
	tmpFile := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}

	cfg, err := LoadFile(tmpFile)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if cfg.Scenario.Name != "json-test" {
		t.Errorf("expected name 'json-test', got '%s'", cfg.Scenario.Name)
	}
	if cfg.Scenario.Chaos.Enabled {
		t.Error("expected chaos to be disabled")
	}
}

func TestLoadFileNotFound(t *testing.T) {
	_, err := LoadFile("/nonexistent/config.yaml")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestLoadFileUnsupportedFormat(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "config.txt")
	if err := os.WriteFile(tmpFile, []byte("test"), 0644); err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}

	_, err := LoadFile(tmpFile)
	if err == nil {
		t.Error("expected error for unsupported format")
	}
}

func TestToScenarioConfig(t *testing.T) {
	cfg := &FileConfig{
		Scenario: ScenarioConfig{
			Name:        "test",
			Description: "Test",
			Duration:    "10s",
			NodeCount:   5,
			Client: ClientConfig{
				Workers:    10,
				WriteRatio: 0.7,
			},
			Chaos: ChaosConfig{
				Enabled:     true,
				Interval:    "2s",
				Targets:     2,
				AttackTypes: []string{"kill", "delay"},
			},
			Recovery: RecoveryConfig{
				Enabled:    true,
				Delay:      "1s",
				MaxRetries: 5,
			},
		},
	}

	scenarioCfg, err := cfg.ToScenarioConfig()
	if err != nil {
		t.Fatalf("failed to convert config: %v", err)
	}

	if scenarioCfg.Name != "test" {
		t.Errorf("expected name 'test', got '%s'", scenarioCfg.Name)
	}
	if scenarioCfg.NodeCount != 5 {
		t.Errorf("expected node count 5, got %d", scenarioCfg.NodeCount)
	}
	if scenarioCfg.ClientWorkers != 10 {
		t.Errorf("expected workers 10, got %d", scenarioCfg.ClientWorkers)
	}
	if scenarioCfg.WriteRatio != 0.7 {
		t.Errorf("expected write ratio 0.7, got %f", scenarioCfg.WriteRatio)
	}
	if !scenarioCfg.EnableChaos {
		t.Error("expected chaos to be enabled")
	}
	if len(scenarioCfg.AttackTypes) != 2 {
		t.Errorf("expected 2 attack types, got %d", len(scenarioCfg.AttackTypes))
	}
}

func TestToScenarioConfigInvalidDuration(t *testing.T) {
	cfg := &FileConfig{
		Scenario: ScenarioConfig{
			Duration: "invalid",
		},
	}

	_, err := cfg.ToScenarioConfig()
	if err == nil {
		t.Error("expected error for invalid duration")
	}
}

func TestToScenarioConfigInvalidAttackType(t *testing.T) {
	cfg := &FileConfig{
		Scenario: ScenarioConfig{
			Chaos: ChaosConfig{
				Enabled:     true,
				AttackTypes: []string{"unknown"},
			},
		},
	}

	_, err := cfg.ToScenarioConfig()
	if err == nil {
		t.Error("expected error for invalid attack type")
	}
}

func TestParseAttackTypes(t *testing.T) {
	tests := []struct {
		input    []string
		expected []chaos.AttackType
		hasError bool
	}{
		{[]string{"kill"}, []chaos.AttackType{chaos.AttackKill}, false},
		{[]string{"suspend"}, []chaos.AttackType{chaos.AttackSuspend}, false},
		{[]string{"delay"}, []chaos.AttackType{chaos.AttackDelay}, false},
		{[]string{"KILL", "SUSPEND"}, []chaos.AttackType{chaos.AttackKill, chaos.AttackSuspend}, false},
		{[]string{"unknown"}, nil, true},
	}

	for _, tt := range tests {
		attacks, err := parseAttackTypes(tt.input)
		if tt.hasError {
			if err == nil {
				t.Errorf("expected error for input %v", tt.input)
			}
			continue
		}
		if err != nil {
			t.Errorf("unexpected error for input %v: %v", tt.input, err)
			continue
		}
		if len(attacks) != len(tt.expected) {
			t.Errorf("expected %d attacks, got %d", len(tt.expected), len(attacks))
		}
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name     string
		config   FileConfig
		hasError bool
	}{
		{
			name:     "valid config",
			config:   FileConfig{},
			hasError: false,
		},
		{
			name: "negative node count",
			config: FileConfig{
				Scenario: ScenarioConfig{NodeCount: -1},
			},
			hasError: true,
		},
		{
			name: "negative workers",
			config: FileConfig{
				Scenario: ScenarioConfig{Client: ClientConfig{Workers: -1}},
			},
			hasError: true,
		},
		{
			name: "invalid write ratio (too high)",
			config: FileConfig{
				Scenario: ScenarioConfig{Client: ClientConfig{WriteRatio: 1.5}},
			},
			hasError: true,
		},
		{
			name: "invalid write ratio (negative)",
			config: FileConfig{
				Scenario: ScenarioConfig{Client: ClientConfig{WriteRatio: -0.1}},
			},
			hasError: true,
		},
		{
			name: "negative chaos targets",
			config: FileConfig{
				Scenario: ScenarioConfig{Chaos: ChaosConfig{Targets: -1}},
			},
			hasError: true,
		},
		{
			name: "negative max retries",
			config: FileConfig{
				Scenario: ScenarioConfig{Recovery: RecoveryConfig{MaxRetries: -1}},
			},
			hasError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.hasError && err == nil {
				t.Error("expected validation error")
			}
			if !tt.hasError && err != nil {
				t.Errorf("unexpected validation error: %v", err)
			}
		})
	}
}
