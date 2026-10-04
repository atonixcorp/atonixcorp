# TinControl — Control Plane

TinControl manages the configuration, tenants, buckets, billing, and cluster topology of Storage Tin.

## Responsibilities

- Tenant creation and management
- Bucket provisioning and policies
- Billing and usage metering
- Cluster configuration (TinMap)
- Monitoring and alerting integration
- Web UI for operators and customers

## Components

### 1. TinControl API
Provides REST/gRPC endpoints for:
- Tenant operations
- Bucket operations
- Billing queries
- Cluster configuration updates

### 2. TinControl Web UI
Built with TypeScript/React:
- Tenant dashboard
- Bucket explorer
- Usage charts
- Cluster health view

### 3. Billing Engine
Tracks:
- Object count
- Storage usage
- API request volume
- Replication and cross‑region traffic

TinControl is the administrative brain of Storage Tin.
