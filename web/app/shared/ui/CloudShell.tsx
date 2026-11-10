"use client";

import Image from "next/image";
import Link from "next/link";
import { usePathname, useSearchParams } from "next/navigation";
import { Suspense, useRef, useState } from "react";
import type { ReactNode } from "react";
import { CloudThemeToggle } from "./CloudThemeToggle";
import { cloudNavigation, isCloudNavigationItemActive } from "./cloud-navigation";
import { CloudNavIcon } from "./CloudNavIcon";
import { scopedCloudHref } from "../lib/cloud-scope";
import styles from "./CloudShell.module.css";
import "./console-design.css";

export function CloudShell({ children }: { children: ReactNode }) {
  const [menuOpen, setMenuOpen] = useState(false);
  const [loggingOut, setLoggingOut] = useState(false);
  const [logoutError, setLogoutError] = useState("");
  const menuButton = useRef<HTMLButtonElement>(null);
  const pathname = usePathname();

  async function handleLogout() {
    if (loggingOut) return;
    setLoggingOut(true);
    setLogoutError("");
    try {
      const response = await fetch("/api/auth/logout", {
        method: "POST",
        credentials: "same-origin",
        cache: "no-store",
        signal: AbortSignal.timeout(10000),
      });
      if (!response.ok) throw new Error("Could not log out. Please try again.");
      // Discard the authenticated client router cache as well as the session.
      window.location.replace("/login");
    } catch {
      setLogoutError("Could not log out. Please check your connection and try again.");
      setLoggingOut(false);
    }
  }

  return (
    <div className={styles.shell} data-cloud-console>
      <a className="skip-link" href="#cloud-product-content">Skip to main content</a>
      <aside className={styles.sidebar} data-menu-open={menuOpen} aria-label="Cloud workspace navigation" onKeyDown={(event) => {
        if (event.key === "Escape" && menuOpen) {
          setMenuOpen(false);
          menuButton.current?.focus();
        }
      }}>
        <div className={styles.sidebarHeader}>
        <Link className={styles.brand} href="/dashboard" aria-label="AtonixCorp Cloud home">
          <span className={styles.brandMark}><Image src="/atonixcorp-icon.png" alt="" width={32} height={32} priority /></span>
          <span>
            <strong>AtonixCorp<span className={styles.brandPeriod}>.</span></strong>
            <small>CLOUD CONSOLE</small>
          </span>
        </Link>
        <button
          className={styles.menuButton}
          ref={menuButton}
          type="button"
          aria-expanded={menuOpen}
          aria-controls="cloud-navigation-panel"
          onClick={() => setMenuOpen((open) => !open)}
        >
          <svg aria-hidden="true" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.7" strokeLinecap="round">
            <path d={menuOpen ? "M6 6l12 12M6 18 18 6" : "M4 6h16M4 12h16M4 18h16"} />
          </svg>
          <span>{menuOpen ? "Close" : "Menu"}</span>
        </button>
        </div>
        <div id="cloud-navigation-panel" className={styles.navigationContainer}>
          <Suspense fallback={<p className={styles.navigationLoading} role="status">Loading navigation...</p>}>
            <CloudNavigation onNavigate={() => setMenuOpen(false)} />
          </Suspense>
        </div>
        <div className={styles.sidebarActions}>
          <Link className={styles.supportLink} href="/support" aria-current={pathname === "/support" ? "page" : undefined} onClick={() => setMenuOpen(false)}>
            <svg aria-hidden="true" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.6" strokeLinecap="round" strokeLinejoin="round">
              <path d="M4 13v-1a8 8 0 0 1 16 0v5a4 4 0 0 1-4 4h-3" /><rect x="2" y="11" width="5" height="7" rx="2" /><rect x="17" y="11" width="5" height="7" rx="2" />
            </svg>
            <span>Support center</span><span className={styles.supportArrow} aria-hidden="true">&#8599;</span>
          </Link>
          <Link className={styles.supportLink} href="/developers"><CloudNavIcon name="overview" /><span>Developer resources</span><span className={styles.supportArrow} aria-hidden="true">&#8599;</span></Link>
          <p className={styles.footerLabel}>BUILD WITH CONTEXT <span>AtonixCorp Cloud</span></p>
        </div>
      </aside>
      <div className={styles.content}>
        <header className={styles.topbar}>
          <nav aria-label="Console breadcrumb" className={styles.consoleBreadcrumb}>
            <Link href="/dashboard"><CloudNavIcon name="overview" /><span>Workspace</span></Link>
            <span aria-hidden="true">/</span><span>Cloud console</span>
          </nav>
          <div className={styles.topbarActions}>
            <Link href="/" className={styles.websiteLink}>Public website <span aria-hidden="true">&#8599;</span></Link>
            <CloudThemeToggle />
            <button type="button" className={styles.logoutButton} onClick={handleLogout} disabled={loggingOut} aria-busy={loggingOut}>
              {loggingOut ? "Logging out..." : "Log out"}
            </button>
          </div>
        </header>
        {logoutError && <p className={styles.logoutError} role="alert">{logoutError}</p>}
        <div id="cloud-product-content" tabIndex={-1} className={styles.pageContent}>{children}</div>
      </div>
    </div>
  );
}

function CloudNavigation({ onNavigate }: { onNavigate: () => void }) {
  const pathname = usePathname();
  const query = useSearchParams();
  const [search, setSearch] = useState("");
  const [expanded, setExpanded] = useState<Record<string, boolean>>({});
  const filter = search.trim().toLowerCase();
  const tenantID = Number(query.get("tenant_id"));
  const projectID = Number(query.get("project_id"));
  const href = (path: string) => scopedCloudHref(path,
    Number.isSafeInteger(tenantID) && tenantID > 0 ? tenantID : 0,
    Number.isSafeInteger(projectID) && projectID > 0 ? projectID : 0);
  const groups = cloudNavigation.map((group) => ({
    ...group,
    expandable: group.items.length > 1,
    items: group.items.filter((item) => `${group.label} ${item.label}`.toLowerCase().includes(filter)),
  })).filter((group) => group.items.length > 0);
  const sections = [...new Set(groups.map((group) => group.section))];

  return <>
    <div className={styles.search}>
      <svg aria-hidden="true" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.7" strokeLinecap="round"><circle cx="10.5" cy="10.5" r="6.5" /><path d="m16 16 4 4" /></svg>
      <input type="search" aria-label="Find a cloud service" placeholder="Find a service..." value={search} onChange={(event) => setSearch(event.target.value)} onKeyDown={(event) => {
        if (event.key === "Escape" && search) { setSearch(""); event.stopPropagation(); }
      }} />
      {search && <button type="button" onClick={() => setSearch("")} aria-label="Clear service search">&times;</button>}
    </div>
    <nav id="cloud-product-navigation" className={styles.navigation} aria-label="Cloud infrastructure">
      {sections.map((section) => <section className={styles.navigationSection} key={section} aria-label={section}>
        <h2>{section}</h2>
        {groups.filter((group) => group.section === section).map((group) => {
          const active = group.items.some((item) => isCloudNavigationItemActive(item.href, pathname, query));
          if (!group.expandable) {
            const item = group.items[0];
            return <Link className={styles.topLink} href={href(item.href)} key={group.label} aria-current={active ? "page" : undefined} onClick={onNavigate}>
              <CloudNavIcon name={group.icon} /><span>{item.label}</span>{active && <span className={styles.activeMarker} aria-hidden="true" />}
            </Link>;
          }
          const open = Boolean(filter) || (expanded[group.label] ?? active);
          const groupID = `cloud-group-${group.icon}`;
          return <div className={styles.navigationGroup} key={group.label}>
            <button type="button" className={styles.groupButton} data-active={active} aria-expanded={open} aria-controls={groupID}
              onClick={() => setExpanded((current) => ({ ...current, [group.label]: !open }))} disabled={Boolean(filter)}>
              <CloudNavIcon name={group.icon} /><span>{group.label}</span>
              <svg className={styles.chevron} aria-hidden="true" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.5"><path d="m6 3 5 5-5 5" /></svg>
            </button>
            <div className={styles.groupItems} id={groupID} hidden={!open}>
              {group.items.map((item) => <Link href={href(item.href)} key={item.href}
                aria-current={isCloudNavigationItemActive(item.href, pathname, query) ? "page" : undefined} onClick={onNavigate}>
                <span className={styles.itemMarker} aria-hidden="true" /><span>{item.label}</span>
                {item.unavailable && <span className={styles.unavailable} title="No console management API is connected">Not connected</span>}
              </Link>)}
            </div>
          </div>;
        })}
      </section>)}
      {!groups.length && <div className={styles.noResults} role="status"><strong>No matching services</strong><p>Try compute, storage or metrics.</p><button type="button" onClick={() => setSearch("")}>Clear search</button></div>}
    </nav>
  </>;
}
