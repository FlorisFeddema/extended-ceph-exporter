package rgwclient

import (
	"context"

	cephadmin "github.com/ceph/go-ceph/rgw/admin"

	"github.com/FlorisFeddema/extended-ceph-exporter/internal/collector/rgw"
)

type BucketSource struct {
	client *Client
}

func NewBucketSource(client *Client) BucketSource {
	return BucketSource{client: client}
}

func (s BucketSource) ListBuckets(ctx context.Context) ([]rgw.Bucket, error) {
	store := s.client.storeLabel(ctx)

	buckets, err := s.client.admin.ListBucketsWithStat(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]rgw.Bucket, 0, len(buckets))
	for _, bucket := range buckets {
		bucketInfo, err := s.client.admin.GetBucketInfo(ctx, cephadmin.Bucket{Bucket: bucket.Bucket})
		quotaEnabled := (*bool)(nil)
		quotaMaxSizeBytes := (*float64)(nil)
		quotaMaxObjects := (*float64)(nil)
		realm := zonegroupRealm(bucket.Zonegroup)
		user := bucket.Owner
		tenant := bucket.Tenant

		if err == nil {
			realm = zonegroupRealm(firstNonEmpty(bucketInfo.Zonegroup, bucket.Zonegroup))
			user = firstNonEmpty(bucketInfo.Owner, bucket.Owner)
			tenant = firstNonEmpty(bucketInfo.Tenant, bucket.Tenant)
			quotaEnabled = bucketInfo.BucketQuota.Enabled
			quotaMaxSizeBytes = quotaSizeLimit(bucketInfo.BucketQuota.Enabled, bucketInfo.BucketQuota.MaxSize, bucketInfo.BucketQuota.MaxSizeKb)
			quotaMaxObjects = quotaObjectsLimit(bucketInfo.BucketQuota.Enabled, bucketInfo.BucketQuota.MaxObjects)
		}

		result = append(result, rgw.Bucket{
			Realm:             realm,
			Store:             store,
			Bucket:            bucket.Bucket,
			User:              user,
			Tenant:            tenant,
			UsageBytes:        uint64PtrFloat(bucket.Usage.RgwMain.Size),
			Objects:           uint64PtrFloat(bucket.Usage.RgwMain.NumObjects),
			QuotaEnabled:      quotaEnabled,
			QuotaMaxSizeBytes: quotaMaxSizeBytes,
			QuotaMaxObjects:   quotaMaxObjects,
		})
	}

	return result, nil
}

func zonegroupRealm(zonegroup string) string {
	if zonegroup == "" {
		return unknownLabelValue
	}

	return zonegroup
}

func uint64PtrFloat(value *uint64) float64 {
	if value == nil {
		return 0
	}

	return float64(*value)
}

func int64PtrFloat(value *int64) float64 {
	if value == nil {
		return 0
	}

	return float64(*value)
}

func boolPtrValue(value *bool) bool {
	if value == nil {
		return false
	}

	return *value
}

func quotaSizeLimit(enabled *bool, maxSize *int64, maxSizeKB *int) *float64 {
	if !boolPtrValue(enabled) {
		return nil
	}

	if maxSize != nil && *maxSize > 0 {
		return float64Ptr(float64(*maxSize))
	}

	if maxSizeKB != nil && *maxSizeKB > 0 {
		return float64Ptr(float64(*maxSizeKB) * 1024)
	}

	return nil
}

func quotaObjectsLimit(enabled *bool, maxObjects *int64) *float64 {
	if !boolPtrValue(enabled) {
		return nil
	}

	if maxObjects != nil && *maxObjects > 0 {
		return float64Ptr(float64(*maxObjects))
	}

	return nil
}

func float64Ptr(value float64) *float64 {
	return &value
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}

	return ""
}
