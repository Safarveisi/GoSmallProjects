package main

import (
	"context"
	"fmt"
	"monitor/collector"
	"monitor/config"
	"monitor/logger"
	"monitor/storage"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Println("Error loading config:", err)
		return
	}
	fmt.Println("Loaded config:", cfg)
	fmt.Println("Log level:", cfg.LogLevel)

	log, err := logger.New(cfg.LogLevel)

	if err != nil {
		fmt.Println("Error setting up logger:", err)
		return
	}
	log.Logger.Info("Logger initialized")

	query := `engine_daemon_network_actions_seconds_count[1m]`
	pColl := collector.NewPrometheusCollector(cfg.PrometheusURL, query, log.Logger)
	snapshot, err := collector.CollectAll(context.Background(),
		[]collector.Collector{pColl},
		log.Logger,
	)
	if err != nil {
		fmt.Println("Error collecting metrics:", err)
		return
	}
	fmt.Printf("Collected metric %s at %s with value %f\n", query, snapshot.Metrics[query].Timestamp, snapshot.Metrics[query].Value)

	store, err := storage.NewSQLite(cfg.DBPath, log.Logger)
	if err != nil {
		fmt.Println("Error creating SQLite DB:", err)
		return
	}
	defer store.Close()

	err = store.Save(context.Background(), snapshot)
	if err != nil {
		fmt.Println("Error saving metrics to DB:", err)
		return
	}
	fmt.Println("Metrics saved to DB successfully")

}
