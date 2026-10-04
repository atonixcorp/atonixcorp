# T3Block Architecture

T3Block is the high‑performance block storage engine of T3Cloud. It provides NVMe‑oF and iSCSI targets, snapshots, replication, and low‑latency volumes for VMs, databases, and HPC workloads.

---

## 1. Core Design Principles

- **Low latency** — NVMe‑optimized data paths  
- **High throughput** — parallel I/O scheduling  
- **Consistency** — atomic writes and crash‑safe metadata  
- **Resilience** — replication and failover  
- **Flexibility** — multiple protocols (NVMe‑oF, iSCSI)  

---

## 2. System Components

### 2.1 Block Nodes
- NVMe SSDs for primary storage  
- Local metadata store  
- Volume manager  
- Snapshot engine  

### 2.2 Protocol Layer
- **NVMe‑oF** target  
- **iSCSI** target  
- Multi‑queue I/O scheduling  

### 2.3 Metadata Layer
- Volume metadata  
- Snapshot metadata  
- Replication state  

### 2.4 Replication Engine
- Synchronous or asynchronous  
- Journal‑based replication  
- Automatic failover  

---

## 3. Volume Model

A volume contains:

- Logical block address space  
- Backing NVMe device  
- Snapshot chain  
- Replication configuration  

Snapshots are copy‑on‑write.

---

## 4. Write Path

1. Host sends write  
2. Protocol layer receives block  
3. Volume manager writes to NVMe  
4. Metadata updated  
5. Replication (if enabled)  

Writes are atomic.

---

## 5. Snapshot Model

Snapshots support:

- Instant creation  
- Copy‑on‑write  
- Cloning  
- Rollback  

Snapshots are stored as metadata + delta blocks.

---

## 6. Replication Model

Modes:

- **Sync** — zero‑RPO  
- **Async** — low‑latency writes  

Replication uses:

- Journaling  
- Delta transfer  
- Automatic resync  

---

## 7. Developer Notes

- NVMe I/O paths must be zero‑copy  
- Metadata must be crash‑safe  
- Replication must handle network partitions  
- Snapshots must not degrade performance  

T3Block powers compute workloads.
