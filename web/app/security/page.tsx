"use client";

import { useCallback, useEffect, useMemo, useRef, useState, type FormEvent } from "react";
import Link from "next/link";
import { CloudShell } from "@/app/shared/ui/CloudShell";
import "./security.css";

type PolicyType = "Identity" | "Network" | "Workload" | "Data" | "Threat detection" | "Vulnerability" | "Compliance";
type PolicyStatus = "Draft" | "Simulated";

type DemoPolicy = {
  id: string;
  name: string;
  type: PolicyType;
  ruleSummary: string;
  scope: string;
  mode: string;
  status: PolicyStatus;
  updatedAt: string;
};

type DemoEvent = {
  id: string;
  severity: "Critical" | "High" | "Medium" | "Low";
  detection: string;
  resource: string;
  source: string;
  status: string;
  owner: string;
  firstSeen: string;
};

type DemoState = {
  version: 1;
  policies: DemoPolicy[];
  events: DemoEvent[];
  activity: string[];
  assets: string[];
};

type UserSession = { user?: { email?: string } };

const STORAGE_KEY = "atonixcorp-security-demo-v1";
const policyTypes: PolicyType[] = [
  "Identity",
  "Network",
  "Workload",
  "Data",
  "Threat detection",
  "Vulnerability",
  "Compliance",
];
const modules = [
  "Overview",
  "Identity and access",
  "Policies",
  "Network security",
  "Workload protection",
  "Data protection",
  "Threat detection",
  "Vulnerabilities",
  "Logs and audit",
  "Compliance",
  "Incidents",
  "Integrations",
];

const sampleState: DemoState = {
  version: 1,
  policies: [
    {
      id: "demo-policy-1",
      name: "Require reviewed workload images",
      type: "Workload",
      ruleSummary: "Simulated rule: review workload image provenance.",
      scope: "Demo project · Development · Region A",
      mode: "Monitor (simulated)",
      status: "Simulated",
      updatedAt: "Demo sample",
    },
    {
      id: "demo-policy-2",
      name: "Review privileged identity access",
      type: "Identity",
      ruleSummary: "Simulated rule: review privileged identity sign-ins.",
      scope: "Demo project · Development · Region A",
      mode: "Simulation",
      status: "Draft",
      updatedAt: "Demo sample",
    },
  ],
  events: [
    {
      id: "demo-event-1",
      severity: "High",
      detection: "Unusual sign-in pattern",
      resource: "demo-identity-alex",
      source: "Demo identity sensor",
      status: "Needs review",
      owner: "Unassigned",
      firstSeen: "Demo sample",
    },
    {
      id: "demo-event-2",
      severity: "Medium",
      detection: "Workload image policy mismatch",
      resource: "demo-worker-a",
      source: "Demo workload sensor",
      status: "Investigating",
      owner: "Demo analyst",
      firstSeen: "Demo sample",
    },
    {
      id: "demo-event-3",
      severity: "Low",
      detection: "Certificate rotation reminder",
      resource: "demo-service-a",
      source: "Demo configuration sensor",
      status: "Open",
      owner: "Unassigned",
      firstSeen: "Demo sample",
    },
  ],
  activity: [
    "Demo analyst created a simulated workload policy",
    "Demo analyst reviewed a simulated identity finding",
    "Demo environment initialized with sample records",
  ],
  assets: ["demo-worker-a", "demo-service-a", "demo-identity-alex", "demo-vm-a"],
};
const emptyDemoState: DemoState = { version: 1, policies: [], events: [], activity: [], assets: [] };

function isDemoState(value: unknown): value is DemoState {
  if (!value || typeof value !== "object") return false;
  const record = value as Record<string, unknown>;
  return record.version === 1 &&
    Array.isArray(record.policies) && record.policies.length <= 100 && record.policies.every((item) => {
      if (!item || typeof item !== "object") return false;
      const policy = item as Record<string, unknown>;
      return typeof policy.id === "string" && policy.id.length <= 100 &&
        typeof policy.name === "string" && policy.name.length <= 100 &&
        policyTypes.includes(policy.type as PolicyType) &&
        typeof policy.ruleSummary === "string" && policy.ruleSummary.length <= 500 &&
        typeof policy.scope === "string" && policy.scope.length <= 300 &&
        typeof policy.mode === "string" && policy.mode.length <= 80 &&
        (policy.status === "Draft" || policy.status === "Simulated") &&
        typeof policy.updatedAt === "string" && policy.updatedAt.length <= 100;
    }) &&
    Array.isArray(record.events) && record.events.length <= 200 && record.events.every((item) => {
      if (!item || typeof item !== "object") return false;
      const event = item as Record<string, unknown>;
      return typeof event.id === "string" && event.id.length <= 100 &&
        ["Critical", "High", "Medium", "Low"].includes(String(event.severity)) &&
        ["detection", "resource", "source", "status", "owner", "firstSeen"]
          .every((field) => typeof event[field] === "string" && (event[field] as string).length <= 300);
    }) &&
    Array.isArray(record.activity) && record.activity.length <= 500 &&
    record.activity.every((item) => typeof item === "string" && item.length <= 500) &&
    Array.isArray(record.assets) && record.assets.length <= 500 &&
    record.assets.every((item) => typeof item === "string" && item.length <= 200);
}

function nowLabel(): string {
  return new Date().toLocaleString();
}

type SecuritySessionResult =
  | { status: "unauthenticated" | "unavailable" }
  | { status: "authenticated"; email: string; demo: DemoState | null; storageError: string };

async function loadSecuritySession(): Promise<SecuritySessionResult> {
  try {
    const response = await fetch("/api/auth/session", { cache: "no-store" });
    if (response.status === 401) return { status: "unauthenticated" };
    if (!response.ok) return { status: "unavailable" };
    const data = await response.json() as UserSession;
    let demo: DemoState | null = null;
    let storageError = "";
    try {
      const stored = window.localStorage.getItem(STORAGE_KEY);
      if (stored) {
        const parsed: unknown = JSON.parse(stored);
        if (isDemoState(parsed)) demo = parsed;
        else storageError = "Saved demo data has an unsupported format. Reset it to start a fresh demo.";
      }
    } catch {
      storageError = "Demo data could not be read from this browser. Check browser storage or reset the demo.";
    }
    return { status: "authenticated", email: data.user?.email ?? "", demo, storageError };
  } catch {
    return { status: "unavailable" };
  }
}

export default function SecurityPage() {
  const [sessionState, setSessionState] = useState<"checking" | "authenticated" | "unauthenticated" | "unavailable">("checking");
  const [userEmail, setUserEmail] = useState("");
  const [storageReady, setStorageReady] = useState(false);
  const [demo, setDemo] = useState<DemoState | null>(null);
  const [storageError, setStorageError] = useState("");
  const [selectedModule, setSelectedModule] = useState("Overview");
  const [project, setProject] = useState("Demo project");
  const [environment, setEnvironment] = useState("Development");
  const [region, setRegion] = useState("Region A");
  const [timeRange, setTimeRange] = useState("24 hours");
  const [severityFilter, setSeverityFilter] = useState("All severities");
  const [statusFilter, setStatusFilter] = useState("All statuses");
  const [showWizard, setShowWizard] = useState(false);
  const [wizardStep, setWizardStep] = useState(0);
  const [policyName, setPolicyName] = useState("");
  const [policyType, setPolicyType] = useState<PolicyType>("Identity");
  const [policyMode, setPolicyMode] = useState("Simulation");
  const [ruleText, setRuleText] = useState("");
  const [wizardError, setWizardError] = useState("");
  const [savedMessage, setSavedMessage] = useState("");
  const [lastUpdated, setLastUpdated] = useState("Not refreshed");
  const [selectedEvent, setSelectedEvent] = useState<DemoEvent | null>(null);
  const wizardRef = useRef<HTMLDialogElement>(null);
  const investigationRef = useRef<HTMLDialogElement>(null);

  useEffect(() => {
    let active = true;
    void loadSecuritySession().then((result) => {
      if (!active) return;
      if (result.status === "authenticated") {
        setUserEmail(result.email);
        setDemo(result.demo);
        setStorageError(result.storageError);
        setStorageReady(true);
        setLastUpdated(nowLabel());
      }
      setSessionState(result.status);
    });
    return () => { active = false; };
  }, []);

  useEffect(() => {
    const dialog = wizardRef.current;
    if (!dialog) return;
    if (showWizard && !dialog.open) dialog.showModal();
    else if (!showWizard && dialog.open) dialog.close();
  }, [showWizard]);

  useEffect(() => {
    const dialog = investigationRef.current;
    if (!dialog) return;
    if (selectedEvent && !dialog.open) dialog.showModal();
    else if (!selectedEvent && dialog.open) dialog.close();
  }, [selectedEvent]);

  const persist = useCallback((next: DemoState | null) => {
    try {
      if (next && !isDemoState(next)) {
        setStorageError("Demo record limits or format were exceeded. Reset the demo or remove older records.");
        return;
      }
      if (next) window.localStorage.setItem(STORAGE_KEY, JSON.stringify(next));
      else window.localStorage.removeItem(STORAGE_KEY);
      setDemo(next);
      setStorageError("");
      setSavedMessage("Saved in this browser only.");
    } catch {
      setStorageError("Could not save demo data in this browser. Check available storage and try again.");
    }
  }, []);

  const visibleEvents = useMemo(() => {
    if (!demo) return [];
    return demo.events.filter((event) =>
      (severityFilter === "All severities" || event.severity === severityFilter) &&
      (statusFilter === "All statuses" || event.status === statusFilter),
    );
  }, [demo, severityFilter, statusFilter]);

  function resetWizard() {
    setShowWizard(false);
    setWizardStep(0);
    setPolicyName("");
    setPolicyType("Identity");
    setPolicyMode("Simulation");
    setRuleText("");
    setWizardError("");
  }

  function createDraft(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!policyName.trim() || !ruleText.trim()) {
      setWizardError("Enter a policy name and at least one demo rule.");
      return;
    }

    const currentDemo = demo ?? emptyDemoState;
    if (currentDemo.policies.length >= 100) {
      setWizardError("This browser demo supports up to 100 policy records. Reset the demo to continue.");
      return;
    }

    const policy: DemoPolicy = {
      id: crypto.randomUUID(),
      name: policyName.trim(),
      type: policyType,
      ruleSummary: ruleText.trim(),
      scope: `${project} · ${environment} · ${region}`,
      mode: `${policyMode} (simulated)`,
      status: "Draft",
      updatedAt: nowLabel(),
    };
    persist({
      ...currentDemo,
      policies: [policy, ...currentDemo.policies],
      activity: [`Created browser-only demo policy “${policy.name}”`, ...currentDemo.activity].slice(0, 500),
    });
    setSelectedModule("Policies");
    resetWizard();
  }

  function simulatePolicy(policy: DemoPolicy) {
    if (!demo) return;
    const simulated = { ...policy, status: "Simulated" as const, updatedAt: nowLabel() };
    persist({
      ...demo,
      policies: demo.policies.map((item) => item.id === policy.id ? simulated : item),
      activity: [`Simulated policy “${policy.name}” in this browser`, ...demo.activity].slice(0, 500),
    });
  }

  function markEventReviewed(event: DemoEvent) {
    if (!demo) return;
    const updated = { ...event, status: "Reviewed", owner: "Demo analyst" };
    persist({
      ...demo,
      events: demo.events.map((item) => item.id === event.id ? updated : item),
      activity: [`Marked simulated event “${event.detection}” as reviewed`, ...demo.activity].slice(0, 500),
    });
    setSelectedEvent(null);
  }

  function renderPolicyWizard() {
    if (!showWizard) return null;
    const steps = ["Policy type", "Scope", "Rules", "Mode", "Review", "Save demo draft"];
    return (
      <dialog ref={wizardRef} className="security-wizard" aria-labelledby="security-wizard-title" onCancel={(event) => {
        event.preventDefault();
        resetWizard();
      }} onClick={(event) => {
        if (event.target === event.currentTarget) resetWizard();
      }}>
          <div className="security-wizard-header">
            <div>
              <p className="security-eyebrow">SIMULATED POLICY WORKFLOW</p>
              <h2 id="security-wizard-title">Create security policy</h2>
            </div>
            <button type="button" className="security-icon-button" onClick={resetWizard} aria-label="Close policy wizard">×</button>
          </div>
          <ol className="security-wizard-steps" aria-label="Policy wizard steps">
            {steps.map((step, index) => (
              <li key={step} aria-current={wizardStep === index ? "step" : undefined}>
                <span>{index + 1}</span><small>{step}</small>
              </li>
            ))}
          </ol>
          <form onSubmit={createDraft}>
            {wizardStep === 0 && (
              <div className="security-form-grid">
                <label>Policy type
                  <select value={policyType} onChange={(event) => {
                    const selected = policyTypes.find((type) => type === event.target.value);
                    if (selected) setPolicyType(selected);
                  }}>
                    {policyTypes.map((type) => <option key={type}>{type}</option>)}
                  </select>
                </label>
                <label>Policy name
                  <input value={policyName} onChange={(event) => setPolicyName(event.target.value)} maxLength={100} required />
                </label>
              </div>
            )}
            {wizardStep === 1 && (
              <div className="security-form-grid">
                <label>Demo project<input value={project} onChange={(event) => setProject(event.target.value)} maxLength={80} /></label>
                <label>Environment
                  <select value={environment} onChange={(event) => setEnvironment(event.target.value)}>
                    <option>Development</option><option>Staging</option><option>Production (simulated)</option>
                  </select>
                </label>
                <label>Region
                  <select value={region} onChange={(event) => setRegion(event.target.value)}>
                    <option>Region A</option><option>Region B</option><option>Region C</option>
                  </select>
                </label>
              </div>
            )}
            {wizardStep === 2 && (
              <label className="security-full-label">Demo rule description
                <textarea value={ruleText} onChange={(event) => setRuleText(event.target.value)} maxLength={500} required rows={4} placeholder="Describe the condition to explore in this simulated policy." />
                <small>This is descriptive demo text only. It is not sent to or evaluated by a policy engine.</small>
              </label>
            )}
            {wizardStep === 3 && (
              <div className="security-form-grid">
                <label>Demo enforcement mode
                  <select value={policyMode} onChange={(event) => setPolicyMode(event.target.value)}>
                    <option>Simulation</option><option>Monitor</option><option>Staged enforcement</option><option>Enforce</option>
                  </select>
                </label>
                <p className="security-notice">Every mode is simulated. This demo cannot deploy, enforce, or change infrastructure.</p>
              </div>
            )}
            {wizardStep === 4 && (
              <div className="security-review">
                <h3>Review simulated policy</h3>
                <dl>
                  <dt>Name</dt><dd>{policyName || "Unnamed policy"}</dd>
                  <dt>Type</dt><dd>{policyType}</dd>
                  <dt>Scope</dt><dd>{project} · {environment} · {region}</dd>
                  <dt>Mode</dt><dd>{policyMode} — simulated only</dd>
                  <dt>Rule description</dt><dd>{ruleText || "No rule description provided"}</dd>
                  <dt>Effect</dt><dd>Saved to browser storage as a draft. No service is contacted.</dd>
                </dl>
              </div>
            )}
            {wizardStep === 5 && (
              <div className="security-notice">
                <h3>Ready to save a demo draft</h3>
                <p>Saving creates a synthetic record in this browser. It does not submit for approval or deploy a policy.</p>
              </div>
            )}
            {wizardError && <p className="security-inline-error" role="alert">{wizardError}</p>}
            <div className="security-wizard-actions">
              <button type="button" className="security-button secondary" onClick={() => wizardStep === 0 ? resetWizard() : setWizardStep((step) => step - 1)}>
                {wizardStep === 0 ? "Cancel" : "Back"}
              </button>
              {wizardStep < steps.length - 1
                ? <button type="button" className="security-button primary" onClick={() => {
                  if (wizardStep === 0 && !policyName.trim()) {
                    setWizardError("Enter a policy name to continue.");
                    return;
                  }
                  if (wizardStep === 2 && !ruleText.trim()) {
                    setWizardError("Describe a demo rule to continue.");
                    return;
                  }
                  setWizardError("");
                  setWizardStep((step) => step + 1);
                }}>Continue</button>
                : <button type="submit" className="security-button primary">Save demo draft</button>}
            </div>
          </form>
      </dialog>
    );
  }

  if (sessionState === "checking" || (sessionState === "authenticated" && !storageReady)) {
    return <CloudShell><main className="security-loading" aria-live="polite"><div className="security-skeleton" /><p>Loading security demo…</p></main></CloudShell>;
  }
  if (sessionState === "unauthenticated") {
    return <CloudShell><main className="security-message"><h1>Sign in to continue</h1><p>Your platform session is required to open this workspace.</p><Link className="security-button primary" href="/login">Sign in</Link></main></CloudShell>;
  }
  if (sessionState === "unavailable") {
    return <CloudShell><main className="security-message" role="alert"><h1>Account service unavailable</h1><p>We could not verify your session. No security data was loaded.</p>        <button className="security-button secondary" onClick={() => window.location.reload()}>Retry</button></main></CloudShell>;
  }

  const hasData = Boolean(demo);
  const activeDemo = demo ?? emptyDemoState;
  const heading = selectedModule === "Overview" ? "Security" : selectedModule;

  return (
    <CloudShell>
      <main className="security-workspace">
        <div className="security-demo-banner" role="note">
          <strong>Interactive demo</strong>
          <span>Resources, findings, and actions are simulated and saved in this browser only. No real cloud infrastructure is provisioned, scanned, protected, or modified.</span>
          <button type="button" onClick={() => {
            persist(null);
            setSelectedModule("Overview");
          }}>Reset demo</button>
          <Link href="/platform">About the platform</Link>
        </div>

        <div className="security-workspace-heading">
          <div>
            <p className="security-eyebrow">ATONIXCORP CLOUD / SECURITY DEMO</p>
            <h1>{heading}</h1>
            <p>Monitor posture, threats, identities, policies, data protection, and compliance across a simulated scope.</p>
          </div>
          <div className="security-heading-actions">
            <Link href="/support">Documentation</Link>
            {hasData && <button className="security-button primary" onClick={() => setShowWizard(true)}>Create security policy</button>}
          </div>
        </div>

        <section className="security-scope-bar" aria-label="Demo scope and telemetry status">
          <label>Project
            <select value={project} onChange={(event) => setProject(event.target.value)}><option>Demo project</option><option>Demo research</option><option>Demo platform</option></select>
          </label>
          <label>Environment
            <select value={environment} onChange={(event) => setEnvironment(event.target.value)}><option>Development</option><option>Staging</option><option>Production (simulated)</option></select>
          </label>
          <label>Region
            <select value={region} onChange={(event) => setRegion(event.target.value)}><option>Region A</option><option>Region B</option><option>Region C</option></select>
          </label>
          <label>Time range
            <select value={timeRange} onChange={(event) => setTimeRange(event.target.value)}><option>24 hours</option><option>7 days</option><option>30 days</option></select>
          </label>
          <div className="security-freshness"><span className="security-status-dot" />Telemetry unavailable <small>Demo only · {lastUpdated}</small></div>
          <button type="button" className="security-refresh" onClick={() => {
            setLastUpdated(nowLabel());
            setSavedMessage("Demo refreshed. No live telemetry source is connected.");
          }}>Refresh demo</button>
        </section>

        <div className="security-layout">
          <nav className="security-sidebar" aria-label="Security workspace">
            <p>SECURITY WORKSPACE</p>
            {modules.map((module) => (
              <button
                key={module}
                type="button"
                className={selectedModule === module ? "active" : ""}
                aria-current={selectedModule === module ? "page" : undefined}
                onClick={() => setSelectedModule(module)}
              >
                <span className="security-nav-mark" aria-hidden="true">{module === "Overview" ? "○" : "·"}</span>{module}
                {module === "Policies" && demo && <small>{demo.policies.length}</small>}
              </button>
            ))}
            <div className="security-sidebar-user"><strong>{userEmail || "Signed-in user"}</strong><small>Demo workspace</small></div>
          </nav>

          <div className="security-content" aria-live="polite">
            {storageError && <div className="security-error" role="alert"><span>{storageError}</span><button onClick={() => persist(null)}>Reset demo storage</button></div>}
            {savedMessage && <p className="security-saved" role="status">{savedMessage}</p>}

            {selectedModule !== "Overview" && selectedModule !== "Policies" && (
              <section className="security-unavailable">
                <p className="security-eyebrow">MODULE NOT CONNECTED</p>
                <h2>{selectedModule}</h2>
                <p>This module is not connected to an authorized security service. No live assets, findings, or permissions are available here.</p>
                <p className="security-muted">This demo intentionally does not invent telemetry or imply protection.</p>
              </section>
            )}

            {selectedModule === "Overview" && !hasData && (
              <>
                <section className="security-metric-grid" aria-label="Security posture summary">
                  {["Overall risk", "Protected resources", "Critical findings", "Policy coverage", "Encryption coverage"].map((metric) => (
                    <article key={metric} className="security-metric-card">
                      <p>{metric}</p><strong>No data</strong><small>Telemetry unavailable · Demo only</small>
                    </article>
                  ))}
                </section>
                <section className="security-empty-state">
                  <span className="security-empty-icon" aria-hidden="true">◇</span>
                  <p className="security-eyebrow">BROWSER-ONLY DEMO</p>
                  <h2>Establish your demo security baseline</h2>
                  <p>Security policies define how identities, networks, workloads, data, and detections should be evaluated within a selected scope. Create a simulated policy to explore validation, simulation, and audit behavior.</p>
                  <div className="security-empty-actions">
                    <button className="security-button primary" onClick={() => setShowWizard(true)}>Create security policy</button>
                    <button className="security-button secondary" onClick={() => {
                      persist(sampleState);
                      setSavedMessage("Sample security records loaded in this browser. All values are simulated.");
                    }}>Load sample data</button>
                  </div>
                  <small>Sample data is synthetic and persists in this browser only.</small>
                </section>
              </>
            )}

            {(selectedModule === "Overview" || selectedModule === "Policies") && hasData && (
              <>
                <div className="security-demo-state"><strong>SIMULATED SAMPLE DATA</strong><span>Not connected to live cloud security services · Scope controls are local demo settings</span></div>
                {selectedModule === "Overview" && (
                  <>
                    <section className="security-metric-grid" aria-label="Simulated security summary">
                      {[
                        ["Overall risk", "Demo: Elevated", `Synthetic sample · ${timeRange}`],
                        ["Protected resources", `${activeDemo.assets.length} simulated`, "Not real infrastructure"],
                        ["Critical findings", "No sample critical findings", "Does not describe live risk"],
                        ["Policy coverage", `${activeDemo.policies.length} demo policies`, "Simulated only"],
                        ["Encryption coverage", "Unknown", "No telemetry source connected"],
                      ].map(([label, value, meta]) => (
                        <article key={label} className="security-metric-card">
                          <p>{label}</p><strong>{value}</strong><small>{meta.replace("{timeRange}", timeRange)}</small>
                        </article>
                      ))}
                    </section>
                    <div className="security-overview-grid">
                      <section className="security-panel">
                        <div className="security-panel-heading"><div><p className="security-eyebrow">SIMULATED</p><h2>Risk by domain</h2></div><span className="security-badge neutral">No live assessment</span></div>
                        {["Identity", "Network", "Workload", "Data", "Vulnerability", "Audit"].map((domain) => (
                          <div className="security-domain-row" key={domain}><span>{domain}</span><strong>Not assessed</strong></div>
                        ))}
                      </section>
                      <section className="security-panel">
                        <div className="security-panel-heading"><div><p className="security-eyebrow">SOURCE STATUS</p><h2>Telemetry health</h2></div></div>
                        {["Identity", "Workload", "Network", "Audit"].map((source) => (
                          <div className="security-telemetry-row" key={source}><span>{source}</span><strong>Unavailable</strong><small>No connector configured</small></div>
                        ))}
                      </section>
                    </div>
                    <section className="security-panel security-events-panel">
                      <div className="security-panel-heading"><div><p className="security-eyebrow">SYNTHETIC RECORDS</p><h2>Demo security events</h2></div><div className="security-filter-row">
                        <label className="visually-hidden" htmlFor="security-severity">Filter by severity</label>
                        <select id="security-severity" value={severityFilter} onChange={(event) => setSeverityFilter(event.target.value)}><option>All severities</option><option>Critical</option><option>High</option><option>Medium</option><option>Low</option></select>
                        <label className="visually-hidden" htmlFor="security-status">Filter by status</label>
                        <select id="security-status" value={statusFilter} onChange={(event) => setStatusFilter(event.target.value)}><option>All statuses</option><option>Needs review</option><option>Investigating</option><option>Open</option><option>Reviewed</option></select>
                      </div></div>
                      {visibleEvents.length ? (
                        <>
                          <div className="security-table-scroll"><table><thead><tr><th>Severity</th><th>Detection</th><th>Affected resource</th><th>Source</th><th>First seen</th><th>Status</th><th>Owner</th></tr></thead>
                            <tbody>{visibleEvents.map((event) => <tr key={event.id}><td><span className={`security-severity ${event.severity.toLowerCase()}`}>{event.severity}</span></td><td><button className="security-event-link" onClick={() => setSelectedEvent(event)}>{event.detection}</button></td><td>{event.resource}</td><td>{event.source}</td><td>{event.firstSeen}</td><td>{event.status}</td><td>{event.owner}</td></tr>)}</tbody>
                          </table></div>
                          <div className="security-event-cards">{visibleEvents.map((event) => <article key={event.id}>
                            <div><span className={`security-severity ${event.severity.toLowerCase()}`}>{event.severity}</span><strong>{event.status}</strong></div>
                            <button className="security-event-link" onClick={() => setSelectedEvent(event)}>{event.detection}</button>
                            <dl><dt>Affected resource</dt><dd>{event.resource}</dd><dt>Source</dt><dd>{event.source}</dd><dt>First seen</dt><dd>{event.firstSeen}</dd><dt>Owner</dt><dd>{event.owner}</dd></dl>
                          </article>)}</div>
                        </>
                      ) : <p className="security-table-empty">No demo events match these filters.</p>}
                      <small className="security-table-note">Simulated example events only. No production event query is connected.</small>
                    </section>
                    <section className="security-panel"><div className="security-panel-heading"><div><p className="security-eyebrow">SYNTHETIC ASSETS</p><h2>Asset visibility</h2></div></div>
                      <div className="security-asset-grid">{activeDemo.assets.map((asset) => <div key={asset}><strong>{asset}</strong><small>Simulated · not connected</small></div>)}</div>
                    </section>
                    <div className="security-overview-grid">
                      <section className="security-panel"><div className="security-panel-heading"><div><p className="security-eyebrow">SIMULATED GUIDANCE</p><h2>Recommendations</h2></div></div>
                        <article className="security-recommendation">
                          <span className="security-badge neutral">Demo suggestion</span>
                          <h3>Review the simulated identity sign-in pattern</h3>
                          <p>Example scope: {project} · {environment}. This suggestion is synthetic and is not based on a live assessment.</p>
                          <small>Impact: illustrative · Effort: one demo review</small>
                          <button className="security-text-button" onClick={() => setSelectedModule("Policies")}>View demo policies</button>
                        </article>
                      </section>
                      <section className="security-panel"><div className="security-panel-heading"><div><p className="security-eyebrow">BROWSER AUDIT</p><h2>Recent demo activity</h2></div></div>
                        <ul className="security-activity-list">{activeDemo.activity.slice(0, 5).map((activity, index) => <li key={`${activity}-${index}`}><span>Demo</span><p>{activity}</p></li>)}</ul>
                      </section>
                    </div>
                  </>
                )}
                {selectedModule === "Policies" && (
                  <section className="security-panel">
                    <div className="security-panel-heading"><div><p className="security-eyebrow">BROWSER-ONLY POLICY RECORDS</p><h2>Simulated policy drafts</h2></div></div>
                    <p className="security-muted">These records are stored in this browser only. Simulation does not invoke an enforcement engine.</p>
                    <div className="security-policy-list">
                      {activeDemo.policies.map((policy) => <article key={policy.id}>
                        <div><h3>{policy.name}</h3><p>{policy.type} · {policy.scope}</p><p>{policy.ruleSummary}</p><small>{policy.mode} · Updated {policy.updatedAt}</small></div>
                        <div className="security-policy-actions"><span className="security-badge neutral">{policy.status}</span>{policy.status === "Draft" && <button className="security-button secondary" onClick={() => simulatePolicy(policy)}>Simulate</button>}</div>
                      </article>)}
                    </div>
                  </section>
                )}
              </>
            )}
          </div>
        </div>
      </main>
      {renderPolicyWizard()}
      {selectedEvent && (
        <dialog ref={investigationRef} className="security-investigation" aria-labelledby="security-investigation-title" onCancel={(event) => {
          event.preventDefault();
          setSelectedEvent(null);
        }} onClick={(event) => {
          if (event.target === event.currentTarget) setSelectedEvent(null);
        }}>
          <div className="security-wizard-header">
            <div><p className="security-eyebrow">SIMULATED INVESTIGATION</p><h2 id="security-investigation-title">{selectedEvent.detection}</h2></div>
            <button type="button" className="security-icon-button" onClick={() => setSelectedEvent(null)} aria-label="Close investigation">×</button>
          </div>
          <section><h3>Summary</h3><p><span className={`security-severity ${selectedEvent.severity.toLowerCase()}`}>{selectedEvent.severity}</span> · {selectedEvent.status} · Demo record only</p><p>{selectedEvent.detection} is a synthetic example and does not represent a verified detection.</p></section>
          <section><h3>Evidence</h3><p>Evidence is unavailable because no telemetry connector is configured. No raw evidence is stored.</p></section>
          <section><h3>Timeline</h3><ol><li>{selectedEvent.firstSeen} — synthetic event created for this demo</li><li>Investigation opened in this browser</li></ol></section>
          <section><h3>Affected assets</h3><p>{selectedEvent.resource} · Simulated asset</p></section>
          <section><h3>Related detections</h3><p>No live correlation is available.</p></section>
          <section><h3>Policy context</h3><p>{activeDemo.policies.length ? `${activeDemo.policies.length} browser-only policy record(s); no enforcement engine is connected.` : "No demo policies have been created."}</p></section>
          <section><h3>Activity history</h3><p>Opened by {userEmail || "signed-in user"} · Browser-only demo</p></section>
          <div className="security-wizard-actions">
            <button type="button" className="security-button secondary" onClick={() => setSelectedEvent(null)}>Close</button>
            {selectedEvent.status !== "Reviewed" && <button type="button" className="security-button primary" onClick={() => markEventReviewed(selectedEvent)}>Mark demo reviewed</button>}
          </div>
        </dialog>
      )}
    </CloudShell>
  );
}
