### *Node Agent Documentation*

```markdown
# AtonixCorp Node Agent
## Overview
The Node Agent is a lightweight daemon running on every compute node in the AtonixCorp cloud. It is responsible for executing containers, collecting metrics, and reporting node health.

Binary location:
cmd/node-agent/main.go

Code

---

## Responsibilities
### 1. Image Management
- Pull images from registry
- Verify image integrity
- Cache images locally

### 2. Container Execution
Uses containerd or Docker API to:
- Start containers
- Stop containers
- Delete containers
- Query container state

### 3. Logging
Streams logs from container stdout/stderr to:
- API service
- Dashboard
- Log storage backend

### 4. Metrics Collection
Collects:
- CPU usage
- Memory usage
- Disk I/O
- Network I/O

### 5. Node Health Reporting
Sends periodic heartbeats to scheduler:
- Available CPU
- Available memory
- Disk pressure
- Network status

---

## Configuration
Node agent loads configuration from environment variables or config files:

NODE_NAME=node-1
REGISTRY_URL=registry.local
ENV=prod

Code

---

## Internal Architecture

### Runtime Module (`internal/runtime`)
Handles container lifecycle operations.

### Metrics Module (`internal/metrics`)
Collects metrics using cgroups or containerd introspection.

### Logging Module (`internal/logging`)
Streams logs using pipes or containerd log drivers.

### Registry Module (`internal/registry`)
Pulls images securely.

---

## Failure Handling
Node agent automatically:
- Restarts failed containers
- Reports failures to scheduler
- Re-pulls corrupted images
- Marks node unhealthy if runtime fails

---

## Summary
The Node Agent is the execution backbone of the AtonixCorp container runtime. It ensures reliable, secure, and observable container execution across distributed nodes.