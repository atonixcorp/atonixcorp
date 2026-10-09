"use client";
import "../../styles/HeroCorporate.css";

export default function Overview() {
  const capabilities = [
    {
      code: "01",
      name: "Cloud Control",
      category: "Orchestration & platform services",
      description:
        "Control‑plane services that coordinate infrastructure, identities, policies, automation, and distributed operations.",
      tags: ["Control plane", "Orchestration", "Identity"],
    },
    {
      code: "02",
      name: "Compute & HPC",
      category: "Workload execution",
      description:
        "Compute agents, cluster scheduling, hardware‑aware placement, and high‑performance environments for scientific and enterprise workloads.",
      tags: ["Compute agents", "HPC scheduling", "Hardware control"],
    },
    {
      code: "03",
      name: "Network & Data",
      category: "Connected infrastructure",
      description:
        "Secure networking, database services, object storage, and governed data flows for systems that need durable access and visibility.",
      tags: ["Networking", "DBaaS", "Storage"],
    },
    {
      code: "04",
      name: "Secure Operations",
      category: "Delivery, protection & insight",
      description:
        "Developer delivery services, security controls, observability, analytics, and audit capabilities for dependable platform operations.",
      tags: ["DevOps", "Security", "Analytics"],
    },
  ];

  const sovereignty = [
    "Protected engineering decisions",
    "Secure‑by‑design operations",
    "Portable infrastructure services",
    "Long‑term platform ownership",
  ];

  const roadmap = [
    {
      phase: "Phase I",
      title: "Foundation & Stabilization",
      description:
        "Core infrastructure, cloud systems, HPC environments, and governance frameworks.",
    },
    {
      phase: "Phase II",
      title: "Infrastructure Expansion",
      description:
        "Multi‑region cloud, HPC, AI, and energy systems at scale.",
    },
    {
      phase: "Phase III",
      title: "Global Integration",
      description:
        "International enterprise, research, and institutional collaboration.",
    },
    {
      phase: "Phase IV",
      title: "International Positioning",
      description:
        "Expanded developer ecosystems and media intelligence capabilities.",
    },
    {
      phase: "Phase V",
      title: "Autonomous Infrastructure Era",
      description:
        "Self‑governing cloud, HPC, and AI operational frameworks.",
    },
    {
      phase: "Phase VI",
      title: "Sovereign Technology Era",
      description:
        "Global operational independence and enduring technological leadership.",
    },
  ];

  return (
    <section className="overview">

      {/* 01 — Platform Overview */}
      <div className="wrap section-block">
        <p className="section-index">01 / Platform Overview</p>

        <h2 className="section-title">
          Infrastructure capabilities designed to operate as a unified platform.
        </h2>

        <p className="section-text">
          AtonixCorp provides a connected foundation for cloud orchestration,
          compute, networking, data services, security, and developer operations.
          Each capability is engineered to function independently while integrating
          seamlessly through shared control, identity, telemetry, and automation layers.
        </p>

        <p className="section-text">
          This approach enables consistent operations, predictable performance,
          and a platform architecture built for modern workloads and long‑term reliability.
        </p>

        <a className="section-link" href="#mission">Learn more about our operating mandate →</a>
      </div>

      {/* 02 — Mission */}
      <div className="wrap section-block">
        <p className="section-index">02 / Mission</p>

        <div className="mission-grid">
          <div className="mission-heading">
            <h2>
              Build.<br />
              Protect.<br />
              <em>Innovate.</em><br />
              Scale.
            </h2>
          </div>

          <div className="mission-detail">
            <p className="section-text">
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
      </div>

      {/* 03 — Platform Capabilities */}
      <div className="wrap section-block">
        <p className="section-index">03 / Platform Capabilities</p>

        <h2 className="section-title">
          One platform.<br />
          <em>Four core capabilities.</em>
        </h2>

        <p className="section-text">
          Focused infrastructure domains, designed to operate together.
        </p>

        <div className="capabilities-grid">
          {capabilities.map((cap) => (
            <article className="cap-card" key={cap.code}>
              <div className="cap-top">
                <span className="cap-code">{cap.code}</span>
                <span className="cap-arrow">↗</span>
              </div>

              <h3 className="cap-name">{cap.name}</h3>
              <p className="cap-category">{cap.category}</p>
              <p className="cap-description">{cap.description}</p>

              <div className="cap-tags">
                {cap.tags.map((tag) => (
                  <span key={tag}>{tag}</span>
                ))}
              </div>
            </article>
          ))}
        </div>
      </div>

      {/* 04 — Technology Sovereignty */}
      <div className="wrap section-block">
        <p className="section-index">04 / Technology Sovereignty</p>

        <h2 className="section-title">
          Independent engineering.<br />
          Zero external influence.
        </h2>

        <p className="section-text">
          Every engineering decision remains protected and aligned with long‑term
          infrastructure stability.
        </p>

        <div className="sovereignty-grid">
          {sovereignty.map((item, index) => (
            <div className="sovereignty-card" key={index}>
              <span className="sovereignty-number">
                {String(index + 1).padStart(2, "0")}
              </span>
              <p>{item}</p>
            </div>
          ))}
        </div>
      </div>

      {/* 06 — Strategic Roadmap */}
      <div className="wrap section-block">
        <p className="section-index">06 / Strategic Roadmap</p>

        <h2 className="section-title">
          Thirty years.<br />
          Six phases.
        </h2>

        <p className="section-text">
          A future built in deliberate phases, advancing independence and infrastructure stability at each horizon.
        </p>

        <div className="roadmap-grid">
          {roadmap.map((phase, index) => (
            <div className="roadmap-card" key={index}>
              <h4>{phase.phase}</h4>
              <p className="roadmap-title">{phase.title}</p>
              <p className="roadmap-description">{phase.description}</p>
            </div>
          ))}
        </div>
      </div>

    </section>
  );
}
