"use client";

import "../../styles/PlatformOverview.css";
import Link from "next/link";

export default function PlatformOverview() {
  return (
    <section className="platform-overview">
      <div className="platform-overview-wrap">

        <p className="platform-index">01 / Platform Overview</p>

        <h1 className="platform-title">
          Infrastructure capabilities designed to operate as a unified platform.
        </h1>

        <p className="platform-description">
          AtonixCorp provides a connected foundation for cloud orchestration, compute,
          networking, data services, security, and developer operations. Each capability
          is engineered to function independently while integrating seamlessly through
          shared control, identity, telemetry, and automation layers.
        </p>

        <p className="platform-description">
          This approach enables consistent operations, predictable performance, and a
          platform architecture built for modern workloads and long‑term reliability.
        </p>

        <Link href="/mandate" className="platform-action">
          Learn more about our operating mandate →
        </Link>

      </div>
    </section>
  );
}
