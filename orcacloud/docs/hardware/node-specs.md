# T3Cloud Hardware — Node Specifications

This document defines the hardware specifications for all node types in T3Cloud. Each node class is optimized for its workload: object storage, block storage, filesystem operations, metadata consistency, and archive retention.

---

## 1. Node Classes

T3Cloud uses five primary node types:

- **Storage Nodes (T3Storage)** — object fragments, erasure coding, repair workers  
- **Block Nodes (T3Block)** — NVMe‑optimized block volumes  
- **Filesystem Nodes (T3FS)** — balanced compute/storage for POSIX workloads  
- **Vault Nodes (T3Vault)** — high‑density cold storage  
- **Metadata Nodes** — Raft consensus, global metadata, TinMap distribution  
- **Gateway Nodes** — API endpoints (S3, NVMe‑oF, iSCSI, NFS, SMB, Vault API)

---

## 2. Storage Nodes (T3Storage)

### Purpose
Store erasure‑coded object fragments and run repair workers.

### Hardware Profile
- **Form Factor:** 2U or 4U storage server  
- **CPU:** Dual‑socket x86_64 or ARM64 (16–32 cores)  
- **Memory:** 256–512 GB DDR4/DDR5 ECC  
- **Storage:**  
  - 12–24 × HDD (10–16 TB)  
  - 2 × NVMe SSD (1.6–3.2 TB) for metadata + small object tier  
- **Networking:** Dual 25/100 GbE (optional RDMA)  
- **Power:** Dual redundant PSUs (1200–1600 W)  
- **Cooling:** High‑airflow fans  

---

## 3. Block Nodes (T3Block)

### Purpose
Provide low‑latency block volumes via NVMe‑oF or iSCSI.

### Hardware Profile
- **Form Factor:** 1U or 2U performance server  
- **CPU:** 24–32 cores  
- **Memory:** 128–256 GB ECC  
- **Storage:**  
  - 8–16 × NVMe SSD (high endurance)  
- **Networking:** Dual 100 GbE  
- **Power:** Dual PSUs  

---

## 4. Filesystem Nodes (T3FS)

### Purpose
Serve NFS/SMB workloads, directory trees, inode metadata.

### Hardware Profile
- **Form Factor:** 1U or 2U  
- **CPU:** 16–24 cores  
- **Memory:** 128–256 GB ECC  
- **Storage:**  
  - 4–8 × NVMe SSD  
  - Optional HDD tier for large files  
- **Networking:** Dual 25/40/100 GbE  

---

## 5. Vault Nodes (T3Vault)

### Purpose
Cold/archive storage with multi‑tier retention.

### Hardware Profile
- **Form Factor:** 4U high‑density storage  
- **CPU:** 8–16 cores  
- **Memory:** 64–128 GB ECC  
- **Storage:**  
  - 24–36 × HDD (16–20 TB)  
  - 1 × NVMe SSD for metadata  
- **Networking:** Dual 10/25 GbE  

---

## 6. Metadata Nodes

### Purpose
Maintain global metadata, Raft consensus, TinMap topology.

### Hardware Profile
- **Form Factor:** 1U performance server  
- **CPU:** 24–32 cores  
- **Memory:** 128–256 GB ECC  
- **Storage:**  
  - 4–8 × NVMe SSD (high endurance)  
- **Networking:** Dual 25/40/100 GbE  

---

## 7. Gateway Nodes

### Purpose
Expose APIs: S3, NVMe‑oF, iSCSI, NFS, SMB, Vault.

### Hardware Profile
- **Form Factor:** 1U compute server  
- **CPU:** 16–24 cores  
- **Memory:** 64–128 GB ECC  
- **Storage:** 1–2 × NVMe SSD  
- **Networking:** Dual 25/40 GbE  

---

## 8. Summary Table

| Node Type       | CPU Cores | RAM        | Storage Layout                     | Network         |
|-----------------|-----------|------------|------------------------------------|------------------|
| Storage Node    | 16–32     | 256–512 GB | HDD tier + NVMe tier               | 25/100 GbE       |
| Block Node      | 24–32     | 128–256 GB | NVMe‑only                          | 100 GbE          |
| Filesystem Node | 16–24     | 128–256 GB | NVMe + optional HDD                | 25/40/100 GbE    |
| Vault Node      | 8–16      | 64–128 GB  | High‑density HDD                   | 10/25 GbE        |
| Metadata Node   | 24–32     | 128–256 GB | NVMe‑only                          | 25/40/100 GbE    |
| Gateway Node    | 16–24     | 64–128 GB  | NVMe‑only                          | 25/40 GbE        |
