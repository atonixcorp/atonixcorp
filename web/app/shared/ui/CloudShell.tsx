"use client";

import Image from "next/image";
import Link from "next/link";
import { usePathname } from "next/navigation";
import { useState } from "react";
import type { ReactNode } from "react";
import { CloudThemeToggle } from "./CloudThemeToggle";
import styles from "./CloudShell.module.css";

const products = [
  { label: "Overview", href: "/dashboard" },
  { label: "Compute", href: "/compute" },
  { label: "Containers", href: "/containers" },
  { label: "Storage", href: "/storage" },
  { label: "Databases", href: "/cloud/databases" },
  { label: "Networking", href: "/networking" },
  { label: "Observability", href: "/observability" },
  { label: "Billing", href: "/billing" },
];

export function CloudShell({ children }: { children: ReactNode }) {
  const pathname = usePathname();
  const [menuOpen, setMenuOpen] = useState(false);

  return (
    <div className={styles.shell}>
      <a className="skip-link" href="#cloud-product-content">Skip to main content</a>
      <header className={styles.header}>
        <Link className={styles.brand} href="/dashboard" aria-label="AtonixCorp Cloud home">
          <Image src="/atonixcorp-icon.png" alt="" width={36} height={36} priority />
          <span>
            <strong>AtonixCorp Cloud</strong>
            <small>Cloud platform</small>
          </span>
        </Link>
        <button
          className={styles.menuButton}
          type="button"
          aria-expanded={menuOpen}
          aria-controls="cloud-product-navigation"
          onClick={() => setMenuOpen((open) => !open)}
        >
          <span>{menuOpen ? "Close" : "Menu"}</span>
        </button>
        <nav
          id="cloud-product-navigation"
          className={`${styles.navigation}${menuOpen ? ` ${styles.navigationOpen}` : ""}`}
          aria-label="Cloud products"
        >
          {products.map((product) => {
            const active = product.href === "/dashboard"
              ? pathname === product.href
              : pathname === product.href || pathname.startsWith(`${product.href}/`) ||
                (product.href === "/cloud/databases" && pathname === "/databases");
            return (
              <Link
                href={product.href}
                key={product.href}
                aria-current={active ? "page" : undefined}
                onClick={() => setMenuOpen(false)}
              >
                {product.label}
              </Link>
            );
          })}
        </nav>
        <CloudThemeToggle />
        <Link className={styles.supportLink} href="/support">Support</Link>
      </header>
      <div id="cloud-product-content" className={styles.content} tabIndex={-1}>
        {children}
      </div>
    </div>
  );
}
