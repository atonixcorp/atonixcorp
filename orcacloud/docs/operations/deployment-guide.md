# T3Cloud Deployment Guide

This guide explains how to deploy T3Cloud across lab, staging, and production environments. It covers cluster initialization, node registration, gateway setup, and operational validation.

---

## 1. Deployment Architecture

T3Cloud is deployed in three layers:

### **1. T3Storage Layer**
Distributed object storage nodes + metadata cluster.

### **2. T3Block / T3FS / T3Vault Layers**
Block, filesystem, and archive nodes depending on workload.

### **3. Gateway Layer**
API endpoints for S3, NVMe‑oF, iSCSI, NFS, SMB, and Vault APIs.

Each layer scales independently.

---

## 2. Prerequisites

### Hardware
- Storage nodes (HDD + NVMe)
- Block nodes (NVMe‑optimized)
- Filesystem nodes (balanced compute/storage)
- Vault nodes (high‑density HDD)
- Metadata nodes (NVMe‑only)
- Gateway nodes (API endpoints)

### Software
- Debian 13 or Ubuntu 24.04 LTS
- Go 1.22+ / Rust toolchain
- Docker or containerd
- Kubernetes (optional)
- Prometheus + Grafana

### Network
- Leaf‑spine fabric
- VLANs for:
  - Storage traffic
  - API traffic
  - Management traffic

---

## 3. Deployment Steps

### Step 1 — Initialize Metadata Cluster
1. Install metadata service.
2. Form Raft cluster:
3. Validate leader election.

### Step 2 — Deploy Storage Nodes
1. Install T3Storage node service.
2. Register node:
3. Start shard store + repair workers.
4. Validate TinMap distribution.

### Step 3 — Deploy Block / FS / Vault Nodes
- T3Block:
- T3FS:
- T3Vault:

### Step 4 — Deploy Gateways
- S3 gateway
- NVMe‑oF gateway
- iSCSI gateway
- NFS/SMB gateway
- Vault gateway

Validate S3:

---

## 4. Post‑Deployment Validation

- S3 PUT/GET tests
- Multipart upload tests
- Block volume attach tests
- Filesystem mount tests
- Archive/recall tests
- Node failure simulation
- Rack isolation simulation

---

## 5. Production Hardening

- Enable TLS everywhere
- Enable rate limiting
- Configure IAM‑style policies
- Enable audit logging
- Configure monitoring + alerting

---

## 6. Deployment Modes

### Lab
- Single rack
- Minimal nodes
- No replication

### Staging
- Multi‑rack
- Full TinMap
- Replication enabled

### Production
- Multi‑region
- Erasure coding
- Cross‑region replication
- Billing + quotas enabled

---

T3Cloud deployments are modular, predictable, and horizontally scalable.
