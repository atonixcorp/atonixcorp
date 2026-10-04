# Monitoring & Alerting

This document describes how T3Cloud is monitored, what metrics are collected, and how alerts are triggered across object, block, filesystem, and archive layers.

---

## 1. Monitoring Stack

T3Cloud uses:

- **Prometheus** — metrics collection  
- **Grafana** — dashboards  
- **Loki** — log aggregation  
- **Alertmanager** — alert routing  

All engines expose metrics via `/metrics` endpoints.

---

## 2. Metrics Overview

### T3Storage Metrics
- Node health
- Disk health
- Shard store latency
- Erasure coding rebuild rate
- Repair queue depth
- Metadata cluster latency
- Placement map consistency

### T3Block Metrics
- Volume latency (read/write)
- NVMe queue depth
- Snapshot creation time
- Replication lag
- Target CPU/memory

### T3FS Metrics
- Inode operations
- Directory traversal latency
- NFS/SMB request rate
- Metadata server load

### T3Vault Metrics
- Tier transitions
- Recall queue depth
- Archive throughput
- Policy execution time

---

## 3. Dashboards

### Cluster Health Dashboard
- Node status
- Rack status
- Region status
- Placement map visualization

### Storage Performance Dashboard
- Read/write latency
- Rebuild activity
- Disk throughput

### Block Performance Dashboard
- NVMe latency
- Replication lag
- Snapshot metrics

### Filesystem Dashboard
- NFS/SMB operations
- Metadata latency

### Vault Dashboard
- Archive throughput
- Recall performance

---

## 4. Alerting Rules

### Critical Alerts
- Metadata cluster quorum lost
- Rack offline
- Node offline > 5 minutes
- Rebuild queue saturation
- S3 error rate > 5%
- NVMe failure detected

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

### T3Storage Logs
- Shard operations
- Repair operations
- Node membership changes

### T3Block Logs
- Volume attach/detach
- Snapshot creation
- Replication events

### T3FS Logs
- NFS/SMB operations
- Metadata changes

### T3Vault Logs
- Archive transitions
- Recall operations

---

## 6. Operational Playbooks

- Node replacement
- Disk replacement
- Rack isolation
- Metadata cluster recovery
- Gateway scaling

---

Monitoring ensures T3Cloud remains healthy, predictable, and performant.
