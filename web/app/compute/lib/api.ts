import type {
  ComputeCatalog,
  ComputeFirewall,
  ComputeInstance,
  ComputeSSHKey,
} from "./types";

type CloudInstanceResponse = {
  id: string;
  name: string;
  region_id: string;
  image_id: string;
  size_id: string;
  network_id: string;
  key_name?: string;
  status: string;
  private_ip?: string;
  public_ip?: string;
  created_at: string;
  updated_at: string;
};

type ComputeCatalogResponse = {
  region_id: string;
  provisioning_enabled: boolean;
  capabilities: { compute: boolean; image: boolean; network: boolean };
  images: Array<{ id: string; name: string; status: string; min_disk_gb: number; min_memory_mb: number }>;
  sizes: Array<{
    id: string;
    name: string;
    vcpus: number;
    memory_mb: number;
    disk_gb: number;
    ephemeral_gb: number;
    swap_mb: number;
  }>;
  networks: Array<{
    id: string;
    name: string;
    status: string;
    admin_state_up: boolean;
    shared: boolean;
    external: boolean;
  }>;
};

export type ComputeTenant = { id: number; name: string };
export type ComputeProject = { id: number; tenant_id: number; name: string };
export type CreateInstanceRequest = {
  tenant_id: number;
  project_id: number;
  name: string;
  region_id: string;
  image_id: string;
  size_id: string;
  network_id: string;
  key_name?: string;
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
  const payload = (await response.json().catch(() => null)) as
    | { error?: string; code?: string }
    | T
    | null;
  if (!response.ok) {
    const error = payload && typeof payload === "object" && "error" in payload
      ? payload.error
      : undefined;
    const requestError = new Error(error || `Compute request failed (${response.status}).`) as Error & { status: number };
    requestError.status = response.status;
    throw requestError;
  }
  if (payload === null) {
    throw new Error("The Compute API returned an empty response.");
  }
  return payload as T;
}

export async function getCatalog(tenantID: number, projectID: number): Promise<ComputeCatalog> {
  const query = new URLSearchParams({ tenant_id: String(tenantID), project_id: String(projectID) });
  const live = await requestJSON<ComputeCatalogResponse>(`/api/compute/catalog?${query}`);
  const available = Boolean(
    live.provisioning_enabled && live.capabilities.compute &&
    live.capabilities.image && live.capabilities.network,
  );
  return {
    provisioning_enabled: available,
    pricing_available: false,
    compute_types: live.capabilities.compute
      ? [{ id: "virtual-machine", name: "Virtual machine", description: "General-purpose virtual machine" }]
      : [],
    regions: live.region_id ? [{
      id: live.region_id,
      name: "Primary region",
      available,
      capacity_status: available ? "available" : "unavailable",
    }] : [],
    images: live.images.map((image) => ({
      id: image.id,
      name: image.name || "Unnamed image",
      status: image.status,
      min_disk_gb: image.min_disk_gb,
      min_memory_mb: image.min_memory_mb,
      available: image.status === "ACTIVE",
    })),
    plans: live.sizes.map((size) => ({
      id: size.id,
      name: size.name || "Standard size",
      family: "General purpose",
      compute_type: "virtual-machine",
      region: live.region_id,
      vcpus: size.vcpus,
      memory_mb: size.memory_mb,
      gpu_count: undefined,
      boot_disk_gb: size.disk_gb,
      ephemeral_disk_gb: size.ephemeral_gb,
      swap_mb: size.swap_mb,
      hourly_price: null,
      monthly_price: null,
      currency: "",
      available: true,
    })),
    networks: live.networks.map((network) => ({
      id: network.id,
      name: network.name || "Virtual network",
      region: live.region_id,
      status: network.status,
      shared: network.shared,
      external: network.external,
      available: network.admin_state_up && network.status !== "DOWN",
    })),
  };
}

export function getTenants(): Promise<ComputeTenant[]> {
  return requestJSON<ComputeTenant[]>("/api/compute/scope");
}

export function getProjects(tenantID: number): Promise<ComputeProject[]> {
  return requestJSON<ComputeProject[]>(`/api/compute/scope?tenant_id=${tenantID}`);
}

export async function getSecurityCatalog(): Promise<{
  sshKeys: ComputeSSHKey[];
  firewalls: ComputeFirewall[];
}> {
  const [sshKeys, firewalls] = await Promise.all([
    requestJSON<ComputeSSHKey[]>("/api/compute/ssh-keys"),
    requestJSON<ComputeFirewall[]>("/api/compute/firewalls"),
  ]);
  return { sshKeys, firewalls };
}

export function getInstance(instanceId: string): Promise<ComputeInstance> {
  return requestJSON<CloudInstanceResponse>(
    `/api/compute/instances/${encodeURIComponent(instanceId)}`,
  ).then(toComputeInstance);
}

export function getInstances(
  tenantID: number,
  projectID: number,
): Promise<ComputeInstance[]> {
  const query = new URLSearchParams({
    tenant_id: String(tenantID),
    project_id: String(projectID),
  });
  return requestJSON<CloudInstanceResponse[]>(`/api/compute/instances?${query}`)
    .then((instances) => instances.map(toComputeInstance));
}

export function getComputeCapabilities(tenantID: number): Promise<{
  organization_provisioning: boolean;
  services: {
    compute: boolean;
    network: boolean;
    image: boolean;
    blockStorage: boolean;
    identity: boolean;
    objectStorage: boolean;
    loadBalancer: boolean;
    dns: boolean;
    keyManagement: boolean;
    orchestration: boolean;
  };
}> {
  return requestJSON<{
    organization_provisioning: boolean;
    services: {
      compute: boolean;
      network: boolean;
      image: boolean;
      blockStorage: boolean;
      identity: boolean;
      objectStorage: boolean;
      loadBalancer: boolean;
      dns: boolean;
      keyManagement: boolean;
      orchestration: boolean;
    };
  }>(`/api/compute/instances/capabilities?tenant_id=${tenantID}`);
}

export function createInstance(
  request: CreateInstanceRequest,
  idempotencyKey: string,
): Promise<ComputeInstance> {
  return requestJSON<CloudInstanceResponse>("/api/compute/instances", {
    method: "POST",
    headers: { "Idempotency-Key": idempotencyKey },
    body: JSON.stringify(request),
  }).then(toComputeInstance);
}

export function operateInstance(
  instanceID: string,
  action: "start" | "stop" | "reboot",
): Promise<ComputeInstance> {
  return requestJSON<CloudInstanceResponse>(
    `/api/compute/instances/${encodeURIComponent(instanceID)}?action=${action}`,
    { method: "POST" },
  ).then(toComputeInstance);
}

export function deleteInstance(instanceID: string): Promise<{ status: string }> {
  return requestJSON<{ status: string }>(`/api/compute/instances/${encodeURIComponent(instanceID)}`, {
    method: "DELETE",
  });
}

function toComputeInstance(instance: CloudInstanceResponse): ComputeInstance {
  const status = normalizeInstanceStatus(instance.status);
  return {
    id: instance.id,
    deployment_id: instance.id,
    name: instance.name,
    hostname: instance.name,
    label: instance.name,
    region: "Primary region",
    compute_type: "virtual-machine",
    plan_id: instance.size_id,
    image_id: instance.image_id,
    authentication_method: instance.key_name ? "ssh-key" : "default",
    network_id: instance.network_id,
    ssh_key_ids: instance.key_name ? [instance.key_name] : [],
    public_ipv4: Boolean(instance.public_ip),
    public_ipv6: false,
    private_networking: true,
    hostname_dns: false,
    tags: [],
    volumes: [],
    status,
    provisioning: [],
    hourly_price: null,
    monthly_price: null,
    currency: "",
    private_ip: instance.private_ip,
    public_ip: instance.public_ip,
    created_at: instance.created_at,
    updated_at: instance.updated_at,
  };
}

function normalizeInstanceStatus(status: string): string {
  switch (status.toUpperCase()) {
    case "RUNNING": return "running";
    case "STOPPED": return "stopped";
    case "ERROR": return "failed";
    case "UNKNOWN": return "unknown";
    case "BUILD":
    case "PROVISIONING":
    case "STARTING":
    case "STOPPING":
    case "REBOOTING":
    case "RESIZING":
    case "DELETING": return "provisioning";
    default: return status.toLowerCase();
  }
}
