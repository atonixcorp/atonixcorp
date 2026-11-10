"use client";

import Link from "next/link";
import { useState } from "react";
import { getCatalog, getInstances, type ComputeTenant } from "@/app/compute/lib/api";
import { getOverview as getStorage, getVolumes, getVolumeTypes } from "@/app/storage/lib/api";
import { getOverview as getHealth, getMetrics, getEvents, getAlerts } from "@/app/observability/lib/api";
import { getDeployments } from "@/app/containers/lib/api";
import { getNetworks } from "@/app/networking/api";
import { useCloudResource, type CloudResource } from "@/app/shared/hooks/useCloudResource";
import { useCloudScope } from "@/app/shared/hooks/useCloudScope";
import {
  EmptyState, QuickActions, ResourceGrid, ResourcePanel, ResourceState,
  ResourceStatus, ResourceSummaryCard, ResourceTable, SearchFilterBar, UtilizationMeter,
} from "@/app/shared/ui/Infrastructure";
import { cloudNavigation } from "@/app/shared/ui/cloud-navigation";
import { allocatedCompute, utilizationSample, recentResources, remainingQuota } from "./infrastructure-summary";
import styles from "./InfrastructureOverview.module.css";

function summaryValue<T>(resource: CloudResource<T>, value: (data: T) => string | number) {
  return resource.state === "ready" ? value(resource.data) : resource.state === "loading" ? "Loading..." : "Unavailable";
}

export default function InfrastructureOverview({ workspaces, revision }: { workspaces: ComputeTenant[]; revision: number }) {
  const scope = useCloudScope(revision, workspaces);
  const { tenantID, projectID, projects } = scope;
  const [search, setSearch] = useState("");
  const [status, setStatus] = useState("");
  const instances = useCloudResource(getInstances, tenantID, projectID, revision);
  const catalog = useCloudResource(getCatalog, tenantID, projectID, revision);
  const networks = useCloudResource(getNetworks, tenantID, projectID, revision);
  const storage = useCloudResource(getStorage, tenantID, projectID, revision);
  const volumes = useCloudResource(getVolumes, tenantID, projectID, revision);
  const volumeTypes = useCloudResource(getVolumeTypes, tenantID, projectID, revision);
  const health = useCloudResource(getHealth, tenantID, projectID, revision);
  const metrics = useCloudResource(getMetrics, tenantID, projectID, revision);
  const events = useCloudResource(getEvents, tenantID, projectID, revision);
  const alerts = useCloudResource(getAlerts, tenantID, projectID, revision);
  const deployments = useCloudResource(getDeployments, tenantID, projectID, revision);
  const allocation = instances.state === "ready"
    ? allocatedCompute(instances.data, catalog.state === "ready" ? catalog.data.plans : undefined) : null;
  const scopedHref = scope.href;
  const createInstanceHref = scopedHref("/compute/deploy");
  const sources = [
    { name: "Compute instances", resource: instances }, { name: "Compute catalog", resource: catalog },
    { name: "Networking", resource: networks }, { name: "Storage summary", resource: storage },
    { name: "Volumes", resource: volumes }, { name: "Infrastructure health", resource: health },
    { name: "Metrics", resource: metrics }, { name: "Events", resource: events },
    { name: "Containers", resource: deployments },
    { name: "Storage zones", resource: volumeTypes }, { name: "Alerts", resource: alerts },
  ];
  const failures = sources.filter((source) => source.resource.state === "error");
  const loading = sources.some((source) => source.resource.state === "loading");
  const recent = recentResources([
    ...(instances.state === "ready" ? instances.data.map((item) => ({ ...item, kind: "Instance", href: `/compute/instances/${encodeURIComponent(item.id)}` })) : []),
    ...(volumes.state === "ready" ? volumes.data.map((item) => ({ ...item, status: item.state, kind: "Volume", href: "/storage?tab=volumes" })) : []),
    ...(deployments.state === "ready" ? deployments.data.map((item) => ({ ...item, status: item.state, kind: "Container", href: `/containers/${encodeURIComponent(item.id)}` })) : []),
  ]);

  return <div className={styles.workspace}>
    <section className={styles.context} id="cloud-context" aria-label="Infrastructure scope">
      <label>Workspace<select value={tenantID || ""} disabled={!workspaces.length} onChange={(event) => {
        scope.selectWorkspace(Number(event.target.value)); setSearch(""); setStatus("");
      }}>
        {!workspaces.length && <option value="">No accessible workspace</option>}
        {workspaces.map((item) => <option key={item.id} value={item.id}>{item.name}</option>)}
      </select></label>
      <label>Project<select value={projectID || ""} disabled={projects.state !== "ready" || !projects.data.length} onChange={(event) => {
        scope.selectProject(Number(event.target.value)); setSearch(""); setStatus("");
      }}>
        {!projectID && <option value="">{projects.state === "loading" ? "Loading projects..." : "No accessible project"}</option>}
        {projects.state === "ready" && projects.data.map((item) => <option key={item.id} value={item.id}>{item.name}</option>)}
      </select></label>
      <div><span>Compute region</span><strong>{catalog.state === "ready" ? catalog.data.regions.map((region) => region.name).join(", ") || "Not configured" : catalog.state === "loading" ? "Loading..." : "Unavailable"}</strong><small>Inventory below covers the selected project across regions.</small></div>
      <div><span>Storage availability zones</span><strong>{volumeTypes.state === "ready"
        ? [...new Set(volumeTypes.data.flatMap((type) => type.availability_zones))].join(", ") || "Not reported"
        : volumeTypes.state === "loading" ? "Loading..." : "Unavailable"}</strong><small>Reported storage placement options, not a zone filter.</small></div>
      <p role="status">{loading ? "Refreshing infrastructure..." : projectID ? "Project-scoped inventory" : "Choose a project to begin"}</p>
    </section>
    {scope.error && <p className={styles.error} role="alert">{scope.error}</p>}
    {projects.state === "error" && <p className={styles.error} role="alert">{projects.error} Use Refresh to retry.</p>}
    {projects.state === "ready" && !projects.data.length && <EmptyState title="No accessible projects" description="Ask your workspace owner to grant project access before managing infrastructure." />}

    <QuickActions>
      {projectID ? <>
        <Link href={createInstanceHref}>+ Create Instance</Link>
        <Link href={scopedHref("/storage?action=create")}>Create Volume</Link>
        <Link href={`${scopedHref("/networking")}#create-network`}>Create Network</Link>
        <Link href={scopedHref("/containers/deploy")}>Deploy Container</Link>
      </> : <button disabled>Select a project to create resources</button>}
    </QuickActions>

    {failures.length > 0 && <div className={styles.attention} role="status">
      <strong>{failures.length} data source{failures.length === 1 ? " requires" : "s require"} attention.</strong> Available resources remain visible; use Refresh to retry.
      <details><summary>View source errors</summary><ul>{sources.map(({ name, resource }) => resource.state === "error" && <li key={name}>{name}: {resource.error}</li>)}</ul></details>
    </div>}

    <ResourceGrid>
      <ResourceSummaryCard icon="compute" title="Compute" value={allocation ? `${allocation.vcpus} vCPU` : instances.state === "loading" || catalog.state === "loading" ? "Loading..." : "Unavailable"} detail="Allocated to listed instances, not free capacity" href={scopedHref("/compute/instances")} />
      <ResourceSummaryCard icon="compute" title="Memory" value={allocation ? `${allocation.memoryGB.toLocaleString()} GiB` : instances.state === "loading" || catalog.state === "loading" ? "Loading..." : "Unavailable"} detail="Instance size allocation, not live memory use" />
      <ResourceSummaryCard icon="storage" title="Block storage" value={summaryValue(storage, (data) => `${data.used_gb.toLocaleString()} GB`)} detail="Provisioned volume capacity" href={scopedHref("/storage")} />
      <ResourceSummaryCard icon="compute" title="Instances" value={summaryValue(instances, (data) => data.length)} detail={instances.state === "ready" ? `${instances.data.filter((item) => item.status === "running").length} running` : "Project instance inventory"} href={scopedHref("/compute/instances")} />
      <ResourceSummaryCard icon="networking" title="Networks" value={summaryValue(networks, (data) => data.length)} detail="Project networks, not connectivity health" href={scopedHref("/networking")} />
      <ResourceSummaryCard icon="storage" title="Volumes" value={summaryValue(storage, (data) => data.volumes)} detail="Managed block volumes" href={scopedHref("/storage")} />
    </ResourceGrid>

    <div className={styles.operations}>
      <ResourcePanel title="Capacity and service coverage" description="Allocation, quotas and physical capacity are different measurements">
        <div className={styles.health}>
          <div><span>Unallocated CPU / RAM capacity</span><ResourceStatus value="Not reported" /></div>
          <p>The compute API does not publish physical capacity or remaining CPU/RAM quotas. Allocated resources above must not be interpreted as available capacity.</p>
          <div><span>Remaining block storage quota</span><strong>{summaryValue(storage, (data) => {
            const remaining = remainingQuota(data.used_gb, data.quota_gb);
            return remaining === null ? "Unavailable" : `${remaining.toLocaleString()} GB`;
          })}</strong></div>
          <div><Link href={scopedHref("/cloud/clusters")}>Clusters / HPC</Link><ResourceStatus value="Not connected" /></div>
          <div><Link href={scopedHref("/cloud/kubernetes")}>Kubernetes cluster management</Link><ResourceStatus value="Not connected" /></div>
          <p>Container workloads below are real deployment records; they do not represent cluster inventory or cluster health.</p>
        </div>
      </ResourcePanel>

      <ResourcePanel title="Alerts and warnings" description="Missing telemetry is not an all-clear signal" action={<Link href={scopedHref("/observability?section=Alerts")}>Alert coverage</Link>}>
        <ResourceState resource={health}>{(data) => <div className={styles.health}>
          <div><span>Active alerts</span><strong>{data.active_alerts ?? "Not reported"}</strong></div>
          <div><span>Resources with warnings</span><strong>{data.warning_resources ?? "Not reported"}</strong></div>
          <div><span>Critical resources</span><strong>{data.critical_resources ?? "Not reported"}</strong></div>
        </div>}</ResourceState>
        <ResourceState resource={alerts}>{(data) => <div className={styles.health}><ResourceStatus value={data.state} /><p>{data.reason}</p></div>}</ResourceState>
      </ResourcePanel>

      <ResourcePanel title="Resource utilization" description="Observed samples, not estimated infrastructure capacity" action={<Link href={scopedHref("/observability?section=Metrics")}>Metrics</Link>}>
        <ResourceState resource={metrics}>{(data) => <>
          {(["cpu", "memory", "storage"] as const).map((kind) => {
            const sample = utilizationSample(data.metrics, `atonix_resource_${kind}_utilization_ratio`);
            return <UtilizationMeter key={kind} label={kind === "cpu" ? "CPU" : kind === "memory" ? "Memory" : "Storage"}
              value={sample.value} detail={sample.count
                ? `Mean of ${sample.count} reported samples; oldest ${new Date(sample.oldest).toLocaleString()}.`
                : data.reason || "No authorized telemetry samples are available."} />;
          })}
        </>}</ResourceState>
        <ResourceState resource={storage}>{(data) => <UtilizationMeter label="Block storage quota"
          value={data.quota_gb > 0 ? data.used_gb / data.quota_gb * 100 : null}
          detail={`${data.used_gb.toLocaleString()} GB provisioned / ${data.quota_gb.toLocaleString()} GB project quota. Not physical disk utilization.`} />}</ResourceState>
      </ResourcePanel>

      <ResourcePanel title="Infrastructure health" description="Health and telemetry reported by Observability" action={<Link href={scopedHref("/observability")}>Monitoring</Link>}>
        <ResourceState resource={health}>{(data) => <div className={styles.health}>
          <div><strong>Overall health</strong><ResourceStatus value={data.health} /></div>
          <div><span>Telemetry</span><ResourceStatus value={data.telemetry_state} /></div>
          {data.resource_groups.filter((group) => group.count > 0).map((group) => <div key={group.kind}><span>{group.kind.replaceAll(".", " ")} ({group.count})</span><ResourceStatus value={group.health} /></div>)}
          <div><span>Active alerts</span><strong>{data.active_alerts ?? "Unavailable"}</strong></div>
          <div><span>Unknown resource health</span><strong>{data.unknown_health}</strong></div>
          <div><span>Failed instances</span><strong>{summaryValue(instances, (items) => items.filter((item) => item.status === "failed").length)}</strong></div>
          <div><span>Volumes reporting errors</span><strong>{summaryValue(volumes, (items) => items.filter((item) => ["ERROR", "FAILED"].includes(item.state.toUpperCase())).length)}</strong></div>
          <p>Running resources and reachable APIs do not imply healthy infrastructure.</p>
          <p>{data.telemetry_freshness ? `Latest telemetry: ${new Date(data.telemetry_freshness).toLocaleString()}` : "No telemetry freshness has been reported."}</p>
        </div>}</ResourceState>
      </ResourcePanel>

      <ResourcePanel title="Instances" description="Current lifecycle state and addresses" action={<Link href={scopedHref("/compute/instances")}>All instances</Link>}>
        <SearchFilterBar search={search} onSearch={setSearch} status={status} onStatus={setStatus}
          statuses={instances.state === "ready" ? [...new Set(instances.data.map((item) => item.status))].sort() : []} />
        <ResourceState resource={instances}>{(items) => {
          const filtered = items.filter((item) => (!status || item.status === status) &&
            `${item.name} ${item.status} ${item.private_ip ?? ""} ${item.public_ip ?? ""}`.toLowerCase().includes(search.trim().toLowerCase()));
          if (!items.length) return <EmptyState title="No instances in this project" description="Create an instance to provision your first virtual machine." action={<Link href={createInstanceHref}>Create Instance</Link>} />;
          if (!filtered.length) return <EmptyState title="No matching instances" description="Try another search or clear the status filter." />;
          return <><ResourceTable caption="Project instances" headings={["Name", "Status", "IP address"]}>
            {filtered.slice(0, 6).map((item) => <tr key={item.id}><td><Link href={scopedHref(`/compute/instances/${encodeURIComponent(item.id)}`)}>{item.name}</Link></td><td><ResourceStatus value={item.status} /></td><td>{item.public_ip || item.private_ip || "Not assigned"}</td></tr>)}
          </ResourceTable><p className={styles.footnote}>Showing {Math.min(filtered.length, 6)} of {filtered.length} matching instances.</p></>;
        }}</ResourceState>
      </ResourcePanel>

      <ResourcePanel title="Storage" description="Managed volumes and their reported state" action={<Link href={scopedHref("/storage")}>Manage storage</Link>}>
        <ResourceState resource={volumes}>{(items) => items.length ? <>
          <ResourceTable caption="Project volumes" headings={["Volume", "Capacity", "Status"]}>
            {items.slice(0, 6).map((item) => <tr key={item.id}><td>{item.name}</td><td>{item.size_gb} GB</td><td><ResourceStatus value={item.state} /></td></tr>)}
          </ResourceTable><p className={styles.footnote}>Showing {Math.min(items.length, 6)} of {items.length} volumes.</p>
        </> : <EmptyState title="No volumes in this project" description="Provision persistent block storage for your workloads." action={<Link href={scopedHref("/storage?action=create")}>Create Volume</Link>} />}</ResourceState>
      </ResourcePanel>

      <ResourcePanel title="Networking" description="Network state is not a connectivity probe" action={<Link href={scopedHref("/networking")}>All networks</Link>}>
        <ResourceState resource={networks}>{(items) => items.length ? <>
          <ResourceTable caption="Project networks" headings={["Network", "Address range", "Status"]}>
            {items.slice(0, 4).map((item) => <tr key={item.id}><td>{item.name}</td><td>{item.cidr || "Not assigned"}</td><td><ResourceStatus value={item.status || "unknown"} /></td></tr>)}
          </ResourceTable><p className={styles.footnote}>Showing {Math.min(items.length, 4)} of {items.length} networks.</p>
        </> : <EmptyState title="No networks in this project" description="Create a virtual network and subnet for your instances." action={<Link href={`${scopedHref("/networking")}#create-network`}>Create Network</Link>} />}</ResourceState>
      </ResourcePanel>

      <ResourcePanel title="Container workloads" description="Deployment state and ready replicas" action={<Link href={scopedHref("/containers")}>All workloads</Link>}>
        <ResourceState resource={deployments}>{(items) => items.length ? <>
          <ResourceTable caption="Project container workloads" headings={["Deployment", "Status", "Replicas"]}>
            {items.slice(0, 4).map((item) => <tr key={item.id}><td><Link href={scopedHref(`/containers/${encodeURIComponent(item.id)}`)}>{item.name}</Link></td><td><ResourceStatus value={item.state} /></td><td>{item.ready_replicas}/{item.desired_replicas}</td></tr>)}
          </ResourceTable><p className={styles.footnote}>Showing {Math.min(items.length, 4)} of {items.length} workloads.</p>
        </> : <EmptyState title="No container workloads" description="Deploy a container to a connected cluster." action={<Link href={scopedHref("/containers/deploy")}>Deploy Container</Link>} />}</ResourceState>
      </ResourcePanel>
    </div>

    <ResourcePanel title="Recent resources" description="Newest instances, volumes and container deployments from the inventories that loaded">
      {recent.length ? <ResourceTable caption="Recently created resources" headings={["Resource", "Type", "Status", "Created"]}>
        {recent.map((item) => <tr key={`${item.kind}:${item.id}`}><td><Link href={scopedHref(item.href)}>{item.name}</Link></td><td>{item.kind}</td><td><ResourceStatus value={item.status} /></td><td>{new Date(item.created_at).toLocaleString()}</td></tr>)}
      </ResourceTable> : <EmptyState title={loading ? "Loading recent resources..." : "No recent resources available"} description="Only successfully loaded inventories contribute to this list. Source failures are reported above." />}
    </ResourcePanel>

    <ResourcePanel title="Recent activity" description="Authorized platform events; not a complete infrastructure audit" action={<Link href={scopedHref("/observability?section=Events")}>All events</Link>}>
      <ResourceState resource={events}>{(items) => items.length ? <ol className={styles.activity}>
        {[...items].sort((a, b) => b.created_at.localeCompare(a.created_at)).slice(0, 8).map((item) => <li key={item.id}>
          <ResourceStatus value={item.severity} /><span>{item.summary}</span><time dateTime={item.created_at}>{new Date(item.created_at).toLocaleString()}</time>
        </li>)}
      </ol> : <EmptyState title="No recent events" description="No events were returned for this project. New operations will appear here when reported by the platform." />}</ResourceState>
    </ResourcePanel>

    <section aria-label="Infrastructure workspaces" className={styles.directory}>
      {cloudNavigation.filter((group) => group.label !== "Overview").map((group) => <div key={group.label}>
        <h2>{group.label}</h2>{group.items.map((item) => <Link href={scopedHref(item.href)} key={item.href}>{item.label}{item.unavailable && <small>Not connected</small>}</Link>)}
      </div>)}
    </section>
  </div>;
}
