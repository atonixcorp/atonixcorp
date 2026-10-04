# Regions & Zones Architecture

T3Cloud supports multi‑region and multi‑zone deployments for durability, performance, and disaster recovery.

---

## 1. Region Model

A region contains:

- Multiple zones  
- Independent power  
- Independent network  
- Independent cooling  

Regions replicate data asynchronously.

---

## 2. Zone Model

A zone contains:

- Multiple racks  
- Independent failure domain  
- Local metadata replicas  
- Local gateway nodes  

Zones replicate synchronously.

---

## 3. Placement Rules

- Fragments must span racks  
- Replicas must span zones  
- Cross‑region replication optional  

---

## 4. Developer Notes

- Region failover must be deterministic  
- Zone isolation must not corrupt metadata  
- Placement engine must be topology‑aware  

Regions and zones ensure durability.
