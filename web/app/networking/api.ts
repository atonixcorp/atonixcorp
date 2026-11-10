export type Network = {
  id: string;
  name: string;
  cidr: string;
  status?: string;
  subnet_status?: string;
};

export type Capabilities = { network_provisioning: boolean; resources: string[] };

export async function requestJSON<T>(path: string, init?: RequestInit): Promise<T> {
  const response = await fetch(path, {
    ...init,
    cache: "no-store",
    headers: {
      ...(init?.body ? { "Content-Type": "application/json" } : {}),
      ...init?.headers,
    },
  });
  const payload: unknown = await response.json().catch(() => null);
  if (!response.ok) {
    const message = payload && typeof payload === "object" && "error" in payload
      ? String(payload.error)
      : `Networking request failed (${response.status}).`;
    throw Object.assign(new Error(message), { status: response.status });
  }
  if (payload === null) throw new Error("Networking service returned an empty response.");
  return payload as T;
}

export function getNetworks(tenantID: number, projectID: number): Promise<Network[]> {
  return requestJSON(`/api/networking/tenants/${tenantID}/projects/${projectID}/networks`);
}

export type FirewallGroup = {
  name: string;
  description: string;
  rules: Array<{
    direction: string;
    ethertype: string;
    protocol: string;
    port_range_min?: number;
    port_range_max?: number;
    remote_ip_prefix?: string;
  }>;
};

export function getFirewallGroups(tenantID: number, projectID: number): Promise<FirewallGroup[]> {
  const query = new URLSearchParams({ tenant_id: String(tenantID), project_id: String(projectID) });
  return requestJSON(`/api/compute/firewalls?${query}`);
}
