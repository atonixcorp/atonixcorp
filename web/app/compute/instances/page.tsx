"use client";

import Link from "next/link";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import {
  createInstance,
  getCatalog,
  getInstances,
  getComputeCapabilities,
  getProjects,
  getTenants,
} from "../lib/api";
import type { ComputeProject, ComputeTenant } from "../lib/api";
import type { ComputeImage, ComputeInstance, ComputeNetwork, ComputePlan } from "../lib/types";

export default function ComputeInstancesPage() {
  const [instances, setInstances] = useState<ComputeInstance[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [filter, setFilter] = useState("");
  const [plans, setPlans] = useState<ComputePlan[]>([]);
  const [images, setImages] = useState<ComputeImage[]>([]);
  const [networks, setNetworks] = useState<ComputeNetwork[]>([]);
  const [catalogProvisioning, setCatalogProvisioning] = useState(false);
  const [planCatalogError, setPlanCatalogError] = useState("");
  const [planCatalogLoading, setPlanCatalogLoading] = useState(true);
  const [tenants, setTenants] = useState<ComputeTenant[]>([]);
  const [projects, setProjects] = useState<ComputeProject[]>([]);
  const [tenantID, setTenantID] = useState("");
  const [projectID, setProjectID] = useState("");
  const [scopeError, setScopeError] = useState("");
  const [organizationProvisioning, setOrganizationProvisioning] = useState(false);
  const [computeRegion, setComputeRegion] = useState("");
  const [capabilityError, setCapabilityError] = useState("");
  const [createOpen, setCreateOpen] = useState(false);
  const [creating, setCreating] = useState(false);
  const [createError, setCreateError] = useState("");
  const createIdempotencyKey = useRef("");
  const [createDraft, setCreateDraft] = useState({
    name: "",
    imageID: "",
    sizeID: "",
    networkID: "",
    keyName: "",
  });

  const loadInstances = useCallback(async () => {
    if (!tenantID || !projectID) {
      setInstances([]);
      setLoading(false);
      return;
    }
    setLoading(true);
    setError("");
    try {
      setInstances(await getInstances(Number(tenantID), Number(projectID)));
    } catch (loadError) {
      setError(loadError instanceof Error ? loadError.message : "Could not load instances.");
    } finally {
      setLoading(false);
    }
  }, [projectID, tenantID]);

  useEffect(() => {
    let active = true;
    void getTenants().then((items) => {
      if (!active) return;
      setTenants(items);
      if (items.length) setTenantID(String(items[0].id));
      else {
        setLoading(false);
        setPlanCatalogLoading(false);
      }
    }).catch((loadError: unknown) => {
      if (active) {
        setScopeError(loadError instanceof Error ? loadError.message : "Could not load workspaces.");
        setLoading(false);
      }
    });
    return () => { active = false; };
  }, []);

  useEffect(() => {
    if (!tenantID) return;
    let active = true;
    void getProjects(Number(tenantID)).then((items) => {
      if (!active) return;
      setProjects(items);
      setProjectID(items.length ? String(items[0].id) : "");
      if (!items.length) {
        setLoading(false);
        setPlanCatalogLoading(false);
      }
    }).catch((loadError: unknown) => {
      if (active) {
        setScopeError(loadError instanceof Error ? loadError.message : "Could not load projects.");
        setLoading(false);
      }
    });
    return () => { active = false; };
  }, [tenantID]);

  useEffect(() => {
    const timeout = window.setTimeout(() => { void loadInstances(); }, 0);
    return () => window.clearTimeout(timeout);
  }, [loadInstances]);

  useEffect(() => {
    if (!tenantID) return;
    let active = true;
    void getComputeCapabilities(Number(tenantID)).then((capabilities) => {
      if (!active) return;
      setOrganizationProvisioning(capabilities.organization_provisioning);
    }).catch((loadError: unknown) => {
      if (active) {
        setCapabilityError(loadError instanceof Error
          ? loadError.message
          : "Compute availability could not be checked.");
      }
    });
    return () => { active = false; };
  }, [tenantID]);

  useEffect(() => {
    if (!tenantID || !projectID) return;
    let active = true;
    void getCatalog(Number(tenantID), Number(projectID)).then((catalog) => {
      if (active) {
        setPlans(catalog.plans);
        setImages(catalog.images);
        setNetworks(catalog.networks);
        setComputeRegion(catalog.regions[0]?.id ?? "");
        setCatalogProvisioning(catalog.provisioning_enabled);
        setCreateDraft((current) => ({
          ...current,
          imageID: catalog.images.find((item) => item.available)?.id ?? "",
          sizeID: catalog.plans.find((item) => item.available)?.id ?? "",
          networkID: catalog.networks.find((item) => item.available)?.id ?? "",
        }));
      }
    }).catch((catalogError: unknown) => {
      if (active) {
        setCatalogProvisioning(false);
        setPlanCatalogError(catalogError instanceof Error
          ? catalogError.message
          : "Plan details are unavailable.");
      }
    }).finally(() => {
      if (active) setPlanCatalogLoading(false);
    });
    return () => { active = false; };
  }, [projectID, tenantID]);

  const filteredInstances = useMemo(() => {
    const query = filter.trim().toLowerCase();
    return query
      ? instances.filter((instance) =>
        `${instance.name} ${instance.label} ${instance.region} ${instance.status}`
          .toLowerCase().includes(query))
      : instances;
  }, [filter, instances]);

  const submitCreate = async () => {
    if (!tenantID || !projectID || !catalogProvisioning) {
      setCreateError("Choose a workspace and project before creating an instance.");
      return;
    }
    setCreating(true);
    setCreateError("");
    if (!createIdempotencyKey.current) createIdempotencyKey.current = crypto.randomUUID();
    try {
      await createInstance({
        tenant_id: Number(tenantID),
        project_id: Number(projectID),
        name: createDraft.name,
        region_id: computeRegion,
        image_id: createDraft.imageID,
        size_id: createDraft.sizeID,
        network_id: createDraft.networkID,
        key_name: createDraft.keyName || undefined,
      }, createIdempotencyKey.current);
      createIdempotencyKey.current = "";
      setCreateDraft({ name: "", imageID: "", sizeID: "", networkID: "", keyName: "" });
      setCreateOpen(false);
      await loadInstances();
    } catch (requestError) {
      setCreateError(requestError instanceof Error
        ? requestError.message
        : "The virtual machine could not be created.");
    } finally {
      setCreating(false);
    }
  };

  return (
    <div className="compute-page">
      <header className="compute-page-heading">
        <div>
          <p className="compute-eyebrow">COMPUTE / INSTANCES</p>
          <h1>Compute instances</h1>
          <p>Manage virtual machines and review their current lifecycle status.</p>
        </div>
        <button
          className="compute-primary-button"
          type="button"
          onClick={() => { setCreateError(""); setCreateOpen((open) => !open); }}
          disabled={!organizationProvisioning || !catalogProvisioning}
          title={organizationProvisioning && catalogProvisioning ? "Create a VM in this organization's project" : "A project and available compute resources are required"}
        >
          ＋ Create instance
        </button>
      </header>

      {capabilityError ? (
        <p className="compute-inline-error" role="alert">
          Instance creation is unavailable because compute services could not be checked: {capabilityError}
        </p>
      ) : !organizationProvisioning && (
        <p className="compute-deploy-notice" role="status">
          Instance creation is unavailable because this organization does not have a validated project.
        </p>
      )}

      {createOpen && (
        <section className="compute-card" aria-label="Create virtual machine">
          <header className="compute-table-toolbar">
            <div>
              <h2>Create a virtual machine</h2>
              <p>Region {computeRegion ? "Primary region" : "not configured"} · organization project</p>
            </div>
            <button type="button" className="compute-secondary-button" onClick={() => setCreateOpen(false)}>
              Cancel
            </button>
          </header>
          <p className="compute-deploy-notice" role="note">
            Resources created here are isolated to this organization’s project.
          </p>
          {createError && <p className="compute-inline-error" role="alert">{createError}</p>}
          <form
            className="compute-deploy-sections"
            onSubmit={(event) => { event.preventDefault(); void submitCreate(); }}
          >
            <label className="compute-filter">
              Instance name
              <input
                required
                maxLength={63}
                pattern="[A-Za-z0-9][A-Za-z0-9._-]{0,62}"
                value={createDraft.name}
                onChange={(event) => setCreateDraft((current) => ({ ...current, name: event.target.value }))}
              />
            </label>
            <label className="compute-filter">
              Operating system image
              <select required value={createDraft.imageID} onChange={(event) => setCreateDraft((current) => ({ ...current, imageID: event.target.value }))}>
                <option value="">Select an active image</option>
                {images.filter((image) => image.available).map((image) => <option value={image.id} key={image.id}>{image.name}</option>)}
              </select>
            </label>
            <label className="compute-filter">
              Instance size
              <select required value={createDraft.sizeID} onChange={(event) => setCreateDraft((current) => ({ ...current, sizeID: event.target.value }))}>
                <option value="">Select an instance size</option>
                {plans.filter((plan) => plan.available).map((plan) => <option value={plan.id} key={plan.id}>{plan.name} · {plan.vcpus} vCPU / {Math.round(plan.memory_mb / 1024)} GB RAM</option>)}
              </select>
            </label>
            <label className="compute-filter">
              Virtual network
              <select required value={createDraft.networkID} onChange={(event) => setCreateDraft((current) => ({ ...current, networkID: event.target.value }))}>
                <option value="">Select a network</option>
                {networks.filter((network) => network.available).map((network) => <option value={network.id} key={network.id}>{network.name}</option>)}
              </select>
            </label>
            <label className="compute-filter">
              SSH keypair name (optional)
              <input
                value={createDraft.keyName}
                onChange={(event) => setCreateDraft((current) => ({ ...current, keyName: event.target.value }))}
              />
            </label>
            {planCatalogError && <p className="compute-inline-error" role="alert">{planCatalogError}</p>}
            <button
              className="compute-primary-button"
              type="submit"
              disabled={creating || !tenantID || !projectID || !organizationProvisioning || !catalogProvisioning}
            >
              {creating ? "Submitting…" : "Create VM"}
            </button>
          </form>
        </section>
      )}

      <section className="compute-card compute-instance-card" aria-label="Compute instances">
        <header className="compute-table-toolbar">
          <div><h2>Instances</h2><p>{loading ? "Loading instances…" : `${instances.length} total`}</p></div>
          <div className="compute-table-actions">
            <label className="compute-filter">
              <span className="sr-only">Workspace</span>
              <select value={tenantID} onChange={(event) => { setScopeError(""); setProjects([]); setProjectID(""); setOrganizationProvisioning(false); setComputeRegion(""); setCapabilityError(""); setPlans([]); setImages([]); setNetworks([]); setCatalogProvisioning(false); setPlanCatalogError(""); setPlanCatalogLoading(true); setTenantID(event.target.value); }} aria-label="Workspace">
                <option value="">Choose workspace</option>
                {tenants.map((tenant) => <option key={tenant.id} value={tenant.id}>{tenant.name}</option>)}
              </select>
            </label>
            <label className="compute-filter">
              <span className="sr-only">Project</span>
              <select value={projectID} onChange={(event) => { setScopeError(""); setPlans([]); setImages([]); setNetworks([]); setComputeRegion(""); setCatalogProvisioning(false); setPlanCatalogError(""); setPlanCatalogLoading(true); setProjectID(event.target.value); }} aria-label="Project" disabled={!projects.length}>
                <option value="">Choose project</option>
                {projects.map((project) => <option key={project.id} value={project.id}>{project.name}</option>)}
              </select>
            </label>
            <label className="compute-filter">
              <span className="sr-only">Filter instances</span>
              <input
                type="search"
                placeholder="Filter by name, region, or status"
                value={filter}
                onChange={(event) => setFilter(event.target.value)}
              />
            </label>
            <button type="button" className="compute-secondary-button" onClick={() => void loadInstances()} disabled={loading}>
              Refresh
            </button>
          </div>
        </header>
        {scopeError && <p className="compute-inline-error" role="alert">{scopeError}</p>}
        {planCatalogError && <p className="compute-inline-error" role="status">{planCatalogError} Plan sizes may be unavailable.</p>}
        {loading ? (
          <div className="compute-table-skeleton" aria-label="Loading instances">
            {[0, 1, 2, 3].map((row) => <div key={row}><i /><i /><i /><i /></div>)}
          </div>
        ) : error ? (
          <div className="compute-state-panel" role="alert">
            <h3>Instances are unavailable</h3>
            <p>{error}</p>
            <button className="compute-secondary-button" type="button" onClick={() => void loadInstances()}>Retry</button>
          </div>
        ) : instances.length === 0 ? (
          <div className="compute-state-panel">
            <span className="compute-empty-icon" aria-hidden="true">▣</span>
            <h3>No instances yet</h3>
            <p>Choose a workspace and project, then create an instance in that project.</p>
            <button
              className="compute-primary-button"
              type="button"
              onClick={() => setCreateOpen(true)}
              disabled={!organizationProvisioning}
            >
              ＋ Create your first instance
            </button>
          </div>
        ) : filteredInstances.length === 0 ? (
          <div className="compute-state-panel">
            <h3>No matching instances</h3>
            <p>Try a different search, or clear the filter to see all instances.</p>
            <button className="compute-secondary-button" type="button" onClick={() => setFilter("")}>Clear filter</button>
          </div>
        ) : (
          <div className="compute-table-scroll">
            <table className="compute-table">
              <thead>
                <tr><th>Name</th><th>Status</th><th>Region</th><th>vCPU</th><th>Memory</th><th>IP address</th><th><span className="sr-only">Actions</span></th></tr>
              </thead>
              <tbody>
                {filteredInstances.map((instance) => (
                  <InstanceRow
                    key={instance.id}
                    instance={instance}
                    plan={plans.find((item) => item.id === instance.plan_id)}
                    planCatalogLoading={planCatalogLoading}
                  />
                ))}
              </tbody>
            </table>
          </div>
        )}
      </section>
    </div>
  );
}

function InstanceRow({
  instance,
  plan,
  planCatalogLoading,
}: {
  instance: ComputeInstance;
  plan?: ComputePlan;
  planCatalogLoading: boolean;
}) {
  return (
    <tr>
      <td><Link className="compute-instance-name" href={`/compute/instances/${encodeURIComponent(instance.id)}`}>{instance.label || instance.name}<small>{instance.hostname}</small></Link></td>
      <td><span className={`compute-instance-status ${statusTone(instance.status)}`}><i />{instance.status}</span></td>
      <td>{instance.region ? "Primary region" : "Not assigned"}</td>
      <td>{plan ? plan.vcpus : planCatalogLoading ? "Loading…" : "—"}</td>
      <td>{plan ? `${Math.round(plan.memory_mb / 1024)} GB` : planCatalogLoading ? "Loading…" : "—"}</td>
      <td>{instance.public_ip || "—"}</td>
      <td><Link className="compute-row-action" href={`/compute/instances/${encodeURIComponent(instance.id)}`}>Manage</Link></td>
    </tr>
  );
}

function statusTone(status: string): string {
  if (status === "running") return "running";
  if (status === "provisioning") return "provisioning";
  if (status === "failed") return "failed";
  return "stopped";
}
