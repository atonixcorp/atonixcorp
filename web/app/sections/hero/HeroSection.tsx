"use client";
import "../../styles/HeroSection.css";

function PlatformIllustration() {
  return (
    <svg
      className="platform-illustration"
      viewBox="0 0 620 620"
      role="img"
      aria-labelledby="platform-illustration-title platform-illustration-description"
    >
      <title id="platform-illustration-title">AtonixCorp Cloud Fabric</title>
      <desc id="platform-illustration-description">
        A connected cloud platform linking compute, orchestration, database,
        robotics, networking, security, observability, and server storage.
      </desc>
      <defs>
        <linearGradient id="platform-brand-gradient" x1="0" y1="0" x2="1" y2="1">
          <stop offset="0" stopColor="#00b86b" />
          <stop offset="0.55" stopColor="#00a6a7" />
          <stop offset="1" stopColor="#0877c9" />
        </linearGradient>
        <linearGradient id="platform-node-gradient" x1="0" y1="0" x2="1" y2="1">
          <stop offset="0" stopColor="#ffffff" />
          <stop offset="1" stopColor="#f0fbfa" />
        </linearGradient>
        <filter id="platform-node-shadow" x="-30%" y="-30%" width="160%" height="170%">
          <feDropShadow dx="0" dy="7" stdDeviation="8" floodColor="#12354a" floodOpacity="0.12" />
        </filter>
      </defs>

      <circle cx="310" cy="310" r="258" fill="none" stroke="#dff5ef" strokeWidth="1.5" />
      <circle cx="310" cy="310" r="215" fill="none" stroke="#dcefeb" strokeWidth="1.5" />
      <circle cx="310" cy="310" r="178" fill="none" stroke="#e7f1f7" strokeWidth="1" strokeDasharray="3 9" />

      <g fill="none" strokeLinecap="round">
        <path d="M310 152V214 M427 193L383 251 M463 310H406 M427 427L383 369 M310 468V406 M193 427L237 369 M157 310H214 M193 193L237 251" stroke="#00a6a7" strokeOpacity=".53" strokeWidth="2" />
      </g>

      <g fill="#00a6a7">
        <circle cx="310" cy="214" r="5" />
        <circle cx="383" cy="251" r="5" />
        <circle cx="406" cy="310" r="5" />
        <circle cx="383" cy="369" r="5" />
        <circle cx="310" cy="406" r="5" />
        <circle cx="237" cy="369" r="5" />
        <circle cx="214" cy="310" r="5" />
        <circle cx="237" cy="251" r="5" />
      </g>

      <g filter="url(#platform-node-shadow)" textAnchor="middle" fontFamily="Arial, sans-serif">
        <g transform="translate(310 96)">
          <circle r="57" fill="url(#platform-node-gradient)" stroke="#d8e8ee" strokeWidth="2" />
          <path d="m-20-13 20-11 20 11v22L0 20l-20-11z" fill="url(#platform-brand-gradient)" />
          <path d="m0-24 20 11L0-2l-20-11z" fill="#17c99a" />
          <path d="M0-2 20-13v22L0 20z" fill="#0877c9" />
          <path d="M0-2v22l-20-11v-22z" fill="#087a83" />
          <text y="42" fill="#15324a" fontSize="12" fontWeight="700">Compute</text>
        </g>

        <g transform="translate(462 158)">
          <circle r="57" fill="url(#platform-node-gradient)" stroke="#d8e8ee" strokeWidth="2" />
          <rect x="-21" y="-19" width="28" height="28" rx="6" fill="#00b86b" />
          <rect x="-8" y="-11" width="28" height="28" rx="6" fill="#0877c9" />
          <rect x="-14" y="-5" width="18" height="18" rx="4" fill="#f4fffd" />
          <text y="42" fill="#15324a" fontSize="11" fontWeight="700">Orchestration</text>
        </g>

        <g transform="translate(525 310)">
          <circle r="57" fill="url(#platform-node-gradient)" stroke="#d8e8ee" strokeWidth="2" />
          <ellipse cx="0" cy="-15" rx="19" ry="7" fill="#1685c5" />
          <path d="M-19-15v29c0 4 9 8 19 8s19-4 19-8v-29" fill="#1685c5" />
          <path d="M-19-5c0 4 9 8 19 8s19-4 19-8M-19 5c0 4 9 8 19 8s19-4 19-8" fill="none" stroke="#bce9fa" strokeWidth="2" />
          <text y="42" fill="#15324a" fontSize="11" fontWeight="700">Cloud Database</text>
        </g>

        <g transform="translate(462 462)">
          <circle r="57" fill="url(#platform-node-gradient)" stroke="#d8e8ee" strokeWidth="2" />
          <path d="m-22 16 9-15 12 4 7-13 13 7-8 15 10 8h-43z" fill="none" stroke="#0877a9" strokeWidth="5" strokeLinejoin="round" />
          <circle cx="-13" cy="1" r="5" fill="#00b86b" />
          <circle cx="0" cy="5" r="5" fill="#00a6a7" />
          <circle cx="6" cy="-8" r="5" fill="#00b86b" />
          <circle cx="19" cy="-1" r="5" fill="#0877c9" />
          <text y="42" fill="#15324a" fontSize="11" fontWeight="700">Robotics</text>
        </g>

        <g transform="translate(310 525)">
          <circle r="57" fill="url(#platform-node-gradient)" stroke="#d8e8ee" strokeWidth="2" />
          <g stroke="#1685c5" strokeWidth="3">
            <path d="M0-13 0 9M-18 2 0 9 18 2" />
          </g>
          <g fill="#00a6a7" stroke="#fff" strokeWidth="2">
            <circle cy="-19" r="8" />
            <circle cx="-21" cy="0" r="8" />
            <circle cx="21" cy="0" r="8" />
            <circle cy="12" r="8" fill="#0877c9" />
          </g>
          <text y="42" fill="#15324a" fontSize="11" fontWeight="700">Network</text>
        </g>

        <g transform="translate(158 462)">
          <circle r="57" fill="url(#platform-node-gradient)" stroke="#d8e8ee" strokeWidth="2" />
          <path d="m0-24 21 8v15c0 13-9 23-21 29C-12 22-21 12-21-1v-15z" fill="url(#platform-brand-gradient)" />
          <rect x="-8" y="-3" width="16" height="14" rx="3" fill="#fff" />
          <path d="M-5-3v-5a5 5 0 0 1 10 0v5" fill="none" stroke="#fff" strokeWidth="3" />
          <text y="42" fill="#15324a" fontSize="11" fontWeight="700">Security</text>
        </g>

        <g transform="translate(95 310)">
          <circle r="57" fill="url(#platform-node-gradient)" stroke="#d8e8ee" strokeWidth="2" />
          <path d="M-21 16h42M-17 10V-4M-5 10v-9M7 10v-18M19 10V0" fill="none" stroke="#0877c9" strokeWidth="6" strokeLinecap="round" />
          <path d="M-21 18h42" stroke="#00b86b" strokeWidth="4" strokeLinecap="round" />
          <text y="42" fill="#15324a" fontSize="11" fontWeight="700">Observability</text>
        </g>

        <g transform="translate(158 158)">
          <circle r="57" fill="url(#platform-node-gradient)" stroke="#d8e8ee" strokeWidth="2" />
          <rect x="-21" y="-17" width="42" height="11" rx="3" fill="#0877c9" />
          <rect x="-21" y="-2" width="42" height="11" rx="3" fill="#1685c5" />
          <rect x="-21" y="13" width="42" height="11" rx="3" fill="#0877a9" />
          <circle cx="13" cy="-11.5" r="2" fill="#00d69b" />
          <circle cx="13" cy="3.5" r="2" fill="#00d69b" />
          <circle cx="13" cy="18.5" r="2" fill="#00d69b" />
          <text y="42" fill="#15324a" fontSize="11" fontWeight="700">Server Storage</text>
        </g>

        <g transform="translate(310 310)">
          <circle r="87" fill="#ffffff" fillOpacity=".92" stroke="#d9efea" strokeWidth="1.5" />
          <circle r="70" fill="#effaf8" />
          <rect x="-37" y="-37" width="74" height="74" rx="22" fill="url(#platform-brand-gradient)" />
          <image href="/atonixcorp-icon.png" x="-25" y="-25" width="50" height="50" preserveAspectRatio="xMidYMid meet" />
          <text y="62" fill="#15324a" fontSize="12" fontWeight="700">AtonixCorp Cloud Fabric</text>
        </g>
      </g>
    </svg>
  );
}

export default function HeroSection() {
  return (
    <section className="enterprise-hero">
      <div className="enterprise-hero-wrap">

        {/* Left Content */}
        <div className="enterprise-hero-content">
          <p className="enterprise-hero-index">00 / Introduction</p>

          <h1 className="enterprise-hero-title">
            <span>One Cloud Platform</span>
            <span className="enterprise-hero-title-accent">for the Systems That Shape Tomorrow</span>
          </h1>

          <p className="enterprise-hero-subtitle">
            Built for ambitious workloads. Designed for critical systems. Ready to grow across industries.
          </p>

          <p className="enterprise-hero-description">
            AtonixCorp is developing a connected cloud foundation for AI, scientific research,
            robotics, aerospace, flight, energy, and enterprise applications—bringing the
            infrastructure for complex work together in one platform.
          </p>

          <a href="#capabilities" className="enterprise-hero-action">
            Explore AtonixCorp Platform →
          </a>
        </div>

        {/* Right Illustration */}
        <div className="enterprise-hero-graphic">
          <PlatformIllustration />
          <p className="enterprise-hero-graphic-caption">
            <span className="enterprise-hero-graphic-indicator" aria-hidden="true" />
            <span>AtonixCorp Cloud Fabric</span>
            <span className="enterprise-hero-graphic-divider" aria-hidden="true" />
            <span>Eight connected capabilities</span>
          </p>
        </div>

      </div>
    </section>
  );
}
