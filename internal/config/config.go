// Package config handles YAML configuration loading with environment
// variable interpolation and validation.
package config

import (
	"fmt"
	"log/slog"
	"os"
	"regexp"

	"gopkg.in/yaml.v3"
)

// Config is the top-level bot configuration.
type Config struct {
	Account   AccountConfig   `yaml:"account"`
	Ogamed    OgamedConfig    `yaml:"ogamed"`
	Features  FeaturesConfig  `yaml:"features"`
	RateLimit RateLimitConfig `yaml:"rateLimit"`
	LogLevel  string          `yaml:"logLevel"`
}

// AccountConfig holds OGame account credentials.
type AccountConfig struct {
	Universe string `yaml:"universe"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

// OgamedConfig holds the ogamed REST API connection settings.
type OgamedConfig struct {
	URL string `yaml:"url"`
}

// FeatureConfig holds the enabled state and poll interval for a bot feature.
type FeatureConfig struct {
	Enabled        bool `yaml:"enabled"`
	PollIntervalMs int  `yaml:"pollIntervalMs"`
}

// FeaturesConfig holds per-feature toggle settings.
type FeaturesConfig struct {
	Defender  FeatureConfig `yaml:"defender"`
	AutoBuild FeatureConfig `yaml:"autoBuild"`
	AutoFarm  FeatureConfig `yaml:"autoFarm"`
}

// RateLimitConfig holds rate limiting configuration for ogamed API calls.
type RateLimitConfig struct {
	DefaultMinDelayMs int                            `yaml:"defaultMinDelayMs"`
	DefaultMaxDelayMs int                            `yaml:"defaultMaxDelayMs"`
	EndpointOverrides map[string]EndpointDelayConfig `yaml:"endpointOverrides"`
}

// EndpointDelayConfig holds per-endpoint delay overrides.
type EndpointDelayConfig struct {
	MinMs int `yaml:"minMs"`
	MaxMs int `yaml:"maxMs"`
}

var envVarPattern = regexp.MustCompile(`\$\{(\w+)\}`)

// Load reads a YAML config file, interpolates ${ENV_VAR} references from
// the environment, parses the YAML, and validates required fields.
func Load(path string, log *slog.Logger) (*Config, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config file: %w", err)
	}

	// Interpolate ${ENV_VAR} references from environment.
	// Missing env vars cause an immediate error with the variable name.
	var interpolationErr error
	interpolated := envVarPattern.ReplaceAllStringFunc(string(raw), func(match string) string {
		if interpolationErr != nil {
			return match // short-circuit on first error
		}
		varName := envVarPattern.FindStringSubmatch(match)[1]
		value, ok := os.LookupEnv(varName)
		if !ok {
			interpolationErr = fmt.Errorf("environment variable %s referenced in config but not set", varName)
			return match
		}
		return value
	})
	if interpolationErr != nil {
		return nil, interpolationErr
	}

	var cfg Config
	if err := yaml.Unmarshal([]byte(interpolated), &cfg); err != nil {
		return nil, fmt.Errorf("parsing YAML config: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return &cfg, nil
}

// Validate checks that all required config fields are present and valid.
func (c *Config) Validate() error {
	if c.Account.Universe == "" {
		return fmt.Errorf("account.universe is required")
	}
	if c.Account.Username == "" {
		return fmt.Errorf("account.username is required")
	}
	if c.Account.Password == "" {
		return fmt.Errorf("account.password is required")
	}
	if c.Ogamed.URL == "" {
		return fmt.Errorf("ogamed.url is required")
	}
	if c.RateLimit.DefaultMinDelayMs < 500 {
		return fmt.Errorf("rateLimit.defaultMinDelayMs must be >= 500ms, got %d", c.RateLimit.DefaultMinDelayMs)
	}
	if c.RateLimit.DefaultMaxDelayMs < c.RateLimit.DefaultMinDelayMs {
		return fmt.Errorf("rateLimit.defaultMaxDelayMs must be >= defaultMinDelayMs")
	}
	return nil
}
