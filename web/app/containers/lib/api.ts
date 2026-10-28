export type Tenant = { id: number; name: string; slug: string };
export type Project = { id: number; tenant_id: number; name: string; slug: string };
export type ContainerRegistry = {
  id: number;
  tenant_id: number;
  project_id: number;
  name: string;
  url: string;
  credentials_stored: boolean;
  created_at: string;
};
export type ContainerPort = { name?: string; container_port: number; service_port?: number; protocol?: "TCP" | "UDP" };
export type ContainerVolume = {
  name: string;
  size_gib: number;
  mount_path: string;
  storage_class?: string;
  access_mode?: string;
};
export type ContainerEnvironmentVariable = { name: string; value: string; secret: boolean };
export type ContainerProbe = {
  path: string;
  port: number;
  initial_delay_seconds?: number;
  period_seconds?: number;
  timeout_seconds?: number;
  failure_threshold?: number;
};
export type ContainerDeploymentSpec = {
  name: string;
  description?: string;
  environment: "development" | "staging" | "production";
  image: string;
  image_pull_policy?: "Always" | "IfNotPresent" | "Never";
  replicas: number;
  cpu_request: string;
  cpu_limit: string;
  memory_request: string;
  memory_limit: string;
  ports: ContainerPort[];
  environment_variables: ContainerEnvironmentVariable[];
  volumes: ContainerVolume[];
  retain_volumes?: boolean;
  readiness_probe?: ContainerProbe;
  liveness_probe?: ContainerProbe;
};
export type ContainerDeployment = {
  id: string;
  tenant_id: number;
  project_id: number;
  name: string;
  state: string;
  current_revision: number;
  desired_replicas: number;
  ready_replicas: number;
  spec: ContainerDeploymentSpec;
  endpoint?: string;
  correlation_id: string;
  provisioning_job_id: string;
  error_code?: string;
  error_details?: string;
  created_at: string;
  updated_at: string;
};
export type ContainerRevision = {
  revision: number;
  spec_hash: string;
  actor_id: number;
  created_at: string;
  spec: ContainerDeploymentSpec;
};
export type ContainerMetrics = {
  cpu_usage_millis: number;
  memory_usage_bytes: number;
  sampled_at: string;
};

async function requestJSON<T>(path: string, init?: RequestInit): Promise<T> {
  const response = await fetch(path, {
    ...init,
    cache: "no-store",
    headers: {
      ...(init?.body ? { "Content-Type": "application/json" } : {}),
      ...init?.headers,
    },
  });
  const result = await response.json().catch(() => null) as { error?: string } | T | null;
  if (!response.ok) {
    const message = result && typeof result === "object" && "error" in result
      ? result.error
      : `Container request failed (${response.status}).`;
    throw new Error(message || `Container request failed (${response.status}).`);
  }
  if (result === null) throw new Error("Container API returned an empty response.");
  return result as T;
}

export function getTenants(): Promise<Tenant[]> {
  return requestJSON("/api/containers/tenants");
}

export function getProjects(tenantID: number): Promise<Project[]> {
  return requestJSON(`/api/containers/tenants/${tenantID}/projects`);
}

export function getRegistries(tenantID: number, projectID: number): Promise<ContainerRegistry[]> {
  const query = new URLSearchParams({ tenant_id: String(tenantID), project_id: String(projectID) });
  return requestJSON(`/api/containers/container-registries?${query}`);
}

export function createRegistry(
  registry: Pick<ContainerRegistry, "tenant_id" | "project_id" | "name" | "url"> & { username?: string; password?: string },
): Promise<ContainerRegistry> {
  return requestJSON("/api/containers/container-registries", {
    method: "POST",
    body: JSON.stringify(registry),
  });
}

export function getDeployments(tenantID: number, projectID: number): Promise<ContainerDeployment[]> {
  const query = new URLSearchParams({ tenant_id: String(tenantID), project_id: String(projectID) });
  return requestJSON(`/api/containers/container-deployments?${query}`);
}

export function getDeployment(id: string): Promise<ContainerDeployment> {
  return requestJSON(`/api/containers/container-deployments/${encodeURIComponent(id)}`);
}

export function getDeploymentEvents(id: string): Promise<{
  audit: Array<Record<string, unknown>> | null;
  runtime: Array<{ type: string; reason: string; message: string; lastTimestamp: string }> | null;
}> {
  return requestJSON(`/api/containers/container-deployments/${encodeURIComponent(id)}/events`);
}

export function getDeploymentLogs(id: string): Promise<string> {
  return fetch(`/api/containers/container-deployments/${encodeURIComponent(id)}/logs`, {
    cache: "no-store",
  }).then(async (response) => {
    const body = await response.text();
    if (!response.ok) throw new Error(body || `Container logs failed (${response.status}).`);
    return body;
  });
}

export function getDeploymentInstances(id: string): Promise<Array<{
  name: string;
  phase: string;
  node: string;
  created_at: string;
  restarts: number;
  ready: boolean;
}>> {
  return requestJSON(`/api/containers/container-deployments/${encodeURIComponent(id)}/instances`);
}

export function getDeploymentRevisions(id: string): Promise<ContainerRevision[]> {
  return requestJSON(`/api/containers/container-deployments/${encodeURIComponent(id)}/revisions`);
}

export function getDeploymentMetrics(id: string): Promise<ContainerMetrics> {
  return requestJSON(`/api/containers/container-deployments/${encodeURIComponent(id)}/metrics`);
}

export function createDeployment(
  tenantID: number,
  projectID: number,
  spec: ContainerDeploymentSpec,
  idempotencyKey: string,
): Promise<ContainerDeployment> {
  return requestJSON("/api/containers/container-deployments", {
    method: "POST",
    headers: { "Idempotency-Key": idempotencyKey },
    body: JSON.stringify({ tenant_id: tenantID, project_id: projectID, spec }),
  });
}

export function operateDeployment(
  id: string,
  action: "start" | "stop" | "restart" | "scale" | "redeploy" | "rollback",
  replicas?: number,
  revision?: number,
): Promise<ContainerDeployment> {
  return requestJSON(`/api/containers/container-deployments/${encodeURIComponent(id)}/${action}`, {
    method: "POST",
    body: action === "scale"
      ? JSON.stringify({ replicas })
      : action === "rollback" ? JSON.stringify({ revision }) : undefined,
  });
}

export function updateDeployment(
  id: string,
  revision: number,
  spec: ContainerDeploymentSpec,
): Promise<ContainerDeployment> {
  return requestJSON(`/api/containers/container-deployments/${encodeURIComponent(id)}`, {
    method: "PATCH",
    headers: { "If-Match": String(revision) },
    body: JSON.stringify({ spec }),
  });
}

export function deleteDeployment(id: string): Promise<ContainerDeployment> {
  return requestJSON(`/api/containers/container-deployments/${encodeURIComponent(id)}`, {
    method: "DELETE",
  });
}
