"use client";

import Link from "next/link";
import { useParams } from "next/navigation";
import { useEffect, useRef, useState } from "react";
import { getInstance } from "../../lib/api";
import type { ComputeInstance } from "../../lib/types";

const provisioningSteps = [
  ["creating_instance", "Request accepted"],
  ["allocating_compute", "Allocating compute"],
  ["attaching_storage", "Attaching storage"],
  ["configuring_network", "Configuring network"],
  ["installing_image", "Installing image"],
  ["applying_ssh_keys", "Applying SSH keys"],
  ["applying_firewall", "Applying firewall"],
  ["running_initialization", "Running initialization"],
  ["health_checks", "Health checks"],
];

export default function ComputeInstanceDetailsPage() {
  const { instanceId } = useParams<{ instanceId: string }>();
  const [instance, setInstance] = useState<ComputeInstance>();
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(true);
  const instanceStatus = useRef("");

  useEffect(() => {
    let active = true;
    let attempts = 0;
    const refresh = async () => {
      try {
        const result = await getInstance(instanceId);
        if (!active) return;
        instanceStatus.current = result.status;
        setInstance(result);
        setError("");
      } catch (requestError) {
        if (!active) return;
        setError(requestError instanceof Error ? requestError.message : "Could not load this instance.");
      } finally {
        if (active) setLoading(false);
      }
    };
    void refresh();
    const timer = window.setInterval(() => {
      attempts += 1;
      if (attempts >= 60 || (instanceStatus.current && instanceStatus.current !== "provisioning")) {
        window.clearInterval(timer);
        return;
      }
      void refresh();
    }, 5000);
    return () => {
      active = false;
      window.clearInterval(timer);
    };
  }, [instanceId]);

  if (loading) {
    return <div className="compute-page"><p className="compute-eyebrow">COMPUTE / INSTANCE</p><h1>Loading instance…</h1><div className="compute-deploy-skeleton"><div><i /><i /><i /></div></div></div>;
  }
  if (!instance) {
    return (
      <div className="compute-page">
        <header className="compute-page-heading"><div><p className="compute-eyebrow">COMPUTE / INSTANCE</p><h1>Instance details</h1></div><Link className="compute-secondary-button" href="/compute/instances">Back to instances</Link></header>
        <div className="compute-state-panel" role="alert"><h3>Instance details are unavailable</h3><p>{error}</p><p>Instance details require an authenticated account and tenant session.</p></div>
      </div>
    );
  }

  return (
    <div className="compute-page">
      <header className="compute-page-heading">
        <div><p className="compute-eyebrow"><Link href="/compute/instances">COMPUTE / INSTANCES</Link> / {instance.id}</p><h1>{instance.label || instance.name}</h1><p>{instance.image_id} · {instance.region.toUpperCase()} · {instance.public_ip || "Public IP pending"}</p></div>
        <div className="compute-detail-actions">
          <span className={`compute-instance-status ${instance.status}`}>{instance.status}</span>
          <button className="compute-secondary-button" type="button" disabled title="Lifecycle actions require the provisioning service.">Start</button>
          <button className="compute-secondary-button" type="button" disabled title="Lifecycle actions require the provisioning service.">Stop</button>
          <button className="compute-secondary-button" type="button" disabled title="Lifecycle actions require the provisioning service.">Restart</button>
        </div>
      </header>
      {error && <p className="compute-inline-error" role="alert">{error}</p>}
      <section className="compute-card compute-timeline-card">
          <header className="compute-table-toolbar"><div><h2>{instance.status === "failed" ? "Provisioning failed" : "Provisioning instance"}</h2><p>Status updates refresh automatically while provisioning.</p></div></header>
          <ol className="compute-timeline">
            {provisioningSteps.map(([key, title]) => {
              const step = instance.provisioning.find((item) => item.name === key);
              const status = step?.status ?? "pending";
              return <li className={`compute-timeline-step ${status}`} key={key}><span aria-hidden="true">{status === "completed" ? "✓" : status === "in_progress" ? "…" : "○"}</span><div><strong>{title}</strong><small>{status === "in_progress" ? "In progress" : status === "completed" ? "Complete" : status === "failed" ? step?.message || "Failed" : status === "queued" ? "Queued" : "Waiting"}</small></div></li>;
            })}
            {instance.status === "running" && <li className="compute-timeline-step completed"><span aria-hidden="true">✓</span><div><strong>Running</strong><small>Instance is running</small></div></li>}
          </ol>
      </section>
      <div className="compute-detail-tabs" role="tablist" aria-label="Instance sections">
        {["Overview", "Metrics", "Networking", "Storage", "Snapshots", "Backups", "Firewall", "Console", "Activity", "Settings"].map((tab) => <button type="button" role="tab" aria-selected={tab === "Overview"} key={tab}>{tab}</button>)}
      </div>
      <section className="compute-card compute-detail-overview">
        <h2>Instance information</h2>
        <dl>
          <div><dt>Status</dt><dd>{instance.status}</dd></div><div><dt>Hostname</dt><dd>{instance.hostname}</dd></div>
          <div><dt>Region</dt><dd>{instance.region.toUpperCase()}</dd></div><div><dt>Image</dt><dd>{instance.image_id}</dd></div>
          <div><dt>Plan</dt><dd>{instance.plan_id}</dd></div><div><dt>Network</dt><dd>{instance.network_id || "Not assigned"}</dd></div>
          <div><dt>Public IPv4</dt><dd>{instance.public_ip || "Pending"}</dd></div><div><dt>Private IPv4</dt><dd>{instance.private_ip || "Pending"}</dd></div>
          <div><dt>Created</dt><dd>{new Date(instance.created_at).toLocaleString()}</dd></div>
        </dl>
      </section>
    </div>
  );
}
