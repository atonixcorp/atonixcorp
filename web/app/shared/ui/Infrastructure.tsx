import Link from "next/link";
import type { ReactNode } from "react";
import type { CloudResource } from "../hooks/useCloudResource";
import { CloudNavIcon } from "./CloudNavIcon";
import type { CloudNavigationIcon } from "./cloud-navigation";
import styles from "./Infrastructure.module.css";

export function CloudPageHeader({ title, description, actions }: { title: string; description: string; actions?: ReactNode }) {
  return <header className={styles.header} data-console-heading><div><p className={styles.eyebrow}>ATONIXCORP CLOUD</p><h1>{title}</h1><p>{description}</p></div>{actions}</header>;
}

export function ResourceGrid({ children }: { children: ReactNode }) {
  return <div className={styles.grid}>{children}</div>;
}

export function ResourceSummaryCard({ title, value, detail, href, icon }: { title: string; value: ReactNode; detail: string; href?: string; icon?: CloudNavigationIcon }) {
  return <article className={styles.summary}>{icon && <span className={styles.summaryIcon}><CloudNavIcon name={icon} /></span>}<h2>{href ? <Link href={href}>{title}</Link> : title}</h2><strong>{value}</strong><p>{detail}</p></article>;
}

export function ResourcePanel({ title, description, action, children }: { title: string; description?: string; action?: ReactNode; children: ReactNode }) {
  return <section className={styles.panel}><header><div><h2>{title}</h2>{description && <p>{description}</p>}</div>{action}</header>{children}</section>;
}

export function ResourceStatus({ value }: { value: string }) {
  const normalized = value.toLowerCase();
  const tone = ["healthy", "running", "active", "available", "succeeded", "connected"].includes(normalized) ? "good"
    : ["critical", "failed", "error", "down"].includes(normalized) ? "bad"
    : ["warning", "degraded", "pending", "provisioning", "building"].includes(normalized) ? "warning" : "neutral";
  return <span className={styles.status} data-tone={tone}><span aria-hidden="true" />{value.replaceAll("_", " ")}</span>;
}

export function EmptyState({ title, description, action }: { title: string; description: string; action?: ReactNode }) {
  return <div className={styles.state}><h3>{title}</h3><p>{description}</p>{action}</div>;
}

export function ResourceState<T>({ resource, children }: { resource: CloudResource<T>; children: (data: T) => ReactNode }) {
  if (resource.state === "idle") return <p className={styles.state}>Select an accessible workspace and project to load resources.</p>;
  if (resource.state === "loading") return <p className={styles.state} role="status">Loading resources...</p>;
  if (resource.state === "error") return <div className={styles.error} role="alert">
    <strong>{resource.status === 403 ? "Access denied. " : resource.status === 401 ? "Sign-in required. " : "Resources unavailable. "}</strong>
    {resource.error} {resource.status === 401 ? <Link href="/login">Sign in</Link> : resource.status === 403 ? "Ask your workspace owner for access." : "Use Refresh to retry."}
  </div>;
  return <>{children(resource.data)}</>;
}

export function UtilizationMeter({ label, value, detail }: { label: string; value: number | null; detail: string }) {
  const available = value !== null && Number.isFinite(value) && value >= 0;
  return <div className={styles.meter}>
    <div><strong>{label}</strong><span>{available ? `${value.toFixed(1)}%` : "Unavailable"}</span></div>
    {available && <progress aria-label={label} max={100} value={Math.min(value, 100)} />}
    <p>{detail}</p>
  </div>;
}

export function ResourceTable({ caption, headings, children }: { caption: string; headings: string[]; children: ReactNode }) {
  return <div className={styles.tableWrap} tabIndex={0} role="region" aria-label={caption}><table className={styles.table}>
    <caption className={styles.srOnly}>{caption}</caption><thead><tr>{headings.map((heading) => <th scope="col" key={heading}>{heading}</th>)}</tr></thead><tbody>{children}</tbody>
  </table></div>;
}

export function QuickActions({ children }: { children: ReactNode }) {
  return <nav className={styles.actions} aria-label="Infrastructure actions">{children}</nav>;
}

export function SearchFilterBar({ search, onSearch, status, onStatus, statuses }: { search: string; onSearch: (value: string) => void; status: string; onStatus: (value: string) => void; statuses: string[] }) {
  return <div className={styles.filters}>
    <label>Search resources<input type="search" value={search} onChange={(event) => onSearch(event.target.value)} placeholder="Name, status or address" /></label>
    <label>Status<select value={status} onChange={(event) => onStatus(event.target.value)}><option value="">All statuses</option>{statuses.map((item) => <option key={item} value={item}>{item}</option>)}</select></label>
  </div>;
}

export function ResourcePagination({ page, total, pageSize, onPage }: { page: number; total: number; pageSize: number; onPage: (page: number) => void }) {
  const pages = Math.max(1, Math.ceil(total / pageSize));
  return <nav className={styles.pagination} aria-label="Resource pages">
    <span>{total} resources / page {page + 1} of {pages}</span>
    <button type="button" disabled={page === 0} onClick={() => onPage(page - 1)}>Previous</button>
    <button type="button" disabled={page + 1 >= pages} onClick={() => onPage(page + 1)}>Next</button>
  </nav>;
}
