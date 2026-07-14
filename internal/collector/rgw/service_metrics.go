package rgw

import "github.com/prometheus/client_golang/prometheus"

type ServiceMetrics struct {
	cacheHits      prometheus.Counter
	cacheMisses    prometheus.Counter
	refreshSuccess prometheus.Counter
	refreshFailure prometheus.Counter
	refreshSeconds prometheus.Histogram
	lastSuccess    prometheus.Gauge
}

func NewServiceMetrics() *ServiceMetrics {
	return &ServiceMetrics{
		cacheHits: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "extended_ceph_exporter_rgw_cache_hits_total",
			Help: "Total number of RGW snapshot cache hits.",
		}),
		cacheMisses: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "extended_ceph_exporter_rgw_cache_misses_total",
			Help: "Total number of RGW snapshot cache misses.",
		}),
		refreshSuccess: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "extended_ceph_exporter_rgw_refresh_success_total",
			Help: "Total number of successful RGW snapshot refreshes.",
		}),
		refreshFailure: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "extended_ceph_exporter_rgw_refresh_failures_total",
			Help: "Total number of failed RGW snapshot refreshes.",
		}),
		refreshSeconds: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    "extended_ceph_exporter_rgw_refresh_duration_seconds",
			Help:    "Duration of RGW snapshot refreshes in seconds.",
			Buckets: prometheus.DefBuckets,
		}),
		lastSuccess: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "extended_ceph_exporter_rgw_last_success_unixtime",
			Help: "Unix timestamp of the last successful RGW snapshot refresh.",
		}),
	}
}

func (m *ServiceMetrics) Collectors() []prometheus.Collector {
	return []prometheus.Collector{
		m.cacheHits,
		m.cacheMisses,
		m.refreshSuccess,
		m.refreshFailure,
		m.refreshSeconds,
		m.lastSuccess,
	}
}
