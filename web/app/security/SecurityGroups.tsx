"use client";

import { useCallback, useEffect, useState, type FormEvent } from "react";
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

export function SecurityGroups({ tenantID, projectID }: { tenantID: string; projectID: string }) {
  const [groups, setGroups] = useState<GroupResponse[]>([]);
  const [name, setName] = useState("atonixcorp-test-security");
  const [loading, setLoading] = useState(true);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState("");
  const [notice, setNotice] = useState("");

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
      setError(cause instanceof Error ? cause.message : "Could not retrieve security groups.");
    } finally {
      setLoading(false);
    }
  }, [tenantID, projectID]);

  useEffect(() => {
    const timeout = window.setTimeout(() => { void refresh(); }, 0);
    return () => window.clearTimeout(timeout);
  }, [refresh]);

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
      setNotice(`Security group "${group.name}" created.`);
      setName("");
      await refresh();
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : "Could not create the security group.");
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <section className="security-provider-panel" aria-labelledby="security-groups-title">
      <div className="security-panel-heading">
        <div>
          <p className="security-eyebrow">NETWORK SECURITY</p>
          <h2 id="security-groups-title">Security groups</h2>
        </div>
        <button type="button" className="security-button secondary" onClick={() => void refresh()} disabled={loading}>
          {loading ? "Refreshing…" : "Refresh"}
        </button>
      </div>
      <p className="security-muted">
        Groups below belong to the selected project. Creating a group adds only
        TCP 443 ingress from the documentation-only range 192.0.2.0/24.
      </p>
      {error && <p className="security-error" role="alert">{error}</p>}
      {notice && <p className="security-saved" role="status">{notice}</p>}
      <form className="security-provider-create" onSubmit={createGroup}>
        <label>New group name
          <input value={name} onChange={(event) => setName(event.target.value)} minLength={1} maxLength={63} required />
        </label>
        <button type="submit" className="security-button primary" disabled={submitting || loading || !projectID}>
          {submitting ? "Creating…" : "Create test security group"}
        </button>
      </form>
      {loading ? <p className="security-table-empty">Loading security groups…</p> : groups.length === 0
        ? <p className="security-table-empty">No security groups returned for this project.</p>
        : <div className="security-provider-table-wrap">
          <table className="security-provider-table">
            <thead><tr><th>Name</th><th>Rules</th></tr></thead>
            <tbody>{groups.map((group) => (
              <tr key={group.name}>
                <td>{group.name}<small>{group.description}</small></td>
                <td>{group.rules.map((rule) => (
                  <small key={`${rule.direction}-${rule.protocol}-${rule.port_range_min ?? ""}-${rule.remote_ip_prefix ?? ""}`}>
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
