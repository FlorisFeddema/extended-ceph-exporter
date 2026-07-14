package rgw

import (
	"context"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

var bucketLabels = []string{"realm", "store", "bucket", "user", "tenant"}

type BucketsCollector struct {
	service        *Service
	scrapeTimeout  time.Duration
	infoDesc       *prometheus.Desc
	usageBytesDesc *prometheus.Desc
	objectsDesc    *prometheus.Desc
	quotaOnDesc    *prometheus.Desc
	quotaSizeDesc  *prometheus.Desc
	quotaObjsDesc  *prometheus.Desc
}

func NewBucketsCollector(service *Service, scrapeTimeout time.Duration) *BucketsCollector {
	return &BucketsCollector{
		service:        service,
		scrapeTimeout:  scrapeTimeout,
		infoDesc:       prometheus.NewDesc("extended_ceph_rgw_bucket_info", "Static presence metric for an RGW bucket.", bucketLabels, nil),
		usageBytesDesc: prometheus.NewDesc("extended_ceph_rgw_bucket_usage_bytes", "Current RGW bucket size in bytes.", bucketLabels, nil),
		objectsDesc:    prometheus.NewDesc("extended_ceph_rgw_bucket_objects", "Current object count for an RGW bucket.", bucketLabels, nil),
		quotaOnDesc:    prometheus.NewDesc("extended_ceph_rgw_bucket_quota_enabled", "Whether the RGW bucket quota is enabled.", bucketLabels, nil),
		quotaSizeDesc:  prometheus.NewDesc("extended_ceph_rgw_bucket_quota_max_size_bytes", "Configured maximum RGW bucket quota size in bytes.", bucketLabels, nil),
		quotaObjsDesc:  prometheus.NewDesc("extended_ceph_rgw_bucket_quota_max_objects", "Configured maximum RGW bucket quota object count.", bucketLabels, nil),
	}
}

func (c *BucketsCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.infoDesc
	ch <- c.usageBytesDesc
	ch <- c.objectsDesc
	ch <- c.quotaOnDesc
	ch <- c.quotaSizeDesc
	ch <- c.quotaObjsDesc
}

func (c *BucketsCollector) Collect(ch chan<- prometheus.Metric) {
	ctx, cancel := context.WithTimeout(context.Background(), c.scrapeTimeout)
	defer cancel()

	snapshot, err := c.service.Snapshot(ctx)
	if err != nil {
		ch <- prometheus.NewInvalidMetric(c.infoDesc, err)
		return
	}

	for _, bucket := range snapshot.Buckets {
		values := []string{bucket.Realm, bucket.Store, bucket.Bucket, bucket.User, bucket.Tenant}
		ch <- prometheus.MustNewConstMetric(c.infoDesc, prometheus.GaugeValue, 1, values...)
		ch <- prometheus.MustNewConstMetric(c.usageBytesDesc, prometheus.GaugeValue, bucket.UsageBytes, values...)
		ch <- prometheus.MustNewConstMetric(c.objectsDesc, prometheus.GaugeValue, bucket.Objects, values...)
		if bucket.QuotaEnabled != nil {
			ch <- prometheus.MustNewConstMetric(c.quotaOnDesc, prometheus.GaugeValue, boolFloat(*bucket.QuotaEnabled), values...)
		}
		if bucket.QuotaMaxSizeBytes != nil {
			ch <- prometheus.MustNewConstMetric(c.quotaSizeDesc, prometheus.GaugeValue, *bucket.QuotaMaxSizeBytes, values...)
		}
		if bucket.QuotaMaxObjects != nil {
			ch <- prometheus.MustNewConstMetric(c.quotaObjsDesc, prometheus.GaugeValue, *bucket.QuotaMaxObjects, values...)
		}
	}
}
