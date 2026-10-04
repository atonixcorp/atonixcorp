"use client";

import "../../styles/PlatformOverview.css";
import Link from "next/link";

export default function PlatformOverview() {
  return (
    <section className="platform-overview" id="platform-overview">
      <div className="platform-overview-wrap">
        <p className="platform-index">01 / Platform Overview</p>

        <div className="platform-overview-content">
          <div className="platform-overview-heading">
            <h2 className="platform-title">
              A foundation for the next generation of complex systems.
            </h2>
            <p className="platform-lead">
              AtonixCorp is building a connected cloud platform to support the work
              behind tomorrow’s most demanding systems—from AI and scientific discovery
              to robotics, aerospace, flight, energy, and enterprise services.
            </p>
          </div>

          <div className="platform-overview-story">
            <p className="platform-description">
              By bringing compute, networking, data, security, and operations together,
              the platform is designed to help teams develop, simulate, deploy, and
              operate complex workloads with room to scale.
            </p>
            <div className="platform-overview-actions">
              <Link href="/platform" className="platform-action">
                Explore the full platform <span aria-hidden="true">→</span>
              </Link>
              <Link href="#mission" className="platform-mission-link">
                Explore our mission
              </Link>
            </div>
          </div>
        </div>
      </div>
    </section>
  );
}
