# Frontend audit baseline

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
