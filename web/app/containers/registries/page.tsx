"use client";

import Link from "next/link";
import { FormEvent, useCallback, useEffect, useState } from "react";
import { createRegistry, getProjects, getRegistries, getTenants } from "../lib/api";
import type { ContainerRegistry, Project, Tenant } from "../lib/api";

function messageOf(error: unknown): string {
  return error instanceof Error ? error.message : "The registry request could not be completed.";
}

export default function ContainerRegistriesPage() {
  const [tenants, setTenants] = useState<Tenant[]>([]);
  const [projects, setProjects] = useState<Project[]>([]);
  const [tenantID, setTenantID] = useState(0);
  const [projectID, setProjectID] = useState(0);
  const [registries, setRegistries] = useState<ContainerRegistry[]>([]);
  const [name, setName] = useState("");
  const [url, setURL] = useState("");
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");
  const [notice, setNotice] = useState("");

  useEffect(() => {
    let active = true;
    void getTenants().then((items) => {
      if (!active) return;
      setTenants(items);
      setTenantID(items[0]?.id ?? 0);
      if (!items.length) setLoading(false);
    }).catch((loadError: unknown) => {
      if (active) {
        setError(messageOf(loadError));
        setLoading(false);
      }
    });
    return () => { active = false; };
  }, []);

  useEffect(() => {
    if (!tenantID) return;
    let active = true;
    void getProjects(tenantID).then((items) => {
      if (active) {
        setProjects(items);
        setProjectID((current) => items.some((item) => item.id === current) ? current : items[0]?.id ?? 0);
      }
    }).catch((loadError: unknown) => {
      if (active) {
        setError(messageOf(loadError));
        setLoading(false);
      }
    });
    return () => { active = false; };
  }, [tenantID]);

  const loadRegistries = useCallback(async () => {
    if (!tenantID || !projectID) {
      setRegistries([]);
      setLoading(false);
      return;
    }
    setLoading(true);
    setError("");
    try {
      setRegistries(await getRegistries(tenantID, projectID));
    } catch (loadError) {
      setError(messageOf(loadError));
    } finally {
      setLoading(false);
    }
  }, [tenantID, projectID]);

  useEffect(() => {
    if (!projectID) return;
    const timer = window.setTimeout(() => { void loadRegistries(); }, 0);
    return () => window.clearTimeout(timer);
  }, [loadRegistries, projectID]);

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setSaving(true);
    setError("");
    setNotice("");
    try {
      await createRegistry({
        tenant_id: tenantID,
        project_id: projectID,
        name: name.trim(),
        url: url.trim(),
        ...(username ? { username } : {}),
        ...(password ? { password } : {}),
      });
      setName("");
      setURL("");
      setUsername("");
      setPassword("");
      setNotice("Registry saved. Credentials are encrypted and are never returned by the API.");
      await loadRegistries();
    } catch (saveError) {
      setError(messageOf(saveError));
    } finally {
      setSaving(false);
    }
  }

  return (
    <>
      <header className="container-heading">
        <div>
          <p className="container-eyebrow"><Link href="/containers">CONTAINERS</Link> / REGISTRIES</p>
          <h1>Container registries</h1>
          <p>Save project-scoped HTTPS registry endpoints and optional credentials.</p>
        </div>
        <Link className="container-button secondary" href="/containers">← Deployments</Link>
      </header>

      <div className="container-card container-registry-scope">
        <label className="container-field">
          <span>Workspace</span>
          <select value={tenantID || ""} onChange={(event) => {
            setProjectID(0);
            setRegistries([]);
            setTenantID(Number(event.target.value));
          }} disabled={!tenants.length}>
            {tenants.map((tenant) => <option key={tenant.id} value={tenant.id}>{tenant.name}</option>)}
          </select>
        </label>
        <label className="container-field">
          <span>Project</span>
          <select value={projectID || ""} onChange={(event) => setProjectID(Number(event.target.value))} disabled={!projects.length}>
            {projects.map((project) => <option key={project.id} value={project.id}>{project.name}</option>)}
          </select>
        </label>
      </div>

      {error && <div className="container-alert" role="alert">{error}{error.toLowerCase().includes("authentication required") && <> <Link href="/login">Sign in</Link> to continue.</>}</div>}
      {notice && <div className="container-success" role="status">{notice}</div>}

      <section className="container-card container-registry-form-card">
        <div className="container-registry-section-heading">
          <h2>Add a registry</h2>
          <p>Credentials are stored encrypted. Use a scoped robot or service account where possible.</p>
        </div>
        <form className="container-registry-form" onSubmit={(event) => void submit(event)}>
          <label className="container-field">
            <span>Registry name</span>
            <input value={name} onChange={(event) => setName(event.target.value)} minLength={2} maxLength={100} placeholder="production-images" required />
          </label>
          <label className="container-field">
            <span>HTTPS registry URL</span>
            <input type="url" value={url} onChange={(event) => setURL(event.target.value)} placeholder="https://registry.example.com" required />
            <small>Do not include credentials or query values in the URL.</small>
          </label>
          <div className="container-field-row">
            <label className="container-field">
              <span>Username (optional)</span>
              <input value={username} onChange={(event) => setUsername(event.target.value)} autoComplete="username" />
            </label>
            <label className="container-field">
              <span>Password / access token (optional)</span>
              <input type="password" value={password} onChange={(event) => setPassword(event.target.value)} autoComplete="new-password" />
            </label>
          </div>
          <button type="submit" className="container-button" disabled={saving || !tenantID || !projectID}>{saving ? "Saving…" : "Save registry"}</button>
        </form>
      </section>

      <section className="container-card container-registry-list-card">
        <header className="container-toolbar">
          <div><h2>Saved registries</h2><p>{loading ? "Loading…" : `${registries.length} registries in this project`}</p></div>
          <button className="container-button secondary" type="button" onClick={() => void loadRegistries()} disabled={loading || !projectID}>Refresh</button>
        </header>
        {loading ? <div className="container-loading" role="status">Loading registries…</div> : registries.length ? (
          <div className="container-table-wrap">
            <table className="container-table">
              <thead><tr><th>Name</th><th>Registry URL</th><th>Credentials</th><th>Added</th></tr></thead>
              <tbody>{registries.map((registry) => (
                <tr key={registry.id}>
                  <td>{registry.name}</td>
                  <td>{registry.url}</td>
                  <td>{registry.credentials_stored ? "Stored encrypted" : "Public registry"}</td>
                  <td>{new Date(registry.created_at).toLocaleString()}</td>
                </tr>
              ))}</tbody>
            </table>
          </div>
        ) : !error ? (
          <div className="container-empty">
            <h2>No registries configured</h2>
            <p>Add a registry endpoint for this project. Saved credentials are not yet connected to workload image-pull secrets.</p>
          </div>
        ) : null}
      </section>
      <div className="container-info">Security note: only HTTPS registry URLs are accepted. Registry credentials are encrypted at rest but are not yet attached to Kubernetes image pulls.</div>
    </>
  );
}
