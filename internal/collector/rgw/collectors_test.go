package rgw

import (
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

func float64Ref(value float64) *float64 {
	return &value
}

func boolRef(value bool) *bool {
	return &value
}

func TestBoolFloat(t *testing.T) {
	if boolFloat(true) != 1 || boolFloat(false) != 0 {
		t.Fatal("boolFloat returned unexpected values")
	}
}

func TestBucketsCollectorCollectsMetrics(t *testing.T) {
	service := NewService(
		StaticBucketSource{Buckets: []Bucket{{
			Realm:             "realm-a",
			Store:             "store-a",
			Bucket:            "bucket-a",
			User:              "user-a",
			Tenant:            "tenant-a",
			UsageBytes:        10,
			Objects:           4,
			QuotaEnabled:      boolRef(true),
			QuotaMaxSizeBytes: float64Ref(20),
			QuotaMaxObjects:   float64Ref(30),
		}}},
		StaticUserSource{},
		time.Minute,
	)

	collector := NewBucketsCollector(service, time.Second)
	registry := prometheus.NewRegistry()
	registry.MustRegister(collector)

	expected := `
# HELP extended_ceph_rgw_bucket_info Static presence metric for an RGW bucket.
# TYPE extended_ceph_rgw_bucket_info gauge
extended_ceph_rgw_bucket_info{bucket="bucket-a",realm="realm-a",store="store-a",tenant="tenant-a",user="user-a"} 1
# HELP extended_ceph_rgw_bucket_objects Current object count for an RGW bucket.
# TYPE extended_ceph_rgw_bucket_objects gauge
extended_ceph_rgw_bucket_objects{bucket="bucket-a",realm="realm-a",store="store-a",tenant="tenant-a",user="user-a"} 4
# HELP extended_ceph_rgw_bucket_quota_enabled Whether the RGW bucket quota is enabled.
# TYPE extended_ceph_rgw_bucket_quota_enabled gauge
extended_ceph_rgw_bucket_quota_enabled{bucket="bucket-a",realm="realm-a",store="store-a",tenant="tenant-a",user="user-a"} 1
# HELP extended_ceph_rgw_bucket_quota_max_objects Configured maximum RGW bucket quota object count.
# TYPE extended_ceph_rgw_bucket_quota_max_objects gauge
extended_ceph_rgw_bucket_quota_max_objects{bucket="bucket-a",realm="realm-a",store="store-a",tenant="tenant-a",user="user-a"} 30
# HELP extended_ceph_rgw_bucket_quota_max_size_bytes Configured maximum RGW bucket quota size in bytes.
# TYPE extended_ceph_rgw_bucket_quota_max_size_bytes gauge
extended_ceph_rgw_bucket_quota_max_size_bytes{bucket="bucket-a",realm="realm-a",store="store-a",tenant="tenant-a",user="user-a"} 20
# HELP extended_ceph_rgw_bucket_usage_bytes Current RGW bucket size in bytes.
# TYPE extended_ceph_rgw_bucket_usage_bytes gauge
extended_ceph_rgw_bucket_usage_bytes{bucket="bucket-a",realm="realm-a",store="store-a",tenant="tenant-a",user="user-a"} 10
`

	if err := testutil.GatherAndCompare(registry, strings.NewReader(expected)); err != nil {
		t.Fatalf("unexpected bucket metrics: %v", err)
	}
}

func TestUsersCollectorCollectsMetrics(t *testing.T) {
	service := NewService(
		StaticBucketSource{},
		StaticUserSource{Users: []User{{
			Realm:             "realm-a",
			Store:             "store-a",
			User:              "user-a",
			Tenant:            "tenant-a",
			UsageBytes:        10,
			Objects:           4,
			BucketCount:       2,
			QuotaEnabled:      boolRef(true),
			QuotaMaxSizeBytes: float64Ref(20),
			QuotaMaxObjects:   float64Ref(30),
			Suspended:         true,
			MaxBuckets:        50,
		}}},
		time.Minute,
	)

	collector := NewUsersCollector(service, time.Second)
	registry := prometheus.NewRegistry()
	registry.MustRegister(collector)

	expected := `
# HELP extended_ceph_rgw_user_bucket_count Number of buckets owned by an RGW user.
# TYPE extended_ceph_rgw_user_bucket_count gauge
extended_ceph_rgw_user_bucket_count{realm="realm-a",store="store-a",tenant="tenant-a",user="user-a"} 2
# HELP extended_ceph_rgw_user_info Static presence metric for an RGW user.
# TYPE extended_ceph_rgw_user_info gauge
extended_ceph_rgw_user_info{realm="realm-a",store="store-a",tenant="tenant-a",user="user-a"} 1
# HELP extended_ceph_rgw_user_max_buckets Configured maximum number of buckets for an RGW user.
# TYPE extended_ceph_rgw_user_max_buckets gauge
extended_ceph_rgw_user_max_buckets{realm="realm-a",store="store-a",tenant="tenant-a",user="user-a"} 50
# HELP extended_ceph_rgw_user_objects Total objects across buckets owned by an RGW user.
# TYPE extended_ceph_rgw_user_objects gauge
extended_ceph_rgw_user_objects{realm="realm-a",store="store-a",tenant="tenant-a",user="user-a"} 4
# HELP extended_ceph_rgw_user_quota_enabled Whether the RGW user quota is enabled.
# TYPE extended_ceph_rgw_user_quota_enabled gauge
extended_ceph_rgw_user_quota_enabled{realm="realm-a",store="store-a",tenant="tenant-a",user="user-a"} 1
# HELP extended_ceph_rgw_user_quota_max_objects Configured maximum RGW user quota object count.
# TYPE extended_ceph_rgw_user_quota_max_objects gauge
extended_ceph_rgw_user_quota_max_objects{realm="realm-a",store="store-a",tenant="tenant-a",user="user-a"} 30
# HELP extended_ceph_rgw_user_quota_max_size_bytes Configured maximum RGW user quota size in bytes.
# TYPE extended_ceph_rgw_user_quota_max_size_bytes gauge
extended_ceph_rgw_user_quota_max_size_bytes{realm="realm-a",store="store-a",tenant="tenant-a",user="user-a"} 20
# HELP extended_ceph_rgw_user_suspended Whether the RGW user is suspended.
# TYPE extended_ceph_rgw_user_suspended gauge
extended_ceph_rgw_user_suspended{realm="realm-a",store="store-a",tenant="tenant-a",user="user-a"} 1
# HELP extended_ceph_rgw_user_usage_bytes Total bytes used across buckets owned by an RGW user.
# TYPE extended_ceph_rgw_user_usage_bytes gauge
extended_ceph_rgw_user_usage_bytes{realm="realm-a",store="store-a",tenant="tenant-a",user="user-a"} 10
`

	if err := testutil.GatherAndCompare(registry, strings.NewReader(expected)); err != nil {
		t.Fatalf("unexpected user metrics: %v", err)
	}
}

func TestCollectorsOmitUnlimitedQuotaMetrics(t *testing.T) {
	service := NewService(
		StaticBucketSource{Buckets: []Bucket{{
			Realm:        "realm-a",
			Store:        "store-a",
			Bucket:       "bucket-a",
			User:         "user-a",
			Tenant:       "tenant-a",
			QuotaEnabled: boolRef(true),
		}}},
		StaticUserSource{Users: []User{{
			Realm:        "realm-a",
			Store:        "store-a",
			User:         "user-a",
			Tenant:       "tenant-a",
			QuotaEnabled: boolRef(true),
		}}},
		time.Minute,
	)

	bucketRegistry := prometheus.NewRegistry()
	bucketRegistry.MustRegister(NewBucketsCollector(service, time.Second))
	userRegistry := prometheus.NewRegistry()
	userRegistry.MustRegister(NewUsersCollector(service, time.Second))

	bucketFamilies, err := bucketRegistry.Gather()
	if err != nil {
		t.Fatalf("gather bucket metrics failed: %v", err)
	}
	userFamilies, err := userRegistry.Gather()
	if err != nil {
		t.Fatalf("gather user metrics failed: %v", err)
	}

	for _, family := range bucketFamilies {
		if family.GetName() == "extended_ceph_rgw_bucket_quota_max_size_bytes" || family.GetName() == "extended_ceph_rgw_bucket_quota_max_objects" {
			if len(family.Metric) != 0 {
				t.Fatalf("expected omitted bucket quota max metrics, got %d", len(family.Metric))
			}
		}
	}

	for _, family := range userFamilies {
		if family.GetName() == "extended_ceph_rgw_user_quota_max_size_bytes" || family.GetName() == "extended_ceph_rgw_user_quota_max_objects" {
			if len(family.Metric) != 0 {
				t.Fatalf("expected omitted user quota max metrics, got %d", len(family.Metric))
			}
		}
	}
}

func TestCollectorsOmitUnknownQuotaMetrics(t *testing.T) {
	service := NewService(
		StaticBucketSource{Buckets: []Bucket{{
			Realm:  "realm-a",
			Store:  "store-a",
			Bucket: "bucket-a",
			User:   "user-a",
			Tenant: "tenant-a",
		}}},
		StaticUserSource{Users: []User{{
			Realm:  "realm-a",
			Store:  "store-a",
			User:   "user-a",
			Tenant: "tenant-a",
		}}},
		time.Minute,
	)

	bucketRegistry := prometheus.NewRegistry()
	bucketRegistry.MustRegister(NewBucketsCollector(service, time.Second))
	userRegistry := prometheus.NewRegistry()
	userRegistry.MustRegister(NewUsersCollector(service, time.Second))

	bucketFamilies, err := bucketRegistry.Gather()
	if err != nil {
		t.Fatalf("gather bucket metrics failed: %v", err)
	}
	userFamilies, err := userRegistry.Gather()
	if err != nil {
		t.Fatalf("gather user metrics failed: %v", err)
	}

	for _, family := range bucketFamilies {
		if family.GetName() == "extended_ceph_rgw_bucket_quota_enabled" || family.GetName() == "extended_ceph_rgw_bucket_quota_max_size_bytes" || family.GetName() == "extended_ceph_rgw_bucket_quota_max_objects" {
			if len(family.Metric) != 0 {
				t.Fatalf("expected omitted unknown bucket quota metrics, got %d", len(family.Metric))
			}
		}
	}

	for _, family := range userFamilies {
		if family.GetName() == "extended_ceph_rgw_user_quota_enabled" || family.GetName() == "extended_ceph_rgw_user_quota_max_size_bytes" || family.GetName() == "extended_ceph_rgw_user_quota_max_objects" {
			if len(family.Metric) != 0 {
				t.Fatalf("expected omitted unknown user quota metrics, got %d", len(family.Metric))
			}
		}
	}
}
