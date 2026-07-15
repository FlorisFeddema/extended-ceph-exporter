package exporter

import (
	"log/slog"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/FlorisFeddema/extended-ceph-exporter/internal/config"
)

func NewHandler(cfg config.Config, registry *prometheus.Registry, loggers ...*slog.Logger) http.Handler {
	logger := slog.Default()
	if len(loggers) > 0 && loggers[0] != nil {
		logger = loggers[0]
	}

	mux := http.NewServeMux()
	metricsHandler := promhttp.HandlerFor(registry, promhttp.HandlerOpts{
		EnableOpenMetrics: true,
	})
	mux.Handle(cfg.MetricsPath, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logger.Info("metrics request", "method", r.Method, "path", r.URL.Path)
		metricsHandler.ServeHTTP(w, r)
	}))
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok\n"))
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("extended-ceph-exporter\n"))
	})

	return mux
}
