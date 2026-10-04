"use client";
import "../../styles/Footer.css";
import Image from "next/image";

export default function Footer() {
  return (
    <footer className="enterprise-footer">
      <div className="wrap footer-mission">
        <div>
          <a className="brand footer-brand" href="#top">
            <Image src="/atonixcorp-icon.png" alt="" width={42} height={42} />
            <span>
              ATONIXCORP
            </span>
          </a>

          <p>Engineering for the infrastructure that powers global digital operations.</p>

          <div className="footer-social">
            <a
              href="https://github.com/Orcastack/atonixcorp"
              target="_blank"
              rel="noreferrer"
              aria-label="AtonixCorp GitHub"
            >
              {/* GitHub Icon */}
              <svg width="22" height="22" viewBox="0 0 24 24" fill="currentColor">
                <path d="M12 .5A12 12 0 0 0 0 12.6c0 5.3 3.4 9.8 8.2 11.4.6.1.8-.3.8-.6v-2c-3.3.7-4-1.6-4-1.6-.5-1.3-1.2-1.6-1.2-1.6-1-.7.1-.7.1-.7 1.2.1 1.8 1.3 1.8 1.3 1 1.8 2.7 1.3 3.4 1 .1-.8.4-1.3.7-1.6-2.7-.3-5.6-1.4-5.6-6.1 0-1.3.5-2.4 1.2-3.3-.1-.3-.5-1.5.1-3.1 0 0 1-.3 3.3 1.3a11 11 0 0 1 6 0c2.3-1.6 3.3-1.3 3.3-1.3.6 1.6.2 2.8.1 3.1.8.9 1.2 2 1.2 3.3 0 4.7-2.9 5.8-5.6 6.1.4.4.8 1.1.8 2.2v3.2c0 .3.2.7.8.6A12 12 0 0 0 24 12.6 12 12 0 0 0 12 .5z" />
              </svg>
            </a>

            <a
              href="https://linkedin.com/company/atonixcorp"
              target="_blank"
              rel="noreferrer"
              aria-label="AtonixCorp LinkedIn"
            >
              {/* LinkedIn Icon */}
              <svg width="22" height="22" viewBox="0 0 24 24" fill="currentColor">
                <path d="M4.98 3.5A2.5 2.5 0 1 1 5 8.5a2.5 2.5 0 0 1-.02-5zM3 9h4v12H3zm7 0h3.8v1.7h.1c.5-.9 1.7-1.8 3.4-1.8 3.6 0 4.7 2.3 4.7 5.4V21h-4v-5.4c0-1.3 0-3-1.8-3s-2.2 1.4-2.2 2.9V21h-4z" />
              </svg>
            </a>

            <a
              href="https://discord.gg/orcastack"
              target="_blank"
              rel="noreferrer"
              aria-label="AtonixCorp Discord"
            >
              {/* Discord Icon */}
              <svg width="22" height="22" viewBox="0 0 24 24" fill="currentColor">
                <path d="M20 4.5c-1.5-.7-3.2-1.2-5-1.4-.2.4-.4.9-.6 1.3-1.8-.3-3.6-.3-5.4 0-.2-.4-.4-.9-.6-1.3-1.8.2-3.5.7-5 1.4C1.4 9.1.9 13.5 1 17.9c1.7 1.3 3.6 2.2 5.6 2.7.4-.6.7-1.2.9-1.9-1-.4-2-.9-2.9-1.6.2-.2.4-.4.6-.6 2.8 1.3 6 1.3 8.8 0 .2.2.4.4.6.6-.9.7-1.9 1.2-2.9 1.6.2.7.5 1.3.9 1.9 2-.5 3.9-1.4 5.6-2.7.2-4.4-.4-8.8-2.2-13.4zM8.5 14.8c-1 0-1.8-.9-1.8-2s.8-2 1.8-2 1.8.9 1.8 2-.8 2-1.8 2zm7 0c-1 0-1.8-.9-1.8-2s.8-2 1.8-2 1.8.9 1.8 2-.8 2-1.8 2z" />
              </svg>
            </a>
          </div>
        </div>

        <div>
          <span>Strategic engagement</span>
          <a href="#contact">
            Start a conversation <b aria-hidden="true">→</b>
          </a>
        </div>
      </div>

      <div className="wrap footer-grid">
        <div className="footer-overview">
          <p>
            AtonixCorp develops mission-critical infrastructure across autonomous cloud,
            high-performance compute, energy systems, and research ecosystems.
          </p>
          <span className="footer-status">
            <i /> Global systems operational
          </span>
        </div>

        <div>
          <h3>Company</h3>
          <a href="#about">About AtonixCorp</a>
          <a href="#mission">Mission & mandate</a>
          <a href="#sovereignty">Corporate governance</a>
          <a href="#roadmap">Strategic roadmap</a>
        </div>

        <div>
          <h3>Infrastructure</h3>
          <a href="#divisions">Cloud & AI systems</a>
          <a href="#divisions">Compute & energy</a>
          <a href="#systems">Security engineering</a>
          <a href="#systems">Developer ecosystem</a>
        </div>

        <div>
          <h3>Resources</h3>
          <a href="#contact">Technical support</a>
          <a href="#contact">Enterprise solutions</a>
          <a href="#contact">Research collaboration</a>
          <a href="#contact">Media center</a>
        </div>
      </div>

      <div className="wrap footer-legal">
        <span>© 2026 AtonixCorp. All rights reserved.</span>

        <div>
          <a href="#top">Privacy</a>
          <a href="#top">Terms of use</a>
          <a href="#top">Accessibility</a>
        </div>

        <a className="back-top" href="#top">
          Back to top ↑
        </a>
      </div>
    </footer>
  );
}
