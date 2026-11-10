import Link from "next/link";
import { CloudPageHeader, QuickActions, ResourcePanel, ResourceStatus } from "./Infrastructure";
import { cloudServices, type CloudService } from "./cloud-navigation";
import { scopedCloudHref } from "../lib/cloud-scope";
import styles from "./CloudServiceWorkspace.module.css";

type Scope = { tenantID: number; projectID: number };

export function CloudServiceWorkspace({ service, scope }: { service: CloudService; scope: Scope }) {
  const partial = service.connected.length > 0;
  const href = (path: string) => scopedCloudHref(path, scope.tenantID, scope.projectID);
  return <main className={styles.page}>
    <nav className={styles.breadcrumb} aria-label="Breadcrumb"><Link href={href("/dashboard")}>Cloud</Link><span>/</span><Link href={href("/cloud/services")}>{service.group}</Link><span>/</span><span aria-current="page">{service.title}</span></nav>
    <CloudPageHeader title={service.title} description={service.description} />
    <section className={styles.availability} aria-labelledby="service-availability-title">
      <ResourceStatus value={partial ? "Limited integration" : "Not connected"} />
      <h2 id="service-availability-title">{partial ? "Available workflows, with clear limits" : "This service is not available in the console yet"}</h2>
      <p>{partial
        ? "Only the workflows listed below are connected. This does not indicate that a cluster or provider is healthy, or that your account has access."
        : "A tenant-authorized resource API and lifecycle integration are required before this workspace can manage infrastructure. No resources, metrics or operational controls are simulated here."}</p>
    </section>
    {service.related && <QuickActions>{service.related.map((link) => <Link href={href(link.href)} key={link.href}>{link.label}</Link>)}</QuickActions>}
    <div className={styles.columns}>
      <ResourcePanel title="Connected workflows" description="Implementation coverage, not live service health">
        {partial ? <ul className={styles.list}>{service.connected.map((item) => <li key={item}>{item}</li>)}</ul>
          : <p className={styles.note}>No console management workflow is connected for this service. An empty inventory would incorrectly imply that the service is operational.</p>}
      </ResourcePanel>
      <ResourcePanel title="Not yet available" description="Required before this service can be considered complete">
        <ul className={styles.list}>{service.pending.map((item) => <li key={item}>{item}</li>)}</ul>
      </ResourcePanel>
    </div>
    <ResourcePanel title="Integration requirements">
      <p className={styles.note}>AtonixCorp APIs must enforce workspace and project permissions, validate requests, persist operations, and reconcile resource state with the configured infrastructure. Creation, updates, deletion, errors and refresh must be verified before enabling these controls.</p>
    </ResourcePanel>
  </main>;
}

export function CloudServiceDirectory({ scope }: { scope: Scope }) {
  const href = (path: string) => scopedCloudHref(path, scope.tenantID, scope.projectID);
  return <main className={styles.page}>
    <CloudPageHeader title="Cloud service directory" description="Explore the complete platform hierarchy and see exactly which workflows are connected. Coverage is not a statement of provider availability, permissions or production readiness." />
    {(["Compute", "Storage", "Network", "Orchestration"] as const).map((group) => <section key={group} className={styles.group} aria-label={group}>
      <h2>{group}</h2>
      <div className={styles.catalog}>
        {cloudServices.filter((service) => service.group === group).map((service) => <article className={styles.card} key={service.id}>
          <div><h3><Link href={href(service.href)}>{service.title}</Link></h3><ResourceStatus value={service.connected.length ? "Partial support" : "Not connected"} /></div>
          <p>{service.description}</p>
          {service.connected.length > 0 && <ul>{service.connected.map((item) => <li key={item}>{item}</li>)}</ul>}
          <details><summary>Missing capabilities ({service.pending.length})</summary><ul>{service.pending.map((item) => <li key={item}>{item}</li>)}</ul></details>
          <Link className={styles.open} href={href(service.href)}>{service.connected.length ? "Open workspace" : "View integration requirements"} <span aria-hidden="true">&rarr;</span></Link>
        </article>)}
      </div>
    </section>)}
  </main>;
}
