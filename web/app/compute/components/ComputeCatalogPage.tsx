"use client";

import Link from "next/link";
import { useState } from "react";
import { getCatalog, getProjects, getTenants } from "../lib/api";
import { useCloudResource } from "@/app/shared/hooks/useCloudResource";
import { CloudPageHeader, EmptyState, ResourcePanel, ResourceState, ResourceStatus, ResourceTable, SearchFilterBar } from "@/app/shared/ui/Infrastructure";

export function ComputeCatalogPage({ section }: { section: "images" | "sizes" }) {
  const [workspace, setWorkspace] = useState(0);
  const [project, setProject] = useState(0);
  const [revision, setRevision] = useState(0);
  const [search, setSearch] = useState("");
  const [status, setStatus] = useState("");
  const [page, setPage] = useState(0);
  const tenants = useCloudResource(getTenants, 1, 1, revision);
  const tenantID = tenants.state === "ready" ? tenants.data.find((item) => item.id === workspace)?.id ?? tenants.data[0]?.id ?? 0 : 0;
  const projects = useCloudResource(getProjects, tenantID, tenantID, revision);
  const projectID = projects.state === "ready" ? projects.data.find((item) => item.id === project)?.id ?? projects.data[0]?.id ?? 0 : 0;
  const catalog = useCloudResource(getCatalog, tenantID, projectID, revision);
  const title = section === "images" ? "Images" : "Instance sizes";

  return <div className="compute-page">
    <CloudPageHeader title={title} description={section === "images"
      ? "Browse images available to your project. Image upload and lifecycle management are not connected."
      : "Compare the CPU, memory and disk specifications available for new instances."}
      actions={projectID
        ? <Link className="compute-primary-button" href={`/compute/deploy?tenant_id=${tenantID}&project_id=${projectID}`}>Create Instance</Link>
        : <button className="compute-primary-button" disabled>Select a project</button>} />
    <ResourcePanel title={`Compute ${section === "images" ? "images" : "sizes"}`} action={<button className="compute-secondary-button" onClick={() => { setPage(0); setRevision((value) => value + 1); }}>Refresh</button>}>
      <div className="compute-table-toolbar">
        <ResourceState resource={tenants}>{(items) => items.length ? <label className="compute-filter">Workspace
          <select value={tenantID} onChange={(event) => { setWorkspace(Number(event.target.value)); setProject(0); setPage(0); }}>
            {items.map((item) => <option key={item.id} value={item.id}>{item.name}</option>)}
          </select></label> : <EmptyState title="No accessible workspace" description="Create a workspace from the Cloud dashboard before using the catalog." action={<Link href="/dashboard">Open dashboard</Link>} />}</ResourceState>
        <ResourceState resource={projects}>{(items) => items.length ? <label className="compute-filter">Project
          <select value={projectID} onChange={(event) => { setProject(Number(event.target.value)); setPage(0); }}>
            {items.map((item) => <option key={item.id} value={item.id}>{item.name}</option>)}
          </select></label> : <EmptyState title="No accessible project" description="Ask your workspace owner for project access." />}</ResourceState>
      </div>
      <SearchFilterBar search={search} onSearch={(value) => { setSearch(value); setPage(0); }} status={status}
        onStatus={(value) => { setStatus(value); setPage(0); }} statuses={section === "images" && catalog.state === "ready"
          ? [...new Set(catalog.data.images.map((item) => item.status))] : ["Available", "Unavailable"]} />
      <ResourceState resource={catalog}>{(data) => {
        const rows = section === "images"
          ? data.images.map((item) => ({ id: item.id, name: item.name, state: item.status, cpu: null, memory: item.min_memory_mb, disk: item.min_disk_gb }))
          : data.plans.map((item) => ({ id: item.id, name: item.name, state: item.available ? "Available" : "Unavailable", cpu: item.vcpus, memory: item.memory_mb, disk: item.boot_disk_gb }));
        const filtered = rows.filter((item) => (!status || item.state === status) && `${item.name} ${item.id} ${item.state}`.toLowerCase().includes(search.trim().toLowerCase()));
        if (!rows.length) return <EmptyState title={`No ${title.toLowerCase()} available`} description="The compute catalog returned no resources for this project. Contact your workspace administrator to configure available resources." />;
        if (!filtered.length) return <EmptyState title="No matching resources" description="Clear the filters or try a different search." />;
        const currentPage = Math.min(page, Math.ceil(filtered.length / 20) - 1);
        return <>
          <ResourceTable caption={`Project ${title.toLowerCase()}`} headings={section === "images" ? ["Image", "Status", "Minimum RAM", "Minimum disk"] : ["Size", "Status", "vCPU", "RAM", "Boot disk"]}>
            {filtered.slice(currentPage * 20, currentPage * 20 + 20).map((item) => <tr key={item.id}>
              <td>{item.name}</td><td><ResourceStatus value={item.state} /></td>
              {section === "sizes" && <td>{item.cpu}</td>}<td>{item.memory} MB</td><td>{item.disk} GB</td>
            </tr>)}
          </ResourceTable>
          <div className="compute-table-toolbar">
            <span>{filtered.length} resources / page {currentPage + 1} of {Math.ceil(filtered.length / 20)}</span>
            <div><button className="compute-secondary-button" disabled={currentPage === 0} onClick={() => setPage(currentPage - 1)}>Previous</button>{" "}
              <button className="compute-secondary-button" disabled={(currentPage + 1) * 20 >= filtered.length} onClick={() => setPage(currentPage + 1)}>Next</button></div>
          </div>
        </>;
      }}</ResourceState>
    </ResourcePanel>
  </div>;
}
