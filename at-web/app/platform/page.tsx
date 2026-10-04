import Link from "next/link";
import "../styles/PlatformPage.css";
import TopHeader from "../sections/header/TopHeader";
import GlobalHeader from "../sections/header/GlobalHeader";
import Footer from "../sections/footer/Footer";

const capabilities = [
  {
    number: "01",
    title: "Cloud Control",
    description:
      "Bring identity, orchestration, policy, and automation into a coordinated operating foundation.",
  },
  {
    number: "02",
    title: "Compute & HPC",
    description:
      "Support demanding applications, large-scale simulation, scientific work, and advanced engineering.",
  },
  {
    number: "03",
    title: "AI Infrastructure",
    description:
      "Create room for accelerated workloads, AI development, and intelligent services to grow.",
  },
  {
    number: "04",
    title: "Network & Data",
    description:
      "Connect workloads to dependable data services, storage, and secure communication.",
  },
  {
    number: "05",
    title: "Secure Operations",
    description:
      "Build protection, observability, and operational insight into the platform experience.",
  },
  {
    number: "06",
    title: "DevSecOps",
    description:
      "Carry secure delivery practices from source and build through release and operation.",
  },
];

const workloads = [
  "Artificial intelligence",
  "Scientific research",
  "Robotics",
  "Aerospace & flight",
  "Energy systems",
  "Enterprise applications",
];

export default function PlatformPage() {
  return (
    <main className="platform-page" id="top">
      <TopHeader />
      <GlobalHeader />

      <section className="platform-page-hero">
        <div className="platform-page-wrap">
          <p className="platform-page-eyebrow"><span /> ATONIXCORP / PLATFORM OVERVIEW</p>
          <h1>
            A connected cloud foundation
            <span>for a world of workloads.</span>
          </h1>
          <p className="platform-page-lead">
            AtonixCorp is building a unified platform to help teams develop, simulate,
            deploy, and operate the complex systems the future depends on.
          </p>
          <div className="platform-page-actions">
            <Link href="/contact" className="platform-page-primary">
              Talk with our team <span aria-hidden="true">→</span>
            </Link>
            <a href="#platform-capabilities" className="platform-page-secondary">
              Explore platform capabilities
            </a>
          </div>
          <div className="platform-page-hero-meta">
            <span>COMPUTE</span><i />
            <span>DATA</span><i />
            <span>NETWORKS</span><i />
            <span>SECURITY</span><i />
            <span>OPERATIONS</span>
          </div>
        </div>
      </section>

      <section className="platform-page-intro">
        <div className="platform-page-wrap platform-page-intro-grid">
          <div>
            <p className="platform-page-index">01 / The platform</p>
            <h2>
              More than infrastructure.
              <span>A foundation to build on.</span>
            </h2>
          </div>
          <div className="platform-page-copy">
            <p>
              Tomorrow’s most demanding work—from AI and scientific discovery to
              robotics, aerospace, flight, energy, and enterprise services—depends on
              systems that can work together. AtonixCorp is bringing compute,
              networking, data, security, and operations into one evolving cloud
              foundation.
            </p>
            <p>
              The goal is a more consistent way to move from an idea or experiment to
              simulation, deployment, and ongoing operation, with a platform designed
              to adapt as workloads and ambitions grow.
            </p>
          </div>
        </div>
      </section>

      <section className="platform-page-capabilities" id="platform-capabilities">
        <div className="platform-page-wrap">
          <header className="platform-page-section-heading">
            <p className="platform-page-index">02 / Platform capabilities</p>
            <h2>Connected capabilities.<span>One operating foundation.</span></h2>
            <p>
              Each domain has a distinct role. Together, they are designed to support
              workloads across their full lifecycle.
            </p>
          </header>
          <div className="platform-page-capability-grid">
            {capabilities.map((capability) => (
              <article className="platform-page-capability" key={capability.number}>
                <span className="platform-page-capability-number">{capability.number}</span>
                <h3>{capability.title}</h3>
                <p>{capability.description}</p>
                <span className="platform-page-card-line" aria-hidden="true" />
              </article>
            ))}
          </div>
        </div>
      </section>

      <section className="platform-page-lifecycle">
        <div className="platform-page-wrap">
          <header className="platform-page-section-heading">
            <p className="platform-page-index">03 / Workload lifecycle</p>
            <h2>From first experiment<span>to dependable operation.</span></h2>
          </header>
          <div className="platform-page-lifecycle-grid">
            <article>
              <span>01 — DEVELOP</span>
              <h3>Build and prepare</h3>
              <p>Give teams a foundation for developing applications, services, and research workflows.</p>
            </article>
            <article>
              <span>02 — SIMULATE</span>
              <h3>Test at meaningful scale</h3>
              <p>Support compute-intensive engineering, scientific analysis, and demanding workloads.</p>
            </article>
            <article>
              <span>03 — OPERATE</span>
              <h3>Deploy with confidence</h3>
              <p>Bring secure operations and ongoing visibility to workloads as they move into use.</p>
            </article>
          </div>
        </div>
      </section>

      <section className="platform-page-workloads">
        <div className="platform-page-wrap platform-page-workloads-grid">
          <div>
            <p className="platform-page-index">04 / Designed for ambitious work</p>
            <h2>One platform.<span>Many frontiers.</span></h2>
            <p>
              A flexible foundation for teams working across disciplines and building
              systems with demanding requirements.
            </p>
          </div>
          <ul>
            {workloads.map((workload, index) => (
              <li key={workload}>
                <span>{String(index + 1).padStart(2, "0")}</span>
                {workload}
                <b aria-hidden="true">↗</b>
              </li>
            ))}
          </ul>
        </div>
      </section>

      <section className="platform-page-cta">
        <div className="platform-page-wrap platform-page-cta-inner">
          <div>
            <p className="platform-page-index">Build what comes next</p>
            <h2>Have a demanding workload in mind?</h2>
            <p>Talk with AtonixCorp about your platform, research, or engineering goals.</p>
          </div>
          <Link href="/contact" className="platform-page-primary">
            Contact AtonixCorp <span aria-hidden="true">→</span>
          </Link>
        </div>
      </section>

      <Footer />
    </main>
  );
}
