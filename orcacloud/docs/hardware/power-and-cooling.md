# T3Cloud Hardware — Power & Cooling

This document defines the power and cooling requirements for T3Cloud deployments.

---

## 1. Power Requirements

### Per Node

| Node Type       | Typical Draw | Peak Draw |
|-----------------|--------------|-----------|
| Storage Node    | 450–650 W    | 800 W     |
| Block Node      | 350–500 W    | 700 W     |
| Filesystem Node | 300–450 W    | 600 W     |
| Vault Node      | 400–600 W    | 750 W     |
| Metadata Node   | 300–450 W    | 600 W     |
| Gateway Node    | 250–350 W    | 450 W     |

### Per Rack
- **Total:** 80–120 kW  
- **Feeds:** Dual 32A or 63A  
- **Redundancy:** A+B power path  

### UPS Requirements
- Minimum 10–15 minutes runtime  
- Generator‑backed recommended  

---

## 2. Cooling Requirements

### Temperature Targets
- **Cold aisle:** 18–22°C  
- **Hot aisle:** 30–40°C  

### Airflow
- Front‑to‑back airflow  
- High‑airflow fans in storage nodes  
- Cold aisle containment recommended  

### Cooling Capacity
- 1 rack requires **20–30 kW** cooling  

---

## 3. Monitoring

### Sensors
- Node‑level thermal sensors  
- Rack‑level temperature probes  
- Humidity sensors (40–60%)  

### Alerts
- Over‑temperature  
- Fan failures  
- PSU failures  
- Hotspot detection  

---

## 4. Environmental Considerations

### Recommended
- Raised floor  
- Dust filtration  
- Vibration isolation  
- EMI shielding  

### Not Recommended
- Mixed airflow  
- Uncontained aisles  
- Single‑feed power  

---

## 5. Summary

T3Cloud requires:

- Redundant power  
- Strong cooling  
- Continuous thermal monitoring  
- Predictable airflow  

These ensure long‑term durability and performance.
