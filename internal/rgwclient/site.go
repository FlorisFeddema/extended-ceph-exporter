package rgwclient

import (
	"context"

	cephadmin "github.com/ceph/go-ceph/rgw/admin"
)

type siteMetadata struct {
	store string
}

const (
	unknownLabelValue = "unknown"
	mixedLabelValue   = "mixed"
)

func (c *Client) storeLabel(ctx context.Context) string {
	c.siteMu.Lock()
	defer c.siteMu.Unlock()

	if c.site != nil {
		return c.site.store
	}

	info, err := c.admin.GetInfo(ctx)
	if err != nil {
		return unknownLabelValue
	}

	c.site = &siteMetadata{store: normalizeStore(info.InfoSpec.StorageBackends)}
	return c.site.store
}

func normalizeStore(backends []cephadmin.StorageBackend) string {
	if len(backends) == 0 {
		return unknownLabelValue
	}

	name := ""
	for _, backend := range backends {
		if backend.Name == "" {
			continue
		}

		if name == "" {
			name = backend.Name
			continue
		}

		if name != backend.Name {
			return mixedLabelValue
		}
	}

	if name == "" {
		return unknownLabelValue
	}

	return name
}
