# Container registries

Registry records are scoped to a tenant and project. The authenticated API
provides:

- `GET /api/v1/container-registries?tenant_id=<id>&project_id=<id>`
- `POST /api/v1/container-registries`

Create requests accept `tenant_id`, `project_id`, `name`, an HTTPS `url`, and
optional `username` and `password` fields. Credentials must be provided as a
pair. URLs containing user information, query parameters, or fragments are
rejected. The API encrypts passwords with `CONTAINER_SECRET_KEY`; responses
only indicate whether credentials are stored and never return username or
password fields.

These records are not yet wired to Kubernetes `imagePullSecrets`, registry
catalog/image enumeration, registry health checks, or credential rotation.
Private image deployment must not be presented as supported until that
integration is complete.
