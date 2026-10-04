# Backup and Disaster Recovery (DR)

This document defines the backup strategy and disaster recovery procedures for Storage Tin.

---

## 1. Backup Strategy

### What We Back Up

- TinCore metadata cluster state
- TinControl configuration
- Tenant and bucket metadata
- Billing records
- Placement maps (TinMap)

### What We Do Not Back Up

- Object data (TinCore handles durability via erasure coding and replication)

---

## 2. Backup Methods

### Metadata Cluster Snapshots
- Raft snapshots taken every 5 minutes
- Stored in secure NVMe tier
- Replicated to secondary region

### TinControl Backups
- Daily configuration dumps
- Encrypted and stored in TinCore bucket
- Versioned backups retained for 30 days

### Billing Backups
- Hourly billing ledger snapshots
- Stored in cold tier

---

## 3. Disaster Recovery Scenarios

### Scenario 1 — Node Failure
- TinCore rebuilds missing fragments
- No manual intervention required

### Scenario 2 — Rack Failure
- TinMap ensures fragments exist in other racks
- Rebuild begins automatically

### Scenario 3 — Region Failure
- Cross-region replication ensures data availability
- TinS3 gateways fail over to secondary region
- TinControl switches to DR mode

### Scenario 4 — Metadata Cluster Loss
- Restore from Raft snapshot
- Rebuild TinMap
- Re-sync TinCore nodes

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
| Metadata           | < 5 min    |
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

Storage Tin is designed for sovereign-grade durability and rapid recovery.
