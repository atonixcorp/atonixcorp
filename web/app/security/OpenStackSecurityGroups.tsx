"use client";

import { useCallback, useEffect, useState, type FormEvent } from "react";
import { getProjects, getTenants, type ComputeProject, type ComputeTenant } from "@/app/compute/lib/api";
import type { ComputeFirewall } from "@/app/compute/lib/types";

type GroupResponse = ComputeFirewall & { rules: NonNullable<ComputeFirewall["rules"]> };

async function requestJSON<T>(path: string, init?: RequestInit): Promise<T> {
  const response = await fetch(path, {
    ...init,
    cache: "no-store",
    headers: {
      ...(init?.body ? { "Content-Type": "application/json" } : {}),
      ...init?.headers,
    },
  });
  const body = await response.json().catch(() => null) as { error?: string } | T | null;
  if (!response.ok) {
    throw new Error(body && typeof body === "object" && "error" in body
      ? body.error || `Request failed (${response.status}).`
      : `Request failed (${response.status}).`);
  }
  if (body === null) throw new Error("The security API returned an empty response.");
  return body as T;
}

export function OpenStackSecurityGroups() {
  const [tenants, setTenants] = useState<ComputeTenant[]>([]);
  const [projects, setProjects] = useState<ComputeProject[]>([]);
  const [tenantID, setTenantID] = useState("");
  const [projectID, setProjectID] = useState("");
  const [groups, setGroups] = useState<GroupResponse[]>([]);
  const [name, setName] = useState("atonixcorp-test-security");
  const [loading, setLoading] = useState(true);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState("");
  const [notice, setNotice] = useState("");

  useEffect(() => {
    let active = true;
    void getTenants().then((items) => {
      if (!active) return;
      setTenants(items);
      setTenantID(items[0] ? String(items[0].id) : "");
      if (items.length === 0) setLoading(false);
    }).catch((cause: unknown) => {
      if (active) {
        setError(cause instanceof Error ? cause.message : "Could not load workspaces.");
        setLoading(false);
      }
    });
    return () => { active = false; };
  }, []);

  useEffect(() => {
    let active = true;
    setProjects([]);
    setProjectID("");
    if (!tenantID) return () => { active = false; };
    void getProjects(Number(tenantID)).then((items) => {
      if (!active) return;
      setProjects(items);
      setProjectID(items[0] ? String(items[0].id) : "");
    }).catch((cause: unknown) => {
      if (active) setError(cause instanceof Error ? cause.message : "Could not load projects.");
    });
    return () => { active = false; };
  }, [tenantID]);

  const refresh = useCallback(async () => {
    if (!tenantID || !projectID) {
      setGroups([]);
      setLoading(false);
      return;
    }
    setLoading(true);
    setError("");
    try {
      const query = new URLSearchParams({ tenant_id: tenantID, project_id: projectID });
      setGroups(await requestJSON<GroupResponse[]>(`/api/compute/firewalls?${query}`));
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : "Could not retrieve OpenStack security groups.");
    } finally {
      setLoading(false);
    }
  }, [tenantID, projectID]);

  useEffect(() => { void refresh(); }, [refresh]);

  async function createGroup(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!tenantID || !projectID) return;
    setSubmitting(true);
    setError("");
    setNotice("");
    try {
      const query = new URLSearchParams({ tenant_id: tenantID, project_id: projectID });
      const group = await requestJSON<GroupResponse>(`/api/compute/firewalls?${query}`, {
        method: "POST",
        body: JSON.stringify({ name }),
      });
      setNotice(`OpenStack security group created: ${group.id}`);
      setName("");
      await refresh();
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : "OpenStack could not create the security group.");
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <section className="security-provider-panel" aria-labelledby="openstack-security-groups-title">
      <div className="security-panel-heading">
        <div>
          <p className="security-eyebrow">LIVE OPENSTACK RESOURCE</p>
          <h2 id="openstack-security-groups-title">Security groups</h2>
        </div>
        <button type="button" className="security-button secondary" onClick={() => void refresh()} disabled={loading}>
          {loading ? "Refreshing…" : "Refresh"}
        </button>
      </div>
      <p className="security-muted">
        Groups and IDs below come from the organization&apos;s mapped OpenStack project. Creating a group adds only
        TCP 443 ingress from the documentation-only range 192.0.2.0/24.
      </p>
      <div className="security-provider-scope">
        <label>Workspace
          <select value={tenantID} onChange={(event) => setTenantID(event.target.value)} disabled={!tenants.length}>
            {tenants.map((tenant) => <option key={tenant.id} value={tenant.id}>{tenant.name}</option>)}
          </select>
        </label>
        <label>Project
          <select value={projectID} onChange={(event) => setProjectID(event.target.value)} disabled={!projects.length}>
            {projects.map((project) => <option key={project.id} value={project.id}>{project.name}</option>)}
          </select>
        </label>
      </div>
      {error && <p className="security-error" role="alert">{error}</p>}
      {notice && <p className="security-saved" role="status">{notice}</p>}
      {!tenants.length && !loading && !error && <p className="security-table-empty">No accessible workspaces.</p>}
      <form className="security-provider-create" onSubmit={createGroup}>
        <label>New group name
          <input value={name} onChange={(event) => setName(event.target.value)} minLength={1} maxLength={63} required />
        </label>
        <button type="submit" className="security-button primary" disabled={submitting || loading || !projectID}>
          {submitting ? "Creating…" : "Create test security group"}
        </button>
      </form>
      {loading ? <p className="security-table-empty">Loading provider security groups…</p> : groups.length === 0
        ? <p className="security-table-empty">No security groups returned for this project.</p>
        : <div className="security-provider-table-wrap">
          <table className="security-provider-table">
            <thead><tr><th>Name</th><th>OpenStack ID</th><th>Rules</th></tr></thead>
            <tbody>{groups.map((group) => (
              <tr key={group.id}>
                <td>{group.name}<small>{group.description}</small></td>
                <td><code>{group.id}</code></td>
                <td>{group.rules.map((rule) => (
                  <small key={rule.id}>
                    {rule.direction} {rule.protocol || "any"}
                    {rule.port_range_min ? `:${rule.port_range_min}${rule.port_range_max !== rule.port_range_min ? `-${rule.port_range_max}` : ""}` : ""}
                    {rule.remote_ip_prefix ? ` from ${rule.remote_ip_prefix}` : ""}
                  </small>
                ))}</td>
              </tr>
            ))}</tbody>
          </table>
        </div>}
    </section>
  );
}
