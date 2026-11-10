export type Product = {
  slug: string;
  name: string;
  category: string;
  summary: string;
  headline: string;
  description: string;
  icon: "compute" | "storage" | "networking" | "containers" | "databases" | "observability";
  workspace: string;
  features: { title: string; description: string }[];
  considerations: string[];
};

export const products: Product[] = [
  { slug: "compute", name: "Cloud Compute", category: "COMPUTE", icon: "compute", workspace: "/compute/instances",
    summary: "Virtual machines for the applications you are building.",
    headline: "Give your applications a place to run.",
    description: "Choose an image, instance size and network. Manage your virtual machines from a project-scoped workspace, with their lifecycle and addresses in view.",
    features: [
      { title: "Start with the right foundation", description: "Browse the images and instance sizes available to your project before you deploy." },
      { title: "Manage the full working day", description: "Start, stop, reboot and delete instances through the AtonixCorp console." },
      { title: "Keep workloads organized", description: "Separate resources by workspace and project, and inspect their current provisioning state." },
    ],
    considerations: ["Available sizes and regions depend on your configured infrastructure.", "Rebuild, resize and browser-console access are not yet connected.", "Pricing and free-capacity estimates are not currently published."] },
  { slug: "storage", name: "Cloud Storage", category: "DATA", icon: "storage", workspace: "/storage",
    summary: "Persistent volumes and an authorized object workspace.",
    headline: "A home for the data behind your work.",
    description: "Provision persistent block volumes and manage your objects without switching between provider dashboards. Keep capacity, quotas and operation state visible.",
    features: [
      { title: "Persistent block volumes", description: "Create volumes using the storage types available to your project and track asynchronous provisioning." },
      { title: "Your object workspace", description: "Upload, list, download and delete authorized objects within your account and workspace." },
      { title: "Capacity in context", description: "Inspect provisioned capacity, storage quotas, placement and encryption information reported by the service." },
    ],
    considerations: ["Block and object storage are independently configured services.", "Volume attachments, snapshots and shared file systems are not yet connected.", "Bucket administration, versioning and lifecycle rules are not available in the console."] },
  { slug: "networking", name: "Cloud Networking", category: "CONNECTIVITY", icon: "networking", workspace: "/networking",
    summary: "Project networks, subnets and firewall visibility.",
    headline: "Connect your workloads with intention.",
    description: "Create project networks and subnets, plan address ranges, and inspect the firewall rules reported by your infrastructure.",
    features: [
      { title: "Networks for each project", description: "Keep network resources within the same ownership boundary as your applications." },
      { title: "Address space you can inspect", description: "Review network CIDRs and subnet provisioning state from a dedicated networking workspace." },
      { title: "See the rules", description: "Browse authorized firewall groups and their ingress and egress rules in a read-only inventory." },
    ],
    considerations: ["Network provisioning must be enabled for your infrastructure.", "Firewall rule editing and network update/delete workflows are not connected.", "Load balancers, managed DNS and private circuits are not currently available."] },
  { slug: "containers", name: "Containers", category: "APPLICATIONS", icon: "containers", workspace: "/containers",
    summary: "Image-based workloads with deployment-level control.",
    headline: "From container image to working application.",
    description: "Register your image sources, configure a deployment and manage its lifecycle. Use deployment details to investigate revisions, replicas, logs and reported metrics.",
    features: [
      { title: "Bring your image", description: "Connect an image registry and keep pull credentials on the server rather than in browser configuration." },
      { title: "Define how your app runs", description: "Configure replicas, resource requests, ports and probes through the supported deployment workflow." },
      { title: "Operate with context", description: "Inspect revisions and runtime information, then scale, restart or roll back supported deployments." },
    ],
    considerations: ["A connected container runtime and authorized project are required.", "Runtime telemetry is only shown when it is reported.", "General Kubernetes cluster and namespace administration is not yet available."] },
  { slug: "databases", name: "Databases", category: "DATA", icon: "databases", workspace: "/cloud/databases",
    summary: "A dedicated workspace for database infrastructure.",
    headline: "Put database infrastructure in its own workspace.",
    description: "Explore configured database engines and plans, create database resources, and track their provisioning operations alongside the rest of your infrastructure.",
    features: [
      { title: "Explore the catalog", description: "Inspect engine, version and plan options made available by your configured database service." },
      { title: "Follow each operation", description: "Keep resource provisioning and operation status visible rather than treating a submitted request as a completed deployment." },
      { title: "Keep ownership explicit", description: "Manage database infrastructure within the platform's workspace and project boundaries." },
    ],
    considerations: ["This is an evolving database infrastructure integration, not a fully managed database SLA.", "Infrastructure readiness does not establish database-engine readiness.", "Complete backup/restore, engine monitoring and credential-management workflows remain under development."] },
  { slug: "observability", name: "Observability", category: "OPERATIONS", icon: "observability", workspace: "/observability",
    summary: "Resource health, reported metrics and platform events.",
    headline: "Understand what your infrastructure is telling you.",
    description: "Bring reported health, resource inventory, metrics and events into one operational workspace. Know the difference between a healthy resource and a missing signal.",
    features: [
      { title: "A resource-first view", description: "Explore authorized resources and their reported health in the context of your project." },
      { title: "Signals with provenance", description: "Read observed metrics and telemetry freshness instead of relying on estimated utilization." },
      { title: "Activity in context", description: "Review platform events and integration availability while investigating changes." },
    ],
    considerations: ["Metrics depend on connected collectors and authorized sources.", "Unavailable telemetry is displayed explicitly, not treated as healthy.", "Complete log aggregation and alert evaluation are not yet connected."] },
];

export const solutions = [
  { slug: "application-teams", name: "Application teams", headline: "Build the application. Keep the infrastructure understandable.",
    summary: "A clear path from a workload to an operational workspace.",
    description: "For teams moving web applications, APIs and background services into a shared cloud environment.",
    priorities: [
      { title: "Choose your runtime", description: "Use a virtual machine when you need operating-system control, or a container deployment when your application is packaged as an image." },
      { title: "Separate environments", description: "Plan project boundaries for development, staging and production before deploying resources." },
      { title: "Plan for operations", description: "Identify the metrics, data-protection requirements and access policies your service needs before launch." },
    ], productSlugs: ["compute", "containers", "observability"], note: "Confirm your runtime, recovery and availability requirements with the team. No uptime or managed-recovery guarantee is implied." },
  { slug: "data-workloads", name: "Data-intensive workloads", headline: "Design the data path before you scale the workload.",
    summary: "Think through storage, processing and data-service boundaries.",
    description: "For teams working with datasets, application persistence and processing jobs that need a deliberate infrastructure plan.",
    priorities: [
      { title: "Map the data lifecycle", description: "Separate uploaded objects, persistent application data and temporary processing outputs." },
      { title: "Match storage to access", description: "Evaluate supported block and object storage workflows against your application's durability and access needs." },
      { title: "Validate the entire system", description: "Test database-engine readiness, network access and recovery outside provisioning status alone." },
    ], productSlugs: ["storage", "databases", "compute"], note: "Managed backups, general file systems and a complete managed-database service are not currently available." },
  { slug: "research", name: "Research & advanced computing", headline: "Start with the research question. Plan the infrastructure around it.",
    summary: "Scope compute-intensive research and future accelerated workflows.",
    description: "For research and engineering teams exploring simulation, model development and demanding computational workloads.",
    priorities: [
      { title: "Define the workload", description: "Share memory, processing, storage, accelerator and scheduling requirements with the infrastructure team." },
      { title: "Build a practical baseline", description: "Evaluate configured virtual machines and object workflows for development and smaller experiments." },
      { title: "Distinguish current from planned", description: "Treat integrated HPC, cluster management and serverless inference as development areas, not available managed services." },
    ], productSlugs: ["compute", "storage", "observability"], note: "Managed HPC clusters, accelerator availability and serverless inference require separate integration and confirmation." },
];

export const publicNavigation = [
  { id: "products", label: "Products", intro: "Infrastructure, without the detours.", description: "Explore the building blocks of your cloud workspace.", href: "/products", cta: "Explore all products",
    groups: [
      { title: "Build & run", links: products.filter((p) => ["compute", "containers", "networking"].includes(p.slug)).map((p) => ({ label: p.name, description: p.summary, href: `/products/${p.slug}`, icon: p.icon })) },
      { title: "Store & operate", links: products.filter((p) => ["storage", "databases", "observability"].includes(p.slug)).map((p) => ({ label: p.name, description: p.summary, href: `/products/${p.slug}`, icon: p.icon })) },
    ] },
  { id: "solutions", label: "Solutions", intro: "Start with what you want to build.", description: "Different workloads. Deliberate infrastructure choices.", href: "/solutions", cta: "Explore solutions",
    groups: [{ title: "By workload", links: solutions.map((s) => ({ label: s.name, description: s.summary, href: `/solutions/${s.slug}`, icon: "compute" as const })) },
      { title: "Plan your platform", links: [
        { label: "Platform architecture", description: "Understand the layers and ownership boundaries.", href: "/platform", icon: "networking" as const },
        { label: "Discuss your requirements", description: "Work through a deployment with our team.", href: "/contact", icon: "account" as const },
      ] }] },
  { id: "developers", label: "Developers", intro: "Understand it. Build on it.", description: "Find the source, service documentation and a path into the platform.", href: "/developers", cta: "Open developer hub",
    groups: [
      { title: "Get started", links: [
        { label: "Developer hub", description: "Your starting point for the platform.", href: "/developers", icon: "containers" as const },
        { label: "Platform documentation", description: "Architecture and service integration guides.", href: "/developers/documentation", icon: "storage" as const },
        { label: "API & CLI resources", description: "Explore interfaces and command-line tooling.", href: "/developers/api-cli", icon: "compute" as const },
      ] },
      { title: "Work together", links: [
        { label: "Open source", description: "Explore the code and contribution guidance.", href: "/developers/open-source", icon: "containers" as const },
        { label: "Community", description: "Connect with the AtonixCorp community.", href: "/developers/community", icon: "account" as const },
        { label: "Support center", description: "Sign in to submit and track a ticket.", href: "/support", icon: "observability" as const },
      ] },
    ] },
  { id: "company", label: "Company", intro: "A platform with a point of view.", description: "Learn what we are building, and how to be part of it.", href: "/about", cta: "About AtonixCorp",
    groups: [
      { title: "AtonixCorp", links: [
        { label: "About us", description: "Our direction and engineering principles.", href: "/about", icon: "account" as const },
        { label: "Platform roadmap", description: "What is connected and what comes next.", href: "/about/roadmap", icon: "networking" as const },
        { label: "Sponsor the work", description: "Support research, technology and community.", href: "/sponsor", icon: "storage" as const },
      ] },
      { title: "Let's connect", links: [
        { label: "Contact our team", description: "Platform questions and partnerships.", href: "/contact", icon: "account" as const },
        { label: "Account portal", description: "Find your next step into AtonixCorp.", href: "/portal", icon: "compute" as const },
      ] },
    ] },
];
