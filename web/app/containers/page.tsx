"use client";

import Link from "next/link";
import { useCallback, useEffect, useState } from "react";
import { getDeployments, getProjects, getTenants } from "./lib/api";
import type { ContainerDeployment, Project, Tenant } from "./lib/api";

function messageOf(error: unknown): string {
  return error instanceof Error ? error.message : "The container service could not complete the request.";
}

export default function ContainerDeploymentsPage() {
  const [tenants, setTenants] = useState<Tenant[]>([]);
  const [projects, setProjects] = useState<Project[]>([]);
  const [tenantID, setTenantID] = useState(0);
  const [projectID, setProjectID] = useState(0);
  const [deployments, setDeployments] = useState<ContainerDeployment[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  useEffect(() => {
    let active = true;
    void getTenants().then((items) => {
      if (!active) return;
      setTenants(items);
      setTenantID(items[0]?.id ?? 0);
      if (!items.length) setLoading(false);
    }).catch((loadError: unknown) => {
      if (!active) return;
      setError(messageOf(loadError));
      setLoading(false);
    });
    return () => { active = false; };
  }, []);

  useEffect(() => {
    if (!tenantID) {
      return;
    }
    let active = true;
    void getProjects(tenantID).then((items) => {
      if (!active) return;
      setProjects(items);
      setProjectID(items[0]?.id ?? 0);
      if (!items.length) setLoading(false);
    }).catch((loadError: unknown) => {
      if (!active) return;
      setError(messageOf(loadError));
      setLoading(false);
    });
    return () => { active = false; };
  }, [tenantID]);

  const loadDeployments = useCallback(async () => {
    if (!tenantID || !projectID) {
      setDeployments([]);
      setLoading(false);
      return;
    }
    setLoading(true);
    setError("");
    try {
      setDeployments(await getDeployments(tenantID, projectID));
    } catch (loadError) {
      setError(messageOf(loadError));
    } finally {
      setLoading(false);
    }
  }, [tenantID, projectID]);

  useEffect(() => {
    if (!projectID) return;
    const timer = window.setTimeout(() => { void loadDeployments(); }, 0);
    return () => window.clearTimeout(timer);
  }, [loadDeployments, projectID]);

  return (
    <>
      <header className="container-heading">
        <div>
          <p className="container-eyebrow">COMPUTE / CONTAINERS</p>
          <h1>Container deployments</h1>
          <p>Deploy and operate workloads on your connected Kubernetes cluster.</p>
        </div>
        <Link className="container-button" href="/containers/deploy">＋ Deploy container</Link>
      </header>

      <section className="container-card" aria-label="Container deployments">
        <header className="container-toolbar">
          <div><h2>Workloads</h2><p>{loading ? "Loading deployments…" : error ? "Deployments unavailable" : `${deployments.length} deployments`}</p></div>
          <div className="container-selectors">
            <label className="container-field">
              <span>Workspace</span>
              <select aria-label="Select workspace" value={tenantID || ""} onChange={(event) => {
                setProjects([]);
                setProjectID(0);
                setLoading(true);
                setError("");
                setTenantID(Number(event.target.value));
              }} disabled={!tenants.length}>
                {tenants.map((tenant) => <option key={tenant.id} value={tenant.id}>{tenant.name}</option>)}
              </select>
            </label>
            <label className="container-field">
              <span>Project</span>
              <select aria-label="Select project" value={projectID || ""} onChange={(event) => setProjectID(Number(event.target.value))} disabled={!projects.length}>
                {projects.map((project) => <option key={project.id} value={project.id}>{project.name}</option>)}
              </select>
            </label>
            <button type="button" className="container-button secondary" onClick={() => void loadDeployments()} disabled={loading || !projectID}>Refresh</button>
          </div>
        </header>

        {error && (
          <div className="container-alert" role="alert">
            {error}{error.toLowerCase().includes("authentication required") && <> <Link href="/login">Sign in</Link> to continue.</>}
          </div>
        )}
        {!loading && !error && !tenants.length && (
          <div className="container-empty">
            <h2>No workspace is available</h2>
            <p>Your account must belong to a workspace and project before it can create deployments. Sign in again or contact a workspace administrator.</p>
          </div>
        )}
        {!loading && !error && tenants.length > 0 && !projects.length && (
          <div className="container-empty">
            <h2>No project is available</h2>
            <p>Ask a workspace owner to add you to a project before deploying containers.</p>
          </div>
        )}
        {loading ? (
          <div className="container-loading" role="status">Loading container deployments…</div>
        ) : !error && projects.length > 0 && deployments.length === 0 ? (
          <div className="container-empty">
            <h2>No deployments yet</h2>
            <p>Deploy an application to the connected Kubernetes cluster. The runtime will report a clear error if no cluster is configured.</p>
            <Link className="container-button" href="/containers/deploy">Start deployment</Link>
          </div>
        ) : !error && deployments.length > 0 ? (
          <div className="container-table-wrap">
            <table className="container-table">
              <thead><tr><th>Name</th><th>Status</th><th>Image</th><th>Replicas</th><th>Updated</th></tr></thead>
              <tbody>
                {deployments.map((deployment) => (
                  <tr key={deployment.id}>
                    <td><Link href={`/containers/${encodeURIComponent(deployment.id)}`}>{deployment.name}</Link></td>
                    <td><span className="container-status" data-state={deployment.state}>{deployment.state.toLowerCase().replaceAll("_", " ")}</span></td>
                    <td>{deployment.spec.image}</td>
                    <td>{deployment.ready_replicas}/{deployment.desired_replicas}</td>
                    <td>{new Date(deployment.updated_at).toLocaleString()}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        ) : null}
      </section>
    </>
  );
}
