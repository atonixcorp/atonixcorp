"use client";

import Link from "next/link";
import { useParams, useRouter } from "next/navigation";
import { useCallback, useEffect, useState } from "react";
import {
  deleteDeployment,
  getDeployment,
  getDeploymentEvents,
  getDeploymentInstances,
  getDeploymentLogs,
  getDeploymentMetrics,
  getDeploymentRevisions,
  operateDeployment,
  updateDeployment,
} from "../lib/api";
import type { ContainerDeployment, ContainerDeploymentSpec, ContainerMetrics, ContainerRevision } from "../lib/api";

type Tab = "overview" | "instances" | "logs" | "events";
type Instance = Awaited<ReturnType<typeof getDeploymentInstances>>[number];
type Events = Awaited<ReturnType<typeof getDeploymentEvents>>;

function messageOf(error: unknown): string {
  return error instanceof Error ? error.message : "The container service could not complete the request.";
}

export default function ContainerDeploymentDetailPage() {
  const params = useParams<{ id: string }>();
  const router = useRouter();
  const id = params.id;
  const [deployment, setDeployment] = useState<ContainerDeployment | null>(null);
  const [instances, setInstances] = useState<Instance[]>([]);
  const [logs, setLogs] = useState("");
  const [events, setEvents] = useState<Events | null>(null);
  const [revisions, setRevisions] = useState<ContainerRevision[]>([]);
  const [metrics, setMetrics] = useState<ContainerMetrics | null>(null);
  const [metricsError, setMetricsError] = useState("");
  const [editSpec, setEditSpec] = useState<ContainerDeploymentSpec | null>(null);
  const [tab, setTab] = useState<Tab>("overview");
  const [loading, setLoading] = useState(true);
  const [acting, setActing] = useState(false);
  const [error, setError] = useState("");
  const [scaleValue, setScaleValue] = useState(1);

  const loadDeployment = useCallback(async () => {
    setLoading(true);
    setError("");
    try {
      const current = await getDeployment(id);
      setDeployment(current);
      setScaleValue(current.desired_replicas);
    } catch (loadError) {
      setError(messageOf(loadError));
    } finally {
      setLoading(false);
    }
  }, [id]);

  useEffect(() => {
    const timer = window.setTimeout(() => { void loadDeployment(); }, 0);
    return () => window.clearTimeout(timer);
  }, [loadDeployment]);

  useEffect(() => {
    let active = true;
    if (tab === "overview") {
      void getDeploymentRevisions(id).then((result) => {
        if (active) setRevisions(result);
      }).catch((loadError: unknown) => {
        if (active) setError(messageOf(loadError));
      });
    }
    if (tab === "overview" || tab === "events") {
      void getDeploymentEvents(id).then((result) => {
        if (active) setEvents(result);
      }).catch((loadError: unknown) => {
        if (active) setError(messageOf(loadError));
      });
    }
    if (tab === "instances") {
      void getDeploymentInstances(id).then((result) => {
        if (active) setInstances(result);
      }).catch((loadError: unknown) => {
        if (active) setError(messageOf(loadError));
      });
    } else if (tab === "logs") {
      void getDeploymentLogs(id).then((result) => {
        if (active) setLogs(result);
      }).catch((loadError: unknown) => {
        if (active) setError(messageOf(loadError));
      });
    }
    return () => { active = false; };
  }, [id, tab]);

  useEffect(() => {
    if (tab !== "overview" || !deployment || !["RUNNING", "DEGRADED"].includes(deployment.state)) return;
    let active = true;
    const refresh = () => {
      void getDeploymentMetrics(id).then((result) => {
        if (active) {
          setMetrics(result);
          setMetricsError("");
        }
      }).catch((loadError: unknown) => {
        if (active) setMetricsError(messageOf(loadError));
      });
    };
    refresh();
    const timer = window.setInterval(refresh, 30000);
    return () => {
      active = false;
      window.clearInterval(timer);
    };
  }, [deployment, id, tab]);

  useEffect(() => {
    if (!deployment || !["PENDING_VALIDATION", "QUEUED", "PROVISIONING", "VERIFYING", "UPDATING", "SCALING", "RESTARTING", "STOPPING", "DELETING"].includes(deployment.state)) return;
    let active = true;
    const timer = window.setInterval(() => {
      void getDeployment(id).then((current) => {
        if (active) setDeployment(current);
      }).catch((loadError: unknown) => {
        if (active) setError(messageOf(loadError));
      });
    }, 5000);
    return () => {
      active = false;
      window.clearInterval(timer);
    };
  }, [deployment, id]);

  useEffect(() => {
    if (tab !== "events") return;
    let active = true;
    const timer = window.setInterval(() => {
      void getDeploymentEvents(id).then((current) => {
        if (active) setEvents(current);
      }).catch((loadError: unknown) => {
        if (active) setError(messageOf(loadError));
      });
    }, 15000);
    return () => {
      active = false;
      window.clearInterval(timer);
    };
  }, [id, tab]);

  async function operate(action: "start" | "stop" | "restart" | "scale" | "delete" | "rollback", revision?: number) {
    if (!deployment || acting) return;
    if (action === "delete" && !window.confirm(`Delete "${deployment.name}" and its Kubernetes resources?`)) return;
    if (action === "rollback" && !window.confirm(`Roll back to revision ${revision}? This creates a new revision from that release.`)) return;
    setActing(true);
    setError("");
    try {
      const updated = action === "delete"
        ? await deleteDeployment(id)
        : await operateDeployment(
          id,
          action,
          action === "scale" ? scaleValue : undefined,
          action === "rollback" ? revision : undefined,
        );
      setDeployment(updated);
      if (action === "rollback") {
        setRevisions(await getDeploymentRevisions(id));
        setEditSpec(null);
      }
      if (action === "delete") {
        router.replace("/containers");
        return;
      }
    } catch (operationError) {
      setError(messageOf(operationError));
    } finally {
      setActing(false);
    }
  }

  async function saveUpdate() {
    if (!deployment || !editSpec || acting) return;
    setActing(true);
    setError("");
    try {
      const updated = await updateDeployment(id, deployment.current_revision, editSpec);
      setDeployment(updated);
      setEditSpec(null);
      setRevisions(await getDeploymentRevisions(id));
    } catch (updateError) {
      setError(messageOf(updateError));
    } finally {
      setActing(false);
    }
  }

  return (
    <>
      <header className="container-heading">
        <div>
          <p className="container-eyebrow"><Link href="/containers">CONTAINERS</Link> / DEPLOYMENT</p>
          <h1>{loading ? "Loading deployment…" : deployment?.name ?? "Deployment"}</h1>
          <p>Inspect the workload state, runtime instances, logs, and provisioning history.</p>
        </div>
        <Link className="container-button secondary" href="/containers">← All deployments</Link>
      </header>

      {error && <div className="container-alert" role="alert">{error}</div>}
      {loading ? <div className="container-card container-loading" role="status">Loading deployment details…</div> : deployment ? (
        <>
          <section className="container-detail-grid" aria-label="Deployment summary">
            <div><small>Status</small><span className="container-status" data-state={deployment.state}>{deployment.state.toLowerCase().replaceAll("_", " ")}</span></div>
            <div><small>Ready replicas</small><strong>{deployment.ready_replicas}/{deployment.desired_replicas}</strong></div>
            <div><small>Environment</small><strong>{deployment.spec.environment}</strong></div>
            <div><small>Image</small><strong>{deployment.spec.image}</strong></div>
          </section>

          <section className="container-card">
            <nav className="container-tabs" aria-label="Deployment details">
              {(["overview", "instances", "logs", "events"] as const).map((item) => (
                <button key={item} type="button" className={`container-tab${tab === item ? " active" : ""}`} onClick={() => { setError(""); setTab(item); }}>
                  {item[0].toUpperCase() + item.slice(1)}
                </button>
              ))}
            </nav>
            <div className="container-detail-body">
              {tab === "overview" && (
                <>
                  {deployment.error_details && <div className="container-alert">{deployment.error_details}</div>}
                  <div className="container-detail-grid">
                    <div><small>Deployment ID</small><strong>{deployment.id}</strong></div>
                    <div><small>Current revision</small><strong>{deployment.current_revision}</strong></div>
                    <div><small>Workspace / project</small><strong>{deployment.tenant_id} / {deployment.project_id}</strong></div>
                    <div><small>CPU request / limit</small><strong>{deployment.spec.cpu_request} / {deployment.spec.cpu_limit}</strong></div>
                    <div><small>Memory request / limit</small><strong>{deployment.spec.memory_request} / {deployment.spec.memory_limit}</strong></div>
                    <div><small>Private service endpoint</small><strong>{deployment.endpoint || "Not assigned"}</strong></div>
                    <div><small>Provisioning job</small><strong>{deployment.provisioning_job_id}</strong></div>
                    <div><small>Correlation ID</small><strong>{deployment.correlation_id}</strong></div>
                    <div><small>Created</small><strong>{new Date(deployment.created_at).toLocaleString()}</strong></div>
                  </div>
                  {metrics && (
                    <div className="container-detail-grid" aria-label="Live resource usage">
                      <div><small>CPU usage</small><strong>{metrics.cpu_usage_millis} millicores</strong></div>
                      <div><small>Memory usage</small><strong>{(metrics.memory_usage_bytes / 1024 / 1024).toFixed(1)} MiB</strong></div>
                      <div><small>Metrics sampled</small><strong>{new Date(metrics.sampled_at).toLocaleTimeString()}</strong></div>
                      <div><small>Refresh interval</small><strong>30 seconds</strong></div>
                    </div>
                  )}
                  {metricsError && <p className="container-muted-note">Metrics unavailable: {metricsError}. The cluster metrics API may not be installed.</p>}
                  <div className="container-detail-actions">
                    {deployment.state === "STOPPED" && <button className="container-button" type="button" disabled={acting} onClick={() => void operate("start")}>Start</button>}
                    {["RUNNING", "DEGRADED"].includes(deployment.state) && (
                      <>
                        <button className="container-button secondary" type="button" disabled={acting} onClick={() => void operate("stop")}>Stop</button>
                        <button className="container-button secondary" type="button" disabled={acting} onClick={() => void operate("restart")}>Restart</button>
                        <label className="container-field">
                          <span className="sr-only">Desired replicas</span>
                          <input type="number" min={1} max={50} value={scaleValue} onChange={(event) => setScaleValue(Number(event.target.value))} aria-label="Desired replicas" />
                        </label>
                        <button className="container-button secondary" type="button" disabled={acting || scaleValue < 1 || scaleValue > 50} onClick={() => void operate("scale")}>Scale</button>
                      </>
                    )}
                    <button className="container-button danger" type="button" disabled={acting} onClick={() => void operate("delete")}>Delete</button>
                    {["RUNNING", "DEGRADED"].includes(deployment.state) && !editSpec && (
                      <button className="container-button secondary" type="button" onClick={() => setEditSpec({ ...deployment.spec })}>Update configuration</button>
                    )}
                  </div>
                  {editSpec && (
                    <div className="container-update-form">
                      <h2>Update deployment</h2>
                      <p>Changes create a new immutable revision. Leave existing secret values blank to preserve them.</p>
                      <div className="container-field-grid">
                        <label className="container-field"><span>Image</span><input value={editSpec.image} onChange={(event) => setEditSpec({ ...editSpec, image: event.target.value })} /></label>
                        <label className="container-field"><span>Replicas</span><input type="number" min={1} max={50} value={editSpec.replicas} onChange={(event) => setEditSpec({ ...editSpec, replicas: Number(event.target.value) })} /></label>
                        <label className="container-field"><span>CPU request</span><input value={editSpec.cpu_request} onChange={(event) => setEditSpec({ ...editSpec, cpu_request: event.target.value })} /></label>
                        <label className="container-field"><span>CPU limit</span><input value={editSpec.cpu_limit} onChange={(event) => setEditSpec({ ...editSpec, cpu_limit: event.target.value })} /></label>
                        <label className="container-field"><span>Memory request</span><input value={editSpec.memory_request} onChange={(event) => setEditSpec({ ...editSpec, memory_request: event.target.value })} /></label>
                        <label className="container-field"><span>Memory limit</span><input value={editSpec.memory_limit} onChange={(event) => setEditSpec({ ...editSpec, memory_limit: event.target.value })} /></label>
                      </div>
                      <div className="container-detail-actions">
                        <button className="container-button" type="button" disabled={acting} onClick={() => void saveUpdate()}>{acting ? "Updating…" : "Apply update"}</button>
                        <button className="container-button secondary" type="button" disabled={acting} onClick={() => setEditSpec(null)}>Cancel</button>
                      </div>
                    </div>
                  )}
                  <div className="container-info">Container networking is cluster-internal. Public endpoints are not configured. Metrics require the Kubernetes Metrics API.</div>
                  <h2>Release history</h2>
                  <div className="container-event-list">
                    {revisions.map((revision) => (
                      <article className="container-event" key={`revision-${revision.revision}`}>
                        <strong>Revision {revision.revision}{revision.revision === deployment.current_revision ? " · current" : ""}</strong>
                        <small>{new Date(revision.created_at).toLocaleString()} · image {revision.spec.image} · hash {revision.spec_hash.slice(0, 12)}</small>
                        {revision.revision < deployment.current_revision && ["RUNNING", "DEGRADED"].includes(deployment.state) && (
                          <button className="container-button secondary" type="button" disabled={acting} onClick={() => void operate("rollback", revision.revision)}>Rollback to this revision</button>
                        )}
                      </article>
                    ))}
                    {!revisions.length && <p>No release history is available.</p>}
                  </div>
                  <h2>Provisioning timeline</h2>
                  <div className="container-event-list">
                    {(events?.audit ?? []).map((event) => (
                      <article className="container-event" key={`timeline-${event.id}`}>
                        <strong><span className="container-status" data-state={String(event.new_state ?? "")}>{String(event.new_state ?? "Event").toLowerCase().replaceAll("_", " ")}</span></strong>
                        <small>{event.created_at ? new Date(String(event.created_at)).toLocaleString() : ""} · {String(event.reason ?? "")}</small>
                      </article>
                    ))}
                    {!(events?.audit?.length) && <p>Provisioning events are not available yet.</p>}
                  </div>
                </>
              )}
              {tab === "instances" && (
                instances.length ? (
                  <div className="container-table-wrap"><table className="container-table">
                    <thead><tr><th>Instance</th><th>Phase</th><th>Ready</th><th>Restarts</th><th>Node</th><th>Created</th></tr></thead>
                    <tbody>{instances.map((instance) => <tr key={instance.name}>
                      <td>{instance.name}</td><td>{instance.phase}</td><td>{instance.ready ? "Ready" : "Not ready"}</td>
                      <td>{instance.restarts}</td><td>{instance.node || "Unscheduled"}</td><td>{new Date(instance.created_at).toLocaleString()}</td>
                    </tr>)}</tbody>
                  </table></div>
                ) : <p>{deployment.ready_replicas ? "No runtime instances were returned." : "No running instances are available."}</p>
              )}
              {tab === "logs" && <pre>{logs || "No container logs returned."}</pre>}
              {tab === "events" && (
                <div className="container-event-list">
                  {(events?.audit ?? []).map((event) => (
                    <article className="container-event" key={`audit-${event.id}`}>
                      <strong>{String(event.new_state ?? "Event")} · {String(event.reason ?? "")}</strong>
                      <small>{event.created_at ? new Date(String(event.created_at)).toLocaleString() : ""} · correlation {String(event.correlation_id ?? deployment.correlation_id)}</small>
                      {Boolean(event.error_details) && <p>{String(event.error_details)}</p>}
                    </article>
                  ))}
                  {(events?.runtime ?? []).map((event, index) => (
                    <article className="container-event" key={`runtime-${event.lastTimestamp}-${index}`}>
                      <strong>{event.type} · {event.reason}</strong>
                      <small>{event.lastTimestamp}</small><p>{event.message}</p>
                    </article>
                  ))}
                  {!(events?.audit?.length) && !(events?.runtime?.length) && <p>No events are available.</p>}
                </div>
              )}
              {(tab === "instances" || tab === "logs" || tab === "events") && (
                <button className="container-button secondary" type="button" onClick={() => {
                  if (tab === "instances") void getDeploymentInstances(id).then(setInstances).catch((loadError: unknown) => setError(messageOf(loadError)));
                  if (tab === "logs") void getDeploymentLogs(id).then(setLogs).catch((loadError: unknown) => setError(messageOf(loadError)));
                  if (tab === "events") void getDeploymentEvents(id).then(setEvents).catch((loadError: unknown) => setError(messageOf(loadError)));
                }}>Refresh {tab}</button>
              )}
            </div>
          </section>
        </>
      ) : (
        <section className="container-card container-empty">
          <h2>Deployment is unavailable</h2>
          <p>{error || "The deployment could not be found or you do not have access to it."}</p>
          {error.toLowerCase().includes("authentication required") && <p><Link href="/login">Sign in</Link> to access container deployments.</p>}
          <button type="button" className="container-button secondary" onClick={() => void loadDeployment()}>Retry</button>
        </section>
      )}
    </>
  );
}
