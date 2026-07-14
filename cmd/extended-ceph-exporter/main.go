package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/FlorisFeddema/extended-ceph-exporter/internal/collector/rgw"
	"github.com/FlorisFeddema/extended-ceph-exporter/internal/config"
	"github.com/FlorisFeddema/extended-ceph-exporter/internal/exporter"
	"github.com/FlorisFeddema/extended-ceph-exporter/internal/rgwclient"
)

func main() {
	cfg := config.Parse()

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: cfg.LogLevel,
	}))

	registry := prometheus.NewRegistry()
	buildInfo := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "extended_ceph_exporter_build_info",
			Help: "Build and version metadata for the exporter.",
		},
		[]string{"version"},
	)
	buildInfo.WithLabelValues("dev").Set(1)
	registry.MustRegister(buildInfo)

	var rgwServiceMetrics *rgw.ServiceMetrics
	if cfg.SelfMetricsEnabled {
		rgwServiceMetrics = rgw.NewServiceMetrics()
		registry.MustRegister(rgwServiceMetrics.Collectors()...)
	}

	bucketSource := rgw.BucketSource(rgw.StaticBucketSource{})
	userSource := rgw.UserSource(rgw.StaticUserSource{})
	if cfg.RGWAdminEndpoint != "" || cfg.RGWAccessKey != "" || cfg.RGWSecretKey != "" {
		client, err := rgwclient.New(cfg)
		if err != nil {
			logger.Error("failed to initialize RGW admin client", "error", err)
			os.Exit(1)
		}

		bucketSource = rgwclient.NewBucketSource(client)
		userSource = rgwclient.NewUserSource(client)
	}

	rgwService := rgw.NewServiceWithMetrics(
		bucketSource,
		userSource,
		cfg.RGWCacheTTL,
		rgwServiceMetrics,
	)
	registry.MustRegister(
		rgw.NewBucketsCollector(rgwService, cfg.RequestTimeout),
		rgw.NewUsersCollector(rgwService, cfg.RequestTimeout),
	)

	server := &http.Server{
		Addr:              cfg.ListenAddress,
		Handler:           exporter.NewHandler(cfg, registry),
		ReadHeaderTimeout: 5 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)
	go func() {
		logger.Info(
			"starting exporter",
			"listen_address", cfg.ListenAddress,
			"metrics_path", cfg.MetricsPath,
			"rgw_cache_ttl", cfg.RGWCacheTTL.String(),
			"rgw_admin_endpoint_configured", cfg.RGWAdminEndpoint != "",
			"self_metrics_enabled", cfg.SelfMetricsEnabled,
		)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		logger.Error("server failed", "error", err)
		os.Exit(1)
	case <-ctx.Done():
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("graceful shutdown failed", "error", err)
		os.Exit(1)
	}
}
