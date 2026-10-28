"use client";

import Link from "next/link";
import { useCallback, useEffect, useMemo, useState, type FormEvent } from "react";
import {
  createVolume,
  deleteVolume,
  getOperation,
  getOperations,
  getOverview,
  getProjects,
  getTenants,
  getVolumeTypes,
  getVolumes,
  type StorageOperation,
  type StorageOverview,
  type StorageProject,
  type StorageTenant,
  type StorageVolume,
  type StorageVolumeType,
} from "./lib/api";
import { CloudShell } from "@/app/shared/ui/CloudShell";
import "./storage.css";

type WorkspaceTab = "overview" | "volumes" | "snapshots" | "objects" | "files" | "operations";

const tabs: Array<{ id: WorkspaceTab; label: string }> = [
  { id: "overview", label: "Overview" },
  { id: "volumes", label: "Volumes" },
  { id: "snapshots", label: "Snapshots" },
  { id: "objects", label: "Object storage" },
  { id: "files", label: "File shares" },
  { id: "operations", label: "Operations" },
];

export default function StoragePage() {
  const [tenants, setTenants] = useState<StorageTenant[]>([]);
  const [projects, setProjects] = useState<StorageProject[]>([]);
  const [tenantID, setTenantID] = useState("");
  const [projectID, setProjectID] = useState("");
  const [types, setTypes] = useState<StorageVolumeType[]>([]);
  const [overview, setOverview] = useState<StorageOverview | null>(null);
  const [volumes, setVolumes] = useState<StorageVolume[]>([]);
  const [operations, setOperations] = useState<StorageOperation[]>([]);
  const [tab, setTab] = useState<WorkspaceTab>("overview");
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [activeOperation, setActiveOperation] = useState<StorageOperation | null>(null);
  const [showWizard, setShowWizard] = useState(false);
  const [step, setStep] = useState<1 | 2>(1);
  const [submitting, setSubmitting] = useState(false);
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [sizeGB, setSizeGB] = useState("100");
  const [storageClass, setStorageClass] = useState("");
  const selectedType = types.find((type) => type.id === storageClass);
  const [deleteProtection, setDeleteProtection] = useState(true);
  const [tagsText, setTagsText] = useState("");

  const selectedProject = useMemo(
    () => projects.find((project) => String(project.id) === projectID),
    [projects, projectID],
  );

  const fetchWorkspace = useCallback(async () => {
    if (!tenantID || !projectID) return;
    return Promise.all([
      getOverview(Number(tenantID), Number(projectID)),
      getVolumes(Number(tenantID), Number(projectID)),
      getOperations(Number(tenantID), Number(projectID)),
    ]);
  }, [tenantID, projectID]);

  const refreshWorkspace = useCallback(async () => {
    const result = await fetchWorkspace();
    if (!result) return;
    const [summary, items, recentOperations] = result;
    setOverview(summary);
    setVolumes(items);
    setOperations(recentOperations);
  }, [fetchWorkspace]);

  useEffect(() => {
    let active = true;
    Promise.all([getTenants(), getVolumeTypes()])
      .then(([availableTenants, availableTypes]) => {
        if (!active) return;
        setTenants(availableTenants);
        setTypes(availableTypes);
        if (availableTenants[0]) setTenantID(String(availableTenants[0].id));
        if (availableTypes[0]) setStorageClass(availableTypes[0].id);
      })
      .catch((loadError: unknown) => {
        if (active) setError(loadError instanceof Error ? loadError.message : "Storage could not be loaded.");
        if (active) setLoading(false);
      });
    return () => { active = false; };
  }, []);

  useEffect(() => {
    if (!tenantID) return;
    let active = true;
    getProjects(Number(tenantID))
      .then((items) => {
        if (!active) return;
        setProjects(items);
        setProjectID((current) => items.some((item) => String(item.id) === current)
          ? current
          : String(items[0]?.id ?? ""));
        if (items.length === 0) setLoading(false);
      })
      .catch((loadError: unknown) => {
        if (active) {
          setError(loadError instanceof Error ? loadError.message : "Projects could not be loaded.");
          setLoading(false);
        }
      });
    return () => { active = false; };
  }, [tenantID]);

  useEffect(() => {
    if (!tenantID || !projectID) return;
    let active = true;
    fetchWorkspace()
      .then((result) => {
        if (!active || !result) return;
        const [summary, items, recentOperations] = result;
        setOverview(summary);
        setVolumes(items);
        setOperations(recentOperations);
        setError("");
      })
      .catch((loadError: unknown) => {
        if (active) setError(loadError instanceof Error ? loadError.message : "Storage could not be loaded.");
      })
      .finally(() => { if (active) setLoading(false); });
    return () => { active = false; };
  }, [tenantID, projectID, fetchWorkspace]);

  useEffect(() => {
    if (!activeOperation || ["SUCCEEDED", "FAILED"].includes(activeOperation.status)) return;
    let active = true;
    const timer = window.setTimeout(() => {
      getOperation(activeOperation.id)
        .then((operation) => {
          if (!active) return;
          setActiveOperation(operation);
          if (["SUCCEEDED", "FAILED"].includes(operation.status)) {
            return refreshWorkspace().catch((loadError: unknown) => {
              setError(loadError instanceof Error ? loadError.message : "Storage could not be refreshed.");
            });
          }
        })
        .catch((loadError: unknown) => {
          if (active) setError(loadError instanceof Error ? loadError.message : "Operation status could not be loaded.");
        });
    }, 1200);
    return () => {
      active = false;
      window.clearTimeout(timer);
    };
  }, [activeOperation, refreshWorkspace]);

  function openWizard() {
    setStep(1);
    setName("");
    setDescription("");
    setSizeGB("100");
    setDeleteProtection(true);
    setTagsText("");
    setError("");
    setShowWizard(true);
  }

  function parseTags(): Record<string, string> {
    return Object.fromEntries(
      tagsText.split("\n")
        .map((line) => line.trim())
        .filter(Boolean)
        .map((line) => {
          const separator = line.indexOf("=");
          return separator < 0
            ? [line, ""]
            : [line.slice(0, separator).trim(), line.slice(separator + 1).trim()];
        }),
    );
  }

  async function handleCreate(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!selectedProject || !tenantID) {
      setError("Choose a workspace and project before creating a volume.");
      return;
    }
    setSubmitting(true);
    setError("");
    try {
      const result = await createVolume({
        tenant_id: Number(tenantID),
        project_id: selectedProject.id,
        name,
        description,
        region: selectedType?.region ?? "",
        availability_zone: "auto",
        storage_class: storageClass,
        size_gb: Number(sizeGB),
        encrypted: selectedType?.encrypted ?? false,
        delete_protection: deleteProtection,
        tags: parseTags(),
      });
      setActiveOperation(result.operation);
      setShowWizard(false);
      setTab("volumes");
      await refreshWorkspace();
    } catch (createError) {
      setError(createError instanceof Error ? createError.message : "Volume request failed.");
    } finally {
      setSubmitting(false);
    }
  }

  async function handleDelete(volume: StorageVolume) {
    if (!window.confirm(`Delete volume "${volume.name}"? This cannot be undone.`)) return;
    setError("");
    try {
      const operation = await deleteVolume(volume.id);
      setActiveOperation(operation);
    } catch (deleteError) {
      setError(deleteError instanceof Error ? deleteError.message : "Volume deletion failed.");
    }
  }

  const formatBytes = (gb: number) => gb >= 1024 ? `${(gb / 1024).toFixed(gb % 1024 ? 1 : 0)} TB` : `${gb} GB`;

  return (
    <CloudShell>
    <main className="storage-console">
      <section className="storage-content">
        <header className="storage-heading">
          <div>
            <p className="storage-eyebrow">DATA SERVICES / STORAGE</p>
            <h1>Storage</h1>
            <p>Manage project-scoped volumes and review storage operations.</p>
          </div>
          <div className="storage-heading-actions">
            <label>
              <span className="storage-visually-hidden">Workspace</span>
              <select aria-label="Workspace" value={tenantID} onChange={(event) => {
                setProjects([]);
                setProjectID("");
                setLoading(true);
                setTenantID(event.target.value);
              }}>
                {tenants.map((tenant) => <option value={tenant.id} key={tenant.id}>{tenant.name}</option>)}
              </select>
            </label>
            <label>
              <span className="storage-visually-hidden">Project</span>
              <select aria-label="Project" value={projectID} onChange={(event) => {
                setLoading(true);
                setProjectID(event.target.value);
              }}>
                {projects.map((project) => <option value={project.id} key={project.id}>{project.name}</option>)}
              </select>
            </label>
            <button type="button" className="storage-primary-button" onClick={openWizard} disabled={!selectedProject || types.length === 0}>
              Create resource <span aria-hidden="true">⌄</span>
            </button>
          </div>
        </header>

        <div className="storage-environment"><span aria-hidden="true">i</span>
          {overview?.driver === "local-file"
            ? "Local development storage driver. Volumes are sparse backing files in the API container’s persistent data volume; they are not attachable compute block devices and do not provide at-rest encryption."
            : overview?.driver === "openstack-cinder"
              ? "OpenStack Cinder block storage. Actual attachment to compute instances and snapshot, backup, object, and file-share workflows are not yet connected."
              : "Storage provisioning is unavailable until a storage backend is configured."}
        </div>

        <nav className="storage-tabs" aria-label="Storage resources">
          {tabs.map((item) => (
            <button type="button" key={item.id} className={tab === item.id ? "active" : ""} onClick={() => setTab(item.id)}>
              {item.label}
              {item.id === "volumes" && overview && <span>{overview.volumes}</span>}
            </button>
          ))}
        </nav>

        {error && <div className="storage-alert" role="alert">{error}{error.includes("Authentication") && <> <Link href="/login">Sign in</Link> to continue.</>}</div>}
        {!selectedProject && !loading && !error && <div className="storage-empty"><h2>No project available</h2><p>Create or join a project before managing storage.</p></div>}
        {loading && <div className="storage-empty" role="status">Loading storage workspace…</div>}

        {!loading && selectedProject && (tab === "overview" || tab === "volumes") && (
          <>
            <section className="storage-metrics" aria-label="Storage summary">
              <article><span>Provisioned capacity</span><strong>{overview ? formatBytes(overview.used_gb) : "—"}</strong><small>{overview ? `${formatBytes(overview.quota_gb)} project quota` : "Loading quota"}</small></article>
              <article><span>Volumes</span><strong>{overview?.volumes ?? "—"}</strong><small>{overview ? `${overview.volume_quota} volume limit` : "Project resources"}</small></article>
              <article><span>Storage driver</span><strong>{overview?.driver ?? "—"}</strong><small>Development backend</small></article>
            </section>
            <section className="storage-table-panel">
              <div className="storage-panel-heading">
                <div><p className="storage-eyebrow">{tab === "overview" ? "PROJECT RESOURCES" : "BLOCK STORAGE"}</p><h2>Volumes</h2><p>Volume records and provisioning state for {selectedProject.name}.</p></div>
                <button type="button" className="storage-secondary-button" onClick={openWizard}>Create volume</button>
              </div>
              {volumes.length === 0
                ? <div className="storage-empty"><span aria-hidden="true">◈</span><h3>No volumes yet</h3><p>Create a development volume to try the asynchronous storage workflow.</p></div>
                : <div className="storage-table-scroll"><table className="storage-table">
                    <thead><tr><th>Name</th><th>Status</th><th>Capacity</th><th>Type</th><th>Protection</th><th>Created</th><th><span className="storage-visually-hidden">Actions</span></th></tr></thead>
                    <tbody>{volumes.map((volume) => (
                      <tr key={volume.id}>
                        <td><strong>{volume.name}</strong><small>{volume.description || volume.id.slice(0, 12)}</small></td>
                        <td><span className={`storage-status ${volume.state.toLowerCase()}`}>{volume.state}</span></td>
                        <td>{formatBytes(volume.size_gb)}</td>
                        <td>{volume.storage_class}</td>
                        <td>{volume.delete_protection ? "Delete protected" : "Unprotected"}</td>
                        <td>{new Date(volume.created_at).toLocaleDateString()}</td>
                        <td>{!volume.delete_protection && volume.state === "AVAILABLE" && <button className="storage-delete-button" type="button" onClick={() => void handleDelete(volume)}>Delete</button>}</td>
                      </tr>
                    ))}</tbody>
                  </table></div>}
            </section>
          </>
        )}

        {!loading && selectedProject && tab !== "overview" && tab !== "volumes" && (
          tab === "operations" ? <section className="storage-table-panel">
            <div className="storage-panel-heading"><div><p className="storage-eyebrow">PROJECT ACTIVITY</p><h2>Operations</h2><p>Asynchronous provisioning and deletion jobs.</p></div></div>
            {operations.length === 0 ? <div className="storage-empty"><h3>No operations yet</h3><p>Volume provisioning and lifecycle operations will be listed here.</p></div>
              : <div className="storage-table-scroll"><table className="storage-table"><thead><tr><th>Operation</th><th>Resource</th><th>Status</th><th>Attempts</th><th>Started</th></tr></thead><tbody>
                {operations.map((operation) => <tr key={operation.id}><td><strong>{operation.action}</strong><small>{operation.id.slice(0, 12)}</small></td><td>{operation.resource_type} · {operation.resource_id.slice(0, 12)}</td><td><span className={`storage-status ${operation.status.toLowerCase()}`}>{operation.status}</span></td><td>{operation.attempts}</td><td>{new Date(operation.created_at).toLocaleString()}</td></tr>)}
              </tbody></table></div>}
          </section> : <div className="storage-empty storage-pending">
            <span aria-hidden="true">◈</span>
            <h2>{tabs.find((item) => item.id === tab)?.label} API is not connected yet</h2>
            <p>This workspace only shows resources backed by the storage API. This resource type is not represented by demo data.</p>
          </div>
        )}

        {activeOperation && <div className={`storage-operation ${activeOperation.status.toLowerCase()}`} role="status">
          <span className="storage-operation-dot" />
          <div><strong>{activeOperation.action === "create" ? "Volume provisioning" : "Volume deletion"} {activeOperation.status.toLowerCase()}</strong><small>Operation {activeOperation.id.slice(0, 12)} · attempt {activeOperation.attempts}</small></div>
          {["SUCCEEDED", "FAILED"].includes(activeOperation.status) && <button type="button" onClick={() => setActiveOperation(null)} aria-label="Dismiss operation status">×</button>}
        </div>}
      </section>

      {showWizard && <div className="storage-modal-backdrop" role="presentation" onMouseDown={(event) => { if (event.target === event.currentTarget) setShowWizard(false); }}>
        <section className="storage-modal" role="dialog" aria-modal="true" aria-labelledby="storage-wizard-title">
          <header><div><p className="storage-eyebrow">BLOCK STORAGE / STEP {step} OF 2</p><h2 id="storage-wizard-title">{step === 1 ? "Create volume" : "Review and create"}</h2></div><button type="button" onClick={() => setShowWizard(false)} aria-label="Close">×</button></header>
          <form onSubmit={handleCreate}>
            {step === 1 ? <>
              <label className="storage-field"><span>Resource name</span><input value={name} onChange={(event) => setName(event.target.value)} minLength={2} maxLength={63} pattern="[a-z0-9]([a-z0-9-]*[a-z0-9])?" placeholder="production-db-volume" required /></label>
              <label className="storage-field"><span>Description <small>Optional</small></span><textarea value={description} onChange={(event) => setDescription(event.target.value)} maxLength={512} rows={2} /></label>
              <div className="storage-field-row">
                <label className="storage-field"><span>Project</span><input value={selectedProject?.name ?? ""} readOnly /></label>
                <label className="storage-field"><span>Region</span><input value={selectedType?.region ?? ""} readOnly /></label>
              </div>
              <div className="storage-field-row">
                <label className="storage-field"><span>Storage type</span><select value={storageClass} onChange={(event) => setStorageClass(event.target.value)} required>{types.map((type) => <option value={type.id} key={type.id}>{type.name}</option>)}</select></label>
                <label className="storage-field"><span>Capacity (GB)</span><input type="number" value={sizeGB} onChange={(event) => setSizeGB(event.target.value)} min={1} max={1024} required /></label>
              </div>
              <label className="storage-field"><span>Tags <small>Optional, one key=value per line</small></span><textarea value={tagsText} onChange={(event) => setTagsText(event.target.value)} rows={3} placeholder={"environment=development\nteam=platform"} /></label>
            </> : <>
              <dl className="storage-review">
                <div><dt>Name</dt><dd>{name}</dd></div><div><dt>Type</dt><dd>{types.find((type) => type.id === storageClass)?.name ?? storageClass}</dd></div>
                <div><dt>Capacity</dt><dd>{sizeGB} GB</dd></div><div><dt>Region</dt><dd>{selectedType?.region}</dd></div>
                <div><dt>Project</dt><dd>{selectedProject?.name}</dd></div><div><dt>Encryption</dt><dd>{selectedType?.encrypted ? "Enabled by selected volume type" : "Not enabled"}</dd></div>
              </dl>
              <label className="storage-checkbox"><input type="checkbox" checked={deleteProtection} onChange={(event) => setDeleteProtection(event.target.checked)} /><span><strong>Enable delete protection</strong><small>Require this control to be disabled before volume deletion.</small></span></label>
              <p className="storage-review-note">The API validates authorization, capacity and project quotas again when you submit. Provisioning is asynchronous and the volume remains in CREATING until the worker completes.</p>
            </>}
            <footer>
              {step === 1
                ? <><button type="button" className="storage-secondary-button" onClick={() => setShowWizard(false)}>Cancel</button><button type="button" className="storage-primary-button" onClick={() => setStep(2)}>Review volume <span aria-hidden="true">→</span></button></>
                : <><button type="button" className="storage-secondary-button" onClick={() => setStep(1)}>Back</button><button type="submit" className="storage-primary-button" disabled={submitting}>{submitting ? "Submitting…" : "Create volume"}</button></>}
            </footer>
          </form>
        </section>
      </div>}
    </main>
    </CloudShell>
  );
}
