export type CloudNavigationIcon = "overview" | "compute" | "containers" | "storage" | "networking" | "observability" | "security" | "databases" | "account";
type NavigationGroup = {
  label: string;
  section: "Workspace" | "Infrastructure" | "Operations" | "Organization";
  icon: CloudNavigationIcon;
  items: Array<{ label: string; href: string; unavailable?: boolean }>;
};

export type CloudService = {
  id: string;
  group: "Compute" | "Storage" | "Network" | "Orchestration";
  title: string;
  href: string;
  description: string;
  connected: string[];
  pending: string[];
  related?: { label: string; href: string }[];
};

// Implementation coverage, not runtime availability or permission grants.
export const cloudServices: CloudService[] = [
  { id: "instances", group: "Compute", title: "Instances", href: "/compute/instances",
    description: "Virtual machines with project-scoped provisioning and lifecycle management.",
    connected: ["Create and list instances", "Start, stop, reboot and delete", "Images, sizes, addresses and provisioning state"],
    pending: ["Rebuild and resize", "Console access", "Volume attachments, snapshots and backups", "Complete instance metrics and metadata editing"] },
  { id: "clusters", group: "Compute", title: "Clusters", href: "/cloud/clusters",
    description: "Cluster and HPC infrastructure for scheduled and accelerated workloads.",
    connected: [], pending: ["Cluster and node inventory", "Head/controller and compute nodes", "Schedulers, queues and jobs", "Accelerators, networking and storage", "Metrics and events"],
    related: [{ label: "Compute instances", href: "/compute/instances" }] },
  { id: "kubernetes", group: "Compute", title: "Kubernetes", href: "/cloud/kubernetes",
    description: "Manage supported container workloads while cluster-level management is being connected.",
    connected: ["Container deployment lifecycle in the workloads workspace", "Per-deployment instances, revisions, logs and metrics when reported"],
    pending: ["Cluster and node administration", "Namespace management", "General workloads, deployments and pods", "Services and ingress administration", "Storage, configuration and secrets", "Cluster-wide events and metrics"],
    related: [{ label: "Manage container workloads", href: "/containers" }, { label: "Deploy a container", href: "/containers/deploy" }] },
  { id: "serverless-inference", group: "Compute", title: "Serverless Inference", href: "/cloud/serverless-inference",
    description: "A dedicated workspace for future managed AI inference infrastructure.",
    connected: [], pending: ["Models and deployments", "Endpoints and inference jobs", "Workers and accelerators", "Usage, metrics and logs", "Autoscaling"],
    related: [{ label: "View agent registry on dashboard", href: "/dashboard" }] },
  { id: "block-storage", group: "Storage", title: "Block Storage", href: "/storage",
    description: "Persistent project volumes with quotas and asynchronous provisioning.",
    connected: ["Create, list and delete eligible volumes", "Capacity, type, zone, encryption and protection state", "Operation status"],
    pending: ["Attach, detach and resize", "Protection updates", "Volume snapshots and backups"] },
  { id: "object-storage", group: "Storage", title: "Object Storage", href: "/storage?tab=objects",
    description: "Store and retrieve objects through the authorized workspace object service.",
    connected: ["Object list, upload, download and delete", "Server-managed storage access"],
    pending: ["Bucket administration", "Access policies and credential management", "Lifecycle and versioning", "Storage usage and metrics"] },
  { id: "file-systems", group: "Storage", title: "File Systems", href: "/cloud/file-systems",
    description: "Shared file storage for application and infrastructure workloads.",
    connected: [], pending: ["File-system inventory and provisioning", "Capacity and protocols", "Mount targets and access policies", "Health, metrics and snapshots"] },
  { id: "storage-gateways", group: "Storage", title: "Storage Gateways", href: "/cloud/storage-gateways",
    description: "Connect supported storage resources through managed storage gateways.",
    connected: [], pending: ["Gateway inventory and lifecycle", "Endpoints and protocols", "Backing storage and access policies", "Connectivity health and metrics"] },
  { id: "databases", group: "Storage", title: "Databases", href: "/cloud/databases",
    description: "A dedicated workspace for database resources and provisioning operations.",
    connected: ["Database catalog, plans and instances", "Provisioning and operation state"],
    pending: ["Managed-engine readiness", "Complete credentials, backup/restore and networking management", "Database metrics and logs"] },
  { id: "backups", group: "Storage", title: "Backups", href: "/cloud/backups",
    description: "Centralized protection and recovery across supported resource types.",
    connected: [], pending: ["Typed backup inventory", "Policies and schedules", "Retention and encryption", "Restore and deletion", "Backup health and events"] },
  { id: "snapshots", group: "Storage", title: "Snapshots", href: "/cloud/snapshots",
    description: "A unified inventory for instance, volume and other supported snapshots.",
    connected: [], pending: ["Source resource type and identity", "Snapshot creation and state", "Capacity and creation time", "Restore and delete"],
    related: [{ label: "Manage volumes", href: "/storage" }, { label: "Manage instances", href: "/compute/instances" }] },
  { id: "vpc-networks", group: "Network", title: "VPC Networks", href: "/networking",
    description: "Project networks, subnets and address ranges.",
    connected: ["Network inventory", "Create network and subnet", "Provider-reported provisioning state"],
    pending: ["Network updates and deletion", "Durable operation reconciliation", "Connectivity telemetry"] },
  { id: "load-balancers", group: "Network", title: "Load Balancers", href: "/cloud/load-balancers",
    description: "Distribute traffic across healthy application backends.",
    connected: [], pending: ["Load balancer lifecycle", "Listeners, protocols and endpoints", "Pools and members", "Health checks and metrics"] },
  { id: "firewall-groups", group: "Network", title: "Firewall Groups", href: "/networking/firewalls",
    description: "Inspect authorized security groups and their provider-reported traffic rules.",
    connected: ["Read-only group and rule inventory"],
    pending: ["Production group creation", "Rule editing and deletion", "Resource assignments", "Group deletion"] },
  { id: "reserved-ips", group: "Network", title: "Reserved IPs", href: "/cloud/reserved-ips",
    description: "Centralized management for public, reserved and floating addresses.",
    connected: [], pending: ["Address allocation and inventory", "Instance associations", "Assign, unassign and release", "Region and quota reporting"],
    related: [{ label: "View instance addresses", href: "/compute/instances" }] },
  { id: "bgp", group: "Network", title: "BGP", href: "/cloud/bgp",
    description: "Manage supported routing peers and route announcements.",
    connected: [], pending: ["BGP sessions and peers", "ASN configuration", "Route announcements", "Session health and events"] },
  { id: "cdn", group: "Network", title: "CDN", href: "/cloud/cdn",
    description: "Content distribution and edge delivery for supported origins.",
    connected: [], pending: ["Distributions and origins", "Domains and TLS", "Cache policy and invalidation", "Traffic and delivery metrics"] },
  { id: "dns", group: "Network", title: "DNS", href: "/cloud/dns",
    description: "Manage DNS zones and records through AtonixCorp.",
    connected: [], pending: ["Zones and record sets", "Record validation and lifecycle", "Delegation and DNSSEC", "Propagation and audit events"] },
  { id: "direct-connect", group: "Network", title: "Direct Connect", href: "/cloud/direct-connect",
    description: "Private connectivity between customer networks and cloud infrastructure.",
    connected: [], pending: ["Connections and locations", "Circuits and virtual interfaces", "Routing and bandwidth", "Provisioning state and link health"] },
  { id: "container-registry", group: "Orchestration", title: "Container Registry", href: "/containers/registries",
    description: "Register image sources for project container deployments.",
    connected: ["Register and list image registries", "Server-held pull credentials"],
    pending: ["Hosted repositories, images and tags", "Registry permissions and usage", "Image lifecycle management"] },
  { id: "isos", group: "Orchestration", title: "ISOs", href: "/cloud/isos",
    description: "Centralized boot and installation media management.",
    connected: [], pending: ["Media upload and inventory", "Integrity validation", "Boot attachment and detachment", "Media deletion"],
    related: [{ label: "Browse compute images", href: "/compute/images" }] },
  { id: "ssh-keys", group: "Orchestration", title: "SSH Keys", href: "/cloud/ssh-keys",
    description: "Manage public SSH keys without exposing private-key material.",
    connected: [], pending: ["Public-key import and validation", "Name, fingerprint and creation time", "Associated resources", "Key removal"],
    related: [{ label: "Deploy with an existing key name", href: "/compute/deploy" }] },
  { id: "startup-scripts", group: "Orchestration", title: "Startup Scripts", href: "/cloud/startup-scripts",
    description: "Reusable initialization configuration for instance deployment.",
    connected: [], pending: ["Script inventory and validation", "Create, update and delete", "Versions and deployment associations", "Execution events"] },
];

function serviceItems(group: CloudService["group"]) {
  return cloudServices.filter((service) => service.group === group).map((service) => ({
    label: service.title, href: service.href, unavailable: !service.connected.length,
  }));
}

export const cloudNavigation: NavigationGroup[] = [
  { label: "Overview", section: "Workspace", icon: "overview", items: [
    { label: "Dashboard", href: "/dashboard" }, { label: "Service directory", href: "/cloud/services" },
  ] },
  { label: "Compute", section: "Infrastructure", icon: "compute", items: [
    ...serviceItems("Compute"),
    { label: "Container workloads", href: "/containers" },
    { label: "Images", href: "/compute/images" },
    { label: "Instance sizes", href: "/compute/sizes" },
  ] },
  { label: "Storage", section: "Infrastructure", icon: "storage", items: serviceItems("Storage") },
  { label: "Network", section: "Infrastructure", icon: "networking", items: serviceItems("Network") },
  { label: "Orchestration", section: "Infrastructure", icon: "containers", items: serviceItems("Orchestration") },
  { label: "Observability", section: "Operations", icon: "observability", items: [
    { label: "Infrastructure health", href: "/observability" },
    { label: "Metrics", href: "/observability?section=Metrics" },
    { label: "Events", href: "/observability?section=Events" },
  ] },
  { label: "Security", section: "Operations", icon: "security", items: [{ label: "Policies & audit", href: "/security" }] },
  { label: "Account", section: "Organization", icon: "account", items: [{ label: "Usage & billing", href: "/billing" }] },
];

export function isCloudNavigationItemActive(href: string, pathname: string, query: URLSearchParams): boolean {
  const [path, search] = href.split("?");
  const matchesPath = pathname === path ||
    (path === "/compute/instances" && (pathname === "/compute" || pathname === "/compute/deploy" || pathname.startsWith("/compute/instances/"))) ||
    (path === "/containers" && pathname.startsWith("/containers/") && !pathname.startsWith("/containers/registries")) ||
    (path === "/cloud/databases" && pathname === "/databases");
  if (!matchesPath) return false;
  if (search) return [...new URLSearchParams(search)].every(([key, value]) => query.get(key) === value);
  return !(pathname === "/storage" && query.get("tab") === "objects") &&
    !(pathname === "/observability" && ["Metrics", "Events"].includes(query.get("section") ?? ""));
}
