package config

import (
	"os"
	"path/filepath"
	"testing"

	"log/slog"
)

func TestLoad_ValidConfig(t *testing.T) {
	t.Setenv("TEST_OGAMEX_PASSWORD", "mysecret123")

	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	err := os.WriteFile(cfgPath, []byte(validYAML), 0644)
	if err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	cfg, err := Load(cfgPath, log)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.OGameX.URL != "https://ogamex.example.com" {
		t.Errorf("OGameX.URL = %q, want %q", cfg.OGameX.URL, "https://ogamex.example.com")
	}
	if cfg.OGameX.Email != "testuser@example.com" {
		t.Errorf("OGameX.Email = %q, want %q", cfg.OGameX.Email, "testuser@example.com")
	}
	if cfg.OGameX.Password != "mysecret123" {
		t.Errorf("OGameX.Password = %q, want %q", cfg.OGameX.Password, "mysecret123")
	}
	if cfg.Features.Defender.Enabled != false {
		t.Error("Features.Defender.Enabled = true, want false")
	}
	if cfg.Features.Defender.PollIntervalMs != 30000 {
		t.Errorf("Features.Defender.PollIntervalMs = %d, want 30000", cfg.Features.Defender.PollIntervalMs)
	}
	if cfg.RateLimit.DefaultMinDelayMs != 2000 {
		t.Errorf("RateLimit.DefaultMinDelayMs = %d, want 2000", cfg.RateLimit.DefaultMinDelayMs)
	}
	if cfg.RateLimit.DefaultMaxDelayMs != 5000 {
		t.Errorf("RateLimit.DefaultMaxDelayMs = %d, want 5000", cfg.RateLimit.DefaultMaxDelayMs)
	}
	if cfg.LogLevel != "info" {
		t.Errorf("LogLevel = %q, want %q", cfg.LogLevel, "info")
	}
}

func TestLoad_EnvInterpolation(t *testing.T) {
	t.Setenv("TEST_OGAMEX_PASS", "secret123")

	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	yaml := `
ogamex:
  url: "https://ogamex.example.com"
  email: "testuser@example.com"
  password: "${TEST_OGAMEX_PASS}"
features:
  defender:
    enabled: false
    pollIntervalMs: 30000
rateLimit:
  defaultMinDelayMs: 2000
  defaultMaxDelayMs: 5000
logLevel: "info"
`
	err := os.WriteFile(cfgPath, []byte(yaml), 0644)
	if err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	cfg, err := Load(cfgPath, log)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.OGameX.Password != "secret123" {
		t.Errorf("Password = %q, want %q", cfg.OGameX.Password, "secret123")
	}
}

func TestLoad_MissingEnvVar(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	yaml := `
ogamex:
  url: "https://ogamex.example.com"
  email: "testuser@example.com"
  password: "${NONEXISTENT_VAR_12345}"
features:
  defender:
    enabled: false
    pollIntervalMs: 30000
rateLimit:
  defaultMinDelayMs: 2000
  defaultMaxDelayMs: 5000
logLevel: "info"
`
	err := os.WriteFile(cfgPath, []byte(yaml), 0644)
	if err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	cfg, err := Load(cfgPath, log)
	if err != nil {
		t.Fatalf("Load() should succeed with unresolved env var (left as literal), got: %v", err)
	}

	if cfg.OGameX.Password != "${NONEXISTENT_VAR_12345}" {
		t.Errorf("Password = %q, want literal ${NONEXISTENT_VAR_12345}", cfg.OGameX.Password)
	}
}

func TestValidate_RequiredFields(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr string
	}{
		{
			name: "missing url",
			config: Config{
				OGameX:    OGameXConfig{URL: "", Email: "u@example.com", Password: "p"},
				RateLimit: RateLimitConfig{DefaultMinDelayMs: 1000, DefaultMaxDelayMs: 2000},
			},
			wantErr: "ogamex.url",
		},
		{
			name: "missing email",
			config: Config{
				OGameX:    OGameXConfig{URL: "https://ogamex.example.com", Email: "", Password: "p"},
				RateLimit: RateLimitConfig{DefaultMinDelayMs: 1000, DefaultMaxDelayMs: 2000},
			},
			wantErr: "ogamex.email",
		},
		{
			name: "missing password",
			config: Config{
				OGameX:    OGameXConfig{URL: "https://ogamex.example.com", Email: "u@example.com", Password: ""},
				RateLimit: RateLimitConfig{DefaultMinDelayMs: 1000, DefaultMaxDelayMs: 2000},
			},
			wantErr: "ogamex.password",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if err == nil {
				t.Fatalf("Validate() should fail, got nil error")
			}
		})
	}
}

func TestValidate_RateLimitMin(t *testing.T) {
	cfg := Config{
		OGameX:    OGameXConfig{URL: "https://ogamex.example.com", Email: "u@example.com", Password: "p"},
		RateLimit: RateLimitConfig{DefaultMinDelayMs: 100, DefaultMaxDelayMs: 200},
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("Validate() should fail with low rate limit, got nil error")
	}
}

func TestValidate_RateLimitMaxLessThanMin(t *testing.T) {
	cfg := Config{
		OGameX:    OGameXConfig{URL: "https://ogamex.example.com", Email: "u@example.com", Password: "p"},
		RateLimit: RateLimitConfig{DefaultMinDelayMs: 5000, DefaultMaxDelayMs: 3000},
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("Validate() should fail when max < min, got nil error")
	}
}

func TestLoad_MissingFile(t *testing.T) {
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	_, err := Load("/nonexistent/path/config.yaml", log)
	if err == nil {
		t.Fatal("Load() should fail with missing file, got nil error")
	}
}

// validYAML is a valid config matching config.example.yaml structure
const validYAML = `
ogamex:
  url: "https://ogamex.example.com"
  email: "testuser@example.com"
  password: "${TEST_OGAMEX_PASSWORD}"
features:
  defender:
    enabled: false
    pollIntervalMs: 30000
rateLimit:
  defaultMinDelayMs: 2000
  defaultMaxDelayMs: 5000
logLevel: "info"
`

func TestDefenderConfig_LoadWithFields(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	yaml := `
ogamex:
  url: "https://ogamex.example.com"
  email: "u@example.com"
  password: "p"
features:
  defender:
    enabled: true
    pollIntervalMs: 15000
    safetyMarginMs: 60000
    recallEnabled: false
    maxReturnFlightS: 300
    minReactionDelayS: 10
    maxReactionDelayS: 60
rateLimit:
  defaultMinDelayMs: 1000
  defaultMaxDelayMs: 2000
logLevel: "debug"
`
	err := os.WriteFile(cfgPath, []byte(yaml), 0644)
	if err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	cfg, err := Load(cfgPath, log)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Features.Defender.Enabled != true {
		t.Error("Defender.Enabled = false, want true")
	}
	if cfg.Features.Defender.PollIntervalMs != 15000 {
		t.Errorf("Defender.PollIntervalMs = %d, want 15000", cfg.Features.Defender.PollIntervalMs)
	}
	if cfg.Features.Defender.SafetyMarginMs != 60000 {
		t.Errorf("Defender.SafetyMarginMs = %d, want 60000", cfg.Features.Defender.SafetyMarginMs)
	}
	if cfg.Features.Defender.RecallEnabled == nil || *cfg.Features.Defender.RecallEnabled != false {
		t.Error("Defender.RecallEnabled should be false")
	}
	if cfg.Features.Defender.MaxReturnFlightS != 300 {
		t.Errorf("Defender.MaxReturnFlightS = %d, want 300", cfg.Features.Defender.MaxReturnFlightS)
	}
	if cfg.Features.Defender.MinReactionDelayS != 10 {
		t.Errorf("Defender.MinReactionDelayS = %d, want 10", cfg.Features.Defender.MinReactionDelayS)
	}
	if cfg.Features.Defender.MaxReactionDelayS != 60 {
		t.Errorf("Defender.MaxReactionDelayS = %d, want 60", cfg.Features.Defender.MaxReactionDelayS)
	}
}

func TestDefenderConfig_Defaults(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	yaml := `
ogamex:
  url: "https://ogamex.example.com"
  email: "u@example.com"
  password: "p"
features:
  defender:
    enabled: true
    pollIntervalMs: 30000
rateLimit:
  defaultMinDelayMs: 1000
  defaultMaxDelayMs: 2000
logLevel: "info"
`
	err := os.WriteFile(cfgPath, []byte(yaml), 0644)
	if err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	cfg, err := Load(cfgPath, log)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Defaults should be applied when zero
	if cfg.Features.Defender.SafetyMarginMs != 120000 {
		t.Errorf("Defender.SafetyMarginMs default = %d, want 120000", cfg.Features.Defender.SafetyMarginMs)
	}
	if cfg.Features.Defender.RecallEnabled == nil || *cfg.Features.Defender.RecallEnabled != true {
		t.Error("Defender.RecallEnabled default should be true")
	}
	if cfg.Features.Defender.MaxReturnFlightS != 600 {
		t.Errorf("Defender.MaxReturnFlightS default = %d, want 600", cfg.Features.Defender.MaxReturnFlightS)
	}
	if cfg.Features.Defender.MinReactionDelayS != 30 {
		t.Errorf("Defender.MinReactionDelayS default = %d, want 30", cfg.Features.Defender.MinReactionDelayS)
	}
	if cfg.Features.Defender.MaxReactionDelayS != 120 {
		t.Errorf("Defender.MaxReactionDelayS default = %d, want 120", cfg.Features.Defender.MaxReactionDelayS)
	}
}

func TestDefenderConfig_Validation(t *testing.T) {
	tests := []struct {
		name    string
		def     DefenderConfig
		wantErr string
	}{
		{
			name: "safety margin too low",
			def: DefenderConfig{
				FeatureConfig:     FeatureConfig{Enabled: true, PollIntervalMs: 30000},
				SafetyMarginMs:    5000,
				MinReactionDelayS: 30,
				MaxReactionDelayS: 120,
			},
			wantErr: "safetyMarginMs",
		},
		{
			name: "min reaction delay too low",
			def: DefenderConfig{
				FeatureConfig:     FeatureConfig{Enabled: true, PollIntervalMs: 30000},
				SafetyMarginMs:    120000,
				MinReactionDelayS: 2,
				MaxReactionDelayS: 120,
			},
			wantErr: "minReactionDelayS",
		},
		{
			name: "max reaction delay less than min",
			def: DefenderConfig{
				FeatureConfig:     FeatureConfig{Enabled: true, PollIntervalMs: 30000},
				SafetyMarginMs:    120000,
				MinReactionDelayS: 60,
				MaxReactionDelayS: 30,
			},
			wantErr: "maxReactionDelayS",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Config{
				OGameX:    OGameXConfig{URL: "https://ogamex.example.com", Email: "u@example.com", Password: "p"},
				Features:  FeaturesConfig{Defender: tt.def},
				RateLimit: RateLimitConfig{DefaultMinDelayMs: 1000, DefaultMaxDelayMs: 2000},
			}
			err := cfg.Validate()
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tt.wantErr)
			}
		})
	}
}
