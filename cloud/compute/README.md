# Compute control-plane API

The compute module exposes a versioned HTTP API for compute catalogs and instance deployments. It stores deployment requests on disk and reports instances as `provisioning` until a compute node reports each completed provisioning step. It never marks an instance `running` merely because an API request was accepted.

## Run locally

From this directory, provide two independent bearer tokens of at least 32 characters and start the API:

```sh
export COMPUTE_API_TOKEN="$(openssl rand -hex 32)"
export COMPUTE_NODE_TOKEN="$(openssl rand -hex 32)"
go run ./cmd/atcompute-api
```

The API listens on `:8580` by default and stores instance records in `/var/lib/atonixcorp-compute/instances.json`. Override these paths with `COMPUTE_API_ADDR` and `COMPUTE_STORE_PATH`. Run the Docker image with a persistent volume mounted at `/var/lib/atonixcorp-compute`.

The root Docker Compose stack requires `COMPUTE_API_TOKEN` and `COMPUTE_NODE_TOKEN` in the root `.env` file. Generate separate values with `openssl rand -hex 32`, then start the stack with `docker compose up -d --build`. The Compose network exposes the Compute API only to the web service; its shared credentials are not passed to browser code.

The Docker build context is this directory:

```sh
docker build -t atonixcorp-compute .
```

## Endpoints

Catalog endpoints are read-only:

- `GET /api/v1/compute/types`
- `GET /api/v1/compute/regions`
- `GET /api/v1/compute/images?region=jhb-1`
- `GET /api/v1/compute/plans?region=jhb-1&type=general&compute_type=shared`
- `GET /api/v1/compute/networks?region=jhb-1`
- `GET /api/v1/compute/ssh-keys`
- `GET /api/v1/compute/firewalls`

Instance endpoints require `Authorization: Bearer $COMPUTE_API_TOKEN`:

- `GET /api/v1/compute/instances`
- `POST /api/v1/compute/instances`
- `GET /api/v1/compute/instances/{instanceID}`
- `GET /api/v1/compute/ssh-keys`
- `POST /api/v1/compute/ssh-keys`

Instance creation accepts a quantity of 1–50, validates the requested region, plan, image, network, SSH keys, firewall, tags, and volumes against the backend catalog, and returns `202 Accepted` with generated instance IDs. Names and hostnames receive `-01`, `-02`, and subsequent suffixes for multi-instance deployments. Prices in the response are calculated from backend plan prices and volume capacity.

SSH keys can be registered using an OpenSSH public key. The API validates the key, stores only its public representation, calculates a fingerprint, and rejects duplicate keys. SSH-key authentication is required; password provisioning is not implemented.

Provisioning updates require `Authorization: Bearer $COMPUTE_NODE_TOKEN`:

- `POST /api/v1/compute/instances/{instanceID}/status`

Node updates must complete the provisioning steps in order. A node may report `running` only after all steps, including `health_checks`, are complete and it supplies its node ID. An update may report `failed` with a failed step and message.

## Provisioning integration status

The dashboard uses the API for its public catalog and non-authoritative estimate. Its instance, SSH-key, and firewall routes remain unavailable until authenticated user and tenant authorization is implemented. The API and persistent deployment records are in place, but the compute scheduler, node registration/dispatch, network/volume provisioning, image installation, and a real hypervisor driver are not yet connected. Until that integration is implemented, accepted deployments correctly remain in `provisioning`; node status callbacks are the only way to advance or fail them. The current catalog and prices are backend seed data and must be replaced with actual region capacity and billing data before production use.

The legacy `cmd/atcompute` executable remains the node-agent entry point. The Docker image runs the new HTTP API entry point, `cmd/atcompute-api`.

These are shared service credentials, not end-user or tenant authentication. Keep the API on a trusted network and do not expose its bearer tokens to browser code. The web dashboard's server-side catalog proxy does not connect deployments or instance records to dashboard users; platform authentication and tenant authorization are required before those controls can safely be exposed to customers.

The KVM/libvirt lifecycle driver now invokes `virsh` for start, graceful shutdown, pause, reboot, and state lookup. It uses `LIBVIRT_URI` or defaults to `qemu:///system`; the node runtime must have `virsh` installed and permission to access that libvirt connection. This lifecycle driver does not create VM domains or provision images, disks, IP addresses, or firewalls.
