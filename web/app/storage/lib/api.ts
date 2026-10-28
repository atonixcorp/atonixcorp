export type StorageTenant = { id: number; name: string; slug: string };
export type StorageProject = { id: number; tenant_id: number; name: string; slug: string };
export type StorageVolumeType = {
  id: string;
  name: string;
  driver: string;
  region: string;
  availability_zones: string[];
  encrypted: boolean;
};
export type StorageOverview = {
  driver: string;
  tags: Record<string, string>;
  volumes: number;
  used_gb: number;
  quota_gb: number;
  volume_quota: number;
};
export type StorageVolume = {
  id: string;
  tenant_id: number;
  project_id: number;
  name: string;
  description?: string;
  region: string;
  availability_zone?: string;
  storage_class: string;
  size_gb: number;
  encrypted: boolean;
  delete_protection: boolean;
  state: string;
  driver: string;
  error_code?: string;
  created_at: string;
  updated_at: string;
};
export type StorageOperation = {
  id: string;
  resource_id: string;
  resource_type: string;
  action: string;
  status: string;
  attempts: number;
  error_code?: string;
  created_at: string;
  completed_at?: string;
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
    | { error?: string }
    | T
    | null;
  if (!response.ok) {
    const error = payload && typeof payload === "object" && "error" in payload
      ? payload.error
      : undefined;
    throw new Error(error || `Storage request failed (${response.status}).`);
  }
  if (payload === null) throw new Error("Storage API returned an empty response.");
  return payload as T;
}

export function getTenants(): Promise<StorageTenant[]> {
  return requestJSON("/api/storage/tenants");
}

export function getProjects(tenantID: number): Promise<StorageProject[]> {
  return requestJSON(`/api/storage/tenants/${tenantID}/projects`);
}

export function getVolumeTypes(): Promise<StorageVolumeType[]> {
  return requestJSON("/api/storage/volume-types");
}

function scopeQuery(tenantID: number, projectID: number): string {
  return new URLSearchParams({
    tenant_id: String(tenantID),
    project_id: String(projectID),
  }).toString();
}

export function getOverview(tenantID: number, projectID: number): Promise<StorageOverview> {
  return requestJSON(`/api/storage/overview?${scopeQuery(tenantID, projectID)}`);
}

export function getVolumes(tenantID: number, projectID: number): Promise<StorageVolume[]> {
  return requestJSON(`/api/storage/volumes?${scopeQuery(tenantID, projectID)}`);
}

export function createVolume(
  request: {
    tenant_id: number;
    project_id: number;
    name: string;
    description: string;
    region: string;
    availability_zone: string;
    storage_class: string;
    size_gb: number;
    encrypted: boolean;
    delete_protection: boolean;
    tags: Record<string, string>;
  },
): Promise<{ volume: StorageVolume; operation: StorageOperation }> {
  return requestJSON("/api/storage/volumes", {
    method: "POST",
    headers: { "Idempotency-Key": crypto.randomUUID() },
    body: JSON.stringify(request),
  });
}

export function getOperation(id: string): Promise<StorageOperation> {
  return requestJSON(`/api/storage/operations/${encodeURIComponent(id)}`);
}

export function getOperations(tenantID: number, projectID: number): Promise<StorageOperation[]> {
  return requestJSON(`/api/storage/operations?${scopeQuery(tenantID, projectID)}`);
}

export function deleteVolume(id: string): Promise<StorageOperation> {
  return requestJSON(`/api/storage/volumes/${encodeURIComponent(id)}`, {
    method: "DELETE",
    headers: { "Idempotency-Key": crypto.randomUUID() },
  });
}
