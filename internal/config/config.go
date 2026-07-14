package config

import (
	"flag"
	"log/slog"
	"os"
	"strconv"
	"time"
)

type Config struct {
	ListenAddress string
	MetricsPath   string

	RequestTimeout time.Duration
	RGWCacheTTL    time.Duration

	RGWAdminEndpoint   string
	RGWAccessKey       string
	RGWSecretKey       string
	SelfMetricsEnabled bool

	LogLevel slog.Level
}

func Parse() Config {
	return parseArgs(flag.CommandLine, nil)
}

func parseArgs(fs *flag.FlagSet, args []string) Config {
	cfg := Config{
		ListenAddress:      envString("EXTENDED_CEPH_EXPORTER_LISTEN_ADDRESS", ":9877"),
		MetricsPath:        envString("EXTENDED_CEPH_EXPORTER_METRICS_PATH", "/metrics"),
		RequestTimeout:     envDuration("EXTENDED_CEPH_EXPORTER_REQUEST_TIMEOUT", 10*time.Second),
		RGWCacheTTL:        envDuration("EXTENDED_CEPH_EXPORTER_RGW_CACHE_TTL", 5*time.Second),
		RGWAdminEndpoint:   envString("EXTENDED_CEPH_EXPORTER_RGW_ADMIN_ENDPOINT", ""),
		RGWAccessKey:       envString("EXTENDED_CEPH_EXPORTER_RGW_ACCESS_KEY", ""),
		RGWSecretKey:       envString("EXTENDED_CEPH_EXPORTER_RGW_SECRET_KEY", ""),
		SelfMetricsEnabled: envBool("EXTENDED_CEPH_EXPORTER_SELF_METRICS_ENABLED", false),
	}
	logLevel := envString("EXTENDED_CEPH_EXPORTER_LOG_LEVEL", "info")

	fs.StringVar(&cfg.ListenAddress, "listen-address", cfg.ListenAddress, "Address to bind the HTTP server to.")
	fs.StringVar(&cfg.MetricsPath, "metrics-path", cfg.MetricsPath, "HTTP path used to expose Prometheus metrics.")
	fs.DurationVar(&cfg.RequestTimeout, "request-timeout", cfg.RequestTimeout, "Timeout for upstream Ceph or RGW requests.")
	fs.DurationVar(&cfg.RGWCacheTTL, "rgw-cache-ttl", cfg.RGWCacheTTL, "Short-lived cache TTL for RGW-derived metrics.")
	fs.StringVar(&cfg.RGWAdminEndpoint, "rgw-admin-endpoint", cfg.RGWAdminEndpoint, "RGW Admin Ops API endpoint, for example https://rook-ceph-rgw-foo.rook-ceph.svc.")
	fs.StringVar(&cfg.RGWAccessKey, "rgw-access-key", cfg.RGWAccessKey, "RGW admin access key used to authenticate Admin Ops API requests.")
	fs.StringVar(&cfg.RGWSecretKey, "rgw-secret-key", cfg.RGWSecretKey, "RGW admin secret key used to authenticate Admin Ops API requests.")
	fs.BoolVar(&cfg.SelfMetricsEnabled, "self-metrics-enabled", cfg.SelfMetricsEnabled, "Expose exporter-internal RGW refresh and cache metrics.")
	fs.StringVar(&logLevel, "log-level", logLevel, "Log level: debug, info, warn, or error.")
	_ = fs.Parse(args)

	cfg.LogLevel = parseLogLevel(logLevel)
	return cfg
}

func parseLogLevel(value string) slog.Level {
	switch value {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

func envString(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}

	return fallback
}

func envDuration(key string, fallback time.Duration) time.Duration {
	value, ok := os.LookupEnv(key)
	if !ok || value == "" {
		return fallback
	}

	parsed, err := time.ParseDuration(value)
	if err != nil {
		return fallback
	}

	return parsed
}

func envBool(key string, fallback bool) bool {
	value, ok := os.LookupEnv(key)
	if !ok || value == "" {
		return fallback
	}

	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}

	return parsed
}
