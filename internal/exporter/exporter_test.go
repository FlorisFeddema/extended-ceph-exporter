package exporter

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"

	"github.com/FlorisFeddema/extended-ceph-exporter/internal/config"
)

func TestNewHandlerRootAndHealth(t *testing.T) {
	handler := NewHandler(config.Config{MetricsPath: "/metrics"}, prometheus.NewRegistry())

	rootReq := httptest.NewRequest(http.MethodGet, "/", nil)
	rootRes := httptest.NewRecorder()
	handler.ServeHTTP(rootRes, rootReq)

	if rootRes.Code != http.StatusOK || rootRes.Body.String() != "extended-ceph-exporter\n" {
		t.Fatalf("unexpected root response: code=%d body=%q", rootRes.Code, rootRes.Body.String())
	}

	healthReq := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	healthRes := httptest.NewRecorder()
	handler.ServeHTTP(healthRes, healthReq)

	if healthRes.Code != http.StatusOK || healthRes.Body.String() != "ok\n" {
		t.Fatalf("unexpected health response: code=%d body=%q", healthRes.Code, healthRes.Body.String())
	}
}

func TestNewHandlerMetrics(t *testing.T) {
	registry := prometheus.NewRegistry()
	gauge := prometheus.NewGauge(prometheus.GaugeOpts{Name: "test_metric", Help: "test help"})
	gauge.Set(1)
	registry.MustRegister(gauge)

	handler := NewHandler(config.Config{MetricsPath: "/metrics"}, registry)
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("unexpected metrics status: %d", res.Code)
	}
	body := res.Body.String()
	if body == "" || !strings.Contains(body, "test_metric") {
		t.Fatalf("expected metrics output to contain test metric, got %q", body)
	}
}

type failingCollector struct{}

func (failingCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- prometheus.NewDesc("test_metric", "test help", nil, nil)
}

func (failingCollector) Collect(ch chan<- prometheus.Metric) {
	ch <- failingMetric{}
}

type failingMetric struct{}

func (failingMetric) Desc() *prometheus.Desc {
	return prometheus.NewDesc("test_metric", "test help", nil, nil)
}

func (failingMetric) Write(*dto.Metric) error {
	return errors.New("secret RGW permission error")
}

func TestNewHandlerHidesMetricsErrors(t *testing.T) {
	registry := prometheus.NewRegistry()
	registry.MustRegister(failingCollector{})
	handler := NewHandler(config.Config{MetricsPath: "/metrics"}, registry)

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)

	if res.Code != http.StatusInternalServerError {
		t.Fatalf("unexpected metrics status: %d", res.Code)
	}
	if res.Body.String() != "internal server error\n" {
		t.Fatalf("unexpected metrics error body: %q", res.Body.String())
	}
	if strings.Contains(res.Body.String(), "secret RGW permission error") {
		t.Fatal("metrics response exposed internal error")
	}
}
