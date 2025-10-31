package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

// Config holds every configurable value for the watcher.
type Config struct {
	// External services
	PrometheusURL string // e.g. http://prometheus:9090

	// Persistence
	DBPath string // path to the SQLite file, e.g. "./data/metrics.db"

	// Server
	LogLevel string // debug|info|warn|error
}

// Load reads configuration from (in decreasing priority):
//  1. command‑line flags (handled later in main - not part of this pkg)
//  2. environment variables (e.g. PROMETHEUS_URL)
//  3. a yaml file (./configs/config.yaml) if it exists.
//
// It returns a fully populated *Config or an error.
func Load() (*Config, error) {
	v := viper.New()

	// Default values – keep them sensible and minimal
	v.SetDefault("PrometheusURL", "http://localhost:9090")
	v.SetDefault("DBPath", "./data/metrics.db")
	v.SetDefault("LogLevel", "info")

	// Environment variables - Viper automatically maps "_" to "." (case-insensitive)
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Optional yaml file - useful for local dev or k8s ConfigMap
	v.SetConfigName("config")
	v.AddConfigPath("./configs")
	_ = v.ReadInConfig() // ignore error - file is optional

	// Populate the struct
	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("cannot decode config: %w", err)
	}

	// Basic validation (extend later as needed)
	if cfg.PrometheusURL == "" {
		return nil, fmt.Errorf("PrometheusURL must not be empty")
	}

	return &cfg, nil
}
