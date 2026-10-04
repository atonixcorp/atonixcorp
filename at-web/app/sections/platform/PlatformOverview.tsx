"use client";

import "../../styles/PlatformOverview.css";
import Link from "next/link";

export default function PlatformOverview() {
  return (
    <section className="platform-overview" id="platform-overview">
      <div className="platform-overview-wrap">

        <p className="platform-index">01 / Platform Overview</p>

        <h2 className="platform-title">
          Infrastructure that works better together.
        </h2>

        <p className="platform-description">
          AtonixCorp brings cloud orchestration, compute, networking, data services,
          security, and developer operations into one connected foundation. Each
          capability can stand on its own and works through shared control, identity,
          telemetry, and automation.
        </p>

        <p className="platform-description">
          The result is a consistent operating experience built to support modern
          workloads and long-term reliability.
        </p>

        <Link href="#mission" className="platform-action">
          Explore our mission <span aria-hidden="true">→</span>
        </Link>

      </div>
    </section>
  );
}
