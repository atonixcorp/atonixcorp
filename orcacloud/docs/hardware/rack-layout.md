# T3Cloud Hardware — Rack Layout

This document defines the physical rack layout for T3Cloud deployments. The design follows a modular, repeatable pattern suitable for hyperscale expansion.

---

## 1. Rack Overview

Each rack contains:

- Storage nodes  
- Block nodes  
- Filesystem nodes  
- Vault nodes  
- Metadata nodes  
- Gateway nodes  
- Top‑of‑rack (ToR) switches  
- Power distribution units (PDUs)

### Standard Rack Specs
- **Height:** 42U  
- **Power:** Dual 32A or 63A feeds  
- **Cooling:** Cold aisle containment recommended  

---

## 2. Example Rack Layout

+------------------------------------------------+
| U42 | ToR Switch 1 (25/100 GbE)                |
| U41 | ToR Switch 2 (25/100 GbE)                |
| U40 | Gateway Node 1                           |
| U39 | Gateway Node 2                           |
| U38 | Metadata Node 1                          |
| U37 | Metadata Node 2                          |
| U36 | Metadata Node 3                          |
| U35 | Storage Node 1                           |
| U34 | Storage Node 2                           |
| U33 | Storage Node 3                           |
| U32 | Storage Node 4                           |
| U31 | Storage Node 5                           |
| U30 | Storage Node 6                           |
| U29 | Block Node 1                             |
| U28 | Block Node 2                             |
| U27 | Filesystem Node 1                        |
| U26 | Filesystem Node 2                        |
| U25 | Vault Node 1                             |
| U24 | Vault Node 2                             |
| ... | Additional Nodes                         |
| U02 | Empty / future expansion                 |
| U01 | PDUs (Left + Right)                      |
+------------------------------------------------+


---

## 3. Network Topology

### Leaf‑Spine Architecture
- **Leaf:** ToR switches  
- **Spine:** Aggregation switches  
- **Fabric:** 25/40/100 GbE  

### Cabling Rules
- Storage nodes → ToR (dual links)  
- Block nodes → ToR (dual links)  
- Filesystem nodes → ToR (dual links)  
- Vault nodes → ToR (dual links)  
- Metadata nodes → ToR (dual links)  
- Gateways → ToR (dual links)  
- ToR → Spine (4–8 uplinks)  

---

## 4. Power Layout

- Dual PDUs per rack  
- Each node connected to both PDUs  
- Power budget per rack: **80–120 kW**  

---

## 5. Thermal Layout

- Cold aisle in front  
- Hot aisle behind  
- Temperature target: **18–22°C**  
- Airflow monitoring required  

---

## 6. Expansion Strategy

Add racks in blocks:

- 6–12 storage nodes  
- 2 block nodes  
- 2 filesystem nodes  
- 2 vault nodes  
- 3 metadata nodes  
- 2 gateway nodes  

Each block increases capacity and performance linearly.
