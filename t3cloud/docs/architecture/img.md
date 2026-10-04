# T3Cloud — Full Architecture Overview

T3Cloud is a hyperscale storage and data‑infrastructure platform designed for cloud providers, AI/ML clusters, HPC environments, and enterprise workloads.  
This README provides a complete architectural overview of the system, including high‑level design, rack layout, data flow, and full ecosystem diagrams.

---

# 🌐 1. T3Cloud High‑Level Architecture Diagram

```mermaid
flowchart LR
    subgraph Client["Clients & Applications"]
        A1["AI/ML Workloads"]
        A2["HPC Clusters"]
        A3["Enterprise Apps"]
        A4["Backup Systems"]
    end

    subgraph Gateway["Gateway Layer"]
        G1["S3 Gateway"]
        G2["NVMe-oF Gateway"]
        G3["iSCSI Gateway"]
        G4["NFS/SMB Gateway"]
        G5["Vault API Gateway"]
    end

    subgraph Metadata["Metadata Cluster (Raft)"]
        M1["Metadata Node 1"]
        M2["Metadata Node 2"]
        M3["Metadata Node 3"]
    end

    subgraph Storage["T3Storage (Object Storage)"]
        S1["Storage Node 1"]
        S2["Storage Node 2"]
        S3["Storage Node 3"]
        S4["Storage Node 4"]
    end

    subgraph Block["T3Block (Block Storage)"]
        B1["Block Node 1"]
        B2["Block Node 2"]
    end

    subgraph FS["T3FS (Filesystem)"]
        F1["FS Node 1"]
        F2["FS Node 2"]
    end

    subgraph Vault["T3Vault (Archive Storage)"]
        V1["Vault Node 1"]
        V2["Vault Node 2"]
    end

    Client --> Gateway
    Gateway --> Metadata
    Gateway --> Storage
    Gateway --> Block
    Gateway --> FS
    Gateway --> Vault

    Metadata --> Storage
    Metadata --> Block
    Metadata --> FS
    Metadata --> Vault
