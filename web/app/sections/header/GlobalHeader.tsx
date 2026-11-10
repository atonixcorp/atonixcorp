"use client";

import Image from "next/image";
import Link from "next/link";
import { useEffect, useRef, useState } from "react";
import { usePathname } from "next/navigation";
import { publicNavigation } from "@/app/shared/marketing/content";
import { CloudNavIcon } from "@/app/shared/ui/CloudNavIcon";
import "../../styles/GlobalHeader.css";

export default function GlobalHeader() {
  const [activeMenu, setActiveMenu] = useState<string | null>(null);
  const [mobileOpen, setMobileOpen] = useState(false);
  const header = useRef<HTMLElement>(null);
  const mobileButton = useRef<HTMLButtonElement>(null);
  const hoverOpened = useRef<string | null>(null);
  const pathname = usePathname();
  const closeAll = () => { setActiveMenu(null); setMobileOpen(false); };

  useEffect(() => {
    if (!activeMenu && !mobileOpen) return;
    const closeOutside = (event: PointerEvent) => {
      if (event.target instanceof Node && !header.current?.contains(event.target)) {
        setActiveMenu(null); setMobileOpen(false);
      }
    };
    document.addEventListener("pointerdown", closeOutside);
    return () => document.removeEventListener("pointerdown", closeOutside);
  }, [activeMenu, mobileOpen]);

  return <header className="global-header" ref={header} onKeyDown={(event) => {
    if (event.key !== "Escape") return;
    event.preventDefault();
    if (activeMenu) {
      header.current?.querySelector<HTMLButtonElement>(`[data-nav-trigger="${activeMenu}"]`)?.focus();
      setActiveMenu(null);
    } else {
      setMobileOpen(false); mobileButton.current?.focus();
    }
  }} onBlur={(event) => {
    if (event.relatedTarget instanceof Node && !event.currentTarget.contains(event.relatedTarget)) closeAll();
  }} onPointerLeave={(event) => {
    if (event.pointerType === "mouse" && !event.currentTarget.contains(document.activeElement)) setActiveMenu(null);
  }}>
    <div className="public-header-row">
      <Link className="global-brand" href="/" aria-label="AtonixCorp home" onClick={closeAll}>
        <Image src="/atonixcorp-icon.png" alt="" width={36} height={36} priority /><span>AtonixCorp<span className="brand-period">.</span></span>
      </Link>
      <button className="public-menu-toggle" ref={mobileButton} type="button" aria-expanded={mobileOpen} aria-controls="public-navigation" onClick={() => { setMobileOpen(!mobileOpen); setActiveMenu(null); }}>
        {mobileOpen ? "Close" : "Menu"}<svg aria-hidden="true" viewBox="0 0 24 24"><path d={mobileOpen ? "M6 6l12 12M6 18 18 6" : "M4 7h16M4 12h16M4 17h16"} /></svg>
      </button>
      <nav className="public-navigation" id="public-navigation" data-open={mobileOpen} aria-label="Main navigation">
        {publicNavigation.map((item) => {
          const open = activeMenu === item.id;
          return <div className="public-nav-group" key={item.id}>
            <button type="button" className="public-nav-trigger" data-nav-trigger={item.id} aria-expanded={open} aria-controls={`public-menu-${item.id}`}
              onClick={(event) => {
                setActiveMenu(event.detail > 0 && hoverOpened.current === item.id ? item.id : open ? null : item.id);
                hoverOpened.current = null;
              }}
              onPointerEnter={(event) => {
                if (event.pointerType === "mouse" && window.matchMedia("(min-width: 1100px)").matches && !open) {
                  hoverOpened.current = item.id;
                  setActiveMenu(item.id);
                }
              }}
              onKeyDown={(event) => {
                if (event.key === "ArrowDown") {
                  event.preventDefault();
                  hoverOpened.current = null;
                  setActiveMenu(item.id);
                  requestAnimationFrame(() => header.current?.querySelector<HTMLAnchorElement>(`#public-menu-${item.id} .public-mega-group a`)?.focus());
                }
              }}>
              {item.label}<svg aria-hidden="true" viewBox="0 0 16 16"><path d="m4 6 4 4 4-4" /></svg>
            </button>
            <div className="public-mega-menu" id={`public-menu-${item.id}`} hidden={!open}>
              <div className="public-mega-inner">
                <aside className="public-mega-intro"><span>EXPLORE ATONIXCORP</span><h2>{item.intro}</h2><p>{item.description}</p><Link href={item.href} onClick={closeAll}>{item.cta} <span aria-hidden="true">&rarr;</span></Link></aside>
                {item.groups.map((group) => <section className="public-mega-group" key={group.title} aria-label={group.title}>
                  <h3>{group.title}</h3>{group.links.map((link) => <Link href={link.href} key={link.href} onClick={closeAll} aria-current={pathname === link.href ? "page" : undefined}>
                    <span className="public-menu-icon"><CloudNavIcon name={link.icon} /></span><span><strong>{link.label}</strong><small>{link.description}</small></span><span className="public-menu-arrow" aria-hidden="true">&rarr;</span>
                  </Link>)}
                </section>)}
              </div>
              <div className="public-mega-bottom"><span>A clearer path from infrastructure to application.</span><Link href="/platform" onClick={closeAll}>Meet the platform <span aria-hidden="true">&rarr;</span></Link></div>
            </div>
          </div>;
        })}
        <Link className="public-nav-link" href="/platform" onClick={closeAll} aria-current={pathname === "/platform" ? "page" : undefined}>Platform</Link>
        <div className="public-header-actions"><Link href="/login" onClick={closeAll}>Log in</Link><Link className="public-header-cta" href="/onboard" onClick={closeAll}>Get started <span aria-hidden="true">&rarr;</span></Link></div>
      </nav>
    </div>
  </header>;
}
