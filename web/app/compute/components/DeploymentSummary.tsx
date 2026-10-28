"use client";

import { useState } from "react";
import type { ComputeCatalog, ComputeEstimate, ComputePlan, DeploymentDraft } from "../lib/types";
import { formatMoney } from "./DeploymentSections";

export function DeploymentSummary({
  catalog, draft, plan, estimate, estimating, blocked, onReview, onDeploy,
}: {
  catalog: ComputeCatalog;
  draft: DeploymentDraft;
  plan?: ComputePlan;
  estimate?: ComputeEstimate;
  estimating: boolean;
  blocked: boolean;
  onReview: () => void;
  onDeploy: () => void;
}) {
  const [expanded, setExpanded] = useState(false);
  const region = catalog.regions.find((item) => item.id === draft.region);
  const image = catalog.images.find((item) => item.id === draft.image);
  const type = catalog.compute_types.find((item) => item.id === draft.computeType);
  const volumeGB = draft.volumes.reduce((sum, volume) => sum + volume.size_gb, 0);

  return (
    <>
      <aside className="compute-summary-card" aria-label="Deployment summary">
        <h2>Deployment summary</h2>
        <div className="compute-summary-line"><span>Compute</span><strong>{type?.name ?? "Choose a type"}</strong></div>
        <div className="compute-summary-line"><span>Region</span><strong>{region ? `${region.name} · ${region.id.toUpperCase()}` : "Choose a region"}</strong></div>
        <div className="compute-summary-line"><span>Image</span><strong>{image?.name ?? "Choose an image"}</strong></div>
        <div className="compute-summary-specs">
          <span>{plan ? `${plan.vcpus} vCPU` : "— vCPU"}</span>
          <span>{plan ? `${Math.round(plan.memory_mb / 1024)} GB RAM` : "— GB RAM"}</span>
          <span>{plan ? `${plan.boot_disk_gb} GB NVMe` : "— GB NVMe"}</span>
          {volumeGB > 0 && <span>+ {volumeGB} GB volume</span>}
        </div>
        <div className="compute-summary-line"><span>Quantity</span><strong>{draft.quantity}</strong></div>
        <div className="compute-summary-network">
          {draft.publicIPv4 && <span>IPv4</span>}
          {draft.publicIPv6 && <span>IPv6</span>}
          {draft.privateNetworking && <span>Private network</span>}
          {!draft.publicIPv4 && !draft.publicIPv6 && !draft.privateNetworking && <span>No network access</span>}
        </div>
        <div className="compute-summary-price">
          <small>{estimating ? "Updating estimate…" : estimate?.authoritative ? "Estimated cost" : "Catalog estimate · not a billing quote"}</small>
          <strong>{estimate ? formatMoney(estimate.monthly_price, estimate.currency) : "—"}<span> / month</span></strong>
          <small>{estimate ? `${formatMoney(estimate.hourly_price, estimate.currency)} / hour` : "Select a plan to estimate"}</small>
        </div>
        <p className="compute-summary-disclaimer">Final pricing and availability must be confirmed by billing and provisioning services.</p>
        <button type="button" className="compute-secondary-button compute-summary-review" onClick={onReview}>Review configuration</button>
        <button type="button" className="compute-primary-button compute-summary-deploy" disabled={blocked} onClick={onDeploy}>
          Deploy instance
        </button>
        {blocked && <small className="compute-summary-blocked">Deployment requires account authentication, SSH keys, and an authoritative billing quote.</small>}
      </aside>
      <div className={`compute-mobile-deploy-bar${expanded ? " expanded" : ""}`}>
        <button type="button" className="compute-mobile-summary-toggle" aria-expanded={expanded} onClick={() => setExpanded(!expanded)}>
          <span><strong>Deployment summary</strong><small>{plan ? `${plan.vcpus} vCPU · ${Math.round(plan.memory_mb / 1024)} GB RAM` : "Choose an instance size"}</small></span>
          <strong>{estimate ? formatMoney(estimate.monthly_price, estimate.currency) : "—"} <span>/ mo</span></strong>
        </button>
        {expanded && <p>{type?.name ?? "Compute type"} · {region?.name ?? "Region"} · {image?.name ?? "Image"} · {draft.quantity} instance(s)</p>}
        <button type="button" className="compute-primary-button" disabled={blocked} onClick={onReview}>Review & deploy</button>
      </div>
    </>
  );
}
