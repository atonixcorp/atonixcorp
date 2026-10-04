import { useCallback, useEffect, useMemo, useState } from "react";
import { getJobLogs, getRunnerState, startPipeline } from "./api.js";

const sidebar = [
  ["Dashboard", "grid"],
  ["Pipelines", "pipeline"],
  ["Deployments", "deploy"],
  ["Infrastructure", "server"],
  ["Kubernetes", "kube"],
  ["Applications", "apps"],
  ["Observability", "chart"],
  ["Security", "shield"],
  ["AI Operations", "brain"],
  ["Automation", "settings"],
  ["Cost Management", "dollar"],
  ["Reports", "report"],
  ["Settings", "settings"],
];

const initialPipelines = [
  { id: "demo-pipe-01", project: "platform-api", ref: "main", status: "success", trigger: "push", started_at: "Today, 10:42", labels: ["main", "go"], demo: true, jobs: [{ id: "d1", name: "Build & test", stage: "test", status: "success" }] },
  { id: "demo-pipe-02", project: "cloud-console", ref: "feature/dashboard", status: "running", trigger: "pull request", started_at: "Today, 10:31", labels: ["frontend"], demo: true, jobs: [{ id: "d2", name: "Security scan", stage: "security", status: "running" }] },
  { id: "demo-pipe-03", project: "data-services", ref: "release/1.4", status: "failed", trigger: "push", started_at: "Today, 09:58", labels: ["release"], demo: true, jobs: [{ id: "d3", name: "Integration tests", stage: "test", status: "failed" }] },
];

const metrics = [
  { label: "Deployment success rate", value: "99.2%", delta: "+0.6%", direction: "up", icon: "deploy", points: "0,27 15,25 28,29 42,14 55,21 68,12 82,17 100,4", color: "blue" },
  { label: "Platform availability", value: "99.98%", delta: "+0.01%", direction: "up", icon: "shield", points: "0,25 15,21 28,23 42,15 55,17 68,8 82,13 100,5", color: "teal" },
  { label: "Security compliance", value: "98.7%", delta: "+1.2%", direction: "up", icon: "shield", points: "0,26 15,23 28,25 42,17 55,21 68,11 82,14 100,5", color: "purple" },
  { label: "AI automation efficiency", value: "87.3%", delta: "+4.8%", direction: "up", icon: "brain", points: "0,29 15,21 28,25 42,17 55,19 68,9 82,14 100,4", color: "purple" },
  { label: "Infrastructure health", value: "96.1%", delta: "+2.1%", direction: "up", icon: "server", points: "0,28 15,23 28,26 42,14 55,19 68,9 82,12 100,4", color: "teal" },
  { label: "Cost optimisation", value: "23.6%", delta: "-3.4%", direction: "down", icon: "clock", points: "0,8 15,18 28,12 42,23 55,16 68,28 82,14 100,7", color: "orange" },
];

const systems = [
  ["OrcaCloud", "Operational", "99.99%", "118 ms"],
  ["OrcaStack", "Operational", "99.98%", "142 ms"],
  ["Data Platform", "Operational", "99.97%", "156 ms"],
  ["Security Services", "Operational", "99.99%", "121 ms"],
  ["Automation", "Operational", "99.98%", "138 ms"],
];

const deployItems = [
  ["api-gateway", "v2.4.1", "2m ago", "Production", "success"],
  ["user-service", "v1.8.3", "12m ago", "Production", "success"],
  ["data-processor", "v2.0.2", "28m ago", "Staging", "success"],
  ["ml-inference", "v2.0.1", "1h ago", "Production", "success"],
  ["web-frontend", "v1.5.2", "2h ago", "Staging", "running"],
];

const jobsByStatus = (pipelines, status) => pipelines.reduce(
  (total, pipeline) => total + (pipeline.jobs || []).filter((job) => job.status === status).length,
  0,
);

function getInitialTheme() {
  try {
    return window.localStorage.getItem("atonixcorp-devops-theme") === "dark" ? "dark" : "light";
  } catch {
    return "light";
  }
}

function Icon({ name, size = 19 }) {
  const paths = {
    grid: "M3 3h7v7H3zM14 3h7v7h-7zM3 14h7v7H3zM14 14h7v7h-7z",
    pipeline: "M6 3v5m12-5v5M6 16v5m12-5v5M6 8h12v8H6z",
    deploy: "M4 19h16M6 16l12-9M14 7h4v4M6 5v7",
    server: "M4 4h16v6H4zM4 14h16v6H4zM8 7h.01M8 17h.01M12 7h5M12 17h5",
    kube: "m12 2 8 4.5v11L12 22l-8-4.5v-11L12 2Zm0 0v8m8-3.5-8 4-8-4m8 4v11",
    apps: "M4 4h6v6H4zM14 4h6v6h-6zM4 14h6v6H4zM14 14h6v6h-6z",
    chart: "M3 3v18h18M7 14l4-4 4 3 6-8",
    shield: "M12 22s8-4 8-11V5l-8-3-8 3v6c0 7 8 11 8 11Zm-3-11 2 2 4-4",
    brain: "M9 4a3 3 0 0 0-5 2 3 3 0 0 0-1 5 3 3 0 0 0 2 5 3 3 0 0 0 5 2V4Zm6 0a3 3 0 0 1 5 2 3 3 0 0 1 1 5 3 3 0 0 1-2 5 3 3 0 0 1-5 2V4ZM9 8h3m3 4h-3m-3 4h3",
    settings: "M12 8a4 4 0 1 0 0 8 4 4 0 0 0 0-8Zm0-5v2m0 14v2M4.9 4.9l1.4 1.4m11.4 11.4 1.4 1.4M3 12h2m14 0h2M4.9 19.1l1.4-1.4M17.7 6.3l1.4-1.4",
    dollar: "M12 2v20m5-16H9.5a3.5 3.5 0 0 0 0 7h5a3.5 3.5 0 0 1 0 7H6",
    report: "M5 3h14v18H5zM8 8h8m-8 4h8m-8 4h5",
    clock: "M12 22a10 10 0 1 0 0-20 10 10 0 0 0 0 20Zm0-16v6l4 2",
    search: "m20 20-4.5-4.5M18 10.5a7.5 7.5 0 1 1-15 0 7.5 7.5 0 0 1 15 0Z",
    bell: "M18 8a6 6 0 0 0-12 0c0 7-3 7-3 9h18c0-2-3-2-3-9Zm-8 12a2 2 0 0 0 4 0",
    plus: "M12 5v14m-7-7h14",
    refresh: "M20 7v5h-5M4 17v-5h5m-3.5-3A7 7 0 0 1 17 6l3 6M4 12l3 6a7 7 0 0 0 11.5-3",
    arrow: "M5 12h14m-6-6 6 6-6 6",
    close: "m18 6-12 12M6 6l12 12",
    check: "m5 12 4 4L19 6",
    moon: "M20.5 14.5A8.5 8.5 0 0 1 9.5 3.5a8.5 8.5 0 1 0 11 11Z",
    sun: "M12 3v2m0 14v2M5.64 5.64l1.42 1.42m9.88 9.88 1.42 1.42M3 12h2m14 0h2M5.64 18.36l1.42-1.42m9.88-9.88 1.42-1.42M16 12a4 4 0 1 1-8 0 4 4 0 0 1 8 0Z",
  };
  return (
    <svg width={size} height={size} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <path d={paths[name] || paths.grid} />
    </svg>
  );
}

function Sparkline({ points, color }) {
  return (
    <svg className={`sparkline ${color}`} viewBox="0 0 100 34" preserveAspectRatio="none" aria-hidden="true">
      <polyline points={points} fill="none" stroke="currentColor" strokeWidth="2.6" vectorEffect="non-scaling-stroke" />
    </svg>
  );
}

function MetricCard({ metric, value, delta }) {
  return (
    <article className={`metric-card ${metric.color}`}>
      <div className="metric-top">
        <span className="metric-icon"><Icon name={metric.icon} size={17} /></span>
        <span className="metric-label">{metric.label}</span>
      </div>
      <div className="metric-bottom">
        <div>
          <strong>{value || metric.value}</strong>
          <small className={metric.direction}>{metric.direction === "up" ? "▲" : "▼"} {delta || metric.delta} <span>vs last week</span></small>
        </div>
        <Sparkline points={metric.points} color={metric.color} />
      </div>
    </article>
  );
}

function StatusBadge({ status }) {
  const normalized = String(status || "unknown").toLowerCase();
  const className = normalized === "success" ? "passed" : normalized;
  return <span className={`status-badge ${className}`}><i />{normalized}</span>;
}

function Header({ mode, theme, onThemeToggle, onModeChange, onRefresh, refreshing, error, query, onQueryChange, onNewPipeline }) {
  return (
    <header className="topbar">
      <a className="brand" href="/" aria-label="AtonixCorp DevSecOps dashboard">
        <img className="brand-logo" src="/atonixcorp-icon.png" alt="" width="42" height="42" />
        <span className="brand-copy"><strong>AtonixCorp <b>DevSecOps Pipeline</b></strong><small>MONITOR&nbsp; | &nbsp;DEPLOY&nbsp; | &nbsp;SECURE&nbsp; | &nbsp;AUTOMATE&nbsp; | &nbsp;SCALE</small></span>
      </a>
      <label className="top-search">
        <Icon name="search" size={17} />
        <input value={query} onChange={(event) => onQueryChange(event.target.value)} placeholder="Search systems, deployments, services..." aria-label="Search pipelines and services" />
        {query && <button type="button" onClick={() => onQueryChange("")} aria-label="Clear search"><Icon name="close" size={14} /></button>}
      </label>
      <div className="top-actions">
        <span className="top-date"><Icon name="clock" size={15} />{new Date().toLocaleDateString(undefined, { weekday: "short", month: "short", day: "2-digit" })}</span>
        <button className="live-status" type="button" onClick={onRefresh} title={error || "Refresh runner connection"}>
          <i className={mode === "live" ? "online" : "demo"} />{mode === "live" ? "Runner Connected" : "Demo Mode"}
        </button>
        <button
          className="theme-toggle"
          type="button"
          onClick={onThemeToggle}
          aria-label={`Switch to ${theme === "dark" ? "light" : "dark"} mode`}
          title={`Switch to ${theme === "dark" ? "light" : "dark"} mode`}
        >
          <Icon name={theme === "dark" ? "sun" : "moon"} size={17} />
        </button>
        <button className="top-icon-button" type="button" onClick={onRefresh} aria-label="Refresh pipeline data" disabled={refreshing}><Icon name="refresh" size={17} /></button>
        <button className={`connect-button ${mode === "live" ? "connected" : ""}`} type="button" onClick={onModeChange}>{mode === "live" ? "Use demo" : "Connect runner"}</button>
        <button className="new-pipeline-button" type="button" onClick={onNewPipeline}><Icon name="plus" size={17} /> New pipeline</button>
      </div>
    </header>
  );
}

function Sidebar({ active, onSelect, pipelineCount }) {
  return (
    <aside className="sidebar">
      <p className="sidebar-label">WORKSPACE</p>
      <nav aria-label="DevSecOps navigation">
        {sidebar.map(([label, icon]) => (
          <button type="button" key={label} className={`sidebar-link ${active === label ? "active" : ""}`} onClick={() => onSelect(label)} aria-current={active === label ? "page" : undefined}>
            <Icon name={icon} size={17} /><span>{label}</span>{label === "Pipelines" && <small>{pipelineCount}</small>}
          </button>
        ))}
      </nav>
      <div className="sidebar-bottom">
        <div className="runner-card"><span className={`runner-dot ${active === "Dashboard" ? "" : ""}`} /><div><strong>{active === "Dashboard" ? "Runner status" : "Workspace"}</strong><small>{active === "Dashboard" ? "Demo runner · idle" : "AtonixCorp Platform"}</small></div><span className="runner-more">···</span></div>
        <div className="sidebar-user"><span className="user-avatar">JD</span><span><strong>Demo Engineer</strong><small>Platform team</small></span><span className="user-menu">⌄</span></div>
      </div>
    </aside>
  );
}

function PipelineTable({ pipelines, onSelect }) {
  if (!pipelines.length) {
    return <div className="empty-pipelines"><span className="empty-icon"><Icon name="pipeline" size={23} /></span><strong>No pipeline runs yet</strong><p>Start a pipeline to see execution status and jobs here.</p></div>;
  }

  return (
    <div className="pipeline-table-wrap">
      <table className="pipeline-table">
        <thead><tr><th>PIPELINE</th><th>BRANCH / REF</th><th>TRIGGER</th><th>JOBS</th><th>STATUS</th><th>STARTED</th><th aria-label="Actions" /></tr></thead>
        <tbody>
          {pipelines.map((pipeline) => (
            <tr key={pipeline.id} onClick={() => onSelect(pipeline)} tabIndex="0" onKeyDown={(event) => { if (event.key === "Enter") onSelect(pipeline); }}>
              <td><span className={`pipeline-status-icon ${pipeline.status}`}><Icon name={pipeline.status === "success" ? "check" : pipeline.status === "failed" ? "close" : "pipeline"} size={14} /></span><span><strong>{pipeline.project}</strong><small>#{String(pipeline.id).slice(-8)}</small></span></td>
              <td><code>{pipeline.ref}</code></td>
              <td>{pipeline.trigger || "manual"}</td>
              <td>{(pipeline.jobs || []).length} job{(pipeline.jobs || []).length === 1 ? "" : "s"}</td>
              <td><StatusBadge status={pipeline.status} />{pipeline.demo && <span className="demo-tag">DEMO</span>}</td>
              <td>{pipeline.started_at}</td>
              <td><button className="row-action" type="button" onClick={(event) => { event.stopPropagation(); onSelect(pipeline); }} aria-label={`View ${pipeline.project} pipeline`}><Icon name="arrow" size={15} /></button></td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

function PipelineModal({ onClose, onCreate, mode, loading }) {
  const [project, setProject] = useState("atonixcorp/devops");
  const [ref, setRef] = useState("main");
  const [workflow, setWorkflow] = useState("api-test");
  const [formError, setFormError] = useState("");

  function submit(event) {
    event.preventDefault();
    if (!project.trim() || !ref.trim()) {
      setFormError("Enter both a project and a branch or ref.");
      return;
    }
    onCreate({ project: project.trim(), ref: ref.trim(), workflow }).catch((error) => setFormError(error.message));
  }

  return (
    <div className="modal-backdrop" role="presentation" onMouseDown={(event) => { if (event.target === event.currentTarget) onClose(); }}>
      <section className="pipeline-modal" role="dialog" aria-modal="true" aria-labelledby="new-pipeline-title">
        <header><div><p className="eyebrow">PIPELINE CONTROL</p><h2 id="new-pipeline-title">Run a pipeline</h2></div><button type="button" className="modal-close" onClick={onClose} aria-label="Close"><Icon name="close" /></button></header>
        {mode === "demo" && <div className="modal-demo-note"><strong>Demo mode</strong> — this creates a sample run in the dashboard only. Connect the runner to execute real jobs.</div>}
        {mode === "live" && <div className="modal-warning"><strong>Live runner:</strong> this submits the selected workflow to your DevOps runner. The current runner supports <code>api-test</code>.</div>}
        <form onSubmit={submit}>
          <label>Project or repository<input value={project} onChange={(event) => setProject(event.target.value)} required maxLength={100} placeholder="organization/project" /></label>
          <label>Branch or ref<input value={ref} onChange={(event) => setRef(event.target.value)} required maxLength={100} placeholder="main" /></label>
          <label>Workflow<select value={workflow} onChange={(event) => setWorkflow(event.target.value)}><option value="api-test">API test suite</option></select></label>
          {formError && <p className="form-error" role="alert">{formError}</p>}
          <div className="modal-actions"><button type="button" className="button-secondary" onClick={onClose}>Cancel</button><button className="button-primary" type="submit" disabled={loading}>{loading ? "Starting..." : mode === "live" ? "Run on runner" : "Create demo run"}<Icon name="arrow" size={15} /></button></div>
        </form>
      </section>
    </div>
  );
}

function PipelineDetails({ pipeline, onClose, mode, onLoadLogs, logs, logsError }) {
  return (
    <div className="modal-backdrop" role="presentation" onMouseDown={(event) => { if (event.target === event.currentTarget) onClose(); }}>
      <section className="pipeline-modal detail-modal" role="dialog" aria-modal="true" aria-labelledby="pipeline-detail-title">
        <header><div><p className="eyebrow">PIPELINE RUN · {String(pipeline.id).slice(-8)}</p><h2 id="pipeline-detail-title">{pipeline.project}</h2></div><button type="button" className="modal-close" onClick={onClose} aria-label="Close"><Icon name="close" /></button></header>
        <div className="detail-meta"><span>Ref <code>{pipeline.ref}</code></span><span>Trigger {pipeline.trigger || "manual"}</span><StatusBadge status={pipeline.status} />{pipeline.demo && <span className="demo-tag">DEMO</span>}</div>
        <h3>Jobs</h3>
        <div className="job-list">{(pipeline.jobs || []).map((job) => (
          <article key={job.id}><span className={`pipeline-status-icon ${job.status}`}><Icon name={job.status === "success" ? "check" : "pipeline"} size={14} /></span><div><strong>{job.name}</strong><small>{job.stage} stage {job.executor ? `· ${job.executor}` : ""}</small></div><StatusBadge status={job.status} />
            {mode === "live" && job.log_endpoint && <button type="button" className="logs-button" onClick={() => onLoadLogs(job.log_endpoint)}>View logs</button>}
          </article>
        ))}</div>
        {logsError && <p className="form-error" role="alert">{logsError}</p>}
        {logs && <pre className="job-logs">{logs}</pre>}
        {pipeline.demo && <p className="modal-demo-note">Illustrative demo run. No job was executed by the runner.</p>}
      </section>
    </div>
  );
}

function DashboardOverview({ pipelines, summary, mode, query, onSelect, onNewPipeline, onNavigate }) {
  const completed = pipelines.filter((pipeline) => ["success", "failed"].includes(pipeline.status)).length;
  const succeeded = pipelines.filter((pipeline) => pipeline.status === "success").length;
  const successRate = completed ? `${Math.round((succeeded / completed) * 100)}%` : null;
  const runningJobs = summary.busy_executors ?? jobsByStatus(pipelines, "running");
  const queuedJobs = summary.queued_jobs ?? jobsByStatus(pipelines, "queued");
  const shownPipelines = pipelines.filter((pipeline) => `${pipeline.project} ${pipeline.ref} ${pipeline.status}`.toLowerCase().includes(query.toLowerCase()));

  return (
    <>
      <div className="page-heading"><div><p className="eyebrow">OVERVIEW / DEVSECOPS</p><h1>Pipeline control center</h1><p>Track delivery health, pipeline runs, and security signals across your platform.</p></div><button className="button-primary" onClick={onNewPipeline}><Icon name="plus" /> Run pipeline</button></div>
      {mode === "demo" && <div className="demo-banner"><span className="demo-info">i</span><p><strong>Interactive demo mode</strong> — dashboard metrics and pipeline history are sample data. Connect a runner to load real runs and execute jobs.</p><span className="demo-banner-tag">SAMPLE DATA</span></div>}
      {mode === "live" && <div className="connected-banner"><span className="demo-info"><Icon name="check" size={13} /></span><p><strong>Runner connected.</strong> Pipeline data is loaded from the DevSecOps runner API.</p><span className="demo-banner-tag">LIVE API</span></div>}
      <section className="metrics-grid" aria-label="Platform metrics">
        {metrics.map((metric) => (
          <MetricCard key={metric.label} metric={metric}
            value={metric.label === "Deployment success rate" && mode === "live" ? successRate ?? "—" : undefined}
            delta={metric.label === "Deployment success rate" && mode === "live" && successRate ? `${succeeded}/${completed} runs` : undefined}
          />
        ))}
      </section>
      <section className="overview-grid">
        <article className="panel fabric-panel">
          <PanelHeading eyebrow="CONNECTED CLOUD FOUNDATION" title="Global platform fabric" action="View platform" onAction={() => onNavigate("Infrastructure")} />
          <p className="panel-subtitle">A unified cloud foundation for a smarter, more connected tomorrow.</p>
          <div className="fabric-content">
            <div className="fabric-principles">
              <strong>One platform.<br />Infinite possibilities.</strong>
              <div><Icon name="pipeline" /><span><b>Build</b><small>Faster</small></span></div>
              <div><Icon name="deploy" /><span><b>Operate</b><small>Smarter</small></span></div>
              <div><Icon name="shield" /><span><b>Secure</b><small>Every layer</small></span></div>
              <div><Icon name="chart" /><span><b>Scale</b><small>Without limits</small></span></div>
            </div>
            <FabricDiagram />
          </div>
        </article>
        <article className="panel health-panel">
          <PanelHeading eyebrow="LIVE SERVICE STATUS" title="Platform health" action="View details" onAction={() => onNavigate("Observability")} />
          <div className="health-overview"><div className="health-ring"><strong>{mode === "demo" ? "99.98%" : runningJobs ? "Monitoring" : "Ready"}</strong></div><div className="health-legend"><strong>Overall platform health</strong><span><i className="good" /> Healthy <b>{mode === "demo" ? 18 : Math.max(0, 18 - runningJobs)}</b></span><span><i className="warning" /> Degraded <b>0</b></span><span><i className="critical" /> Critical <b>0</b></span></div></div>
          <MiniBars />
        </article>
        <article className="panel deployments-panel">
          <PanelHeading eyebrow="DELIVERY ACTIVITY" title="Recent deployments" action="View all" onAction={() => onNavigate("Deployments")} />
          <div className="deploy-list">{deployItems.slice(0, 4).map(([name, version, when, env, status]) => <div className="deploy-row" key={name}><span className={`deploy-check ${status}`}>{status === "success" ? <Icon name="check" size={12} /> : "•••"}</span><strong>{name}</strong><code>{version}</code><small>{when}</small><span className={`env-tag ${env.toLowerCase()}`}>{env}</span></div>)}</div>
        </article>
        <article className="panel systems-panel">
          <PanelHeading eyebrow="SERVICE MONITORING" title="System status" action="All services" onAction={() => onNavigate("Infrastructure")} />
          <table className="systems-table"><thead><tr><th>Service</th><th>Status</th><th>Uptime</th><th>Latency</th></tr></thead><tbody>{systems.map(([name, status, uptime, latency]) => <tr key={name}><td><span className="system-service-icon"><Icon name="server" size={14} /></span>{name}</td><td><span className="system-operation"><i />{mode === "live" ? "Connected" : status}</span></td><td>{mode === "live" ? "—" : uptime}</td><td>{mode === "live" ? "—" : latency}</td></tr>)}</tbody></table>
        </article>
        <article className="panel pipeline-panel">
          <PanelHeading eyebrow={mode === "live" ? "CI/CD · RUNNER API" : "CI/CD · DEMO DATA"} title="Recent pipeline runs" action={mode === "live" ? `${pipelines.length} total` : "View all"} onAction={() => onNavigate("Pipelines")} />
          {mode === "live" && <p className="panel-subtitle live-source">Source: connected runner API</p>}
          <PipelineTable pipelines={shownPipelines.slice(0, 4)} onSelect={onSelect} />
        </article>
        <article className="panel incidents-panel">
          <PanelHeading eyebrow="OPERATIONS" title="Active incidents" action="Incident center" onAction={() => onNavigate("Security")} />
          <div className="incident-list"><Incident severity="high" title="High memory usage" detail="Compute cluster · prod-east" when="12m ago" state="Investigating" /><Incident severity="info" title="Pod restart loop" detail="ml-inference · staging" when="28m ago" state="Monitoring" /><Incident severity="warning" title="Elevated API latency" detail="api-gateway · us-west" when="1h ago" state="Investigating" /><Incident severity="success" title="Resolved: DB connection" detail="data-platform · production" when="2h ago" state="Resolved" /></div>
        </article>
        <article className="panel capacity-panel">
          <PanelHeading eyebrow="WORKLOAD UTILISATION" title="Infrastructure capacity" action="Capacity plan" onAction={() => onNavigate("Infrastructure")} />
          <div className="capacity-content"><div className="capacity-ring"><strong>68%<small>total capacity</small></strong></div><div className="capacity-legend"><span><i className="blue" />Compute <b>68%</b></span><span><i className="teal" />Storage <b>54%</b></span><span><i className="purple" />Networking <b>42%</b></span><span><i className="violet" />Memory <b>62%</b></span></div></div>
        </article>
        <article className="panel security-panel"><PanelHeading eyebrow="CONTINUOUS ASSURANCE" title="Security alerts" action="View security" onAction={() => onNavigate("Security")} /><AlertList /></article>
        <article className="panel recommendations-panel"><PanelHeading eyebrow="INTELLIGENT OPERATIONS" title="AI recommendations" action="Review all" onAction={() => onNavigate("AI Operations")} /><RecommendationList /></article>
        <article className="panel operations-panel"><PanelHeading eyebrow="PLATFORM TELEMETRY" title="Pipeline activity" action="Open pipelines" onAction={() => onNavigate("Pipelines")} /><div className="pipeline-stats"><span><strong>{pipelines.length}</strong><small>Pipeline runs</small></span><span><strong>{runningJobs}</strong><small>Running jobs</small></span><span><strong>{queuedJobs}</strong><small>Queued jobs</small></span></div><div className="activity-spark"><ActivityChart /></div><button className="text-action" type="button" onClick={onNewPipeline}>Start a pipeline <Icon name="arrow" size={14} /></button></article>
      </section>
    </>
  );
}

function PanelHeading({ eyebrow, title, action, onAction }) {
  return <header className="panel-heading"><div><p>{eyebrow}</p><h2>{title}</h2></div>{onAction ? <button type="button" className="panel-action" onClick={onAction}>{action}<Icon name="arrow" size={13} /></button> : <span className="panel-action panel-action-label">{action}</span>}</header>;
}

function FabricDiagram() {
  const nodes = [["Cloud Platform", "cloud", "node-cloud"], ["Kubernetes", "kube", "node-kube"], ["Storage", "server", "node-storage"], ["Networking", "apps", "node-network"], ["Observability", "chart", "node-observe"], ["Automation", "settings", "node-automation"], ["Data Platform", "report", "node-data"], ["HPC", "server", "node-hpc"], ["DevSecOps", "shield", "node-devsecops"], ["AI Operations", "brain", "node-ai"]];
  return <div className="fabric-diagram"><div className="fabric-orbit orbit-one" /><div className="fabric-orbit orbit-two" /><div className="fabric-center"><span className="brand-mark"><span /></span><strong>AtonixCorp</strong></div>{nodes.map(([label, icon, position]) => <div className={`fabric-node ${position}`} key={label}><span><Icon name={icon} size={19} /></span><small>{label}</small></div>)}</div>;
}

function MiniBars() {
  return <div className="mini-bars" aria-label="Service health in the last 24 hours">{[29, 41, 36, 57, 46, 67, 55, 73, 61, 85, 66, 78].map((height, index) => <i key={index} style={{ height: `${height}%` }} />)}<small>LAST 24 HOURS</small></div>;
}

function Incident({ severity, title, detail, when, state }) {
  return <div className="incident"><span className={`incident-icon ${severity}`}>{severity === "success" ? <Icon name="check" size={12} /> : severity === "high" ? "!" : severity === "warning" ? "▲" : "i"}</span><div><strong>{title}</strong><small>{detail}</small></div><time>{when}</time><span className={`incident-state ${severity}`}>{state}</span></div>;
}

function AlertList() {
  const alerts = [["high", "Multiple failed login attempts", "8m ago", "High"], ["warning", "Unusual API access pattern", "34m ago", "Medium"], ["info", "New vulnerable dependency", "2h ago", "Low"], ["info", "SSL certificate expiring", "5h ago", "Low"]];
  return <div className="alert-list">{alerts.map(([level, title, when, priority]) => <div className="alert-row" key={title}><span className={`alert-symbol ${level}`}>{level === "high" ? "!" : level === "warning" ? "▲" : "i"}</span><strong>{title}</strong><small>{when}</small><span className={`priority ${level}`}>{priority}</span></div>)}</div>;
}

function RecommendationList() {
  const items = [["brain", "Scale compute resources", "Predicted 80% CPU in 2 hours", "High"], ["shield", "Optimise storage tiering", "Move 12 TB to cold storage", "Medium"], ["pipeline", "Update dependency", "New secure version available", "Low"]];
  return <div className="recommendation-list">{items.map(([icon, title, description, priority]) => <div className="recommendation-row" key={title}><span><Icon name={icon} size={15} /></span><div><strong>{title}</strong><small>{description}</small></div><b className={priority.toLowerCase()}>{priority}</b></div>)}</div>;
}

function ActivityChart() {
  return <svg viewBox="0 0 320 68" preserveAspectRatio="none" aria-label="Illustrative pipeline activity trend"><path d="M0 56 C20 54 22 37 43 39 S70 49 88 35 S112 16 131 26 S158 49 178 35 S200 14 219 22 S243 47 261 30 S290 16 320 20" fill="none" stroke="#1878d8" strokeWidth="2.4" /><path d="M0 62H320" stroke="#e4eaf2" strokeWidth="1" /></svg>;
}

function ServiceWorkspace({ active, pipelines, onSelect, onNewPipeline, mode }) {
  const content = {
    Pipelines: ["Pipeline runs", "Review run status, job stages, branch refs, and runner results."],
    Deployments: ["Deployments", "Track release activity across production and staging environments."],
    Infrastructure: ["Infrastructure", "Review platform components, capacity, and connected services."],
    Kubernetes: ["Kubernetes", "Inspect cluster delivery and container workload health."],
    Applications: ["Applications", "Manage application delivery and release history."],
    Observability: ["Observability", "Monitor service health, latency, and operational signals."],
    Security: ["Security center", "Review security alerts, compliance signals, and policy findings."],
    "AI Operations": ["AI operations", "Explore intelligent operations and workload recommendations."],
    Automation: ["Automation", "Review automation workflows and delivery controls."],
    "Cost Management": ["Cost management", "Review usage signals and platform cost optimisation."],
    Reports: ["Reports", "Pipeline and platform reporting workspace."],
    Settings: ["Workspace settings", "Configure this dashboard and runner connection."],
  };
  const [title, description] = content[active] || content.Pipelines;
  return <section className="workspace-page"><div className="page-heading"><div><p className="eyebrow">ATONIXCORP / DEVSECOPS</p><h1>{title}</h1><p>{description}</p></div>{active === "Pipelines" && <button className="button-primary" onClick={onNewPipeline}><Icon name="plus" /> Run pipeline</button>}</div>{active === "Pipelines" ? <article className="panel workspace-pipelines"><PanelHeading eyebrow={mode === "live" ? "RUNNER API" : "SAMPLE DATA"} title={`${pipelines.length} pipeline runs`} action={mode === "live" ? "Connected" : "Demo mode"} /><PipelineTable pipelines={pipelines} onSelect={onSelect} /></article> : <div className="workspace-placeholder panel"><span className="workspace-placeholder-icon"><Icon name={sidebar.find(([name]) => name === active)?.[1] || "grid"} size={25} /></span><h2>{title} workspace</h2><p>{description} Connect AtonixCorp services to populate this view with live data.</p><button className="text-action" type="button" onClick={onNewPipeline}>Open pipeline control <Icon name="arrow" size={14} /></button></div>}</section>;
}

export default function App() {
  const [active, setActive] = useState("Dashboard");
  const [theme, setTheme] = useState(getInitialTheme);
  const [mode, setMode] = useState("demo");
  const [pipelines, setPipelines] = useState(initialPipelines);
  const [runnerSummary, setRunnerSummary] = useState({});
  const [runnerError, setRunnerError] = useState("");
  const [refreshing, setRefreshing] = useState(false);
  const [query, setQuery] = useState("");
  const [modalOpen, setModalOpen] = useState(false);
  const [creating, setCreating] = useState(false);
  const [selectedPipeline, setSelectedPipeline] = useState(null);
  const [logs, setLogs] = useState("");
  const [logsError, setLogsError] = useState("");
  const [toast, setToast] = useState("");

  const refresh = useCallback(async () => {
    setRefreshing(true);
    setRunnerError("");
    try {
      const runner = await getRunnerState();
      setPipelines(runner.pipelines);
      setRunnerSummary(runner.summary);
      setMode("live");
      setToast("Connected to the DevSecOps runner.");
    } catch (error) {
      setRunnerError(error.message || "Could not connect to the runner.");
      setMode("demo");
      setPipelines(initialPipelines);
      setRunnerSummary({});
      setToast("Runner unavailable. Showing the interactive demo dashboard.");
    } finally {
      setRefreshing(false);
    }
  }, []);

  useEffect(() => {
    if (mode !== "live") return undefined;
    const timer = window.setInterval(async () => {
      try {
        const runner = await getRunnerState();
        setPipelines(runner.pipelines);
        setRunnerSummary(runner.summary);
        setRunnerError("");
      } catch (error) {
        setRunnerError(error.message || "Runner connection was lost.");
        setPipelines(initialPipelines);
        setRunnerSummary({});
        setMode("demo");
      }
    }, 5000);
    return () => window.clearInterval(timer);
  }, [mode]);

  useEffect(() => {
    if (!toast) return undefined;
    const timer = window.setTimeout(() => setToast(""), 3500);
    return () => window.clearTimeout(timer);
  }, [toast]);

  useEffect(() => {
    const handleKeys = (event) => {
      if (event.key === "Escape") {
        setModalOpen(false);
        setSelectedPipeline(null);
      }
      if ((event.metaKey || event.ctrlKey) && event.key.toLowerCase() === "k") {
        event.preventDefault();
        document.getElementById("devops-search")?.focus();
      }
    };
    window.addEventListener("keydown", handleKeys);
    return () => window.removeEventListener("keydown", handleKeys);
  }, []);

  const filteredPipelines = useMemo(() => pipelines.filter((pipeline) => `${pipeline.project} ${pipeline.ref} ${pipeline.status}`.toLowerCase().includes(query.toLowerCase())), [pipelines, query]);

  async function createRun({ project, ref, workflow }) {
    setCreating(true);
    try {
      if (mode === "live") {
        const pipeline = await startPipeline({ project, ref, workflow });
        setPipelines((items) => [pipeline, ...items]);
        setToast(`Pipeline started for ${project}.`);
      } else {
        const demoPipeline = {
          id: `demo-${Date.now()}`,
          project,
          ref,
          status: "success",
          trigger: "manual",
          started_at: new Date().toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" }),
          labels: ["demo", workflow],
          demo: true,
          jobs: [
            { id: `demo-${Date.now()}-1`, name: "Build & package", stage: "build", status: "success" },
            { id: `demo-${Date.now()}-2`, name: "Security gates", stage: "security", status: "success" },
            { id: `demo-${Date.now()}-3`, name: "Test & approve", stage: "test", status: "success" },
            { id: `demo-${Date.now()}-4`, name: "Deploy & observe", stage: "deploy", status: "success" },
          ],
        };
        setPipelines((items) => [demoPipeline, ...items]);
        setToast(`Demo pipeline created for ${project}; no jobs were executed.`);
      }
      setModalOpen(false);
    } catch (error) {
      throw new Error(error.message || "Could not start the pipeline.");
    } finally {
      setCreating(false);
    }
  }

  async function loadLogs(endpoint) {
    setLogs("");
    setLogsError("");
    try {
      setLogs(await getJobLogs(endpoint));
    } catch (error) {
      setLogsError(error.message || "Could not load job logs.");
    }
  }

  const recentPipelines = filteredPipelines;

  return (
    <div className="devops-app" data-theme={theme}>
      <Header mode={mode} theme={theme} onThemeToggle={() => {
        const nextTheme = theme === "dark" ? "light" : "dark";
        setTheme(nextTheme);
        try {
          window.localStorage.setItem("atonixcorp-devops-theme", nextTheme);
        } catch {
          setToast("Theme changed, but this browser could not save your preference.");
        }
      }} onModeChange={mode === "live" ? () => { setPipelines(initialPipelines); setRunnerSummary({}); setMode("demo"); setToast("Returned to demo mode."); } : refresh} onRefresh={refresh} refreshing={refreshing} error={runnerError} query={query} onQueryChange={setQuery} onNewPipeline={() => setModalOpen(true)} />
      <div className="app-shell">
        <Sidebar active={active} onSelect={setActive} pipelineCount={pipelines.length} />
        <main className="main-content">
          {active === "Dashboard" ? <DashboardOverview pipelines={recentPipelines} summary={runnerSummary} mode={mode} query={query} onSelect={setSelectedPipeline} onNewPipeline={() => setModalOpen(true)} onNavigate={setActive} /> : <ServiceWorkspace active={active} pipelines={recentPipelines} onSelect={setSelectedPipeline} onNewPipeline={() => setModalOpen(true)} mode={mode} />}
          <footer className="app-footer"><span>© 2026 AtonixCorp · DevSecOps Pipeline</span><span>{mode === "live" ? "Runner API connected" : "Interactive prototype · Demo data"}</span><span>Build · Innovate · Scale · A More Connected Tomorrow</span></footer>
        </main>
      </div>
      {modalOpen && <PipelineModal onClose={() => setModalOpen(false)} onCreate={createRun} mode={mode} loading={creating} />}
      {selectedPipeline && <PipelineDetails pipeline={selectedPipeline} onClose={() => { setSelectedPipeline(null); setLogs(""); setLogsError(""); }} mode={mode} onLoadLogs={loadLogs} logs={logs} logsError={logsError} />}
      {toast && <div className="toast" role="status">{toast}</div>}
    </div>
  );
}
