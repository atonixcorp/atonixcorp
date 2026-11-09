# AtonixCorp development environment

The root Compose stack is the existing entry point for the cloud web app and
its core APIs. It does not orchestrate every project in this repository. The
root stack and selected standalone images were built and checked on this host
on 2026-11-09; see the standalone project status below for incomplete areas.

## Prerequisites

- Docker Engine with Docker Compose v2, permission to use the daemon, and
	network access to pull images and build dependencies.
- OpenSSL for generating development secrets. An SMTP server reachable from
	the API container is needed for email delivery.
- Optional: a reachable OpenStack deployment and a project-scoped application
	credential for Nova, Neutron, or Cinder. Kubernetes credentials are required
	for container deployment operations.

## Configuration

From the repository root, copy `.env.example` to `.env` and edit the copy:

```sh
cp .env.example .env
```

Replace every required placeholder. Generate independent random values for
`POSTGRES_PASSWORD`, `JWT_SECRET`, `COMPUTE_API_TOKEN`, `COMPUTE_NODE_TOKEN`,
and `AGENT_API_TOKEN` using `openssl rand -hex 32` separately for each value.
Generate `CONTAINER_SECRET_KEY` with `openssl rand -base64 32`. Keep these
values private and never commit `.env`. Set `SMTP_HOST`, `SMTP_PORT`,
`SMTP_USER`, `SMTP_PASSWORD`, and `SMTP_RECIPIENT` to values for a real SMTP
service; placeholders allow configuration loading but do not deliver mail.
`SPONSOR_RECIPIENT` is optional. `AUTH_COOKIE_SECURE=false` is needed for
local HTTP; use secure cookies with HTTPS outside local development.

Compose reads root `.env` for interpolation and passes it to the API. The web
service receives explicit server-side URLs and the agent token. Use Docker DNS
names such as `api` and `agent` inside containers, not host `localhost`.
The published ports below bind to host loopback only. These development
settings and secrets are not suitable for staging or production. Existing
PostgreSQL volumes retain their original password when `.env` changes.

`STORAGE_DRIVER=local-file` uses development-only sparse files; it is not an
attached or encrypted block device. For OpenStack or libvirt storage, see
[storage configuration](api/docs/storage.md). Set `CONTAINER_KUBECONFIG` to
an existing host kubeconfig for Kubernetes; without one, container deployments
are read-only.

## Start and stop

```sh
docker compose up -d --build
docker compose ps
docker compose logs -f
docker compose down
```

Run these commands from the repository root. `down` preserves named volumes;
do not remove volumes unless you intend to delete development data. If a
standalone agent is already listening on port 8000, stop it before starting
the Compose agent. To run the agent alone without Docker, see
[agent instructions](atonixcorp-agent/README.md); its default loopback mode
has no authentication unless `AGENT_API_TOKEN` is set.

## Service registry

All Compose start commands are `docker compose up -d --build` from the root.
Internal URLs are reachable only from the Compose network, not the browser.
The status column describes the last check on this host, not a guarantee of
current health.

| Service | URL | Port | Dependencies | Last checked | Start command |
| --- | --- | --- | --- | --- | --- |
| Web | http://localhost:3000 | 3000 | API, database API, compute, networking; agent proxy uses agent | HEALTHY | Root Compose |
| Main API, auth, storage | http://localhost:8080 | 8080 | PostgreSQL; SMTP and infrastructure integrations as configured | HEALTHY | Root Compose |
| PostgreSQL | `127.0.0.1:5433` | 5433 host / 5432 internal | Persistent volume | HEALTHY | Root Compose |
| Database API | `http://database:8085` | 8085 internal | PostgreSQL, main API; OpenStack for provisioning | HEALTHY | Root Compose |
| Networking API | `http://networking:8080` | 8080 internal | PostgreSQL, main API; Neutron optional | HEALTHY | Root Compose |
| Compute catalog API | `http://compute:8580` | 8580 internal | Persistent volume; node integration incomplete | HEALTHY | Root Compose |
| Agent API | http://localhost:8000/healthz | 8000 | Local JSON state | HEALTHY in Compose | Root Compose or `cd atonixcorp-agent && python3 -m api.server` |
| OpenStack Keystone | Deployment-specific; `http://192.168.178.131/identity/v3` on this host | 80 here | External DevStack | HTTP reachable only; auth unverified | External infrastructure |

The web app proxies database, networking, storage, compute, and agent requests
through same-origin `/api/...` routes. Auth is part of the main API, not a
separate authentication service. The storage worker runs in the main API;
container workers only start with a working Kubernetes provisioner.

## Standalone project status

These projects are not orchestrated by root Compose. Standalone containers
that are running bind their host ports to loopback.

| Project | Build | Runtime check | Status |
| --- | --- | --- | --- |
| Analytics | PASS | PASS | Running on `127.0.0.1:8090`; event ingestion persists to PostgreSQL and the usage endpoint reports stored events. Collector/ingestor binaries are built but not running as separate workers. |
| DevOps | PASS | PARTIAL | Running; all ten HTTP `/healthz` endpoints returned 200. gRPC APIs and external dependencies were not integration-tested. |
| Container runtime | PASS | PARTIAL | Running on `127.0.0.1:8082`; `GET /api/v1/nodes` responds. There is no health route or authentication, so do not expose it beyond loopback. |
| HPC database API | PASS | PARTIAL | Running on `127.0.0.1:8091`; dataset registration/listing responds. State is in-memory and endpoints do not authenticate callers. |
| HPC service | FAIL | BLOCKED | The image does not build: the job/metadata repositories and Slurm discovery constructor are not implemented, and startup wiring has incomplete placeholders. |
| Emailserver | PASS | BLOCKED | Image builds, but no container is running. PostgreSQL and MinIO are not provisioned here; SMTP TLS files must be mounted under `/run/secrets/emailserver`. A private key was removed from the working tree; rotate it and purge it from Git history if it was ever used. |
| DevTools portal | BLOCKED | BLOCKED | The API/backend build contexts have no Dockerfiles, and the frontend package manifest and HTML entrypoint are empty. Its Compose file also claims host port 80, which is already occupied. |
| CLI | PASS | N/A | Image builds; this is an on-demand CLI, not a daemon. |

## Health and verification

`docker compose ps` reports container state and available healthchecks.
Interpret **RUNNING** as a process started, **HEALTHY** as its configured
probe passing, **DEGRADED** as a running service with unavailable or disabled
dependencies, and **DOWN** as stopped or unreachable. There is no unified
degraded-status endpoint yet. The main API and agent health endpoints are
process checks and do not establish end-to-end dependency health.

```sh
curl -fsS http://localhost:8080/health
curl -fsS http://localhost:8000/healthz
curl -fsS http://localhost:3000/api/agent/registry
docker compose exec -T db pg_isready -U atonixcorp -d atonixcorp
docker compose exec -T database /usr/local/bin/atdb-health
docker compose exec -T compute wget -qO- http://127.0.0.1:8580/health
docker compose exec -T networking wget -qO- http://127.0.0.1:8080/health
```

The configured health checks, internal API checks, web authentication flow,
agent proxy, and scoped storage smoke test passed on this host. The full API Go
test suite also passed. A successful agent proxy response checks the
web-to-agent path, not model readiness. An empty registry is valid.

The CI workflow builds and starts the root Compose stack with short-lived,
random development credentials, checks each container healthcheck and internal
API, then exercises the web signup/login/session/logout flow and agent proxy.
To run the same HTTP/authentication smoke test against a locally started stack:

```sh
python3 scripts/compose-smoke-test.py
```

The smoke test creates a uniquely named ordinary test user. CI also bootstraps
and tests a Super Admin using disposable credentials. The persistent root
database has not been bootstrapped with a Super Admin. To create its first
account, start the services and run this command from an interactive terminal:

```sh
docker compose exec api /app/atonix-admin
```

The command prompts for name, email, and a twice-entered password without
echoing it. It only creates the first Super Admin, stores a bcrypt hash, and
rejects later bootstrap attempts. Super Admin status is stored in PostgreSQL
and checked on every authenticated API request, so role changes take effect
immediately. A Super Admin can access valid resources across tenants and
projects; ordinary users remain membership-scoped. The platform user listing
and role-management endpoints are `GET /api/v1/admin/users` and
`PATCH /api/v1/admin/users/{userID}/platform-role`. Role changes are audited,
and the final Super Admin cannot demote themselves.
Logout clears the browser cookie. Bearer JWTs are stateless and remain valid
until their 24-hour expiry; server-side per-token revocation is not implemented.

## OpenStack and other integrations

Configure the backend-only OpenStack variables in `.env` as described in
[Nova setup](api/docs/openstack-compute.md), [database setup](cloud/database/README.md),
and [networking setup](cloud/network/README.md). Never put credentials in
browser variables. `OPENSTACK_ALLOW_INSECURE_HTTP=true` is only for an
isolated, trusted development network and transmits credentials without TLS.
Shared-project provisioning is explicitly opt-in and shares the provider's
quota and security boundary. An HTTP response from Keystone does not verify
credentials, service catalog discovery, or real provisioning.

The agent proxy is configured in root Compose, but prediction requires a
promoted model and real telemetry. The compute catalog is seed data and its
node provisioning is incomplete; see [compute status](cloud/compute/README.md).
Managed-database appliance bootstrap and several Neutron operations are not
implemented. Analytics, DevOps, email daemons, container runtime, and other
standalone projects are not included in root Compose; their observed status and
limitations are listed above. SMTP delivery, OpenStack/Neutron, Kubernetes, and
Slurm integrations remain unconfigured or unverified.
If the separate DevOps gateway's local authentication is enabled, configure
its local-admin username, email, and password out of band; it no longer has
default local-admin credentials.
The separate `cloud/devtools` stack and the HPC service are incomplete and
must not be treated as production-ready. OpenHPC is outside this setup.

## Troubleshooting

- `docker: command not found` or daemon unavailable: install/start Docker and
	Compose and grant the current user access, then rerun the startup command.
- Check `docker compose logs --tail=100 <service>` for startup, migration,
	authentication, or provider errors. Avoid sharing logs containing secrets.
- For occupied ports, stop the conflicting process or change the host port
	mappings consistently; an existing OpenStack web server may already use 80.
- A web proxy 502/503 can mean its backend is down or unconfigured. Check
	Compose health and the server-side service URL before changing browser URLs.
- For database authentication errors, check the configured password against
	the existing PostgreSQL volume; do not delete the volume as a first step.
- A healthy process can still have disabled OpenStack, Kubernetes, SMTP, or
	provisioning features. Verify each integration with its component guide.
