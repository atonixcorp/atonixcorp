# Provider boundary

AtonixCorp owns the customer-facing resource model, API contract, IDs, status
language, and activity messages. Infrastructure providers are implementation
details behind AtonixCorp services and adapters.

## Customer-facing contract

- Use AtonixCorp service and resource names such as Compute Instance, Network,
  Block Storage, Security Group, Database, and HPC Job.
- Return AtonixCorp resource IDs. Provider resource IDs, project IDs, credential
  material, and provider status fields must not be serialized to customer APIs.
- Translate provider failures into stable AtonixCorp error codes and concise
  messages. Preserve diagnostic details in server-side logs or internal records,
  associated with a request or correlation ID where available.
- Customer UI code must call AtonixCorp API routes, not provider APIs or SDKs.
- Keep provider names, service names, environment variables, and technical
  setup instructions in adapter code and internal operational documentation.
- Do not globally rename provider terminology. Classify each occurrence by
  whether it is internal implementation or customer experience.

## Existing boundaries

- Compute catalog entries are mapped to AtonixCorp IDs, and
  `ComputeProviderRegistry` selects the compute adapter. Account credentials
  implement `CloudAccountCredentials`; validation, encoding, and legacy
  decoding belong to the adapter. `OpenStackAccountCredentials` remains the
  only production credential implementation, and the operator enrollment
  endpoint deliberately still accepts its OpenStack-specific contract.
- Storage resolves drivers through `BlockDriverRegistry`; local storage and the
  Cinder adapter are registered by the server. Generic scoped factories no
  longer require `CinderConfig`; configuration belongs to the Cinder factory.
- Networking exposes AtonixCorp network and subnet IDs through its core API.
  The Neutron adapter's provider IDs are not part of the network JSON model.
  Its adapter reads legacy and versioned encrypted organization credentials.
- Database policy, plans, idempotency, and operation persistence belong to the
  domain controller. `DatabaseProvisioner` owns capability checks and resource
  provisioning; only the OpenStack adapter is wired into the executables.
  Images, flavors, networks, attachments, and infrastructure status are adapter
  configuration/state, not database domain fields. Infrastructure readiness
  still does not mean managed-engine readiness.
- HPC requests select an execution profile, not a backend, partition, or
  scheduler accounting identity. Profiles resolve to private scheduler bindings.
  Slurm and Kubernetes adapters exist, but the HPC executable is not currently
  integrated or buildable end to end. Do not advertise them as supported HPC
  providers based on these types or unit tests.
- The container image-pull policy uses a platform string in the request;
  conversion to the Kubernetes SDK type happens in the adapter. Remaining
  volume/probe/resource semantics are still Kubernetes-oriented and need further
  review before claiming support for a different container runtime.

## Capability and audit status

This is a targeted source audit, not proof of complete platform integration.
"Implemented" means code exists; "wired" means a startup path selects it;
"tested" below means local unit/compile checks, not live infrastructure.
No service is declared production-ready by this refactor.

| Service | Current boundary / backing | Remaining integration or coupling |
| --- | --- | --- |
| Compute | Generic domain/catalog/credential interfaces; OpenStack wired | Only OpenStack implemented; operator enrollment still provider-specific; live provisioning not retested |
| Network | Generic resource IDs; Neutron adapter; dual credential decoding tested | Only Neutron provisioning; durable operations and other network resources incomplete |
| Storage | Static/scoped driver registry; Cinder configuration adapter-owned | Cinder/local/libvirt implementations are not interchangeable capability promises; live drivers not retested |
| Database | Generic plans/resources/provisioner; OpenStack wired; recovery tests pass | Shared-project adapter configuration; engine bootstrap incomplete; live schema migration test not run |
| HPC | Domain execution profiles mapped to private scheduler targets; job tests pass | Existing inventory/Kubernetes compile errors; repositories and event publisher remain unwired |
| Kubernetes | Existing container provisioner with adapter-side SDK conversion | Public probes, volumes, and quantity semantics still reflect the current runtime; no additional runtime claimed |
| Identity | Platform auth is separate from Keystone operator credentials | `cloud/core/keystone` remains provider integration code, not universal identity support |
| Security | Platform policies/audit separate from provider security groups | Neutron security-group operations are not a completed generic security provider contract |
| Load balancing | No integrated independent multi-provider capability established | Do not advertise support from wireframes or provider names |
| DNS | Network inventory is not evidence of a managed DNS service | Dedicated provider interface/integration not established by this audit |
| Images | Compute catalog maps images to scoped platform IDs | Current catalog adapter is OpenStack-backed; provider image administration is not a universal domain API |
| Billing | No pricing/capacity inferred from provider catalog | Authoritative billing/provider pricing integration remains unavailable |
| Observability | Platform resource kinds and response filtering; provider sources remain internal | Configured source URLs do not establish collectors, real metrics, or complete provider telemetry |
| Orchestration | Platform workflow scaffolding exists | Legacy `cloud/controlplane` still imports concrete Keystone/Nova clients; its constructor does not match the current scheduler signature and is not part of the tested module paths |

Compute capabilities come from the current adapter's discovered catalog.
Database catalog availability calls the configured provisioner's `Supports`
method for engine/version/plan/region; that indicates configured provisioning,
not engine readiness. HPC rejects unknown or disabled execution profiles.
There is no unified all-service provider capability registry yet. Test-only
providers are never registered by production startup or listed in the UI.

## Persistence compatibility

New account records encrypt the full adapter credential payload and record
`provider` and `credential_version=1`. Version 0 OpenStack secret-only records
remain readable through the OpenStack decoder. The old
`open_stack_project_id` column is retained as a compatibility mapping for
generic account scope; its existing uniqueness constraint remains conservative.
Upgrade the API and networking service together: old networking binaries do
not understand version 1 payloads. Existing keys and ciphertext are preserved.

Database metadata migration adds `provider_name` and private JSONB
`provider_metadata`, copying legacy OpenStack resource IDs/status/attachment
state without dropping old columns. New writes use private metadata; old
columns are no longer updated, so rolling back to an old binary needs a
deliberate metadata backfill. A guarded PostgreSQL test exists in
`cloud/database/store/postgres_test.go`; it was compiled but not run against
PostgreSQL on this host (Docker access denied and `initdb` absent).

HPC request JSON now uses explicit snake_case fields and requires
`execution_profile`. Legacy `backend`, `partition`, and `account` request
fields are rejected. Previously persisted jobs need an execution-profile
mapping before refresh/cancel; no production repository migration is claimed.

Adding another provider requires registering an adapter and translating its
catalog, resources, status, and errors into these AtonixCorp contracts. It does
not justify exposing provider credentials or identifiers in the frontend.

## Review checklist

1. Is the API request expressed using AtonixCorp concepts and IDs?
2. Are provider resource IDs and credentials kept out of response models?
3. Are backend errors translated before crossing the customer API boundary?
4. Are provider details retained only in authorized server-side diagnostics?
5. Does the UI and customer documentation describe the AtonixCorp capability,
   rather than the underlying provider implementation?
