"use client";

import "../styles/Portal.css";

import TopHeader from "../sections/header/TopHeader";
import GlobalHeader from "../sections/header/GlobalHeader";
import Footer from "../sections/footer/Footer";


export default function PortalLanding() {
  return (
    <main className="portal-landing" id="top">

      <TopHeader />
      <GlobalHeader />

      <section className="portal-hero">
        <div className="portal-hero-wrap">
          <div className="portal-hero-copy">
            <p className="portal-index"><span /> AtonixCorp Portal</p>

            <h1 className="portal-title">
              Infrastructure for
              <span className="portal-title-accent"> what comes next.</span>
            </h1>

            <p className="portal-description">
              Enter a connected platform being built for AI, research, advanced engineering,
              enterprise applications, and the critical systems of tomorrow.
            </p>

            <div className="portal-actions">
              <a href="/login" className="portal-btn primary">
                Sign in <span aria-hidden="true">→</span>
              </a>
              <a href="/onboard" className="portal-btn secondary">Request access</a>
            </div>
          </div>

          <aside className="portal-pipeline" aria-label="Illustrative DevSecOps delivery pipeline">
            <div className="portal-preview-heading">
              <div>
                <span className="portal-preview-kicker">SECURE DELIVERY, BUILT IN</span>
                <h2>DevSecOps pipeline</h2>
              </div>
              <span className="portal-pipeline-status">REFERENCE FLOW</span>
            </div>
            <div className="portal-pipeline-flow">
              <div className="portal-pipeline-step">
                <span className="portal-pipeline-step-icon" aria-hidden="true">01</span>
                <span className="portal-pipeline-step-copy"><strong>Plan & code</strong><small>Versioned source and peer review</small></span>
                <span className="portal-pipeline-chip">SOURCE</span>
              </div>
              <div className="portal-pipeline-step">
                <span className="portal-pipeline-step-icon" aria-hidden="true">02</span>
                <span className="portal-pipeline-step-copy"><strong>Build & package</strong><small>Repeatable builds and artifacts</small></span>
                <span className="portal-pipeline-chip">CONTINUOUS</span>
              </div>
              <div className="portal-pipeline-step security">
                <span className="portal-pipeline-step-icon" aria-hidden="true">03</span>
                <span className="portal-pipeline-step-copy"><strong>Security gates</strong><small>Code, dependency, and secret analysis</small></span>
                <span className="portal-pipeline-chip">SHIFT LEFT</span>
              </div>
              <div className="portal-pipeline-step">
                <span className="portal-pipeline-step-icon" aria-hidden="true">04</span>
                <span className="portal-pipeline-step-copy"><strong>Test & approve</strong><small>Quality, policy, and release review</small></span>
                <span className="portal-pipeline-chip">ASSURANCE</span>
              </div>
              <div className="portal-pipeline-step deploy">
                <span className="portal-pipeline-step-icon" aria-hidden="true">05</span>
                <span className="portal-pipeline-step-copy"><strong>Deploy & observe</strong><small>Controlled release and telemetry</small></span>
                <span className="portal-pipeline-chip">OPERATE</span>
              </div>
            </div>
            <p className="portal-preview-foot">
              <strong>Security is continuous.</strong>
              <span>Controls and policy checks are considered throughout delivery.</span>
            </p>
          </aside>
        </div>
      </section>

      <section className="portal-capabilities">
        <div className="portal-cap-section">
          <header className="portal-cap-heading">
            <p className="portal-cap-eyebrow">A platform with purpose</p>
            <h2>One foundation. <span>Many frontiers.</span></h2>
            <p>Explore the domains coming together to support the next generation of applications and infrastructure.</p>
          </header>
          <div className="portal-cap-wrap">
            <article className="portal-cap-card">
              <span className="portal-cap-number">01 / CONTROL</span>
              <h3>Cloud Control</h3>
              <p>Coordinate infrastructure, identity, and automation through a unified platform foundation.</p>
            </article>

            <article className="portal-cap-card">
              <span className="portal-cap-number">02 / COMPUTE</span>
              <h3>Compute & HPC</h3>
              <p>Support demanding workloads, advanced research, simulation, and engineering.</p>
            </article>

            <article className="portal-cap-card">
              <span className="portal-cap-number">03 / INTELLIGENCE</span>
              <h3>AI Infrastructure</h3>
              <p>Build a foundation for accelerated computing, model development, and AI services.</p>
            </article>

            <article className="portal-cap-card">
              <span className="portal-cap-number">04 / CONNECTIVITY</span>
              <h3>Network & Data</h3>
              <p>Connect applications to resilient data services, storage, and secure networks.</p>
            </article>

            <article className="portal-cap-card">
              <span className="portal-cap-number">05 / TRUST</span>
              <h3>Secure Operations</h3>
              <p>Bring protection, observability, delivery, and operational insight together.</p>
            </article>
          </div>
        </div>
      </section>

      <Footer />

    </main>
  );
}
