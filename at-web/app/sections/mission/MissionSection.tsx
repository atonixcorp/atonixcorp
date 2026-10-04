"use client";

import "../../styles/MissionSection.css";

export default function MissionSection() {
  return (
    <section className="mission" id="mission">
      <div className="mission-wrap">

        {/* Left Heading */}
        <div className="mission-heading">
          <p className="mission-index">02 / Mission</p>

          <h2 className="mission-title">
            Build.<br />
            Protect.<br />
            <span className="mission-accent">Innovate.</span><br />
            Scale.
          </h2>
        </div>

        {/* Right Details */}
        <div className="mission-detail">
          <p className="mission-summary">
            We build dependable, secure infrastructure for the systems people and
            organizations rely on every day.
          </p>

          <ul className="mission-list">
            <li>Run enterprise workloads with predictable performance</li>
            <li>Protect infrastructure with security built into every layer</li>
            <li>Advance research with high-performance computing</li>
            <li>Simplify operations across distributed environments</li>
          </ul>
        </div>

      </div>
    </section>
  );
}
