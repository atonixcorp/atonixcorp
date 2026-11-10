"use client";

import { useEffect, useState, type FormEvent } from "react";
import { CloudShell } from "@/app/shared/ui/CloudShell";
import { getNetworks, requestJSON, type Network, type Capabilities } from "./api";
import { initialScopeID } from "@/app/shared/lib/cloud-scope";
import { useCloudResource } from "@/app/shared/hooks/useCloudResource";
import { EmptyState, ResourcePagination, ResourceState, ResourceStatus, ResourceTable, SearchFilterBar } from "@/app/shared/ui/Infrastructure";

type Tenant = { id: number; name: string };
type Project = { id: number; tenant_id: number; name: string };
export default function NetworkingPage() {
  const [tenants, setTenants] = useState<Tenant[]>([]);
  const [projects, setProjects] = useState<Project[]>([]);
  const [tenantID, setTenantID] = useState("");
  const [projectID, setProjectID] = useState("");
  const [capabilities, setCapabilities] = useState<Capabilities | null>(null);
  const [revision, setRevision] = useState(0);
  const [search, setSearch] = useState("");
  const [status, setStatus] = useState("");
  const [sort, setSort] = useState("asc");
  const [page, setPage] = useState(0);
  const networks = useCloudResource(getNetworks, Number(tenantID), Number(projectID), revision);
  const [name, setName] = useState("");
  const [cidr, setCIDR] = useState("");
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");
  const [notice, setNotice] = useState("");

  useEffect(() => {
    let active = true;
    Promise.all([
      requestJSON<Tenant[]>("/api/storage/tenants"),
      requestJSON<Capabilities>("/api/networking/capabilities"),
    ])
      .then(([availableTenants, serviceCapabilities]) => {
        if (!active) return;
        setTenants(availableTenants);
        setCapabilities(serviceCapabilities);
        setTenantID(String(initialScopeID(availableTenants, "tenant_id") || ""));
      })
      .catch((cause: unknown) => {
        if (active) setError(cause instanceof Error ? cause.message : "Networking could not be loaded.");
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
        const initialProject = initialScopeID(items, "project_id", Number(tenantID));
        setProjectID((current) => items.some((item) => String(item.id) === current)
          ? current
          : String(initialProject || ""));
      })
      .catch((cause: unknown) => {
        if (active) setError(cause instanceof Error ? cause.message : "Projects could not be loaded.");
      });
    return () => { active = false; };
  }, [tenantID]);

  async function createNetwork(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!tenantID || !projectID) {
      setError("Select a workspace and project before creating a network.");
      return;
    }
    setSaving(true);
    setError("");
    setNotice("");
    const networkCIDR = cidr;
    try {
      const created = await requestJSON<Network>(
        `/api/networking/tenants/${encodeURIComponent(tenantID)}/projects/${encodeURIComponent(projectID)}/networks`,
        {
          method: "POST",
          body: JSON.stringify({ name, cidr }),
        },
      );
      setName("");
      setCIDR("");
      try {
        await requestJSON(
          `/api/networking/tenants/${encodeURIComponent(tenantID)}/projects/${encodeURIComponent(projectID)}/networks/${encodeURIComponent(created.id)}/subnets`,
          {
            method: "POST",
            body: JSON.stringify({ network_id: created.id, cidr: networkCIDR, gateway_ip: "" }),
          },
        );
        await requestJSON<Network[]>(
          `/api/networking/tenants/${encodeURIComponent(tenantID)}/projects/${encodeURIComponent(projectID)}/networks`,
        );
        setNotice("Network and subnet were created and verified.");
      } catch (subnetError) {
        setError(`Network was created, but its subnet could not be verified: ${
          subnetError instanceof Error ? subnetError.message : "Networking service returned an unknown error."
        }`);
      }
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : "Network could not be created.");
    } finally {
      setSaving(false);
      setRevision((value) => value + 1);
    }
  }

  return (
    <CloudShell>
    <main className="network-console">
      <section className="network-content">
        <header className="network-heading">
          <div>
            <p className="network-eyebrow">CLOUD CONNECTIVITY</p>
            <h1>VPC Networks</h1>
            <p>Manage project networks and their address ranges.</p>
          </div>
          <div className="network-scope">
            <label>
              <span>Workspace</span>
              <select value={tenantID} onChange={(event) => {
                setTenantID(event.target.value);
                setProjects([]);
                setProjectID("");
                setError("");
                setNotice("");
                setSearch("");
                setStatus("");
              }} disabled={loading || saving}>
                {!tenantID && <option value="">Choose workspace</option>}
                {tenants.map((tenant) => <option value={tenant.id} key={tenant.id}>{tenant.name}</option>)}
              </select>
            </label>
            <label>
              <span>Project</span>
              <select value={projectID} onChange={(event) => {
                setProjectID(event.target.value); setError(""); setNotice(""); setSearch(""); setStatus("");
              }} disabled={loading || saving || !projects.length}>
                {!projectID && <option value="">Choose project</option>}
                {projects.map((project) => <option value={project.id} key={project.id}>{project.name}</option>)}
              </select>
            </label>
          </div>
        </header>
        {error && <p className="network-alert" role="alert">{error}</p>}
        {notice && <p className="network-notice" role="status">{notice}</p>}
        {capabilities && !capabilities.network_provisioning && (
          <section className="network-unavailable" role="status">
            <h2>Network provisioning is not enabled</h2>
            <p>The service is available for viewing, but networking is not configured to apply and confirm changes.</p>
          </section>
        )}
        <div className="network-layout">
          <section className="network-panel">
            <header><div><p className="network-eyebrow">PROJECT INVENTORY</p><h2>Virtual networks</h2><p>{networks.state === "ready" ? `${networks.data.length} total` : networks.state === "loading" ? "Loading networks..." : "Inventory unavailable"}</p></div>
              <button type="button" onClick={() => setRevision((value) => value + 1)} disabled={!projectID || networks.state === "loading"}>Refresh</button>
            </header>
            <SearchFilterBar search={search} onSearch={(value) => { setSearch(value); setPage(0); }} status={status} onStatus={(value) => { setStatus(value); setPage(0); }}
              statuses={networks.state === "ready" ? [...new Set(networks.data.map((item) => item.status || "unknown"))] : []} />
            <label>Sort by name <select value={sort} onChange={(event) => { setSort(event.target.value); setPage(0); }}><option value="asc">A to Z</option><option value="desc">Z to A</option></select></label>
            <ResourceState resource={networks}>{(items) => {
              const filtered = items.filter((item) => (!status || (item.status || "unknown") === status) &&
                `${item.name} ${item.cidr} ${item.status ?? ""}`.toLowerCase().includes(search.trim().toLowerCase()))
                .sort((a, b) => a.name.localeCompare(b.name) * (sort === "asc" ? 1 : -1));
              if (!items.length) return <EmptyState title="No networks in this project" description="Create a virtual network and address range for your infrastructure." action={<a href="#create-network">Create Network</a>} />;
              if (!filtered.length) return <EmptyState title="No matching networks" description="Try another search or clear the status filter." />;
              const currentPage = Math.min(page, Math.ceil(filtered.length / 20) - 1);
              return <><ResourceTable caption="Project networks" headings={["Name", "IPv4 / IPv6 range", "Subnet", "Status"]}>
                {filtered.slice(currentPage * 20, currentPage * 20 + 20).map((network) => <tr key={network.id}><td>{network.name}</td><td>{network.cidr}</td><td>{network.subnet_status ?? "Not created"}</td><td><ResourceStatus value={network.status || "unknown"} /></td></tr>)}
              </ResourceTable><ResourcePagination page={currentPage} total={filtered.length} pageSize={20} onPage={setPage} /></>;
            }}</ResourceState>
          </section>
          <section className="network-panel" id="create-network">
            <header><div><p className="network-eyebrow">ADDRESS PLANNING</p><h2>Create a virtual network</h2></div></header>
            <form onSubmit={createNetwork}>
              <label><span>Network name</span><input value={name} onChange={(event) => setName(event.target.value)} required minLength={2} maxLength={63} /></label>
              <label><span>Network CIDR</span><input value={cidr} onChange={(event) => setCIDR(event.target.value)} placeholder="10.40.0.0/16" required /></label>
              <button type="submit" disabled={saving || loading || !capabilities?.network_provisioning || !projectID}>
                {saving ? "Submitting…" : "Create network"}
              </button>
            </form>
          </section>
        </div>
        <p>Network update and deletion are not connected in this console. Reported provisioning state is not a connectivity health check.</p>
      </section>
      <style jsx>{`
        .network-console{min-height:100vh;background:var(--color-background);color:var(--color-text-primary);font-family:var(--font-sans)}
        .network-content{max-width:1200px;margin:0 auto;padding:42px 24px}.network-heading{display:flex;justify-content:space-between;gap:32px;align-items:end;margin-bottom:28px}
        h1{font-size:var(--font-size-title);margin:8px 0}.network-heading p{color:var(--color-text-secondary)}.network-eyebrow{font-size:var(--font-size-caption);letter-spacing:.12em;color:var(--color-text-muted);margin:0}
        .network-scope{display:flex;gap:12px}.network-scope label,.network-panel form label{display:grid;gap:7px;font-size:13px;font-weight:600}
        select,input{min-height:40px;padding:0 10px;border:1px solid var(--color-border);border-radius:var(--radius-sm);background:var(--color-surface);color:var(--color-text-primary)}
        .network-layout{display:grid;grid-template-columns:1.2fr .8fr;gap:18px}.network-panel,.network-unavailable{background:var(--color-surface);border:1px solid var(--color-border);border-radius:var(--radius-lg);padding:22px}
        .network-panel header{display:flex;justify-content:space-between;align-items:center;border-bottom:1px solid #eef1f5;padding-bottom:15px;margin-bottom:16px}
        h2{font-size:var(--font-size-heading);margin:6px 0}.network-panel table{width:100%;border-collapse:collapse;text-align:left}.network-panel th,.network-panel td{padding:12px 8px;border-bottom:1px solid var(--color-border);font-size:var(--font-size-body)}
        .network-empty{color:var(--color-text-secondary);padding:20px 4px}.network-panel form{display:grid;gap:16px}.network-panel button{height:42px;border:0;border-radius:var(--radius-sm);background:var(--color-primary);color:white;font-weight:700;cursor:pointer}.network-panel button:hover:not(:disabled){background:var(--color-primary-hover)}.network-panel button:disabled{background:var(--color-disabled);cursor:not-allowed}
        .network-unavailable{border-color:#f1d59a;background:#fffaf0;margin-bottom:18px}.network-unavailable p{margin-bottom:0;color:var(--color-warning)}.network-alert,.network-notice{padding:12px 16px;border-radius:var(--radius-md)}.network-alert{background:#fff1f0;color:var(--color-error)}.network-notice{background:#e9f7ef;color:var(--color-success)}
        @media(max-width:760px){.network-heading{display:block}.network-scope{margin-top:20px;flex-direction:column}.network-layout{grid-template-columns:1fr}}
      `}</style>
    </main>
    </CloudShell>
  );
}
