"use client";

import { useState } from "react";
import type { ReactNode } from "react";
import type {
  ComputeCatalog,
  ComputeImage,
  ComputePlan,
  ComputeSSHKey,
  DeploymentDraft,
} from "../lib/types";

export function ComputeTypeSection({
  catalog, draft, update,
}: { catalog: ComputeCatalog; draft: DeploymentDraft; update: (patch: Partial<DeploymentDraft>) => void }) {
  return (
    <Section id="compute-type" number="01" title="Compute type" description="Choose the performance profile that fits your workload.">
      <div className="compute-choice-grid">
        {catalog.compute_types.map((type) => (
          <button
            type="button"
            className={`compute-choice-card${draft.computeType === type.id ? " selected" : ""}`}
            aria-pressed={draft.computeType === type.id}
            key={type.id}
            onClick={() => update({ computeType: type.id, plan: "" })}
          >
            <span className="compute-choice-mark" aria-hidden="true">{draft.computeType === type.id ? "✓" : "○"}</span>
            <strong>{type.name}</strong>
            <small>{type.description}</small>
          </button>
        ))}
      </div>
      {catalog.compute_types.length === 0 && <InlineEmpty>Compute types are not available.</InlineEmpty>}
    </Section>
  );
}

export function RegionSection({
  catalog, draft, update,
}: { catalog: ComputeCatalog; draft: DeploymentDraft; update: (patch: Partial<DeploymentDraft>) => void }) {
  const grouped = catalog.regions.reduce<Record<string, typeof catalog.regions>>((groups, region) => {
    groups[region.country] = [...(groups[region.country] ?? []), region];
    return groups;
  }, {});
  return (
    <Section id="region" number="02" title="Region" description="Pick where your instance and its data will be located.">
      {Object.entries(grouped).map(([country, regions]) => (
        <div className="compute-region-group" key={country}>
          <h3>{country}</h3>
          <div className="compute-region-grid">
            {regions.map((region) => (
              <button
                type="button"
                key={region.id}
                disabled={!region.available}
                className={`compute-region-card${draft.region === region.id ? " selected" : ""}`}
                aria-pressed={draft.region === region.id}
                onClick={() => update({ region: region.id, plan: "", network: "" })}
              >
                <span className="compute-choice-mark" aria-hidden="true">{draft.region === region.id ? "✓" : "○"}</span>
                <strong>{region.name}</strong>
                <code>{region.id.toUpperCase()}</code>
                <small>{region.available ? "Available capacity" : region.capacity_status.replaceAll("_", " ")}</small>
              </button>
            ))}
          </div>
        </div>
      ))}
    </Section>
  );
}

export function ImageSection({
  images, draft, update,
}: { images: ComputeImage[]; draft: DeploymentDraft; update: (patch: Partial<DeploymentDraft>) => void }) {
  const [source, setSource] = useState("operating_system");
  const sources = [
    { id: "operating_system", label: "Operating systems" },
    { id: "marketplace", label: "Marketplace" },
    { id: "snapshot", label: "Snapshots" },
    { id: "custom", label: "Custom images" },
  ];
  const options = images.filter((image) => image.source === source && image.available);
  return (
    <Section id="image" number="03" title="Image" description="Select the operating system installed on your instance.">
      <div className="compute-source-tabs" role="tablist" aria-label="Image source">
        {sources.map((item) => (
          <button type="button" role="tab" aria-selected={source === item.id} key={item.id} onClick={() => setSource(item.id)}>
            {item.label}
          </button>
        ))}
      </div>
      {options.length ? (
        <div className="compute-image-grid">
          {options.map((image) => (
            <button type="button" key={image.id} className={`compute-image-card${draft.image === image.id ? " selected" : ""}`} aria-pressed={draft.image === image.id} onClick={() => update({ image: image.id })}>
              <span className="compute-os-mark">{image.family.slice(0, 1)}</span>
              <span><strong>{image.name}</strong><small>{image.version}</small></span>
              <span className="compute-choice-mark" aria-hidden="true">{draft.image === image.id ? "✓" : "○"}</span>
            </button>
          ))}
        </div>
      ) : (
        <InlineEmpty>No images are currently available from this source. Select an available source or retry the catalog.</InlineEmpty>
      )}
    </Section>
  );
}

export function PlanSection({
  plans, draft, update, currency,
}: { plans: ComputePlan[]; draft: DeploymentDraft; update: (patch: Partial<DeploymentDraft>) => void; currency: string }) {
  const families = Array.from(new Set(plans.map((plan) => plan.family)));
  const [family, setFamily] = useState("");
  const selectedFamily = families.includes(family) ? family : (families[0] ?? "");
  const visiblePlans = plans.filter((plan) => plan.family === selectedFamily && plan.available);
  return (
    <Section id="plan" number="04" title="Instance size" description="Plan specifications and rates come from the Compute API.">
      {families.length > 1 && (
        <div className="compute-source-tabs" role="tablist" aria-label="Plan family">
          {families.map((item) => (
            <button type="button" role="tab" aria-selected={selectedFamily === item} key={item} onClick={() => setFamily(item)}>
              {familyName(item)}
            </button>
          ))}
        </div>
      )}
      {visiblePlans.length ? (
        <div className="compute-table-scroll">
          <table className="compute-table compute-plan-table">
            <thead><tr><th>Plan</th><th>vCPU</th><th>Memory</th><th>NVMe</th><th>Transfer</th><th>Monthly estimate</th></tr></thead>
            <tbody>{visiblePlans.map((plan) => (
              <tr key={plan.id} className={draft.plan === plan.id ? "selected" : ""}>
                <td><button className="compute-plan-select" type="button" aria-pressed={draft.plan === plan.id} onClick={() => update({ plan: plan.id })}>{plan.name}</button></td>
                <td>{plan.vcpus}{plan.gpu_count ? ` · ${plan.gpu_count} GPU` : ""}</td>
                <td>{Math.round(plan.memory_mb / 1024)} GB</td>
                <td>{plan.boot_disk_gb} GB</td>
                <td>{plan.transfer_tb} TB</td>
                <td>{formatMoney(plan.monthly_price, plan.currency || currency)}<small className="compute-plan-hourly">{formatMoney(plan.hourly_price, plan.currency || currency)} / hour</small></td>
              </tr>
            ))}</tbody>
          </table>
        </div>
      ) : <InlineEmpty>No instance sizes are currently available for this selection.</InlineEmpty>}
    </Section>
  );
}

export function StorageSection({
  plan, draft, update,
}: { plan?: ComputePlan; draft: DeploymentDraft; update: (patch: Partial<DeploymentDraft>) => void }) {
  const addVolume = () => update({
    volumes: [...draft.volumes, { name: `data-${draft.volumes.length + 1}`, type: "nvme_ssd", size_gb: 100 }],
  });
  return (
    <Section id="storage" number="05" title="Storage" description="The boot disk is included with your selected plan.">
      <div className="compute-boot-disk"><span>Boot disk</span><strong>{plan ? `${plan.boot_disk_gb} GB NVMe SSD` : "Select an instance size"}</strong></div>
      <div className="compute-volume-heading"><div><strong>Additional block volumes</strong><small>Volumes have an independent lifecycle.</small></div><button type="button" className="compute-secondary-button" onClick={addVolume}>＋ Add volume</button></div>
      {draft.volumes.map((volume, index) => (
        <div className="compute-volume-row" key={`${index}-${volume.name}`}>
          <label>Volume name<input value={volume.name} maxLength={63} onChange={(event) => replaceVolume(draft, update, index, { name: event.target.value })} /></label>
          <label>Type<select value={volume.type} onChange={(event) => replaceVolume(draft, update, index, { type: event.target.value as "nvme_ssd" })}><option value="nvme_ssd">NVMe SSD</option></select></label>
          <label>Capacity (GB)<input type="number" min={1} max={10000} value={volume.size_gb} onChange={(event) => replaceVolume(draft, update, index, { size_gb: Number(event.target.value) })} /></label>
          <button type="button" className="compute-remove-button" aria-label={`Remove ${volume.name} volume`} onClick={() => update({ volumes: draft.volumes.filter((_, itemIndex) => itemIndex !== index) })}>Remove</button>
        </div>
      ))}
    </Section>
  );
}

export function NetworkSection({
  catalog, draft, update,
}: { catalog: ComputeCatalog; draft: DeploymentDraft; update: (patch: Partial<DeploymentDraft>) => void }) {
  const networks = catalog.networks.filter((item) => item.region === draft.region && item.available);
  return (
    <Section id="network" number="06" title="Networking" description="Attach to an existing network and choose public connectivity.">
      <label className="compute-field">Virtual private cloud
        <select value={draft.network} onChange={(event) => update({ network: event.target.value })}>
          <option value="">No VPC selected</option>
          {networks.map((network) => <option value={network.id} key={network.id}>{network.name} · {network.id}</option>)}
        </select>
      </label>
      <div className="compute-toggle-grid">
        <Toggle label="Public IPv4" description="Assign a public IPv4 address." checked={draft.publicIPv4} onChange={(checked) => update({ publicIPv4: checked })} />
        <Toggle label="Public IPv6" description="Enable public IPv6 connectivity." checked={draft.publicIPv6} onChange={(checked) => update({ publicIPv6: checked })} />
        <Toggle label="Private networking" description="Connect to the selected private network." checked={draft.privateNetworking} onChange={(checked) => update({ privateNetworking: checked })} />
        <Toggle label="Hostname DNS" description="Publish a DNS record for this hostname." checked={draft.hostnameDNS} onChange={(checked) => update({ hostnameDNS: checked })} />
      </div>
    </Section>
  );
}

export function SecuritySection({
  keys, keyError, draft, update, onRetry,
}: { keys: ComputeSSHKey[]; keyError: string; draft: DeploymentDraft; update: (patch: Partial<DeploymentDraft>) => void; onRetry: () => void }) {
  const toggle = (id: string) => update({
    sshKeyIds: draft.sshKeyIds.includes(id)
      ? draft.sshKeyIds.filter((key) => key !== id)
      : [...draft.sshKeyIds, id],
  });
  return (
    <Section id="security" number="07" title="Security" description="SSH key authentication is recommended. Password provisioning is not enabled.">
      <div className="compute-security-heading"><strong>SSH keys</strong><span>Required</span></div>
      {keyError ? (
        <div className="compute-inline-error" role="alert"><p>{keyError}</p><button type="button" className="compute-secondary-button" onClick={onRetry}>Retry</button></div>
      ) : keys.length ? (
        <div className="compute-check-list">{keys.map((key) => (
          <label className="compute-check-row" key={key.id}>
            <input type="checkbox" checked={draft.sshKeyIds.includes(key.id)} onChange={() => toggle(key.id)} />
            <span><strong>{key.name}</strong><small>{key.key_type} · {key.fingerprint}</small></span>
          </label>
        ))}</div>
      ) : (
        <InlineEmpty>Add an SSH public key to your account before deploying an instance.</InlineEmpty>
      )}
      <label className="compute-field compute-firewall-field">Firewall group
        <select value={draft.firewall} onChange={(event) => update({ firewall: event.target.value })}>
          <option value="">No firewall group</option>
        </select>
        <small>Firewall inventory will appear once account security is connected.</small>
      </label>
    </Section>
  );
}

export function InitializationSection({
  draft, update,
}: { draft: DeploymentDraft; update: (patch: Partial<DeploymentDraft>) => void }) {
  return (
    <Section id="initialization" number="08" title="Initialization" description="Optionally provide cloud-init or user data to run during setup.">
      <label className="compute-field">Cloud-init / user data
        <textarea value={draft.userData} maxLength={65536} rows={7} placeholder="#cloud-config" onChange={(event) => update({ userData: event.target.value })} />
        <small>{draft.userData.length.toLocaleString()} / 65,536 characters · Templates can be added later.</small>
      </label>
    </Section>
  );
}

export function IdentitySection({
  draft, update,
}: { draft: DeploymentDraft; update: (patch: Partial<DeploymentDraft>) => void }) {
  const [tagInput, setTagInput] = useState("");
  const addTag = () => {
    const tag = tagInput.trim();
    if (tag && !draft.tags.includes(tag) && draft.tags.length < 20) update({ tags: [...draft.tags, tag] });
    setTagInput("");
  };
  return (
    <Section id="identity" number="09" title="Server identity" description="Set a hostname, a readable label, and the number of instances.">
      <div className="compute-form-grid">
        <label className="compute-field">Hostname<input value={draft.hostname} maxLength={63} placeholder="worker-01" onChange={(event) => update({ hostname: event.target.value })} /></label>
        <label className="compute-field">Server label<input value={draft.label} maxLength={128} placeholder="Production worker" onChange={(event) => update({ label: event.target.value })} /></label>
        <label className="compute-field compute-quantity-field">Number of instances
          <span className="compute-quantity-control">
            <button type="button" aria-label="Decrease instance quantity" disabled={draft.quantity <= 1} onClick={() => update({ quantity: draft.quantity - 1 })}>−</button>
            <input aria-label="Instance quantity" type="number" min={1} max={50} value={draft.quantity} onChange={(event) => update({ quantity: Number(event.target.value) })} />
            <button type="button" aria-label="Increase instance quantity" disabled={draft.quantity >= 50} onClick={() => update({ quantity: draft.quantity + 1 })}>＋</button>
          </span>
        </label>
        <label className="compute-field">Base name<input value={draft.name} maxLength={63} placeholder="worker" onChange={(event) => update({ name: event.target.value })} /><small>{draft.quantity > 1 ? `Names: ${draft.name || "worker"}-01 … ${draft.name || "worker"}-${String(draft.quantity).padStart(2, "0")}` : "Choose the instance name."}</small></label>
      </div>
      <label className="compute-field">Tags
        <span className="compute-tag-entry"><input value={tagInput} maxLength={63} placeholder="production" onChange={(event) => setTagInput(event.target.value)} onKeyDown={(event) => { if (event.key === "Enter") { event.preventDefault(); addTag(); } }} /><button type="button" className="compute-secondary-button" onClick={addTag}>Add tag</button></span>
      </label>
      <div className="compute-tag-list">{draft.tags.map((tag) => <span key={tag}>{tag}<button type="button" aria-label={`Remove ${tag} tag`} onClick={() => update({ tags: draft.tags.filter((item) => item !== tag) })}>×</button></span>)}</div>
    </Section>
  );
}

export function ReviewSection({
  catalog, draft, plan, onEdit, onDeploy, deployBlocked, submitting,
}: {
  catalog: ComputeCatalog;
  draft: DeploymentDraft;
  plan?: ComputePlan;
  onEdit: (id: string) => void;
  onDeploy: () => void;
  deployBlocked: boolean;
  submitting: boolean;
}) {
  const entries = [
    { id: "compute-type", title: "Compute", value: `${catalog.compute_types.find((item) => item.id === draft.computeType)?.name ?? "Not selected"}${plan ? ` · ${plan.vcpus} vCPU · ${Math.round(plan.memory_mb / 1024)} GB RAM` : ""}` },
    { id: "region", title: "Location", value: catalog.regions.find((item) => item.id === draft.region)?.name ?? "Not selected" },
    { id: "image", title: "Image", value: catalog.images.find((item) => item.id === draft.image)?.name ?? "Not selected" },
    { id: "storage", title: "Storage", value: `${plan?.boot_disk_gb ?? "—"} GB boot disk · ${draft.volumes.reduce((sum, item) => sum + item.size_gb, 0)} GB attached` },
    { id: "network", title: "Network", value: `${draft.network || "No VPC"} · ${draft.publicIPv4 ? "IPv4 " : ""}${draft.publicIPv6 ? "IPv6" : ""}` },
    { id: "security", title: "Authentication", value: `${draft.sshKeyIds.length} SSH key(s)` },
    { id: "security", title: "Firewall", value: draft.firewall || "None selected" },
    { id: "initialization", title: "Initialization", value: draft.userData ? "User data configured" : "None" },
    { id: "identity", title: "Identity", value: `${draft.hostname || draft.name || "Hostname"} · ${draft.label || "No label"}` },
    { id: "identity", title: "Quantity", value: String(draft.quantity) },
  ];
  return (
    <Section id="review" number="10" title="Review & deploy" description="Check the configuration before continuing to account authorization.">
      <div className="compute-review-list">{entries.map((item, index) => (
        <div className="compute-review-row" key={`${item.title}-${index}`}>
          <span><small>{item.title}</small><strong>{item.value}</strong></span>
          <button type="button" onClick={() => onEdit(item.id)}>Edit</button>
        </div>
      ))}</div>
      <div className="compute-review-actions">
        <p>Live deployment remains disabled until authentication, tenant authorization, and authoritative billing are configured.</p>
        <button type="button" className="compute-primary-button" disabled={deployBlocked} onClick={onDeploy}>
          {submitting ? "Submitting…" : "Deploy instance"}
        </button>
      </div>
    </Section>
  );
}

export function Section({
  id, number, title, description, children,
}: { id: string; number: string; title: string; description: string; children: ReactNode }) {
  return (
    <section className="compute-card compute-deploy-section" id={id} tabIndex={-1}>
      <header className="compute-section-heading">
        <span>{number}</span><div><h2>{title}</h2><p>{description}</p></div>
      </header>
      <div className="compute-section-body">{children}</div>
    </section>
  );
}

function Toggle({ label, description, checked, onChange }: { label: string; description: string; checked: boolean; onChange: (value: boolean) => void }) {
  return (
    <label className="compute-toggle">
      <span><strong>{label}</strong><small>{description}</small></span>
      <input type="checkbox" checked={checked} onChange={(event) => onChange(event.target.checked)} />
    </label>
  );
}

function InlineEmpty({ children }: { children: ReactNode }) {
  return <div className="compute-inline-empty">{children}</div>;
}

function familyName(family: string): string {
  return family.split(/[_-]/).map((part) => part.charAt(0).toUpperCase() + part.slice(1)).join(" ");
}

export function formatMoney(value: number, currency: string): string {
  try {
    return new Intl.NumberFormat(undefined, { style: "currency", currency, maximumFractionDigits: 2 }).format(value);
  } catch {
    return `${currency} ${value.toFixed(2)}`;
  }
}

function replaceVolume(
  draft: DeploymentDraft,
  update: (patch: Partial<DeploymentDraft>) => void,
  index: number,
  patch: Partial<DeploymentDraft["volumes"][number]>,
) {
  update({ volumes: draft.volumes.map((volume, itemIndex) => itemIndex === index ? { ...volume, ...patch } : volume) });
}
