# Frontend audit baseline

## Dedicated public destinations (2026-11-10)

Public website scope only: developer documentation, API/CLI resources, open
source, community and the platform roadmap now have dedicated routes:
`/developers/documentation`, `/developers/api-cli`, `/developers/open-source`,
`/developers/community` and `/about/roadmap`. Each has its own page content,
primary heading, title and description under the existing public shell.

The mega menus, developer hub and footer now link to these pages rather than
hub/about fragments or sending menu selections directly to GitHub. External
technical references and community channels remain clearly labeled links within
the relevant pages. The hub and about page retain brief discovery links instead
of embedding the full community and roadmap content. Existing section bookmarks
still reach those discovery links. Console routing and authentication are unchanged.

The public integration suite includes the five new routes and a regression
check requiring every mega-menu item to resolve to a distinct same-site page
within its menu, without a homepage or fragment destination.

Validation: affected-file ESLint, TypeScript/production builds and all five
public integration tests passed on the preview and deployed web container,
covering 25 public pages. Browser checks exercised six menu destinations at
390px and 1440px, plus keyboard navigation, direct reload and browser Back.
Each destination retained its own URL/content and closed the menu on navigation.
Only the web container was replaced; all seven root services remained healthy.

## Console logout (2026-11-10)

The shared console header provides a keyboard-accessible **Log out** button on
desktop and mobile. It uses the existing same-origin POST `/api/auth/logout`
endpoint to expire the HTTP-only session cookie, then performs a full-document
replacement to `/login` so authenticated client router state is discarded.
While pending, the button is disabled. Failed or timed-out requests leave the
page in place, show an accessible error and allow retry. The header wraps on
narrow screens rather than hiding account controls.

Validation: ESLint, TypeScript, the production build and thirteen existing tests
passed. Browser checks on an isolated localhost hostname verified keyboard
activation, pending state, HTTP/network failure and retry, cookie removal,
redirect to sign-in and the protected dashboard redirect after logout. The
control remained visible without page overflow at 320px, 390px and 1440px.
The web container was rebuilt and deployed; all seven services are healthy.

## Console visual alignment (2026-11-10)

The internal console now uses the public website's blue/navy visual language
without changing resource API contracts, authorization or provisioning behavior.

- `CloudShell` provides a theme-aware sidebar, consistent brand treatment,
  console utility header, developer/public-site links and an always-accessible
  theme control. Tablet navigation collapses at 900px instead of squeezing the
  service content beside a full sidebar.
- Dashboard summary cards have service icons, larger values, clearer explanatory
  text and responsive columns. Context selectors, operational panels, status
  badges, activity and service-directory cards share the same spacing and surfaces.
- `shared/ui/console-design.css` is deliberately scoped to `[data-cloud-console]`.
  It aligns existing compute, containers, storage, databases, networking,
  observability, billing and support layouts without replacing their handlers.
  Module-based security, firewall and capability workspaces use the same console
  tokens. Public pages do not receive these console overrides.
- Headers, cards, tables, empty states, inputs and buttons use consistent
  typography, blue accents, borders and focus treatment in light and dark mode.
  Unknown metrics and unconnected services remain explicitly identified.
- Mobile compute filters now wrap. The table scroll container also establishes
  the containing block for its screen-reader-only action label, preventing that
  absolutely positioned label from creating page-wide overflow.

Verification: affected-file ESLint, TypeScript, optimized production builds and
all thirteen existing cloud/public integration tests passed. Seventeen internal
routes were checked at 390px, 768px and 1440px, including desktop dark mode:
one main landmark/primary heading per route, and no page-level horizontal
overflow after correcting the compute toolbar/table. Resource tables retain
their own horizontal scrolling.

Native browser checks exercised service search, scope-preserving navigation,
two-stage Escape and focus return on mobile, skip navigation, theme persistence,
instance filtering, pagination's disabled boundary and switching to an empty
project. Visual inventory checks used an isolated read-only fixture API and a
dedicated preview hostname, not production credentials or resources. Error and
unavailable states were included; this is not a live provisioning/mutation or
full assistive-technology certification.

Deployment: rebuilt and replaced only the root Compose `web` service. All seven
root services remained healthy. The deployed homepage returns HTTP 200,
unauthenticated dashboard access redirects to login, and all four public
integration tests pass against port 3000. Existing dependency advisories noted
below remain unresolved; this visual update does not establish security readiness.

## Public website redesign (2026-11-10)

The public website now has a separate, original marketing design system and
shared `PublicShell`. The homepage introduces the platform instead of embedding
the company, roadmap, contact and portal pages. Vultr was a reference for the
requested navigation organization, not a source of copied branding or assets;
its live site could not be inspected beyond access verification.

### Delivered pages and navigation

- Twenty public destinations: homepage; product catalog and six product pages;
  solution directory and three workload guides; architecture; developer hub;
  company/direction; account portal; contact; sponsorship; sign-in; onboarding.
- Product and solution details have distinct content, metadata, related services
  and explicit integration limitations. Unknown slugs return 404.
- Four responsive mega-navigation groups replace the crowded menu of repeated
  homepage anchors. Desktop supports hover/click, ArrowDown, Escape/focus return
  and dismissal outside the header. Mobile provides an expandable navigation
  and accordion groups without duplicating its links in a second DOM tree.
- Public pages share a skip link, one main landmark, visible focus treatments,
  a consistent footer and real destinations. Privacy/accessibility links are
  accurately labeled contact routes, not invented legal policies.
- Contact, sponsorship, checkout configuration and authentication handlers are
  preserved. Sign-in and onboarding now have individual document metadata.
  The existing support workspace remains separate from the marketing shell.
- No invented customer logos, published prices, SLA, regional capacity or live
  health metrics were added. The architecture illustration is labeled conceptual.

### Verification

- Affected-file ESLint, TypeScript and the optimized Next.js build passed.
- Thirteen Node tests passed: nine existing cloud regressions and four public
  integration tests. Public checks cover all twenty routes, unique headings and
  titles, landmarks, internal links/anchors, unknown-slug 404s, capability
  limitations and existing sign-in redirects. The support shell remains public
  with authentication enforced by its existing APIs.
- Browser checks at 390, 768 and 1440 pixels covered all twenty public pages
  (sixty page/viewport combinations):
  no horizontal page overflow and one primary heading/main per page.
- Native keyboard checks verified all four desktop groups, ArrowDown into a
  visible link, Escape and focus return. Mouse checks verified hover, first-click
  retention, second-click closure and outside dismissal. Mobile checks verified
  opening, ArrowDown and two-stage Escape with focus restored to the menu button.
- Existing required-field validation, signup password mismatch, sponsorship
  package/amount synchronization and explicit contact failure feedback were
  checked. The isolated preview deliberately used an unavailable upstream API;
  no real accounts, contact messages or sponsorship payments were created.
- The web image was rebuilt and the web container alone was recreated. All seven
  root Compose services remained healthy; no backend service or volume was
  restarted or removed. The four public integration tests also passed against
  the deployed site on port 3000. Deployed-browser checks confirmed mobile
  catalog links, menu dismissal after navigation and skip-link focus.
- The container dependency installation reported four runtime dependency
  advisories (one moderate, two high, one critical), and thirteen including
  development dependencies. Dependency upgrades were not part of this visual
  redesign; these advisories need separate investigation before production
  certification.

Run the public integration tests against a local running production preview:

```sh
cd web
npm run build
API_URL=http://127.0.0.1:9 npm start -- --hostname 127.0.0.1 --port 4317
# In another terminal, from web/:
PUBLIC_TEST_BASE_URL=http://127.0.0.1:4317 npm run test:public
npm run test:cloud
```

These are focused frontend checks, not a full assistive-technology audit,
performance certification or validation of live cloud-provider operations.
The backend capability limits below still apply. The remaining sections record
the earlier cloud increment and original baseline, not unresolved claims about
the redesigned public navigation.

## Cloud control-plane specification audit (2026-11-10)

The supplied cloud-console specification is the migration contract. Existing
uncommitted frontend work is being extended, not replaced. This source audit
does **not** certify production readiness or live provider operations.

### Capability inventory

| Product | Existing authenticated implementation | Missing / not to be presented as working |
| --- | --- | --- |
| Dashboard | Workspace/project inventory, compute allocation, block quota, observed utilization, resource health, events | Physical CPU/RAM capacity, complete alert rules, cluster inventory and universal live health |
| Instances | Project catalog, create/list/detail, start/stop/reboot/delete, asynchronous status refresh | Rebuild, resize, console, attachments, snapshots and complete per-instance telemetry |
| Clusters / HPC | Separate `cloud/hpc` service and scheduler adapters | Integrated console gateway, inventory and scheduler authorization; see provider-boundary audit |
| Kubernetes | Container deployment/revision/lifecycle APIs, per-deployment pods/logs/metrics | Cluster CRUD and general namespace/workload/service/ingress/configuration management |
| Serverless Inference | Separate agent registry is **not** a serverless inference service | Deployment, endpoints, jobs, scaling, workers and usage API |
| Block Storage | Volume types, quotas, list/create/delete, operations and Cinder/local drivers | Attach/detach/resize, protection updates, backups and snapshots |
| Object Storage | Tenant-scoped object upload/list/download/delete; optional MinIO adapter | Customer bucket administration, lifecycle/versioning/policy and credential management |
| File Systems / Storage Gateways | No authenticated console management contract located | Inventory and full lifecycle API |
| Databases | Dedicated catalog/list/create/detail/operations UI and database service | Managed-engine readiness, complete backup/restore/logging workflows |
| Backups / Snapshots | No centralized lifecycle contract | Typed source resources, policies, schedules, restore, retention and deletion |
| VPC Networks | Project networks/subnets, create and provider confirmation | Update/delete and durable operations in the console |
| Firewall Groups | Project-authorized security-group/rule list API | Rule editing, assignments, delete; current backend create is a fixed **test-only** rule, not production firewall creation |
| Reserved IPs / Load Balancers / BGP / CDN / DNS / Direct Connect | No complete authenticated console management contract | Resource inventory and provider-backed lifecycle APIs; provider catalog presence alone is insufficient |
| Container Registry | Project registry registration/list with server-held credentials | Hosted repository/image/tag administration and usage |
| ISOs / SSH Keys / Startup Scripts | Image catalog and optional existing key name on VM creation | Media/key/script CRUD; current SSH-key route is an unconditional 401 stub, not a usable management API |

### Existing architecture and preservation requirements

- Routes: existing route families above remain supported. Dedicated capability
  workspaces use `/cloud/[service]`; `/cloud/databases` retains its existing page.
  Unsupported workspaces are explicitly marked **Not connected**, never counted
  as completed services. `/cloud/services` is an implementation-coverage directory,
  not a runtime health or provider-capability registry.
- Shared UI: `CloudShell`, `Infrastructure`, `CloudNavIcon`, theme tokens and
  feature-specific forms already exist. Reuse these rather than designing every
  page independently.
- API clients: `compute/lib/api.ts`, `storage/lib/api.ts`,
  `containers/lib/api.ts`, `observability/lib/api.ts`, `networking/api.ts`,
  database and object-storage clients call same-origin Next.js gateways.
  Gateways forward the HttpOnly session to AtonixCorp services, not providers.
- Authentication: `web/proxy.ts` verifies protected page sessions through
  `/api/v1/auth/me`. Gateways check cookies; write gateways check request origin.
  Backend project membership and write-role checks remain authoritative.
  A visible action is not a permission grant; forbidden and unavailable responses
  must remain explicit and must not become empty inventories.
- Providers: compute uses the registered OpenStack adapter; Cinder and local
  block drivers are separate from the optional MinIO object adapter. Ceph is
  an infrastructure dependency, not an independently proven console integration.
  See [`provider-boundary.md`](../api/docs/provider-boundary.md).
- State: local React state/effects and `useCloudResource` scope requests by
  tenant/project/revision and discard obsolete completions. No shared global
  infrastructure store or browser-to-provider connection is needed.
- Realtime: the console uses polling (VM transitions, volume operations, agent
  registry); no unified authenticated WebSocket inventory stream was established.
- Existing tests: `test:cloud` covers summary calculations, deep-link scope and
  navigation. Go suites cover provider boundaries, authorization and service
  operations. Compose smoke scripts cover broader deployments. These are not
  substitutes for live create/update/delete/provider reconciliation tests.
- Source-visible gaps: dead instance-detail tabs, missing service navigation,
  allocation versus physical-capacity ambiguity, and unavailable services
  represented by generic empty pages. Preserve working actions while addressing
  these gaps; do not infer missing data as zero.

### Delivery and acceptance gates

1. Cloud shell/navigation/dashboard: grouped complete hierarchy, meaningful
   service-availability states, project context, observed metrics, health,
   recent resources/activity and scoped quick actions.
2. Compute: preserve VM creation/lifecycle; expose exact unsupported operations.
   HPC and general Kubernetes remain gated on tenant-authorized contracts.
3. Storage: preserve block/object/database flows; separate missing shared-file,
   gateway, backup and snapshot capabilities.
4. Network: retain VPC flows and expose real firewall inventory; do not turn
   catalog discovery into fictional load balancers, DNS or other services.
5. Orchestration: retain registry registration; show explicit media/key/script
   dependencies without fake forms or private-key material.
6. Backend/provider wiring: requires implementations and configured infrastructure
   for each missing contract above, not additional environment placeholders.
7. RBAC/validation: retain server authorization, test failure paths, never use
   frontend filtering as access control.
8. Responsive/accessibility: shared responsive panels, table overflow, labels,
   keyboard navigation, focus visibility and explicit unavailable states.
9. Testing/performance: run focused tests, TypeScript, lint/build and browser
   checks; record actual results below.
10. Production readiness: remains blocked until each service passes authenticated
    route/render/read/create/update/delete/error/refresh tests against a configured
    provider, with persisted operation reconciliation, telemetry, recovery,
    quotas and permission tests. No production-readiness claim follows from
    adding navigation or passing a frontend build.

### Implemented and verified in this increment

- Added all 23 requested product destinations under Compute, Storage, Network
  and Orchestration, retaining the existing operational pages. A searchable,
  collapsible sidebar and service directory distinguish partial support from
  services without connected management APIs.
- Extended the operational dashboard with storage-zone context, remaining
  block quota, explicit CPU/RAM capacity limits, cluster/Kubernetes integration
  status, alert coverage and merged recent-resource inventory.
- Preserved workspace/project scope in dashboard, sidebar, service-directory
  and related-workflow links. Scope changes discard previous project data;
  invalid service-page scope is rejected rather than silently defaulted.
- Added real read-only firewall group/rule inventory using the existing
  authenticated API, with search, direction filtering, sorting, pagination and
  explicit permission/dependency states. Fixed-rule test creation is not exposed.
- Added VM status filtering, sorting, pagination and boot-disk specifications;
  replaced inert detail tabs with working workspace links, a refresh action,
  API-reported references and an explicit unsupported-operations list.
- Added volume search/status/sort/pagination, zone/encryption/attachment details,
  and network sorting/pagination. Existing lifecycle actions remain intact.

Validation:

| Check | Result |
| --- | --- |
| `node --experimental-strip-types --test app/dashboard/infrastructure-summary.test.mjs` (from `web`) | 9 tests passed |
| TypeScript `--noEmit --incremental false` | Passed |
| ESLint on all affected TypeScript/TSX and the cloud test file | Passed |
| Next.js production build, including generated route types | Passed |
| `git diff --check` | Passed |
| All 23 catalog destinations using the production build | HTTP 200 with isolated test authentication |
| Unknown service / malformed service scope | HTTP 404 |
| Unauthenticated firewall page / API | Login redirect / HTTP 401 |
| Browser dashboard with explicit API fixtures | Allocation, remaining quota, zones, empty-project switching, refresh and scoped links verified |
| Browser firewall inventory | 20/3-row pagination, search, HTTP 403 feedback and removal of previous-project rows verified |
| Browser VM and volume inventory | Boot-disk display, status filtering and no-matching-volume state verified |
| Responsive checks | Dashboard had no page-level horizontal overflow at 390, 768 and 1440 CSS pixels; service directory at 390; mobile navigation/search exercised |
| Service-directory and related-workflow links | Selected workspace/project retained |

Browser tests used a separate loopback server and explicit fixtures outside the
repository, not fabricated production data. The test services were stopped and
the fixture session removed. These checks do **not** establish live provider
create/update/delete success, full accessibility conformance, load performance,
or production readiness. No running shared deployment was rebuilt or restarted.
The backend gaps in the capability inventory remain open requirements.

## Scope and status

This is the read-only Phase 1 audit baseline for the platform-wide redesign
brief. It inventories the current frontend and records source-visible risks
before shared visual changes begin. It is not a visual redesign, accessibility
certification, or approval to remove legacy files. Browser checks across every
route and viewport remain outstanding.

## Route inventory

| Route family | Routes |
| --- | --- |
| Public and corporate | `/`, `/platform`, `/contact`, `/sponsor`, `/portal`, `/support` |
| Authentication | `/login`, `/onboard` |
| Cloud overview | `/dashboard` |
| Compute | `/compute`, `/compute/deploy`, `/compute/instances`, `/compute/instances/[instanceId]`, `/compute/[section]` (including dynamic sections such as `/compute/ssh-keys`) |
| Containers | `/containers`, `/containers/deploy`, `/containers/[id]`, `/containers/registries` |
| Storage and data | `/storage`, `/databases`, `/cloud/databases` |
| Networking | `/networking` |
| Next.js API routes found in `app/api` | `/api/contact`, `/api/sponsor`, `/api/auth/*`, `/api/compute/*`, `/api/containers/*`, `/api/databases/*`, `/api/networking/*`, `/api/storage/*`, `/api/sponsor` |

The route list is from the current `web/app` tree. It should be confirmed
against deployed routes before implementation begins.

## Existing UI foundation

- Root styles are imported from [`app/layout.tsx`](./app/layout.tsx): `base.css`,
  `theme.css`, and `layout.css`.
- Public-site shell components include `TopHeader`, `GlobalHeader`, and
  `Footer`, under `app/sections/header` and `app/sections/footer`.
- Existing primitives include `Button`, `Card`, `Grid`, and `Section` under
  `app/shared/ui`. Shared hooks and navigation helpers are under
  `app/shared/hooks` and `app/shared/lib`.
- The compute area has `app/compute/components/ComputeShell.tsx`; containers
  have a separate `app/containers/layout.tsx`. Dashboard, storage, databases,
  and networking own additional page-level shell treatments.
- Assets include `public/atonixcorp-icon.png`, `public/dasboard.png`,
  `public/Devop-dasboard-ui.jpeg`, `public/heroimage/*`, and
  `public/icons/{github,gmail,google,linkedin}.svg`. The apparent typo in
  `dasboard.png` is not grounds for renaming or deletion without checking
  references.
- Root text styling uses `Inter, system-ui, sans-serif`; the footer also
  references `var(--font-geist-sans)`. The root layout does not use `next/font`.
- Icons are mixed: local dashboard SVG rendering, inline SVG, social SVG assets,
  and text glyphs. No platform-wide icon API was found.

## Styling and token baseline

The UI combines root global CSS, component/page imports, and feature-specific
stylesheets. Large feature styles include dashboard, compute, containers,
storage, and databases CSS. Networking currently embeds styling in its page.
Both `postcss.config.js` and `postcss.config.mjs` exist; which one is active must
be verified before changing build configuration.

[`app/styles/theme.css`](./app/styles/theme.css) has a small set of generic
variables but does not define the complete semantic system requested in the
brief. Dashboard and compute define separate local palettes, while marketing
styles use many literal color, spacing, radius, and shadow values. Breakpoints
also vary by feature. There is no single established token map for focus,
semantic status colors, typography, spacing, elevation, or responsive widths.

## Duplication map

| Area | Evidence and consequence |
| --- | --- |
| Public shell | Public pages assemble headers and footers themselves; for example `app/home/HomePage.tsx` and `app/portal/page.tsx`. A shell change needs public-route regression checks. |
| Cloud shell | Compute, containers, dashboard, storage, databases, and networking each handle shell/navigation differently. Shared-shell consolidation is high-risk and must preserve authentication, project selection, and product-specific actions. |
| Forms and controls | Login, onboarding, sponsor, containers, storage, and databases have distinct control classes and state styling. Extract only after capturing existing validation and submission behavior. |
| Navigation | Service links are repeated across dashboard and product pages rather than driven by one authoritative cloud navigation model. |
| Icons and brand treatments | The dashboard has a local icon renderer, while other areas use SVGs and glyphs; service subtitles and wordmarks vary. |
| CSS and sections | Repeated palette/breakpoint values exist across feature stylesheets. Similar `DivisionsSection.tsx` files exist under both `sections/division` and `sections/divisions`; verify references before consolidation. |

## Issue register

Findings below are source observations, not a completed manual or automated
accessibility audit.

| Priority | Finding | Evidence / risk |
| --- | --- | --- |
| High | No global skip link is apparent; several route shells include their own `<main>`, creating a risk of duplicate main landmarks. | `app/layout.tsx`; `app/containers/layout.tsx`; validate every route's rendered landmark tree. |
| High | Public navigation dropdowns appear to lack menu semantics, explicit trigger relationships, and managed Escape/focus behavior. | `app/sections/header/GlobalHeader.tsx`; keyboard-test dropdown and mobile behavior. |
| High | Social sign-in buttons appear in login/onboarding without an evident functional handler. | `app/login/page.tsx`; `app/onboard/page.tsx`; verify intended product behavior before modifying or removing. |
| High | Footer labels for Privacy, Terms, and Accessibility point to `/#top`. | `app/sections/footer/Footer.tsx`; destinations require product/legal confirmation. |
| Medium | "Uses Cases" appears in public navigation. | `app/sections/header/GlobalHeader.tsx`; likely copy correction. |
| Medium | Product naming varies among "AtonixCorp Platform", "AtonixCorp Cloud", "Cloud platform", "Container control plane", and uppercase service labels. | Establish a terminology/brand decision before replacing copy. |
| Medium | Breakpoints and mobile navigation differ by feature; networking has an inline 760px breakpoint. | Dashboard CSS, compute CSS, GlobalHeader, and `app/networking/page.tsx`; test desktop through mobile. |
| Medium | Text glyphs are used as controls/icons; accessible labels and target sizing need route-level review. | Example: `app/storage/page.tsx`; do not infer decorative glyphs are accessible controls without testing. |
| Medium | Font references are inconsistent and not loaded through one app-level strategy. | `app/styles/base.css`, `app/styles/Footer.css`, and `app/layout.tsx`. |
| Medium | Dashboard is a large monolithic component, increasing review and regression risk. | `app/dashboard/page.tsx`; component extraction should preserve existing state and navigation behavior. |
| Review | "Service integration pending" and disabled deployment states are visible product states. | `app/compute/[section]/page.tsx`, `app/compute/components/DeploymentSections.tsx`; keep accurate capability limits. |
| Review | "Request access", "Sign in", and "Create account" imply potentially different account flows. | Login/onboarding; resolve canonical terms and flow ownership with product. |
| Review | Asset `dasboard.png`, duplicate section directory names, and two PostCSS configs may be legacy artifacts. | Confirm references, build behavior, and deployment usage before cleanup. |

## Responsive, accessibility, and performance test gaps

- No full route-by-route viewport evidence was available in this audit. Baselines
  are needed at wide desktop, laptop, tablet, and mobile widths.
- Test keyboard-only navigation, visible focus, dropdown dismissal, form labels
  and errors, button names, landmarks, contrast, reduced motion, and dialog
  focus behavior.
- Operational tables need narrow-screen validation; choose controlled
  horizontal scrolling or a suitable responsive representation per route.
- Large page/CSS modules include dashboard, portal, compute deployment sections,
  and dashboard styles. Confirm production bundle impact before splitting.
- `HomePage` composes multiple sections; evaluate route-level loading only after
  measuring actual performance rather than assuming it is a bottleneck.
- Generated `.next` output and `node_modules` are not source assets; confirm
  ignore/build hygiene before counting them as repository payload.

## Terminology decisions needed

Proposed canonical concepts to confirm:

- **Workspace / tenant:** the customer ownership boundary shown to users.
- **Project:** the resource grouping inside a workspace.
- **Member / user:** a person with a role in a workspace or project.
- **Application / service / deployment / resource:** retain only where each
  means a distinct product entity.
- **Platform / cloud / portal:** choose product names for the public site and
  authenticated shell; do not silently treat these as synonyms.

The OAuth availability, legal destinations, and account creation/request-access
flow also need product owners to confirm expected behavior.

## Risk and migration sequence

1. **P0 — approve this inventory and behavior baseline.** Confirm route scope,
   terminology, legal destinations, and which authentication integrations are
   intentionally unavailable. Capture current screenshots and interaction
   behavior for the critical routes.
2. **P1 — establish tokens and accessibility primitives.** Define semantic
   color, type, spacing, radii, focus, motion, and breakpoint tokens; standardize
   font and icon choices. Add global landmarks and skip navigation.
3. **P1 — build the cloud shell and public shell separately.** Use explicit
   route groups where appropriate; preserve the current role, auth, and
   workspace/project behavior. Add keyboard and mobile navigation patterns.
4. **P1 — normalize terminology and controls.** Migrate forms, feedback,
   statuses, and tables with loading/empty/error/disabled states intact.
5. **P2 — migrate route families.** Public pages; authentication; dashboard and
   compute; containers; storage/data; networking. Run targeted functional and
   viewport QA after each shared-component consumer migration.
6. **P2/P3 — cleanup only after reference and build verification.** Review
   duplicate sections, PostCSS configs, assets, and old CSS after all imports,
   dynamic references, and deploy steps are checked.
7. **Final QA.** Route-by-route desktop/tablet/mobile, keyboard, accessibility,
   business-flow regression, and production build checks.

### Shared-change risk

Changes to `app/layout.tsx`, `GlobalHeader`, `Footer`, theme/base CSS, cloud
navigation, typography, icons, shared form primitives, or breakpoints are
high-risk cross-route changes. Each must be tested against all public and
authenticated consumers. Page-local copy or styles are lower risk only when
they do not affect shared imports or existing flows.

## Audit limitations

This report is the required baseline, not a claim that the platform is already
accessible, responsive, or visually consistent. Automated axe checks, visual
screenshots at the required widths, keyboard walkthroughs, production asset
analysis, and route-by-route interaction regression remain to be performed
during approved implementation phases.
