"use client";

import Link from "next/link";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useRouter } from "next/navigation";
import {
  getCatalog,
  getEstimate,
  getSecurityCatalog,
  deployInstances,
} from "../lib/api";
import {
  draftStorageKey,
  restoreDraft,
  saveDraft,
  validateDraft,
} from "../lib/deployment-state";
import type {
  ComputeCatalog,
  ComputeEstimate,
  ComputeSSHKey,
  DeploymentDraft,
} from "../lib/types";
import { initialDraft } from "../lib/types";
import {
  ComputeTypeSection,
  IdentitySection,
  ImageSection,
  InitializationSection,
  NetworkSection,
  PlanSection,
  RegionSection,
  ReviewSection,
  SecuritySection,
  StorageSection,
} from "../components/DeploymentSections";
import { DeploymentSummary } from "../components/DeploymentSummary";

const emptyCatalog: ComputeCatalog = {
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
  const [securityError, setSecurityError] = useState("");
  const [sshKeys, setSSHKeys] = useState<ComputeSSHKey[]>([]);
  const [draft, setDraft] = useState<DeploymentDraft>(initialDraft);
  const [draftReady, setDraftReady] = useState(false);
  const [estimate, setEstimate] = useState<ComputeEstimate>();
  const [estimateError, setEstimateError] = useState("");
  const [estimating, setEstimating] = useState(false);
  const [fieldError, setFieldError] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [submitError, setSubmitError] = useState("");
  const [showReview, setShowReview] = useState(false);
  const [draftSaved, setDraftSaved] = useState(true);
  const estimateRequest = useRef(0);

  const update = useCallback((patch: Partial<DeploymentDraft>) => {
    setDraft((current) => ({ ...current, ...patch }));
    setFieldError("");
    setSubmitError("");
  }, []);

  const loadCatalog = useCallback(async () => {
    setLoading(true);
    setCatalogError("");
    try {
      const data = await getCatalog();
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
      try {
        const security = await getSecurityCatalog();
        setSSHKeys(security.sshKeys);
        setSecurityError("");
      } catch (error) {
        setSSHKeys([]);
        setSecurityError(error instanceof Error ? error.message : "Account security is unavailable.");
      }
    } catch (error) {
      setCatalogError(error instanceof Error ? error.message : "Compute catalog is unavailable.");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    const timeout = window.setTimeout(() => { void loadCatalog(); }, 0);
    return () => window.clearTimeout(timeout);
  }, [loadCatalog]);

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

  useEffect(() => {
    if (!draftReady) return;
    const requestId = ++estimateRequest.current;
    if (!draft.plan) {
      const timeout = window.setTimeout(() => {
        setEstimate(undefined);
        setEstimating(false);
        setEstimateError("");
      }, 0);
      return () => window.clearTimeout(timeout);
    }
    const timeout = window.setTimeout(() => {
      setEstimating(true);
      setEstimateError("");
      void getEstimate(draft).then((result) => {
        if (estimateRequest.current !== requestId) return;
        setEstimate(result);
      }).catch((error: unknown) => {
        if (estimateRequest.current !== requestId) return;
        setEstimate(undefined);
        setEstimateError(error instanceof Error ? error.message : "Could not calculate the estimate.");
      }).finally(() => {
        if (estimateRequest.current === requestId) setEstimating(false);
      });
    }, 350);
    return () => {
      window.clearTimeout(timeout);
      estimateRequest.current += 1;
    };
  }, [draft, draftReady]);

  const validate = (requireSSHKey: boolean) => validateDraft(draft, catalog, sshKeys, requireSSHKey);
  const allErrors = validate(true);
  const missingAuthoritativePricing = !estimate || !estimate.authoritative;
  const deploymentBlocked = Boolean(
    loading || catalogError || securityError || allErrors || estimating ||
    estimateError || missingAuthoritativePricing || submitting,
  );

  const goToSection = (section: string) => {
    setShowReview(section === "review");
    window.requestAnimationFrame(() => {
      document.getElementById(section)?.scrollIntoView({ behavior: "smooth", block: "start" });
      document.getElementById(section)?.focus({ preventScroll: true });
    });
  };

  const openReview = () => {
    const issue = validate(false);
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
    const issue = validate(true);
    if (issue) {
      setFieldError(issue.message);
      goToSection(issue.section);
      return;
    }
    if (!estimate?.authoritative) {
      setSubmitError("Deployment is paused until an authenticated billing service returns an authoritative quote.");
      return;
    }
    setSubmitting(true);
    setSubmitError("");
    try {
      const idempotencyKey = crypto.randomUUID();
      const result = await deployInstances(draft, idempotencyKey);
      window.localStorage.removeItem(draftStorageKey);
      const firstInstance = result.instances[0];
      if (!firstInstance?.id) throw new Error("The Compute API accepted no instance IDs.");
      router.push(`/compute/instances/${encodeURIComponent(firstInstance.id)}`);
    } catch (error) {
      setSubmitError(error instanceof Error ? error.message : "The deployment could not be submitted.");
    } finally {
      setSubmitting(false);
    }
  };

  if (loading && !catalog.compute_types.length) {
    return <ComputeLoadingState />;
  }

  if (catalogError && !catalog.compute_types.length) {
    return (
      <div className="compute-page">
        <header className="compute-page-heading"><div><p className="compute-eyebrow">COMPUTE / DEPLOY</p><h1>Deploy instance</h1></div><Link className="compute-secondary-button" href="/compute/instances">Back to instances</Link></header>
        <div className="compute-state-panel" role="alert"><h3>Compute options could not be loaded</h3><p>{catalogError}</p><button className="compute-secondary-button" type="button" onClick={() => void loadCatalog()}>Retry</button></div>
      </div>
    );
  }

  return (
    <div className="compute-page compute-deploy-page">
      <header className="compute-page-heading">
        <div><p className="compute-eyebrow">COMPUTE / DEPLOY</p><h1>Deploy an instance</h1><p>Configure a virtual machine. Advanced settings are available below.</p></div>
        <Link className="compute-secondary-button" href="/compute/instances">Cancel</Link>
      </header>
      <div className="compute-deploy-notice" role="status">
        <strong>Deployment preview</strong>
        <span>This dashboard currently uses demo sign-in. Account authorization, tenant-scoped SSH keys, and live billing are not connected, so live deployment is disabled.</span>
      </div>
      {!draftSaved && <p className="compute-inline-error" role="status">This browser could not save your deployment draft.</p>}
      {fieldError && <p className="compute-inline-error" role="alert">{fieldError}</p>}
      {submitError && <p className="compute-inline-error" role="alert">{submitError}</p>}
      {estimateError && <p className="compute-inline-error" role="alert">{estimateError}</p>}

      <div className="compute-deploy-layout">
        <div className="compute-deploy-sections">
          <ComputeTypeSection catalog={catalog} draft={draft} update={update} />
          <RegionSection catalog={catalog} draft={draft} update={update} />
          <ImageSection images={catalog.images} draft={draft} update={update} />
          <PlanSection plans={availablePlans} draft={draft} update={update} currency={availablePlans[0]?.currency ?? "ZAR"} />
          <StorageSection plan={selectedPlan} draft={draft} update={update} />
          <NetworkSection catalog={catalog} draft={draft} update={update} />
          <SecuritySection keys={sshKeys} keyError={securityError} draft={draft} update={update} onRetry={() => void loadCatalog()} />
          <InitializationSection draft={draft} update={update} />
          <IdentitySection draft={draft} update={update} />
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
          estimate={estimate}
          estimating={estimating}
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
