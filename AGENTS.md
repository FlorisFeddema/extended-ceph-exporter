# Repository Guidelines

## Project Scope

`extended-ceph-exporter` is a Go Prometheus exporter that complements, rather than replaces, the Ceph exporter shipped with Rook. Keep changes focused on missing Ceph observability, starting with RGW bucket and user metrics. Avoid duplicating metrics already supplied by the standard exporter.

The primary deployment target is Kubernetes with Rook. Helm, Prometheus Operator integration, and Grafana dashboard support are first-class project concerns.

## Project Structure

- `cmd/extended-ceph-exporter/` contains the executable entrypoint and process wiring.
- `internal/config/` owns CLI and environment configuration.
- `internal/rgwclient/` contains RGW Admin API transport, authentication, and response models.
- `internal/collector/rgw/` contains RGW metric collection and metric definitions.
- `internal/exporter/` owns Prometheus registry and HTTP exposition wiring.
- `charts/extended-ceph-exporter/` contains the Helm chart, templates, values, and chart tests or fixtures.
- `testdata/` is for reusable sanitized Ceph or RGW response fixtures when they are needed.
- `.devcontainer/` defines the supported development environment.

Keep implementation packages under `internal/`. Add `pkg/` only when a package is deliberately intended for external import. Keep Kubernetes resources in the Helm chart rather than embedding YAML in Go code.

## Development Commands

Run commands from the repository root:

```bash
gofmt -w .
go test ./...
go test -cover ./...
go vet ./...
go build ./...
golangci-lint run
helm lint charts/extended-ceph-exporter
helm template extended-ceph-exporter charts/extended-ceph-exporter
```

Before submitting a change, run formatting, the full Go tests, vet, and the relevant Helm validation. Use `go test -race ./...` when changing concurrency, caching, or shared collector state. Do not require a live Ceph cluster for ordinary unit tests; gate live-cluster tests behind an explicit flag or environment variable.

The production image is built with:

```bash
docker build -t extended-ceph-exporter:dev .
```

Use the `.devcontainer` setup when the local machine does not provide the pinned project tooling. Do not add a new build system or Makefile unless it materially improves the documented workflow.

## Go Style and Design

- Follow `gofmt` and idiomatic Go naming: short lowercase package names, `CamelCase` exported identifiers, and `camelCase` unexported identifiers.
- Keep collectors small and focused on one metric domain.
- Prefer explicit dependency injection over package-global clients or registries.
- Use table-driven tests for parsing, configuration, client behavior, and metric collection.
- Keep scrape-time work bounded. A short-lived in-memory cache is acceptable for slow-changing RGW data, but document its TTL and stale-data behavior.
- Preserve Prometheus naming, label, and error-reporting conventions. Do not introduce high-cardinality labels without an explicit reason.
- Keep `/metrics` behavior and existing metric names backward compatible unless the change explicitly requires a migration.

## Testing

Place tests next to the code they cover in `*_test.go` files. Use deterministic HTTP fixtures or test servers for RGW client tests and sanitized files under `testdata/` for larger payloads. Test both successful collection and failure behavior, including malformed responses, authentication failures, timeouts, and partial metric collection where applicable.

For Helm changes, run `helm lint` and render the chart with representative combinations of RGW credentials, `ServiceMonitor`, and Rook object-store-user settings. Never use real cluster credentials or endpoints in fixtures.

## Kubernetes and Helm

Keep the default chart safe to install without creating credentials. Prefer existing Kubernetes `Secret` references for RGW access and secret keys; do not pass secrets in container arguments or commit them to values files. Preserve the optional `ServiceMonitor` and `CephObjectStoreUser` integrations and document changes to their values.

Use the exporter’s default port and metrics path unless a compatibility change is intentional. Keep manifests compatible with Rook-managed clusters and avoid requiring cluster-wide permissions when namespace-scoped permissions are sufficient.

## Security and Data Handling

Never commit `.env` files, access keys, secret keys, kubeconfig files, cluster credentials, or unsanitized Ceph hostnames and endpoints. Redact credentials from logs, test failures, examples, and pull requests. Validate TLS and authentication settings rather than silently weakening them for convenience.

## Commits and Pull Requests

Use short, imperative commit subjects under roughly 72 characters. Pull requests should include a concise summary, testing performed, relevant issue or context, and sample metric output or rendered manifests when behavior or deployment output changes.
