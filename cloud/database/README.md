# AtonixCorp Database API

The database API is an authenticated service behind the web application's
`/api/databases` proxy. It stores database requests in the shared PostgreSQL
database, checks project membership using the existing `projects` and
`project_memberships` tables, and provisions Nova servers with Cinder data
volumes when OpenStack is configured.

## Run locally with Compose

Set the repository's existing required `JWT_SECRET`, `COMPUTE_API_TOKEN`, and
`COMPUTE_NODE_TOKEN` values, then start the stack with:

```sh
docker compose up -d --build
```

The database API listens on port `8085` inside the Compose network and is not
published directly to the host. The dashboard is available at
`http://localhost:3000/databases` after signing in. The service uses
`ATONIXCORP_DB_DSN`, or the standard `DB_HOST`, `DB_USER`, `DB_PASSWORD`,
`DB_NAME`, and `DB_PORT` settings, to connect to the shared PostgreSQL
database.

Without OpenStack configuration, the API still starts and serves its catalog
and existing database inventory, but database creation is disabled.

## OpenStack configuration

The service accepts the same Keystone application credential configured for
Nova. Credentials remain in the root `.env`, not YAML. Configure:

- `OPENSTACK_AUTH_URL`, `OPENSTACK_APPLICATION_CREDENTIAL_ID`, and
  `OPENSTACK_APPLICATION_CREDENTIAL_SECRET`
- `OPENSTACK_SHARED_PROJECT_ID`, `OPENSTACK_REGION`, and `OPENSTACK_INTERFACE`
- `OPENSTACK_ALLOW_INSECURE_HTTP=true` only for isolated DevStack development
- `OPENSTACK_DATABASE_NETWORK_ID` and optionally
  `OPENSTACK_DATABASE_SECURITY_GROUP_ID`
- `OPENSTACK_DATABASE_FLAVOR_BASIC`, `OPENSTACK_DATABASE_FLAVOR_STANDARD`,
  and `OPENSTACK_DATABASE_FLAVOR_PREMIUM`
- Prepared appliance image IDs, keyed by engine and version:
  `OPENSTACK_DATABASE_IMAGE_POSTGRES_16`,
  `OPENSTACK_DATABASE_IMAGE_POSTGRES_17`,
  `OPENSTACK_DATABASE_IMAGE_MYSQL_8_4`,
  `OPENSTACK_DATABASE_IMAGE_MARIADB_11_4`, and
  `OPENSTACK_DATABASE_IMAGE_REDIS_7`

Check the project's provider-side options with `openstack --os-cloud atonixcorp
flavor list`, `openstack --os-cloud atonixcorp network list --project
<project-id>`, and `openstack --os-cloud atonixcorp image list`. Configure a
prepared, tested database appliance image for each enabled engine/version;
ordinary cloud images are not database appliances and will not produce a
ready-to-use managed database. Configure `OPENSTACK_DATABASE_NETWORK_ID` and
the flavor/image IDs in the root `.env` before exposing database creation.

The service discovers Nova and Cinder endpoints from Keystone's service
catalog; explicit `OPENSTACK_COMPUTE_ENDPOINT` and
`OPENSTACK_VOLUME_ENDPOINT` values may override discovery. The catalog only
enables engines and plans for which the corresponding prepared image, flavor,
network, and OpenStack client configuration are present. All AtonixCorp
database projects use the same OpenStack project and therefore share its
provider-level quota and security boundary.

## Current capability boundary

Provisioning requests require an `Idempotency-Key`. The database row,
operation, initial lifecycle event, outbox entry, and scoped idempotency key
are persisted together. Replaying a key with the same request returns the
original resource; reusing it for a different request returns HTTP 409.
Operations expose a status resource and a bounded server-sent event stream.

Provisioning persists Nova/Cinder resource IDs before continuing. A request
can remain in `provisioning` while OpenStack resources are building, then moves
to `starting` after the VM and data volume are available. `starting` is
deliberately not reported as healthy: appliance bootstrap, engine health
checks, credential/secret delivery, backups, monitoring, replicas, and
deletion/rollback workflows are not implemented. Do not use the reported VM
state as proof that a database is ready for connections. Selecting the
OpenHPC workload profile is outside the scope of this OpenStack integration;
OpenHPC is not provisioned or enabled by these changes.

Database inventory responses expose the provider-returned Nova server ID,
server status and fixed IP, plus the Cinder volume ID, status, attachment
state, and configured network ID. These fields describe infrastructure
resources only; a `starting` database is not a healthy database endpoint.
