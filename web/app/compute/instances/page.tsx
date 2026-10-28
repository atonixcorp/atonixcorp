"use client";

import Link from "next/link";
import { useCallback, useEffect, useMemo, useState } from "react";
import { getCatalog, getInstances } from "../lib/api";
import type { ComputeInstance, ComputePlan } from "../lib/types";

export default function ComputeInstancesPage() {
  const [instances, setInstances] = useState<ComputeInstance[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [filter, setFilter] = useState("");
  const [plans, setPlans] = useState<ComputePlan[]>([]);
  const [planCatalogError, setPlanCatalogError] = useState("");
  const [planCatalogLoading, setPlanCatalogLoading] = useState(true);

  const loadInstances = useCallback(async () => {
    setLoading(true);
    setError("");
    try {
      setInstances(await getInstances());
    } catch (loadError) {
      setError(loadError instanceof Error ? loadError.message : "Could not load instances.");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    const timeout = window.setTimeout(() => { void loadInstances(); }, 0);
    return () => window.clearTimeout(timeout);
  }, [loadInstances]);

  useEffect(() => {
    let active = true;
    void getCatalog().then((catalog) => {
      if (active) setPlans(catalog.plans);
    }).catch((catalogError: unknown) => {
      if (active) {
        setPlanCatalogError(catalogError instanceof Error
          ? catalogError.message
          : "Plan details are unavailable.");
      }
    }).finally(() => {
      if (active) setPlanCatalogLoading(false);
    });
    return () => { active = false; };
  }, []);

  const filteredInstances = useMemo(() => {
    const query = filter.trim().toLowerCase();
    return query
      ? instances.filter((instance) =>
        `${instance.name} ${instance.label} ${instance.region} ${instance.status}`
          .toLowerCase().includes(query))
      : instances;
  }, [filter, instances]);

  return (
    <div className="compute-page">
      <header className="compute-page-heading">
        <div>
          <p className="compute-eyebrow">COMPUTE / INSTANCES</p>
          <h1>Compute instances</h1>
          <p>Manage virtual machines and review their current lifecycle status.</p>
        </div>
        <Link className="compute-primary-button" href="/compute/deploy">＋ Deploy instance</Link>
      </header>

      <section className="compute-card compute-instance-card" aria-label="Compute instances">
        <header className="compute-table-toolbar">
          <div><h2>Instances</h2><p>{loading ? "Loading instances…" : `${instances.length} total`}</p></div>
          <div className="compute-table-actions">
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
            <p>Choose a compute plan and operating system to prepare your first instance.</p>
            <Link className="compute-primary-button" href="/compute/deploy">＋ Deploy your first instance</Link>
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
      <td>{instance.region.toUpperCase()}</td>
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
