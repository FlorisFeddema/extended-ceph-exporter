# Copilot Instructions for `extended-ceph-exporter`

## Build, test, and lint commands

Run from repository root.

```bash
# Format and formatting check
gofmt -w .
gofmt -l .

# Go tests
go test ./...
go test -cover ./...
go test -race ./...   # when changing shared state/caching/concurrency

# Run a single test (name and package)
go test ./internal/collector/rgw -run TestServiceSnapshotCachesResults

# Build and static checks
go build ./...
go vet ./...
golangci-lint run

# Helm validation and rendering
helm lint charts/extended-ceph-exporter
helm template extended-ceph-exporter charts/extended-ceph-exporter
```

## High-level architecture

- `cmd/extended-ceph-exporter/main.go` wires configuration, logging, Prometheus registry, collectors, and HTTP server.
- `internal/config` defines CLI flag + environment-variable config resolution (`EXTENDED_CEPH_EXPORTER_*`), with flags overriding env defaults.
- `internal/rgwclient` wraps `go-ceph` RGW Admin Ops calls and maps API responses into exporter domain models.
- `internal/collector/rgw` is split into:
  - `Service`: shared snapshot/cache layer (single cache used by both collectors per scrape window).
  - `BucketsCollector` and `UsersCollector`: Prometheus collectors reading from that shared snapshot.
  - Optional service self-metrics (`extended_ceph_exporter_rgw_*`) registered when enabled.
- `internal/exporter` provides HTTP handlers:
  - `/metrics` via `promhttp`
  - `/healthz`
  - root info endpoint (`/`)
- Helm chart (`charts/extended-ceph-exporter`) is a first-class deployment surface:
  - injects runtime config via env vars
  - supports credential sourcing from existing/generated Secrets
  - optional `ServiceMonitor`
  - optional Rook `CephObjectStoreUser`
  - ships Grafana dashboard `ConfigMap`

## Key conventions in this repository

- **Scope boundary**: this exporter extends (does not replace) the default Ceph exporter; avoid adding metrics already covered by standard Ceph/Rook exporters.
- **RGW label normalization**:
  - `realm` and `store` are required dimensions for RGW metrics.
  - Missing values map to `"unknown"`.
  - Conflicting multi-source values map to `"mixed"`.
- **Quota metric semantics**:
  - `*_quota_enabled` may exist independently.
  - `*_quota_max_*` metrics are omitted when quota is unlimited, disabled, or unavailable from API responses.
- **Error exposure rule on `/metrics`**:
  - internal collector errors are logged, but HTTP responses mask details with generic `500 internal server error` (no sensitive backend messages in body).
- **Collector design pattern**:
  - keep metric-domain collectors thin; place expensive RGW enumeration behind the shared cache service (`RGWCacheTTL`).
  - preserve backward-compatible metric names/labels unless a migration is intentional.
- **Helm credential precedence**:
  - credentials are expected from Kubernetes Secrets (existing secret, chart-created secret, or Rook-generated secret naming).
  - avoid passing RGW secrets in container args or committed values.
