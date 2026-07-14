package rgwclient

import (
	"context"

	cephadmin "github.com/ceph/go-ceph/rgw/admin"

	"github.com/FlorisFeddema/extended-ceph-exporter/internal/collector/rgw"
)

type UserSource struct {
	client *Client
}

func NewUserSource(client *Client) UserSource {
	return UserSource{client: client}
}

func (s UserSource) ListUsers(ctx context.Context) ([]rgw.User, error) {
	store := s.client.storeLabel(ctx)

	userIDs, err := s.client.admin.GetUsers(ctx)
	if err != nil {
		return nil, err
	}

	if userIDs == nil {
		return nil, nil
	}

	result := make([]rgw.User, 0, len(*userIDs))
	for _, userID := range *userIDs {
		user, err := s.client.admin.GetUser(ctx, cephadmin.User{
			ID:           userID,
			GenerateStat: new(true),
		})
		if err != nil {
			return nil, err
		}

		quota, quotaErr := s.client.admin.GetUserQuota(ctx, cephadmin.QuotaSpec{UID: userID})
		buckets, err := s.client.admin.ListUsersBucketsWithStat(ctx, userID)
		if err != nil {
			return nil, err
		}

		quotaEnabled := (*bool)(nil)
		quotaMaxSizeBytes := (*float64)(nil)
		quotaMaxObjects := (*float64)(nil)
		if quotaErr == nil {
			quotaEnabled = quota.Enabled
			quotaMaxSizeBytes = quotaSizeLimit(quota.Enabled, quota.MaxSize, quota.MaxSizeKb)
			quotaMaxObjects = quotaObjectsLimit(quota.Enabled, quota.MaxObjects)
		}

		result = append(result, rgw.User{
			Realm:             userRealm(buckets),
			Store:             store,
			User:              user.ID,
			Tenant:            user.Tenant,
			UsageBytes:        uint64PtrFloat(user.Stat.Size),
			Objects:           uint64PtrFloat(user.Stat.NumObjects),
			BucketCount:       float64(len(buckets)),
			QuotaEnabled:      quotaEnabled,
			QuotaMaxSizeBytes: quotaMaxSizeBytes,
			QuotaMaxObjects:   quotaMaxObjects,
			Suspended:         intPtrBool(user.Suspended),
			MaxBuckets:        intPtrFloat(user.MaxBuckets),
		})
	}

	return result, nil
}

func userRealm(buckets []cephadmin.Bucket) string {
	realm := ""
	for _, bucket := range buckets {
		if bucket.Zonegroup == "" {
			continue
		}

		if realm == "" {
			realm = bucket.Zonegroup
			continue
		}

		if realm != bucket.Zonegroup {
			return mixedLabelValue
		}
	}

	if realm == "" {
		return unknownLabelValue
	}

	return realm
}

func intPtrBool(value *int) bool {
	if value == nil {
		return false
	}

	return *value != 0
}

func intPtrFloat(value *int) float64 {
	if value == nil {
		return 0
	}

	return float64(*value)
}
