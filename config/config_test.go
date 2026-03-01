package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadWithDefaults(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir := t.TempDir()
	
	// Test loading with no config file - should use defaults
	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Load() with no config should not error, got: %v", err)
	}
	
	// Verify defaults
	if cfg.Redis.Address != "localhost:6379" {
		t.Errorf("Expected default address 'localhost:6379', got '%s'", cfg.Redis.Address)
	}
	if cfg.Redis.DB != 0 {
		t.Errorf("Expected default DB 0, got %d", cfg.Redis.DB)
	}
	if cfg.Scanner.BatchSize != 100 {
		t.Errorf("Expected default batch_size 100, got %d", cfg.Scanner.BatchSize)
	}
	if cfg.Scanner.Workers != 20 {
		t.Errorf("Expected default workers 20, got %d", cfg.Scanner.Workers)
	}
	if cfg.Output.Format != "table" {
		t.Errorf("Expected default format 'table', got '%s'", cfg.Output.Format)
	}
	if cfg.Output.CliffThreshold != 20 {
		t.Errorf("Expected default cliff_threshold 20, got %d", cfg.Output.CliffThreshold)
	}
	if cfg.Output.CliffWindowMins != 5 {
		t.Errorf("Expected default cliff_window_minutes 5, got %d", cfg.Output.CliffWindowMins)
	}
	
	_ = tmpDir // Use tmpDir to avoid unused variable warning
}

func TestLoadWithCustomConfig(t *testing.T) {
	// Create a temporary config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test-config.yaml")
	
	configContent := `redis:
  address: redis.example.com:6380
  password: secret123
  db: 2
  tls: true

scanner:
  batch_size: 200
  workers: 50

groups:
  - name: "Test Group"
    pattern: "test:*"

output:
  format: json
  sort_by: keys
  cliff_threshold: 30
  cliff_window_minutes: 10
  cliff_ignore_groups:
    - "Test Group"
`
	
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}
	
	// Load the custom config
	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}
	
	// Verify custom values
	if cfg.Redis.Address != "redis.example.com:6380" {
		t.Errorf("Expected address 'redis.example.com:6380', got '%s'", cfg.Redis.Address)
	}
	if cfg.Redis.Password != "secret123" {
		t.Errorf("Expected password 'secret123', got '%s'", cfg.Redis.Password)
	}
	if cfg.Redis.DB != 2 {
		t.Errorf("Expected DB 2, got %d", cfg.Redis.DB)
	}
	if !cfg.Redis.TLS {
		t.Error("Expected TLS to be true")
	}
	if cfg.Scanner.BatchSize != 200 {
		t.Errorf("Expected batch_size 200, got %d", cfg.Scanner.BatchSize)
	}
	if cfg.Scanner.Workers != 50 {
		t.Errorf("Expected workers 50, got %d", cfg.Scanner.Workers)
	}
	if len(cfg.Groups) != 1 {
		t.Errorf("Expected 1 group, got %d", len(cfg.Groups))
	} else {
		if cfg.Groups[0].Name != "Test Group" {
			t.Errorf("Expected group name 'Test Group', got '%s'", cfg.Groups[0].Name)
		}
		if cfg.Groups[0].Pattern != "test:*" {
			t.Errorf("Expected pattern 'test:*', got '%s'", cfg.Groups[0].Pattern)
		}
	}
	if cfg.Output.Format != "json" {
		t.Errorf("Expected format 'json', got '%s'", cfg.Output.Format)
	}
	if cfg.Output.CliffThreshold != 30 {
		t.Errorf("Expected cliff_threshold 30, got %d", cfg.Output.CliffThreshold)
	}
	if cfg.Output.CliffWindowMins != 10 {
		t.Errorf("Expected cliff_window_minutes 10, got %d", cfg.Output.CliffWindowMins)
	}
	if len(cfg.Output.CliffIgnoreGroups) != 1 || cfg.Output.CliffIgnoreGroups[0] != "Test Group" {
		t.Errorf("Expected cliff_ignore_groups ['Test Group'], got %v", cfg.Output.CliffIgnoreGroups)
	}
}

func TestLoadWithInvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "invalid.yaml")
	
	// Write invalid YAML
	invalidYAML := `redis:
  address: localhost:6379
  invalid yaml here [[[
`
	
	if err := os.WriteFile(configPath, []byte(invalidYAML), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}
	
	// Should return error for invalid YAML
	_, err := Load(configPath)
	if err == nil {
		t.Error("Expected error for invalid YAML, got nil")
	}
}

func TestLoadWithNonExistentFile(t *testing.T) {
	// Explicitly specify a non-existent file
	_, err := Load("/tmp/definitely-does-not-exist-redis-profiler-test.yaml")
	if err == nil {
		t.Error("Expected error for non-existent file, got nil")
	}
}

func TestLoadWithPartialConfig(t *testing.T) {
	// Test that partial config merges with defaults
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "partial.yaml")
	
	partialConfig := `redis:
  address: custom.redis:6379

scanner:
  workers: 10
`
	
	if err := os.WriteFile(configPath, []byte(partialConfig), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}
	
	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}
	
	// Verify custom values
	if cfg.Redis.Address != "custom.redis:6379" {
		t.Errorf("Expected custom address, got '%s'", cfg.Redis.Address)
	}
	if cfg.Scanner.Workers != 10 {
		t.Errorf("Expected custom workers 10, got %d", cfg.Scanner.Workers)
	}
	
	// Verify defaults are still applied
	if cfg.Scanner.BatchSize != 100 {
		t.Errorf("Expected default batch_size 100, got %d", cfg.Scanner.BatchSize)
	}
	if cfg.Redis.DB != 0 {
		t.Errorf("Expected default DB 0, got %d", cfg.Redis.DB)
	}
}

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *Config
		wantErr bool
	}{
		{
			name: "valid config",
			cfg: &Config{
				Redis: RedisConfig{
					Address: "localhost:6379",
					DB:      0,
				},
				Scanner: ScannerConfig{
					BatchSize: 100,
					Workers:   20,
				},
			},
			wantErr: false,
		},
		{
			name: "negative batch size",
			cfg: &Config{
				Scanner: ScannerConfig{
					BatchSize: -1,
					Workers:   20,
				},
			},
			wantErr: true,
		},
		{
			name: "zero workers",
			cfg: &Config{
				Scanner: ScannerConfig{
					BatchSize: 100,
					Workers:   0,
				},
			},
			wantErr: true,
		},
		{
			name: "negative workers",
			cfg: &Config{
				Scanner: ScannerConfig{
					BatchSize: 100,
					Workers:   -5,
				},
			},
			wantErr: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
