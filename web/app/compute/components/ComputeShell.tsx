"use client";

import { usePathname } from "next/navigation";
import type { ReactNode } from "react";
import { CloudShell } from "@/app/shared/ui/CloudShell";
import Link from "next/link";

const items = [
  { label: "Overview", href: "/compute", icon: "◫" },
  { label: "Instances", href: "/compute/instances", icon: "▣" },
  { label: "Dedicated Compute", href: "/compute/dedicated", icon: "◈" },
  { label: "Images", href: "/compute/images", icon: "▤" },
  { label: "Snapshots", href: "/compute/snapshots", icon: "◷" },
  { label: "SSH Keys", href: "/compute/ssh-keys", icon: "⌘" },
  { label: "Instance Groups", href: "/compute/groups", icon: "▦" },
  { label: "Settings", href: "/compute/settings", icon: "⚙" },
];

export function ComputeShell({ children }: { children: ReactNode }) {
  const pathname = usePathname();

  return (
    <CloudShell>
      <main className="compute-console">
        <div className="compute-layout">
          <aside className="compute-sidebar" aria-label="Compute navigation">
            <div className="compute-sidebar-heading">COMPUTE</div>
            <nav>
              {items.map((item) => {
                const active = item.href === "/compute"
                  ? pathname === "/compute"
                  : pathname === item.href || pathname.startsWith(`${item.href}/`);
                return (
                  <Link
                    className={`compute-nav-link${active ? " active" : ""}`}
                    href={item.href}
                    key={item.href}
                    aria-current={active ? "page" : undefined}
                  >
                    <span aria-hidden="true">{item.icon}</span>
                    {item.label}
                  </Link>
                );
              })}
            </nav>
            <div className="compute-sidebar-note">
              <span className="compute-status-indicator" />
              <div><strong>Compute API</strong><small>Catalog connection enabled</small></div>
            </div>
          </aside>
          <section className="compute-main">{children}</section>
        </div>
      </main>
    </CloudShell>
  );
}
