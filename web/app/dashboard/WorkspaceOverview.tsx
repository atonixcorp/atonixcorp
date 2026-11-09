"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useEffect, useState } from "react";
import { CloudShell } from "@/app/shared/ui/CloudShell";
import styles from "./WorkspaceOverview.module.css";

type SessionState = "checking" | "connected" | "unavailable";
type AgentState = "checking" | "connected" | "unavailable";
type WorkspaceState = "checking" | "available" | "empty" | "unavailable";
type Workspace = { id: number; name: string; slug: string };

type AgentRegistry = {
  models?: AgentModel[];
};

type AgentModel = {
  name?: string;
  version?: string;
  stage?: string;
  production_version?: string;
  metadata?: { family?: string };
  versions?: Record<string, { version: string; stage?: string }>;
};

const resourceWorkspaces = [
  { name: "Compute", href: "/compute", detail: "Instances and compute catalog" },
  { name: "Networking", href: "/networking", detail: "Networks and subnets" },
  { name: "Storage", href: "/storage", detail: "Block storage" },
  { name: "Databases", href: "/cloud/databases", detail: "Managed database requests" },
  { name: "Containers", href: "/containers", detail: "Container deployments" },
  { name: "Security", href: "/security", detail: "Policies and security controls" },
  { name: "Observability", href: "/observability", detail: "Metrics, events, and alerts" },
  { name: "Billing", href: "/billing", detail: "Usage and billing workspace" },
];

export default function WorkspaceOverview() {
  const router = useRouter();
  const [session, setSession] = useState<SessionState>("checking");
  const [email, setEmail] = useState("");
  const [agent, setAgent] = useState<AgentState>("checking");
  const [agentModels, setAgentModels] = useState<AgentModel[] | null>(null);
  const [workspaces, setWorkspaces] = useState<Workspace[]>([]);
  const [workspaceState, setWorkspaceState] = useState<WorkspaceState>("checking");
  const [workspaceName, setWorkspaceName] = useState("");
  const [workspaceSlug, setWorkspaceSlug] = useState("");
  const [workspaceError, setWorkspaceError] = useState("");
  const [creatingWorkspace, setCreatingWorkspace] = useState(false);
  const [refreshKey, setRefreshKey] = useState(0);

  useEffect(() => {
    let active = true;

    async function checkSession() {
      try {
        const response = await fetch("/api/auth/session", {
          cache: "no-store",
          signal: AbortSignal.timeout(7000),
        });
        if (response.status === 401) {
          router.replace("/login");
          return;
        }
        const payload = await response.json().catch(() => null) as {
          user?: { email?: string };
        } | null;
        if (!response.ok || !payload?.user?.email) throw new Error("Unavailable");
        if (active) {
          setEmail(payload.user.email);
          setSession("connected");
        }
      } catch {
        if (active) setSession("unavailable");
      }
    }

    async function checkAgent() {
      try {
        const response = await fetch("/api/agent/registry", {
          cache: "no-store",
          signal: AbortSignal.timeout(7000),
        });
        if (!response.ok) throw new Error("Unavailable");
        const registry = await response.json() as AgentRegistry;
        if (active) {
          setAgentModels(Array.isArray(registry.models) ? registry.models : null);
          setAgent("connected");
        }
      } catch {
        if (active) {
          setAgentModels(null);
          setAgent("unavailable");
        }
      }
    }

    async function checkWorkspaces() {
      try {
        const response = await fetch("/api/compute/scope", {
          cache: "no-store",
          signal: AbortSignal.timeout(7000),
        });
        if (response.status === 401) {
          router.replace("/login?next=%2Fdashboard");
          return;
        }
        if (!response.ok) throw new Error("Unavailable");
        const items = await response.json() as Workspace[];
        if (!Array.isArray(items)) throw new Error("Unavailable");
        if (active) {
          setWorkspaces(items);
          setWorkspaceState(items.length ? "available" : "empty");
        }
      } catch {
        if (active) setWorkspaceState("unavailable");
      }
    }

    void checkSession();
    void checkAgent();
    void checkWorkspaces();
    const agentRefresh = window.setInterval(() => { void checkAgent(); }, 30_000);
    return () => {
      active = false;
      window.clearInterval(agentRefresh);
    };
  }, [refreshKey, router]);

  const refresh = () => {
    setSession("checking");
    setAgent("checking");
    setWorkspaceState("checking");
    setRefreshKey((key) => key + 1);
  };

  async function createWorkspace(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setCreatingWorkspace(true);
    setWorkspaceError("");
    try {
      const response = await fetch("/api/compute/scope", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ name: workspaceName, slug: workspaceSlug }),
      });
      const result = await response.json().catch(() => null) as Workspace & { error?: string } | null;
      if (!response.ok || !result?.id) {
        throw new Error(result?.error || "Could not create workspace.");
      }
      setWorkspaces([result]);
      setWorkspaceState("available");
      setWorkspaceName("");
      setWorkspaceSlug("");
    } catch (error) {
      setWorkspaceError(error instanceof Error ? error.message : "Could not create workspace.");
    } finally {
      setCreatingWorkspace(false);
    }
  }

  return (
    <CloudShell>
      <main className={styles.page}>
        <header className={styles.heading}>
          <div>
            <p className={styles.eyebrow}>WORKSPACE OVERVIEW</p>
            <h1>Cloud overview</h1>
            <p className={styles.intro}>A live view of configured platform connections and available resource workspaces.</p>
          </div>
          <button className={styles.refresh} type="button" onClick={refresh}>
            <svg aria-hidden="true" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round">
              <path d="M20 7v5h-5M4 17v-5h5" />
              <path d="M5.6 9a7 7 0 0 1 11.6-2L20 12M4 12l2.8 5a7 7 0 0 0 11.6-2" />
            </svg>
            Refresh status
          </button>
        </header>

        <section className={styles.context} aria-label="Current account context">
          <div>
            <span>Account</span>
            <strong>{session === "connected" ? email : session === "checking" ? "Checking session" : "Unavailable"}</strong>
          </div>
          <div>
            <span>Workspace</span>
            <strong>{workspaceState === "checking" ? "Checking workspaces" : workspaceState === "available" ? workspaces[0]?.name : workspaceState === "empty" ? "None created" : "Unavailable"}</strong>
          </div>
          <div>
            <span>Region</span>
            <strong>Not selected</strong>
          </div>
          <p>Region selection and resource summaries are not available in the dashboard yet.</p>
        </section>

        {workspaceState === "empty" && (
          <section className={styles.workspaceSetup} aria-labelledby="workspace-setup-title">
            <div>
              <p className={styles.eyebrow}>FIRST WORKSPACE</p>
              <h2 id="workspace-setup-title">Create your workspace</h2>
              <p>A default project will be created and assigned to your account as owner.</p>
            </div>
            <form onSubmit={createWorkspace}>
              <label>
                <span>Workspace name</span>
                <input value={workspaceName} onChange={(event) => setWorkspaceName(event.target.value)} maxLength={255} required />
              </label>
              <label>
                <span>Workspace slug</span>
                <input value={workspaceSlug} onChange={(event) => setWorkspaceSlug(event.target.value)} pattern="[a-z0-9](?:[a-z0-9-]{0,78}[a-z0-9])?" maxLength={80} required />
              </label>
              <button type="submit" disabled={creatingWorkspace}>{creatingWorkspace ? "Creating…" : "Create workspace"}</button>
            </form>
            {workspaceError && <p className={styles.pageError} role="alert">{workspaceError}</p>}
          </section>
        )}
        {workspaceState === "unavailable" && <p className={styles.pageError} role="status">Workspace access could not be verified. Try refreshing the dashboard.</p>}

        <section className={styles.connections} aria-labelledby="connections-title">
          <div className={styles.sectionHeading}>
            <div>
              <h2 id="connections-title">Platform connections</h2>
              <p>Verified using the current account session and agent registry endpoints.</p>
            </div>
          </div>
          <div className={styles.connectionGrid}>
            <Connection name="Platform API" state={session} detail={session === "connected" ? "Authenticated session verified" : session === "checking" ? "Checking authenticated session" : "Session service unavailable"} />
            <Connection name="AtonixCorp Agent" state={agent} detail={agent === "connected" ? agentModels === null ? "Registry connected; model inventory unavailable" : `${agentModels.length} models registered · ${agentModels.filter((model) => Boolean(model.production_version)).length} in production` : agent === "checking" ? "Checking agent registry" : "Agent API unavailable"} />
          </div>
        </section>

        <section className={styles.modelRegistry} aria-labelledby="model-registry-title">
          <div className={styles.sectionHeading}>
            <div>
              <h2 id="model-registry-title">Agent model registry</h2>
              <p>Read-only inventory from the Agent API. Recommendations require live workload telemetry, which is not connected.</p>
            </div>
          </div>
          {agent === "checking" ? (
            <p className={styles.registryState} role="status">Checking the model registry…</p>
          ) : agent === "unavailable" ? (
            <p className={styles.registryState}>Model inventory is unavailable because the Agent API could not be reached.</p>
          ) : agentModels === null ? (
            <p className={styles.registryState}>The registry responded without a model inventory.</p>
          ) : agentModels.length === 0 ? (
            <p className={styles.registryState}>No models are registered.</p>
          ) : (
            <ul className={styles.modelList}>
              {agentModels.map((model, index) => {
                const versions = Object.values(model.versions ?? {});
                return (
                  <li className={styles.modelItem} key={`${model.name ?? "model"}-${index}`}>
                    <div className={styles.modelName}>
                      <strong>{model.name || "Unnamed model"}</strong>
                      <small>{model.metadata?.family ?? "Model family not specified"}</small>
                    </div>
                    <div className={styles.modelVersions}>
                      {versions.length > 0
                        ? versions.map((version) => <span key={version.version}>{version.version} · {version.stage ?? "stage unspecified"}</span>)
                        : <span>{model.version ?? "Version not specified"} · {model.stage ?? "stage unspecified"}</span>}
                    </div>
                    <span className={model.production_version ? styles.production : styles.notProduction}>
                      {model.production_version ? `Production · ${model.production_version}` : "No production version"}
                    </span>
                  </li>
                );
              })}
            </ul>
          )}
        </section>

        <section className={styles.resources} aria-labelledby="resources-title">
          <div className={styles.sectionHeading}>
            <div>
              <h2 id="resources-title">Resource workspaces</h2>
              <p>Open a platform area. Resource totals are not currently available from a unified summary API.</p>
            </div>
          </div>
          <div className={styles.workspaceGrid}>
            {resourceWorkspaces.map((workspace) => (
              <Link className={styles.workspaceLink} href={workspace.href} key={workspace.name}>
                <span className={styles.workspaceMark} aria-hidden="true">{workspace.name.slice(0, 1)}</span>
                <span className={styles.workspaceText}>
                  <strong>{workspace.name}</strong>
                  <small>{workspace.detail}</small>
                </span>
                <span className={styles.workspaceArrow} aria-hidden="true">↗</span>
              </Link>
            ))}
          </div>
        </section>

        <section className={styles.unavailable} aria-labelledby="live-data-title">
          <div className={styles.unavailableHeader}>
            <span className={styles.infoMark} aria-hidden="true">i</span>
            <div>
              <h2 id="live-data-title">Live resource summary unavailable</h2>
              <p>The platform does not yet expose an authorized dashboard aggregation API for this account.</p>
            </div>
          </div>
          <dl className={styles.dataList}>
            <div><dt>Instances, networks, and volumes</dt><dd>Unavailable</dd></div>
            <div><dt>Quotas and utilization</dt><dd>Unavailable</dd></div>
            <div><dt>Recent platform activity</dt><dd>Not integrated</dd></div>
            <div><dt>Infrastructure health</dt><dd>Not aggregated</dd></div>
          </dl>
        </section>

        {session === "unavailable" && (
          <p className={styles.pageError} role="status">Account status could not be verified. Some workspaces may not be available.</p>
        )}
      </main>
    </CloudShell>
  );
}

function Connection({ name, state, detail }: { name: string; state: SessionState | AgentState; detail: string }) {
  const label = state === "connected" ? "Connected" : state === "checking" ? "Checking" : "Unavailable";
  return (
    <article className={styles.connection}>
      <div className={`${styles.connectionStatus} ${styles[state]}`} aria-label={label}>
        <span aria-hidden="true" />{label}
      </div>
      <h3>{name}</h3>
      <p>{detail}</p>
    </article>
  );
}