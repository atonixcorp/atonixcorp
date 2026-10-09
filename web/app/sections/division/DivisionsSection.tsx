"use client";
import "../../styles/DivisionsSection.css";
export default function DivisionsSection() {
  const divisions = [
    {
      code: "01",
      name: "Cloud Control",
      category: "Orchestration & platform services",
      description:
        "Control-plane services that coordinate infrastructure, identities, policies, automation, and distributed operations.",
      capabilities: ["Control plane", "Orchestration", "Identity"],
    },
    {
      code: "02",
      name: "Compute & HPC",
      category: "Workload execution",
      description:
        "Compute agents, cluster scheduling, hardware-aware placement, and high-performance environments for scientific and enterprise workloads.",
      capabilities: ["Compute agents", "HPC scheduling", "Hardware control"],
    },
    {
      code: "03",
      name: "Network & Data",
      category: "Connected infrastructure",
      description:
        "Secure networking, database services, object storage, and governed data flows for systems that need durable access and visibility.",
      capabilities: ["Networking", "DBaaS", "Storage"],
    },
    {
      code: "04",
      name: "Secure Operations",
      category: "Delivery, protection & insight",
      description:
        "Developer delivery services, security controls, observability, analytics, and audit capabilities for dependable platform operations.",
      capabilities: ["DevOps", "Security", "Analytics"],
    },
  ];

  return (
    <section className="section divisions" id="divisions">
      <div className="wrap">
        <p className="section-index">03 / Platform capabilities</p>

        <div className="section-heading">
          <h2>
            One platform.<br />
            <em>Four core capabilities.</em>
          </h2>
          <p>Focused infrastructure domains, designed to operate together.</p>
        </div>

        <div className="division-grid">
          {divisions.map((division) => (
            <article className="division-card" key={division.name}>
              <div className="division-top">
                <span>{division.code}</span>
                <span className="arrow">↗</span>
              </div>

              <h3>{division.name}</h3>
              <p className="division-category">{division.category}</p>
              <p>{division.description}</p>

              <div className="capabilities">
                {division.capabilities.map((cap) => (
                  <span key={cap}>{cap}</span>
                ))}
              </div>
            </article>
          ))}
        </div>
      </div>
    </section>
  );
}
