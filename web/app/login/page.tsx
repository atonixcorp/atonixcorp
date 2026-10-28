"use client";

import "../styles/LoginSignup.css";
import Image from "next/image";
import Link from "next/link";
import { useState } from "react";
import { useRouter } from "next/navigation";
import TopHeader from "../sections/header/TopHeader";
import GlobalHeader from "../sections/header/GlobalHeader";
import Footer from "../sections/footer/Footer";

export default function LoginSignup() {
  const [loginError, setLoginError] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const router = useRouter();

  return (
    <main className="login-page">

      <TopHeader />
      <GlobalHeader />

      <section className="login-hero">
        <div className="login-wrap">
          <div className="login-intro">
            <p className="login-eyebrow"><span /> ATONIXCORP / SECURE ACCESS</p>
            <h1 className="login-title">
              Welcome back.
              <span>Your work starts here.</span>
            </h1>
            <p className="login-subtitle">
              Sign in to continue to your AtonixCorp workspace and platform services.
            </p>
            <div className="login-assurance">
              <span className="login-assurance-icon" aria-hidden="true">✓</span>
              <p><strong>One account, connected services</strong><span>Access your platform experience in one place.</span></p>
            </div>
            <div className="login-intro-note">
              <span>ATONIXCORP</span>
              <p>Infrastructure for the systems shaping what comes next.</p>
            </div>
          </div>

          <div className="login-card">
            <div className="login-card-heading">
              <div>
                <p className="login-card-kicker">ACCOUNT ACCESS</p>
                <h2 className="login-card-title">Sign in</h2>
              </div>
              <span className="login-lock" aria-hidden="true">⌑</span>
            </div>
            <form
              className="login-form"
              onSubmit={(event) => {
                event.preventDefault();
                const formData = new FormData(event.currentTarget);
                const email = String(formData.get("email") ?? "").trim().toLowerCase();
                const password = String(formData.get("password") ?? "");
                setSubmitting(true);
                setLoginError("");
                void fetch("/api/auth/login", {
                  method: "POST",
                  headers: { "Content-Type": "application/json" },
                  body: JSON.stringify({ email, password }),
                }).then(async (response) => {
                  const payload = await response.json().catch(() => null) as { error?: string } | null;
                  if (!response.ok) throw new Error(payload?.error || "Could not sign in.");
                  router.push("/dashboard");
                }).catch((error: unknown) => {
                  setLoginError(error instanceof Error ? error.message : "Could not sign in.");
                }).finally(() => setSubmitting(false));
              }}
            >
              <label className="login-field">
                <span>Email address</span>
                <input
                  type="email"
                  name="email"
                  placeholder="you@company.com"
                  autoComplete="email"
                  className="login-input"
                  required
                  onChange={() => setLoginError("")}
                />
              </label>
              <label className="login-field">
                <span>Password</span>
                <input
                  type="password"
                  name="password"
                  placeholder="Enter your password"
                  autoComplete="current-password"
                  className="login-input"
                  required
                  onChange={() => setLoginError("")}
                />
              </label>
              <button className="login-btn primary" type="submit" disabled={submitting}>
                {submitting ? "Signing in…" : "Sign in"} <span aria-hidden="true">→</span>
              </button>
            </form>
            {loginError && <p className="login-notice" role="alert">{loginError}</p>}
            <p className="login-dashboard-preview">
              Need an account? <Link href="/onboard">Request access <span aria-hidden="true">→</span></Link>
            </p>

            <div className="login-divider"><span>or continue with</span></div>
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
            <p className="signup-text">
              New to AtonixCorp? <a href="/onboard">Request access <span aria-hidden="true">→</span></a>
            </p>
          </div>
        </div>
      </section>

      <Footer />
    </main>
  );
}
