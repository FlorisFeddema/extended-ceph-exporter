# RGW Metrics Design v1

## Purpose

This document defines the first metric set for `extended-ceph-exporter`.

Version 1 focuses on two collectors:

- `rgw_buckets`
- `rgw_users`

The goal is to expose RGW bucket and user inventory, usage, and quota state that is not cleanly available through the default Ceph and Rook Prometheus surfaces.

## Scope

This exporter is intended for a single Ceph cluster that may contain multiple RGW realms and stores. Metrics must therefore include both `realm` and `store` as labels.

This metric set should complement the default Ceph exporter, not duplicate standard Ceph daemon or cluster metrics.

## Label Rules

Common labels:

- `realm`
- `store`
- `tenant` when the RGW deployment uses tenants

Bucket collector labels:

- `bucket`
- `user`

User collector labels:

- `user`

The following must not be exposed as labels:

- access keys
- email addresses
- display names
- endpoint names
- pod names
- store names

## RGW Bucket Metrics

| Metric | Type | Labels | Description |
| --- | --- | --- | --- |
| `extended_ceph_rgw_bucket_info` | gauge | `realm`, `store`, `bucket`, `user`, `tenant` | Static presence metric with value `1` |
| `extended_ceph_rgw_bucket_usage_bytes` | gauge | `realm`, `store`, `bucket`, `user`, `tenant` | Current bucket size in bytes |
| `extended_ceph_rgw_bucket_objects` | gauge | `realm`, `store`, `bucket`, `user`, `tenant` | Current object count |
| `extended_ceph_rgw_bucket_quota_enabled` | gauge | `realm`, `store`, `bucket`, `user`, `tenant` | `1` when bucket quota is enabled, otherwise `0` |
| `extended_ceph_rgw_bucket_quota_max_size_bytes` | gauge | `realm`, `store`, `bucket`, `user`, `tenant` | Configured maximum bucket size |
| `extended_ceph_rgw_bucket_quota_max_objects` | gauge | `realm`, `store`, `bucket`, `user`, `tenant` | Configured maximum object count |

## RGW User Metrics

| Metric | Type | Labels | Description |
| --- | --- | --- | --- |
| `extended_ceph_rgw_user_info` | gauge | `realm`, `store`, `user`, `tenant` | Static presence metric with value `1` |
| `extended_ceph_rgw_user_usage_bytes` | gauge | `realm`, `store`, `user`, `tenant` | Total bytes used across user-owned buckets |
| `extended_ceph_rgw_user_objects` | gauge | `realm`, `store`, `user`, `tenant` | Total objects across user-owned buckets |
| `extended_ceph_rgw_user_bucket_count` | gauge | `realm`, `store`, `user`, `tenant` | Number of buckets owned by the user |
| `extended_ceph_rgw_user_quota_enabled` | gauge | `realm`, `store`, `user`, `tenant` | `1` when user quota is enabled, otherwise `0` |
| `extended_ceph_rgw_user_quota_max_size_bytes` | gauge | `realm`, `store`, `user`, `tenant` | Configured maximum user quota size |
| `extended_ceph_rgw_user_quota_max_objects` | gauge | `realm`, `store`, `user`, `tenant` | Configured maximum user quota object count |
| `extended_ceph_rgw_user_suspended` | gauge | `realm`, `store`, `user`, `tenant` | `1` when the user is suspended, otherwise `0` |
| `extended_ceph_rgw_user_max_buckets` | gauge | `realm`, `store`, `user`, `tenant` | Configured maximum number of buckets |

## Implementation Notes

- All v1 metrics are gauges because they represent current RGW admin or usage state.
- Expensive RGW enumeration should be hidden behind a short-lived in-memory cache.
- Unlimited quotas must be handled consistently. In v1, `quota_enabled` remains a separate metric, while `quota_max_*` metrics are omitted when the quota is unlimited, disabled, or not returned by the API.
- Derived ratios such as quota usage percent should not be exported as first-class metrics. They belong in PromQL and Grafana.

## Non-Goals for v1

The following are explicitly out of scope for the first version:

- per-request or per-operation metrics
- bucket tags as labels
- access key or subuser metrics
- RBD metrics
- bucket rate-limit metrics
- bucket sharding and versioning metrics

These can be revisited after the first RGW bucket and user collectors are stable in a real cluster.
