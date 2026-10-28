"use client";

import { useState } from "react";
import { CloudShell } from "@/app/shared/ui/CloudShell";
import "./billing.css";

const sections = [
  "Overview",
  "Usage",
  "Cost Explorer",
  "Invoices",
  "Payments",
  "Credits",
  "Budgets",
  "Pricing",
  "Billing Settings",
] as const;
type Section = (typeof sections)[number];

const monthlySummary = [
  { label: "Estimated cost", value: "—" },
  { label: "Current usage", value: "—" },
  { label: "Credits", value: "—" },
  { label: "Outstanding balance", value: "—" },
];

const serviceCosts = [
  "Compute",
  "Containers",
  "Storage",
  "Databases",
  "Networking",
  "GPU / HPC",
];

const sectionDescriptions: Record<Exclude<Section, "Overview">, string> = {
  Usage: "Metered usage from Compute, Containers, Storage, Databases, Networking, GPU / HPC, and other connected services.",
  "Cost Explorer": "Explore actual service costs over time after usage metering and pricing are connected.",
  Invoices: "Issued invoices and their payment status will appear here.",
  Payments: "Payment methods and transaction history will appear here.",
  Credits: "Available account credits and credit activity will appear here.",
  Budgets: "Budget limits and spend notifications will be available when billing data is connected.",
  Pricing: "AtonixCorp pricing catalog and rate configuration are not connected yet.",
  "Billing Settings": "Account billing contacts, currency, and payment preferences will be available here.",
};

function UnavailableNotice({ title }: { title: string }) {
  return (
    <div className="billing-unavailable" role="status">
      <span aria-hidden="true">i</span>
      <div>
        <strong>{title}</strong>
        <p>Billing integration is not connected yet. No live usage, costs, invoices, or account balances are available.</p>
      </div>
    </div>
  );
}

export default function BillingPage() {
  const [section, setSection] = useState<Section>("Overview");

  return (
    <CloudShell>
      <main className="billing-page">
        <header className="billing-heading">
          <div>
            <p className="billing-eyebrow">ATONIXCORP CLOUD / BILLING</p>
            <h1>Billing</h1>
            <p>Track usage, costs, invoices, and payments across your AtonixCorp Cloud account.</p>
          </div>
          <span className="billing-status"><i /> Integration not connected</span>
        </header>

        <div className="billing-layout">
          <nav className="billing-nav" aria-label="Billing pages">
            <span>Billing</span>
            {sections.map((item) => (
              <button
                key={item}
                type="button"
                aria-current={section === item ? "page" : undefined}
                onClick={() => setSection(item)}
              >
                {item}
              </button>
            ))}
          </nav>

          <div className="billing-content">
            {section === "Overview" ? (
              <>
                <UnavailableNotice title="Billing integration is not connected yet" />
                <section className="billing-section">
                  <header><div><h2>Current month</h2><p>Amounts will be calculated from verified usage and configured AtonixCorp rates.</p></div></header>
                  <div className="billing-summary-grid">
                    {monthlySummary.map((item) => (
                      <article key={item.label}>
                        <span>{item.label}</span>
                        <strong>{item.value}</strong>
                        <small>Unavailable until billing is connected</small>
                      </article>
                    ))}
                  </div>
                </section>

                <section className="billing-section">
                  <header><div><h2>Cost by service</h2><p>Service totals appear when each resource service reports billable usage.</p></div></header>
                  <div className="billing-service-list">
                    {serviceCosts.map((service) => (
                      <div key={service}><span>{service}</span><strong>—</strong></div>
                    ))}
                  </div>
                </section>

                <section className="billing-section">
                  <header><div><h2>Recent usage</h2><p>Usage records are supplied automatically by connected cloud services.</p></div></header>
                  <div className="billing-empty">
                    <strong>No billing data available</strong>
                    <p>Once resource services report usage and billing is connected, usage and estimated costs will appear here. No billing resource needs to be created.</p>
                  </div>
                </section>
              </>
            ) : (
              <section className="billing-section billing-detail">
                <p className="billing-eyebrow">BILLING / {section.toUpperCase()}</p>
                <h2>{section}</h2>
                <p>{sectionDescriptions[section]}</p>
                <UnavailableNotice title={`${section} data is not available`} />
              </section>
            )}
          </div>
        </div>
      </main>
    </CloudShell>
  );
}
