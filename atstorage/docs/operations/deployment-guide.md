# Storage Tin Deployment Guide

This guide describes how to deploy the Storage Tin platform across lab, staging, and production environments. It covers TinCore, TinS3, TinControl, networking, and cluster bootstrap procedures.

---

## 1. Deployment Architecture

Storage Tin is deployed in three layers:

1. **TinCore Layer**  
   Distributed object storage nodes (data + metadata).

2. **TinS3 Layer**  
   Stateless S3 API gateways.

3. **TinControl Layer**  
   Control plane, billing, tenant management, and UI.

Each layer scales independently.

---

## 2. Prerequisites

### Hardware
- Storage nodes installed and racked
- Metadata nodes online
- Gateway nodes online
- ToR switches configured (25/100 GbE)
- PDUs connected (A+B power)

### Software
- Debian 13 or Ubuntu 24.04 LTS
- Go 1.22+ and Rust toolchain
- Docker or containerd
- Kubernetes (optional)
- Prometheus + Grafana stack

### Network
- Leaf-spine fabric operational
- VLANs configured for:
  - Storage traffic
  - API traffic
  - Management traffic

---

## 3. Deployment Steps

### Step 1 — Bootstrap Metadata Cluster
1. Install TinCore metadata service.
2. Initialize Raft cluster:
3. Verify consensus health.

### Step 2 — Deploy TinCore Storage Nodes
1. Install TinCore node service.
2. Register each node:
3. Start shard store and repair workers.
4. Validate placement map (TinMap).

### Step 3 — Deploy TinS3 Gateways
1. Install gateway service.
2. Configure routing to TinCore metadata cluster.
3. Start S3 API server.
4. Validate:

### Step 4 — Deploy TinControl
1. Deploy TinControl API.
2. Deploy TinControl Web UI.
3. Configure billing engine.
4. Create initial tenant:

---

## 4. Post-Deployment Validation

- S3 PUT/GET tests
- Multipart upload tests
- Erasure coding placement tests
- Node failure simulation
- Rack failure simulation
- Cross-region replication tests (if enabled)

---

## 5. Production Hardening

- Enable TLS everywhere
- Configure IAM-style policies
- Enable rate limiting
- Configure monitoring and alerting
- Enable audit logging

---

## 6. Deployment Modes

### Lab Mode
- Single rack
- Minimal nodes
- No replication

### Staging Mode
- Multi-rack
- Full TinMap
- Replication enabled

### Production Mode
- Multi-region
- Erasure coding
- Cross-region replication
- Billing + quotas enabled

---

Storage Tin deployments are predictable, modular, and horizontally scalable.
