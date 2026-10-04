# Monitoring and Alerting for Storage Tin

This document describes the monitoring stack, metrics, dashboards, and alerting rules required to operate Storage Tin at scale.

---

## 1. Monitoring Stack

Storage Tin uses:

- **Prometheus** — metrics collection
- **Grafana** — dashboards
- **Loki** — log aggregation
- **Alertmanager** — alert routing

All components integrate with TinCore, TinS3, and TinControl.

---

## 2. Metrics Overview

### TinCore Metrics
- Node health
- Disk health
- Shard store latency
- Erasure coding rebuild rate
- Repair queue depth
- Metadata cluster latency
- Placement map consistency

### TinS3 Metrics
- Request rate (RPS)
- Latency (p50, p95, p99)
- Error rates
- Multipart upload performance
- SigV4 validation failures

### TinControl Metrics
- Tenant usage
- Bucket count
- Billing events
- API latency

---

## 3. Dashboards

### Core Dashboards

1. **Cluster Health Dashboard**
   - Node status
   - Rack status
   - Region status
   - Placement map visualization

2. **TinCore Performance Dashboard**
   - Read/write latency
   - Rebuild activity
   - Disk throughput

3. **TinS3 API Dashboard**
   - S3 request heatmap
   - Error breakdown
   - Gateway CPU/memory

4. **TinControl Dashboard**
   - Tenant usage charts
   - Billing metrics
   - API performance

---

## 4. Alerting Rules

### Critical Alerts
- Metadata cluster quorum lost
- Rack offline
- Node offline > 5 minutes
- Rebuild queue saturation
- S3 error rate > 5%
- Disk failure detected

### Warning Alerts
- High latency (p99 > threshold)
- High CPU/memory usage
- Slow rebuild rate
- Gateway overload

### Informational Alerts
- New tenant created
- Bucket nearing quota
- Region replication lag

---

## 5. Logging

### TinCore Logs
- Shard operations
- Repair operations
- Node membership changes

### TinS3 Logs
- API requests
- Auth failures
- Routing decisions

### TinControl Logs
- Tenant/bucket operations
- Billing events
- Configuration changes

---

## 6. Operational Playbooks

- Node replacement
- Disk replacement
- Rack isolation
- Metadata cluster recovery
- Gateway scaling

---

Monitoring ensures Storage Tin remains healthy, predictable, and performant.
