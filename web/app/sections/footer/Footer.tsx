"use client";
import "../../styles/Footer.css";
import Image from "next/image";
import Link from "next/link";

export default function Footer() {
  return (
    <footer className="enterprise-footer">
      <div className="wrap footer-mission">
        <div>
          <Link className="brand footer-brand" href="/" aria-label="AtonixCorp home">
            <Image src="/atonixcorp-icon.png" alt="" width={42} height={42} />
            <span>
              ATONIXCORP
            </span>
          </Link>

          <p className="footer-tagline">
            Infrastructure for the systems shaping what comes next.
          </p>

          <nav className="footer-social" aria-label="AtonixCorp social links">
            <a
              href="https://github.com/atonixcorp"
              target="_blank"
              rel="noopener noreferrer"
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
              rel="noopener noreferrer"
              aria-label="AtonixCorp LinkedIn"
            >
              {/* LinkedIn Icon */}
              <svg width="22" height="22" viewBox="0 0 24 24" fill="currentColor">
                <path d="M4.98 3.5A2.5 2.5 0 1 1 5 8.5a2.5 2.5 0 0 1-.02-5zM3 9h4v12H3zm7 0h3.8v1.7h.1c.5-.9 1.7-1.8 3.4-1.8 3.6 0 4.7 2.3 4.7 5.4V21h-4v-5.4c0-1.3 0-3-1.8-3s-2.2 1.4-2.2 2.9V21h-4z" />
              </svg>
            </a>

            <a
              href="https://discord.gg/EGqJ778a55"
              target="_blank"
              rel="noopener noreferrer"
              aria-label="AtonixCorp Discord"
            >
              {/* Discord Icon */}
              <svg width="22" height="22" viewBox="0 0 24 24" fill="currentColor">
                <path d="M20 4.5c-1.5-.7-3.2-1.2-5-1.4-.2.4-.4.9-.6 1.3-1.8-.3-3.6-.3-5.4 0-.2-.4-.4-.9-.6-1.3-1.8.2-3.5.7-5 1.4C1.4 9.1.9 13.5 1 17.9c1.7 1.3 3.6 2.2 5.6 2.7.4-.6.7-1.2.9-1.9-1-.4-2-.9-2.9-1.6.2-.2.4-.4.6-.6 2.8 1.3 6 1.3 8.8 0 .2.2.4.4.6.6-.9.7-1.9 1.2-2.9 1.6.2.7.5 1.3.9 1.9 2-.5 3.9-1.4 5.6-2.7.2-4.4-.4-8.8-2.2-13.4zM8.5 14.8c-1 0-1.8-.9-1.8-2s.8-2 1.8-2 1.8.9 1.8 2-.8 2-1.8 2zm7 0c-1 0-1.8-.9-1.8-2s.8-2 1.8-2 1.8.9 1.8 2-.8 2-1.8 2z" />
              </svg>
            </a>

            <a
              href="https://mastodon.social/@atonixcorp"
              target="_blank"
              rel="noopener noreferrer"
              aria-label="AtonixCorp on Mastodon"
            >
              <svg viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
                <path d="M21.6 7.1c-.3-2.2-2.3-3.9-4.6-4.3a29 29 0 0 0-10 0C4.6 3.2 2.6 4.9 2.3 7.1c-.5 3.1-.5 6.1 0 9.2.4 2.4 2.4 4.1 4.8 4.5 2.3.4 4.7.4 7 0 1.3-.2 2.5-.7 3.4-1.6l-.8-1.4c-.8.5-1.7.8-2.7.9-1.6.2-3.2.2-4.8 0-1.5-.2-2.6-1.1-2.8-2.5v-.7c1.6.4 3.2.6 4.8.6 1.7 0 3.4-.2 5-.6 2.3-.6 3.9-2.3 4.2-4.5.4-2.7.4-5.4.2-7.9ZM17 13.2h-2.4V8.5c0-1-.4-1.5-1.2-1.5-.9 0-1.4.6-1.4 1.8v2.6H9.6V8.8c0-1.2-.5-1.8-1.4-1.8-.8 0-1.2.5-1.2 1.5v4.7H4.6V8.4c0-1 .3-1.8.9-2.4.6-.6 1.3-.9 2.3-.9 1.1 0 2 .4 2.6 1.3l.6 1 .6-1c.6-.9 1.5-1.3 2.6-1.3 1 0 1.7.3 2.3.9.6.6.9 1.4.9 2.4v4.8Z" />
              </svg>
            </a>

            <Link href="/contact" aria-label="AtonixCorp WeChat contact">
              <svg viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
                <path d="M9.5 3C5.4 3 2 5.7 2 9c0 1.8 1 3.4 2.6 4.5L4 16l2.8-1.4c.8.3 1.7.4 2.7.4h.3a5.7 5.7 0 0 1-.3-1.8c0-3.3 3.1-6 7-6h.3C16 4.8 13 3 9.5 3Zm-2.7 7a1 1 0 1 1 0-2 1 1 0 0 1 0 2Zm5.4 0a1 1 0 1 1 0-2 1 1 0 0 1 0 2Zm4.3-1.5c-3 0-5.5 1.9-5.5 4.3s2.5 4.3 5.5 4.3c.7 0 1.4-.1 2-.3l2.3 1.2-.5-2.1c1.1-.8 1.7-1.9 1.7-3.1 0-2.4-2.5-4.3-5.5-4.3Zm-2 5.3a.8.8 0 1 1 0-1.6.8.8 0 0 1 0 1.6Zm4 0a.8.8 0 1 1 0-1.6.8.8 0 0 1 0 1.6Z" />
              </svg>
            </Link>
          </nav>
        </div>

        <div className="footer-engagement">
          <span>Strategic engagement</span>
          <Link href="/contact">
            Start a conversation <b aria-hidden="true">↗</b>
          </Link>
        </div>
      </div>

      <div className="wrap footer-grid">
        <div className="footer-overview">
          <p>
            AtonixCorp is building a connected cloud platform for AI, research, robotics,
            aerospace, energy, enterprise, and the critical systems of tomorrow.
          </p>
          <span className="footer-status">
            <i /> Global systems operational
          </span>
        </div>

        <nav aria-label="Company links">
          <h3>Company</h3>
          <Link href="/platform">About AtonixCorp</Link>
          <Link href="/#mission">Mission & mandate</Link>
          <Link href="/#sovereignty">Corporate governance</Link>
          <Link href="/#roadmap">Strategic roadmap</Link>
        </nav>

        <nav aria-label="Infrastructure links">
          <h3>Infrastructure</h3>
          <Link href="/#capabilities">Cloud & AI systems</Link>
          <Link href="/#capabilities">Compute & energy</Link>
          <Link href="/#sovereignty">Security engineering</Link>
          <Link href="/#capabilities">Developer ecosystem</Link>
        </nav>

        <nav aria-label="Resource links">
          <h3>Resources</h3>
          <Link href="/contact">Technical support</Link>
          <Link href="/contact">Enterprise solutions</Link>
          <Link href="/contact">Research collaboration</Link>
          <Link href="/contact">Media center</Link>
        </nav>
      </div>

      <div className="wrap footer-legal">
        <span>© 2026 AtonixCorp. All rights reserved.</span>

        <div>
          <Link href="/#top">Privacy</Link>
          <Link href="/#top">Terms of use</Link>
          <Link href="/#top">Accessibility</Link>
        </div>

        <Link className="back-top" href="/#top">
          Back to top ↑
        </Link>
      </div>
    </footer>
  );
}
