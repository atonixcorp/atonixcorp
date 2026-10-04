# TinCore — Distributed Object Storage Engine

TinCore is the heart of Storage Tin. It manages object storage, metadata, placement, durability, and cluster health.

## Responsibilities

- Object lifecycle (PUT, GET, DELETE, multipart)
- Erasure coding and replication
- Placement across disks, nodes, racks, and regions
- Metadata indexing and versioning
- Self‑healing and rebuild operations
- Cluster membership and health monitoring

## Components

### 1. TinCore Node
Runs on storage servers and manages:
- Shard storage
- Local metadata
- Background repair workers

### 2. Metadata Cluster
Provides:
- Global object index
- Bucket and tenant metadata
- Placement maps (TinMap)
- Consensus (Raft)

### 3. TinMap
Defines:
- Disks → Nodes → Racks → Regions
- Failure domains
- Erasure coding profiles

## Data Model

- **Object** — key, attributes, versions  
- **Fragment** — erasure‑coded shard  
- **Metadata entry** — maps keys to fragments  

## Failure Handling

TinCore detects failures, marks unhealthy components, recalculates placement, and rebuilds missing fragments automatically.

TinCore is designed for petabyte‑scale deployments with billions of objects.
