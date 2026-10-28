"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useEffect, useRef, useState } from "react";
import { createDeployment, getProjects, getTenants } from "../lib/api";
import type {
  ContainerDeploymentSpec,
  ContainerEnvironmentVariable,
  ContainerPort,
  ContainerVolume,
  Project,
  Tenant,
} from "../lib/api";

const steps = ["Workspace", "Image", "Resources", "Networking", "Environment", "Storage", "Health", "Review"];
const initialPorts: ContainerPort[] = [{ container_port: 8080, service_port: 80, protocol: "TCP" }];
const draftStorageKey = "atonixcorp-container-deployment-draft";

type DeploymentDraft = {
  tenantID: number;
  projectID: number;
  name: string;
  description: string;
  environment: ContainerDeploymentSpec["environment"];
  image: string;
  replicas: number;
  cpuRequest: string;
  cpuLimit: string;
  memoryRequest: string;
  memoryLimit: string;
  ports: ContainerPort[];
  environmentVariables: ContainerEnvironmentVariable[];
  volumes: ContainerVolume[];
  readinessPath: string;
  livenessPath: string;
  retainVolumes: boolean;
  step: number;
};

function messageOf(error: unknown): string {
  return error instanceof Error ? error.message : "The container service could not complete the request.";
}

export default function ContainerDeployPage() {
  const router = useRouter();
  const [step, setStep] = useState(0);
  const [tenants, setTenants] = useState<Tenant[]>([]);
  const [projects, setProjects] = useState<Project[]>([]);
  const [tenantID, setTenantID] = useState(0);
  const [projectID, setProjectID] = useState(0);
  const [scopeLoading, setScopeLoading] = useState(true);
  const [draftLoaded, setDraftLoaded] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState("");
  const retryKey = useRef<{ fingerprint: string; key: string } | null>(null);
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [environment, setEnvironment] = useState<ContainerDeploymentSpec["environment"]>("development");
  const [image, setImage] = useState("");
  const [replicas, setReplicas] = useState(1);
  const [cpuRequest, setCPURequest] = useState("250m");
  const [cpuLimit, setCPULimit] = useState("1000m");
  const [memoryRequest, setMemoryRequest] = useState("256Mi");
  const [memoryLimit, setMemoryLimit] = useState("1Gi");
  const [ports, setPorts] = useState<ContainerPort[]>(initialPorts);
  const [environmentVariables, setEnvironmentVariables] = useState<ContainerEnvironmentVariable[]>([]);
  const [volumes, setVolumes] = useState<ContainerVolume[]>([]);
  const [readinessPath, setReadinessPath] = useState("");
  const [livenessPath, setLivenessPath] = useState("");
  const [retainVolumes, setRetainVolumes] = useState(true);

  useEffect(() => {
    let active = true;
    void getTenants().then((items) => {
      if (!active) return;
      setTenants(items);
      try {
        const stored = sessionStorage.getItem(draftStorageKey);
        if (stored) {
          const draft = JSON.parse(stored) as Partial<DeploymentDraft>;
          if (draft && typeof draft === "object") {
            const tenantID = items.some((item) => item.id === draft.tenantID) ? draft.tenantID! : items[0]?.id ?? 0;
            setTenantID(tenantID);
            setProjectID(typeof draft.projectID === "number" ? draft.projectID : 0);
            if (typeof draft.name === "string") setName(draft.name);
            if (typeof draft.description === "string") setDescription(draft.description);
            if (draft.environment === "development" || draft.environment === "staging" || draft.environment === "production") setEnvironment(draft.environment);
            if (typeof draft.image === "string") setImage(draft.image);
            if (typeof draft.replicas === "number") setReplicas(draft.replicas);
            if (typeof draft.cpuRequest === "string") setCPURequest(draft.cpuRequest);
            if (typeof draft.cpuLimit === "string") setCPULimit(draft.cpuLimit);
            if (typeof draft.memoryRequest === "string") setMemoryRequest(draft.memoryRequest);
            if (typeof draft.memoryLimit === "string") setMemoryLimit(draft.memoryLimit);
            if (Array.isArray(draft.ports)) setPorts(draft.ports);
            if (Array.isArray(draft.environmentVariables)) {
              setEnvironmentVariables(draft.environmentVariables.map((item) => ({
                ...item,
                value: item.secret ? "" : item.value,
              })));
            }
            if (Array.isArray(draft.volumes)) setVolumes(draft.volumes);
            if (typeof draft.readinessPath === "string") setReadinessPath(draft.readinessPath);
            if (typeof draft.livenessPath === "string") setLivenessPath(draft.livenessPath);
            if (typeof draft.retainVolumes === "boolean") setRetainVolumes(draft.retainVolumes);
            if (typeof draft.step === "number") setStep(Math.max(0, Math.min(draft.step, steps.length - 1)));
          }
        } else {
          setTenantID(items[0]?.id ?? 0);
        }
      } catch {
        setError("The saved deployment draft could not be restored.");
        setTenantID(items[0]?.id ?? 0);
      }
      setDraftLoaded(true);
      setScopeLoading(false);
    }).catch((loadError: unknown) => {
      if (active) {
        setError(messageOf(loadError));
        setScopeLoading(false);
        setDraftLoaded(true);
      }
    });
    return () => { active = false; };
  }, []);

  useEffect(() => {
    if (!draftLoaded || submitting) return;
    const draft: DeploymentDraft = {
      tenantID, projectID, name, description, environment, image, replicas,
      cpuRequest, cpuLimit, memoryRequest, memoryLimit, ports,
      environmentVariables: environmentVariables.map((item) => ({
        ...item,
        value: item.secret ? "" : item.value,
      })),
      volumes, readinessPath, livenessPath, retainVolumes, step,
    };
    try {
      sessionStorage.setItem(draftStorageKey, JSON.stringify(draft));
    } catch {
      window.setTimeout(() => {
        setError("The browser could not save this draft. Keep this page open to avoid losing your work.");
      }, 0);
    }
  }, [
    draftLoaded, submitting, tenantID, projectID, name, description, environment, image,
    replicas, cpuRequest, cpuLimit, memoryRequest, memoryLimit, ports, environmentVariables,
    volumes, readinessPath, livenessPath, retainVolumes, step,
  ]);

  useEffect(() => {
    const hasWork = Boolean(name || image || description || environmentVariables.length || volumes.length);
    if (!hasWork) return;
    const warnBeforeLeaving = (event: BeforeUnloadEvent) => {
      event.preventDefault();
      event.returnValue = "";
    };
    window.addEventListener("beforeunload", warnBeforeLeaving);
    return () => window.removeEventListener("beforeunload", warnBeforeLeaving);
  }, [name, image, description, environmentVariables.length, volumes.length]);

  useEffect(() => {
    if (!tenantID) {
      return;
    }
    let active = true;
    void getProjects(tenantID).then((items) => {
      if (active) {
        setProjects(items);
        setProjectID((current) => items.some((item) => item.id === current)
          ? current
          : items[0]?.id ?? 0);
      }
    }).catch((loadError: unknown) => {
      if (active) setError(messageOf(loadError));
    });
    return () => { active = false; };
  }, [tenantID]);

  function updatePort(index: number, patch: Partial<ContainerPort>) {
    setPorts((current) => current.map((port, itemIndex) => itemIndex === index ? { ...port, ...patch } : port));
  }

  function updateEnvironmentVariable(index: number, patch: Partial<ContainerEnvironmentVariable>) {
    setEnvironmentVariables((current) => current.map((item, itemIndex) => itemIndex === index ? { ...item, ...patch } : item));
  }

  function updateVolume(index: number, patch: Partial<ContainerVolume>) {
    setVolumes((current) => current.map((volume, itemIndex) => itemIndex === index ? { ...volume, ...patch } : volume));
  }

  function validateCurrentStep(): boolean {
    setError("");
    if (step === 0 && (!tenantID || !projectID)) {
      setError("Choose a workspace and project before continuing.");
      return false;
    }
    if (step === 1) {
      if (!/^[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?$/.test(name.trim())) {
        setError("Deployment name must be a lowercase DNS label with up to 63 characters.");
        return false;
      }
      if (!image.trim()) {
        setError("Enter the image reference from your container registry.");
        return false;
      }
      if (environment === "production" && !/@sha256:[a-f0-9]{64}$/.test(image.trim())) {
        setError("Production images must be pinned by an immutable sha256 digest.");
        return false;
      }
    }
    if (step === 3 && ports.some((port) => port.container_port < 1 || port.container_port > 65535 ||
      (port.service_port !== undefined && (port.service_port < 1 || port.service_port > 65535)))) {
      setError("Ports must be between 1 and 65535.");
      return false;
    }
    if (step === 4 && environmentVariables.some((item) => !/^[A-Za-z_][A-Za-z0-9_]*$/.test(item.name) || !item.value)) {
      setError("Environment variable names must be valid, and each variable needs a value.");
      return false;
    }
    if (step === 5 && volumes.some((volume) => !volume.name || !volume.mount_path || volume.size_gib < 1)) {
      setError("Every volume needs a name, mount path, and size of at least 1 GiB.");
      return false;
    }
    return true;
  }

  function nextStep() {
    if (validateCurrentStep()) setStep((current) => Math.min(current + 1, steps.length - 1));
  }

  async function submitDeployment() {
    if (!validateCurrentStep()) return;
    if (!tenantID || !projectID) {
      setError("Choose a workspace and project before creating the deployment.");
      setStep(0);
      return;
    }
    setSubmitting(true);
    setError("");
    const spec: ContainerDeploymentSpec = {
      name: name.trim(),
      description: description.trim(),
      environment,
      image: image.trim(),
      replicas,
      cpu_request: cpuRequest,
      cpu_limit: cpuLimit,
      memory_request: memoryRequest,
      memory_limit: memoryLimit,
      ports: ports.filter((port) => port.container_port > 0),
      environment_variables: environmentVariables,
      volumes,
      retain_volumes: retainVolumes,
      ...(readinessPath ? { readiness_probe: { path: readinessPath, port: ports[0]?.container_port ?? 8080 } } : {}),
      ...(livenessPath ? { liveness_probe: { path: livenessPath, port: ports[0]?.container_port ?? 8080 } } : {}),
    };
    const fingerprint = JSON.stringify({ tenantID, projectID, spec });
    const key = retryKey.current?.fingerprint === fingerprint
      ? retryKey.current.key
      : crypto.randomUUID();
    retryKey.current = { fingerprint, key };
    try {
      const deployment = await createDeployment(tenantID, projectID, spec, key);
      try {
        sessionStorage.removeItem(draftStorageKey);
      } catch {
        setError("Deployment was created, but the saved browser draft could not be cleared.");
      }
      router.push(`/containers/${encodeURIComponent(deployment.id)}`);
    } catch (submitError) {
      setError(messageOf(submitError));
    } finally {
      setSubmitting(false);
    }
  }

  const currentTenant = tenants.find((tenant) => tenant.id === tenantID);
  const currentProject = projects.find((project) => project.id === projectID);
  const visiblePorts = ports.filter((port) => port.container_port > 0);

  return (
    <>
      <header className="container-heading">
        <div>
          <p className="container-eyebrow">CONTAINERS / NEW DEPLOYMENT</p>
          <h1>Deploy a container</h1>
          <p>Configure and launch a workload in your selected Kubernetes project.</p>
        </div>
        <Link className="container-button secondary" href="/containers" onClick={(event) => {
          if ((name || image || description || environmentVariables.length || volumes.length) &&
              !window.confirm("Leave this deployment wizard? Your non-secret draft will be saved, but secret values must be entered again.")) {
            event.preventDefault();
          }
        }}>Cancel</Link>
      </header>

      <div className="container-deploy-shell">
        <nav className="container-card container-steps" aria-label="Deployment steps">
          {steps.map((label, index) => (
            <button key={label} type="button" className={`container-step${step === index ? " active" : ""}`} onClick={() => {
              if (index <= step || (index === step + 1 && validateCurrentStep())) setStep(index);
            }}>
              <span>{index + 1}</span>{label}
            </button>
          ))}
        </nav>

        <section className="container-card container-wizard-card" aria-live="polite">
          <h2>{steps[step]}</h2>
          <p>{[
            "Choose who owns and can manage this deployment.",
            "Select the image and runtime environment.",
            "Set the workload replicas and compute limits.",
            "Expose container ports through an internal cluster service.",
            "Configure non-secret values and encrypted secrets.",
            "Attach persistent storage to the workload.",
            "Set HTTP health checks to monitor readiness.",
            "Review the deployment configuration before submitting.",
          ][step]}</p>

          {scopeLoading || !draftLoaded ? <div className="container-loading">Loading workspaces and restoring your draft…</div> : (
            <div className="container-wizard-fields">
              {step === 0 && (
                <div className="container-field-row">
                  <label className="container-field">
                    <span>Workspace</span>
                    <select value={tenantID || ""} onChange={(event) => {
                      setProjects([]);
                      setProjectID(0);
                      setTenantID(Number(event.target.value));
                    }}>
                      {tenants.map((tenant) => <option key={tenant.id} value={tenant.id}>{tenant.name}</option>)}
                    </select>
                  </label>
                  <label className="container-field">
                    <span>Project</span>
                    <select value={projectID || ""} onChange={(event) => setProjectID(Number(event.target.value))}>
                      {projects.map((project) => <option key={project.id} value={project.id}>{project.name}</option>)}
                    </select>
                  </label>
                  {!tenants.length && <div className="container-info">No workspace is available for this account. Ask an administrator to add you to a project.</div>}
                </div>
              )}
              {step === 1 && (
                <>
                  <label className="container-field"><span>Deployment name</span><input value={name} onChange={(event) => setName(event.target.value.toLowerCase())} placeholder="payments-api" maxLength={63} required /><small>Lowercase letters, numbers, and hyphens.</small></label>
                  <label className="container-field"><span>Container image</span><input value={image} onChange={(event) => setImage(event.target.value)} placeholder="registry.example.com/team/app:1.0.0" required /><small>Use an immutable sha256 digest for production.</small></label>
                  <div className="container-field-row">
                    <label className="container-field"><span>Environment</span><select value={environment} onChange={(event) => setEnvironment(event.target.value as ContainerDeploymentSpec["environment"])}><option value="development">Development</option><option value="staging">Staging</option><option value="production">Production</option></select></label>
                    <label className="container-field"><span>Description (optional)</span><input value={description} onChange={(event) => setDescription(event.target.value)} maxLength={2000} placeholder="What this workload does" /></label>
                  </div>
                </>
              )}
              {step === 2 && (
                <>
                  <label className="container-field"><span>Replicas</span><input type="number" min={1} max={50} value={replicas} onChange={(event) => setReplicas(Number(event.target.value))} /></label>
                  <div className="container-field-grid">
                    <label className="container-field"><span>CPU request</span><input value={cpuRequest} onChange={(event) => setCPURequest(event.target.value)} placeholder="250m" /></label>
                    <label className="container-field"><span>CPU limit</span><input value={cpuLimit} onChange={(event) => setCPULimit(event.target.value)} placeholder="1000m" /></label>
                    <label className="container-field"><span>Memory request</span><input value={memoryRequest} onChange={(event) => setMemoryRequest(event.target.value)} placeholder="256Mi" /></label>
                    <label className="container-field"><span>Memory limit</span><input value={memoryLimit} onChange={(event) => setMemoryLimit(event.target.value)} placeholder="1Gi" /></label>
                  </div>
                </>
              )}
              {step === 3 && (
                <>
                  {ports.map((port, index) => (
                    <div className="container-field-row" key={`port-${index}`}>
                      <label className="container-field"><span>Container port</span><input type="number" min={1} max={65535} value={port.container_port || ""} onChange={(event) => updatePort(index, { container_port: Number(event.target.value) })} /></label>
                      <label className="container-field"><span>Service port</span><input type="number" min={1} max={65535} value={port.service_port || ""} onChange={(event) => updatePort(index, { service_port: Number(event.target.value) })} /></label>
                      <label className="container-field"><span>Protocol</span><select value={port.protocol ?? "TCP"} onChange={(event) => updatePort(index, { protocol: event.target.value as "TCP" | "UDP" })}><option>TCP</option><option>UDP</option></select></label>
                    </div>
                  ))}
                  <button className="container-button secondary" type="button" onClick={() => setPorts((current) => [...current, { container_port: 8080, service_port: 80, protocol: "TCP" }])}>＋ Add port</button>
                  <small>Services are private to the Kubernetes cluster; public load balancers are not configured.</small>
                </>
              )}
              {step === 4 && (
                <>
                  {environmentVariables.map((item, index) => (
                    <div className="container-variable-row" key={`env-${index}`}>
                      <label className="container-field"><span>Name</span><input value={item.name} onChange={(event) => updateEnvironmentVariable(index, { name: event.target.value })} placeholder="API_TOKEN" /></label>
                      <label className="container-field"><span>Value</span><input type={item.secret ? "password" : "text"} value={item.value} onChange={(event) => updateEnvironmentVariable(index, { value: event.target.value })} autoComplete="new-password" /></label>
                      <label className="container-check"><input type="checkbox" checked={item.secret} onChange={(event) => updateEnvironmentVariable(index, { secret: event.target.checked })} /> Secret</label>
                      <button type="button" className="container-button danger" onClick={() => setEnvironmentVariables((current) => current.filter((_, itemIndex) => itemIndex !== index))}>Remove</button>
                    </div>
                  ))}
                  <button type="button" className="container-button secondary" onClick={() => setEnvironmentVariables((current) => [...current, { name: "", value: "", secret: false }])}>＋ Add variable</button>
                  <small>Secret values are encrypted before persistence and never displayed again after submission.</small>
                </>
              )}
              {step === 5 && (
                <>
                  {volumes.map((volume, index) => (
                    <div className="container-field-grid" key={`volume-${index}`}>
                      <label className="container-field"><span>Volume name</span><input value={volume.name} onChange={(event) => updateVolume(index, { name: event.target.value })} placeholder="app-data" /></label>
                      <label className="container-field"><span>Size (GiB)</span><input type="number" min={1} value={volume.size_gib || ""} onChange={(event) => updateVolume(index, { size_gib: Number(event.target.value) })} /></label>
                      <label className="container-field"><span>Mount path</span><input value={volume.mount_path} onChange={(event) => updateVolume(index, { mount_path: event.target.value })} placeholder="/var/lib/app" /></label>
                      <label className="container-field"><span>Storage class (optional)</span><input value={volume.storage_class ?? ""} onChange={(event) => updateVolume(index, { storage_class: event.target.value })} /></label>
                      <button type="button" className="container-button danger" onClick={() => setVolumes((current) => current.filter((_, itemIndex) => itemIndex !== index))}>Remove volume</button>
                    </div>
                  ))}
                  <button type="button" className="container-button secondary" onClick={() => setVolumes((current) => [...current, { name: "", size_gib: 1, mount_path: "" }])}>＋ Add persistent volume</button>
                  {volumes.length > 0 && <label className="container-check"><input type="checkbox" checked={retainVolumes} onChange={(event) => setRetainVolumes(event.target.checked)} /> Retain volumes when deleting this deployment</label>}
                  <small>Storage provisioning depends on an available Kubernetes storage class.</small>
                </>
              )}
              {step === 6 && (
                <>
                  <label className="container-field"><span>Readiness HTTP path (optional)</span><input value={readinessPath} onChange={(event) => setReadinessPath(event.target.value)} placeholder="/health/ready" /></label>
                  <label className="container-field"><span>Liveness HTTP path (optional)</span><input value={livenessPath} onChange={(event) => setLivenessPath(event.target.value)} placeholder="/health/live" /></label>
                  <small>Health checks use the first configured container port. Leave blank to use Kubernetes defaults.</small>
                </>
              )}
              {step === 7 && (
                <>
                  <div className="container-review">
                    <div><small>Workspace / project</small><strong>{currentTenant?.name ?? "—"} / {currentProject?.name ?? "—"}</strong></div>
                    <div><small>Deployment</small><strong>{name || "—"} · {environment}</strong></div>
                    <div><small>Container image</small><strong>{image || "—"}</strong></div>
                    <div><small>Replicas</small><strong>{replicas}</strong></div>
                    <div><small>Resources</small><strong>{cpuRequest} CPU · {memoryRequest} memory request</strong></div>
                    <div><small>Ports</small><strong>{ports.filter((port) => port.container_port > 0).map((port) => `${port.service_port ?? port.container_port} → ${port.container_port}/${port.protocol ?? "TCP"}`).join(", ") || "None"}</strong></div>
                    <div><small>Environment variables</small><strong>{environmentVariables.length ? `${environmentVariables.length} configured; secret values masked` : "None"}</strong></div>
                    <div><small>Persistent volumes</small><strong>{volumes.length || "None"}</strong></div>
                  </div>
                  <div className="container-info">Deployment requires a configured Kubernetes runtime. This creates a private ClusterIP service; no public endpoint is assigned.</div>
                </>
              )}
            </div>
          )}

          {error && (
            <div className="container-alert" role="alert">
              {error}{error.toLowerCase().includes("authentication required") && <> <Link href="/login">Sign in</Link> to continue.</>}
            </div>
          )}
          <div className="container-wizard-actions">
            <button type="button" className="container-button secondary" onClick={() => { setError(""); setStep((current) => Math.max(current - 1, 0)); }} disabled={step === 0 || submitting}>Back</button>
            {step < steps.length - 1
              ? <button type="button" className="container-button" onClick={nextStep} disabled={scopeLoading}>Continue</button>
              : <button type="button" className="container-button" onClick={() => void submitDeployment()} disabled={submitting || scopeLoading || !projectID}>{submitting ? "Submitting…" : "Create deployment"}</button>}
          </div>
        </section>
        <aside className="container-card container-sticky-summary" aria-label="Deployment summary">
          <div className="container-summary-heading">
            <div><strong>Deployment summary</strong><small>Draft auto-saved in this browser</small></div>
            <span aria-hidden="true">●</span>
          </div>
          <dl>
            <div><dt>Workspace</dt><dd>{currentTenant?.name || "Choose workspace"}</dd></div>
            <div><dt>Project</dt><dd>{currentProject?.name || "Choose project"}</dd></div>
            <div><dt>Name</dt><dd>{name || "—"}</dd></div>
            <div><dt>Environment</dt><dd>{environment}</dd></div>
            <div><dt>Image</dt><dd>{image || "—"}</dd></div>
            <div><dt>Compute</dt><dd>{replicas} replica{replicas === 1 ? "" : "s"} · {cpuRequest} CPU · {memoryRequest}</dd></div>
            <div><dt>Service ports</dt><dd>{visiblePorts.length ? visiblePorts.map((port) => `${port.service_port ?? port.container_port}→${port.container_port}`).join(", ") : "None"}</dd></div>
            <div><dt>Variables / volumes</dt><dd>{environmentVariables.length} / {volumes.length}</dd></div>
          </dl>
          <p>Secret values are intentionally not saved. Re-enter them after restoring a draft.</p>
        </aside>
      </div>
    </>
  );
}
