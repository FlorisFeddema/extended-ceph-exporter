package rgwclient

import (
	"context"
	"errors"
	"testing"

	cephadmin "github.com/ceph/go-ceph/rgw/admin"

	"github.com/FlorisFeddema/extended-ceph-exporter/internal/collector/rgw"
)

type fakeAdmin struct {
	info             cephadmin.Info
	infoErr          error
	infoCalls        int
	buckets          []cephadmin.Bucket
	bucketsErr       error
	bucketInfoByName map[string]cephadmin.Bucket
	bucketInfoErr    error
	userIDs          *[]string
	userIDsErr       error
	users            map[string]cephadmin.User
	userErr          error
	quotaByUID       map[string]cephadmin.QuotaSpec
	quotaErr         error
}

func (f *fakeAdmin) GetInfo(context.Context) (cephadmin.Info, error) {
	f.infoCalls++
	if f.infoErr != nil {
		return cephadmin.Info{}, f.infoErr
	}
	return f.info, nil
}

func (f *fakeAdmin) ListBucketsWithStat(context.Context) ([]cephadmin.Bucket, error) {
	if f.bucketsErr != nil {
		return nil, f.bucketsErr
	}
	return f.buckets, nil
}

func (f *fakeAdmin) GetBucketInfo(_ context.Context, bucket cephadmin.Bucket) (cephadmin.Bucket, error) {
	if f.bucketInfoErr != nil {
		return cephadmin.Bucket{}, f.bucketInfoErr
	}
	return f.bucketInfoByName[bucket.Bucket], nil
}

func (f *fakeAdmin) GetUsers(context.Context) (*[]string, error) {
	if f.userIDsErr != nil {
		return nil, f.userIDsErr
	}
	return f.userIDs, nil
}

func (f *fakeAdmin) GetUser(_ context.Context, user cephadmin.User) (cephadmin.User, error) {
	if f.userErr != nil {
		return cephadmin.User{}, f.userErr
	}
	return f.users[user.ID], nil
}

func (f *fakeAdmin) GetUserQuota(_ context.Context, quota cephadmin.QuotaSpec) (cephadmin.QuotaSpec, error) {
	if f.quotaErr != nil {
		return cephadmin.QuotaSpec{}, f.quotaErr
	}
	return f.quotaByUID[quota.UID], nil
}

func TestStoreLabelCachesValue(t *testing.T) {
	admin := &fakeAdmin{
		info: cephadmin.Info{
			InfoSpec: struct {
				StorageBackends []cephadmin.StorageBackend `json:"storage_backends"`
			}{StorageBackends: []cephadmin.StorageBackend{{Name: "beast"}}},
		},
	}
	client := &Client{admin: admin}

	first := client.storeLabel(context.Background())
	second := client.storeLabel(context.Background())

	if first != "beast" || second != "beast" {
		t.Fatalf("unexpected store labels: %q %q", first, second)
	}
	if admin.infoCalls != 1 {
		t.Fatalf("expected cached info lookup, got %d calls", admin.infoCalls)
	}
}

func TestStoreLabelRetriesAfterFailure(t *testing.T) {
	admin := &fakeAdmin{infoErr: errors.New("unavailable")}
	client := &Client{admin: admin}

	got := client.storeLabel(context.Background())
	if got != unknownLabelValue {
		t.Fatalf("expected %q on error, got %q", unknownLabelValue, got)
	}
	if admin.infoCalls != 1 {
		t.Fatalf("expected one call, got %d", admin.infoCalls)
	}

	// Simulate recovery: clear the error and verify the next call retries.
	admin.infoErr = nil
	admin.info = cephadmin.Info{
		InfoSpec: struct {
			StorageBackends []cephadmin.StorageBackend `json:"storage_backends"`
		}{StorageBackends: []cephadmin.StorageBackend{{Name: "beast"}}},
	}
	got = client.storeLabel(context.Background())
	if got != "beast" {
		t.Fatalf("expected store label after recovery, got %q", got)
	}
	if admin.infoCalls != 2 {
		t.Fatalf("expected retry call, got %d total calls", admin.infoCalls)
	}
}

func TestBucketSourceListBucketsMapsFields(t *testing.T) {
	size := uint64(12)
	objects := uint64(3)
	maxSize := int64(100)
	maxObjects := int64(40)
	admin := &fakeAdmin{
		info: cephadmin.Info{
			InfoSpec: struct {
				StorageBackends []cephadmin.StorageBackend `json:"storage_backends"`
			}{StorageBackends: []cephadmin.StorageBackend{{Name: "beast"}}},
		},
		buckets: []cephadmin.Bucket{{
			Bucket:    "bucket-a",
			Zonegroup: "realm-a",
			Owner:     "user-a",
			Tenant:    "tenant-a",
			Usage: cephadmin.BucketUsage{
				RgwMain: cephadmin.RgwUsage{Size: &size, NumObjects: &objects},
			},
		}},
		bucketInfoByName: map[string]cephadmin.Bucket{
			"bucket-a": {
				Bucket:    "bucket-a",
				Zonegroup: "realm-a",
				Owner:     "user-a",
				Tenant:    "tenant-a",
				BucketQuota: cephadmin.QuotaSpec{
					Enabled:    new(true),
					MaxSize:    &maxSize,
					MaxObjects: &maxObjects,
				},
			},
		},
	}

	source := NewBucketSource(&Client{admin: admin})
	buckets, err := source.ListBuckets(context.Background())
	if err != nil {
		t.Fatalf("ListBuckets failed: %v", err)
	}

	if len(buckets) != 1 {
		t.Fatalf("unexpected bucket count: %d", len(buckets))
	}
	got := buckets[0]
	if got.Zonegroup != "realm-a" || got.Store != "beast" || got.UsageBytes != 12 || got.Objects != 3 || got.QuotaEnabled == nil || !*got.QuotaEnabled {
		t.Fatalf("unexpected mapped bucket: %+v", got)
	}
	if got.QuotaMaxSizeBytes == nil || *got.QuotaMaxSizeBytes != 100 || got.QuotaMaxObjects == nil || *got.QuotaMaxObjects != 40 {
		t.Fatalf("unexpected bucket quota mapping: %+v", got)
	}
}

func TestUserSourceListUsersMapsFields(t *testing.T) {
	userIDs := []string{"user-a"}
	suspended := 1
	maxBuckets := 7
	size := uint64(11)
	objects := uint64(5)
	maxSizeKB := 2
	maxObjects := int64(90)

	admin := &fakeAdmin{
		info: cephadmin.Info{
			InfoSpec: struct {
				StorageBackends []cephadmin.StorageBackend `json:"storage_backends"`
			}{StorageBackends: []cephadmin.StorageBackend{{Name: "beast"}}},
		},
		userIDs: &userIDs,
		users: map[string]cephadmin.User{
			"user-a": {
				ID:         "user-a",
				Tenant:     "tenant-a",
				Suspended:  &suspended,
				MaxBuckets: &maxBuckets,
				Stat:       cephadmin.UserStat{Size: &size, NumObjects: &objects},
			},
		},
		quotaByUID: map[string]cephadmin.QuotaSpec{
			"user-a": {
				Enabled:    new(true),
				MaxSizeKb:  &maxSizeKB,
				MaxObjects: &maxObjects,
			},
		},
	}

	source := NewUserSource(&Client{admin: admin})
	inputBuckets := []rgw.Bucket{
		{User: "user-a", Zonegroup: "realm-a"},
		{User: "user-a", Zonegroup: "realm-a"},
	}
	users, err := source.ListUsers(context.Background(), inputBuckets)
	if err != nil {
		t.Fatalf("ListUsers failed: %v", err)
	}

	if len(users) != 1 {
		t.Fatalf("unexpected user count: %d", len(users))
	}
	got := users[0]
	if got.Zonegroup != "realm-a" || got.Store != "beast" || got.BucketCount != 2 || !got.Suspended {
		t.Fatalf("unexpected mapped user: %+v", got)
	}
	if got.QuotaEnabled == nil || !*got.QuotaEnabled {
		t.Fatalf("unexpected user quota enabled mapping: %+v", got)
	}
	if got.QuotaMaxSizeBytes == nil || *got.QuotaMaxSizeBytes != 2048 || got.QuotaMaxObjects == nil || *got.QuotaMaxObjects != 90 {
		t.Fatalf("unexpected user quota mapping: %+v", got)
	}
}

func TestUserSourcePropagatesErrors(t *testing.T) {
	userIDs := []string{"user-a"}
	source := NewUserSource(&Client{admin: &fakeAdmin{
		info:    cephadmin.Info{},
		userIDs: &userIDs,
		userErr: errors.New("boom"),
	}})

	_, err := source.ListUsers(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestBucketSourceOmitsUnlimitedQuotaValues(t *testing.T) {
	admin := &fakeAdmin{
		info: cephadmin.Info{
			InfoSpec: struct {
				StorageBackends []cephadmin.StorageBackend `json:"storage_backends"`
			}{StorageBackends: []cephadmin.StorageBackend{{Name: "beast"}}},
		},
		buckets: []cephadmin.Bucket{{
			Bucket:    "bucket-a",
			Zonegroup: "realm-a",
			Owner:     "user-a",
		}},
		bucketInfoByName: map[string]cephadmin.Bucket{
			"bucket-a": {
				Bucket: "bucket-a",
				BucketQuota: cephadmin.QuotaSpec{
					Enabled: new(true),
				},
			},
		},
	}

	buckets, err := NewBucketSource(&Client{admin: admin}).ListBuckets(context.Background())
	if err != nil {
		t.Fatalf("ListBuckets failed: %v", err)
	}

	if buckets[0].QuotaMaxSizeBytes != nil || buckets[0].QuotaMaxObjects != nil {
		t.Fatalf("expected omitted unlimited bucket quota metrics, got %+v", buckets[0])
	}
}

func TestBucketSourceOmitsQuotaOnBucketInfoErrors(t *testing.T) {
	size := uint64(12)
	objects := uint64(3)
	admin := &fakeAdmin{
		info: cephadmin.Info{
			InfoSpec: struct {
				StorageBackends []cephadmin.StorageBackend `json:"storage_backends"`
			}{StorageBackends: []cephadmin.StorageBackend{{Name: "beast"}}},
		},
		buckets: []cephadmin.Bucket{{
			Bucket:    "bucket-a",
			Zonegroup: "realm-a",
			Owner:     "user-a",
			Tenant:    "tenant-a",
			Usage: cephadmin.BucketUsage{
				RgwMain: cephadmin.RgwUsage{Size: &size, NumObjects: &objects},
			},
		}},
		bucketInfoErr: errors.New("boom"),
	}

	buckets, err := NewBucketSource(&Client{admin: admin}).ListBuckets(context.Background())
	if err != nil {
		t.Fatalf("ListBuckets failed: %v", err)
	}
	if len(buckets) != 1 {
		t.Fatalf("unexpected bucket count: %d", len(buckets))
	}
	if buckets[0].UsageBytes != 12 || buckets[0].Objects != 3 || buckets[0].Zonegroup != "realm-a" || buckets[0].User != "user-a" || buckets[0].Tenant != "tenant-a" {
		t.Fatalf("expected bucket usage and identity to survive quota failure, got %+v", buckets[0])
	}
	if buckets[0].QuotaEnabled != nil || buckets[0].QuotaMaxSizeBytes != nil || buckets[0].QuotaMaxObjects != nil {
		t.Fatalf("expected omitted bucket quota on info errors, got %+v", buckets[0])
	}
}

func TestUserSourceOmitsUnlimitedQuotaValues(t *testing.T) {
	userIDs := []string{"user-a"}
	admin := &fakeAdmin{
		info: cephadmin.Info{
			InfoSpec: struct {
				StorageBackends []cephadmin.StorageBackend `json:"storage_backends"`
			}{StorageBackends: []cephadmin.StorageBackend{{Name: "beast"}}},
		},
		userIDs: &userIDs,
		users: map[string]cephadmin.User{
			"user-a": {ID: "user-a"},
		},
		quotaByUID: map[string]cephadmin.QuotaSpec{
			"user-a": {Enabled: new(true)},
		},
	}

	users, err := NewUserSource(&Client{admin: admin}).ListUsers(context.Background(), nil)
	if err != nil {
		t.Fatalf("ListUsers failed: %v", err)
	}

	if users[0].QuotaMaxSizeBytes != nil || users[0].QuotaMaxObjects != nil {
		t.Fatalf("expected omitted unlimited user quota metrics, got %+v", users[0])
	}
	if users[0].QuotaEnabled == nil || !*users[0].QuotaEnabled {
		t.Fatalf("expected known user quota enabled metric, got %+v", users[0])
	}
}

func TestUserSourceOmitsQuotaOnQuotaErrors(t *testing.T) {
	userIDs := []string{"user-a"}
	size := uint64(11)
	objects := uint64(5)
	admin := &fakeAdmin{
		info: cephadmin.Info{
			InfoSpec: struct {
				StorageBackends []cephadmin.StorageBackend `json:"storage_backends"`
			}{StorageBackends: []cephadmin.StorageBackend{{Name: "beast"}}},
		},
		userIDs: &userIDs,
		users: map[string]cephadmin.User{
			"user-a": {
				ID:   "user-a",
				Stat: cephadmin.UserStat{Size: &size, NumObjects: &objects},
			},
		},
		quotaErr: errors.New("boom"),
	}

	inputBuckets := []rgw.Bucket{
		{User: "user-a", Zonegroup: "realm-a"},
		{User: "user-a", Zonegroup: "realm-a"},
	}
	users, err := NewUserSource(&Client{admin: admin}).ListUsers(context.Background(), inputBuckets)
	if err != nil {
		t.Fatalf("ListUsers failed: %v", err)
	}

	if users[0].UsageBytes != 11 || users[0].Objects != 5 || users[0].BucketCount != 2 || users[0].Zonegroup != "realm-a" {
		t.Fatalf("expected user usage and bucket count to survive quota failure, got %+v", users[0])
	}
	if users[0].QuotaEnabled != nil || users[0].QuotaMaxSizeBytes != nil || users[0].QuotaMaxObjects != nil {
		t.Fatalf("expected omitted user quota metrics on quota errors, got %+v", users[0])
	}
}
