package rgw

import (
	"context"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"
)

type Service struct {
	bucketSource BucketSource
	userSource   UserSource
	cacheTTL     time.Duration
	metrics      *ServiceMetrics
	now          func() time.Time

	mu       sync.Mutex
	cached   Snapshot
	expires  time.Time
	hasCache bool

	group singleflight.Group
}

func NewService(bucketSource BucketSource, userSource UserSource, cacheTTL time.Duration) *Service {
	return NewServiceWithMetrics(bucketSource, userSource, cacheTTL, nil)
}

func NewServiceWithMetrics(bucketSource BucketSource, userSource UserSource, cacheTTL time.Duration, metrics *ServiceMetrics) *Service {
	return &Service{
		bucketSource: bucketSource,
		userSource:   userSource,
		cacheTTL:     cacheTTL,
		metrics:      metrics,
		now:          time.Now,
	}
}

func (s *Service) Snapshot(ctx context.Context) (Snapshot, error) {
	// Fast path: serve from cache without blocking the fetch.
	s.mu.Lock()
	if s.hasCache && s.now().Before(s.expires) {
		cached := s.cached
		s.mu.Unlock()
		if s.metrics != nil {
			s.metrics.cacheHits.Inc()
		}
		return cached, nil
	}
	s.mu.Unlock()

	if s.metrics != nil {
		s.metrics.cacheMisses.Inc()
	}

	// Slow path: deduplicate concurrent refreshes so only one upstream fetch
	// runs at a time; other callers wait and share the result.
	v, err, _ := s.group.Do("snapshot", func() (any, error) {
		return s.refresh(ctx)
	})
	if err != nil {
		return Snapshot{}, err
	}
	return v.(Snapshot), nil
}

func (s *Service) refresh(ctx context.Context) (Snapshot, error) {
	refreshStart := s.now()

	buckets, err := s.bucketSource.ListBuckets(ctx)
	if err != nil {
		s.observeRefreshFailure(refreshStart)
		return Snapshot{}, err
	}

	users, err := s.userSource.ListUsers(ctx, buckets)
	if err != nil {
		s.observeRefreshFailure(refreshStart)
		return Snapshot{}, err
	}

	refreshedAt := s.now()
	snap := Snapshot{
		Buckets:     buckets,
		Users:       users,
		CollectedAt: refreshedAt,
	}

	s.mu.Lock()
	s.cached = snap
	s.expires = refreshedAt.Add(s.cacheTTL)
	s.hasCache = true
	s.mu.Unlock()

	s.observeRefreshSuccess(refreshStart, refreshedAt)
	return snap, nil
}

func (s *Service) observeRefreshSuccess(start, end time.Time) {
	if s.metrics == nil {
		return
	}

	s.metrics.refreshSuccess.Inc()
	s.metrics.refreshSeconds.Observe(end.Sub(start).Seconds())
	s.metrics.lastSuccess.Set(float64(end.Unix()))
}

func (s *Service) observeRefreshFailure(start time.Time) {
	if s.metrics == nil {
		return
	}

	s.metrics.refreshFailure.Inc()
	s.metrics.refreshSeconds.Observe(s.now().Sub(start).Seconds())
}

type StaticBucketSource struct {
	Buckets []Bucket
}

func (s StaticBucketSource) ListBuckets(_ context.Context) ([]Bucket, error) {
	return s.Buckets, nil
}

type StaticUserSource struct {
	Users []User
}

func (s StaticUserSource) ListUsers(_ context.Context, _ []Bucket) ([]User, error) {
	return s.Users, nil
}
