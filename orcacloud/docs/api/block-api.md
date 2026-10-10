# T3Block API Reference

T3Block provides high‑performance block storage via NVMe‑oF and iSCSI. This API allows developers to create volumes, manage snapshots, configure replication, and attach volumes to compute nodes.

---

## 1. API Endpoints

Base URL: http(s)://0.0.0.0:9001/block


Protocols:

- NVMe‑oF  
- iSCSI  
- HTTP/JSON API  
- gRPC API  

---

## 2. Volume Operations

### Create Volume

POST /v1/volumes
{
"name": "db-volume",
"size_gb": 200,
"replication": "sync"
}

### Delete Volume
DELETE /v1/volumes/{volume_id}

### List Volumes
GET /v1/volumes

### Get Volume Info
GET /v1/volumes/{volume_id}
---

## 3. Snapshot Operations

### Create Snapshot
POST /v1/volumes/{volume_id}/snapshots

### List Snapshots
GET /v1/volumes/{volume_id}/snapshots


### Delete Snapshot
DELETE /v1/snapshots/{snapshot_id}

---

## 4. Replication Operations

### Enable Replication
POST /v1/volumes/{volume_id}/replication
{
"mode": "sync",
"target": "region-b"
}


### Disable Replication
DELETE /v1/volumes/{volume_id}/replication


## 5. Attach / Detach

### Attach Volume
POST /v1/volumes/{volume_id}/attach
{
"node": "compute01"
}


### Detach Volume
POST /v1/volumes/{volume_id}/detach


## 6. Developer Notes

- Snapshots are instant (COW)  
- Replication uses journal‑based delta transfer  
- NVMe‑oF recommended for high‑performance workloads  
- iSCSI recommended for legacy environments  

T3Block powers VM, database, and HPC workloads.

