package rgw

import (
	"context"
	"time"
)

type Bucket struct {
	Zonegroup         string
	Store             string
	Bucket            string
	User              string
	Tenant            string
	UsageBytes        float64
	Objects           float64
	QuotaEnabled      *bool
	QuotaMaxSizeBytes *float64
	QuotaMaxObjects   *float64
}

type User struct {
	Zonegroup         string
	Store             string
	User              string
	Tenant            string
	UsageBytes        float64
	Objects           float64
	BucketCount       float64
	QuotaEnabled      *bool
	QuotaMaxSizeBytes *float64
	QuotaMaxObjects   *float64
	Suspended         bool
	MaxBuckets        float64
}

type BucketSource interface {
	ListBuckets(ctx context.Context) ([]Bucket, error)
}

type UserSource interface {
	ListUsers(ctx context.Context, buckets []Bucket) ([]User, error)
}

type Snapshot struct {
	Buckets     []Bucket
	Users       []User
	CollectedAt time.Time
}
