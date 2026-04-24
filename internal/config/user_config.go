package config

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/viper"
	"github.com/txdywy/inice/internal/model"
)

// Load reads configuration from file, env vars, and returns merged UserConfig.
// Priority: env var > config file > defaults.
func Load(cfgFile string) (*model.UserConfig, error) {
	cfg := model.DefaultUserConfig()

	v := viper.New()
	v.SetConfigType("yaml")

	// Determine config path
	if cfgFile != "" {
		v.SetConfigFile(cfgFile)
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("cannot find home directory: %w", err)
		}
		v.AddConfigPath(home)
		v.SetConfigName(".inice")
	}

	v.SetDefault("router.port", cfg.Router.Port)
	v.SetDefault("router.user", cfg.Router.User)
	v.SetDefault("shadow.preferred_core", cfg.Shadow.PreferredCore)
	v.SetDefault("shadow.base_port", cfg.Shadow.BasePort)
	v.SetDefault("testing.concurrency", cfg.Testing.Concurrency)
	v.SetDefault("testing.timeout", cfg.Testing.Timeout)
	v.SetDefault("testing.warmup_probes", cfg.Testing.WarmupProbes)
	v.SetDefault("testing.measurement_probes", cfg.Testing.MeasurementProbes)
	v.SetDefault("output.mode", cfg.Output.Mode)
	v.SetDefault("output.format", cfg.Output.Format)

	// Bind env vars
	v.BindEnv("router.password", "INICE_SSH_PASSWORD")
	v.BindEnv("router.host", "INICE_ROUTER_HOST")
	v.BindEnv("router.port", "INICE_ROUTER_PORT")
	v.BindEnv("router.user", "INICE_ROUTER_USER")
	v.BindEnv("router.key_file", "INICE_SSH_KEY_FILE")

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("config file error: %w", err)
		}
		// Config file not found is OK — use defaults + env vars
	}

	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("cannot unmarshal config: %w", err)
	}

	return &cfg, nil
}

// ParseTestConfig converts the user config into a TestConfig with parsed duration.
func ParseTestConfig(cfg *model.UserConfig) (model.TestConfig, error) {
	tc := model.DefaultTestConfig()
	if cfg.Testing.Concurrency > 0 {
		tc.Concurrency = cfg.Testing.Concurrency
	}
	if cfg.Testing.WarmupProbes > 0 {
		tc.WarmupProbes = cfg.Testing.WarmupProbes
	}
	if cfg.Testing.MeasurementProbes > 0 {
		tc.MeasurementProbes = cfg.Testing.MeasurementProbes
	}
	if cfg.Testing.Timeout != "" {
		d, err := time.ParseDuration(cfg.Testing.Timeout)
		if err != nil {
			return tc, fmt.Errorf("invalid testing.timeout: %w", err)
		}
		tc.Timeout = d
	}
	return tc, nil
}
