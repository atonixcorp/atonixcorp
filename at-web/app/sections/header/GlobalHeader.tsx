"use client";

import "../../styles/GlobalHeader.css";
import Image from "next/image";
import { useEffect, useRef, useState } from "react";

type NavItem = {
  label: string;
  href?: string;
  items?: { label: string; href: string }[];
};

const navigation: NavItem[] = [
  {
    label: "Software",
    items: [
      { label: "Overview", href: "#overview" },
      { label: "AtonixCorp Components", href: "#atonixcorp-components" },
      { label: "Development Tools", href: "#development-tools" },
      { label: "AtonixCorp Map", href: "#atonixcorp-map" },
      { label: "Configuration", href: "#configuration" },
      { label: "Security", href: "#security" },
      { label: "Marketplace", href: "#marketplace" },
      { label: "Software Catalog", href: "#software" },
    ],
  },
  {
    label: "Platform",
    items: [
      { label: "Cloud Control", href: "#divisions" },
      { label: "Compute & HPC", href: "#divisions" },
      { label: "Network & Data", href: "#divisions" },
      { label: "Secure Operations", href: "#divisions" },
      { label: "Platform Governance", href: "#sovereignty" },
      { label: "Platform Architecture", href: "#architecture" },
    ],
  },
  {
    label: "Uses",
    items: [
      { label: "Enterprise Workloads", href: "#usecases-enterprise" },
      { label: "Scientific Research & HPC", href: "#usecases-hpc" },
      { label: "Critical Infrastructure", href: "#usecases-infrastructure" },
      { label: "Energy‑Driven Computing", href: "#usecases-energy" },
      { label: "Developer Automation", href: "#usecases-developer" },
      { label: "Security & Identity", href: "#usecases-security" },
    ],
  },
  {
    label: "Solutions",
    items: [
      { label: "Cloud & AI Systems", href: "#solutions-cloud" },
      { label: "Compute & Energy Systems", href: "#solutions-compute" },
      { label: "Security Engineering", href: "#solutions-security" },
      { label: "Developer Ecosystem", href: "#solutions-developer" },
      { label: "Operational Intelligence", href: "#solutions-analytics" },
    ],
  },
  {
    label: "Operations",
    items: [
      { label: "Developer Delivery", href: "#systems" },
      { label: "Analytics & Monitoring", href: "#systems" },
      { label: "Security & Identity", href: "#systems" },
      { label: "Platform Policies", href: "#sovereignty" },
      { label: "Platform Roadmap", href: "#roadmap" },
    ],
  },
  {
    label: "Engineering",
    items: [
      { label: "High‑Performance Workloads", href: "#divisions" },
      { label: "Data Services", href: "#divisions" },
      { label: "Infrastructure Automation", href: "#systems" },
      { label: "Platform Architecture", href: "#sovereignty" },
      { label: "Energy Systems", href: "#energy-systems" },
    ],
  },
  {
    label: "DevHub",
    items: [
      { label: "Documentation", href: "#developer-docs" },
      { label: "API Reference", href: "#developer-api" },
      { label: "CLI Tools", href: "#developer-cli" },
      { label: "SDKs", href: "#developer-sdks" },
      { label: "Tutorials", href: "#developer-tutorials" },
      { label: "Open Innovation", href: "#developer-open" },
    ],
  },
  {
    label: "Community",
    items: [
      { label: "Forums", href: "#community-forums" },
      { label: "Events", href: "#community-events" },
      { label: "Engineering Blog", href: "#community-blog" },
      { label: "Research Network", href: "#community-research" },
      { label: "Ambassador Program", href: "#community-ambassadors" },
    ],
  },
  {
    label: "Contact",
    items: [
      { label: "Contact Us", href: "#contact" },
      { label: "Support Center", href: "mailto:dev@atonixcorp.com" },
      { label: "Enterprise Solutions", href: "mailto:partners@atonixcorp.com" },
      { label: "Partnerships", href: "mailto:partners@atonixcorp.com" },
      { label: "Careers", href: "mailto:info@atonixcorp.com" },
    ],
  },
];

export default function GlobalHeader() {
  const [activeMenu, setActiveMenu] = useState<string | null>(null);
  const [mobileOpen, setMobileOpen] = useState(false);
  const headerRef = useRef<HTMLElement>(null);

  useEffect(() => {
    const closeOnOutsideClick = (event: MouseEvent) => {
      if (!headerRef.current?.contains(event.target as Node)) {
        setActiveMenu(null);
      }
    };
    document.addEventListener("mousedown", closeOnOutsideClick);
    return () => document.removeEventListener("mousedown", closeOnOutsideClick);
  }, []);

  const closeAll = () => {
    setActiveMenu(null);
    setMobileOpen(false);
  };

  return (
    <header className="global-header" ref={headerRef}>
      <div className="header-row wrap">

        {/* Brand */}
        <a className="global-brand" href="#top" aria-label="AtonixCorp home" onClick={closeAll}>
          <Image
            src="/atonixcorp-icon.png"
            alt="AtonixCorp logo"
            width={34}
            height={34}
            priority
          />
          <span>
            ATONIXCORP
          </span>
        </a>

        {/* Navigation */}
        <nav className="global-nav" aria-label="Global navigation">
          {navigation.map((item) =>
            item.items ? (
              <div className="nav-menu" key={item.label}>
                <button
                  className={activeMenu === item.label ? "nav-trigger open" : "nav-trigger"}
                  type="button"
                  aria-expanded={activeMenu === item.label}
                  onClick={() =>
                    setActiveMenu(activeMenu === item.label ? null : item.label)
                  }
                >
                  {item.label}

                  {/* ENTERPRISE CHEVRON */}
                  <span className="nav-chevron" aria-hidden="true">
                    <svg width="12" height="12" viewBox="0 0 24 24">
                      <path
                        d="M6 9l6 6 6-6"
                        fill="none"
                        stroke="currentColor"
                        strokeWidth="2"
                        strokeLinecap="round"
                        strokeLinejoin="round"
                      />
                    </svg>
                  </span>
                </button>

                <div className={activeMenu === item.label ? "dropdown visible" : "dropdown"}>
                  {item.items.map((subItem) => (
                    <a href={subItem.href} key={subItem.label} onClick={closeAll}>
                      {subItem.label}
                      <span aria-hidden="true">↗</span>
                    </a>
                  ))}
                </div>
              </div>
            ) : (
              <a className="nav-trigger home-link" href={item.href} key={item.label}>
                {item.label}
              </a>
            )
          )}
        </nav>

        {/* Mobile toggle */}
        <button
          className={mobileOpen ? "menu-toggle open" : "menu-toggle"}
          type="button"
          aria-expanded={mobileOpen}
          aria-label="Toggle navigation menu"
          onClick={() => setMobileOpen(!mobileOpen)}
        >
          <span />
          <span />
        </button>
      </div>

      {/* Mobile nav */}
      <nav className={mobileOpen ? "mobile-nav visible" : "mobile-nav"} aria-label="Mobile global navigation">
        {navigation.map((item) => (
          <div key={item.label} className="mobile-nav-group">
            {item.href ? (
              <a href={item.href} onClick={closeAll}>
                {item.label}
              </a>
            ) : (
              <>
                <span>{item.label}</span>
                {item.items?.map((subItem) => (
                  <a href={subItem.href} key={subItem.label} onClick={closeAll}>
                    {subItem.label}
                  </a>
                ))}
              </>
            )}
          </div>
        ))}
      </nav>
    </header>
  );
}
