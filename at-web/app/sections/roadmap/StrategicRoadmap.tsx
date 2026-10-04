"use client";

import "../../styles/StrategicRoadmap.css";

export default function StrategicRoadmap() {
  return (
    <section className="roadmap" id="roadmap">
      <div className="roadmap-wrap">

        {/* Section Header */}
        <div className="roadmap-header">
          <p className="roadmap-index">05 / Strategic Roadmap</p>

          <h2 className="roadmap-title">
            Thirty years.<br />
            Six phases.<br />
            <span className="roadmap-accent">
              A future built in deliberate phases, advancing independence and infrastructure stability at each horizon.
            </span>
          </h2>
        </div>

        {/* Roadmap Timeline */}
        <div className="roadmap-timeline">

          {/* Phase I */}
          <div className="phase-card">
            <div className="phase-number">Phase I</div>
            <h3 className="phase-name">Foundation & Stabilization</h3>
            <p className="phase-description">
              Core infrastructure, cloud systems, HPC environments, and governance frameworks.
            </p>
          </div>

          {/* Phase II */}
          <div className="phase-card">
            <div className="phase-number">Phase II</div>
            <h3 className="phase-name">Infrastructure Expansion</h3>
            <p className="phase-description">
              Multi‑region cloud, HPC, AI, and energy systems at scale.
            </p>
          </div>

          {/* Phase III */}
          <div className="phase-card">
            <div className="phase-number">Phase III</div>
            <h3 className="phase-name">Global Integration</h3>
            <p className="phase-description">
              International enterprise, research, and institutional collaboration.
            </p>
          </div>

          {/* Phase IV */}
          <div className="phase-card">
            <div className="phase-number">Phase IV</div>
            <h3 className="phase-name">International Positioning</h3>
            <p className="phase-description">
              Expanded developer ecosystems and media intelligence capabilities.
            </p>
          </div>

          {/* Phase V */}
          <div className="phase-card">
            <div className="phase-number">Phase V</div>
            <h3 className="phase-name">Autonomous Infrastructure Era</h3>
            <p className="phase-description">
              Self‑governing cloud, HPC, and AI operational frameworks.
            </p>
          </div>

          {/* Phase VI */}
          <div className="phase-card">
            <div className="phase-number">Phase VI</div>
            <h3 className="phase-name">Sovereign Technology Era</h3>
            <p className="phase-description">
              Global operational independence and enduring technological leadership.
            </p>
          </div>

        </div>
      </div>
    </section>
  );
}
