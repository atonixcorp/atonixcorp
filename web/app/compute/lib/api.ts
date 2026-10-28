import type {
  ComputeCatalog,
  ComputeEstimate,
  ComputeFirewall,
  ComputeInstance,
  ComputeSSHKey,
  DeploymentDraft,
} from "./types";

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

export function getCatalog(): Promise<ComputeCatalog> {
  return requestJSON<ComputeCatalog>("/api/compute/catalog");
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

export function getInstances(): Promise<ComputeInstance[]> {
  return requestJSON<ComputeInstance[]>("/api/compute/instances");
}

export function getInstance(instanceId: string): Promise<ComputeInstance> {
  return requestJSON<ComputeInstance>(
    `/api/compute/instances/${encodeURIComponent(instanceId)}`,
  );
}

export function getEstimate(draft: DeploymentDraft): Promise<ComputeEstimate> {
  return requestJSON("/api/compute/estimate", {
    method: "POST",
    body: JSON.stringify({
      region: draft.region,
      compute_type: draft.computeType,
      plan_id: draft.plan,
      quantity: draft.quantity,
      volumes: draft.volumes,
    }),
  });
}

export function deployInstances(
  draft: DeploymentDraft,
  idempotencyKey: string,
): Promise<{ deployment_id: string; status: string; instances: ComputeInstance[] }> {
  return requestJSON("/api/compute/instances", {
    method: "POST",
    headers: { "Idempotency-Key": idempotencyKey },
    body: JSON.stringify({
      name: draft.name,
      hostname: draft.hostname,
      label: draft.label,
      region: draft.region,
      compute_type: draft.computeType,
      plan_id: draft.plan,
      image_id: draft.image,
      network_id: draft.network,
      ssh_key_ids: draft.sshKeyIds,
      firewall_id: draft.firewall,
      public_ipv4: draft.publicIPv4,
      public_ipv6: draft.publicIPv6,
      private_networking: draft.privateNetworking,
      hostname_dns: draft.hostnameDNS,
      user_data: draft.userData,
      tags: draft.tags,
      quantity: draft.quantity,
      volumes: draft.volumes,
    }),
  });
}
