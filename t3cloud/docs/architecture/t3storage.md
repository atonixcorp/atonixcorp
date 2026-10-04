# T3Storage Architecture

T3Storage is the distributed object storage engine powering T3Cloud. It provides S3‑compatible APIs, erasure‑coded durability, multi‑tenant isolation, and high‑throughput streaming for modern cloud workloads.

---

## 1. Core Design Principles

- **S3 compatibility** — seamless integration with AWS CLI, SDKs, and tooling  
- **Horizontal scalability** — add nodes, racks, or regions without redesign  
- **Durability** — erasure coding, replication, and rack‑aware placement  
- **Performance** — NVMe acceleration, parallel multipart uploads  
- **Fault tolerance** — automatic rebuild, self‑healing, and topology awareness  

---

## 2. System Components

### 2.1 Gateway Layer (S3 API)
- Handles HTTP/S3 requests  
- Validates SigV4 signatures  
- Streams data to/from storage nodes  
- Enforces bucket policies and quotas  

### 2.2 Metadata Cluster
- Raft‑based consensus  
- Stores object index, bucket metadata, placement maps  
- Ensures strong consistency  

### 2.3 Storage Nodes
- Store erasure‑coded fragments  
- Maintain local metadata journals  
- Run background repair workers  
- Report health and capacity  

### 2.4 TinMap (Topology Engine)
- Maps disks → nodes → racks → regions  
- Ensures fragments are placed across failure domains  

---

## 3. Data Model

Each object consists of:

- Object metadata (key, size, checksum, timestamps)  
- Fragment list (locations, indexes)  
- Version ID (optional)  
- Bucket + tenant association  

Fragments are stored on multiple nodes according to EC profile.

---

## 4. Durability Model

Supported profiles:

- **EC 4+2** — medium objects  
- **EC 6+3** — large objects  
- **Replication 3×** — small objects  

Durability is guaranteed across:

- Disk failures  
- Node failures  
- Rack failures  
- Region failures (if replication enabled)  

---

## 5. Write Path

1. Gateway receives PUT  
2. Metadata cluster selects placement  
3. Gateway streams fragments to nodes  
4. Nodes write fragments + local metadata  
5. Metadata cluster commits object entry  
6. Gateway returns success  

Atomicity is guaranteed.

---

## 6. Read Path

1. Gateway resolves fragment locations  
2. Nodes stream fragments  
3. Gateway reconstructs object (if EC)  
4. Data returned to client  

Reads are parallelized for speed.

---

## 7. Rebuild & Self‑Healing

Triggered when:

- Node offline  
- Disk failure  
- Rack isolation  

Rebuild steps:

1. Identify missing fragments  
2. Reconstruct using surviving fragments  
3. Write to new node  
4. Update metadata  

---

## 8. Developer Notes

- All metadata operations must be strongly consistent  
- Fragment writes must be idempotent  
- Multipart uploads must follow AWS semantics  
- Rebuild operations must be throttled  
- TinMap updates must propagate cluster‑wide  

T3Storage is the backbone of T3Cloud.
