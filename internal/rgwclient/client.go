package rgwclient

import (
	"context"
	"net/http"
	"sync"

	cephadmin "github.com/ceph/go-ceph/rgw/admin"

	"github.com/FlorisFeddema/extended-ceph-exporter/internal/config"
)

type adminAPI interface {
	GetInfo(ctx context.Context) (cephadmin.Info, error)
	ListBucketsWithStat(ctx context.Context) ([]cephadmin.Bucket, error)
	GetBucketInfo(ctx context.Context, bucket cephadmin.Bucket) (cephadmin.Bucket, error)
	GetUsers(ctx context.Context) (*[]string, error)
	GetUser(ctx context.Context, user cephadmin.User) (cephadmin.User, error)
	GetUserQuota(ctx context.Context, quota cephadmin.QuotaSpec) (cephadmin.QuotaSpec, error)
	ListUsersBucketsWithStat(ctx context.Context, uid string) ([]cephadmin.Bucket, error)
}

type Client struct {
	admin  adminAPI
	site   *siteMetadata
	siteMu sync.Mutex
}

func New(cfg config.Config) (*Client, error) {
	adminAPI, err := cephadmin.New(
		cfg.RGWAdminEndpoint,
		cfg.RGWAccessKey,
		cfg.RGWSecretKey,
		&http.Client{Timeout: cfg.RequestTimeout},
	)
	if err != nil {
		return nil, err
	}

	return &Client{admin: adminAPI}, nil
}
