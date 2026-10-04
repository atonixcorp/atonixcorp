"use client";

import "../styles/Onboard.css";
import Image from "next/image";
import TopHeader from "../sections/header/TopHeader";
import GlobalHeader from "../sections/header/GlobalHeader";
import Footer from "../sections/footer/Footer";

export default function OnboardPage() {
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

            <form className="onboard-form">
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
              <button className="onboard-btn primary" type="submit">
                Create account <span aria-hidden="true">→</span>
              </button>
            </form>

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
              Already have an account? <a href="/login">Sign in <span aria-hidden="true">→</span></a>
            </p>
          </div>
        </div>
      </section>

      <Footer />
    </main>
  );
}
