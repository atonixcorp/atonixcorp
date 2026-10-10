# Backup & Disaster Recovery (DR)

This document defines T3Cloud’s backup strategy and disaster recovery procedures across object, block, filesystem, and archive engines.

---

## 1. Backup Strategy

### What We Back Up
- Metadata cluster state
- T3Block volume metadata
- T3FS inode + directory metadata
- T3Vault policy metadata
- TinMap topology
- Billing records

### What We Do Not Back Up
- Object data (T3Storage handles durability via EC/replication)
- Block data (replication handles durability)
- Filesystem blocks (replication handles durability)

---

## 2. Backup Methods

### Metadata Cluster Snapshots
- Raft snapshots every 5 minutes
- Stored in NVMe tier
- Replicated to secondary region

### T3Block Metadata Backups
- Snapshot metadata exported daily
- Stored in T3Storage bucket

### T3FS Metadata Backups
- Inode + directory tree dumps
- Stored in warm tier

### Billing Backups
- Hourly ledger snapshots
- Stored in cold tier

---

## 3. Disaster Recovery Scenarios

### Scenario 1 — Node Failure
- Automatic rebuild (T3Storage)
- Automatic replication failover (T3Block)
- No manual intervention

### Scenario 2 — Rack Failure
- Placement ensures fragments exist elsewhere
- Rebuild begins automatically

### Scenario 3 — Region Failure
- DR region takes over
- Gateways fail over
- Metadata restored from snapshot

### Scenario 4 — Metadata Cluster Loss
- Restore from Raft snapshot
- Rebuild TinMap
- Re-sync nodes

---

## 4. Recovery Time Objectives (RTO)

| Failure Type       | RTO        |
|--------------------|------------|
| Node failure       | < 5 min    |
| Rack failure       | < 30 min   |
| Region failure     | < 1 hour   |
| Metadata loss      | < 15 min   |

---

## 5. Recovery Point Objectives (RPO)

| Data Type          | RPO        |
|--------------------|------------|
| Object data        | 0 (EC/replication) |
| Block metadata     | < 5 min    |
| FS metadata        | < 5 min    |
| Billing            | < 1 hour   |

---

## 6. DR Testing

Quarterly DR tests include:

- Node kill tests  
- Rack isolation tests  
- Region failover tests  
- Metadata restore tests  
- Billing recovery tests  

---

T3Cloud is designed for high durability and rapid recovery.
