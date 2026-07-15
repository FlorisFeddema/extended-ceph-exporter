package exporter

import (
	"bytes"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

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
		ErrorLog:          prometheusErrorLogger{logger: logger},
	})
	mux.Handle(cfg.MetricsPath, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logger.Info("metrics request", "method", r.Method, "path", r.URL.Path)
		bufferedResponse := newBufferedResponseWriter()
		metricsHandler.ServeHTTP(bufferedResponse, r)
		if bufferedResponse.status >= http.StatusInternalServerError {
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		bufferedResponse.writeTo(w)
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

type prometheusErrorLogger struct {
	logger *slog.Logger
}

func (l prometheusErrorLogger) Println(values ...interface{}) {
	l.logger.Error("metrics collection failed", "error", strings.TrimSpace(fmt.Sprintln(values...)))
}

type bufferedResponseWriter struct {
	header      http.Header
	body        bytes.Buffer
	status      int
	wroteHeader bool
}

func newBufferedResponseWriter() *bufferedResponseWriter {
	return &bufferedResponseWriter{header: make(http.Header)}
}

func (w *bufferedResponseWriter) Header() http.Header {
	return w.header
}

func (w *bufferedResponseWriter) WriteHeader(status int) {
	if w.wroteHeader {
		return
	}
	w.status = status
	w.wroteHeader = true
}

func (w *bufferedResponseWriter) Write(data []byte) (int, error) {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}
	return w.body.Write(data)
}

func (w *bufferedResponseWriter) writeTo(destination http.ResponseWriter) {
	for key, values := range w.header {
		for _, value := range values {
			destination.Header().Add(key, value)
		}
	}
	if w.status == 0 {
		w.status = http.StatusOK
	}
	destination.WriteHeader(w.status)
	_, _ = destination.Write(w.body.Bytes())
}
