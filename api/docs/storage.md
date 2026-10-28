# Storage API

The authenticated storage control plane implements an asynchronous,
project-scoped block-volume workflow with a local development driver and an
OpenStack Cinder driver:

- `GET /api/v1/storage/volume-types`
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

## OpenStack Cinder

The Compose stack defaults to `STORAGE_DRIVER=local-file`. To enable Cinder,
set `STORAGE_DRIVER=cinder` and configure these variables in the root `.env`
file (never commit credentials):

- `CINDER_AUTH_URL`: HTTPS Keystone v3 URL, for example `https://keystone.example/v3`
- `CINDER_USERNAME`, `CINDER_PASSWORD`: dedicated service-account credentials
- `CINDER_PROJECT_ID`, `CINDER_USER_DOMAIN_ID`: project scope and user domain
- `CINDER_REGION`, `CINDER_INTERFACE`: catalog region and `public`, `internal`,
  or `admin` endpoint interface
- `CINDER_VOLUME_TYPE_IDS`: comma-separated allowlist of Cinder volume type IDs
- `CINDER_AVAILABILITY_ZONES`: optional comma-separated allowed zone names

The driver selects a matching HTTPS `volumev3` catalog endpoint, obtains and
refreshes Keystone tokens, limits selectable types to the configured allowlist,
uses provider metadata to recover create retries, and waits for Cinder status
before completing create/delete operations. Configure a least-privilege
provider account, network access, and Cinder policy to match the operations
required by this API. A usable Cinder endpoint and credentials are not present
in this development environment, so live provisioning must be verified against
the target OpenStack deployment before production use.

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
