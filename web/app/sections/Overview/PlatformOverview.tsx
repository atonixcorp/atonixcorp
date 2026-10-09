import React from "react";
import "./PlatformOverview.css"; // optional custom styling

export default function PlatformOverview() {
  return (
    <div className="platform-overview-page">

      {/* Page Header */}
      <header className="overview-header">
        <h1>AtonixCorp Cloud Platform</h1>
        <p>
          Unified compute, storage, networking, HPC, and scientific workloads —
          powered by the AtonixCorp control plane and OrcaCloud orchestration.
        </p>
      </header>

      {/* Navigation Tabs */}
      <nav className="overview-nav">
        <ul>
          <li><a href="#overview">Overview</a></li>
          <li><a href="#devtools">Development Tools</a></li>
          <li><a href="#api">API Reference</a></li>
          <li><a href="#cli">CLI Tools</a></li>
          <li><a href="#sdks">SDKs</a></li>
          <li><a href="#docs">Documentation</a></li>
        </ul>
      </nav>

      {/* Overview Section */}
      <section id="overview" className="section-block">
        <h2>What is AtonixCorp Cloud?</h2>
        <p>
          AtonixCorp is a modern cloud operating system that manages compute,
          storage, networking, HPC clusters, scientific workloads, and
          automation through a unified orchestration engine. It provides
          dashboards, CLI tools, SDKs, and APIs for developers and researchers.
        </p>
      </section>

      {/* Development Tools */}
      <section id="devtools" className="section-block">
        <h2>Development Tools</h2>
        <p>
          Build cloud-native applications using DevTools: SDKs, CLI, templates,
          and automation workflows.
        </p>
      </section>

      {/* API Reference */}
      <section id="api" className="section-block">
        <h2>API Reference</h2>
        <p>
          Explore the full OpenAPI specification for compute, network, storage,
          HPC, scientific workloads, and orchestration.
        </p>
      </section>

      {/* CLI Tools */}
      <section id="cli" className="section-block">
        <h2>CLI Tools</h2>
        <p>
          Manage your cloud resources using the AtonixCorp CLI with commands for
          compute, network, storage, orchestration, and HPC.
        </p>
      </section>

      {/* SDKs */}
      <section id="sdks" className="section-block">
        <h2>SDKs</h2>
        <p>
          Official SDKs available for Go, Python, JavaScript, Java, and more.
        </p>
      </section>

      {/* Documentation */}
      <section id="docs" className="section-block">
        <h2>Documentation Hub</h2>
        <p>
          Access guides, tutorials, architecture diagrams, and workflow
          examples for building on AtonixCorp Cloud.
        </p>
      </section>

    </div>
  );
}
