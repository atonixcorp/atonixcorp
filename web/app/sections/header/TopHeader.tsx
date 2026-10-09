"use client";

import "../../styles/TopHeader.css";
import Link from "next/link";

export default function TopHeader() {
  return (
    <div className="top-header">
      <div className="top-header-wrap">

        {/* LEFT — ORCASTACK PROJECT (with dropdown) */}
        <div className="dropdown-parent top-left">
          Developer Community

          <div className="top-dropdown">
            <a href="https://github.com/atonixcorp" target="_blank" rel="noopener noreferrer">GitHub ↗</a>
            <a href="https://discord.gg/EGqJ778a55" target="_blank" rel="noopener noreferrer">Discord ↗</a>
            <a href="https://atonixcorp.slack.com/archives/C0C6G060JL" target="_blank" rel="noopener noreferrer">Slack ↗</a>
            <a href="https://linkedin.com/company/atonixcorp" target="_blank" rel="noopener noreferrer">LinkedIn ↗</a>
          </div>
        </div>

        {/* CENTER — SOCIAL ICONS */}
        <div className="header-social">

          {/* LinkedIn */}
          <a className="social-link linkedin" href="https://linkedin.com/company/atonixcorp" target="_blank" rel="noopener noreferrer" aria-label="AtonixCorp on LinkedIn">
            <svg width="16" height="16" fill="currentColor" viewBox="0 0 24 24">
              <path d="M4.98 3.5C4.98 4.88 3.87 6 2.5 6S0 4.88 0 3.5 1.12 1 2.5 1s2.48 1.12 2.48 2.5zM.5 8h4V24h-4V8zm7.5 0h3.8v2.2h.1c.5-.9 1.8-2.2 4-2.2 4.3 0 5.1 2.8 5.1 6.5V24h-4v-8.2c0-2-.1-4.5-2.7-4.5-2.7 0-3.1 2.1-3.1 4.3V24h-4V8z"/>
            </svg>
          </a>

          {/* X */}
          <a className="social-link x" href="https://x.com/atonixcorp" target="_blank" rel="noopener noreferrer" aria-label="AtonixCorp on X">
            <svg width="16" height="16" fill="currentColor" viewBox="0 0 24 24">
              <path d="M18.244 1.016h3.73l-8.16 9.33 9.57 12.638h-7.47l-5.85-7.65-6.69 7.65H.684l8.73-9.98L.5 1.016h7.64l5.24 6.89z"/>
            </svg>
          </a>

          {/* GitHub */}
          <a className="social-link github" href="https://github.com/atonixcorp" target="_blank" rel="noopener noreferrer" aria-label="AtonixCorp on GitHub">
            <svg width="16" height="16" fill="currentColor" viewBox="0 0 24 24">
              <path d="M12 .5C5.73.5.5 5.74.5 12.02c0 5.1 3.29 9.43 7.86 10.96.58.1.79-.25.79-.56v-2.17c-3.2.7-3.87-1.55-3.87-1.55-.53-1.36-1.3-1.72-1.3-1.72-1.06-.73.08-.72.08-.72 1.17.08 1.78 1.2 1.78 1.2 1.04 1.78 2.73 1.27 3.4.97.1-.76.41-1.27.75-1.56-2.55-.29-5.23-1.28-5.23-5.7 0-1.26.45-2.3 1.2-3.11-.12-.29-.52-1.46.11-3.04 0 0 .97-.31 3.18 1.18a11.1 11.1 0 0 1 5.8 0c2.2-1.49 3.17-1.18 3.17-1.18.63 1.58.23 2.75.11 3.04.75.81 1.2 1.85 1.2 3.11 0 4.43-2.69 5.41-5.25 5.69.42.36.8 1.08.8 2.18v3.23c0 .31.21.67.8.56A10.52 10.52 0 0 0 23.5 12C23.5 5.74 18.27.5 12 .5z"/>
            </svg>
          </a>

          {/* Mastodon */}
          <a
            className="social-link mastodon"
            href="https://mastodon.social/@atonixcorp"
            target="_blank"
            rel="me noopener noreferrer"
            aria-label="AtonixCorp on Mastodon"
          >
            <svg width="16" height="16" fill="currentColor" viewBox="0 0 24 24" aria-hidden="true">
              <path d="M21.6 7.1c-.3-2.2-2.3-3.9-4.6-4.3a29 29 0 0 0-10 0C4.6 3.2 2.6 4.9 2.3 7.1c-.5 3.1-.5 6.1 0 9.2.4 2.4 2.4 4.1 4.8 4.5 2.3.4 4.7.4 7 0 1.3-.2 2.5-.7 3.4-1.6l-.8-1.4c-.8.5-1.7.8-2.7.9-1.6.2-3.2.2-4.8 0-1.5-.2-2.6-1.1-2.8-2.5v-.7c1.6.4 3.2.6 4.8.6 1.7 0 3.4-.2 5-.6 2.3-.6 3.9-2.3 4.2-4.5.4-2.7.4-5.4.2-7.9ZM17 13.2h-2.4V8.5c0-1-.4-1.5-1.2-1.5-.9 0-1.4.6-1.4 1.8v2.6H9.6V8.8c0-1.2-.5-1.8-1.4-1.8-.8 0-1.2.5-1.2 1.5v4.7H4.6V8.4c0-1 .3-1.8.9-2.4.6-.6 1.3-.9 2.3-.9 1.1 0 2 .4 2.6 1.3l.6 1 .6-1c.6-.9 1.5-1.3 2.6-1.3 1 0 1.7.3 2.3.9.6.6.9 1.4.9 2.4v4.8Z" />
            </svg>
          </a>

        </div>

        {/* RIGHT — PORTAL (with dropdown) */}
        <div className="dropdown-parent top-right">
          Portal

          <div className="top-dropdown right">
            <Link href="/portal">Portal Home</Link>
            <Link href="/login">Login</Link>
            <Link href="/onboard">Onboard</Link>
          </div>
        </div>

      </div>
    </div>
  );
}
