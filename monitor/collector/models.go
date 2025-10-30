package collector

import "time"

// Metric holds a single numeric value together with its timestamp.
type Metric struct {
	Name      string    // e.g. "model_inference_latency_ms"
	Value     float64   // numeric value
	Timestamp time.Time // time the metric was observed (usually now)
}

// MetricsSnapshot is the result of a single collection cycle.
// All metrics share the same collection timestamp.
type MetricsSnapshot struct {
	CollectedAt time.Time         // when the collection happened
	Metrics     map[string]Metric // key = metric name
}

// NewSnapshot creates an empty snapshot with the supplied time.
func NewSnapshot(ts time.Time) *MetricsSnapshot {
	return &MetricsSnapshot{
		CollectedAt: ts,
		Metrics:     make(map[string]Metric),
	}
}
