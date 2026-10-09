"use client";
import "../../styles/OperationalRail.css";

function RailIcon({ kind }: { kind: "cloud" | "compute" | "ai" | "network" | "operations" }) {
  if (kind === "cloud") {
    return (
      <svg className="rail-icon-svg" viewBox="0 0 64 64" fill="none" aria-hidden="true">
        <path className="rail-icon-fill" d="M18 43h29a10 10 0 0 0 1-20 16 16 0 0 0-30-3 12 12 0 0 0 0 23Z" />
        <path d="M18 43h29a10 10 0 0 0 1-20 16 16 0 0 0-30-3 12 12 0 0 0 0 23Z" />
        <path d="M23 31h18M27 25l-5 6 5 6M37 25l5 6-5 6" />
        <circle className="rail-icon-dot" cx="14" cy="20" r="3" />
        <circle className="rail-icon-dot" cx="50" cy="17" r="3" />
      </svg>
    );
  }

  if (kind === "compute") {
    return (
      <svg className="rail-icon-svg" viewBox="0 0 64 64" fill="none" aria-hidden="true">
        <rect className="rail-icon-fill" x="18" y="18" width="28" height="28" rx="5" />
        <rect x="18" y="18" width="28" height="28" rx="5" />
        <rect x="25" y="25" width="14" height="14" rx="2" />
        <path d="M25 11v7M32 11v7M39 11v7M25 46v7M32 46v7M39 46v7M11 25h7M11 32h7M11 39h7M46 25h7M46 32h7M46 39h7" />
        <path className="rail-icon-accent" d="M29 29h6v6h-6z" />
      </svg>
    );
  }

  if (kind === "ai") {
    return (
      <svg className="rail-icon-svg" viewBox="0 0 64 64" fill="none" aria-hidden="true">
        <rect className="rail-icon-fill" x="19" y="19" width="26" height="26" rx="6" />
        <rect x="19" y="19" width="26" height="26" rx="6" />
        <path d="M27 34a5 5 0 0 1 10 0v5H27v-5ZM29 28h.01M35 28h.01M24 14v5M32 11v8M40 14v5M24 45v5M32 45v8M40 45v5M14 24h5M11 32h8M14 40h5M45 24h5M45 32h8M45 40h5" />
        <path className="rail-icon-accent" d="M27 39h10" />
      </svg>
    );
  }

  if (kind === "network") {
    return (
      <svg className="rail-icon-svg" viewBox="0 0 64 64" fill="none" aria-hidden="true">
        <path d="M18 19 32 32 47 17M18 19l2 27 12-14 13 15 2-30M20 46l27-1" />
        <circle className="rail-icon-fill" cx="18" cy="19" r="6" />
        <circle className="rail-icon-fill" cx="47" cy="17" r="6" />
        <circle className="rail-icon-fill" cx="32" cy="32" r="7" />
        <circle className="rail-icon-fill" cx="20" cy="46" r="6" />
        <circle className="rail-icon-fill" cx="47" cy="45" r="6" />
        <circle cx="18" cy="19" r="6" />
        <circle cx="47" cy="17" r="6" />
        <circle cx="32" cy="32" r="7" />
        <circle cx="20" cy="46" r="6" />
        <circle cx="47" cy="45" r="6" />
      </svg>
    );
  }

  return (
    <svg className="rail-icon-svg" viewBox="0 0 64 64" fill="none" aria-hidden="true">
      <path className="rail-icon-fill" d="m32 9 19 7v14c0 12-8 21-19 26C21 51 13 42 13 30V16l19-7Z" />
      <path d="m32 9 19 7v14c0 12-8 21-19 26C21 51 13 42 13 30V16l19-7Z" />
      <path d="m23 31 6 6 13-14" />
      <path className="rail-icon-accent" d="M48 12v-6M45 9h6" />
    </svg>
  );
}

export default function OperationalRail() {
  return (
    <section className="operational-rail">
      <div className="rail-wrap">

        <div className="rail-item">
          <RailIcon kind="cloud" />
          <h4>Cloud Control</h4>
          <p>Unified orchestration, identity, policy, and automation services.</p>
        </div>

        <div className="rail-item">
          <RailIcon kind="compute" />
          <h4>Compute & HPC</h4>
          <p>High-performance execution environments for demanding workloads.</p>
        </div>

        <div className="rail-item">
          <RailIcon kind="ai" />
          <h4>AI Infrastructure</h4>
          <p>Accelerated platforms and model-serving foundations for AI workloads.</p>
        </div>

        <div className="rail-item">
          <RailIcon kind="network" />
          <h4>Network & Data</h4>
          <p>Secure connectivity, governed storage, and durable data services.</p>
        </div>

        <div className="rail-item">
          <RailIcon kind="operations" />
          <h4>Secure Operations</h4>
          <p>Delivery, protection, analytics, and operational intelligence.</p>
        </div>

      </div>
    </section>
  );
}
