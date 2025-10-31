package storage

import (
	"context"
	"monitor/collector"
	"time"
)

// MetricRecord is a single persisted metric row.
type MetricRecord struct {
	ID        int64     // auto-increment primary key (mostly for internal use)
	Timestamp time.Time // when the metric was collected
	Name      string    // metric name, e.g. "error_rate"
	Value     float64   // numeric value
}

// Store abstracts a persistence back-end for metric snapshots.
type Store interface {
	// Save stores all metrics from a snapshot in a single transaction.
	// The implementation must guarantee atomicity - either all rows are
	// written or none.
	Save(ctx context.Context, snap *MetricsSnapshot) error

	// Close releases any resources (e.g. DB connections).
	Close() error
}

// MetricsSnapshot is re-exported here so callers do not need to import
// the collector package just to call Store.Save().
type MetricsSnapshot = collector.MetricsSnapshot
