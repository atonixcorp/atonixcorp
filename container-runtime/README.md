# AtonixCorp Container Runtime
##  Cloud • Automated Compute • Intelligent Container Execution

The **AtonixCorp Container Runtime** is the core execution engine of the AtonixCorp Cloud Stack. It provides a fully automated, secure, and governed environment for building, storing, scheduling, and running containers across distributed compute nodes. The runtime abstracts away all complexity of Docker, Kubernetes, and container orchestration, enabling developers to deploy workloads with zero setup while giving operators full control, observability, and governance.

---

## What the Container Runtime Is Meant to Do

The AtonixCorp Container Runtime is designed to provide a complete, automated container‑execution workflow inside the sovereign cloud. Its responsibilities include:

### **1. Automated Container Builds**
Users upload code — the runtime:
- Detects the language (Python, Node.js, Go, Java, etc.)
- Generates Dockerfiles automatically
- Builds container images using BuildKit or Docker API
- Produces structured build logs
- Stores images in the internal registry

This eliminates the need for users to install Docker or write Dockerfiles.

---

### **2. Secure Internal Image Storage**
The runtime integrates with internal registries such as:
- Harbor  
- Docker Registry v2  
- MinIO‑backed registries  

It manages:
- Image push/pull  
- Authentication  
- Versioning  
- Tenant isolation  

---

### **3. Distributed Scheduling**
The scheduler:
- Selects the best compute node for each workload  
- Balances CPU, RAM, and disk usage  
- Enforces tenant isolation  
- Supports multi‑node and multi‑zone deployments  
- Reschedules workloads automatically on node failure  

This ensures optimal resource usage and high availability.

---

### **4. Container Execution via Node Agent**
Each compute node runs a lightweight agent responsible for:
- Pulling images  
- Starting containers  
- Stopping containers  
- Deleting containers  
- Streaming logs  
- Collecting metrics  
- Reporting node health  

The node agent is the execution backbone of the runtime.

---

### **5. Unified Lifecycle Management**
The runtime exposes clean lifecycle operations:
- Start  
- Stop  
- Restart  
- Delete  
- Query status  

These operations are used by dashboards, automation systems, and orchestration layers.

---

### **6. Real‑Time Logs and Metrics**
The runtime provides:
- Real‑time log streaming  
- CPU/RAM/disk/network metrics  
- Container health checks  
- Node telemetry  

This enables deep observability across the cloud.

---

### **7. Enterprise‑Grade Security**
The runtime enforces:
- Rootless containers  
- Namespace isolation  
- Seccomp profiles  
- No privileged containers  
- No host mounts  
- No host networking  
- Tenant‑aware scheduling  

This ensures safe multi‑tenant operation inside the sovereign cloud.

---

## Project Structure

atonixcorp/container-runtime/
├── cmd/
│   ├── api/              # Public API service
│   ├── scheduler/        # Distributed scheduler
│   └── node-agent/       # Compute node agent
├── internal/
│   ├── build/            # Build system
│   ├── registry/         # Registry integration
│   ├── runtime/          # Container runtime abstraction
│   ├── scheduler/        # Scheduling logic
│   ├── lifecycle/        # Lifecycle operations
│   ├── logging/          # Log streaming
│   ├── metrics/          # Metrics collection
│   └── config/           # Configuration loader
├── pkg/
│   ├── api/              # Shared API types
│   ├── models/           # Core data models
│   ├── clients/          # Client SDKs
│   └── utils/            # Utilities
├── deploy/
│   ├── k8s/              # Kubernetes manifests
│   ├── systemd/          # Systemd service files
│   └── docker-compose/   # Local development stack
├── configs/              # Environment configs
├── scripts/              # Build/run automation
├── docs/                 # Architecture & API documentation
└── README.md             # Project overview


---

## Core Workflow

1. User uploads project  
2. API triggers build system  
3. Build system generates image  
4. Image pushed to registry  
5. Scheduler selects node  
6. Node agent pulls image  
7. Runtime starts container  
8. Logs + metrics stream back  

This workflow powers container execution inside the AtonixCorp sovereign cloud.

---

## Development

### Build all services
```bash
make build
Run locally
bash
scripts/run-local.sh