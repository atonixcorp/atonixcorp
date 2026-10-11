# AtonixCorp DevSecOps Platform

## Complete UI Architecture, User Flows, API Contracts, Engineering Requirements, and Definition of Done

This README defines the complete AtonixCorp DevSecOps experience from **Run Pipeline** through **Settings**.

It provides product engineers, frontend developers, backend engineers, platform engineers, security engineers, QA engineers, SRE teams, and product designers with a shared implementation reference for:

- Navigation
- User workflows
- UI states
- API contracts
- Event contracts
- Permissions
- Security controls
- Observability
- Testing
- Release acceptance

---

## Table of Contents

1. [Platform Scope and Navigation](#1-platform-scope-and-navigation)
2. [Run Pipeline and Project Onboarding](#2-run-pipeline-and-project-onboarding)
3. [Pipelines  - Gerrit3. [Pipelines, Deployments, and Artifacts](#3-pipelines-deployments-and-artifacts)
  - Harness
  - Azure Repos
  - Custom Git URL

## 2.3 Authentication

- Use provider OAuth or application installation where supported.
- Display all requested permissions before redirecting the user.
- Store provider tokens in the secrets service.
- Never store provider tokens in browser local storage or session storage.
- Support token rotation, revocation, and provider reconnection.
- Record connection activity in the audit log.

## 2.4 Repository Selection

The repository selector must provide:

- Search
- Pagination
- Repository name
- Repository type
- Default branch
- Access status
- Selected count
- Bulk selection
- Import validation
- Permission errors
- Empty-state handling

## 2.5 Import Orchestration

The repository import process must run the following stages:

1. Clone repository.
2. Detect programming languages.
3. Detect Dockerfiles.
4. Detect infrastructure-as-code files.
5. Generate an SBOM.
6. Run security scans.
7. Prepare pipeline templates.
8. Validate repository permissions.
9. Register repository webhooks.
10. Mark the import as complete.

Progress must be streamed through either:

- Server-Sent Events
- WebSocket

The import process must persist resumable job state.

## 2.6 Pipeline Setup

Users can select one of the following configuration methods:

### Standard Pipeline

- Auto-detected build steps
- Auto-detected test steps
- Auto-detected security checks
- Recommended default configuration

### Advanced Pipeline

- Editable stages
- Editable jobs
- Custom execution order
- Custom approval gates

### Custom YAML

- Upload YAML
- Paste YAML
- Validate YAML schema
- Preview generated pipeline
- Detect missing secrets
- Detect incompatible runners

Before pipeline creation, validate:

- Pipeline schema
- User permissions
- Referenced secrets
- Runner compatibility
- Provider connection
- Repository access
- Branch configuration
- Security requirements

## 2.7 Pipeline Ready State

Confirm that:

- Repository is connected.
- Webhooks are active.
- CI/CD tokens are configured.
- Security scanners are enabled.
- Pipeline configuration is valid.

Available actions:

- `Run Pipeline Now`
- `View Pipeline Configuration`

## 2.8 Core Endpoints

```text
POST /projects
POST /connections/source
GET  /repositories
POST /imports
GET  /jobs/{id}
POST /pipelines
POST /pipelines/{id}/runs
4. [Infrastructure, Kubernetes, and Applications](#4-infrastructure-kubernetes-and-applications)
5. [Security Center and AI Operations](#5-security-center-and-ai-operations)
6. [Automation Workspace](#6-automation-workspace)
7. [Observability and Cost Management](#7-observability-and-cost-management)
8. [Reports, Settings, and Administration](#8-reports-settings-and-administration)
9. [Engineering Architecture and Data Contracts](#9-engineering-architecture-and-data-contracts)
10. [Security, Quality, and Release Requirements](#10-security-quality-and-release-requirements)
11. [Definition of Done](#11-definition-of-done)

---

# 1. Platform Scope and Navigation

- **Primary navigation order:** Dashboard → Pipelines → Deployments → Infrastructure → Kubernetes → Applications → Security → AI Operations → Automation → Observability → Cost Management → Reports → Settings.
- **Global shell:** Product wordmark, workspace selector, universal search, environment indicator, notifications, help, user menu, breadcrumb, page title, contextual actions, and persistent left navigation.
- **Shared states:** Loading, empty, onboarding, syncing, populated, degraded, permission denied, partial failure, offline, and destructive-action confirmation.
- **Design baseline:** Responsive desktop-first layout, accessible color contrast, keyboard navigation, visible focus indicators, tooltips for icons, timestamps in the user’s locale, and skeleton loaders instead of layout shifts.
- **Tenancy:** Every API request and event must carry organization, workspace, project, environment, actor, correlation ID, and authorization context.

---

# 2. Run Pipeline and Project Onboarding

## 2.1 Run Pipeline Empty State

**Frame name:** `RunPipeline_EmptyState`

- Detect when no project exists.
- Display the following actions:
  - `Create Project`
  - `Import Project`
- Do not display pipeline controls until a project exists.
- Explain that a project must be created or imported before a pipeline can run.

## 2.2 Project Setup Modal

**Frame name:** `ProjectSetup_Modal`

Allow the user to:

- Create a native AtonixCorp project.
- Import a project from:
  - GitHub
  - GitLab
  - Bitbucket
