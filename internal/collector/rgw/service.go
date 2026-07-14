package rgw

import (
	"context"
	"sync"
	"time"
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
	s.mu.Lock()
	defer s.mu.Unlock()

	now := s.now()
	if s.hasCache && now.Before(s.expires) {
		if s.metrics != nil {
			s.metrics.cacheHits.Inc()
		}
		return s.cached, nil
	}
	if s.metrics != nil {
		s.metrics.cacheMisses.Inc()
	}
	refreshStart := now

	buckets, err := s.bucketSource.ListBuckets(ctx)
	if err != nil {
		s.observeRefreshFailure(refreshStart)
		return Snapshot{}, err
	}

	users, err := s.userSource.ListUsers(ctx)
	if err != nil {
		s.observeRefreshFailure(refreshStart)
		return Snapshot{}, err
	}

	refreshedAt := s.now()
	s.cached = Snapshot{
		Buckets:     buckets,
		Users:       users,
		CollectedAt: refreshedAt,
	}
	s.expires = refreshedAt.Add(s.cacheTTL)
	s.hasCache = true
	s.observeRefreshSuccess(refreshStart, refreshedAt)

	return s.cached, nil
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

func (s StaticUserSource) ListUsers(_ context.Context) ([]User, error) {
	return s.Users, nil
}
