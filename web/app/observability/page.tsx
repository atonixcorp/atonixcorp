"use client";

import { useEffect, useMemo, useState } from "react";
import { useRouter } from "next/navigation";
import { CloudShell } from "@/app/shared/ui/CloudShell";
import {
  getAlerts,
  getDashboards,
  getEvents,
  getIntegrations,
  getLogs,
  getMetrics,
  getOverview,
  getProjects,
  getResource,
  getResourceMetrics,
  getResources,
  getTenants,
  type Availability,
  type Dashboard,
  type EventRecord,
  type Integration,
  type MetricsResponse,
  type Overview,
  type Resource,
  type ObservabilityProject,
  type ObservabilityTenant,
} from "./lib/api";
import "./observability.css";

const sections = [
  "Overview",
  "Infrastructure",
  "Metrics",
  "Logs",
  "Events",
  "Alerts",
  "Dashboards",
  "Resource Health",
  "Integrations",
] as const;
type Section = (typeof sections)[number];
const detailTabs = ["Overview", "Metrics", "Logs", "Events", "Alerts", "Configuration"] as const;
type DetailTab = (typeof detailTabs)[number];

function messageOf(error: unknown): string {
  return error instanceof Error ? error.message : "The observability service could not complete the request.";
}

function statusOf(error: unknown): number | undefined {
  return error && typeof error === "object" && "status" in error && typeof error.status === "number"
    ? error.status
    : undefined;
}

function formatTime(value: string): string {
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? "—" : date.toLocaleString();
}

function HealthBadge({ value }: { value: string }) {
  return <span className={`observability-health observability-health-${value.toLowerCase().replaceAll(" ", "-")}`}>{value}</span>;
}

export default function ObservabilityPage() {
  const router = useRouter();
  const [tenants, setTenants] = useState<ObservabilityTenant[]>([]);
  const [projects, setProjects] = useState<ObservabilityProject[]>([]);
  const [tenantID, setTenantID] = useState(0);
  const [projectID, setProjectID] = useState(0);
  const [section, setSection] = useState<Section>("Overview");
  const [overview, setOverview] = useState<Overview | null>(null);
  const [resources, setResources] = useState<Resource[]>([]);
  const [integrations, setIntegrations] = useState<Integration[]>([]);
  const [loading, setLoading] = useState(true);
  const [loadedScope, setLoadedScope] = useState("");
  const [error, setError] = useState("");
  const [refreshKey, setRefreshKey] = useState(0);
  const [sectionLoading, setSectionLoading] = useState(false);
  const [metrics, setMetrics] = useState<{ state: string; reason: string; metrics: MetricsResponse["metrics"] } | null>(null);
  const [logs, setLogs] = useState<Availability | null>(null);
  const [alerts, setAlerts] = useState<Availability | null>(null);
  const [events, setEvents] = useState<EventRecord[]>([]);
  const [dashboards, setDashboards] = useState<Dashboard[]>([]);
  const [selectedResource, setSelectedResource] = useState<Resource | null>(null);
  const [detailTab, setDetailTab] = useState<DetailTab>("Overview");
  const [resourceMetrics, setResourceMetrics] = useState<MetricsResponse | null>(null);

  useEffect(() => {
    let active = true;
    getTenants().then((items) => {
      if (!active) return;
      setTenants(items);
      setTenantID(items[0]?.id ?? 0);
      if (!items.length) setLoading(false);
    }).catch((loadError: unknown) => {
      if (!active) return;
      if (statusOf(loadError) === 401) router.replace("/login");
      setError(messageOf(loadError));
      setLoading(false);
    });
    return () => { active = false; };
  }, [router]);

  useEffect(() => {
    if (!tenantID) return;
    let active = true;
    const loadProjects = async () => {
      try {
        const items = await getProjects(tenantID);
        if (!active) return;
        setProjects(items);
        setProjectID(items[0]?.id ?? 0);
        if (!items.length) setLoading(false);
      } catch (loadError) {
        if (!active) return;
        if (statusOf(loadError) === 401) router.replace("/login");
        setError(messageOf(loadError));
        setLoading(false);
      }
    };
    void loadProjects();
    return () => { active = false; };
  }, [tenantID, router]);

  useEffect(() => {
    let active = true;
    const loadWorkspace = async () => {
    if (!tenantID || !projectID) {
        await Promise.resolve();
        if (active) {
          setOverview(null);
          setResources([]);
          setIntegrations([]);
          setLoading(false);
          setLoadedScope("");
        }
      return;
    }
    try {
      const [nextOverview, nextResources, nextIntegrations] = await Promise.all([
        getOverview(tenantID, projectID),
        getResources(tenantID, projectID),
        getIntegrations(tenantID, projectID),
      ]);
      setOverview(nextOverview);
      setResources(nextResources);
      setIntegrations(nextIntegrations);
      setSelectedResource((current) => current
        ? nextResources.find((item) => item.id === current.id && item.resource_type === current.resource_type) ?? null
        : null);
      setLoadedScope(`${tenantID}/${projectID}`);
      setError("");
    } catch (loadError) {
      if (statusOf(loadError) === 401) router.replace("/login");
      setError(messageOf(loadError));
    } finally {
      setLoading(false);
    }
    };
    void loadWorkspace();
    return () => { active = false; };
  }, [tenantID, projectID, router, refreshKey]);

  useEffect(() => {
    if (!tenantID || !projectID) return;
    let active = true;
    const request = async () => {
      try {
        if (section === "Metrics") {
          const result = await getMetrics(tenantID, projectID);
          if (active) setMetrics(result);
        } else if (section === "Logs") {
          const result = await getLogs(tenantID, projectID);
          if (active) setLogs(result);
        } else if (section === "Events") {
          const result = await getEvents(tenantID, projectID);
          if (active) setEvents(result);
        } else if (section === "Alerts") {
          const result = await getAlerts(tenantID, projectID);
          if (active) setAlerts(result);
        } else if (section === "Dashboards") {
          const result = await getDashboards(tenantID, projectID);
          if (active) setDashboards(result);
        }
      } catch (loadError) {
        if (active) {
          if (statusOf(loadError) === 401) router.replace("/login");
          setError(messageOf(loadError));
        }
      } finally {
        if (active) setSectionLoading(false);
      }
    };
    void request();
    return () => { active = false; };
  }, [section, tenantID, projectID, router]);

  useEffect(() => {
    if (!selectedResource || !tenantID || !projectID || detailTab !== "Metrics") return;
    let active = true;
    getResourceMetrics(tenantID, projectID, selectedResource.resource_type, selectedResource.id)
      .then((result) => { if (active) setResourceMetrics(result); })
      .catch((loadError: unknown) => {
        if (active) setError(messageOf(loadError));
      });
    return () => { active = false; };
  }, [selectedResource, detailTab, tenantID, projectID]);

  const projectName = useMemo(
    () => projects.find((project) => project.id === projectID)?.name ?? "Select a project",
    [projects, projectID],
  );
  const selectedGroup = overview?.resource_groups.find((group) => group.kind === selectedResource?.resource_type);

  const openResource = async (resource: Resource) => {
    try {
      const detail = await getResource(tenantID, projectID, resource.resource_type, resource.id);
      setSelectedResource(detail);
      setDetailTab("Overview");
      setResourceMetrics(null);
    } catch (loadError) {
      if (statusOf(loadError) === 401) router.replace("/login");
      setError(messageOf(loadError));
    }
  };

  const selectSection = (nextSection: Section) => {
    setSection(nextSection);
    setSectionLoading(Boolean(projectID) && ["Metrics", "Logs", "Events", "Alerts", "Dashboards"].includes(nextSection));
  };
  const refreshData = () => {
    setError("");
    setLoading(true);
    setRefreshKey((key) => key + 1);
  };

  const renderResourceTable = (items: Resource[]) => (
    <div className="observability-table-wrap">
      <table className="observability-table">
        <thead><tr><th>Resource</th><th>Type</th><th>Provider</th><th>Resource state</th><th>Monitoring</th><th>Last discovered</th></tr></thead>
        <tbody>
          {items.map((resource) => (
            <tr key={`${resource.resource_type}:${resource.id}`}>
              <td><button className="observability-resource-link" type="button" onClick={() => void openResource(resource)}>{resource.name}</button><small>{resource.id}</small></td>
              <td>{resource.resource_type}</td>
              <td>{resource.provider}</td>
              <td><HealthBadge value={resource.status || "unknown"} /></td>
              <td><HealthBadge value={resource.monitoring_status.replaceAll("_", " ")} /></td>
              <td>{formatTime(resource.last_seen_at)}</td>
            </tr>
          ))}
          {!items.length && <tr><td className="observability-table-empty" colSpan={6}>No connected AtonixCorp resources were discovered in this project.</td></tr>}
        </tbody>
      </table>
    </div>
  );

  const renderSection = () => {
    if (!projectID) {
      return <div className="observability-empty"><h2>No project selected</h2><p>Select a workspace with a project to view its inventory.</p></div>;
    }
    if (section === "Overview") {
      return (
        <>
          <section className="observability-health-grid" aria-label="Infrastructure health">
            <article><span>Healthy</span><strong>{overview?.telemetry_state === "available" ? overview.healthy_resources ?? "—" : "—"}</strong><small>Based on fresh telemetry</small></article>
            <article><span>Warning</span><strong>{overview?.telemetry_state === "available" ? overview.warning_resources ?? "—" : "—"}</strong><small>Based on fresh telemetry</small></article>
            <article><span>Critical</span><strong>{overview?.telemetry_state === "available" ? overview.critical_resources ?? "—" : "—"}</strong><small>Based on fresh telemetry</small></article>
            <article><span>Unknown</span><strong>{overview?.unknown_health ?? "—"}</strong><small>Health cannot be verified without telemetry</small></article>
          </section>
          <section className="observability-card observability-overview-card">
            <header><div><h2>Infrastructure health</h2><p>Resource counts are discovered from connected platform services. Health values require live telemetry.</p></div><HealthBadge value={overview?.health ?? "unknown"} /></header>
            <div className="observability-resource-groups">
              {overview?.resource_groups.map((group) => (
                <article key={group.kind}>
                  <span>{group.kind.replaceAll(".", " ")}</span>
                  <strong>{group.count}</strong>
                  <small>{group.source.replaceAll("_", " ")} · health {group.health}</small>
                </article>
              ))}
            </div>
          </section>
          <section className="observability-card">
            <header><div><h2>Active alerts</h2><p>Alerts are not reported as zero when no evaluator is connected.</p></div></header>
            <div className="observability-unavailable"><strong>Alert status unavailable</strong><p>No telemetry-backed alert evaluator is configured for this project.</p></div>
          </section>
          <section className="observability-card">
            <header><div><h2>Recent infrastructure</h2><p>{resources.length} registered resource{resources.length === 1 ? "" : "s"} · {projectName}</p></div><button type="button" onClick={() => setSection("Infrastructure")}>View infrastructure</button></header>
            {renderResourceTable(resources.slice(0, 8))}
          </section>
        </>
      );
    }
    if (section === "Infrastructure" || section === "Resource Health") {
      return <section className="observability-card"><header><div><h2>{section}</h2><p>Resources originate from their cloud-service integrations, not from Observability.</p></div><button type="button" onClick={refreshData}>Refresh</button></header>{renderResourceTable(resources)}</section>;
    }
    if (section === "Metrics") {
      return <section className="observability-card"><header><div><h2>Project metrics</h2><p>Metrics are displayed only when a configured adapter supplies real samples.</p></div></header>{sectionLoading ? <p className="observability-loading">Loading metrics status…</p> : metrics?.state === "available" && metrics.metrics.length ? <div className="observability-table-wrap"><table className="observability-table"><thead><tr><th>Resource ID</th><th>Type</th><th>Metric</th><th>Value</th><th>Observed</th></tr></thead><tbody>{metrics.metrics.map((metric, index) => <tr key={`${metric.resource_id}-${metric.name}-${index}`}><td>{metric.resource_id ?? "—"}</td><td>{metric.resource_type ?? "—"}</td><td>{metric.name}</td><td>{metric.value} {metric.unit}</td><td>{formatTime(metric.observed_at)}</td></tr>)}</tbody></table></div> : <div className="observability-unavailable"><strong>No telemetry samples available</strong><p>{metrics?.reason ?? "No connected source has supplied project metrics."}</p></div>}</section>;
    }
    if (section === "Logs") {
      return <section className="observability-card"><header><div><h2>Logs</h2><p>Application and infrastructure logs from authorized collectors.</p></div></header><div className="observability-unavailable"><strong>{logs?.state === "available" ? "Log source connected" : "No log source connected"}</strong><p>{logs?.reason ?? "Log data is unavailable until an authorized log collector is configured."}</p></div></section>;
    }
    if (section === "Events") {
      return <section className="observability-card"><header><div><h2>Platform events</h2><p>Security audit activity currently available for this project.</p></div></header>{sectionLoading ? <p className="observability-loading">Loading events…</p> : events.length ? <div className="observability-event-list">{events.map((event) => <article key={event.id}><HealthBadge value={event.severity} /><div><strong>{event.summary}</strong><p>{event.source} · resource {event.resource_id || "platform"}</p></div><time>{formatTime(event.created_at)}</time></article>)}</div> : <div className="observability-empty"><strong>No security events recorded</strong><p>This is an empty result from the project audit records, not a claim that every service has event telemetry connected.</p></div>}</section>;
    }
    if (section === "Alerts") {
      return <section className="observability-card"><header><div><h2>Alerts</h2><p>Alert status depends on configured telemetry and evaluation rules.</p></div></header><div className="observability-unavailable"><strong>{alerts?.state === "available" ? "Alert evaluator connected" : "Alert evaluator unavailable"}</strong><p>{alerts?.reason ?? "No telemetry-backed alert evaluator is configured for this project."}</p></div></section>;
    }
    if (section === "Dashboards") {
      return <section className="observability-card"><header><div><h2>Platform dashboards</h2><p>Built-in views organize telemetry when their sources are connected.</p></div></header>{sectionLoading ? <p className="observability-loading">Loading dashboards…</p> : <div className="observability-dashboard-grid">{dashboards.map((dashboard) => <article key={dashboard.id}><span>Platform dashboard</span><h3>{dashboard.name}</h3><p>{dashboard.description}</p><small>{dashboard.resource_kinds.join(" · ")}</small></article>)}</div>}</section>;
    }
    return <section className="observability-card"><header><div><h2>Integrations</h2><p>Provider connections are server-side; configured endpoints remain unverified until collection succeeds.</p></div></header><div className="observability-integration-list">{integrations.map((item) => <article key={item.provider}><div><strong>{item.name}</strong><p>{item.description}</p></div><HealthBadge value={item.state.replaceAll("_", " ")} /></article>)}</div></section>;
  };

  const renderResourceDetail = () => {
    if (!selectedResource) return null;
    return (
      <section className="observability-card observability-detail">
        <header><div><p className="observability-eyebrow">RESOURCE DETAIL</p><h2>{selectedResource.name}</h2><p>{selectedResource.resource_type} · {selectedResource.provider} · {selectedResource.region || projectName}</p></div><button type="button" onClick={() => setSelectedResource(null)}>Close</button></header>
        <nav className="observability-detail-tabs" aria-label="Resource details">
          {detailTabs.map((tab) => <button key={tab} type="button" aria-current={detailTab === tab ? "page" : undefined} onClick={() => setDetailTab(tab)}>{tab}</button>)}
        </nav>
        {detailTab === "Overview" && <div className="observability-detail-overview"><div><span>Resource state</span><strong>{selectedResource.status}</strong></div><div><span>Monitoring state</span><strong>{selectedResource.monitoring_status.replaceAll("_", " ")}</strong></div><div><span>Health</span><strong>Unknown</strong></div><div><span>Last discovered</span><strong>{formatTime(selectedResource.last_seen_at)}</strong></div><p>{selectedGroup ? `Collector family: ${selectedGroup.source.replaceAll("_", " ")}` : "No collector has supplied health telemetry for this resource."}</p></div>}
        {detailTab === "Metrics" && <div className="observability-unavailable"><strong>{resourceMetrics?.state === "available" ? "Metrics available" : "No metric samples available"}</strong><p>{resourceMetrics?.metrics.length ? resourceMetrics.metrics.map((metric) => `${metric.name}: ${metric.value} ${metric.unit} (${formatTime(metric.observed_at)})`).join(" · ") : "No collector currently supplies verified metrics for this resource."}</p></div>}
        {detailTab === "Logs" && <div className="observability-unavailable"><strong>Logs unavailable</strong><p>No authorized log adapter is connected to this resource.</p></div>}
        {detailTab === "Events" && <div className="observability-unavailable"><strong>Resource event stream unavailable</strong><p>Platform security events are shown in the Events section; provider event collection is not connected.</p></div>}
        {detailTab === "Alerts" && <div className="observability-unavailable"><strong>Resource alert state unavailable</strong><p>No telemetry-backed alert evaluator is configured.</p></div>}
        {detailTab === "Configuration" && <div className="observability-detail-overview"><div><span>Resource ID</span><strong>{selectedResource.id}</strong></div><div><span>Project</span><strong>{selectedResource.project_id}</strong></div><div><span>Provider</span><strong>{selectedResource.provider}</strong></div><div><span>Region</span><strong>{selectedResource.region || "Not reported"}</strong></div></div>}
      </section>
    );
  };

  return (
    <CloudShell>
      <main className="observability-page">
        <header className="observability-heading">
          <div><p className="observability-eyebrow">ATONIXCORP CLOUD PLATFORM SERVICE</p><h1>Observability</h1><p>Monitor discovered infrastructure, telemetry, and platform activity across connected services.</p></div>
          <div className="observability-scope">
            <label><span>Workspace</span><select value={tenantID || ""} onChange={(event) => { setSelectedResource(null); setProjectID(0); setTenantID(Number(event.target.value)); }} disabled={!tenants.length}><option value="">Select workspace</option>{tenants.map((tenant) => <option key={tenant.id} value={tenant.id}>{tenant.name}</option>)}</select></label>
            <label><span>Project</span><select value={projectID || ""} onChange={(event) => { setSelectedResource(null); setProjectID(Number(event.target.value)); }} disabled={!projects.length}><option value="">Select project</option>{projects.map((project) => <option key={project.id} value={project.id}>{project.name}</option>)}</select></label>
          </div>
        </header>

        <nav className="observability-nav" aria-label="Observability sections">
          {sections.map((item) => <button key={item} type="button" aria-current={section === item ? "page" : undefined} onClick={() => selectSection(item)}>{item}</button>)}
        </nav>

        {error && <div className="observability-error" role="alert"><strong>Observability data could not be fully loaded.</strong><span>{error}</span><button type="button" onClick={refreshData}>Retry</button></div>}
        <div className="observability-service-notice"><strong>Built-in platform service</strong><span>Resources are discovered from their source services. No Observability resource needs to be created.</span></div>

        {loading || (projectID > 0 && loadedScope !== `${tenantID}/${projectID}` && !error)
          ? <div className="observability-loading" role="status">Loading project inventory and source status…</div>
          : renderSection()}
        {renderResourceDetail()}
      </main>
    </CloudShell>
  );
}
