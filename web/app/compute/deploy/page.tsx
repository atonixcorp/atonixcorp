"use client";

import Link from "next/link";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useRouter } from "next/navigation";
import {
  getCatalog,
  getProjects,
  getTenants,
  createInstance,
} from "../lib/api";
import type { ComputeProject, ComputeTenant } from "../lib/api";
import { restoreDraft, saveDraft } from "../lib/deployment-state";
import type {
  ComputeCatalog,
  DeploymentDraft,
} from "../lib/types";
import { initialDraft } from "../lib/types";
import {
  ComputeTypeSection,
  ImageSection,
  NetworkSection,
  PlanSection,
  RegionSection,
  ReviewSection,
} from "../components/DeploymentSections";
import { DeploymentSummary } from "../components/DeploymentSummary";
import { initialScopeID } from "@/app/shared/lib/cloud-scope";

const emptyCatalog: ComputeCatalog = {
  provisioning_enabled: false,
  pricing_available: false,
  compute_types: [],
  regions: [],
  images: [],
  plans: [],
  networks: [],
};

export default function DeployInstancePage() {
  const router = useRouter();
  const [catalog, setCatalog] = useState<ComputeCatalog>(emptyCatalog);
  const [catalogError, setCatalogError] = useState("");
  const [loading, setLoading] = useState(true);
  const [scopeError, setScopeError] = useState("");
  const [tenants, setTenants] = useState<ComputeTenant[]>([]);
  const [projects, setProjects] = useState<ComputeProject[]>([]);
  const [tenantID, setTenantID] = useState("");
  const [projectID, setProjectID] = useState("");
  const [draft, setDraft] = useState<DeploymentDraft>(initialDraft);
  const [draftReady, setDraftReady] = useState(false);
  const [fieldError, setFieldError] = useState("");
  const [submitError, setSubmitError] = useState("");
  const [showReview, setShowReview] = useState(false);
  const [draftSaved, setDraftSaved] = useState(true);
  const [submitting, setSubmitting] = useState(false);
  const idempotencyKey = useRef("");

  const update = useCallback((patch: Partial<DeploymentDraft>) => {
    idempotencyKey.current = "";
    setDraft((current) => ({ ...current, ...patch }));
    setFieldError("");
    setSubmitError("");
  }, []);

  const loadCatalog = useCallback(async () => {
    if (!tenantID || !projectID) return;
    setLoading(true);
    setCatalogError("");
    try {
      const data = await getCatalog(Number(tenantID), Number(projectID));
      setCatalog(data);
      setDraft((current) => {
        const saved = restoreDraft();
        const base = saved ?? current;
        const computeType = data.compute_types.some((item) => item.id === base.computeType)
          ? base.computeType : (data.compute_types[0]?.id ?? "");
        const region = data.regions.some((item) => item.id === base.region && item.available)
          ? base.region : (data.regions.find((item) => item.available)?.id ?? "");
        const planOptions = data.plans.filter((item) =>
          item.available && item.region === region && item.compute_type === computeType);
        const plan = planOptions.some((item) => item.id === base.plan)
          ? base.plan : (planOptions[0]?.id ?? "");
        const image = data.images.some((item) => item.id === base.image && item.available)
          ? base.image : (data.images.find((item) => item.available)?.id ?? "");
        const network = data.networks.some((item) =>
          item.id === base.network && item.region === region && item.available)
          ? base.network : (data.networks.find((item) => item.region === region && item.available)?.id ?? "");
        return { ...base, computeType, region, plan, image, network };
      });
      setDraftReady(true);
    } catch (error) {
      setCatalogError(error instanceof Error ? error.message : "Compute catalog is unavailable.");
    } finally {
      setLoading(false);
    }
  }, [projectID, tenantID]);

  useEffect(() => {
    let active = true;
    void getTenants().then((items) => {
      if (!active) return;
      setTenants(items);
      setTenantID(String(initialScopeID(items, "tenant_id") || ""));
      if (!items.length) {
        setScopeError("No accessible workspaces were found.");
        setLoading(false);
      }
    }).catch((error: unknown) => {
      if (active) {
        setScopeError(error instanceof Error ? error.message : "Could not load workspaces.");
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
      setProjectID(String(initialScopeID(items, "project_id", Number(tenantID)) || ""));
      setScopeError(items.length ? "" : "No accessible projects were found.");
      if (!items.length) setLoading(false);
    }).catch((error: unknown) => {
      if (active) {
        setScopeError(error instanceof Error ? error.message : "Could not load projects.");
        setLoading(false);
      }
    });
    return () => { active = false; };
  }, [tenantID]);

  useEffect(() => {
    if (!tenantID || !projectID) return;
    const timeout = window.setTimeout(() => { void loadCatalog(); }, 0);
    return () => window.clearTimeout(timeout);
  }, [loadCatalog, projectID, tenantID]);

  useEffect(() => {
    if (!draftReady) return;
    const timeout = window.setTimeout(() => setDraftSaved(saveDraft(draft)), 0);
    return () => window.clearTimeout(timeout);
  }, [draft, draftReady]);

  const availablePlans = useMemo(
    () => catalog.plans.filter((plan) => plan.available && plan.region === draft.region && plan.compute_type === draft.computeType),
    [catalog.plans, draft.computeType, draft.region],
  );
  const selectedPlan = availablePlans.find((plan) => plan.id === draft.plan);

  const validate = () => {
    if (!catalog.compute_types.some((item) => item.id === draft.computeType)) return { section: "compute-type", message: "Choose an available compute type." };
    if (!catalog.regions.some((item) => item.id === draft.region && item.available)) return { section: "region", message: "Choose an available region." };
    if (!catalog.images.some((item) => item.id === draft.image && item.available)) return { section: "image", message: "Choose an available operating system image." };
    if (!catalog.plans.some((item) => item.id === draft.plan && item.available)) return { section: "plan", message: "Choose an available instance size." };
    if (!catalog.networks.some((item) => item.id === draft.network && item.available)) return { section: "network", message: "Choose an available network." };
    if (!/^[a-zA-Z0-9][a-zA-Z0-9._-]{0,62}$/.test(draft.name)) return { section: "identity", message: "Use 1–63 letters, numbers, dots, underscores, or hyphens for the instance name." };
    return null;
  };
  const allErrors = validate();
  const deploymentBlocked = Boolean(
    !tenantID || !projectID || !catalog.provisioning_enabled || loading ||
    catalogError || scopeError || allErrors || submitting ||
    !projects.some((project) => project.id === Number(projectID) && project.tenant_id === Number(tenantID)),
  );

  const goToSection = (section: string) => {
    setShowReview(section === "review");
    window.requestAnimationFrame(() => {
      document.getElementById(section)?.scrollIntoView({ behavior: "smooth", block: "start" });
      document.getElementById(section)?.focus({ preventScroll: true });
    });
  };

  const openReview = () => {
    const issue = validate();
    if (issue) {
      setFieldError(issue.message);
      goToSection(issue.section);
      return;
    }
    setFieldError("");
    setShowReview(true);
    goToSection("review");
  };

  const deploy = async () => {
    if (deploymentBlocked || !tenantID || !projectID) return;
    if (!idempotencyKey.current) idempotencyKey.current = crypto.randomUUID();
    setSubmitting(true);
    setSubmitError("");
    try {
      const instance = await createInstance({
        tenant_id: Number(tenantID),
        project_id: Number(projectID),
        name: draft.name,
        region_id: draft.region,
        image_id: draft.image,
        size_id: draft.plan,
        network_id: draft.network,
      }, idempotencyKey.current);
      idempotencyKey.current = "";
      router.push(`/compute/instances/${encodeURIComponent(instance.id)}`);
    } catch (error) {
      setSubmitError(error instanceof Error ? error.message : "The virtual machine could not be created.");
    } finally {
      setSubmitting(false);
    }
  };

  if ((loading && !catalog.compute_types.length) || (!tenants.length && !scopeError)) {
    return <ComputeLoadingState />;
  }

  if ((catalogError || scopeError) && !catalog.compute_types.length) {
    return (
      <div className="compute-page">
        <header className="compute-page-heading"><div><p className="compute-eyebrow">COMPUTE / DEPLOY</p><h1>Deploy instance</h1></div><Link className="compute-secondary-button" href="/compute/instances">Back to instances</Link></header>
        <div className="compute-state-panel" role="alert"><h3>Compute options could not be loaded</h3><p>{catalogError || scopeError}</p><button className="compute-secondary-button" type="button" onClick={() => void loadCatalog()}>Retry</button></div>
      </div>
    );
  }

  return (
    <div className="compute-page compute-deploy-page">
      <header className="compute-page-heading">
        <div><p className="compute-eyebrow">COMPUTE / DEPLOY</p><h1>Deploy an instance</h1><p>Launch one virtual machine using the resources available to your project.</p></div>
        <Link className="compute-secondary-button" href="/compute/instances">Cancel</Link>
      </header>
      <div className="compute-deploy-notice" role="status">
        <strong>{catalog.provisioning_enabled ? "Compute resources available" : "Deployment unavailable"}</strong>
        <span>{catalog.provisioning_enabled ? "Images, sizes, and networks below are available to the selected project. Pricing is unavailable." : "A valid project and available compute, image, and network services are required."}</span>
      </div>
      {scopeError && <p className="compute-inline-error" role="alert">{scopeError}</p>}
      <div className="compute-form-grid">
        <label className="compute-field">Workspace<select value={tenantID} onChange={(event) => { idempotencyKey.current = ""; setCatalog(emptyCatalog); setCatalogError(""); setScopeError(""); setLoading(true); setProjects([]); setProjectID(""); setTenantID(event.target.value); }}>{tenants.map((tenant) => <option key={tenant.id} value={tenant.id}>{tenant.name}</option>)}</select></label>
        <label className="compute-field">Project<select value={projectID} onChange={(event) => { idempotencyKey.current = ""; setCatalog(emptyCatalog); setCatalogError(""); setScopeError(""); setLoading(true); setProjectID(event.target.value); }}>{projects.map((project) => <option key={project.id} value={project.id}>{project.name}</option>)}</select></label>
      </div>
      {catalogError && <p className="compute-inline-error" role="alert">{catalogError}</p>}
      {!draftSaved && <p className="compute-inline-error" role="status">This browser could not save your deployment draft.</p>}
      {fieldError && <p className="compute-inline-error" role="alert">{fieldError}</p>}
      {submitError && <p className="compute-inline-error" role="alert">{submitError}</p>}

      <div className="compute-deploy-layout">
        <div className="compute-deploy-sections">
          <ComputeTypeSection catalog={catalog} draft={draft} update={update} />
          <RegionSection catalog={catalog} draft={draft} update={update} />
          <ImageSection images={catalog.images} draft={draft} update={update} />
          <PlanSection plans={availablePlans} draft={draft} update={update} />
          <NetworkSection catalog={catalog} draft={draft} update={update} />
          <section className="compute-card compute-deploy-section" id="identity" tabIndex={-1}>
            <header className="compute-section-heading"><span>07</span><div><h2>Server identity</h2><p>Choose a name for your virtual machine.</p></div></header>
            <div className="compute-section-body"><label className="compute-field">Instance name<input value={draft.name} maxLength={63} pattern="[A-Za-z0-9][A-Za-z0-9._-]{0,62}" required placeholder="worker-01" onChange={(event) => update({ name: event.target.value })} /></label></div>
          </section>
          {showReview && (
            <ReviewSection
              catalog={catalog}
              draft={draft}
              plan={selectedPlan}
              onEdit={goToSection}
              onDeploy={() => void deploy()}
              deployBlocked={deploymentBlocked}
              submitting={submitting}
            />
          )}
          {!showReview && (
            <section className="compute-review-entry">
              <button type="button" className="compute-primary-button" onClick={openReview}>Review configuration <span aria-hidden="true">→</span></button>
              <span>Review will preserve your selections.</span>
            </section>
          )}
        </div>
        <DeploymentSummary
          catalog={catalog}
          draft={draft}
          plan={selectedPlan}
          blocked={deploymentBlocked}
          onReview={openReview}
          onDeploy={() => void deploy()}
        />
      </div>
    </div>
  );
}

function ComputeLoadingState() {
  return (
    <div className="compute-page">
      <div className="compute-page-heading"><div><p className="compute-eyebrow">COMPUTE / DEPLOY</p><h1>Deploy an instance</h1><p>Loading available compute options…</p></div></div>
      <div className="compute-deploy-skeleton" aria-label="Loading deployment options">
        {[0, 1, 2, 3].map((item) => <div key={item}><i /><i /><i /></div>)}
      </div>
    </div>
  );
}
