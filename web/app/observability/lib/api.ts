export type ObservabilityTenant = { id: number; name: string };
export type ObservabilityProject = { id: number; tenant_id: number; name: string };
export type Resource = {
  id: string;
  resource_type: string;
  provider: string;
  name: string;
  region?: string;
  project_id: number;
  status: string;
  monitoring_status: string;
  last_seen_at: string;
};
export type ResourceGroup = {
  kind: string;
  count: number;
  health: string;
  source: string;
};
export type Overview = {
  health: string;
  healthy_resources: number | null;
  warning_resources: number | null;
  critical_resources: number | null;
  unknown_health: number;
  active_alerts: number | null;
  telemetry_state: string;
  telemetry_freshness: string | null;
  resource_groups: ResourceGroup[];
  metric_availability: string;
};
export type Integration = {
  name: string;
  provider: string;
  state: string;
  description: string;
};
export type Metric = {
  name: string;
  value: number;
  unit: string;
  resource_id?: string;
  resource_type?: string;
  observed_at: string;
};
export type MetricsResponse = {
  resource_id: string;
  resource_type: string;
  state: string;
  metrics: Metric[];
};
export type EventRecord = {
  id: string;
  resource_id: string;
  source: string;
  severity: string;
  summary: string;
  created_at: string;
};
export type Availability = { state: string; reason: string };
export type Dashboard = {
  id: string;
  name: string;
  description: string;
  resource_kinds: string[];
};

async function requestJSON<T>(path: string): Promise<T> {
  const response = await fetch(path, { cache: "no-store" });
  const result = await response.json().catch(() => null) as { error?: string; code?: string } | T | null;
  if (!response.ok) {
    const message = result && typeof result === "object" && "error" in result
      ? result.error
      : `Observability request failed (${response.status}).`;
    const error = new Error(message || `Observability request failed (${response.status}).`);
    Object.assign(error, { status: response.status });
    throw error;
  }
  if (result === null) throw new Error("Observability API returned an empty response.");
  return result as T;
}

export function getTenants(): Promise<ObservabilityTenant[]> {
  return fetch("/api/storage/tenants", { cache: "no-store" }).then(async (response) => {
    const result = await response.json().catch(() => null) as { error?: string } | ObservabilityTenant[] | null;
    if (!response.ok || !Array.isArray(result)) {
      throw new Error(result && !Array.isArray(result) && result.error
        ? result.error
        : "Workspace list could not be loaded.");
    }
    return result;
  });
}

export function getProjects(tenantID: number): Promise<ObservabilityProject[]> {
  return fetch(`/api/storage/tenants/${tenantID}/projects`, { cache: "no-store" }).then(async (response) => {
    const result = await response.json().catch(() => null) as { error?: string } | ObservabilityProject[] | null;
    if (!response.ok || !Array.isArray(result)) {
      throw new Error(result && !Array.isArray(result) && result.error
        ? result.error
        : "Project list could not be loaded.");
    }
    return result;
  });
}

function scopePath(tenantID: number, projectID: number, suffix: string): string {
  return `/api/observability/tenants/${tenantID}/projects/${projectID}/${suffix}`;
}

export const getOverview = (tenantID: number, projectID: number) =>
  requestJSON<Overview>(scopePath(tenantID, projectID, "overview"));
export const getResources = (tenantID: number, projectID: number) =>
  requestJSON<Resource[]>(scopePath(tenantID, projectID, "resources"));
export const getIntegrations = (tenantID: number, projectID: number) =>
  requestJSON<Integration[]>(scopePath(tenantID, projectID, "integrations"));
export const getMetrics = (tenantID: number, projectID: number) =>
  requestJSON<{ state: string; reason: string; metrics: Metric[] }>(scopePath(tenantID, projectID, "metrics"));
export const getEvents = (tenantID: number, projectID: number) =>
  requestJSON<EventRecord[]>(scopePath(tenantID, projectID, "events"));
export const getLogs = (tenantID: number, projectID: number) =>
  requestJSON<Availability>(scopePath(tenantID, projectID, "logs"));
export const getAlerts = (tenantID: number, projectID: number) =>
  requestJSON<Availability>(scopePath(tenantID, projectID, "alerts"));
export const getDashboards = (tenantID: number, projectID: number) =>
  requestJSON<Dashboard[]>(scopePath(tenantID, projectID, "dashboards"));
export const getResource = (tenantID: number, projectID: number, type: string, id: string) =>
  requestJSON<Resource>(scopePath(tenantID, projectID, `resources/${encodeURIComponent(type)}/${encodeURIComponent(id)}`));
export const getResourceMetrics = (tenantID: number, projectID: number, type: string, id: string) =>
  requestJSON<MetricsResponse>(scopePath(tenantID, projectID, `resources/${encodeURIComponent(type)}/${encodeURIComponent(id)}/metrics`));
