"use client";

import Link from "next/link";
import { Suspense, useState } from "react";
import { useCloudScope } from "@/app/shared/hooks/useCloudScope";
import { useCloudResource } from "@/app/shared/hooks/useCloudResource";
import { CloudShell } from "@/app/shared/ui/CloudShell";
import { CloudPageHeader, EmptyState, ResourcePanel, ResourcePagination, ResourceState, ResourceTable } from "@/app/shared/ui/Infrastructure";
import { getFirewallGroups } from "../api";
import styles from "./page.module.css";

export default function FirewallGroupsRoute() {
  return <CloudShell><Suspense fallback={<p role="status">Loading firewall workspace...</p>}><FirewallGroups /></Suspense></CloudShell>;
}

function FirewallGroups() {
  const [revision, setRevision] = useState(0);
  const [search, setSearch] = useState("");
  const [direction, setDirection] = useState("");
  const [sort, setSort] = useState("asc");
  const [page, setPage] = useState(0);
  const scope = useCloudScope(revision);
  const groups = useCloudResource(getFirewallGroups, scope.tenantID, scope.projectID, revision);
  const reset = () => { setSearch(""); setDirection(""); setPage(0); };
  return <main className={styles.page}>
    <CloudPageHeader title="Firewall Groups" description="Inspect project-authorized firewall groups and their traffic rules."
      actions={<button type="button" className={styles.button} onClick={() => { setPage(0); setRevision((value) => value + 1); }}>Refresh</button>} />
    <div className={styles.scope}>
      <ResourceState resource={scope.tenants}>{(items) => items.length ? <label>Workspace<select value={scope.tenantID || ""} onChange={(event) => { scope.selectWorkspace(Number(event.target.value)); reset(); }}>
        <option value="" disabled>Choose workspace</option>{items.map((item) => <option value={item.id} key={item.id}>{item.name}</option>)}
      </select></label> : <EmptyState title="No accessible workspace" description="Create a workspace or ask its owner for access." action={<Link href="/dashboard">Open dashboard</Link>} />}</ResourceState>
      <ResourceState resource={scope.projects}>{(items) => items.length ? <label>Project<select value={scope.projectID || ""} onChange={(event) => { scope.selectProject(Number(event.target.value)); reset(); }}>
        <option value="" disabled>Choose project</option>{items.map((item) => <option value={item.id} key={item.id}>{item.name}</option>)}
      </select></label> : <EmptyState title="No accessible project" description="Ask your workspace owner to grant project access." />}</ResourceState>
    </div>
    {scope.error && <p role="alert" className={styles.error}>{scope.error}</p>}
    <p className={styles.notice}>Read-only integration. Production group creation, rule editing, resource assignments and deletion are not connected. Existing backend test-group creation is deliberately not exposed as production firewall management.</p>
    <ResourcePanel title="Firewall inventory" description="Group presence is not proof of effective protection or network health.">
      <div className={styles.filters}>
        <label>Search groups<input type="search" value={search} placeholder="Name, description or rule" onChange={(event) => { setSearch(event.target.value); setPage(0); }} /></label>
        <label>Rule direction<select value={direction} onChange={(event) => { setDirection(event.target.value); setPage(0); }}><option value="">All directions</option><option value="ingress">Ingress</option><option value="egress">Egress</option></select></label>
        <label>Sort by name<select value={sort} onChange={(event) => { setSort(event.target.value); setPage(0); }}><option value="asc">A to Z</option><option value="desc">Z to A</option></select></label>
      </div>
      <ResourceState resource={groups}>{(items) => {
        const query = search.trim().toLowerCase();
        const filtered = items.filter((item) =>
          (!direction || item.rules.some((rule) => rule.direction === direction)) &&
          `${item.name} ${item.description} ${item.rules.map((rule) => `${rule.protocol} ${rule.remote_ip_prefix ?? ""}`).join(" ")}`.toLowerCase().includes(query),
        ).sort((a, b) => a.name.localeCompare(b.name) * (sort === "asc" ? 1 : -1));
        if (!items.length) return <EmptyState title="No firewall groups returned" description="The service returned no groups for this scope. Creation and rule management are not available in this console." />;
        if (!filtered.length) return <EmptyState title="No matching groups" description="Clear the filters or try a different search." />;
        const currentPage = Math.min(page, Math.ceil(filtered.length / 20) - 1);
        return <>
          <ResourceTable caption="Authorized firewall groups" headings={["Group", "Description", "Rules"]}>
            {filtered.slice(currentPage * 20, currentPage * 20 + 20).map((group, index) => <tr key={`${group.name}:${index}`}>
              <td>{group.name}</td><td className={styles.description}>{group.description || "No description"}</td>
              <td><details><summary>{group.rules.length} rules - view details</summary>
                {group.rules.length ? <ul className={styles.rules}>{group.rules.map((rule, ruleIndex) => <li key={ruleIndex}>
                  <strong>{rule.direction} / {rule.ethertype} / {rule.protocol || "Any protocol"}</strong>
                  <span>Ports: {rule.port_range_min === undefined ? "Any / not specified" : `${rule.port_range_min}${rule.port_range_max === undefined ? "" : ` - ${rule.port_range_max}`}`}</span>
                  <span>Remote CIDR: {rule.remote_ip_prefix || "Not specified by API"}</span>
                </li>)}</ul> : <p>No rules reported.</p>}
              </details></td>
            </tr>)}
          </ResourceTable>
          <ResourcePagination page={currentPage} total={filtered.length} pageSize={20} onPage={setPage} />
        </>;
      }}</ResourceState>
    </ResourcePanel>
  </main>;
}
