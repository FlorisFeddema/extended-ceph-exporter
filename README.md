# extended-ceph-exporter

`extended-ceph-exporter` is a Go-based Prometheus exporter that fills in gaps left by the default Ceph exporter shipped with Rook.

## Why This Exists

The Ceph exporter we run today does not expose all metrics we need. The first focus is RGW visibility, especially:

- RGW bucket metrics
- RGW user metrics

Later, the scope may expand to other missing Ceph metrics, such as RBD volume sizes and related storage details.

This project is not a replacement for the existing Ceph exporter. It is an extension that is meant to run alongside the standard exporter and provide additional metrics that are currently missing.

## Project Goals

- Close observability gaps in the default Ceph exporter
- Stay efficient in CPU and memory usage
- Fit naturally into a Kubernetes-based Rook deployment
- Expose Prometheus-compatible metrics without duplicating what the main Ceph exporter already provides

## Target Environment

The primary deployment target is Kubernetes. The exporter is expected to integrate tightly with Rook and be packaged for deployment with Helm.

The project should also include:

- Prometheus Operator integration, including a `ServiceMonitor`
- One or more Grafana dashboards for the new metrics

## Implementation Direction

- Built in Golang
- Designed for low resource usage
- Intended to complement, not replace, the existing Ceph exporter
- Focused on practical metrics coverage for real cluster operations
- Allowed to use a short-lived cache for slower-moving metrics to reduce load on RGW and Ceph APIs

As the implementation grows, this repository should remain centered on that scope: extend Ceph observability where the default exporter falls short, while staying simple to deploy and efficient to run.

## Initial Roadmap

1. Establish the basic exporter skeleton in Go with Prometheus metric exposition.
2. Add the first RGW collectors for bucket-level and user-level metrics.
3. Package the exporter for Kubernetes deployment with a Helm chart.
4. Add Rook-friendly configuration and service discovery defaults.
5. Integrate with the Prometheus Operator using a `ServiceMonitor`.
6. Ship one or more Grafana dashboards for the new metrics.
7. Evaluate additional gap-filling collectors, such as RBD volume size metrics.

The roadmap should stay incremental: get useful RGW coverage working first, then expand into adjacent missing metrics only where the default exporter still falls short.

## Architecture Sketch

The expected shape of the project is:

- A small Go exporter process exposing `/metrics`
- One or more Ceph-aware collectors focused on missing metric domains
- An RGW integration layer for bucket and user data collection
- Kubernetes deployment assets, primarily a Helm chart
- Prometheus Operator integration through a `ServiceMonitor`
- Grafana dashboards built around the additional exported metrics

At a high level, the flow is:

1. The exporter connects to Ceph or RGW-related APIs.
2. It collects only the missing data it is responsible for.
3. It may keep that data in a short-lived in-memory cache for a few seconds when the metrics do not change rapidly.
4. It exposes the current metric set in Prometheus format on `/metrics`.
5. Prometheus scrapes the exporter next to the standard Ceph exporter.
6. Grafana dashboards visualize the extended metric set.

## Design Boundaries

- Do not duplicate metrics already provided well by the default Ceph exporter.
- Prefer predictable, low-overhead collection over broad but expensive scraping.
- A small cache is acceptable when it reduces scrape-time pressure on RGW without making the data meaningfully stale.
- Keep deployment simple for Rook-managed Kubernetes clusters.
- Treat Helm, `ServiceMonitor`, and dashboard support as first-class deliverables, not afterthoughts.

## Development Environment

The repository includes a `.devcontainer` setup for a consistent development environment. It installs Go, `kubectl`, Helm, `jq`, `yq`, `golangci-lint`, `curl`, Node.js, and the OpenAI Codex CLI.

The container is configured to run as the non-root `dev` user, sets `no-new-privileges`, and drops all Linux capabilities to reduce the chance of privilege escalation inside the environment. The `dev` user is created at image build time so the devcontainer CLI can attach reliably. The Codex CLI is installed as a pinned version instead of pulling the latest release at container creation time.

To run the development environment:

1. Run `devcontainer up --workspace-folder .` from the repository root.
2. Start a shell in the container with `devcontainer exec --workspace-folder . zsh`.
3. Wait for the post-create steps to install dependencies and Codex CLI.
4. Authenticate Codex inside the container with `codex --login`.
5. Start working from the repository root inside the container.

Optional environment verification:

```bash
kubectl version --client
helm version --short
jq --version
yq --version
golangci-lint version
curl --version
node --version
npm --version
codex --version
```

Useful commands:

- `go build ./...`
- `go test ./...`
- `golangci-lint run`
- `codex`

## Container Image

The repository includes a production-oriented multi-stage `Dockerfile` for building the exporter as a small Linux container image.

Build the image locally:

```bash
docker build -t extended-ceph-exporter:dev .
```

Run it locally:

```bash
docker run --rm -p 9877:9877 \
  extended-ceph-exporter:dev \
  --listen-address=:9877
```

Run it against RGW:

```bash
docker run --rm -p 9877:9877 \
  extended-ceph-exporter:dev \
  --listen-address=:9877 \
  --rgw-admin-endpoint=https://rgw.example \
  --rgw-access-key=ACCESS_KEY \
  --rgw-secret-key=SECRET_KEY
```

The image runs the statically compiled exporter on port `9877` by default and is intended to be the base image for later Kubernetes and Helm deployment work.

## Helm Chart

A Helm chart is available in [`charts/extended-ceph-exporter`](./charts/extended-ceph-exporter).

Render the default manifests:

```bash
helm template extended-ceph-exporter charts/extended-ceph-exporter
```

Install with an existing secret for RGW credentials:

```bash
helm upgrade --install extended-ceph-exporter charts/extended-ceph-exporter \
  --set rgw.adminEndpoint=https://rgw.example \
  --set rgw.credentials.existingSecret.name=rgw-admin-credentials
```

Enable Prometheus Operator integration:

```bash
helm upgrade --install extended-ceph-exporter charts/extended-ceph-exporter \
  --set serviceMonitor.enabled=true
```

Optionally create a Rook `CephObjectStoreUser` resource:

```bash
helm upgrade --install extended-ceph-exporter charts/extended-ceph-exporter \
  --set rook.clusterNamespace=rook-ceph \
  --set rook.objectStoreUser.enabled=true \
  --set rook.objectStoreUser.store=my-store
```

The chart supports both an optional `ServiceMonitor` and an optional `CephObjectStoreUser` resource. The exporter configuration is injected through environment variables so access keys and secret keys can be sourced from Kubernetes `Secret` objects without passing them as container arguments.

When the object-store user secret is managed or copied under another name in the exporter namespace, override the reference with `rook.objectStoreUser.secretName`:

```bash
helm upgrade --install extended-ceph-exporter charts/extended-ceph-exporter \
  --set rook.objectStoreUser.enabled=true \
  --set rook.objectStoreUser.store=my-store \
  --set rook.objectStoreUser.secretName=my-rgw-credentials
```

If it is not set, the chart uses Rook's generated secret name for the object-store user.

### RGW Admin Permissions

The exporter calls the RGW Admin Ops API. Its credential user needs these read-only Admin Ops capabilities:

- `info=read` for RGW site information
- `users=read` for user details and user quotas
- `buckets=read` for bucket details, statistics, and bucket listings

Grant them with `radosgw-admin`:

```bash
radosgw-admin caps add \
  --uid=extended-ceph-exporter \
  --caps="info=read;users=read;buckets=read"
```

Verify the effective capabilities:

```bash
radosgw-admin user info --uid=extended-ceph-exporter
```

The `rook.objectStoreUser.capabilities` values configure S3 user capabilities; they do not grant RGW Admin Ops access. Do not use `--admin` or `--system` unless broader access is explicitly required.

## Grafana Dashboard

The Helm chart now ships an RGW overview dashboard as a `ConfigMap`. By default it is labeled with `grafana_dashboard: "1"` so common Grafana sidecar patterns can discover and import it automatically.

Render only dashboard manifest:

```bash
helm template extended-ceph-exporter charts/extended-ceph-exporter \
  --show-only templates/grafana-dashboard-configmap.yaml
```

Disable dashboard deployment:

```bash
helm upgrade --install extended-ceph-exporter charts/extended-ceph-exporter \
  --set grafanaDashboard.enabled=false
```

Override dashboard labels:

```bash
helm upgrade --install extended-ceph-exporter charts/extended-ceph-exporter \
  --set grafanaDashboard.labels.grafana_dashboard=1
```

## Release Automation

GitHub Actions release workflow lives in `.github/workflows/release.yaml`.

- Push tag like `0.0.1` to publish multi-arch Docker image to `ghcr.io/florisfeddema/extended-ceph-exporter` with `0.0.1` and `latest` tags.
- Same workflow packages chart from `charts/extended-ceph-exporter` and publishes it as OCI artifact to `ghcr.io/florisfeddema/charts/extended-ceph-exporter:0.0.1`.
- Workflow fails if git tag and chart `version` in `Chart.yaml` do not match.
