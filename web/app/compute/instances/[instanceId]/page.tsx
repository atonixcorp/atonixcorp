"use client";

import Link from "next/link";
import { useParams, useRouter } from "next/navigation";
import { useEffect, useRef, useState } from "react";
import { deleteInstance, getInstance, operateInstance } from "../../lib/api";
import type { ComputeInstance } from "../../lib/types";

export default function ComputeInstanceDetailsPage() {
  const { instanceId } = useParams<{ instanceId: string }>();
  const router = useRouter();
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
      router.push("/compute/instances");
    } catch (requestError) {
      setError(requestError instanceof Error ? requestError.message : "Could not delete this instance.");
      setOperation("");
    }
  };

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
        <div><p className="compute-eyebrow"><Link href="/compute/instances">COMPUTE / INSTANCES</Link> / {instance.id}</p><h1>{instance.name}</h1><p>{instance.region ? "Primary region" : "Not assigned"}</p></div>
        <div className="compute-detail-actions">
          <span className={`compute-instance-status ${instance.status}`}>{instance.status}</span>
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
      <div className="compute-detail-tabs" role="tablist" aria-label="Instance sections">
        {["Overview", "Metrics", "Networking", "Storage", "Snapshots", "Backups", "Firewall", "Console", "Activity", "Settings"].map((tab) => <button type="button" role="tab" aria-selected={tab === "Overview"} key={tab}>{tab}</button>)}
      </div>
      <section className="compute-card compute-detail-overview">
        <h2>Instance information</h2>
        <dl>
          <div><dt>Resource ID</dt><dd>{instance.id}</dd></div>
          <div><dt>Status</dt><dd>{instance.status}</dd></div>
          <div><dt>Region</dt><dd>{instance.region}</dd></div>
          <div><dt>Floating IP</dt><dd>{instance.public_ip || "Not assigned"}</dd></div><div><dt>Fixed IP</dt><dd>{instance.private_ip || "Not assigned"}</dd></div>
          <div><dt>Created</dt><dd>{new Date(instance.created_at).toLocaleString()}</dd></div>
        </dl>
      </section>
    </div>
  );
}
