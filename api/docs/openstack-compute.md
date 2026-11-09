# OpenStack compute integration

The main AtonixCorp API keeps OpenStack credentials on the server. Customer
requests use AtonixCorp sessions and resolve resources through persisted
organization/project mappings; they do not send Keystone credentials or use
Nova IDs as customer resource IDs.

## Configuration and account enrollment

Set these values in the root `.env` file:

- `OPENSTACK_AUTH_URL`
- `OPENSTACK_REGION` (defaults to `RegionOne`)
- `OPENSTACK_INTERFACE` (`public`, `internal`, or `admin`; defaults to `public`)
- `OPENSTACK_ACCOUNT_ENCRYPTION_KEY`: base64-encoded 32-byte key used to
  encrypt organization-scoped provider credential payloads at rest
- `OPENSTACK_ACCOUNT_MANAGEMENT_TOKEN`: a separate random token of at least
  32 characters for operator-only account enrollment
- `OPENSTACK_ACCOUNT_MANAGEMENT_TRUSTED_PROXY_CIDRS`: optional comma-separated
  CIDRs for TLS-terminating reverse proxies; direct HTTPS works without this
  setting
- `OPENSTACK_ACCOUNT_MANAGEMENT_ALLOW_LOCAL_HTTP=true`: optional local
  development setting that permits account enrollment only from a loopback
  peer; never use it when the API is reachable from an untrusted network

Generate the encryption key and management token with a trusted local secret
manager. Do not commit `.env`, `clouds.yaml`, `secure.yaml`, or any `*-openrc.sh`
file. Back up the encryption key securely; losing it makes stored credential
secrets unrecoverable. Rotate it only after re-encrypting existing mappings.

For each AtonixCorp organization, an operator must provision a distinct
Keystone project and a project-scoped application credential outside
AtonixCorp. Do not use a customer-facing or domain-admin credential. Register
the mapping through the internal API over TLS:

```http
PUT /internal/cloud/accounts/{tenantID}
Authorization: Bearer <OPENSTACK_ACCOUNT_MANAGEMENT_TOKEN>
Content-Type: application/json

{
  "project_id": "<keystone-project-id>",
  "credential_id": "<application-credential-id>",
  "credential_secret": "<application-credential-secret>",
  "region": "RegionOne"
}
```

For TLS terminated at a reverse proxy, configure only the proxy's exact CIDRs
in `OPENSTACK_ACCOUNT_MANAGEMENT_TRUSTED_PROXY_CIDRS`. The proxy must replace
`X-Forwarded-Proto` with the actual client-facing scheme; forwarded headers
from other peers are not trusted. Do not expose the API directly to untrusted
networks when enabling this setting. The local HTTP option is limited to
loopback peers and should only be used for isolated development.

The API authenticates with the supplied application credential and verifies
that its Keystone token is scoped to the supplied project and that Nova is
available in the selected region/interface before persisting the mapping.
Credential payloads are AES-GCM encrypted in the database and never returned
by the API. A Keystone project can be mapped to only one AtonixCorp
organization. The account-management token is not an AtonixCorp customer
session and must only be used by trusted operators.

The OpenStack CLI cloud entry in `clouds.yaml` is named `atonixcorp`; for
operator-side discovery use commands such as:

```sh
openstack --os-cloud atonixcorp token issue
openstack --os-cloud atonixcorp service list
openstack --os-cloud atonixcorp endpoint list
openstack --os-cloud atonixcorp catalog list
```

The API uses the mapped application credentials for customer compute
operations. It does not read `clouds.yaml` or `admin-openrc.sh`. HTTPS is
required unless `OPENSTACK_ALLOW_INSECURE_HTTP=true` is explicitly enabled for
isolated development; never use that option in production.

## API

All customer endpoints require an AtonixCorp bearer session. Organization
membership and project membership are checked before reading or mutating
resources.

| Method | Path | Purpose |
|---|---|---|
| `GET` | `/api/v1/cloud/catalog?tenant_id={id}&project_id={id}` | Return project-authorized images, instance sizes, networks, and service capabilities |
| `GET` | `/api/v1/cloud/security-groups?tenant_id={id}&project_id={id}` | List Neutron security groups and rules in the organization's mapped OpenStack project |
| `POST` | `/api/v1/cloud/security-groups?tenant_id={id}&project_id={id}` | Create a mapped-project security group with a restricted TCP 443 test rule |
| `GET` | `/api/v1/cloud/instances/capabilities?tenant_id={id}` | Return service capabilities discovered for that organization's account |
| `GET` | `/api/v1/cloud/instances?tenant_id={id}&project_id={id}` | List mapped AtonixCorp instances in an authorized project |
| `POST` | `/api/v1/cloud/instances` | Request an instance; requires an `Idempotency-Key` |
| `GET` | `/api/v1/cloud/instances/{id}` | Get an instance after checking stored ownership |
| `POST` | `/api/v1/cloud/instances/{id}/start` | Start a stopped instance |
| `POST` | `/api/v1/cloud/instances/{id}/stop` | Stop an active instance |
| `POST` | `/api/v1/cloud/instances/{id}/reboot` | Soft-reboot an active instance |
| `DELETE` | `/api/v1/cloud/instances/{id}` | Request instance deletion |
| `GET` | `/api/v1/cloud/operations?tenant_id={id}&project_id={id}` | List recent compute operations for an authorized project |
| `GET` | `/api/v1/cloud/operations/{id}` | Get an operation after checking project membership |

The catalog endpoint authorizes both tenant and project membership, resolves
the organization's configured infrastructure account, and reads its live
image, size, and network catalog. Provider identifiers are translated to
persistent AtonixCorp catalog IDs scoped to the tenant, project, provider, and
resource type. The response exposes `region_id`, `images`, `sizes`, and
`networks` using those AtonixCorp IDs. IDs remain stable across catalog
refreshes, and provider IDs are never sent back by the customer UI.
Provider errors are returned as errors; the API does not substitute sample
rows. Size metadata includes vCPU, RAM, root disk, ephemeral disk, and swap.
No AtonixCorp plan or pricing is inferred; pricing is explicitly unavailable.

The compute service depends on provider-neutral compute, catalog, and
project-scoped provisioning interfaces. A startup registry resolves
account-scoped provider factories by their stable provider name. OpenStack is
the only compute adapter registered today. Credentials implement a generic
interface, with OpenStack fields and validation in `OpenStackAccountCredentials`.
Legacy secret-only records remain readable; new records encrypt the complete
credential payload. The enrollment endpoint above remains an explicit
OpenStack operator integration. These changes do not establish another working
provider or a completed multi-provider platform.

Instance creation accepts `tenant_id`, `project_id`, `name`, `region_id`,
`image_id`, `size_id`, `network_id`, and optional `key_name`. Catalog IDs are
resolved only within the authenticated tenant/project/provider scope, then
translated to provider IDs inside the service before provisioning. Instance
responses expose the AtonixCorp resource ID and catalog IDs, along with
normalized platform status; provider instance IDs, provider type, and raw
provider status remain internal. Fixed and floating IPs are reported from the
provider's address metadata. Existing infrastructure instances are not
automatically imported.

Nova lifecycle statuses are normalized for the platform API (for example,
`ACTIVE` becomes `RUNNING` and `SHUTOFF` becomes `STOPPED`). Raw provider
statuses and resource IDs are retained internally for reconciliation.

Compute create, lifecycle, and delete requests now persist operation records
and append-only audit events. Create/start/stop/reboot/delete operations remain
`RUNNING` until the background reconciler observes the corresponding resource
state in Nova; operation records expose only AtonixCorp resource IDs and
normalized outcomes. The reconciler runs every 30 seconds when the OpenStack
provider is configured.

Security group reads and creation use the organization's mapped OpenStack
credentials and project. Customer responses omit provider IDs and expose only
group names and rules. The initial test
creation only allows inbound TCP 443 from `192.0.2.0/24`, a documentation-only
range; it does not expose a service to ordinary public clients. Security group
changes are not yet durable AtonixCorp operations and are not attached to
instances automatically.

## Current implementation boundary

The main API supports organization-scoped compute account mappings and instance
list, get, create, delete, start, stop, and reboot operations, along with live
project-scoped image, size, and network catalogs. Service
availability is derived from the Keystone catalog for the configured region
and interface. Compute subscription quotas, AtonixCorp billing/pricing,
resize/rebuild, and administrative controls are not yet implemented. Storage
uses organization-scoped Cinder credentials for mapped accounts. The separate
networking service decodes the same organization account mapping through its
Neutron adapter, including legacy and versioned credentials. Its durable operations, reconciliation,
quotas, and audit support remain outstanding.
