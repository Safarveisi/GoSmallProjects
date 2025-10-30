package collector

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"go.uber.org/zap"
)

// Collector is the public contract any metric source must satisfy.
type Collector interface {
	// Collect fetches metrics from its source and returns a map of
	// metric name -> value. The timestamp is the moment the data was
	// retrieved (i.e., time.Now()).
	Collect(ctx context.Context) (map[string]float64, error)
}

// CollectAll runs every registered collector, merges the results and
// returns a single snapshot.  The order of collectors does not matter.
func CollectAll(ctx context.Context, colls []Collector, log *zap.Logger) (*MetricsSnapshot, error) {
	snap := NewSnapshot(time.Now())

	for _, c := range colls {
		m, err := c.Collect(ctx)
		if err != nil {
			// We log the error but continue with other collectors - a single
			// failing source should not stop the whole pipeline.
			log.Error("collector failed", zap.Error(err))
			continue
		}
		for name, val := range m {
			snap.Metrics[name] = Metric{
				Name:      name,
				Value:     val,
				Timestamp: snap.CollectedAt,
			}
		}
	}
	return snap, nil
}

// PrometheusCollector - pulls a single query from Prometheus.

// PrometheusCollector implements Collector.
// It issues a standard Prometheus HTTP API query (`/api/v1/query`) and
// extracts the first sample from the result set.  For a production‑grade
// version you would handle multiple series, matrix queries, etc., but
// for this small project we keep it simple.
type PrometheusCollector struct {
	BaseURL   string       // e.g. "http://localhost:9090"
	Query     string       // PromQL expression, e.g. "rate(http_requests_total[1m])"
	HTTP      *http.Client // injected for testability (may be nil -> default client)
	Log       *zap.Logger  // logger for debugging
	UserAgent string       // optional
}

// PrometheusAPIResponse – minimal subset of the JSON returned by /api/v1/query.
type prometheusAPIResponse struct {
	Status string `json:"status"`
	Data   struct {
		ResultType string             `json:"resultType"` // we expect "matrix"
		Result     []prometheusSeries `json:"result"`
	} `json:"data"`
}

// prometheusSeries represents a single time‑series returned by the query.
type prometheusSeries struct {
	Metric map[string]string `json:"metric"` // we only need the metric labels
	Values [][]interface{}   `json:"values"` // each entry is [ <timestamp>, "<value>" ]
}

// NewPrometheusCollector returns a ready‑to‑use collector.
func NewPrometheusCollector(baseURL, query string, log *zap.Logger) *PrometheusCollector {
	return &PrometheusCollector{
		BaseURL:   baseURL,
		Query:     query,
		HTTP:      &http.Client{Timeout: 10 * time.Second},
		Log:       log,
		UserAgent: "model-health-watcher/0.1",
	}
}

// Collect implements the Collector interface.
func (p *PrometheusCollector) Collect(ctx context.Context) (map[string]float64, error) {
	u, err := url.Parse(p.BaseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid prometheus base url: %s", err)
	}
	u.Path = "/api/v1/query"
	q := u.Query()
	q.Set("query", p.Query)
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	if p.UserAgent != "" {
		req.Header.Set("User-Agent", p.UserAgent)
	}
	resp, err := p.HTTP.Do(req)
	if err != nil {
		return nil, fmt.Errorf("prometheus request error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("prometheus returned %d: %s", resp.StatusCode, string(b))
	}

	var apiResp prometheusAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("failed to decode prometheus response: %w", err)
	}
	if apiResp.Status != "success" {
		return nil, fmt.Errorf("prometheus query not successful: %s", apiResp.Status)
	}
	if len(apiResp.Data.Result) == 0 {
		return nil, fmt.Errorf("prometheus query returned no results")
	}

	// Take the first series / sample.
	sample := apiResp.Data.Result[0].Values[0]
	// sample[0] is timestamp (float64 seconds since epoch), sample[1] is string value.
	valStr, ok := sample[1].(string)
	if !ok {
		return nil, fmt.Errorf("unexpected value type in prometheus response")
	}
	val, err := parseFloat(valStr)
	if err != nil {
		return nil, fmt.Errorf("cannot parse prometheus value %q: %w", valStr, err)
	}

	// Use the metric name from the query as the key.
	metricName := p.Query
	return map[string]float64{metricName: val}, nil
}

// ModelAPICollector - fetches model-specific metrics.

// ModelAPICollector calls a REST endpoint exposed by the model serving
// system (e.g., TensorFlow-Serving / TorchServe) that returns JSON.
// The JSON format is deliberately kept generic - a flat map of name -> value.
// Example response:
//
//	{
//	  "latency_ms": 12.3,
//	  "error_rate": 0.02,
//	  "confidence_histogram": {"0.0-0.5": 120, "0.5-1.0": 80}
//	}
//
// The collector extracts only the top-level numeric fields and discards
// any nested objects (they can be handled later if needed).
type ModelAPICollector struct {
	BaseURL   string       // e.g. "http://model-serving:8501/v1/models/myModel/metrics"
	HTTP      *http.Client // injected for testability
	Log       *zap.Logger
	UserAgent string
}

// NewModelAPICollector creates a collector instance.
func NewModelAPICollector(baseURL string, log *zap.Logger) *ModelAPICollector {
	return &ModelAPICollector{
		BaseURL:   baseURL,
		HTTP:      &http.Client{Timeout: 10 * time.Second},
		Log:       log,
		UserAgent: "model-health-watcher/0.1",
	}
}

// Collect fetches the JSON payload and extracts numeric fields.
func (m *ModelAPICollector) Collect(ctx context.Context) (map[string]float64, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, m.BaseURL, nil)
	if err != nil {
		return nil, err
	}
	if m.UserAgent != "" {
		req.Header.Set("User-Agent", m.UserAgent)
	}
	resp, err := m.HTTP.Do(req)
	if err != nil {
		return nil, fmt.Errorf("model API request error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("model API returned %d: %s", resp.StatusCode, string(b))
	}

	// Decode into a generic map.
	var raw map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("failed to decode model API JSON: %w", err)
	}

	metrics := make(map[string]float64)
	for k, v := range raw {
		// We only keep top-level scalar numbers.
		switch num := v.(type) {
		case float64:
			metrics[k] = num
		case json.Number:
			f, _ := num.Float64()
			metrics[k] = f
		case string:
			// Try to parse a numeric string (e.g., "0.03").
			if f, err := parseFloat(num); err == nil {
				metrics[k] = f
			}
		default:
			// ignore non-numeric / nested structures.
			m.Log.Debug("skipping non-numeric model metric", zap.String("key", k))
		}
	}
	if len(metrics) == 0 {
		return nil, fmt.Errorf("no numeric metrics found in model API response")
	}
	return metrics, nil
}

// Helper functions (kept private to this file)

func parseFloat(s string) (float64, error) {
	// strconv.ParseFloat handles both integer and floating point strings.
	// We import strconv only here to avoid polluting the public API.
	return strconv.ParseFloat(s, 64)
}
