# AtonixCorp Container Runtime Architecture
## Overview
The AtonixCorp Container Runtime is a sovereign, cloud‑native execution platform designed to build, store, schedule, and run containers across distributed compute nodes. It provides a fully automated workflow where users can deploy workloads without interacting directly with Docker, Kubernetes, or container runtimes.

This document describes the internal architecture, major subsystems, and operational flow of the container runtime.

---

## Core Components
The platform consists of eight major subsystems:

### 1. Build System
Responsible for:
- Auto‑detecting project language
- Generating Dockerfiles
- Building container images using BuildKit or Docker API
- Producing build logs
- Returning image identifiers

Located in: `internal/build/`

---

### 2. Registry Integration
Handles:
- Pushing built images to the internal registry
- Pulling images to compute nodes
- Authenticating registry operations

Supports Harbor, Docker Registry v2, or custom MinIO‑backed registries.

Located in: `internal/registry/`

---

### 3. Scheduler
Determines which compute node should run a container based on:
- Available CPU and memory
- Node health
- Placement strategy
- Tenant isolation rules

Located in: `internal/scheduler/`

---

### 4. Node Agent
A lightweight daemon running on each compute node. It:
- Pulls images
- Starts/stops containers
- Streams logs
- Collects metrics
- Reports node health

Binary located in: `cmd/node-agent/`

---

### 5. Runtime Engine
Abstracts containerd or Docker APIs to:
- Start containers
- Stop containers
- Delete containers
- Query container state

Located in: `internal/runtime/`

---

### 6. Lifecycle Manager
Provides high‑level operations:
- Start
- Stop
- Restart
- Delete

Used by the API service and scheduler.

Located in: `internal/lifecycle/`

---

### 7. Logging Pipeline
Streams logs from containers to the API and dashboard.

Located in: `internal/logging/`

---

### 8. Metrics Pipeline
Collects CPU, RAM, disk, and network metrics for each container.

Located in: `internal/metrics/`

---

## Control Plane Services

### API Service (`cmd/api`)
Exposes REST/GRPC endpoints for:
- Building containers
- Running containers
- Fetching logs
- Fetching metrics
- Managing lifecycle

### Scheduler Service (`cmd/scheduler`)
Runs continuously and assigns workloads to nodes.

---

## Execution Flow
1. **User uploads project**  
2. **API triggers build system**  
3. **Build system generates image**  
4. **Image pushed to registry**  
5. **Scheduler selects node**  
6. **Node agent pulls image**  
7. **Runtime starts container**  
8. **Logs + metrics stream back**  
9. **Lifecycle operations available**

---

## Security Model
- Rootless containers
- Namespace isolation
- Seccomp profiles
- No privileged containers
- No host mounts
- Tenant isolation enforced at scheduler level

---

## High Availability
- Multi‑node scheduling
- Automatic rescheduling on node failure
- Registry replication
- Stateless API and scheduler services

---

## Summary
The AtonixCorp Container Runtime is a fully governed, automated, and secure container execution platform designed for sovereign cloud environments. It abstracts all complexity from the user while providing enterprise‑grade control and observability.
