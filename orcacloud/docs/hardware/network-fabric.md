# T3Cloud Hardware — Network Fabric

This document describes the network architecture powering T3Cloud, including leaf‑spine topology, VLAN segmentation, bandwidth requirements, and routing strategy.

---

## 1. Network Architecture

T3Cloud uses a **leaf‑spine fabric** for predictable, low‑latency, high‑bandwidth communication.

### Leaf Layer (ToR)
- Connects directly to nodes  
- Provides 25/40/100 GbE ports  
- Dual ToR switches per rack  

### Spine Layer
- Aggregates all racks  
- Provides 100/200 GbE uplinks  
- Ensures non‑blocking fabric  

---

## 2. VLAN Segmentation

T3Cloud uses strict network segmentation:

| VLAN | Purpose |
|------|---------|
| Storage VLAN | Node‑to‑node traffic (EC, rebuild, replication) |
| API VLAN | Gateway traffic (S3, NVMe‑oF, NFS, SMB) |
| Management VLAN | SSH, metrics, control plane |
| Replication VLAN | Cross‑rack or cross‑region replication |

Segmentation improves security and performance.

---

## 3. Bandwidth Requirements

### Storage Nodes
- Dual 25/100 GbE  
- Required for EC parallel writes  

### Block Nodes
- Dual 100 GbE  
- Required for NVMe‑oF  

### Filesystem Nodes
- Dual 25/40/100 GbE  

### Vault Nodes
- Dual 10/25 GbE  

### Metadata Nodes
- Dual 25/40/100 GbE  

### Gateways
- Dual 25/40 GbE  

---

## 4. Routing Strategy

### East‑West Traffic
- Node‑to‑node  
- EC encoding  
- Rebuild operations  
- Metadata replication  

### North‑South Traffic
- Client‑to‑gateway  
- Gateway‑to‑node  

Routing is optimized for:

- Low latency  
- High throughput  
- Failure isolation  

---

## 5. RDMA (Optional)

RDMA (RoCEv2) can be enabled for:

- High‑performance EC  
- NVMe‑oF acceleration  
- Low‑latency metadata operations  

---

## 6. Network Monitoring

Monitor:

- Link health  
- Packet loss  
- Latency  
- EC throughput  
- Rebuild bandwidth  
- Gateway saturation  

---

## 7. Summary

The T3Cloud network fabric is designed for:

- High bandwidth  
- Low latency  
- Fault isolation  
- Horizontal scalability  

It is the backbone of all T3Cloud data movement.
