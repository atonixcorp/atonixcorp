"use client";

import "../../styles/CoreCapabilities.css";

export default function CoreCapabilities() {
  return (
    <section className="capabilities" id="capabilities">
      <div className="cap-wrap">

        {/* Section Header */}
        <div className="cap-header">
          <p className="cap-index">03 / Platform Capabilities</p>

          <h2 className="cap-title">
            One platform.<br />
            Four core capabilities.<br />
            <span className="cap-accent">Focused infrastructure domains, designed to operate together.</span>
          </h2>
        </div>

        {/* Capability Grid */}
        <div className="cap-grid">

          {/* 01 — Cloud Control */}
          <div className="cap-card">
            <div className="cap-number">01 ↗</div>
            <h3 className="cap-name">Cloud Control</h3>
            <p className="cap-summary">Orchestration & platform services</p>

            <p className="cap-description">
              Control‑plane services that coordinate infrastructure, identities, policies,
              automation, and distributed operations.
            </p>

            <ul className="cap-list">
              <li>Control plane</li>
              <li>Orchestration</li>
              <li>Identity</li>
            </ul>
          </div>

          {/* 02 — Compute & HPC */}
          <div className="cap-card">
            <div className="cap-number">02 ↗</div>
            <h3 className="cap-name">Compute & HPC</h3>
            <p className="cap-summary">Workload execution</p>

            <p className="cap-description">
              Compute agents, cluster scheduling, hardware‑aware placement, and
              high‑performance environments for scientific and enterprise workloads.
            </p>

            <ul className="cap-list">
              <li>Compute agents</li>
              <li>HPC scheduling</li>
              <li>Hardware control</li>
            </ul>
          </div>

          {/* 03 — Network & Data */}
          <div className="cap-card">
            <div className="cap-number">03 ↗</div>
            <h3 className="cap-name">Network & Data</h3>
            <p className="cap-summary">Connected infrastructure</p>

            <p className="cap-description">
              Secure networking, database services, object storage, and governed data flows
              for systems that need durable access and visibility.
            </p>

            <ul className="cap-list">
              <li>Networking</li>
              <li>DBaaS</li>
              <li>Storage</li>
            </ul>
          </div>

          {/* 04 — Secure Operations */}
          <div className="cap-card">
            <div className="cap-number">04 ↗</div>
            <h3 className="cap-name">Secure Operations</h3>
            <p className="cap-summary">Delivery, protection & insight</p>

            <p className="cap-description">
              Developer delivery services, security controls, observability, analytics,
              and audit capabilities for dependable platform operations.
            </p>

            <ul className="cap-list">
              <li>DevOps</li>
              <li>Security</li>
              <li>Analytics</li>
            </ul>
          </div>

        </div>
      </div>
    </section>
  );
}
