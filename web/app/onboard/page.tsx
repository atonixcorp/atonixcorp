"use client";

import "../styles/Onboard.css";
import Image from "next/image";
import Link from "next/link";
import { useState } from "react";
import TopHeader from "../sections/header/TopHeader";
import GlobalHeader from "../sections/header/GlobalHeader";
import Footer from "../sections/footer/Footer";

export default function OnboardPage() {
  const [error, setError] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [created, setCreated] = useState(false);
  return (
    <main className="onboard-page">

      <TopHeader />
      <GlobalHeader />

      <section className="onboard-hero">
        <div className="onboard-wrap">
          <div className="onboard-intro">
            <p className="onboard-eyebrow"><span /> ATONIXCORP / ACCESS</p>
            <h1 className="onboard-title">
              Create your account.
              <span>Start building what comes next.</span>
            </h1>
            <p className="onboard-subtitle">
              Request your place in a connected platform for AI, research, advanced
              engineering, and the systems shaping tomorrow.
            </p>
            <div className="onboard-benefits" aria-label="Platform focus areas">
              <p><span>01</span> Cloud and compute foundations</p>
              <p><span>02</span> AI, research, and engineering workloads</p>
              <p><span>03</span> Secure operations across the lifecycle</p>
            </div>
          </div>

          <div className="onboard-card">
            <div className="onboard-card-heading">
              <div>
                <p className="onboard-card-kicker">GET STARTED</p>
                <h2 className="onboard-card-title">Create an account</h2>
              </div>
              <span className="onboard-card-step">01 <i /> 02</span>
            </div>

            <form className="onboard-form" onSubmit={(event) => {
              event.preventDefault();
              const form = event.currentTarget;
              const data = new FormData(form);
              const password = String(data.get("password") ?? "");
              if (password !== String(data.get("confirmPassword") ?? "")) {
                setError("Passwords do not match.");
                return;
              }
              setSubmitting(true);
              setError("");
              void fetch("/api/auth/signup", {
                method: "POST",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify({
                  fullName: String(data.get("name") ?? ""),
                  email: String(data.get("email") ?? ""),
                  password,
                }),
              }).then(async (response) => {
                const result = await response.json().catch(() => null) as { error?: string } | null;
                if (!response.ok) throw new Error(result?.error || "Could not create account.");
                setCreated(true);
                form.reset();
              }).catch((submitError: unknown) => {
                setError(submitError instanceof Error ? submitError.message : "Could not create account.");
              }).finally(() => setSubmitting(false));
            }}>
              <label className="onboard-field">
                <span>Full name</span>
                <input
                  type="text"
                  name="name"
                  placeholder="Your full name"
                  autoComplete="name"
                  className="onboard-input"
                  required
                />
              </label>
              <label className="onboard-field">
                <span>Work email</span>
                <input
                  type="email"
                  name="email"
                  placeholder="you@company.com"
                  autoComplete="email"
                  className="onboard-input"
                  required
                />
              </label>
              <div className="onboard-password-row">
                <label className="onboard-field">
                  <span>Create password</span>
                  <input
                    type="password"
                    name="password"
                    placeholder="Create a password"
                    autoComplete="new-password"
                    className="onboard-input"
                    minLength={12}
                    maxLength={72}
                    required
                  />
                </label>
                <label className="onboard-field">
                  <span>Confirm password</span>
                  <input
                    type="password"
                    name="confirmPassword"
                    placeholder="Repeat password"
                    autoComplete="new-password"
                    className="onboard-input"
                    required
                  />
                </label>
              </div>
              <button className="onboard-btn primary" type="submit" disabled={submitting}>
                {submitting ? "Creating account…" : "Create account"} <span aria-hidden="true">→</span>
              </button>
            </form>
            {error && <p className="login-notice" role="alert">{error}</p>}
            {created && <p className="login-notice" role="status">Account created. <Link href="/login">Sign in to open your workspace.</Link></p>}

            <div className="onboard-divider"><span>or continue with</span></div>

            <div className="oauth-row">
              <button className="oauth-btn github" type="button">
                <Image src="/icons/github.svg" alt="" aria-hidden="true" width={16} height={16} />
                GitHub
              </button>
              <button className="oauth-btn linkedin" type="button">
                <Image src="/icons/linkedin.svg" alt="" aria-hidden="true" width={16} height={16} />
                LinkedIn
              </button>
              <button className="oauth-btn google" type="button">
                <Image src="/icons/google.svg" alt="" aria-hidden="true" width={16} height={16} />
                Google
              </button>
              <button className="oauth-btn gmail" type="button">
                <Image src="/icons/gmail.svg" alt="" aria-hidden="true" width={16} height={16} />
                Gmail
              </button>
            </div>

            <p className="login-text">
              Already have an account? <Link href="/login">Sign in <span aria-hidden="true">→</span></Link>
            </p>
          </div>
        </div>
      </section>

      <Footer />
    </main>
  );
}
