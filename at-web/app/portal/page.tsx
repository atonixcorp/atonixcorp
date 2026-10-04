"use client";

import "../styles/Portal.css";

import TopHeader from "../sections/header/TopHeader";
import GlobalHeader from "../sections/header/GlobalHeader";
import Footer from "../sections/footer/Footer";


export default function PortalLanding() {
  return (
    <main className="portal-landing">

      <TopHeader />
      <GlobalHeader />

      {/* HERO */}
      <section className="portal-hero">
        <div className="portal-hero-wrap">

          <p className="portal-index">Portal Access</p>

          <h1 className="portal-title">
            Welcome to the AtonixCorp Portal.<br />
            <span className="portal-accent">
              Your gateway to cloud, compute, network, data, and secure‑operations services.
            </span>
          </h1>

          <p className="portal-description">
            Access platform capabilities, manage workloads, monitor infrastructure, and operate
            securely across distributed environments — all from one unified control plane.
          </p>

          <div className="portal-actions">
            <a href="/login" className="portal-btn primary">Login</a>
            <a href="/onboard" className="portal-btn secondary">Onboard</a>
          </div>

        </div>
      </section>

      {/* MINI CAPABILITY GRID */}
      <section className="portal-capabilities">
        <div className="portal-cap-wrap">

          <div className="portal-cap-card">
            <h3>Cloud Control</h3>
            <p>Identity, orchestration, automation, and platform services.</p>
          </div>

          <div className="portal-cap-card">
            <h3>Compute & HPC</h3>
            <p>High‑performance execution environments for demanding workloads.</p>
          </div>

          <div className="portal-cap-card">
            <h3>Network & Data</h3>
            <p>Secure connectivity, governed storage, and durable data services.</p>
          </div>

          <div className="portal-cap-card">
            <h3>Secure Operations</h3>
            <p>Delivery, protection, analytics, and operational intelligence.</p>
          </div>

        </div>
      </section>

      <Footer />

    </main>
  );
}
