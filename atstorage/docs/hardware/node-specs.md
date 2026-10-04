# Storage Tin Hardware — Node Specifications

This document defines the hardware specifications for all node types in the Storage Tin platform. Nodes are divided into three categories:

- Storage Nodes (TinCore data nodes)
- Metadata Nodes (TinCore metadata cluster)
- Gateway Nodes (TinS3 API gateways)

---

## 1. Storage Nodes (TinCore Data Nodes)

Storage nodes provide the raw capacity and durability for object fragments. They are optimized for high‑density HDD storage with NVMe acceleration.

### Hardware Profile

- **Form Factor:** 2U or 4U storage server
- **CPU:** Dual-socket x86_64 or ARM64, 16–32 cores total
- **Memory:** 256–512 GB DDR4/DDR5 ECC
- **Storage:**
  - 12–24 × HDD bays (10–16 TB drives)
  - 2 × NVMe SSDs (1.6–3.2 TB) for metadata and small-object tier
- **Networking:**
  - Dual 25 GbE or 100 GbE NICs
  - Optional RDMA (RoCEv2)
- **Power:** Dual redundant PSUs (1200–1600 W)
- **Cooling:** High‑airflow fans with thermal monitoring

### Responsibilities

- Store erasure-coded fragments
- Run repair workers
- Maintain local metadata journals
- Participate in TinCore cluster membership

---

## 2. Metadata Nodes (TinCore Metadata Cluster)

Metadata nodes maintain global indexes, placement maps, and cluster configuration. They require high IOPS and strong consistency.

### Hardware Profile

- **Form Factor:** 1U performance server
- **CPU:** 24–32 cores
- **Memory:** 128–256 GB ECC
- **Storage:**
  - 4–8 × NVMe SSDs (high endurance)
  - No HDDs
- **Networking:** Dual 25/40/100 GbE
- **Power:** Dual redundant PSUs

### Responsibilities

- Maintain global object index
- Store bucket and tenant metadata
- Run Raft consensus
- Serve TinCore placement queries

---

## 3. Gateway Nodes (TinS3 API Gateways)

Gateway nodes handle S3 API traffic and route requests to TinCore.

### Hardware Profile

- **Form Factor:** 1U compute server
- **CPU:** 16–24 cores
- **Memory:** 64–128 GB ECC
- **Storage:** 1–2 × NVMe SSDs (OS + logs)
- **Networking:** Dual 25/40 GbE

### Responsibilities

- Serve S3 API requests
- Validate SigV4 signatures
- Enforce quotas and rate limits
- Stream data to/from TinCore nodes

---

## Summary Table

| Node Type       | CPU Cores | RAM        | Storage Layout                     | Network         |
|-----------------|-----------|------------|------------------------------------|------------------|
| Storage Node    | 16–32     | 256–512 GB | HDD tier + NVMe tier               | 25/100 GbE       |
| Metadata Node   | 24–32     | 128–256 GB | NVMe-only                          | 25/40/100 GbE    |
| Gateway Node    | 16–24     | 64–128 GB  | NVMe-only                          | 25/40 GbE        |

