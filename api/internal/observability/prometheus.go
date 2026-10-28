package observability

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"atonixcorp/api/internal/models"
)

var ErrTelemetryUnavailable = errors.New("telemetry provider is unavailable")

type prometheusAdapter struct {
	baseURL string
	token   string
	client  *http.Client
}

func newPrometheusAdapterFromEnvironment() (*prometheusAdapter, error) {
	rawURL := strings.TrimSpace(os.Getenv("OBSERVABILITY_PROMETHEUS_URL"))
	if rawURL == "" {
		return nil, nil
	}
	parsed, err := url.Parse(rawURL)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") ||
		parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return nil, errors.New("OBSERVABILITY_PROMETHEUS_URL must be an HTTP(S) base URL without credentials or query values")
	}
	return &prometheusAdapter{
		baseURL: strings.TrimRight(parsed.String(), "/"),
		token:   os.Getenv("OBSERVABILITY_PROMETHEUS_TOKEN"),
		client:  &http.Client{Timeout: 5 * time.Second},
	}, nil
}

func (p *prometheusAdapter) Name() string { return "prometheus" }

func (p *prometheusAdapter) Availability(ctx context.Context) (string, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, p.baseURL+"/-/ready", nil)
	if err != nil {
		return "unavailable", err
	}
	p.addAuthorization(request)
	response, err := p.client.Do(request)
	if err != nil {
		return "unavailable", nil
	}
	defer response.Body.Close()
	if response.StatusCode >= 200 && response.StatusCode < 300 {
		return "connected", nil
	}
	return "unavailable", nil
}

func (p *prometheusAdapter) Metrics(ctx context.Context, resource models.ObservabilityResource) ([]Metric, error) {
	selector := fmt.Sprintf(
		`{__name__=~"^(atonix_resource_cpu_utilization_ratio|atonix_resource_memory_utilization_ratio|atonix_resource_storage_utilization_ratio|atonix_resource_network_receive_bytes_per_second|atonix_resource_network_transmit_bytes_per_second)$",tenant_id=%q,project_id=%q,resource_type=%q,resource_id=%q}`,
		strconv.FormatUint(uint64(resource.TenantID), 10),
		strconv.FormatUint(uint64(resource.ProjectID), 10),
		resource.ResourceType,
		resource.ResourceID,
	)
	metrics, err := p.query(ctx, selector)
	if err != nil {
		return nil, err
	}
	for i := range metrics {
		if metrics[i].ResourceID != resource.ResourceID || metrics[i].ResourceType != resource.ResourceType ||
			metrics[i].TenantID != resource.TenantID || metrics[i].ProjectID != resource.ProjectID {
			return nil, fmt.Errorf("%w: provider returned a sample outside the requested resource scope", ErrTelemetryUnavailable)
		}
	}
	return metrics, nil
}

func (p *prometheusAdapter) ProjectMetrics(ctx context.Context, scope Scope) ([]Metric, error) {
	selector := fmt.Sprintf(
		`{__name__=~"^(atonix_resource_cpu_utilization_ratio|atonix_resource_memory_utilization_ratio|atonix_resource_storage_utilization_ratio|atonix_resource_network_receive_bytes_per_second|atonix_resource_network_transmit_bytes_per_second)$",tenant_id=%q,project_id=%q}`,
		strconv.FormatUint(uint64(scope.TenantID), 10),
		strconv.FormatUint(uint64(scope.ProjectID), 10),
	)
	metrics, err := p.query(ctx, selector)
	if err != nil {
		return nil, err
	}
	result := metrics[:0]
	for _, metric := range metrics {
		if metric.TenantID != scope.TenantID || metric.ProjectID != scope.ProjectID ||
			metric.ResourceID == "" || metric.ResourceType == "" {
			return nil, fmt.Errorf("%w: provider returned a sample outside the requested project scope", ErrTelemetryUnavailable)
		}
		result = append(result, metric)
	}
	return result, nil
}

func (p *prometheusAdapter) query(ctx context.Context, selector string) ([]Metric, error) {
	queryURL := p.baseURL + "/api/v1/query?query=" + url.QueryEscape(selector)
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, queryURL, nil)
	if err != nil {
		return nil, err
	}
	p.addAuthorization(request)
	response, err := p.client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("%w: query failed", ErrTelemetryUnavailable)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, fmt.Errorf("%w: provider returned HTTP %d", ErrTelemetryUnavailable, response.StatusCode)
	}
	var payload struct {
		Status string `json:"status"`
		Data   struct {
			ResultType string `json:"resultType"`
			Result     []struct {
				Metric map[string]string `json:"metric"`
				Value  []json.RawMessage `json:"value"`
			} `json:"result"`
		} `json:"data"`
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, 1024*1024)).Decode(&payload); err != nil {
		return nil, fmt.Errorf("%w: invalid provider response", ErrTelemetryUnavailable)
	}
	if payload.Status != "success" || payload.Data.ResultType != "vector" {
		return nil, fmt.Errorf("%w: provider returned an unsupported query result", ErrTelemetryUnavailable)
	}
	metrics := make([]Metric, 0, len(payload.Data.Result))
	for _, sample := range payload.Data.Result {
		name := sample.Metric["__name__"]
		unit, supported := prometheusMetricUnits[name]
		if !supported || len(sample.Value) != 2 {
			continue
		}
		var unixSeconds float64
		var rawValue string
		if err := json.Unmarshal(sample.Value[0], &unixSeconds); err != nil {
			return nil, fmt.Errorf("%w: invalid sample timestamp", ErrTelemetryUnavailable)
		}
		if err := json.Unmarshal(sample.Value[1], &rawValue); err != nil {
			return nil, fmt.Errorf("%w: invalid sample value", ErrTelemetryUnavailable)
		}
		value, err := strconv.ParseFloat(rawValue, 64)
		if err != nil || math.IsNaN(value) || math.IsInf(value, 0) {
			return nil, fmt.Errorf("%w: invalid sample value", ErrTelemetryUnavailable)
		}
		if unit == "ratio" {
			if value < 0 || value > 1 {
				return nil, fmt.Errorf("%w: utilization ratio is outside the expected 0-1 range", ErrTelemetryUnavailable)
			}
			value *= 100
			unit = "%"
		}
		metrics = append(metrics, Metric{
			Name: name, Value: value, Unit: unit,
			ResourceID: sample.Metric["resource_id"], ResourceType: sample.Metric["resource_type"],
			TenantID: parseUintLabel(sample.Metric["tenant_id"]), ProjectID: parseUintLabel(sample.Metric["project_id"]),
			ObservedAt: time.UnixMilli(int64(unixSeconds * 1000)).UTC(),
		})
	}
	if len(metrics) == 0 {
		return nil, nil
	}
	return metrics, nil
}

func parseUintLabel(value string) uint {
	parsed, err := strconv.ParseUint(value, 10, 64)
	if err != nil {
		return 0
	}
	return uint(parsed)
}

func (p *prometheusAdapter) addAuthorization(request *http.Request) {
	if p.token != "" {
		request.Header.Set("Authorization", "Bearer "+p.token)
	}
}

var prometheusMetricUnits = map[string]string{
	"atonix_resource_cpu_utilization_ratio":             "ratio",
	"atonix_resource_memory_utilization_ratio":          "ratio",
	"atonix_resource_storage_utilization_ratio":         "ratio",
	"atonix_resource_network_receive_bytes_per_second":  "bytes/second",
	"atonix_resource_network_transmit_bytes_per_second": "bytes/second",
}
