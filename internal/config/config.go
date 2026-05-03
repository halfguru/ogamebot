// Package config handles YAML configuration loading with environment
// variable interpolation and validation.
package config

import (
	"fmt"
	"log/slog"
	"os"
	"regexp"

	"github.com/user/ogame-bot/internal/model"

	"gopkg.in/yaml.v3"
)

// Config is the top-level bot configuration.
type Config struct {
	Account   AccountConfig   `yaml:"account"`
	Ogamed    OgamedConfig    `yaml:"ogamed"`
	OGameX    OGameXConfig    `yaml:"ogamex"`
	Features  FeaturesConfig  `yaml:"features"`
	RateLimit RateLimitConfig `yaml:"rateLimit"`
	Dashboard DashboardConfig `yaml:"dashboard"`
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

// OGameXConfig holds the OGameX server connection and auth settings.
type OGameXConfig struct {
	URL      string `yaml:"url"`
	Email    string `yaml:"email"`
	Password string `yaml:"password"`
}

// FeatureConfig holds the enabled state and poll interval for a bot feature.
type FeatureConfig struct {
	Enabled        bool `yaml:"enabled"`
	PollIntervalMs int  `yaml:"pollIntervalMs"`
}

// DefenderConfig holds the defender feature settings including safety margins.
type DefenderConfig struct {
	FeatureConfig     `yaml:",inline"`
	SafetyMarginMs    int  `yaml:"safetyMarginMs"`
	RecallEnabled     *bool `yaml:"recallEnabled"`
	MaxReturnFlightS  int  `yaml:"maxReturnFlightS"`
	MinReactionDelayS int  `yaml:"minReactionDelayS"`
	MaxReactionDelayS int  `yaml:"maxReactionDelayS"`
}

// DefenderDefaults applies default values for zero-valued DefenderConfig fields.
func (d *DefenderConfig) DefenderDefaults() {
	if d.SafetyMarginMs == 0 {
		d.SafetyMarginMs = 120000
	}
	if d.RecallEnabled == nil {
		v := true
		d.RecallEnabled = &v
	}
	if d.MaxReturnFlightS == 0 {
		d.MaxReturnFlightS = 600
	}
	if d.MinReactionDelayS == 0 {
		d.MinReactionDelayS = 30
	}
	if d.MaxReactionDelayS == 0 {
		d.MaxReactionDelayS = 120
	}
}

// IsRecallEnabled returns the RecallEnabled value, defaulting to true.
func (d *DefenderConfig) IsRecallEnabled() bool {
	if d.RecallEnabled == nil {
		return true
	}
	return *d.RecallEnabled
}

// FeaturesConfig holds per-feature toggle settings.
type FeaturesConfig struct {
	Defender  DefenderConfig  `yaml:"defender"`
	AutoBuild AutoBuildConfig `yaml:"autoBuild"`
	AutoFarm  AutoFarmConfig  `yaml:"autoFarm"`
}

// AutoBuildConfig holds the auto-build feature settings including per-building caps.
type AutoBuildConfig struct {
	FeatureConfig   `yaml:",inline"`
	MaxLevels       map[string]int            `yaml:"maxLevels"`       // global defaults: {"MetalMine": 30, ...}
	PlanetOverrides map[string]map[string]int `yaml:"planetOverrides"` // per-planet: {"Homeworld": {"MetalMine": 35}}
}

// AutoBuildDefaults applies default max-level caps when not set.
func (a *AutoBuildConfig) AutoBuildDefaults() {
	if a.MaxLevels == nil {
		a.MaxLevels = map[string]int{
			"MetalMine": 30, "CrystalMine": 28, "DeuteriumSynthesizer": 26,
			"SolarPlant": 26, "FusionReactor": 20,
		}
	}
}

// GalaxyRange is aliased here to avoid circular import with model package.
type GalaxyRange = model.GalaxyRange

// AutoFarmConfig holds the auto-farm feature settings.
type AutoFarmConfig struct {
	FeatureConfig      `yaml:",inline"`
	GalaxyRanges       []GalaxyRange `yaml:"galaxyRanges"`
	MinProfitThreshold int64         `yaml:"minProfitThreshold"` // minimum metal-equivalent profit to attack
	MaxProbesPerTarget int           `yaml:"maxProbesPerTarget"` // probes sent per espionage
	MaxAttacksPerCycle int           `yaml:"maxAttacksPerCycle"` // max attacks per poll cycle
	SkipDefended       bool          `yaml:"skipDefended"`       // skip targets with defense
}

// AutoFarmDefaults applies default values for zero-valued AutoFarmConfig fields.
func (a *AutoFarmConfig) AutoFarmDefaults() {
	if a.MaxProbesPerTarget == 0 {
		a.MaxProbesPerTarget = 5
	}
	if a.MaxAttacksPerCycle == 0 {
		a.MaxAttacksPerCycle = 3
	}
	if a.MinProfitThreshold == 0 {
		a.MinProfitThreshold = 10000
	}
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

// DashboardConfig holds the web dashboard settings.
type DashboardConfig struct {
	Enabled     bool     `yaml:"enabled"`
	Port        int      `yaml:"port"`
	CorsOrigins []string `yaml:"corsOrigins"`
}

var envVarPattern = regexp.MustCompile(`\$\{(\w+)\}`)

func Load(path string, log *slog.Logger) (*Config, error) {
	cfg := Config{
		Ogamed: OgamedConfig{
			URL: envOrDefault("OGAMED_URL", "http://ogamed:8080"),
		},
		OGameX: OGameXConfig{
			URL:      os.Getenv("OGAMEX_URL"),
			Email:    os.Getenv("OGAMEX_EMAIL"),
			Password: os.Getenv("OGAMEX_PASSWORD"),
		},
		Account: AccountConfig{
			Universe: os.Getenv("OGAMED_UNIVERSE"),
			Username: os.Getenv("OGAMED_USERNAME"),
			Password: os.Getenv("OGAMED_PASSWORD"),
		},
		RateLimit: RateLimitConfig{
			DefaultMinDelayMs: 2000,
			DefaultMaxDelayMs: 5000,
		},
		Dashboard: DashboardConfig{
			Enabled:     true,
			Port:        3000,
			CorsOrigins: []string{"*"},
		},
		LogLevel: "info",
	}

	if _, err := os.Stat(path); err == nil {
		raw, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("reading config file: %w", err)
		}

		interpolated := envVarPattern.ReplaceAllStringFunc(string(raw), func(match string) string {
			varName := envVarPattern.FindStringSubmatch(match)[1]
			value, ok := os.LookupEnv(varName)
			if !ok {
				return match
			}
			return value
		})

		if err := yaml.Unmarshal([]byte(interpolated), &cfg); err != nil {
			return nil, fmt.Errorf("parsing YAML config: %w", err)
		}
	} else {
		log.Info("No config.yaml found, using env vars and defaults")
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return &cfg, nil
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
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
	if c.OGameX.URL != "" {
		if c.OGameX.Email == "" {
			return fmt.Errorf("ogamex.email is required when ogamex is configured")
		}
		if c.OGameX.Password == "" {
			return fmt.Errorf("ogamex.password is required when ogamex is configured")
		}
	}
	if c.RateLimit.DefaultMinDelayMs < 500 {
		return fmt.Errorf("rateLimit.defaultMinDelayMs must be >= 500ms, got %d", c.RateLimit.DefaultMinDelayMs)
	}
	if c.RateLimit.DefaultMaxDelayMs < c.RateLimit.DefaultMinDelayMs {
		return fmt.Errorf("rateLimit.defaultMaxDelayMs must be >= defaultMinDelayMs")
	}

	// Dashboard defaults
	if c.Dashboard.Port == 0 {
		c.Dashboard.Port = 3000
	}

	// Apply defender defaults before validation
	c.Features.Defender.DefenderDefaults()

	// Apply auto-build defaults before validation
	c.Features.AutoBuild.AutoBuildDefaults()

	// Apply auto-farm defaults before validation
	c.Features.AutoFarm.AutoFarmDefaults()

	if c.Features.Defender.Enabled {
		if c.Features.Defender.SafetyMarginMs < 10000 {
			return fmt.Errorf("features.defender.safetyMarginMs must be >= 10000ms, got %d", c.Features.Defender.SafetyMarginMs)
		}
		if c.Features.Defender.MinReactionDelayS < 5 {
			return fmt.Errorf("features.defender.minReactionDelayS must be >= 5s, got %d", c.Features.Defender.MinReactionDelayS)
		}
		if c.Features.Defender.MaxReactionDelayS < c.Features.Defender.MinReactionDelayS {
			return fmt.Errorf("features.defender.maxReactionDelayS must be >= minReactionDelayS")
		}
	}

	// AutoBuild validation
	if c.Features.AutoBuild.Enabled {
		if c.Features.AutoBuild.PollIntervalMs > 0 && c.Features.AutoBuild.PollIntervalMs < 10000 {
			return fmt.Errorf("features.autoBuild.pollIntervalMs must be >= 10000ms, got %d", c.Features.AutoBuild.PollIntervalMs)
		}
	}
	// MaxLevel validation: each max level must be in range [1, 100]
	for name, maxLvl := range c.Features.AutoBuild.MaxLevels {
		if maxLvl < 1 || maxLvl > 100 {
			return fmt.Errorf("features.autoBuild.maxLevels[%s] must be in range [1, 100], got %d", name, maxLvl)
		}
		if maxLvl > 50 {
			// Would log warning but Validate doesn't have access to logger;
			// callers can check after load. Validation still passes.
		}
	}

	// AutoFarm validation
	if c.Features.AutoFarm.Enabled {
		if c.Features.AutoFarm.PollIntervalMs > 0 && c.Features.AutoFarm.PollIntervalMs < 60000 {
			return fmt.Errorf("features.autoFarm.pollIntervalMs must be >= 60000ms, got %d", c.Features.AutoFarm.PollIntervalMs)
		}
		if len(c.Features.AutoFarm.GalaxyRanges) == 0 {
			return fmt.Errorf("features.autoFarm.galaxyRanges must have at least one range when enabled")
		}
	}

	return nil
}
