# Data Flow Architecture

This document describes how data moves through T3Cloud across object, block, filesystem, and archive engines.

---

## 1. Object Storage Flow (T3Storage)

### Write
1. Gateway receives request  
2. Metadata cluster selects placement  
3. Fragments streamed to nodes  
4. Metadata committed  

### Read
1. Metadata resolves fragments  
2. Nodes stream fragments  
3. Gateway reconstructs object  

---

## 2. Block Storage Flow (T3Block)

### Write
1. Host sends block  
2. Protocol layer receives  
3. NVMe writes  
4. Metadata updated  
5. Replication (optional)  

### Read
1. Host requests block  
2. Node reads NVMe  
3. Data returned  

---

## 3. Filesystem Flow (T3FS)

### Write
1. Protocol server receives write  
2. Node writes block  
3. Metadata updated  

### Read
1. Metadata resolves block list  
2. Node streams blocks  

---

## 4. Archive Flow (T3Vault)

### Archive
1. Policy triggers archival  
2. Object moved to cold tier  
3. Metadata updated  

### Recall
1. User requests object  
2. Background restore  
3. Object promoted  

---

## 5. Developer Notes

- All flows must be observable via metrics  
- Metadata operations must be atomic  
- Data paths must be zero‑copy where possible  

Data flow is the backbone of T3Cloud.
