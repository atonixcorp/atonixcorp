export function initialScopeID(
  items: Array<{ id: number }>,
  parameter: "tenant_id" | "project_id",
  tenantID?: number,
): number {
  return resolveScopeID(items, parameter, new URLSearchParams(window.location.search), tenantID);
}

export function resolveScopeID(
  items: Array<{ id: number }>,
  parameter: "tenant_id" | "project_id",
  query: URLSearchParams,
  tenantID?: number,
): number {
  const requested = query.get(parameter);
  if (!requested || (parameter === "project_id" && query.get("tenant_id") !== String(tenantID))) {
    return items[0]?.id ?? 0;
  }
  const match = items.find((item) => String(item.id) === requested);
  if (!match) throw new Error(`The requested ${parameter === "tenant_id" ? "workspace" : "project"} is not accessible. Choose an accessible scope to continue.`);
  return match.id;
}

export function scopedCloudHref(path: string, tenantID: number, projectID: number): string {
  if (!tenantID) return path;
  const [location, hash] = path.split("#");
  const [pathname, search] = location.split("?");
  const query = new URLSearchParams(search);
  query.set("tenant_id", String(tenantID));
  if (projectID) query.set("project_id", String(projectID));
  else query.delete("project_id");
  return `${pathname}?${query}${hash ? `#${hash}` : ""}`;
}
