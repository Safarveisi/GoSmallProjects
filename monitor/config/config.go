package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config holds every configurable value for the watcher.
type Config struct {
	// External services
	PrometheusURL   string // e.g. http://prometheus:9090
	ModelAPIURL     string // e.g. http://model-serving:8501/v1/models/myModel/metrics
	AlertWebhookURL string // optional, POST JSON payload when drift detected
	// Persistence
	DBPath string // path to the SQLite file, e.g. "./data/metrics.db"

	// Drift detection
	// Simple threshold for the error-rate metric (0-1). If the metric stays above
	// this value for ConsecutivePeriods minutes, an alert is generated.
	ErrorRateThreshold float64
	ConsecutivePeriods int           // how many minutes the threshold must be violated
	DriftCheckInterval time.Duration // how often we run the drift checks

	// Server
	Port     int    // HTTP server port, e.g. 8080
	LogLevel string // debug|info|warn|error

	// Misc
	ShutdownTimeout time.Duration // graceful shutdown timeout
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
	v.SetDefault("ModelAPIURL", "http://localhost:8501/v1/models/default/metrics")
	v.SetDefault("AlertWebhookURL", "")
	v.SetDefault("DBPath", "./data/metrics.db")
	v.SetDefault("ErrorRateThreshold", 0.05)
	v.SetDefault("ConsecutivePeriods", 5)
	v.SetDefault("DriftCheckInterval", "1m")
	v.SetDefault("Port", 8080)
	v.SetDefault("LogLevel", "info")
	v.SetDefault("ShutdownTimeout", "10s")

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

	// Convert some string-based durations parsed by Viper to time.Duration
	if d, err := time.ParseDuration(v.GetString("DriftCheckInterval")); err == nil {
		cfg.DriftCheckInterval = d
	} else {
		return nil, fmt.Errorf("invalid DriftCheckInterval: %w", err)
	}
	if d, err := time.ParseDuration(v.GetString("ShutdownTimeout")); err == nil {
		cfg.ShutdownTimeout = d
	} else {
		return nil, fmt.Errorf("invalid ShutdownTimeout: %w", err)
	}

	// Basic validation (extend later as needed)
	if cfg.PrometheusURL == "" {
		return nil, fmt.Errorf("PrometheusURL must not be empty")
	}
	if cfg.ModelAPIURL == "" {
		return nil, fmt.Errorf("ModelAPIURL must not be empty")
	}
	if cfg.Port <= 0 || cfg.Port > 65535 {
		return nil, fmt.Errorf("port must be a valid TCP port")
	}
	if cfg.ErrorRateThreshold < 0 || cfg.ErrorRateThreshold > 1 {
		return nil, fmt.Errorf("ErrorRateThreshold must be between 0 and 1")
	}
	if cfg.ConsecutivePeriods <= 0 {
		return nil, fmt.Errorf("ConsecutivePeriods must be > 0")
	}
	if cfg.DriftCheckInterval <= 0 {
		return nil, fmt.Errorf("DriftCheckInterval must be > 0")
	}

	return &cfg, nil
}
