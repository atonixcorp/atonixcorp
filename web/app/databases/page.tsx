"use client";

import { useCallback, useEffect, useMemo, useRef, useState, type FormEvent } from "react";
import "./databases.css";
import { CloudShell } from "@/app/shared/ui/CloudShell";

type Tenant = { id: number; name: string };
type Project = { id: number; tenant_id: number; name: string };
type Engine = {
  id: string;
  name: string;
  category: string;
  description: string;
  port: number;
  versions: string[];
  enabled: boolean;
};
type Plan = {
  id: string;
  cpu: number;
  memory_mb: number;
  storage_gb: number;
  engines: string[];
  high_availability: boolean;
  enabled: boolean;
};
type DatabaseInstance = {
  id: string;
  tenant_id: string;
  project_id: string;
  name: string;
  engine: string;
  engine_version: string;
  plan: string;
  workload_profile: string;
  cpu: number;
  memory_mb: number;
  storage_gb: number;
  region: string;
  status: string;
  endpoint?: string;
  port: number;
  created_at: string;
};
type DatabaseOperation = {
  id: string;
  database_id: string;
  status: string;
  stage: string;
  reason?: string;
  attempt: number;
};
type CreateDatabaseResponse = {
  database: DatabaseInstance;
  operation: DatabaseOperation;
  status_url: string;
  event_stream_url: string;
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
  const body = await response.text();
  let payload: unknown;
  try {
    payload = body ? JSON.parse(body) : null;
  } catch {
    payload = body;
  }
  if (!response.ok) {
    const message = payload && typeof payload === "object" && "error" in payload
      ? String(payload.error)
      : typeof payload === "string" && payload.trim()
        ? payload.trim()
        : `Database request failed (${response.status}).`;
    throw new Error(message);
  }
  if (payload === null) throw new Error("The database service returned an empty response.");
  return payload as T;
}

function statusLabel(status: string): string {
  switch (status.toLowerCase()) {
    case "provisioning": return "Provisioning";
    case "starting": return "Starting · engine health check pending";
    case "running": return "Available";
    case "error": return "Provisioning failed";
    default: return status;
  }
}

export default function DatabasesPage() {
  const [tenants, setTenants] = useState<Tenant[]>([]);
  const [projects, setProjects] = useState<Project[]>([]);
  const [tenantID, setTenantID] = useState("");
  const [projectID, setProjectID] = useState("");
  const [engines, setEngines] = useState<Engine[]>([]);
  const [plans, setPlans] = useState<Plan[]>([]);
  const [instances, setInstances] = useState<DatabaseInstance[]>([]);
  const [name, setName] = useState("");
  const [engineID, setEngineID] = useState("");
  const [version, setVersion] = useState("");
  const [planID, setPlanID] = useState("");
  const [workloadProfile, setWorkloadProfile] = useState("general-purpose");
  const [loading, setLoading] = useState(true);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState("");
  const [notice, setNotice] = useState("");
  const [activeOperation, setActiveOperation] = useState<DatabaseOperation | null>(null);
  const [operationStreamURL, setOperationStreamURL] = useState("");
  const idempotencyRef = useRef<{ fingerprint: string; key: string } | null>(null);

  const availableEngines = useMemo(() => engines.filter((engine) => engine.enabled), [engines]);
  const selectedEngine = engines.find((engine) => engine.id === engineID);
  const availablePlans = useMemo(
    () => plans.filter((plan) => plan.enabled && plan.engines.includes(engineID)),
    [plans, engineID],
  );
  const selectedProject = projects.find((project) => String(project.id) === projectID);

  const refreshInstances = useCallback(async () => {
    if (!tenantID || !projectID) {
      return;
    }
    const items = await requestJSON<DatabaseInstance[]>(
      `/api/databases/tenants/${encodeURIComponent(tenantID)}/projects/${encodeURIComponent(projectID)}/databases`,
    );
    setInstances(items);
  }, [tenantID, projectID]);

  useEffect(() => {
    let active = true;
    Promise.all([
      requestJSON<Tenant[]>("/api/storage/tenants"),
      requestJSON<Engine[]>("/api/databases/database-engines"),
      requestJSON<Record<string, Plan>>("/api/databases/database-plans"),
    ])
      .then(([availableTenants, catalog, planCatalog]) => {
        if (!active) return;
        const availablePlans = Object.values(planCatalog);
        setTenants(availableTenants);
        setEngines(catalog);
        setPlans(availablePlans);
        const defaultEngine = catalog.find((engine) => engine.enabled) ?? catalog[0];
        setEngineID(defaultEngine?.id ?? "");
        setVersion(defaultEngine?.versions[0] ?? "");
        setPlanID(availablePlans.find((plan) =>
          plan.enabled && defaultEngine?.id && plan.engines.includes(defaultEngine.id),
        )?.id ?? "");
        if (availableTenants[0]) setTenantID(String(availableTenants[0].id));
      })
      .catch((loadError: unknown) => {
        setError(loadError instanceof Error ? loadError.message : "Database service could not be loaded.");
      })
      .finally(() => {
        if (active) setLoading(false);
      });
    return () => { active = false; };
  }, []);

  useEffect(() => {
    if (!tenantID) return;
    let active = true;
    requestJSON<Project[]>(`/api/storage/tenants/${encodeURIComponent(tenantID)}/projects`)
      .then((items) => {
        if (!active) return;
        setProjects(items);
        setProjectID((current) => items.some((project) => String(project.id) === current)
          ? current
          : String(items[0]?.id ?? ""));
      })
      .catch((loadError: unknown) => {
        if (active) setError(loadError instanceof Error ? loadError.message : "Projects could not be loaded.");
      });
    return () => { active = false; };
  }, [tenantID]);

  useEffect(() => {
    let active = true;
    Promise.resolve()
      .then(refreshInstances)
      .catch((loadError: unknown) => {
        if (active) setError(loadError instanceof Error ? loadError.message : "Databases could not be loaded.");
      })
      .finally(() => {
        if (active) setLoading(false);
      });
    return () => { active = false; };
  }, [refreshInstances]);

  useEffect(() => {
    if (!instances.some((instance) => instance.status === "provisioning")) return;
    const timer = window.setInterval(() => {
      refreshInstances().catch((loadError: unknown) => {
        setError(loadError instanceof Error ? loadError.message : "Provisioning status could not be refreshed.");
      });
    }, 3000);
    return () => window.clearInterval(timer);
  }, [instances, refreshInstances]);

  useEffect(() => {
    if (!operationStreamURL) return;
    const stream = new EventSource(operationStreamURL);
    const onOperation = (event: Event) => {
      const message = event as MessageEvent<string>;
      try {
        const operation = JSON.parse(message.data) as DatabaseOperation;
        setActiveOperation(operation);
        if (["blocked", "failed", "completed", "cancelled"].includes(operation.status)) {
          void refreshInstances().catch((loadError: unknown) => {
            setError(loadError instanceof Error ? loadError.message : "Database inventory could not be refreshed.");
          });
          stream.close();
        }
      } catch {
        setError("The database operation returned an invalid status update.");
        stream.close();
      }
    };
    stream.addEventListener("operation", onOperation);
    return () => {
      stream.removeEventListener("operation", onOperation);
      stream.close();
    };
  }, [operationStreamURL, refreshInstances]);

  async function createDatabase(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!tenantID || !selectedProject || !availableEngines.some((engine) => engine.id === engineID) ||
      !availablePlans.some((plan) => plan.id === planID)) {
      setError("Choose an enabled project, database engine, and plan.");
      return;
    }
    setSubmitting(true);
    setError("");
    setNotice("");
    try {
      const request = {
        name,
        engine: engineID,
        engine_version: version,
        plan: planID,
        workload_profile: workloadProfile,
      };
      const fingerprint = JSON.stringify([tenantID, projectID, request]);
      if (idempotencyRef.current?.fingerprint !== fingerprint) {
        idempotencyRef.current = { fingerprint, key: crypto.randomUUID() };
      }
      const accepted = await requestJSON<CreateDatabaseResponse>(
        `/api/databases/tenants/${encodeURIComponent(tenantID)}/projects/${encodeURIComponent(projectID)}/databases`,
        {
          method: "POST",
          headers: { "Idempotency-Key": idempotencyRef.current.key },
          body: JSON.stringify(request),
        },
      );
      idempotencyRef.current = null;
      setName("");
      setActiveOperation(accepted.operation);
      setOperationStreamURL(`/api/databases/${accepted.event_stream_url.replace(/^\/api\/v1\//, "")}`);
      setNotice(`${accepted.database.name}: ${accepted.operation.stage} (${accepted.operation.status}).`);
      await refreshInstances();
    } catch (createError) {
      setError(createError instanceof Error ? createError.message : "Database creation failed.");
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <CloudShell>
    <main className="database-console">
      <div className="database-content">
        <header className="database-heading">
          <div>
            <p className="database-eyebrow">DATA SERVICES / MANAGED DATABASES</p>
            <h1>Database platform</h1>
            <p>Create managed databases and track their setup status.</p>
          </div>
          <div className="database-scope">
            <label>
              <span>Workspace</span>
              <select value={tenantID} onChange={(event) => {
                setTenantID(event.target.value);
                setProjectID("");
                setProjects([]);
                setInstances([]);
              }} disabled={loading}>
                {tenants.map((tenant) => <option key={tenant.id} value={tenant.id}>{tenant.name}</option>)}
              </select>
            </label>
            <label>
              <span>Project</span>
              <select value={projectID} onChange={(event) => setProjectID(event.target.value)} disabled={loading || projects.length === 0}>
                {projects.map((project) => <option key={project.id} value={project.id}>{project.name}</option>)}
              </select>
            </label>
          </div>
        </header>

        {error && <div className="database-alert" role="alert">{error}</div>}
        {notice && <div className="database-notice" role="status">{notice}</div>}
        {activeOperation && (
          <div className="database-operation" role="status" aria-live="polite">
            <strong>Operation {activeOperation.id}</strong>
            <span>{activeOperation.stage.replaceAll("_", " ")} · {activeOperation.status}</span>
            {activeOperation.reason && <small>{activeOperation.reason}</small>}
          </div>
        )}

        {!loading && availableEngines.length === 0 && (
          <section className="database-unconfigured">
            <h2>Database provisioning is not available</h2>
            <p>The database service is running, but no database engines or service plans are currently enabled. Please contact your platform administrator.</p>
          </section>
        )}

        <div className="database-layout">
          <section className="database-panel">
            <div className="database-panel-heading">
              <div><p className="database-eyebrow">DATABASE INVENTORY</p><h2>Instances</h2></div>
              <span>{instances.length} total</span>
            </div>
            {loading ? (
              <p className="database-empty">Loading database inventory…</p>
            ) : instances.length === 0 ? (
              <p className="database-empty">{selectedProject ? `No databases in ${selectedProject.name}.` : "Select a project to view its databases."}</p>
            ) : (
              <div className="database-table-wrap">
                <table>
                  <thead><tr><th>Name</th><th>Engine</th><th>Plan</th><th>Profile</th><th>Status</th><th>Created</th></tr></thead>
                  <tbody>
                    {instances.map((instance) => (
                      <tr key={instance.id}>
                        <td><strong>{instance.name}</strong><small>{instance.region || "Region pending"}</small></td>
                        <td>{instance.engine} {instance.engine_version}</td>
                        <td>{instance.plan}<small>{instance.cpu} vCPU · {instance.memory_mb / 1024} GiB · {instance.storage_gb} GB</small></td>
                        <td>{instance.workload_profile}</td>
                        <td><span className={`database-status ${instance.status}`}>{statusLabel(instance.status)}</span></td>
                        <td>{new Date(instance.created_at).toLocaleString()}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </section>

          <section className="database-panel database-create-panel">
            <div className="database-panel-heading">
              <div><p className="database-eyebrow">PROVISIONING</p><h2>Create a database</h2></div>
            </div>
            <form onSubmit={createDatabase}>
              <label><span>Database name</span><input value={name} onChange={(event) => setName(event.target.value)} minLength={2} maxLength={63} pattern="[a-z][a-z0-9-]+" placeholder="research-db" required /></label>
              <div className="database-field-row">
                <label><span>Engine</span><select value={engineID} onChange={(event) => {
                  const nextEngineID = event.target.value;
                  const nextEngine = engines.find((engine) => engine.id === nextEngineID);
                  setEngineID(nextEngineID);
                  setVersion(nextEngine?.versions[0] ?? "");
                  setPlanID(plans.find((plan) => plan.enabled && plan.engines.includes(nextEngineID))?.id ?? "");
                }} disabled={availableEngines.length === 0} required>
                  {engines.map((engine) => <option value={engine.id} key={engine.id} disabled={!engine.enabled}>{engine.name}{engine.enabled ? "" : " · unavailable"}</option>)}
                </select></label>
                <label><span>Version</span><select value={version} onChange={(event) => setVersion(event.target.value)} disabled={!selectedEngine?.enabled} required>
                  {(selectedEngine?.versions ?? []).map((item) => <option value={item} key={item}>{item}</option>)}
                </select></label>
              </div>
              <div className="database-field-row">
                <label><span>Plan</span><select value={planID} onChange={(event) => setPlanID(event.target.value)} disabled={availablePlans.length === 0} required>
                  {availablePlans.map((plan) => <option value={plan.id} key={plan.id}>{plan.id} · {plan.cpu} vCPU · {plan.storage_gb} GB</option>)}
                </select></label>
                <label><span>Workload profile</span><select value={workloadProfile} onChange={(event) => setWorkloadProfile(event.target.value)}>
                  <option value="general-purpose">General purpose</option>
                  <option value="transaction-processing">Transaction processing</option>
                  <option value="analytics">Analytics</option>
                  <option value="ai-ml">AI / ML</option>
                  <option value="hpc">OpenHPC</option>
                </select></label>
              </div>
              <p className="database-help">OpenHPC is available as a workload profile. Slurm connection bindings are not enabled until database health checks and secret delivery are configured.</p>
              <button type="submit" disabled={submitting || loading || !availablePlans.length || !tenantID || !projectID}>
                {submitting ? "Submitting…" : "Create database"}
              </button>
            </form>
          </section>
        </div>
      </div>
    </main>
    </CloudShell>
  );
}
