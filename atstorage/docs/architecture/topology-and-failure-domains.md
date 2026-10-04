# Topology and Failure Domains

Storage Tin is designed to survive disk, node, rack, and region failures without data loss.

## Topology Model

TinMap defines the physical and logical layout:

- **Disk** — lowest unit
- **Node** — server containing multiple disks
- **Rack** — group of nodes sharing power/network
- **Region** — data center or geographic location

## Failure Domains

TinCore places fragments across domains to ensure durability:

- Disk failure → rebuild from other disks
- Node failure → rebuild from other nodes
- Rack failure → fragments stored in other racks
- Region failure → cross‑region replication

## Erasure Coding Profiles

Examples:
- EC 6+3 (large objects)
- EC 4+2 (medium objects)
- Replication 3× (small objects)

TinMap ensures optimal placement for performance and durability.
