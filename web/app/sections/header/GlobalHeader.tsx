"use client";

import "../../styles/GlobalHeader.css";
import Image from "next/image";
import Link from "next/link";
import { useEffect, useRef, useState } from "react";

type NavItem = {
  label: string;
  href?: string;
  target?: "_blank";
  rel?: string;
  items?: { label: string; href: string }[];
};

const navigation: NavItem[] = [
  {
    label: "Software",
    items: [
      { label: "Overview", href: "/platform" },
      { label: "AtonixCorp Components", href: "#capabilities" },
      { label: "Development Tools", href: "#capabilities" },
      { label: "AtonixCorp Map", href: "/platform" },
      { label: "Configuration", href: "#capabilities" },
      { label: "Security", href: "#sovereignty" },
      { label: "Marketplace", href: "#capabilities" },
      { label: "Software Catalog", href: "#capabilities" },
    ],
  },
  {
    label: "Platform",
    items: [
      { label: "Cloud Control", href: "#capabilities" },
      { label: "Compute & HPC", href: "#capabilities" },
      { label: "Network & Data", href: "#capabilities" },
      { label: "Secure Operations", href: "#capabilities" },
      { label: "Platform Governance", href: "#sovereignty" },
      { label: "Platform Architecture", href: "/platform" },
    ],
  },
  {
    label: "Uses Cases",
    items: [
      { label: "Enterprise Workloads", href: "#capabilities" },
      { label: "Scientific Research & HPC", href: "#capabilities" },
      { label: "Critical Infrastructure", href: "#mission" },
      { label: "Energy‑Driven Computing", href: "#mission" },
      { label: "Developer Automation", href: "#capabilities" },
      { label: "Security & Identity", href: "#sovereignty" },
    ],
  },
  {
    label: "Solutions",
    items: [
      { label: "Cloud & AI Systems", href: "#capabilities" },
      { label: "Compute & Energy Systems", href: "#capabilities" },
      { label: "Security Engineering", href: "#sovereignty" },
      { label: "Developer Ecosystem", href: "#capabilities" },
      { label: "Operational Intelligence", href: "#capabilities" },
    ],
  },
  {
    label: "Operations",
    items: [
      { label: "Developer Delivery", href: "#capabilities" },
      { label: "Analytics & Monitoring", href: "#capabilities" },
      { label: "Security & Identity", href: "#sovereignty" },
      { label: "Platform Policies", href: "#sovereignty" },
      { label: "Platform Roadmap", href: "#roadmap" },
    ],
  },
  {
    label: "Engineering",
    items: [
      { label: "High‑Performance Workloads", href: "#capabilities" },
      { label: "Data Services", href: "#capabilities" },
      { label: "Infrastructure Automation", href: "#capabilities" },
      { label: "Platform Architecture", href: "/platform" },
      { label: "Energy Systems", href: "#mission" },
    ],
  },
  {
    label: "Developers",
    items: [
      { label: "Documentation", href: "#capabilities" },
      { label: "API Reference", href: "#capabilities" },
      { label: "CLI Tools", href: "#capabilities" },
      { label: "SDKs", href: "#capabilities" },
      { label: "Tutorials", href: "#capabilities" },
      { label: "Open Innovation", href: "#mission" },
    ],
  },
  {
    label: "Community",
    items: [
      { label: "Forums", href: "#contact" },
      { label: "Events", href: "#contact" },
      { label: "Engineering Blog", href: "#capabilities" },
      { label: "Research Network", href: "#mission" },
      { label: "Ambassador Program", href: "#contact" },
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
  {
    label: "Sponsor",
    href: "https://www.atonixcorp.com/sponsor",
    target: "_blank",
    rel: "noopener noreferrer",
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

  const resolveHref = (href?: string) =>
    href ? (href.startsWith("#") ? `/${href}` : href) : "/";

  return (
    <header className="global-header" ref={headerRef}>
      <div className="header-row wrap">

        {/* Brand */}
        <Link className="global-brand" href="/" aria-label="AtonixCorp home" onClick={closeAll}>
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
        </Link>

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
                    <a href={resolveHref(subItem.href)} key={subItem.label} onClick={closeAll}>
                      {subItem.label}
                      <span aria-hidden="true">↗</span>
                    </a>
                  ))}
                </div>
              </div>
            ) : (
              <a
                className="nav-trigger home-link"
                href={resolveHref(item.href)}
                target={item.target}
                rel={item.rel}
                key={item.label}
              >
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
              <a href={resolveHref(item.href)} target={item.target} rel={item.rel} onClick={closeAll}>
                {item.label}
              </a>
            ) : (
              <>
                <span>{item.label}</span>
                {item.items?.map((subItem) => (
                  <a href={resolveHref(subItem.href)} key={subItem.label} onClick={closeAll}>
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
