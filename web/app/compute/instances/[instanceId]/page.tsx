"use client";

import Link from "next/link";
import { useParams, useRouter, useSearchParams } from "next/navigation";
import { Suspense, useEffect, useRef, useState } from "react";
import { deleteInstance, getInstance, operateInstance } from "../../lib/api";
import type { ComputeInstance } from "../../lib/types";
import { scopedCloudHref } from "@/app/shared/lib/cloud-scope";
import { cloudServices } from "@/app/shared/ui/cloud-navigation";

export default function ComputeInstanceDetailsRoute() {
  return <Suspense fallback={<p role="status">Loading instance...</p>}><ComputeInstanceDetailsPage /></Suspense>;
}

function ComputeInstanceDetailsPage() {
  const { instanceId } = useParams<{ instanceId: string }>();
  const router = useRouter();
  const query = useSearchParams();
  const href = (path: string) => scopedCloudHref(path, Number(query.get("tenant_id")), Number(query.get("project_id")));
  const [instance, setInstance] = useState<ComputeInstance>();
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(true);
  const [operation, setOperation] = useState("");
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
      if (attempts >= 60 || (
        instanceStatus.current &&
        !["provisioning", "starting", "stopping", "rebooting", "resizing", "deleting"].includes(instanceStatus.current)
      )) {
        window.clearInterval(timer);
        return;
      }
      void refresh();
    }, 5000);
    return () => {
      active = false;
      window.clearInterval(timer);
    };
  }, [instance?.status, instanceId]);

  const operate = async (action: "start" | "stop" | "reboot") => {
    setOperation(action);
    setError("");
    try {
      const updated = await operateInstance(instanceId, action);
      instanceStatus.current = updated.status;
      setInstance(updated);
    } catch (requestError) {
      setError(requestError instanceof Error ? requestError.message : `Could not ${action} this instance.`);
    } finally {
      setOperation("");
    }
  };

  const remove = async () => {
    if (!window.confirm("Delete this instance? This action cannot be undone.")) return;
    setOperation("delete");
    setError("");
    try {
      await deleteInstance(instanceId);
      router.push(href("/compute/instances"));
    } catch (requestError) {
      setError(requestError instanceof Error ? requestError.message : "Could not delete this instance.");
      setOperation("");
    }
  };

  const refresh = async () => {
    setOperation("refresh");
    setError("");
    try {
      const updated = await getInstance(instanceId);
      instanceStatus.current = updated.status;
      setInstance(updated);
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : "Could not refresh this instance.");
    } finally {
      setOperation("");
    }
  };

  if (loading) {
    return <div className="compute-page"><p className="compute-eyebrow">COMPUTE / INSTANCE</p><h1>Loading instance…</h1><div className="compute-deploy-skeleton"><div><i /><i /><i /></div></div></div>;
  }
  if (!instance) {
    return (
      <div className="compute-page">
        <header className="compute-page-heading"><div><p className="compute-eyebrow">COMPUTE / INSTANCE</p><h1>Instance details</h1></div><Link className="compute-secondary-button" href={href("/compute/instances")}>Back to instances</Link></header>
        <div className="compute-state-panel" role="alert"><h3>Instance details are unavailable</h3><p>{error}</p><p>Instance details require an authenticated account and tenant session.</p></div>
        <button className="compute-secondary-button" onClick={() => void refresh()} disabled={Boolean(operation)}>Retry</button>
      </div>
    );
  }

  return (
    <div className="compute-page">
      <header className="compute-page-heading">
        <div><p className="compute-eyebrow"><Link href={href("/compute/instances")}>COMPUTE / INSTANCES</Link> / {instance.id}</p><h1>{instance.name}</h1><p>{instance.region ? "Primary region" : "Not assigned"}</p></div>
        <div className="compute-detail-actions">
          <span className={`compute-instance-status ${instance.status}`}>{instance.status}</span>
          <button className="compute-secondary-button" type="button" disabled={Boolean(operation)} onClick={() => void refresh()}>{operation === "refresh" ? "Refreshing..." : "Refresh"}</button>
          <button className="compute-secondary-button" type="button" disabled={Boolean(operation) || instance.status !== "stopped"} onClick={() => void operate("start")}>{operation === "start" ? "Starting…" : "Start"}</button>
          <button className="compute-secondary-button" type="button" disabled={Boolean(operation) || instance.status !== "running"} onClick={() => void operate("stop")}>{operation === "stop" ? "Stopping…" : "Stop"}</button>
          <button className="compute-secondary-button" type="button" disabled={Boolean(operation) || instance.status !== "running"} onClick={() => void operate("reboot")}>{operation === "reboot" ? "Restarting…" : "Restart"}</button>
          <button className="compute-secondary-button" type="button" disabled={Boolean(operation) || instance.status === "provisioning"} onClick={() => void remove()}>{operation === "delete" ? "Deleting…" : "Delete"}</button>
        </div>
      </header>
      {error && <p className="compute-inline-error" role="alert">{error}</p>}
      <section className="compute-card compute-timeline-card" aria-live="polite">
        <header className="compute-table-toolbar">
          <div>
            <h2>Instance status</h2>
            <p>{instance.status}</p>
            <p>Status refreshes automatically while the instance is changing.</p>
          </div>
        </header>
      </section>
      <nav className="compute-detail-actions" aria-label="Related resource workspaces">
        <Link className="compute-secondary-button" href={href("/observability?section=Metrics")}>Project metrics</Link>
        <Link className="compute-secondary-button" href={href("/observability?section=Events")}>Project activity</Link>
        <Link className="compute-secondary-button" href={href("/networking/firewalls")}>Firewall inventory</Link>
        <Link className="compute-secondary-button" href={href("/storage")}>Project volumes</Link>
      </nav>
      <section className="compute-card compute-detail-overview">
        <h2>Instance information</h2>
        <dl>
          <div><dt>Resource ID</dt><dd>{instance.id}</dd></div>
          <div><dt>Status</dt><dd>{instance.status}</dd></div>
          <div><dt>Region</dt><dd>{instance.region}</dd></div>
          <div><dt>Floating IP</dt><dd>{instance.public_ip || "Not assigned"}</dd></div><div><dt>Fixed IP</dt><dd>{instance.private_ip || "Not assigned"}</dd></div>
          <div><dt>Created</dt><dd>{new Date(instance.created_at).toLocaleString()}</dd></div>
          <div><dt>Updated</dt><dd>{new Date(instance.updated_at).toLocaleString()}</dd></div>
          <div><dt>Image reference</dt><dd>{instance.image_id || "Not reported"}</dd></div>
          <div><dt>Size reference</dt><dd>{instance.plan_id || "Not reported"}</dd></div>
          <div><dt>Network reference</dt><dd>{instance.network_id || "Not reported"}</dd></div>
          <div><dt>SSH key name</dt><dd>{instance.ssh_key_ids.join(", ") || "Not reported"}</dd></div>
        </dl>
      </section>
      <section className="compute-card compute-detail-overview">
        <h2>Not yet connected</h2>
        <p>These operations are not available through the current instance API. No inactive tabs or simulated controls are shown.</p>
        <ul>{cloudServices.find((service) => service.id === "instances")?.pending.map((item) => <li key={item}>{item}</li>)}</ul>
        <p>Power and provisioning information are currently supplied as one lifecycle status, not independently measured signals.</p>
      </section>
    </div>
  );
}
