"use client";

import { useEffect, useState, type FormEvent } from "react";
import Link from "next/link";
import { getProjects, getTenants, type ComputeProject, type ComputeTenant } from "@/app/compute/lib/api";
import { CloudShell } from "@/app/shared/ui/CloudShell";
import { SecurityGroups } from "./SecurityGroups";
import styles from "./SecurityConsole.module.css";

type SecurityCapabilities = {
  "security.policy.read"?: boolean;
  "security.policy.create"?: boolean;
  "security.policy.simulate"?: boolean;
  "security.policy.approve"?: boolean;
  "security.policy.deploy"?: boolean;
  "security.policy.rollback"?: boolean;
  "security.audit.read"?: boolean;
};

type SecurityOverview = SecurityCapabilities & {
  policy_drafts: number;
  policy_simulated: number;
  telemetry_state: string;
  telemetry_freshness: string | null;
  data_classification: string;
  masking_status: string;
  source_quality: string;
};

type PolicyRule = { field: string; operator: string; value: string; effect: string };
type SecurityPolicy = {
  id: number;
  name: string;
  type: string;
  description: string;
  rules: PolicyRule[];
  state: string;
  version: number;
  created_at: string;
  updated_at: string;
  data_classification: string;
  masking_status: string;
  source_quality: string;
};
type PolicyPage = SecurityCapabilities & { items: SecurityPolicy[]; next_cursor?: string; limit: number };
type AuditEntry = {
  id: number;
  actor_id: number;
  action: string;
  resource_type: string;
  resource_id: string;
  result: string;
  reason: string;
  created_at: string;
};
type AuditPage = { items: AuditEntry[]; next_cursor?: string; limit: number };
type Tab = "Overview" | "Policies" | "Network security" | "Audit";
type ScopedSecurityData = {
  key: string;
  overview: SecurityOverview | null;
  policyPage: PolicyPage | null;
  auditPage: AuditPage | null;
  overviewError: string;
  policyError: string;
  auditError: string;
};
type RuleField = { value: string; label: string; operators: string[] };
type PolicyDefinition = { value: string; label: string; fields: RuleField[] };

const policyDefinitions: PolicyDefinition[] = [
  { value: "identity", label: "Identity", fields: [
    { value: "identity.role", label: "Identity role", operators: ["equals", "not_equals", "in", "not_in"] },
    { value: "identity.mfa_required", label: "MFA required", operators: ["equals", "not_equals"] },
  ] },
  { value: "network", label: "Network", fields: [
    { value: "network.source_cidr", label: "Source CIDR", operators: ["equals", "not_equals", "in", "not_in"] },
    { value: "network.destination_port", label: "Destination port", operators: ["equals", "not_equals", "in", "not_in", "greater_than", "greater_than_or_equal", "less_than", "less_than_or_equal"] },
  ] },
  { value: "workload", label: "Workload", fields: [
    { value: "workload.image_digest", label: "Image digest", operators: ["equals", "not_equals", "in", "not_in"] },
    { value: "workload.image_signed", label: "Image signed", operators: ["equals", "not_equals"] },
  ] },
  { value: "data", label: "Data", fields: [
    { value: "data.encryption_enabled", label: "Encryption enabled", operators: ["equals", "not_equals"] },
    { value: "data.classification", label: "Data classification", operators: ["equals", "not_equals", "in", "not_in"] },
  ] },
  { value: "threat_detection", label: "Threat detection", fields: [
    { value: "event.severity", label: "Event severity", operators: ["equals", "not_equals", "in", "not_in"] },
    { value: "event.action", label: "Event action", operators: ["equals", "not_equals", "in", "not_in"] },
  ] },
  { value: "vulnerability", label: "Vulnerability", fields: [
    { value: "vulnerability.cvss", label: "CVSS score", operators: ["equals", "not_equals", "greater_than", "greater_than_or_equal", "less_than", "less_than_or_equal"] },
    { value: "vulnerability.fix_available", label: "Fix available", operators: ["equals", "not_equals"] },
  ] },
  { value: "compliance", label: "Compliance", fields: [
    { value: "compliance.control", label: "Control", operators: ["equals", "not_equals", "in", "not_in"] },
    { value: "compliance.framework", label: "Framework", operators: ["equals", "not_equals"] },
  ] },
];

const effects = ["alert", "require_review", "allow", "deny"];
const tabs: Tab[] = ["Overview", "Policies", "Network security", "Audit"];

async function requestJSON<T>(path: string, init?: RequestInit): Promise<T> {
  const response = await fetch(path, {
    ...init,
    cache: "no-store",
    headers: {
      ...(init?.body ? { "Content-Type": "application/json" } : {}),
      ...init?.headers,
    },
  });
  const payload = await response.json().catch(() => null) as unknown;
  if (!response.ok) {
    const message = payload && typeof payload === "object" && "error" in payload
      ? String(payload.error)
      : `Security API request failed (${response.status}).`;
    throw new Error(message);
  }
  if (payload === null) throw new Error("The Security API returned an empty response.");
  return payload as T;
}

function errorMessage(error: unknown): string {
  return error instanceof Error ? error.message : "The request could not be completed.";
}

function errorStatus(error: unknown): number | null {
  if (!error || typeof error !== "object" || !("status" in error)) return null;
  return typeof error.status === "number" ? error.status : null;
}

function readable(value: string): string {
  return value.replaceAll("_", " ").replace(/\b\w/g, (letter) => letter.toUpperCase());
}

function dateLabel(value: string): string {
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? value : date.toLocaleString();
}

export function SecurityConsole() {
  const [tenants, setTenants] = useState<ComputeTenant[]>([]);
  const [projects, setProjects] = useState<ComputeProject[]>([]);
  const [tenantID, setTenantID] = useState("");
  const [projectID, setProjectID] = useState("");
  const [scopeLoading, setScopeLoading] = useState(true);
  const [projectsTenantID, setProjectsTenantID] = useState("");
  const [scopeError, setScopeError] = useState("");
  const [scopeErrorStatus, setScopeErrorStatus] = useState<number | null>(null);
  const [tab, setTab] = useState<Tab>("Overview");
  const [securityData, setSecurityData] = useState<ScopedSecurityData>({
    key: "", overview: null, policyPage: null, auditPage: null,
    overviewError: "", policyError: "", auditError: "",
  });
  const [reloadKey, setReloadKey] = useState(0);
  const [createOpen, setCreateOpen] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [createError, setCreateError] = useState("");
  const [notice, setNotice] = useState("");
  const [policyName, setPolicyName] = useState("");
  const [policyDescription, setPolicyDescription] = useState("");
  const [policyType, setPolicyType] = useState(policyDefinitions[0].value);
  const [ruleField, setRuleField] = useState(policyDefinitions[0].fields[0].value);
  const [ruleOperator, setRuleOperator] = useState("equals");
  const [ruleValue, setRuleValue] = useState("");
  const [ruleEffect, setRuleEffect] = useState(effects[0]);

  const definition = policyDefinitions.find((item) => item.value === policyType) ?? policyDefinitions[0];
  const selectedField = definition.fields.find((item) => item.value === ruleField) ?? definition.fields[0];
  const scopePath = tenantID && projectID
    ? `/api/security/tenants/${encodeURIComponent(tenantID)}/projects/${encodeURIComponent(projectID)}`
    : "";
  const requestKey = `${scopePath}:${reloadKey}`;
  const currentData = securityData.key === requestKey ? securityData : null;
  const overview = currentData?.overview ?? null;
  const policyPage = currentData?.policyPage ?? null;
  const auditPage = currentData?.auditPage ?? null;
  const overviewError = currentData?.overviewError ?? "";
  const policyError = currentData?.policyError ?? "";
  const auditError = currentData?.auditError ?? "";
  const projectLoading = Boolean(tenantID && projectsTenantID !== tenantID);
  const loading = Boolean(scopePath && !currentData);
  const canCreatePolicy = Boolean(overview?.["security.policy.create"] ?? policyPage?.["security.policy.create"]);

  useEffect(() => {
    let active = true;
    void getTenants().then((items) => {
      if (!active) return;
      setTenants(items);
      setScopeErrorStatus(null);
      setTenantID((current) => current && items.some((tenant) => String(tenant.id) === current)
        ? current
        : String(items[0]?.id ?? ""));
      setScopeLoading(false);
    }).catch((error: unknown) => {
      if (!active) return;
      setScopeError(errorMessage(error));
      setScopeErrorStatus(errorStatus(error));
      setScopeLoading(false);
    });
    return () => { active = false; };
  }, []);

  useEffect(() => {
    let active = true;
    if (!tenantID) return () => { active = false; };
    void getProjects(Number(tenantID)).then((items) => {
      if (!active) return;
      setProjects(items);
      setProjectID(String(items[0]?.id ?? ""));
      setProjectsTenantID(tenantID);
      setScopeErrorStatus(null);
    }).catch((error: unknown) => {
      if (active) {
        setScopeError(errorMessage(error));
        setScopeErrorStatus(errorStatus(error));
        setProjectsTenantID(tenantID);
      }
    });
    return () => { active = false; };
  }, [tenantID]);

  useEffect(() => {
    let active = true;
    if (!scopePath) return () => { active = false; };
    const auditQuery = new URLSearchParams({ limit: "50", since_hours: "24" });
    void Promise.allSettled([
      requestJSON<SecurityOverview>(`${scopePath}/overview`),
      requestJSON<PolicyPage>(`${scopePath}/policies?limit=50`),
      requestJSON<AuditPage>(`${scopePath}/audit?${auditQuery}`),
    ]).then(([overviewResult, policiesResult, auditResult]) => {
      if (!active) return;
      setSecurityData({
        key: requestKey,
        overview: overviewResult.status === "fulfilled" ? overviewResult.value : null,
        policyPage: policiesResult.status === "fulfilled" ? policiesResult.value : null,
        auditPage: auditResult.status === "fulfilled" ? auditResult.value : null,
        overviewError: overviewResult.status === "rejected" ? errorMessage(overviewResult.reason) : "",
        policyError: policiesResult.status === "rejected" ? errorMessage(policiesResult.reason) : "",
        auditError: auditResult.status === "rejected" ? errorMessage(auditResult.reason) : "",
      });
    });
    return () => { active = false; };
  }, [scopePath, requestKey]);

  async function createPolicy(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!scopePath || !canCreatePolicy) return;
    setSubmitting(true);
    setCreateError("");
    setNotice("");
    try {
      const policy = await requestJSON<SecurityPolicy>(`${scopePath}/policies`, {
        method: "POST",
        headers: { "Idempotency-Key": crypto.randomUUID() },
        body: JSON.stringify({
          name: policyName,
          type: policyType,
          description: policyDescription,
          rules: [{ field: ruleField, operator: ruleOperator, value: ruleValue, effect: ruleEffect }],
        }),
      });
      setNotice(`Policy draft “${policy.name}” was created.`);
      setPolicyName("");
      setPolicyDescription("");
      setRuleValue("");
      setCreateOpen(false);
      setTab("Policies");
      setReloadKey((key) => key + 1);
    } catch (error) {
      setCreateError(errorMessage(error));
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <CloudShell>
      <main className={styles.page}>
        <header className={styles.heading}>
          <div>
            <p className={styles.eyebrow}>ATONIXCORP CLOUD / SECURITY</p>
            <h1>Security</h1>
            <p>Scoped policy records, network groups, and audit events from the platform API.</p>
          </div>
          <button
            className={styles.refresh}
            type="button"
            onClick={() => scopePath ? setReloadKey((key) => key + 1) : window.location.reload()}
            disabled={loading || scopeLoading || projectLoading}
          >
            Refresh
          </button>
        </header>

        <section className={styles.scope} aria-label="Security workspace scope">
          <label>Workspace
            <select value={tenantID} onChange={(event) => { setScopeError(""); setProjects([]); setProjectID(""); setProjectsTenantID(""); setTenantID(event.target.value); }} disabled={scopeLoading || !tenants.length}>
              {tenants.map((tenant) => <option key={tenant.id} value={tenant.id}>{tenant.name}</option>)}
            </select>
          </label>
          <label>Project
            <select value={projectID} onChange={(event) => setProjectID(event.target.value)} disabled={scopeLoading || projectLoading || !projects.length}>
              {projects.map((project) => <option key={project.id} value={project.id}>{project.name}</option>)}
            </select>
          </label>
          <p>Data is restricted to the selected workspace and project.</p>
        </section>

        {scopeError && (
          <p className={styles.error} role="alert">
            {scopeErrorStatus === 401
              ? "Your session is missing or expired."
              : scopeErrorStatus === 403
                ? "Your account is authenticated but does not have permission for this workspace."
                : scopeError}
            {scopeErrorStatus === 401 && <> <Link href="/login?next=%2Fsecurity">Sign in</Link></>}
            {scopeErrorStatus !== 403 && <> Retry using the refresh control.</>}
          </p>
        )}
        {scopeLoading && <p className={styles.loading} role="status">Loading accessible workspaces…</p>}
        {!scopeLoading && !scopeError && tenants.length === 0 && <p className={styles.empty}>No accessible workspaces were returned for this account.</p>}
        {!scopeLoading && !projectLoading && !scopeError && tenants.length > 0 && projects.length === 0 && tenantID && <p className={styles.empty}>No projects are available in this workspace.</p>}

        <nav className={styles.tabs} role="tablist" aria-label="Security sections">
          {tabs.map((item, index) => (
            <button
              key={item}
              id={`security-tab-${index}`}
              type="button"
              role="tab"
              aria-selected={tab === item}
              aria-controls="security-tab-panel"
              tabIndex={tab === item ? 0 : -1}
              onClick={() => setTab(item)}
              onKeyDown={(event) => {
                if (!["ArrowRight", "ArrowLeft", "Home", "End"].includes(event.key)) return;
                event.preventDefault();
                const current = tabs.indexOf(item);
                const next = event.key === "Home" ? 0
                  : event.key === "End" ? tabs.length - 1
                    : (current + (event.key === "ArrowRight" ? 1 : -1) + tabs.length) % tabs.length;
                setTab(tabs[next]);
                document.getElementById(`security-tab-${next}`)?.focus();
              }}
            >
              {item}
            </button>
          ))}
        </nav>

        {scopePath && (
          <div className={styles.content} id="security-tab-panel" role="tabpanel" aria-labelledby={`security-tab-${tabs.indexOf(tab)}`}>
            {loading && <p className={styles.loading} role="status">Loading security data…</p>}
            {notice && <p className={styles.notice} role="status">{notice}</p>}

            {tab === "Overview" && (
              <>
                {overviewError ? <p className={styles.error} role="alert">Overview unavailable: {overviewError}</p> : overview && (
                  <>
                    <section className={styles.metrics} aria-label="Security API summary">
                      <Metric label="Policy drafts" value={String(overview.policy_drafts)} />
                      <Metric label="Telemetry" value={readable(overview.telemetry_state)} />
                      <Metric label="Data classification" value={readable(overview.data_classification)} />
                      <Metric label="Source quality" value={readable(overview.source_quality)} />
                    </section>
                    <section className={styles.status}>
                      <div><h2>Telemetry and policy capabilities</h2><p>Values below are returned by the Security API for the selected project.</p></div>
                      <dl>
                        <div><dt>Telemetry freshness</dt><dd>{overview.telemetry_freshness ? dateLabel(overview.telemetry_freshness) : "No live telemetry"}</dd></div>
                        <div><dt>Masking status</dt><dd>{readable(overview.masking_status)}</dd></div>
                        <div><dt>Policy simulation</dt><dd>Unavailable</dd></div>
                        <div><dt>Policy deployment</dt><dd>{overview["security.policy.deploy"] ? "Available" : "Unavailable"}</dd></div>
                      </dl>
                    </section>
                    <p className={styles.disclaimer}>Security findings, asset posture, and live telemetry are not exposed by the current API. No risk score or incident data is inferred.</p>
                  </>
                )}
              </>
            )}

            {tab === "Policies" && (
              <section className={styles.section} aria-labelledby="policies-title">
                <div className={styles.sectionHeading}>
                  <div><h2 id="policies-title">Security policies</h2><p>Draft records stored by the platform. Drafts are not enforced.</p></div>
                  {canCreatePolicy && <button className={styles.primary} type="button" onClick={() => { setCreateError(""); setCreateOpen((open) => !open); }}>{createOpen ? "Close form" : "Create draft"}</button>}
                </div>
                {policyError && <p className={styles.error} role="alert">Policies unavailable: {policyError}</p>}
                {!canCreatePolicy && !policyError && <p className={styles.muted}>Policy creation is not authorized for this project.</p>}
                {createOpen && canCreatePolicy && (
                  <form className={styles.createForm} onSubmit={createPolicy}>
                    <label>Policy name<input value={policyName} onChange={(event) => setPolicyName(event.target.value)} required minLength={1} maxLength={100} /></label>
                    <label>Type<select value={policyType} onChange={(event) => {
                      const next = policyDefinitions.find((item) => item.value === event.target.value) ?? policyDefinitions[0];
                      setPolicyType(next.value);
                      setRuleField(next.fields[0].value);
                      setRuleOperator(next.fields[0].operators[0]);
                    }}>{policyDefinitions.map((item) => <option key={item.value} value={item.value}>{item.label}</option>)}</select></label>
                    <label>Description<textarea value={policyDescription} onChange={(event) => setPolicyDescription(event.target.value)} maxLength={1000} rows={2} /></label>
                    <fieldset>
                      <legend>Rule</legend>
                      <label>Field<select value={ruleField} onChange={(event) => {
                        const next = definition.fields.find((item) => item.value === event.target.value) ?? definition.fields[0];
                        setRuleField(next.value);
                        setRuleOperator(next.operators[0]);
                      }}>{definition.fields.map((field) => <option key={field.value} value={field.value}>{field.label}</option>)}</select></label>
                      <label>Operator<select value={ruleOperator} onChange={(event) => setRuleOperator(event.target.value)}>{selectedField.operators.map((operator) => <option key={operator} value={operator}>{readable(operator)}</option>)}</select></label>
                      <label>Value<input value={ruleValue} onChange={(event) => setRuleValue(event.target.value)} required maxLength={256} /></label>
                      <label>Effect<select value={ruleEffect} onChange={(event) => setRuleEffect(event.target.value)}>{effects.map((effect) => <option key={effect} value={effect}>{readable(effect)}</option>)}</select></label>
                    </fieldset>
                    {createError && <p className={styles.error} role="alert">{createError}</p>}
                    <div className={styles.formFooter}>
                      <p>Saving creates a real platform draft. It does not activate or enforce the policy.</p>
                      <button className={styles.primary} type="submit" disabled={submitting}>{submitting ? "Creating…" : "Save draft"}</button>
                    </div>
                  </form>
                )}
                {policyPage?.items.length === 0 && !policyError && <p className={styles.empty}>No policy drafts exist in this project.</p>}
                {policyPage && policyPage.items.length > 0 && (
                  <div className={styles.tableWrap}>
                    <table><thead><tr><th>Name</th><th>Type</th><th>State</th><th>Rules</th><th>Updated</th></tr></thead>
                      <tbody>{policyPage.items.map((policy) => <tr key={policy.id}>
                        <td><strong>{policy.name}</strong><small>{policy.description || "No description"}</small></td>
                        <td>{readable(policy.type)}</td><td>{readable(policy.state)}</td><td>{policy.rules.length}</td><td>{dateLabel(policy.updated_at)}</td>
                      </tr>)}</tbody>
                    </table>
                  </div>
                )}
                <p className={styles.disclaimer}>Simulation is currently unavailable. Approval, deployment, and rollback are not implemented.</p>
              </section>
            )}

            {tab === "Network security" && <SecurityGroups tenantID={tenantID} projectID={projectID} />}

            {tab === "Audit" && (
              <section className={styles.section} aria-labelledby="audit-title">
                <div className={styles.sectionHeading}><div><h2 id="audit-title">Security audit</h2><p>Authorized audit events from the last 24 hours.</p></div></div>
                {auditError && <p className={styles.error} role="alert">Audit data unavailable: {auditError}</p>}
                {auditPage?.items.length === 0 && !auditError && <p className={styles.empty}>No audit events were returned for this time range.</p>}
                {auditPage && auditPage.items.length > 0 && (
                  <div className={styles.tableWrap}>
                    <table><thead><tr><th>Time</th><th>Action</th><th>Resource</th><th>Result</th><th>Actor ID</th></tr></thead>
                      <tbody>{auditPage.items.map((entry) => <tr key={entry.id}>
                        <td>{dateLabel(entry.created_at)}</td><td>{readable(entry.action)}</td><td>{entry.resource_type}{entry.resource_id ? ` · ${entry.resource_id}` : ""}</td><td>{readable(entry.result)}</td><td>{entry.actor_id}</td>
                      </tr>)}</tbody>
                    </table>
                  </div>
                )}
              </section>
            )}
          </div>
        )}
      </main>
    </CloudShell>
  );
}

function Metric({ label, value }: { label: string; value: string }) {
  return <article className={styles.metric}><span>{label}</span><strong>{value}</strong></article>;
}