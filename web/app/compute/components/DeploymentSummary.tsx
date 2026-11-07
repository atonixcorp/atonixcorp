"use client";

import { useState } from "react";
import type { ComputeCatalog, ComputePlan, DeploymentDraft } from "../lib/types";

export function DeploymentSummary({
  catalog, draft, plan, blocked, onReview, onDeploy,
}: {
  catalog: ComputeCatalog;
  draft: DeploymentDraft;
  plan?: ComputePlan;
  blocked: boolean;
  onReview: () => void;
  onDeploy: () => void;
}) {
  const [expanded, setExpanded] = useState(false);
  const region = catalog.regions.find((item) => item.id === draft.region);
  const image = catalog.images.find((item) => item.id === draft.image);
  const network = catalog.networks.find((item) => item.id === draft.network);
  const type = catalog.compute_types.find((item) => item.id === draft.computeType);

  return (
    <>
      <aside className="compute-summary-card" aria-label="Deployment summary">
        <h2>Deployment summary</h2>
        <div className="compute-summary-line"><span>Compute</span><strong>{type?.name ?? "Choose a type"}</strong></div>
        <div className="compute-summary-line"><span>Region</span><strong>{region ? `${region.name} · ${region.id.toUpperCase()}` : "Choose a region"}</strong></div>
        <div className="compute-summary-line"><span>Image</span><strong>{image?.name ?? "Choose an image"}</strong></div>
        <div className="compute-summary-line"><span>Flavor</span><strong>{plan?.name ?? "Choose a flavor"}</strong></div>
        <div className="compute-summary-line"><span>Network</span><strong>{network?.name ?? "Choose a network"}</strong></div>
        <div className="compute-summary-specs">
          <span>{plan ? `${plan.vcpus} vCPU` : "— vCPU"}</span>
          <span>{plan ? `${Math.round(plan.memory_mb / 1024)} GB RAM` : "— GB RAM"}</span>
          <span>{plan ? `${plan.boot_disk_gb} GB root disk` : "— GB root disk"}</span>
        </div>
        <div className="compute-summary-price">
          <small>Pricing</small>
          <strong>Unavailable</strong>
        </div>
        <p className="compute-summary-disclaimer">Prices are not provided by the OpenStack account. Deployment is not blocked by missing pricing.</p>
        <button type="button" className="compute-secondary-button compute-summary-review" onClick={onReview}>Review configuration</button>
        <button type="button" className="compute-primary-button compute-summary-deploy" disabled={blocked} onClick={onDeploy}>
          Deploy instance
        </button>
        {blocked && <small className="compute-summary-blocked">Choose an authorized project and all required live provider resources.</small>}
      </aside>
      <div className={`compute-mobile-deploy-bar${expanded ? " expanded" : ""}`}>
        <button type="button" className="compute-mobile-summary-toggle" aria-expanded={expanded} onClick={() => setExpanded(!expanded)}>
          <span><strong>Deployment summary</strong><small>{plan ? `${plan.vcpus} vCPU · ${Math.round(plan.memory_mb / 1024)} GB RAM` : "Choose an instance size"}</small></span>
          <strong>Pricing unavailable</strong>
        </button>
        {expanded && <p>{type?.name ?? "Compute type"} · {region?.name ?? "Region"} · {image?.name ?? "Image"}</p>}
        <button type="button" className="compute-primary-button" disabled={blocked} onClick={onReview}>Review & deploy</button>
      </div>
    </>
  );
}
