package config

import (
	"flag"
	"log/slog"
	"os"
	"testing"
	"time"
)

func TestParseArgsDefaults(t *testing.T) {
	fs := flag.NewFlagSet("test", flag.ContinueOnError)

	cfg := parseArgs(fs, nil)

	if cfg.ListenAddress != ":9877" {
		t.Fatalf("unexpected listen address: %q", cfg.ListenAddress)
	}
	if cfg.MetricsPath != "/metrics" {
		t.Fatalf("unexpected metrics path: %q", cfg.MetricsPath)
	}
	if cfg.RequestTimeout != 10*time.Second {
		t.Fatalf("unexpected request timeout: %s", cfg.RequestTimeout)
	}
	if cfg.RGWCacheTTL != 5*time.Second {
		t.Fatalf("unexpected RGW cache TTL: %s", cfg.RGWCacheTTL)
	}
	if cfg.SelfMetricsEnabled {
		t.Fatal("expected self metrics to default to disabled")
	}
	if cfg.LogLevel != slog.LevelInfo {
		t.Fatalf("unexpected log level: %v", cfg.LogLevel)
	}
}

func TestParseArgsCustomValues(t *testing.T) {
	fs := flag.NewFlagSet("test", flag.ContinueOnError)

	cfg := parseArgs(fs, []string{
		"--listen-address=:9999",
		"--metrics-path=/custom-metrics",
		"--request-timeout=30s",
		"--rgw-cache-ttl=15s",
		"--rgw-admin-endpoint=https://rgw.example",
		"--rgw-access-key=access",
		"--rgw-secret-key=secret",
		"--self-metrics-enabled",
		"--log-level=debug",
	})

	if cfg.ListenAddress != ":9999" || cfg.MetricsPath != "/custom-metrics" {
		t.Fatalf("unexpected network config: %+v", cfg)
	}
	if cfg.RequestTimeout != 30*time.Second || cfg.RGWCacheTTL != 15*time.Second {
		t.Fatalf("unexpected durations: %+v", cfg)
	}
	if cfg.RGWAdminEndpoint != "https://rgw.example" || cfg.RGWAccessKey != "access" || cfg.RGWSecretKey != "secret" {
		t.Fatalf("unexpected RGW config: %+v", cfg)
	}
	if !cfg.SelfMetricsEnabled {
		t.Fatal("expected self metrics to be enabled")
	}
	if cfg.LogLevel != slog.LevelDebug {
		t.Fatalf("unexpected log level: %v", cfg.LogLevel)
	}
}

func TestParseLogLevel(t *testing.T) {
	tests := map[string]slog.Level{
		"debug":   slog.LevelDebug,
		"warn":    slog.LevelWarn,
		"error":   slog.LevelError,
		"info":    slog.LevelInfo,
		"unknown": slog.LevelInfo,
	}

	for input, expected := range tests {
		if got := parseLogLevel(input); got != expected {
			t.Fatalf("parseLogLevel(%q) = %v, want %v", input, got, expected)
		}
	}
}

func TestParseArgsEnvDefaults(t *testing.T) {
	t.Setenv("EXTENDED_CEPH_EXPORTER_LISTEN_ADDRESS", ":1234")
	t.Setenv("EXTENDED_CEPH_EXPORTER_METRICS_PATH", "/env-metrics")
	t.Setenv("EXTENDED_CEPH_EXPORTER_REQUEST_TIMEOUT", "45s")
	t.Setenv("EXTENDED_CEPH_EXPORTER_RGW_CACHE_TTL", "20s")
	t.Setenv("EXTENDED_CEPH_EXPORTER_RGW_ADMIN_ENDPOINT", "https://env-rgw.example")
	t.Setenv("EXTENDED_CEPH_EXPORTER_RGW_ACCESS_KEY", "env-access")
	t.Setenv("EXTENDED_CEPH_EXPORTER_RGW_SECRET_KEY", "env-secret")
	t.Setenv("EXTENDED_CEPH_EXPORTER_SELF_METRICS_ENABLED", "true")
	t.Setenv("EXTENDED_CEPH_EXPORTER_LOG_LEVEL", "warn")

	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	cfg := parseArgs(fs, nil)

	if cfg.ListenAddress != ":1234" || cfg.MetricsPath != "/env-metrics" {
		t.Fatalf("unexpected env network config: %+v", cfg)
	}
	if cfg.RequestTimeout != 45*time.Second || cfg.RGWCacheTTL != 20*time.Second {
		t.Fatalf("unexpected env durations: %+v", cfg)
	}
	if cfg.RGWAdminEndpoint != "https://env-rgw.example" || cfg.RGWAccessKey != "env-access" || cfg.RGWSecretKey != "env-secret" {
		t.Fatalf("unexpected env RGW config: %+v", cfg)
	}
	if !cfg.SelfMetricsEnabled || cfg.LogLevel != slog.LevelWarn {
		t.Fatalf("unexpected env flags: %+v", cfg)
	}
}

func TestParseArgsFlagsOverrideEnv(t *testing.T) {
	t.Setenv("EXTENDED_CEPH_EXPORTER_LISTEN_ADDRESS", ":1234")
	t.Setenv("EXTENDED_CEPH_EXPORTER_SELF_METRICS_ENABLED", "false")

	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	cfg := parseArgs(fs, []string{
		"--listen-address=:9999",
		"--self-metrics-enabled=true",
	})

	if cfg.ListenAddress != ":9999" {
		t.Fatalf("expected flag to override env, got %q", cfg.ListenAddress)
	}
	if !cfg.SelfMetricsEnabled {
		t.Fatal("expected self metrics to be enabled by flag override")
	}
}

func TestEnvFallbackParsing(t *testing.T) {
	if err := os.Setenv("EXTENDED_CEPH_EXPORTER_REQUEST_TIMEOUT", "not-a-duration"); err != nil {
		t.Fatalf("setenv failed: %v", err)
	}
	if got := envDuration("EXTENDED_CEPH_EXPORTER_REQUEST_TIMEOUT", 10*time.Second); got != 10*time.Second {
		t.Fatalf("expected fallback duration, got %s", got)
	}

	if err := os.Setenv("EXTENDED_CEPH_EXPORTER_SELF_METRICS_ENABLED", "not-a-bool"); err != nil {
		t.Fatalf("setenv failed: %v", err)
	}
	if got := envBool("EXTENDED_CEPH_EXPORTER_SELF_METRICS_ENABLED", false); got {
		t.Fatal("expected fallback bool to remain false")
	}
}
