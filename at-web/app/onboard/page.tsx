"use client";

import "../styles/Onboard.css";
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

          <h1 className="onboard-title">
            Create Your AtonixCorp Account
          </h1>

          <p className="onboard-subtitle">
            Begin your journey into autonomous cloud, compute, network, data, and secure‑operations services.
          </p>

          {/* Onboard Card */}
          <div className="onboard-card">

            <h2 className="onboard-card-title">Create an account</h2>

            {/* Signup Form */}
            <form className="onboard-form">
              <input type="text" placeholder="Full name" className="onboard-input" />
              <input type="email" placeholder="Email address" className="onboard-input" />
              <input type="password" placeholder="Create password" className="onboard-input" />
              <input type="password" placeholder="Confirm password" className="onboard-input" />

              <button className="onboard-btn primary">Create Account</button>
            </form>

            <div className="onboard-divider">
              <span>or sign up with</span>
            </div>

            {/* Horizontal OAuth */}
            <div className="oauth-row">

              <button className="oauth-btn github">
                <img src="/icons/github.svg" alt="GitHub" />
                GitHub
              </button>

              <button className="oauth-btn linkedin">
                <img src="/icons/linkedin.svg" alt="LinkedIn" />
                LinkedIn
              </button>

              <button className="oauth-btn google">
                <img src="/icons/google.svg" alt="Google" />
                Google
              </button>

              <button className="oauth-btn gmail">
                <img src="/icons/gmail.svg" alt="Gmail" />
                Gmail
              </button>

            </div>

            <p className="login-text">
              Already have an account? <a href="/login">Login →</a>
            </p>

          </div>

        </div>
      </section>

      <Footer />
    </main>
  );
}
