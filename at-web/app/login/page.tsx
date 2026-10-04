"use client";

import "../styles/LoginSignup.css";
import TopHeader from "../sections/header/TopHeader";
import GlobalHeader from "../sections/header/GlobalHeader";
import Footer from "../sections/footer/Footer";

export default function LoginSignup() {
  return (
    <main className="login-page">

      <TopHeader />
      <GlobalHeader />

      <section className="login-hero">
        <div className="login-wrap">

          <h1 className="login-title">
            Access Your AtonixCorp Portal
          </h1>

          <p className="login-subtitle">
            Secure authentication for cloud, compute, network, data, and operational services.
          </p>

          {/* Login Card */}
          <div className="login-card">

            <h2 className="login-card-title">Sign in to your account</h2>

            {/* Email Login */}
            <form className="login-form">
              <input type="email" placeholder="Email address" className="login-input" />
              <input type="password" placeholder="Password" className="login-input" />

              <button className="login-btn primary">Login</button>
            </form>

            <div className="login-divider">
              <span>or continue with</span>
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

            <p className="signup-text">
              Don’t have an account? <a href="/onboard">Create one →</a>
            </p>

          </div>

        </div>
      </section>

      <Footer />
    </main>
  );
}
