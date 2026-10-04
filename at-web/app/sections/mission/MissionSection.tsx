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
            We engineer infrastructure that is autonomous, secure, energy‑efficient,
            and capable of supporting the world’s most advanced digital operations.
          </p>

          <ul className="mission-list">
            <li>Power enterprise workloads with precision and reliability</li>
            <li>Secure critical infrastructure against digital and physical threats</li>
            <li>Accelerate research through high‑performance computing</li>
            <li>Enable autonomous operations across distributed environments</li>
          </ul>
        </div>

      </div>
    </section>
  );
}
