"use client";

import Image from "next/image";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { FormEvent, useEffect, useMemo, useState } from "react";
import { useCloudTheme } from "@/app/shared/lib/cloud-theme";
import "./dashboard.css";

type Service = {
  name: string;
  icon: string;
  kind: string;
};

type CloudResource = {
  id: string;
  name: string;
  kind: string;
  status: string;
  region: string;
  created: string;
};

type CloudActivity = {
  time: string;
  action: string;
  resource: string;
  status: string;
  user: string;
};

type AgentModel = {
  name: string;
  version?: string;
  stage?: string;
  production_version?: string;
  metadata?: {
    family?: string;
  };
  versions?: Record<string, {
    version: string;
    stage?: string;
  }>;
};

type AgentRegistry = {
  models?: AgentModel[];
};

const navigation: Service[] = [
  { name: "Home", icon: "home", kind: "home" },
  { name: "Compute", icon: "compute", kind: "Virtual machine" },
  { name: "Containers", icon: "containers", kind: "Kubernetes cluster" },
  { name: "Storage", icon: "storage", kind: "Storage bucket" },
  { name: "Databases", icon: "database", kind: "Database" },
  { name: "Networking", icon: "network", kind: "Network" },
  { name: "Security", icon: "security", kind: "Security policy" },
  { name: "AI & Machine Learning", icon: "ai", kind: "AI workspace" },
  { name: "Observability", icon: "observability", kind: "Observability" },
  { name: "Billing", icon: "billing", kind: "Billing" },
];

const resourceTypes = [
  { label: "Virtual machine", icon: "compute" },
  { label: "Kubernetes cluster", icon: "containers" },
  { label: "Storage bucket", icon: "storage" },
  { label: "Database", icon: "database" },
  { label: "Network", icon: "network" },
  { label: "AI workspace", icon: "ai" },
  { label: "Security policy", icon: "security" },
];

const resourceSummary = [
  { name: "Virtual machines", kind: "Virtual machine", count: 12, delta: "+2 this week", icon: "compute", tone: "blue" },
  { name: "Kubernetes clusters", kind: "Kubernetes cluster", count: 4, delta: "+1 this week", icon: "containers", tone: "purple" },
  { name: "Storage buckets", kind: "Storage bucket", count: 18, delta: "+3 this week", icon: "storage", tone: "green" },
  { name: "Databases", kind: "Database", count: 6, delta: "+1 this week", icon: "database", tone: "blue" },
];

const agentModelFamilies = [
  { name: "Anomaly detection", family: "anomaly", description: "Flag unusual workload and infrastructure signals.", icon: "observability" },
  { name: "Autoscaling", family: "autoscale", description: "Recommend scale-up, hold, or scale-down from utilization.", icon: "containers" },
  { name: "Forecasting", family: "forecasting", description: "Project future resource usage from observed time series.", icon: "chart" },
  { name: "Resource optimization", family: "resource_opt", description: "Recommend resource sizing from observed workload usage.", icon: "compute" },
];

const initialActivities: CloudActivity[] = [
  { time: "Nov 30, 19:14", action: "Launched instance", resource: "web-prod-01", status: "Success", user: "j.davis" },
  { time: "Nov 30, 17:42", action: "Updated cluster", resource: "prod-cluster", status: "Success", user: "k.mokoena" },
  { time: "Nov 30, 15:28", action: "Created storage bucket", resource: "analytics-data", status: "Success", user: "s.naidoo" },
  { time: "Nov 30, 13:11", action: "Modified security group", resource: "sg-0a3f2e7c", status: "Success", user: "j.davis" },
  { time: "Nov 30, 11:06", action: "Created database", resource: "customer-db", status: "Success", user: "l.chen" },
  { time: "Nov 30, 09:23", action: "Updated IAM policy", resource: "admin-access", status: "Success", user: "t.morales" },
];

const regions = [
  "North America · us-west",
  "Europe · eu-central",
  "Asia Pacific · ap-southeast",
  "South America · sa-east",
];
function DashboardIcon({ name }: { name: string }) {
  const paths: Record<string, string> = {
    home: "M3 10.5 12 3l9 7.5v9a1.5 1.5 0 0 1-1.5 1.5h-15A1.5 1.5 0 0 1 3 19.5v-9ZM9 21v-7h6v7",
    compute: "M8 3v3m8-3v3M8 18v3m8-3v3M3 8h3m-3 8h3m12-8h3m-3 8h3M7 6h10a1 1 0 0 1 1 1v10a1 1 0 0 1-1 1H7a1 1 0 0 1-1-1V7a1 1 0 0 1 1-1Z",
    containers: "m12 3 9 5-9 5-9-5 9-5Zm-9 9 9 5 9-5M3 16l9 5 9-5",
    storage: "M4 5c0-1.1 3.6-2 8-2s8 .9 8 2-3.6 2-8 2-8-.9-8-2Zm0 0v14c0 1.1 3.6 2 8 2s8-.9 8-2V5M4 12c0 1.1 3.6 2 8 2s8-.9 8-2",
    database: "M4 5c0-1.1 3.6-2 8-2s8 .9 8 2-3.6 2-8 2-8-.9-8-2Zm0 0v14c0 1.1 3.6 2 8 2s8-.9 8-2V5M4 12c0 1.1 3.6 2 8 2s8-.9 8-2",
    network: "M12 12V4m0 8-7 5m7-5 7 5M12 3a2 2 0 1 0 0 4 2 2 0 0 0 0-4ZM5 15a2 2 0 1 0 0 4 2 2 0 0 0 0-4Zm14 0a2 2 0 1 0 0 4 2 2 0 0 0 0-4Z",
    security: "M12 22s8-4 8-11V5l-8-3-8 3v6c0 7 8 11 8 11Zm-3-11 2 2 4-4",
    ai: "M9 3a3 3 0 0 0-3 3v1a3 3 0 0 0-2 3 3 3 0 0 0 1 2.2A3.5 3.5 0 0 0 7 19h2m6-16a3 3 0 0 1 3 3v1a3 3 0 0 1 2 3 3 3 0 0 1-1 2.2A3.5 3.5 0 0 1 17 19h-2m-3-16v18m-5-9h5m2 0h5m-10 4h3m2-8h3",
    observability: "M3 3v18h18M7 14l4-4 4 3 6-7",
    billing: "M3 6h18v12H3zM3 10h18m-14 4h4",
    globe: "M21 12a9 9 0 1 1-18 0 9 9 0 0 1 18 0ZM3.6 9h16.8M3.6 15h16.8M12 3a15 15 0 0 1 0 18 15 15 0 0 1 0-18Z",
    bell: "M18 8a6 6 0 0 0-12 0c0 7-3 7-3 9h18c0-2-3-2-3-9Zm-8 12a2 2 0 0 0 4 0",
    help: "M12 18h.01M9.1 9a3 3 0 1 1 5.2 2c-1.2 1.1-2.3 1.4-2.3 3m9-2a9 9 0 1 1-18 0 9 9 0 0 1 18 0Z",
    search: "m20 20-4.2-4.2M18 10.5a7.5 7.5 0 1 1-15 0 7.5 7.5 0 0 1 15 0Z",
    plus: "M12 5v14m-7-7h14",
    moon: "M20.5 14.5A8.5 8.5 0 0 1 9.5 3.5a8.5 8.5 0 1 0 11 11Z",
    sun: "M12 3v2m0 14v2M5.64 5.64l1.42 1.42m9.88 9.88 1.42 1.42M3 12h2m14 0h2M5.64 18.36l1.42-1.42m9.88-9.88 1.42-1.42M16 12a4 4 0 1 1-8 0 4 4 0 0 1 8 0Z",
  };

  return (
    <svg aria-hidden="true" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round">
      <path d={paths[name] ?? paths.home} />
    </svg>
  );
}

function ResourceCard({
  item,
  count,
  onClick,
}: {
  item: (typeof resourceSummary)[number];
  count: number;
  onClick: () => void;
}) {
  return (
    <button className="cloud-resource" type="button" onClick={onClick} aria-label={`Open ${item.name}`}>
      <span className={`cloud-resource-icon ${item.tone}`}><DashboardIcon name={item.icon} /></span>
      <strong>{count}</strong>
      <span className="cloud-resource-name">{item.name}</span>
      <span className="cloud-resource-delta">{item.delta}</span>
    </button>
  );
}

export default function DashboardPage() {
  const router = useRouter();
  const [theme, toggleTheme] = useCloudTheme();
  const [selectedService, setSelectedService] = useState("Home");
  const [resources, setResources] = useState<CloudResource[]>([]);
  const [activities, setActivities] = useState(initialActivities);
  const [region, setRegion] = useState(regions[0]);
  const [query, setQuery] = useState("");
  const [createOpen, setCreateOpen] = useState(false);
  const [createType, setCreateType] = useState(resourceTypes[0].label);
  const [createName, setCreateName] = useState("");
  const [notificationOpen, setNotificationOpen] = useState(false);
  const [profileOpen, setProfileOpen] = useState(false);
  const [toast, setToast] = useState("");
  const [userEmail, setUserEmail] = useState("");
  const [authStatus, setAuthStatus] = useState<"loading" | "authenticated" | "unauthenticated" | "unavailable">("loading");
  const [agentModels, setAgentModels] = useState<AgentModel[]>([]);
  const [agentError, setAgentError] = useState("");
  const [agentConnected, setAgentConnected] = useState(false);
  const [agentLoading, setAgentLoading] = useState(true);

  useEffect(() => {
    let active = true;
    const refreshAgent = async () => {
      try {
        const response = await fetch("/api/agent/registry", { cache: "no-store" });
        if (!response.ok) {
          const error = await response.json().catch(() => ({}));
          throw new Error(error.error || `Agent API returned ${response.status}.`);
        }
        const registry = (await response.json()) as AgentRegistry;
        if (!active) return;
        setAgentModels(Array.isArray(registry.models) ? registry.models : []);
        setAgentError("");
        setAgentConnected(true);
      } catch (error) {
        if (!active) return;
        setAgentModels([]);
        setAgentError(error instanceof Error ? error.message : "Could not connect to the Agent API.");
        setAgentConnected(false);
      } finally {
        if (active) setAgentLoading(false);
      }
    };
    void refreshAgent();
    const timer = window.setInterval(refreshAgent, 30000);
    return () => {
      active = false;
      window.clearInterval(timer);
    };
  }, []);

  useEffect(() => {
    const frame = window.requestAnimationFrame(() => {
      void fetch("/api/auth/session", { cache: "no-store" }).then(async (response) => {
        const payload = await response.json().catch(() => null) as {
          user?: { email?: string };
        } | null;
        if (response.status === 401) {
          setAuthStatus("unauthenticated");
          router.replace("/login");
          return;
        }
        if (!response.ok || !payload?.user?.email) {
          setAuthStatus("unavailable");
          return;
        }
        setUserEmail(payload.user.email);
        setAuthStatus("authenticated");
        try {
          const saved = localStorage.getItem("atonixcorp-demo-resources");
          if (saved) setResources(JSON.parse(saved) as CloudResource[]);
        } catch {
          setToast("Saved dashboard resources could not be loaded from this browser.");
        }
      }).catch(() => setAuthStatus("unavailable"));
    });
    return () => window.cancelAnimationFrame(frame);
  }, [router]);

  useEffect(() => {
    const handleShortcut = (event: KeyboardEvent) => {
      if ((event.metaKey || event.ctrlKey) && event.key.toLowerCase() === "k") {
        event.preventDefault();
        document.getElementById("cloud-global-search")?.focus();
      }
      if (event.key === "Escape") {
        setCreateOpen(false);
        setNotificationOpen(false);
        setProfileOpen(false);
      }
    };
    window.addEventListener("keydown", handleShortcut);
    return () => window.removeEventListener("keydown", handleShortcut);
  }, []);

  useEffect(() => {
    if (!toast) return;
    const timeout = window.setTimeout(() => setToast(""), 3500);
    return () => window.clearTimeout(timeout);
  }, [toast]);

  const selected = navigation.find((item) => item.name === selectedService) ?? navigation[0];
  const productionModels = agentModels.filter((model) => model.production_version);
  const filteredResources = useMemo(() => {
    const serviceItems = resources.filter((resource) =>
      selected.kind === "home" || selected.kind === "Observability" || selected.kind === "Billing"
        ? true
        : resource.kind === selected.kind,
    );
    if (!query.trim()) return serviceItems;
    const search = query.trim().toLowerCase();
    return serviceItems.filter((resource) =>
      `${resource.name} ${resource.kind} ${resource.region}`.toLowerCase().includes(search),
    );
  }, [query, resources, selected.kind]);
  const searchMatches = query.trim()
    ? navigation.filter((item) => item.name.toLowerCase().includes(query.trim().toLowerCase()))
    : [];
  const resourceMatches = query.trim()
    ? resources.filter((resource) => `${resource.name} ${resource.kind}`.toLowerCase().includes(query.trim().toLowerCase()))
    : [];

  const openCreate = (kind?: string) => {
    const availableType = resourceTypes.find((type) => type.label === kind);
    setCreateType(availableType?.label ?? resourceTypes[0].label);
    setCreateName("");
    setCreateOpen(true);
  };

  const createResource = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    const trimmedName = createName.trim();
    if (!trimmedName) {
      setToast("Enter a name for this resource.");
      return;
    }
    const resource: CloudResource = {
      id: `demo-${Date.now()}`,
      name: trimmedName,
      kind: createType,
      status: "Ready (demo)",
      region,
      created: new Date().toLocaleString(),
    };
    const updated = [resource, ...resources];
    setResources(updated);
    let storageUnavailable = false;
    try {
      localStorage.setItem("atonixcorp-demo-resources", JSON.stringify(updated));
    } catch {
      storageUnavailable = true;
    }
    const activity: CloudActivity = {
      time: new Date().toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" }),
      action: `Created ${createType.toLowerCase()}`,
      resource: trimmedName,
      status: "Demo",
      user: userEmail.split("@")[0],
    };
    setActivities((items) => [activity, ...items]);
    setCreateOpen(false);
    setSelectedService(navigation.find((item) => item.kind === createType)?.name ?? "Home");
    setToast(storageUnavailable
      ? `${trimmedName} added for this session, but browser storage is unavailable.`
      : `${trimmedName} added to your demo workspace.`);
  };

  const signOut = async () => {
    await fetch("/api/auth/logout", { method: "POST" });
    router.push("/login");
  };

  const suggestedAction = selected.kind === "home" ? resourceTypes[0].label : selected.kind;
  const canCreate = resourceTypes.some((item) => item.label === suggestedAction);

  if (authStatus !== "authenticated") {
    return (
      <main className="cloud-dashboard">
        <a className="skip-link" href="#cloud-account-state">Skip to main content</a>
        <section className="cloud-empty-state" role={authStatus === "unavailable" ? "alert" : undefined}>
          <h1 id="cloud-account-state" tabIndex={-1}>{authStatus === "unavailable" ? "Account service unavailable" : "Checking your account…"}</h1>
          <p>{authStatus === "unavailable"
            ? "The dashboard requires an authenticated account session. Please retry after the account service is available."
            : "Your authenticated session is being verified."}</p>
          {authStatus === "unavailable" && <button type="button" onClick={() => window.location.reload()}>Retry</button>}
        </section>
      </main>
    );
  }

  return (
    <main className="cloud-dashboard" data-theme={theme}>
      <a className="skip-link" href="#cloud-content">Skip to main content</a>
      <header className="cloud-topbar">
        <Link className="cloud-brand" href="/dashboard" aria-label="AtonixCorp Cloud dashboard" onClick={() => setSelectedService("Home")}>
          <Image src="/atonixcorp-icon.png" alt="" width={42} height={42} priority />
          <span><strong>AtonixCorp Cloud</strong><small>BUILD SCALE SECURE TOGETHER</small></span>
        </Link>
        <div className="cloud-search-wrap">
          <label className="cloud-search">
            <DashboardIcon name="search" />
            <input
              type="search"
              id="cloud-global-search"
              aria-label="Search services and resources"
              placeholder="Search services, resources, and documentation"
              value={query}
              onChange={(event) => setQuery(event.target.value)}
              onKeyDown={(event) => {
                if (event.key === "Escape") setQuery("");
                if (event.key === "Enter" && searchMatches[0]) {
                  if (searchMatches[0].name === "Observability") {
                    router.push("/observability");
                    setQuery("");
                    return;
                  }
                  setSelectedService(searchMatches[0].name);
                  setQuery("");
                }
              }}
            />
            {query && <button type="button" aria-label="Clear search" onClick={() => setQuery("")}>×</button>}
            <kbd>⌘ K</kbd>
          </label>
          {query && (
            <div className="cloud-search-results" role="listbox" aria-label="Search results">
              {searchMatches.map((item) => (
                <button key={item.name} type="button" onClick={() => {
                  setQuery("");
                  if (item.name === "Compute") {
                    router.push("/compute");
                    return;
                  }
                  if (item.name === "Containers") {
                    router.push("/containers");
                    return;
                  }
                  if (item.name === "Storage") {
                    router.push("/storage");
                    return;
                  }
                  if (item.name === "Databases") {
                    router.push("/cloud/databases");
                    return;
                  }
                  if (item.name === "Networking") {
                    router.push("/networking");
                    return;
                  }
                  if (item.name === "Observability") {
                    router.push("/observability");
                    return;
                  }
                  if (item.name === "Billing") {
                    router.push("/billing");
                    return;
                  }
                  setSelectedService(item.name);
                }}>
                  <DashboardIcon name={item.icon} /><span>{item.name}</span><small>Service</small>
                </button>
              ))}
              {resourceMatches.map((resource) => (
                <button key={resource.id} type="button" onClick={() => {
                  if (resource.kind === "Virtual machine") {
                    setQuery("");
                    router.push("/compute/instances");
                    return;
                  }
                  if (resource.kind === "Storage bucket") {
                    setQuery("");
                    router.push("/storage");
                    return;
                  }
                  setSelectedService(navigation.find((item) => item.kind === resource.kind)?.name ?? "Home");
                  setQuery(resource.name);
                }}>
                  <DashboardIcon name={resourceTypes.find((item) => item.label === resource.kind)?.icon ?? "compute"} /><span>{resource.name}</span><small>{resource.kind}</small>
                </button>
              ))}
              {!searchMatches.length && !resourceMatches.length && <p>No matching services or demo resources.</p>}
            </div>
          )}
        </div>
        <div className="cloud-top-actions">
          <button
            className="cloud-icon-button cloud-theme-toggle"
            type="button"
            aria-label={`Switch to ${theme === "dark" ? "light" : "dark"} mode`}
            title={`Switch to ${theme === "dark" ? "light" : "dark"} mode`}
            onClick={toggleTheme}
          >
            <DashboardIcon name={theme === "dark" ? "sun" : "moon"} />
          </button>
          <label className="cloud-region">
            <DashboardIcon name="globe" />
            <select aria-label="Select region" value={region} onChange={(event) => setRegion(event.target.value)}>
              {regions.map((item) => <option key={item}>{item}</option>)}
            </select>
          </label>
          <div className="cloud-popover-wrap">
            <button className="cloud-icon-button" type="button" aria-label="Notifications" aria-expanded={notificationOpen} onClick={() => { setNotificationOpen(!notificationOpen); setProfileOpen(false); }}>
              <DashboardIcon name="bell" /><i />
            </button>
            {notificationOpen && (
              <div className="cloud-popover cloud-notification-popover">
                <strong>Recent activity</strong>
                <p>{activities[0]?.action ?? "No recent activity"}</p>
                <small>{activities[0]?.resource ?? "Your demo workspace is ready"}</small>
                <button type="button" onClick={() => {
                  setNotificationOpen(false);
                  setSelectedService("Home");
                  window.setTimeout(() => document.getElementById("activity")?.scrollIntoView({ behavior: "smooth" }), 0);
                }}>View activity</button>
              </div>
            )}
          </div>
          <Link className="cloud-icon-button" href="/support" aria-label="Help and support"><DashboardIcon name="help" /></Link>
          <span className="cloud-top-divider" />
          <div className="cloud-popover-wrap">
            <button className="cloud-profile" type="button" aria-label="Open account menu" aria-expanded={profileOpen} onClick={() => { setProfileOpen(!profileOpen); setNotificationOpen(false); }}>
              <span className="cloud-avatar">{userEmail.slice(0, 2).toUpperCase()}</span>
              <span><strong>{userEmail.split("@")[0]}</strong><small>Account</small></span>
              <span className="cloud-chevron">⌄</span>
            </button>
            {profileOpen && (
              <div className="cloud-popover cloud-profile-popover">
                <strong>{userEmail}</strong>
                <small>Authenticated account</small>
                <button type="button" onClick={signOut}>Sign out</button>
              </div>
            )}
          </div>
        </div>
      </header>

      <div className="cloud-app-layout">
        <aside className="cloud-sidebar" aria-label="Cloud services">
          <nav>
            {navigation.map((item) => (
              item.name === "Compute" ? (
                <Link
                  key={item.name}
                  className="cloud-nav-item"
                  href="/compute"
                >
                  <DashboardIcon name={item.icon} /><span>{item.name}</span>
                </Link>
              ) : item.name === "Containers" ? (
                <Link
                  key={item.name}
                  className="cloud-nav-item"
                  href="/containers"
                >
                  <DashboardIcon name={item.icon} /><span>{item.name}</span>
                </Link>
              ) : item.name === "Storage" ? (
                <Link
                  key={item.name}
                  className="cloud-nav-item"
                  href="/storage"
                >
                  <DashboardIcon name={item.icon} /><span>{item.name}</span>
                </Link>
              ) : item.name === "Databases" ? (
                <Link
                  key={item.name}
                  className="cloud-nav-item"
                  href="/cloud/databases"
                >
                  <DashboardIcon name={item.icon} /><span>{item.name}</span>
                </Link>
              ) : item.name === "Networking" ? (
                <Link
                  key={item.name}
                  className="cloud-nav-item"
                  href="/networking"
                >
                  <DashboardIcon name={item.icon} /><span>{item.name}</span>
                </Link>
              ) : item.name === "Security" ? (
                <Link
                  key={item.name}
                  className="cloud-nav-item"
                  href="/security"
                >
                  <DashboardIcon name={item.icon} /><span>{item.name}</span>
                </Link>
              ) : item.name === "Observability" ? (
                <Link
                  key={item.name}
                  className="cloud-nav-item"
                  href="/observability"
                >
                  <DashboardIcon name={item.icon} /><span>{item.name}</span>
                </Link>
              ) : item.name === "Billing" ? (
                <Link
                  key={item.name}
                  className="cloud-nav-item"
                  href="/billing"
                >
                  <DashboardIcon name={item.icon} /><span>{item.name}</span>
                </Link>
              ) : (
                <button
                  key={item.name}
                  className={`cloud-nav-item${selectedService === item.name ? " active" : ""}`}
                  type="button"
                  aria-current={selectedService === item.name ? "page" : undefined}
                  onClick={() => { setSelectedService(item.name); setQuery(""); }}
                >
                  <DashboardIcon name={item.icon} /><span>{item.name}</span>
                </button>
              )
            ))}
            <Link className="cloud-nav-item" href="/support"><DashboardIcon name="help" /><span>Support</span></Link>
          </nav>
          <div className="cloud-sidebar-status">
            <span className="cloud-status-dot" />
            <div><strong>Demo mode</strong><small>Live connection not configured</small></div>
            <span aria-hidden="true">›</span>
          </div>
          <div className="cloud-sidebar-foot">
            <strong>AtonixCorp Cloud</strong>
            <span>Dashboard prototype</span>
            <small>Sample data · Not connected to live services</small>
          </div>
        </aside>

        <div className="cloud-main">
          <div className="cloud-content" id="cloud-content" tabIndex={-1}>
            <div className="cloud-page-heading" id="overview">
              <div>
                <p className="cloud-eyebrow">{selectedService === "Home" ? "WELCOME TO ATONIXCORP CLOUD" : `ATONIXCORP CLOUD / ${selectedService.toUpperCase()}`}</p>
                <h1>{selectedService === "Home" ? "Cloud overview" : selectedService}</h1>
                <p>{selectedService === "Home" ? "Monitor and manage your cloud workspace from one place." : selectedService === "AI & Machine Learning" ? "AI & Machine Learning workspace in your AtonixCorp demo environment." : `${selectedService} workspace in your AtonixCorp demo environment.`}</p>
              </div>
              <div className="cloud-heading-actions">
                {selectedService !== "AI & Machine Learning" && selectedService !== "Billing" && (
                  <button className="cloud-create-button" type="button" onClick={() => openCreate(canCreate ? suggestedAction : undefined)}>
                    <DashboardIcon name="plus" /> Create resource <span>⌄</span>
                  </button>
                )}
                <button className="cloud-services-button" type="button" onClick={() => { setSelectedService("Home"); setQuery(""); }}>
                  <DashboardIcon name="containers" /> View overview
                </button>
              </div>
            </div>

            <div className="cloud-demo-notice">
              <span aria-hidden="true">i</span>
              <p><strong>Interactive demo</strong> — created resources are saved in this browser only. No real cloud resources are provisioned.</p>
              <Link href="/platform">About the platform <span aria-hidden="true">→</span></Link>
            </div>

            {selectedService === "Home" ? (
              <section className="cloud-overview-grid" aria-label="Cloud status overview">
                <article className="cloud-panel cloud-resource-panel" id="resources">
                  <header className="cloud-panel-heading">
                    <div><h2>Resource summary</h2><p>Sample resources across all regions</p></div>
                    <Link href="/compute">View services <span aria-hidden="true">→</span></Link>
                  </header>
                  <div className="cloud-resource-grid">
                    {resourceSummary.map((item) => (
                      <ResourceCard key={item.kind} item={item} count={item.count + resources.filter((resource) => resource.kind === item.kind).length} onClick={() => {
                        if (item.kind === "Virtual machine") {
                          router.push("/compute/instances");
                          return;
                        }
                        setSelectedService(navigation.find((service) => service.kind === item.kind)?.name ?? "Home");
                      }} />
                    ))}
                  </div>
                </article>

                <article className="cloud-panel cloud-health-panel">
                  <header className="cloud-panel-heading"><div><h2>System health</h2><p>Demo status</p></div><Link href="/observability">Overview <span aria-hidden="true">→</span></Link></header>
                  <div className="cloud-health-content">
                    <span className="cloud-health-check" aria-hidden="true">i</span>
                    <div><strong>Cloud connection not configured</strong><p>Connect an environment to see live health and availability.</p></div>
                  </div>
                  <div className="cloud-uptime"><strong>—</strong><span>Uptime metrics appear when connected</span></div>
                </article>

                <article className="cloud-panel cloud-agent-panel">
                  <header className="cloud-panel-heading"><div><h2>AtonixCorp Agent</h2><p>Model readiness · advisory only</p></div><span className={`cloud-agent-state${!agentConnected && !agentLoading ? " unavailable" : ""}`}>{agentLoading ? "Connecting" : agentConnected ? "Connected" : "Unavailable"}</span></header>
                  {agentLoading ? (
                    <div className="cloud-agent-body"><strong>Checking Agent API</strong><p>Loading production model status.</p></div>
                  ) : agentError ? (
                    <div className="cloud-agent-body"><strong>Agent API unavailable</strong><p>{agentError}</p></div>
                  ) : (
                    <div className="cloud-agent-body">
                      <strong>{agentModels.filter((model) => model.production_version).length} production models</strong>
                      <p>
                        {agentModels.some((model) => model.production_version)
                          ? "Waiting for live Cloud telemetry. The selected database metrics API currently returns placeholders, so no predictions are generated."
                          : "Register and promote production models. Demo resources and placeholder metrics are not used for predictions."}
                      </p>
                    </div>
                  )}
                </article>

                <article className="cloud-panel cloud-security-panel">
                  <header className="cloud-panel-heading"><div><h2>Security posture</h2><p>Workspace overview</p></div><button type="button" onClick={() => router.push("/security")}>View insights <span aria-hidden="true">→</span></button></header>
                  <div className="cloud-security-content">
                    <div className="cloud-security-ring"><span>—</span></div>
                    <div><strong>Awaiting connection</strong><p>Security insights will appear when your environment is connected.</p></div>
                  </div>
                  <div className="cloud-security-footer"><span aria-hidden="true">!</span><div><strong>Security recommendations</strong><p>Connect a workspace to review recommendations.</p></div></div>
                </article>

                <article className="cloud-panel cloud-cost-panel">
                  <header className="cloud-panel-heading"><div><h2>Cost overview</h2><p>Sample trend · not a bill</p></div><button type="button" onClick={() => router.push("/billing")}>View billing <span aria-hidden="true">→</span></button></header>
                  <div className="cloud-cost-content">
                    <div className="cloud-cost-total"><span>Current month</span><strong>—</strong><small>Cost data unavailable</small></div>
                    <div className="cloud-chart" aria-label="Illustrative cost trend chart">
                      <div className="cloud-chart-ylabels"><span>R 40K</span><span>R 30K</span><span>R 20K</span><span>R 10K</span><span>R 0</span></div>
                      <svg viewBox="0 0 400 138" role="img" aria-label="Example trend only, without real billing data">
                        <defs><linearGradient id="cloudChartFill" x1="0" x2="0" y1="0" y2="1"><stop offset="0%" stopColor="#00af9b" stopOpacity=".24" /><stop offset="100%" stopColor="#00af9b" stopOpacity=".02" /></linearGradient></defs>
                        <path className="cloud-chart-gridline" d="M0 12H400M0 42H400M0 72H400M0 102H400M0 132H400M50 0V132M120 0V132M190 0V132M260 0V132M330 0V132M399 0V132" />
                        <path className="cloud-chart-area" d="M0 113 C20 104 25 88 47 79 S76 72 96 63 S127 76 146 83 S171 101 193 87 S225 60 247 70 S275 64 295 50 S322 24 343 31 S367 58 383 52 S395 54 400 57 L400 132 L0 132Z" />
                        <path className="cloud-chart-line" d="M0 113 C20 104 25 88 47 79 S76 72 96 63 S127 76 146 83 S171 101 193 87 S225 60 247 70 S275 64 295 50 S322 24 343 31 S367 58 383 52 S395 54 400 57" />
                      </svg>
                      <div className="cloud-chart-xlabels"><span>Nov 1</span><span>Nov 8</span><span>Nov 15</span><span>Nov 22</span><span>Nov 30</span></div>
                    </div>
                  </div>
                </article>

                <article className="cloud-panel cloud-quick-panel">
                  <header className="cloud-panel-heading"><div><h2>Quick actions</h2><p>Common tasks · demo only</p></div></header>
                  <div className="cloud-quick-grid">
                    <Link href="/compute/deploy"><DashboardIcon name="compute" /><span><strong>Launch instance</strong><small>Configure a virtual machine</small></span><b>›</b></Link>
                    <Link href="/containers/deploy"><DashboardIcon name="containers" /><span><strong>Deploy container</strong><small>Launch an app on Kubernetes</small></span><b>›</b></Link>
                    <Link href="/cloud/databases"><DashboardIcon name="database" /><span><strong>Add database</strong><small>Set up a managed database</small></span><b>›</b></Link>
                    <Link href="/networking"><DashboardIcon name="network" /><span><strong>Configure network</strong><small>VPCs and address ranges</small></span><b>›</b></Link>
                  </div>
                </article>

                <article className="cloud-panel cloud-activity-panel" id="activity">
                  <header className="cloud-panel-heading"><div><h2>Recent activity</h2><p>Example entries and your demo activity</p></div><button type="button" onClick={() => document.getElementById("activity")?.scrollIntoView({ behavior: "smooth" })}>All activity <span aria-hidden="true">→</span></button></header>
                  <ActivityTable activities={activities} />
                </article>

                <article className="cloud-panel cloud-regions-panel">
                  <header className="cloud-panel-heading"><div><h2>Regions & availability</h2><p>Illustrative deployment footprint</p></div><button type="button" onClick={() => setRegion(regions[(regions.indexOf(region) + 1) % regions.length])}>Change region <span aria-hidden="true">→</span></button></header>
                  <RegionMap />
                </article>
              </section>
            ) : selectedService === "AI & Machine Learning" ? (
              <section className="cloud-ai-workspace" aria-label="AI and Machine Learning workspace">
                <article className={`cloud-panel cloud-ai-status${agentError ? " unavailable" : ""}`}>
                  <span className="cloud-ai-status-icon"><DashboardIcon name={agentLoading ? "ai" : agentError ? "security" : "ai"} /></span>
                  <div>
                    <strong>{agentLoading ? "Connecting to AtonixCorp Agent" : agentError ? "Agent connection unavailable" : "AtonixCorp Agent connected"}</strong>
                    <p>{agentLoading ? "Checking registered models." : agentError ? agentError : "Registry status is live. Recommendations remain advisory and require connected Cloud telemetry."}</p>
                  </div>
                  <span className={`cloud-agent-state${!agentConnected && !agentLoading ? " unavailable" : ""}`}>{agentLoading ? "Connecting" : agentConnected ? "Connected" : "Unavailable"}</span>
                </article>

                <div className="cloud-ai-summary" aria-label="AI and Machine Learning readiness">
                  <article className="cloud-panel"><span>Agent API</span><strong>{agentLoading ? "Checking" : agentConnected ? "Connected" : "Unavailable"}</strong><small>Model registry connection</small></article>
                  <article className="cloud-panel"><span>Production models</span><strong>{agentLoading || agentError ? "—" : productionModels.length}</strong><small>Explicitly promoted versions</small></article>
                  <article className="cloud-panel"><span>Cloud telemetry</span><strong>Not connected</strong><small>Live workload metrics required</small></article>
                </div>

                <div className="cloud-ai-grid">
                  <article className="cloud-panel cloud-ai-models">
                    <header className="cloud-panel-heading">
                      <div><h2>Model registry</h2><p>Production and staging model versions from the Agent</p></div>
                      <span>{agentLoading || agentError ? "—" : `${agentModels.length} model${agentModels.length === 1 ? "" : "s"}`}</span>
                    </header>
                    {agentLoading ? (
                      <div className="cloud-ai-empty">Loading registered models…</div>
                    ) : agentError ? (
                      <div className="cloud-ai-empty">Model inventory is unavailable. Check the Agent API connection.</div>
                    ) : agentModels.length === 0 ? (
                      <div className="cloud-ai-empty">No models are registered yet. Register and promote a model through the Agent ModelOps API to make it available here.</div>
                    ) : (
                      <div className="cloud-ai-model-list">
                        {agentModels.map((model) => {
                          const versions = Object.values(model.versions ?? {});
                          return (
                            <article className="cloud-ai-model" key={model.name}>
                              <span className="cloud-ai-model-icon"><DashboardIcon name="ai" /></span>
                              <div className="cloud-ai-model-main">
                                <strong>{model.name}</strong>
                                <small>{model.metadata?.family ?? "Model family not specified"}</small>
                                <div className="cloud-ai-version-list">
                                  {versions.length ? versions.map((version) => (
                                    <span key={version.version}><code>{version.version}</code> {version.stage ?? "staging"}</span>
                                  )) : <span><code>{model.version ?? "—"}</code> {model.stage ?? "staging"}</span>}
                                </div>
                              </div>
                              <span className={`cloud-ai-model-stage${model.production_version ? " production" : ""}`}>
                                {model.production_version ? `Production · ${model.production_version}` : "No production version"}
                              </span>
                            </article>
                          );
                        })}
                      </div>
                    )}
                  </article>

                  <article className="cloud-panel cloud-ai-recommendations">
                    <header className="cloud-panel-heading"><div><h2>AI recommendations</h2><p>Model outputs · advisory only</p></div><span className="cloud-agent-state">Human review</span></header>
                    <div className="cloud-ai-telemetry">
                      <span><DashboardIcon name="observability" /></span>
                      <div><strong>Waiting for live Cloud telemetry</strong><p>The available database metrics endpoint currently returns placeholder values. Demo resources and placeholder data are not sent to models.</p></div>
                    </div>
                    <div className="cloud-ai-empty">
                      {agentError
                        ? "Connect the Agent API and live Cloud telemetry to see model recommendations."
                        : productionModels.length
                          ? `${productionModels.length} production model${productionModels.length === 1 ? " is" : "s are"} ready. Recommendations will appear when real workload metrics are available.`
                          : "Promote a production model and connect real workload metrics to enable recommendations."}
                    </div>
                  </article>
                </div>

                <article className="cloud-panel cloud-ai-capabilities">
                  <header className="cloud-panel-heading"><div><h2>Supported model families</h2><p>Implemented baseline capabilities · registry status shown per family</p></div></header>
                  <div className="cloud-ai-family-grid">
                    {agentModelFamilies.map((item) => {
                      const count = agentModels.filter((model) => model.metadata?.family === item.family).length;
                      return (
                        <article key={item.family}>
                          <span><DashboardIcon name={item.icon} /></span>
                          <div><strong>{item.name}</strong><p>{item.description}</p></div>
                          <small>{agentLoading || agentError ? "Status unavailable" : count ? `${count} registered` : "Not registered"}</small>
                        </article>
                      );
                    })}
                  </div>
                </article>

                <article className="cloud-panel cloud-ai-safety">
                  <DashboardIcon name="security" />
                  <div><strong>Recommendations never apply changes</strong><p>The Agent only provides advisory output. It does not provision resources, alter scaling, or modify Cloud configuration.</p></div>
                </article>
              </section>
            ) : selectedService === "Billing" ? (
              <article className="cloud-panel cloud-service-info">
                <span className="cloud-service-info-icon"><DashboardIcon name={selected.icon} /></span>
                <div>
                  <p className="cloud-eyebrow">BILLING WORKSPACE</p>
                  <h2>Billing data is not connected</h2>
                  <p>Usage and billing data will appear when the billing service is integrated.</p>
                  <button className="cloud-create-button" type="button" onClick={() => setSelectedService("Home")}>Return to overview <span>→</span></button>
                </div>
              </article>
            ) : (
              <section className="cloud-panel cloud-service-workspace">
                <header className="cloud-panel-heading">
                  <div><h2>{selectedService} resources</h2><p>{filteredResources.length} demo resource{filteredResources.length === 1 ? "" : "s"} · {region}</p></div>
                  {canCreate && <button type="button" onClick={() => openCreate(suggestedAction)}><DashboardIcon name="plus" /> Create {selected.kind.toLowerCase()}</button>}
                </header>
                {filteredResources.length > 0 ? (
                  <div className="cloud-table-scroll">
                    <table className="cloud-resource-table">
                      <thead><tr><th>Name</th><th>Type</th><th>Status</th><th>Region</th><th>Created</th></tr></thead>
                      <tbody>{filteredResources.map((resource) => (
                        <tr key={resource.id}><td><strong>{resource.name}</strong></td><td>{resource.kind}</td><td><span className="cloud-demo-status">{resource.status}</span></td><td>{resource.region}</td><td>{resource.created}</td></tr>
                      ))}</tbody>
                    </table>
                  </div>
                ) : (
                  <div className="cloud-empty-state">
                    <span><DashboardIcon name={selected.icon} /></span>
                    <h3>{query ? "No matching resources" : `No demo ${selectedService.toLowerCase()} yet`}</h3>
                    <p>{query ? "Try another search term or clear the search field." : "Create a sample resource here. It will be saved locally in this browser for your demo."}</p>
                    {canCreate && <button className="cloud-create-button" type="button" onClick={() => openCreate(suggestedAction)}><DashboardIcon name="plus" /> Create {selected.kind.toLowerCase()}</button>}
                  </div>
                )}
              </section>
            )}

            <footer className="cloud-page-footer">
              <span>ATONIXCORP CLOUD · INTERACTIVE DEMO</span>
              <span>Actions are stored locally; no live service provisioning.</span>
              <Link href="/contact">Contact the team</Link>
            </footer>
          </div>
        </div>
      </div>

      {createOpen && (
        <div className="cloud-modal-backdrop" role="presentation" onMouseDown={(event) => { if (event.target === event.currentTarget) setCreateOpen(false); }}>
          <section className="cloud-create-modal" role="dialog" aria-modal="true" aria-labelledby="cloud-create-title">
            <header>
              <div><p className="cloud-eyebrow">DEMO PROVISIONING</p><h2 id="cloud-create-title">Create a resource</h2></div>
              <button type="button" aria-label="Close dialog" onClick={() => setCreateOpen(false)}>×</button>
            </header>
            <p className="cloud-modal-notice">This creates a local demo entry only. It does not provision real infrastructure.</p>
            <form onSubmit={createResource}>
              <label>Resource type
                <select value={createType} onChange={(event) => setCreateType(event.target.value)}>
                  {resourceTypes.map((type) => <option key={type.label}>{type.label}</option>)}
                </select>
              </label>
              <label>Resource name
                <input autoFocus value={createName} onChange={(event) => setCreateName(event.target.value)} placeholder="e.g. research-instance-01" required maxLength={48} />
              </label>
              <label>Region
                <select value={region} onChange={(event) => setRegion(event.target.value)}>
                  {regions.map((item) => <option key={item}>{item}</option>)}
                </select>
              </label>
              <div className="cloud-modal-actions">
                <button className="cloud-modal-cancel" type="button" onClick={() => setCreateOpen(false)}>Cancel</button>
                <button className="cloud-create-button" type="submit"><DashboardIcon name="plus" /> Add demo resource</button>
              </div>
            </form>
          </section>
        </div>
      )}

      {toast && <div className="cloud-toast" role="status">{toast}</div>}
    </main>
  );
}

function ActivityTable({ activities }: { activities: CloudActivity[] }) {
  return (
    <div className="cloud-table-scroll">
      <table>
        <thead><tr><th>Time</th><th>Action</th><th>Resource</th><th>Status</th><th>User</th></tr></thead>
        <tbody>{activities.slice(0, 8).map((activity, index) => (
          <tr key={`${activity.time}-${activity.resource}-${index}`}>
            <td>{activity.time}</td><td>{activity.action}</td><td className="cloud-table-resource">{activity.resource}</td>
            <td><span className={`cloud-activity-state${activity.status === "Demo" ? " demo" : ""}`}><i />{activity.status}</span></td><td>{activity.user}</td>
          </tr>
        ))}</tbody>
      </table>
    </div>
  );
}

function RegionMap() {
  return (
    <>
      <div className="cloud-map" aria-label="Illustrative map showing example cloud regions">
        <svg viewBox="0 0 640 260" aria-hidden="true">
          <g className="cloud-map-dots">
            <path d="M53 63 70 48l24 5 8 13 19 3 8 14-13 9-4 16-17 8-2 19-19 2-6-19-19-10-8-18 12-12-8-15Zm44 82 20 4 8 14-8 18-15-5-12-17 7-14ZM274 60l16-8 18 3 12 12-8 11-17 2-3 14-12 4-9-9 5-14-12-4Zm15 45 19 1 12 18-6 25-12 17-10-13 1-19-8-13 4-16Zm49-67 32-8 34 8 15 13 36 3 14 16-11 15-31-3-15 10-27-6-21 8-14-14-19 1-5-20-18-8 10-15Zm31 96 16-6 12 11-7 15-16 4-5-24Zm124 1 22-8 20 9 5 18-16 16-20-3-16-14 5-18Z" />
          </g>
        </svg>
        <span className="cloud-map-pin pin-na"><i />North America<small>us-west</small></span>
        <span className="cloud-map-pin pin-eu"><i />Europe<small>eu-central</small></span>
        <span className="cloud-map-pin pin-ap"><i />Asia Pacific<small>ap-southeast</small></span>
        <span className="cloud-map-pin pin-sa"><i />South America<small>sa-east</small></span>
        <span className="cloud-map-pin pin-au"><i />Australia<small>ap-southeast</small></span>
      </div>
      <div className="cloud-map-legend"><span><i />Example region</span><span><i />Planned footprint</span></div>
    </>
  );
}
