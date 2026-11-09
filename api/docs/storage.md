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

## Application object storage

Object storage is separate from the block-volume drivers above. Set
`STORAGE_PROVIDER=minio`, `MINIO_ENDPOINT` (host:port, without a scheme),
`MINIO_ACCESS_KEY`, `MINIO_SECRET_KEY`, `MINIO_BUCKET` (defaults to
`atonixcorp-bucket1`), `MINIO_USE_SSL=true`, and optionally `MINIO_REGION`.
Compose passes these through its API env file. Provision the bucket separately;
the application does not create buckets or prefixes. Prefixes are object keys,
not directories. Use a dedicated application identity scoped to this bucket,
with only the S3 bucket listing/location and object get, put and delete
permissions needed by the application. Do not use root credentials; `samorca`
is explicitly rejected, but operators must also ensure the configured identity is
not any other administrator. Keep secrets in a secret manager or an untracked
environment file; rotate by updating the identity/secret and restarting the API.
Use TLS in production. For isolated development only,
`MINIO_USE_SSL=false` also requires `MINIO_ALLOW_INSECURE_DEVELOPMENT=true`.

An example identity policy for this bucket is:

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {"Effect": "Allow", "Action": ["s3:GetBucketLocation", "s3:ListBucket"],
     "Resource": ["arn:aws:s3:::atonixcorp-bucket1"]},
    {"Effect": "Allow", "Action": ["s3:GetObject", "s3:PutObject", "s3:DeleteObject"],
     "Resource": ["arn:aws:s3:::atonixcorp-bucket1/*"]}
  ]
}
```

Attach it to a dedicated application identity with the administrator's MinIO
tools, keep the bucket private, and test the policy against the deployed MinIO
version. Do not place administrator credentials in the application environment.
For rotation, create or update the application identity, deploy the new secret,
restart the API, verify storage health, then revoke the old credential.

The authenticated `POST /api/v1/files/upload` accepts multipart fields
`tenant_id` and `file`. Tenant owners, admins and deployers can upload their
own private files, up to 10 MiB. Allowed detected types are JPEG, PNG, GIF,
WebP, PDF, plain text, ZIP, gzip and generic binary; the declared MIME type
must match the server's content sniffing. Content sniffing is not malware
scanning or full file-format validation. Filenames must be
ASCII letters, digits, dot, underscore or hyphen, beginning with a letter or
digit. Uploads use `uploads/{tenant_id}/{user_id}/{uuid}/{filename}`.
The response contains the database object reference and SHA-256 checksum, not a
filesystem path. `GET /api/v1/files/{id}/download` streams a private file;
`GET /api/v1/files/{id}/download-url` issues a five-minute bearer URL; and
`DELETE /api/v1/files/{id}` deletes it. Access requires both ownership and
current tenant membership; deletion also requires a write role. Presigned URLs
must be treated as secrets and cannot be revoked before expiry. The configured
MinIO endpoint must be reachable by the URL recipient for presigned downloads.
Only TLS-protected clients should receive these URLs in production.

### Browser-direct uploads

The Storage page's **Object storage** tab lists the current user's private
objects and uploads directly to MinIO. The browser computes SHA-256, then calls
`POST /api/v1/files/direct-uploads` with JSON containing `tenant_id`,
`category`, `filename`, `content_type`, `size_bytes`, `checksum` (lowercase
hex SHA-256), and the category's fields:

| Category | Additional fields |
| --- | --- |
| uploads | none |
| images | `subcategory` |
| documents | `subcategory` (document type) |
| backups, logs | `service` |
| datasets | `project`, `dataset`, `version` |
| models | `project`, `model`, `version` |
| users | `subcategory` |

These are validated key labels, not grants to actual project resources.
Dataset and model project labels are automatically scoped to tenant and owner.
All objects are private and readable only by their owner while they remain a
tenant member. Upload and delete require an owner, admin or deployer tenant role;
creating backups and logs also requires platform super-admin status. The
backend generates unique keys using the shared conventions.

The authorization response contains `object_id`, `expires_at`, and `upload`
with a URL and signed form fields. POST the form fields and a final `file`
field to that URL, then call
`POST /api/v1/files/direct-uploads/{object_id}/complete`. The signed form
enforces the declared exact size, with a maximum of 100 MiB, and expires after
15 minutes. Completion checks membership again, verifies the size, content type
and SHA-256 by streaming the object, and conditionally copies the verified
staging object to its private final key before making metadata active.
`GET /api/v1/files/objects?tenant_id=1&limit=50&offset=0` lists the owner's
active objects (maximum limit 100). The web client hashes files in browser
memory; completion reads the object through the API for verification, even
though the initial upload bypasses the API.

Configure MinIO CORS for the exact public web origin and POST, and ensure the
same TLS-enabled `MINIO_ENDPOINT` is reachable and trusted from both the API and
browser. On MinIO versions that support the server-wide setting, configure
`MINIO_API_CORS_ALLOW_ORIGIN=https://app.example.com` on the **MinIO server**
(replace the example origin), restart it, and verify an OPTIONS preflight from
both the permitted and an unpermitted origin. This setting affects all buckets
on that server. Bucket-level `mc cors set` is not supported by every MinIO
version; use it only if your deployment supports it. Avoid wildcard origins in
production.

The signed POST is a bearer capability and can be replayed until expiry, but
only against its staging key; it cannot overwrite the final object. The API
cleans up staging objects after expiry. As a backstop for crashes, late
in-flight uploads and replays, configure bucket lifecycle expiration for
`staging/`, including noncurrent versions if versioning is enabled. For
example, `mc ilm rule add --prefix staging/ --expire-days 1
ADMIN_ALIAS/atonixcorp-bucket1` applies to current objects; configure
noncurrent version cleanup separately as appropriate. Apply bucket quotas and
monitor storage usage. The in-process API rate limit is not a distributed
quota. Delete creates a delete marker on versioned buckets; retention and
permanent removal of older versions require separate operator policies.

Metadata and access audit events are stored in the database; object bytes are
stored in MinIO. Uploads are recorded as `pending` before provider writes and
become `active` after verification. Expired direct uploads are marked `expired`.
Interrupted operations can leave `pending`, `verifying` or `deleting` records
for operator reconciliation: inspect the object and audit records before
completing or removing them. Object storage can
be unavailable without taking down unrelated API functionality. Platform
super admins can check `GET /api/v1/admin/health/storage`; `/health` remains
the application liveness check.

The provider interface also supports stat, list, copy and presigned put for
trusted backend services. The shared object-key builder defines conventions for
uploads, images, documents, backups, logs, datasets, models and users; these
prefixes do not grant authorization. Do not use the low-level provider directly
in a business service without implementing its metadata, validation,
authorization and audit flow.

### Legacy local file import

Legacy files are never served automatically because the old records did not
include owners or tenants. After reviewing ownership, prepare an operator-only
CSV with a header and one mapping per row:

```csv
legacy_file_id,owner_id,tenant_id
1,123,45
```

Run the API database migrations by starting the API first. Give the import
command read-only access to the historical API working directory (which
contains `uploads/`), configure the same database and object-storage environment,
and run:

```sh
/app/migrate-files -legacy-root /path/to/old-api-working-directory \
  -mapping /path/to/reviewed-mapping.csv -actor-id PLATFORM_SUPER_ADMIN_ID
```

The command is included in the API image; mount the legacy directory and CSV
read-only when running it in a container. The selected owner must currently have
a write role in the mapped tenant. Imports stream the source file, verify its
stored size and declared content type, record SHA-256 and the administrative
actor, and leave the original intact. Only regular files under `uploads/` are
accepted. Unsupported MIME types, missing files, inconsistent metadata and
failed or conflicting prior imports stop the command for manual review.
Re-running the same successful mapping is idempotent. Inspect and reconcile
pending imports before retrying; never infer or bulk-assign unknown ownership.

To run the MinIO integration test against a disposable, pre-provisioned bucket,
configure the variables above with a dedicated test identity and run
`MINIO_INTEGRATION_TEST=true go test -count=1 ./internal/objectstorage/minio`
from `api/`. The test writes and deletes test objects in all eight prefixes,
tests the private upload metadata and signed download workflow, and checks
provider listing, copy, stat, existence and deletion. It is not a substitute
for validating production TLS, bucket policy, network reachability, and the
actual deployed database and API.

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
Storage API still does not implement snapshots, block-volume backups, file
shares, compute attachment, resizing, or provider capacity/performance
guarantees. Cinder encryption is selected by the configured volume type rather
than by claiming that the local driver encrypts data.
