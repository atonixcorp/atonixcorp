# AtonixCorp Observability

Observability is a built-in platform service. It is not a provisionable cloud resource. Resource records are reconciled from owning services and are scoped by tenant and project.

## Current API surface

All routes require the normal AtonixCorp bearer session and verify both tenant and project membership:

```text
GET /api/v1/observability/tenants/{tenantID}/projects/{projectID}/overview
GET /api/v1/observability/tenants/{tenantID}/projects/{projectID}/resources
GET /api/v1/observability/tenants/{tenantID}/projects/{projectID}/resources/{resourceType}/{resourceID}
GET /api/v1/observability/tenants/{tenantID}/projects/{projectID}/resources/{resourceType}/{resourceID}/metrics
GET /api/v1/observability/tenants/{tenantID}/projects/{projectID}/{metrics|logs|events|alerts|dashboards|integrations}
```

The registry currently discovers API-managed storage volumes and container workloads from their authoritative PostgreSQL tables. A reconciliation request upserts discovered records; resources removed from those source tables are marked deleted and are not returned. There is intentionally no Observability resource-creation endpoint.

The collector contract is resource-oriented: each `ResourceCollector` translates a source's native inventory into a registry record, and each `TelemetryAdapter` supplies metrics for a known resource. This keeps provider-specific discovery and metric availability separate from the UI's normalized response shape. OpenStack, Kubernetes, database, network, and Slurm resource families are represented in the overview taxonomy, but are not presented as discovered until an owning integration supplies them.

## Metrics adapter contract

The optional Prometheus-compatible adapter is server-side. Configure `OBSERVABILITY_PROMETHEUS_URL` and, when required, `OBSERVABILITY_PROMETHEUS_TOKEN` for the API service. It reports `connected` only when the Prometheus readiness endpoint responds successfully. It queries only these normalized metric families and requires every series to carry the exact `tenant_id`, `project_id`, `resource_type`, and `resource_id` labels:

| Metric family | Unit |
| --- | --- |
| `atonix_resource_cpu_utilization_ratio` | ratio (returned as percent) |
| `atonix_resource_memory_utilization_ratio` | ratio (returned as percent) |
| `atonix_resource_storage_utilization_ratio` | ratio (returned as percent) |
| `atonix_resource_network_receive_bytes_per_second` | bytes/second |
| `atonix_resource_network_transmit_bytes_per_second` | bytes/second |

This prevents exposing arbitrary PromQL or cross-project series through the tenant API. A reachable Prometheus server without matching series does not make resource metrics available.

## Integration boundaries

- `CEILOMETER_API_URL` and `OBSERVABILITY_SLURM_API_URL` are reported as configured-but-unverified when set. They do not mean Ceilometer or Slurm collection is active: Keystone credential lifecycle, provider-specific discovery, and normalized metric ingestion still need to be implemented and verified.
- The current source adapters cover volumes and container workloads only. Database instances/services, OpenStack compute and network resources, object storage, hosts, and OpenHPC/Slurm inventory are not yet imported into the registry.
- Metrics are unknown unless a configured adapter returns samples. The API does not generate sample values or report `healthy` based on a resource's provisioning state.
- Logs and alert evaluation return unavailable until their respective authorized collectors/evaluator are connected. The Events API returns platform container/storage operation events and includes security audit events only for users with the existing elevated audit permissions.
- Built-in dashboard definitions organize resource families; custom dashboard persistence and alert-rule authoring are not implemented.
