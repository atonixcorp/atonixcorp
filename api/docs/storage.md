# Storage API

The authenticated storage control plane implements an asynchronous,
project-scoped block-volume workflow with a local development driver and an
OpenStack Cinder driver:

- `GET /api/v1/storage/volume-types?tenant_id={id}&project_id={id}`
- `GET /api/v1/storage/overview?tenant_id={id}&project_id={id}`
- `GET /api/v1/storage/volumes?tenant_id={id}&project_id={id}`
- `POST /api/v1/storage/volumes` with an `Idempotency-Key` header
- `GET /api/v1/storage/volumes/{id}`
- `DELETE /api/v1/storage/volumes/{id}` with an `Idempotency-Key` header
- `GET /api/v1/storage/operations/{id}`

Volume creation validates project membership, write role, name, size, selected
driver capabilities, and project quotas. It commits a `CREATING` resource, a
pending operation, and an audit event before returning `202 Accepted`. The worker
creates resources idempotently and polls the provider before marking a volume
`AVAILABLE`. Provider IDs are stored separately from customer IDs. Deletion is
also asynchronous and requires delete protection to be disabled. Transient
provider failures are retried up to three times. Project defaults are 20
volumes and 1024 GiB.

Storage operations resolve named drivers through a startup `BlockDriverRegistry`.
Static development drivers and account-scoped driver factories share the same
`BlockDriver` contract. The server registers the configured local driver and
the Cinder factory; persisted volumes resolve their original driver by name,
and scoped operations revalidate the mapped project before contacting it.
Additional storage adapters can register without adding provider-specific
branches to the volume service. The organization account credential format and
the current account-scoped factory remain OpenStack-specific.

## OpenStack Cinder

Map each organization to its own OpenStack project and application credential
using the cloud account API, and configure:

- `OPENSTACK_AUTH_URL` (or `CINDER_AUTH_URL`), `OPENSTACK_REGION`
- `OPENSTACK_INTERFACE` (or `CINDER_INTERFACE`)
- `OPENSTACK_ALLOW_INSECURE_HTTP=true` only for isolated DevStack development
- `CINDER_VOLUME_TYPE_IDS`: comma-separated allowlist of Cinder volume type IDs
- `CINDER_ENCRYPTED_VOLUME_TYPE_IDS` and `CINDER_UNENCRYPTED_VOLUME_TYPE_IDS`:
  optional encryption classifications for allowed types when the mapped
  application credential cannot read Cinder's admin-only encryption endpoint
- `CINDER_AVAILABILITY_ZONES`: optional comma-separated allowed zone names

`OPENSTACK_ACCOUNT_ENCRYPTION_KEY` must be configured to decrypt the stored
mapping. For a mapped organization, Storage always uses its mapped application
credential, project, and region, regardless of `STORAGE_DRIVER`. Keystone's
token project and the catalog endpoint project are checked before any Cinder
request. No shared-project credential is used as a fallback. Configure
least-privilege Cinder policy and network access for each mapped account.
The type allowlist and zone settings apply to all organizations.
Encryption classifications must refer to IDs in `CINDER_VOLUME_TYPE_IDS` and
must not conflict. When Cinder returns an encryption status, it is checked
against any configured classification; a configured classification is used
only when Cinder denies access to that status endpoint.

An unmapped organization can use a development driver only when explicitly
configured with `STORAGE_DRIVER=local-file` or `libvirt-qemu`. With
`STORAGE_DRIVER=cinder` or no development driver, unmapped organizations cannot
provision. Mapping, decryption, configuration, and provider errors never fall
back to local or shared Cinder. Existing local volumes continue to use their
original local driver. Cinder volumes retain their mapped provider project ID
internally; if the mapping is changed or removed, operations fail closed until
the original project is restored. Older shared-Cinder records without this
project binding are not accessed automatically and require an operator-led
migration; do not bind them to a different project.

Volume list and detail requests inspect Cinder after authorization and
reconcile persisted size, zone and lifecycle state; a missing provider volume
is marked `ERROR`. Provider IDs, provider status, zones, and attachment
identifiers remain internal and are omitted from customer responses. Provider
failures are reported rather than silently presenting stale Cinder details.
The UI refresh button triggers a fresh read. Live provisioning must be verified
against the target OpenStack deployment.

## Libvirt/QEMU host storage

The API also supports host-backed qcow2 images through `virsh` and `qemu-img`.
This backend uses a local Libvirt connection to verify host access and creates
files in a dedicated directory mounted into the API container. It does not
create or modify Libvirt storage pools, attach disks to guest domains, or
provide encryption. Use the separate Compose override so local and Cinder
configurations do not need host Libvirt mounts:

```sh
docker compose -f docker-compose.yml -f docker-compose.libvirt.yml up -d --build
```

Set `LIBVIRT_SOCKET_HOST_PATH`, `LIBVIRT_GID`, and
`LIBVIRT_STORAGE_HOST_PATH` in the local `.env`. Use a host group that grants
the API user socket access and grants the host QEMU process access to the
dedicated storage directory.
The socket must be an authorized Libvirt Unix socket, and the storage directory
must already exist. Grant the API container's `app` user access through that
numeric supplementary group ID; make sure the host QEMU process can read the
created images. The driver creates images with owner/group-only read/write
permissions, so use a dedicated directory whose group is shared with the host
QEMU process. Use a dedicated directory, not a shared image or pool directory.
The driver uses `LIBVIRT_URI` (defaulted by the override to
`qemu+unix:///system?socket=/run/libvirt/libvirt-sock`) and stores qcow2 images
named from the generated Atonix volume ID. To return to the default local
driver, omit the override and recreate the API service.

The local driver remains development-only: its volumes are sparse backing files,
not attachable block devices, and do not provide at-rest encryption. The current
Storage API still does not implement snapshots, backups, object buckets, file
shares, compute attachment, resizing, or provider capacity/performance
guarantees. Cinder encryption is selected by the configured volume type rather
than by claiming that the local driver encrypts data.
