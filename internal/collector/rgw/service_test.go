package rgw

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

type countingBucketSource struct {
	buckets []Bucket
	err     error
	calls   int
}

func (s *countingBucketSource) ListBuckets(context.Context) ([]Bucket, error) {
	s.calls++
	if s.err != nil {
		return nil, s.err
	}
	return s.buckets, nil
}

type countingUserSource struct {
	users []User
	err   error
	calls int
}

func (s *countingUserSource) ListUsers(context.Context) ([]User, error) {
	s.calls++
	if s.err != nil {
		return nil, s.err
	}
	return s.users, nil
}

func TestServiceSnapshotCachesResults(t *testing.T) {
	bucketSource := &countingBucketSource{buckets: []Bucket{{Bucket: "a"}}}
	userSource := &countingUserSource{users: []User{{User: "u"}}}
	service := NewService(bucketSource, userSource, time.Minute)
	now := time.Unix(100, 0)
	service.now = func() time.Time { return now }

	first, err := service.Snapshot(context.Background())
	if err != nil {
		t.Fatalf("first snapshot failed: %v", err)
	}
	second, err := service.Snapshot(context.Background())
	if err != nil {
		t.Fatalf("second snapshot failed: %v", err)
	}

	if bucketSource.calls != 1 || userSource.calls != 1 {
		t.Fatalf("expected one upstream call, got buckets=%d users=%d", bucketSource.calls, userSource.calls)
	}
	if len(first.Buckets) != 1 || len(second.Users) != 1 {
		t.Fatalf("unexpected snapshot contents: first=%+v second=%+v", first, second)
	}
}

func TestServiceSnapshotRefreshesAfterTTL(t *testing.T) {
	bucketSource := &countingBucketSource{buckets: []Bucket{{Bucket: "a"}}}
	userSource := &countingUserSource{users: []User{{User: "u"}}}
	service := NewService(bucketSource, userSource, time.Second)
	now := time.Unix(100, 0)
	service.now = func() time.Time { return now }

	_, _ = service.Snapshot(context.Background())
	now = now.Add(2 * time.Second)
	_, _ = service.Snapshot(context.Background())

	if bucketSource.calls != 2 || userSource.calls != 2 {
		t.Fatalf("expected refresh after TTL, got buckets=%d users=%d", bucketSource.calls, userSource.calls)
	}
}

func TestServiceSnapshotReturnsBucketError(t *testing.T) {
	service := NewService(&countingBucketSource{err: errors.New("boom")}, &countingUserSource{}, time.Minute)

	_, err := service.Snapshot(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestServiceSnapshotReturnsUserError(t *testing.T) {
	service := NewService(&countingBucketSource{}, &countingUserSource{err: errors.New("boom")}, time.Minute)

	_, err := service.Snapshot(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestServiceMetricsTrackCacheAndRefresh(t *testing.T) {
	bucketSource := &countingBucketSource{buckets: []Bucket{{Bucket: "a"}}}
	userSource := &countingUserSource{users: []User{{User: "u"}}}
	metrics := NewServiceMetrics()
	service := NewServiceWithMetrics(bucketSource, userSource, time.Minute, metrics)
	now := time.Unix(100, 0)
	service.now = func() time.Time { return now }

	registry := prometheus.NewRegistry()
	registry.MustRegister(metrics.Collectors()...)

	if _, err := service.Snapshot(context.Background()); err != nil {
		t.Fatalf("first snapshot failed: %v", err)
	}
	if _, err := service.Snapshot(context.Background()); err != nil {
		t.Fatalf("second snapshot failed: %v", err)
	}

	expected := `
# HELP extended_ceph_exporter_rgw_cache_hits_total Total number of RGW snapshot cache hits.
# TYPE extended_ceph_exporter_rgw_cache_hits_total counter
extended_ceph_exporter_rgw_cache_hits_total 1
# HELP extended_ceph_exporter_rgw_cache_misses_total Total number of RGW snapshot cache misses.
# TYPE extended_ceph_exporter_rgw_cache_misses_total counter
extended_ceph_exporter_rgw_cache_misses_total 1
# HELP extended_ceph_exporter_rgw_last_success_unixtime Unix timestamp of the last successful RGW snapshot refresh.
# TYPE extended_ceph_exporter_rgw_last_success_unixtime gauge
extended_ceph_exporter_rgw_last_success_unixtime 100
# HELP extended_ceph_exporter_rgw_refresh_failures_total Total number of failed RGW snapshot refreshes.
# TYPE extended_ceph_exporter_rgw_refresh_failures_total counter
extended_ceph_exporter_rgw_refresh_failures_total 0
# HELP extended_ceph_exporter_rgw_refresh_success_total Total number of successful RGW snapshot refreshes.
# TYPE extended_ceph_exporter_rgw_refresh_success_total counter
extended_ceph_exporter_rgw_refresh_success_total 1
`

	if err := testutil.GatherAndCompare(
		registry,
		strings.NewReader(expected),
		"extended_ceph_exporter_rgw_cache_hits_total",
		"extended_ceph_exporter_rgw_cache_misses_total",
		"extended_ceph_exporter_rgw_last_success_unixtime",
		"extended_ceph_exporter_rgw_refresh_failures_total",
		"extended_ceph_exporter_rgw_refresh_success_total",
	); err != nil {
		t.Fatalf("unexpected service metrics: %v", err)
	}
}

func TestServiceMetricsTrackFailures(t *testing.T) {
	metrics := NewServiceMetrics()
	service := NewServiceWithMetrics(&countingBucketSource{err: errors.New("boom")}, &countingUserSource{}, time.Minute, metrics)
	now := time.Unix(200, 0)
	service.now = func() time.Time { return now }

	registry := prometheus.NewRegistry()
	registry.MustRegister(metrics.Collectors()...)

	if _, err := service.Snapshot(context.Background()); err == nil {
		t.Fatal("expected snapshot error")
	}

	expected := `
# HELP extended_ceph_exporter_rgw_cache_hits_total Total number of RGW snapshot cache hits.
# TYPE extended_ceph_exporter_rgw_cache_hits_total counter
extended_ceph_exporter_rgw_cache_hits_total 0
# HELP extended_ceph_exporter_rgw_cache_misses_total Total number of RGW snapshot cache misses.
# TYPE extended_ceph_exporter_rgw_cache_misses_total counter
extended_ceph_exporter_rgw_cache_misses_total 1
# HELP extended_ceph_exporter_rgw_refresh_failures_total Total number of failed RGW snapshot refreshes.
# TYPE extended_ceph_exporter_rgw_refresh_failures_total counter
extended_ceph_exporter_rgw_refresh_failures_total 1
# HELP extended_ceph_exporter_rgw_refresh_success_total Total number of successful RGW snapshot refreshes.
# TYPE extended_ceph_exporter_rgw_refresh_success_total counter
extended_ceph_exporter_rgw_refresh_success_total 0
`

	if err := testutil.GatherAndCompare(
		registry,
		strings.NewReader(expected),
		"extended_ceph_exporter_rgw_cache_hits_total",
		"extended_ceph_exporter_rgw_cache_misses_total",
		"extended_ceph_exporter_rgw_refresh_failures_total",
		"extended_ceph_exporter_rgw_refresh_success_total",
	); err != nil {
		t.Fatalf("unexpected failure metrics: %v", err)
	}
}
