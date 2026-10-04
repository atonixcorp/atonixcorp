OrcaCompute
OrcaCompute is a high‑performance, distributed compute orchestration engine designed for cloud, HPC, and AI workloads. It provides a modular, Nova‑inspired architecture for provisioning, scheduling, and managing virtual machine instances across heterogeneous hypervisors.

Overview
OrcaCompute is the compute subsystem of the OrcaCloud platform. It delivers:

Distributed VM lifecycle management

Intelligent scheduling and placement

Hypervisor abstraction (KVM/QEMU/libvirt)

Multi‑tenant isolation

Integrated networking and storage workflows

High‑availability control plane

Extensible drivers for future bare‑metal, container, and GPU workloads

The system is built using modern Go engineering practices, clean separation of concerns, and a modular architecture optimized for scale.

Core Features
Compute API for instance lifecycle operations

Scheduler with filters, weighers, and placement engine

Compute Agent running on each hypervisor node

Placement Service for resource inventory and allocations

Messaging layer using RPC and event bus

Hypervisor drivers for libvirt and QEMU

Networking drivers for bridges, OVS, and IPAM

Storage drivers for QCOW2, image caching, and volume attachment

Deployment automation via Docker, Terraform, and Kubernetes

Operational tooling for monitoring, metrics, and health checks

Architecture
OrcaCompute is composed of four major subsystems:

1. API Service
Handles REST requests for instance creation, deletion, flavors, hosts, and metadata.
Includes RBAC, rate limiting, request logging, and schema validation.

2. Scheduler
Determines optimal host placement using resource filters, weighers, and a deterministic placement engine.

3. Compute Agent
Runs on each compute node and performs VM lifecycle operations, disk preparation, image conversion, network setup, and heartbeat reporting.

4. Placement Service
Tracks host inventory, resource allocations, NUMA topology, overcommit rules, and reclaim operations.

Project Structure
Code
orca-compute/
├── api/
├── scheduler/
├── compute-agent/
├── placement/
├── messaging/
├── images/
├── config/
├── pkg/
├── internal/
├── deployments/
├── scripts/
├── tests/
└── docs/
Each subsystem is isolated for clarity, testing, and scalability.

Hypervisor Support
Current support:

KVM/QEMU via libvirt

QCOW2 and raw disk formats

Planned: live migration, CPU pinning, NUMA awareness

Future drivers may include VMware, Xen, bare‑metal, and GPU virtualization.

Networking
Networking is modular and driver‑based:

Linux bridges

Open vSwitch

IPAM for static and dynamic assignment

Planned: security groups, VXLAN overlays

Storage
Storage subsystem provides:

Image download and caching

QCOW2 conversion

Volume attachment

Planned: snapshots, Ceph/RBD integration

Monitoring and Health
OrcaCompute exposes:

Prometheus metrics

Structured logs

Health endpoints

Event bus notifications

Deployment
Deployment options include:

Docker for local development

Terraform for cloud provisioning

Kubernetes for production control plane

Compute agents run as systemd services or DaemonSets.

Testing
The project includes:

API tests

Scheduler tests

Compute agent tests

Integration tests

CI scripts

Documentation
Documentation is located in docs/:

Architecture diagrams

Scheduler design

Placement model

API specification (OpenAPI)

Operations runbook

Development Requirements
Go 1.21+

Libvirt and QEMU

Debian 13 recommended

Redis (optional cache)

PostgreSQL (primary database)

Roadmap
Live migration

NUMA‑aware scheduling

GPU scheduling

Bare‑metal provisioning

Multi‑region federation

High‑availability control plane