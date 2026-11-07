import type {
  ComputeCatalog,
  ComputeFirewall,
  ComputeInstance,
  ComputeSSHKey,
} from "./types";

type NovaCloudInstance = {
  id: string;
  provider_id: string;
  provider_status: string;
  name: string;
  region: string;
  image_id: string;
  flavor_id: string;
  network_id: string;
  key_name?: string;
  status: string;
  private_ip?: string;
  public_ip?: string;
  created_at: string;
  updated_at: string;
};

type OpenStackCatalogResponse = {
  region: string;
  provisioning_enabled: boolean;
  capabilities: { compute: boolean; image: boolean; network: boolean };
  images: Array<{ id: string; name: string; status: string; min_disk_gb: number; min_memory_mb: number }>;
  flavors: Array<{
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
export type NovaCreateRequest = {
  tenant_id: number;
  project_id: number;
  name: string;
  region: string;
  image_id: string;
  flavor_id: string;
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
    throw new Error(error || `Compute request failed (${response.status}).`);
  }
  if (payload === null) {
    throw new Error("The Compute API returned an empty response.");
  }
  return payload as T;
}

export async function getCatalog(tenantID: number, projectID: number): Promise<ComputeCatalog> {
  const query = new URLSearchParams({ tenant_id: String(tenantID), project_id: String(projectID) });
  const live = await requestJSON<OpenStackCatalogResponse>(`/api/compute/catalog?${query}`);
  const available = Boolean(
    live.provisioning_enabled && live.capabilities.compute &&
    live.capabilities.image && live.capabilities.network,
  );
  return {
    provisioning_enabled: available,
    pricing_available: false,
    compute_types: live.capabilities.compute
      ? [{ id: "openstack", name: "OpenStack", description: "Provider-backed virtual machine" }]
      : [],
    regions: live.region ? [{
      id: live.region,
      name: live.region,
      country: "OpenStack",
      available,
      capacity_status: available ? "provider_catalog_available" : "provider_capabilities_unavailable",
    }] : [],
    images: live.images.map((image) => ({
      id: image.id,
      name: image.name || image.id,
      status: image.status,
      min_disk_gb: image.min_disk_gb,
      min_memory_mb: image.min_memory_mb,
      available: image.status === "ACTIVE",
    })),
    plans: live.flavors.map((flavor) => ({
      id: flavor.id,
      name: flavor.name || flavor.id,
      family: "OpenStack flavors",
      compute_type: "openstack",
      region: live.region,
      vcpus: flavor.vcpus,
      memory_mb: flavor.memory_mb,
      gpu_count: undefined,
      boot_disk_gb: flavor.disk_gb,
      ephemeral_disk_gb: flavor.ephemeral_gb,
      swap_mb: flavor.swap_mb,
      hourly_price: null,
      monthly_price: null,
      currency: "",
      available: true,
    })),
    networks: live.networks.map((network) => ({
      id: network.id,
      name: network.name || network.id,
      region: live.region,
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
  return requestJSON<NovaCloudInstance>(
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
  return requestJSON<NovaCloudInstance[]>(`/api/compute/instances?${query}`)
    .then((instances) => instances.map(toComputeInstance));
}

export function getNovaCapabilities(tenantID: number): Promise<{
  organization_provisioning: boolean;
  region: string;
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
    region: string;
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

export function createNovaInstance(
  request: NovaCreateRequest,
  idempotencyKey: string,
): Promise<ComputeInstance> {
  return requestJSON<NovaCloudInstance>("/api/compute/instances", {
    method: "POST",
    headers: { "Idempotency-Key": idempotencyKey },
    body: JSON.stringify(request),
  }).then(toComputeInstance);
}

export function operateInstance(
  instanceID: string,
  action: "start" | "stop" | "reboot",
): Promise<ComputeInstance> {
  return requestJSON<NovaCloudInstance>(
    `/api/compute/instances/${encodeURIComponent(instanceID)}?action=${action}`,
    { method: "POST" },
  ).then(toComputeInstance);
}

export function deleteInstance(instanceID: string): Promise<{ status: string }> {
  return requestJSON<{ status: string }>(`/api/compute/instances/${encodeURIComponent(instanceID)}`, {
    method: "DELETE",
  });
}

function toComputeInstance(instance: NovaCloudInstance): ComputeInstance {
  const status = normalizeInstanceStatus(instance.status);
  return {
    id: instance.id,
    provider_id: instance.provider_id,
    provider_status: instance.provider_status,
    deployment_id: instance.id,
    name: instance.name,
    hostname: instance.name,
    label: instance.name,
    region: instance.region,
    compute_type: "openstack",
    plan_id: instance.flavor_id,
    image_id: instance.image_id,
    authentication_method: instance.key_name ? "ssh-key" : "provider-default",
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
