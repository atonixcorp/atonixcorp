"use client";

import "../../styles/TechnologySovereignty.css";

export default function TechnologySovereignty() {
  return (
    <section className="sovereignty" id="sovereignty">
      <div className="sov-wrap">

        {/* Section Header */}
        <div className="sov-header">
          <p className="sov-index">04 / Technology Sovereignty</p>

          <h2 className="sov-title">
            Independent engineering. Long-term control.
            <span className="sov-accent">
              Build and operate infrastructure with clear ownership, secure foundations, and room to evolve.
            </span>
          </h2>
        </div>

        {/* Sovereignty Grid */}
        <div className="sov-grid">

          <div className="sov-card">
            <div className="sov-number">01</div>
            <h3 className="sov-name">Protected engineering decisions</h3>
            <p className="sov-description">
              Keep architecture, operations, and security decisions accountable to the platform’s long-term goals.
            </p>
          </div>

          <div className="sov-card">
            <div className="sov-number">02</div>
            <h3 className="sov-name">Secure‑by‑design operations</h3>
            <p className="sov-description">
              Make security part of the foundation with hardened defaults and continuous protection.
            </p>
          </div>

          <div className="sov-card">
            <div className="sov-number">03</div>
            <h3 className="sov-name">Portable infrastructure services</h3>
            <p className="sov-description">
              Move platform capabilities across environments while preserving portability and resilience.
            </p>
          </div>

          <div className="sov-card">
            <div className="sov-number">04</div>
            <h3 className="sov-name">Long‑term platform ownership</h3>
            <p className="sov-description">
              Maintain lifecycle ownership to support stable, predictable infrastructure over time.
            </p>
          </div>

        </div>
      </div>
    </section>
  );
}
