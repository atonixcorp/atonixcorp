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
            Independent engineering.<br />
            Zero external influence.<br />
            <span className="sov-accent">
              Every engineering decision remains protected and aligned with long‑term infrastructure stability.
            </span>
          </h2>
        </div>

        {/* Sovereignty Grid */}
        <div className="sov-grid">

          <div className="sov-card">
            <div className="sov-number">01</div>
            <h3 className="sov-name">Protected engineering decisions</h3>
            <p className="sov-description">
              All architectural, operational, and security decisions remain internally governed and shielded from external influence.
            </p>
          </div>

          <div className="sov-card">
            <div className="sov-number">02</div>
            <h3 className="sov-name">Secure‑by‑design operations</h3>
            <p className="sov-description">
              Infrastructure is built with embedded security controls, hardened defaults, and continuous protection mechanisms.
            </p>
          </div>

          <div className="sov-card">
            <div className="sov-number">03</div>
            <h3 className="sov-name">Portable infrastructure services</h3>
            <p className="sov-description">
              Platform capabilities operate independently across environments, ensuring portability, resilience, and long‑term continuity.
            </p>
          </div>

          <div className="sov-card">
            <div className="sov-number">04</div>
            <h3 className="sov-name">Long‑term platform ownership</h3>
            <p className="sov-description">
              AtonixCorp maintains full lifecycle control over infrastructure, ensuring stability, predictability, and sovereign operation.
            </p>
          </div>

        </div>
      </div>
    </section>
  );
}
