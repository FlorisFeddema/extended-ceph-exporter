package rgwclient

import (
	"context"
	"errors"
	"testing"

	cephadmin "github.com/ceph/go-ceph/rgw/admin"
)

type fakeAdmin struct {
	info                    cephadmin.Info
	infoErr                 error
	infoCalls               int
	buckets                 []cephadmin.Bucket
	bucketsErr              error
	bucketInfoByName        map[string]cephadmin.Bucket
	bucketInfoErr           error
	userIDs                 *[]string
	userIDsErr              error
	users                   map[string]cephadmin.User
	userErr                 error
	quotaByUID              map[string]cephadmin.QuotaSpec
	quotaErr                error
	userBucketsByUID        map[string][]cephadmin.Bucket
	userBucketsWithStatsErr error
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

func (f *fakeAdmin) ListUsersBucketsWithStat(_ context.Context, uid string) ([]cephadmin.Bucket, error) {
	if f.userBucketsWithStatsErr != nil {
		return nil, f.userBucketsWithStatsErr
	}
	return f.userBucketsByUID[uid], nil
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

func TestBucketSourceListBucketsMapsFields(t *testing.T) {
	enabled := true
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
			Usage: struct {
				RgwMain      cephadmin.RgwUsage `json:"rgw.main"`
				RgwMultimeta cephadmin.RgwUsage `json:"rgw.multimeta"`
			}{
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
					Enabled:    &enabled,
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
	if got.Realm != "realm-a" || got.Store != "beast" || got.UsageBytes != 12 || got.Objects != 3 || got.QuotaEnabled == nil || !*got.QuotaEnabled {
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
	enabled := true
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
				Enabled:    &enabled,
				MaxSizeKb:  &maxSizeKB,
				MaxObjects: &maxObjects,
			},
		},
		userBucketsByUID: map[string][]cephadmin.Bucket{
			"user-a": {
				{Zonegroup: "realm-a"},
				{Zonegroup: "realm-a"},
			},
		},
	}

	source := NewUserSource(&Client{admin: admin})
	users, err := source.ListUsers(context.Background())
	if err != nil {
		t.Fatalf("ListUsers failed: %v", err)
	}

	if len(users) != 1 {
		t.Fatalf("unexpected user count: %d", len(users))
	}
	got := users[0]
	if got.Realm != "realm-a" || got.Store != "beast" || got.BucketCount != 2 || !got.Suspended {
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

	_, err := source.ListUsers(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestBucketSourceOmitsUnlimitedQuotaValues(t *testing.T) {
	enabled := true
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
					Enabled: &enabled,
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
			Usage: struct {
				RgwMain      cephadmin.RgwUsage `json:"rgw.main"`
				RgwMultimeta cephadmin.RgwUsage `json:"rgw.multimeta"`
			}{
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
	if buckets[0].UsageBytes != 12 || buckets[0].Objects != 3 || buckets[0].Realm != "realm-a" || buckets[0].User != "user-a" || buckets[0].Tenant != "tenant-a" {
		t.Fatalf("expected bucket usage and identity to survive quota failure, got %+v", buckets[0])
	}
	if buckets[0].QuotaEnabled != nil || buckets[0].QuotaMaxSizeBytes != nil || buckets[0].QuotaMaxObjects != nil {
		t.Fatalf("expected omitted bucket quota on info errors, got %+v", buckets[0])
	}
}

func TestUserSourceOmitsUnlimitedQuotaValues(t *testing.T) {
	userIDs := []string{"user-a"}
	enabled := true
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
			"user-a": {Enabled: &enabled},
		},
		userBucketsByUID: map[string][]cephadmin.Bucket{
			"user-a": nil,
		},
	}

	users, err := NewUserSource(&Client{admin: admin}).ListUsers(context.Background())
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
		userBucketsByUID: map[string][]cephadmin.Bucket{
			"user-a": {
				{Zonegroup: "realm-a"},
				{Zonegroup: "realm-a"},
			},
		},
	}

	users, err := NewUserSource(&Client{admin: admin}).ListUsers(context.Background())
	if err != nil {
		t.Fatalf("ListUsers failed: %v", err)
	}

	if users[0].UsageBytes != 11 || users[0].Objects != 5 || users[0].BucketCount != 2 || users[0].Realm != "realm-a" {
		t.Fatalf("expected user usage and bucket count to survive quota failure, got %+v", users[0])
	}
	if users[0].QuotaEnabled != nil || users[0].QuotaMaxSizeBytes != nil || users[0].QuotaMaxObjects != nil {
		t.Fatalf("expected omitted user quota metrics on quota errors, got %+v", users[0])
	}
}
