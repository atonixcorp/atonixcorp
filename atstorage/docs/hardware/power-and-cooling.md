# Storage Tin Hardware — Power and Cooling

This document defines the power and cooling requirements for Storage Tin deployments.

---

## 1. Power Requirements

### Per Node

| Node Type       | Typical Draw | Peak Draw |
|-----------------|--------------|-----------|
| Storage Node    | 450–650 W    | 800 W     |
| Metadata Node   | 300–450 W    | 600 W     |
| Gateway Node    | 250–350 W    | 450 W     |

### Per Rack

- **Total:** 80–120 kW depending on density
- **Feeds:** Dual 32A or 63A feeds
- **Redundancy:** A+B power path

### UPS Requirements

- Minimum 10–15 minutes runtime
- Generator-backed recommended for sovereign cloud deployments

---

## 2. Cooling Requirements

### Temperature Targets

- **Cold aisle:** 18–22°C
- **Hot aisle:** 30–40°C

### Airflow

- Front-to-back airflow for all nodes
- High‑airflow fans in storage nodes
- Cold aisle containment recommended

### Cooling Capacity

- 1 rack requires **20–30 kW** cooling capacity
- CRAC or in-row cooling supported

---

## 3. Monitoring

### Sensors

- Node-level thermal sensors
- Rack-level temperature probes
- Humidity sensors (40–60% recommended)

### Alerts

- Over-temperature warnings
- Fan failures
- PSU failures
- Hotspot detection

---

## 4. Environmental Considerations

### Recommended

- Raised floor (optional)
- Dust filtration
- Vibration isolation
- EMI shielding for sensitive metadata nodes

### Not Recommended

- Mixed hot/cold airflow
- Uncontained aisles
- Single-feed power

---

## 5. Summary

Storage Tin requires:

- Redundant power
- Strong cooling
- Continuous thermal monitoring
- Predictable airflow

These ensure long-term durability and performance for TinCore, TinS3, and TinControl nodes.
