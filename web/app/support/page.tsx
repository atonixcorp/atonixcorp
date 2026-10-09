"use client";

import React from "react";

export default function SupportPage() {
  return (
    <main className="page-shell">
      <section className="page-hero">
        <h1>Support Portal</h1>
        <p>Get help with AtonixCorp cloud, engineering workloads, and platform operations.</p>
      </section>

      <section className="page-content">
        <div className="form-card">
          <div className="form-group">
            <label htmlFor="subject">Subject</label>
            <input id="subject" type="text" placeholder="Issue summary" />
          </div>

          <div className="form-group">
            <label htmlFor="details">Details</label>
            <textarea
              id="details"
              rows={4}
              placeholder="Describe the problem, workload, or environment."
            />
          </div>

          <button type="button" className="primary-button">
            Submit ticket
          </button>

          <div className="form-footer">
            <a href="/login">Back to login</a>
            <a href="/onboard">Go to onboarding</a>
          </div>
        </div>
      </section>
    </main>
  );
}
