package rgw

import (
	"context"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

var userLabels = []string{"realm", "store", "user", "tenant"}

type UsersCollector struct {
	service         *Service
	scrapeTimeout   time.Duration
	infoDesc        *prometheus.Desc
	usageBytesDesc  *prometheus.Desc
	objectsDesc     *prometheus.Desc
	bucketCountDesc *prometheus.Desc
	quotaOnDesc     *prometheus.Desc
	quotaSizeDesc   *prometheus.Desc
	quotaObjsDesc   *prometheus.Desc
	suspendedDesc   *prometheus.Desc
	maxBucketsDesc  *prometheus.Desc
}

func NewUsersCollector(service *Service, scrapeTimeout time.Duration) *UsersCollector {
	return &UsersCollector{
		service:         service,
		scrapeTimeout:   scrapeTimeout,
		infoDesc:        prometheus.NewDesc("extended_ceph_rgw_user_info", "Static presence metric for an RGW user.", userLabels, nil),
		usageBytesDesc:  prometheus.NewDesc("extended_ceph_rgw_user_usage_bytes", "Total bytes used across buckets owned by an RGW user.", userLabels, nil),
		objectsDesc:     prometheus.NewDesc("extended_ceph_rgw_user_objects", "Total objects across buckets owned by an RGW user.", userLabels, nil),
		bucketCountDesc: prometheus.NewDesc("extended_ceph_rgw_user_bucket_count", "Number of buckets owned by an RGW user.", userLabels, nil),
		quotaOnDesc:     prometheus.NewDesc("extended_ceph_rgw_user_quota_enabled", "Whether the RGW user quota is enabled.", userLabels, nil),
		quotaSizeDesc:   prometheus.NewDesc("extended_ceph_rgw_user_quota_max_size_bytes", "Configured maximum RGW user quota size in bytes.", userLabels, nil),
		quotaObjsDesc:   prometheus.NewDesc("extended_ceph_rgw_user_quota_max_objects", "Configured maximum RGW user quota object count.", userLabels, nil),
		suspendedDesc:   prometheus.NewDesc("extended_ceph_rgw_user_suspended", "Whether the RGW user is suspended.", userLabels, nil),
		maxBucketsDesc:  prometheus.NewDesc("extended_ceph_rgw_user_max_buckets", "Configured maximum number of buckets for an RGW user.", userLabels, nil),
	}
}

func (c *UsersCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.infoDesc
	ch <- c.usageBytesDesc
	ch <- c.objectsDesc
	ch <- c.bucketCountDesc
	ch <- c.quotaOnDesc
	ch <- c.quotaSizeDesc
	ch <- c.quotaObjsDesc
	ch <- c.suspendedDesc
	ch <- c.maxBucketsDesc
}

func (c *UsersCollector) Collect(ch chan<- prometheus.Metric) {
	ctx, cancel := context.WithTimeout(context.Background(), c.scrapeTimeout)
	defer cancel()

	snapshot, err := c.service.Snapshot(ctx)
	if err != nil {
		ch <- prometheus.NewInvalidMetric(c.infoDesc, err)
		return
	}

	for _, user := range snapshot.Users {
		values := []string{user.Realm, user.Store, user.User, user.Tenant}
		ch <- prometheus.MustNewConstMetric(c.infoDesc, prometheus.GaugeValue, 1, values...)
		ch <- prometheus.MustNewConstMetric(c.usageBytesDesc, prometheus.GaugeValue, user.UsageBytes, values...)
		ch <- prometheus.MustNewConstMetric(c.objectsDesc, prometheus.GaugeValue, user.Objects, values...)
		ch <- prometheus.MustNewConstMetric(c.bucketCountDesc, prometheus.GaugeValue, user.BucketCount, values...)
		if user.QuotaEnabled != nil {
			ch <- prometheus.MustNewConstMetric(c.quotaOnDesc, prometheus.GaugeValue, boolFloat(*user.QuotaEnabled), values...)
		}
		if user.QuotaMaxSizeBytes != nil {
			ch <- prometheus.MustNewConstMetric(c.quotaSizeDesc, prometheus.GaugeValue, *user.QuotaMaxSizeBytes, values...)
		}
		if user.QuotaMaxObjects != nil {
			ch <- prometheus.MustNewConstMetric(c.quotaObjsDesc, prometheus.GaugeValue, *user.QuotaMaxObjects, values...)
		}
		ch <- prometheus.MustNewConstMetric(c.suspendedDesc, prometheus.GaugeValue, boolFloat(user.Suspended), values...)
		ch <- prometheus.MustNewConstMetric(c.maxBucketsDesc, prometheus.GaugeValue, user.MaxBuckets, values...)
	}
}
