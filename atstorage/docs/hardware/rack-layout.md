# Storage Tin Hardware — Rack Layout

This document defines the physical rack layout for Storage Tin deployments. The design follows a modular, repeatable pattern suitable for hyperscale expansion.

---

## 1. Rack Overview

Each rack contains:

- TinCore storage nodes
- TinCore metadata nodes
- TinS3 gateway nodes
- Top-of-rack (ToR) switches
- Power distribution units (PDUs)

### Standard Rack

- **Height:** 42U
- **Power:** Dual 32A feeds
- **Cooling:** Cold aisle containment recommended

---

## 2. Rack Layout (Example)

+------------------------------------------------+
| U42 | ToR Switch 1 (25/100 GbE)                |
| U41 | ToR Switch 2 (25/100 GbE)                |
| U40 | Gateway Node 1 (TinS3)                   |
| U39 | Gateway Node 2 (TinS3)                   |
| U38 | Metadata Node 1                          |
| U37 | Metadata Node 2                          |
| U36 | Metadata Node 3                          |
| U35 | Storage Node 1                           |
| U34 | Storage Node 2                           |
| U33 | Storage Node 3                           |
| U32 | Storage Node 4                           |
| U31 | Storage Node 5                           |
| U30 | Storage Node 6                           |
| ... | Additional Storage Nodes                 |
| U02 | Empty / future expansion                 |
| U01 | PDUs (Left + Right)                      |
+------------------------------------------------+


---

## 3. Network Topology

### Leaf-Spine Architecture

- **Leaf:** ToR switches in each rack
- **Spine:** Aggregation switches connecting all racks
- **Fabric:** 25/100 GbE

### Cabling

- Storage nodes → ToR (dual links)
- Metadata nodes → ToR (dual links)
- Gateway nodes → ToR (dual links)
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

- 1 rack = 6–12 storage nodes
- 3 metadata nodes
- 2 gateway nodes

Each block increases capacity and performance linearly.

