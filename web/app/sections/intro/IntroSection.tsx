"use client";

import "../../styles/IntroSection.css";

export default function IntroSection() {
  return (
    <section className="intro section wrap" id="about">
      <p className="section-index">01 / Platform Overview</p>

      <div className="intro-content">
        <h2 className="intro-title">
          Infrastructure capabilities designed to operate as a unified platform.
        </h2>

        <div className="intro-body">
          <p>
            AtonixCorp provides a connected foundation for cloud orchestration,
            compute, networking, data services, security, and developer operations.
            Each capability is engineered to function independently while integrating
            seamlessly through shared control, identity, telemetry, and automation layers.
          </p>

          <p>
            This unified approach enables consistent operations, predictable performance,
            and a platform architecture built for modern workloads and long‑term reliability.
          </p>

          <a className="text-link dark-link" href="#mission">
            Learn more about our operating mandate →
          </a>
        </div>
      </div>
    </section>
  );
}
