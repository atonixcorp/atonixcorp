import Link from "next/link";
import PublicShell from "../../shared/marketing/PublicShell";
import { PublicCTA } from "../../shared/marketing/PublicComponents";
import styles from "../../shared/marketing/PublicSite.module.css";

export const metadata = {
  title: "Platform Documentation | AtonixCorp",
  description: "Start with AtonixCorp setup guidance, then explore focused compute, storage, container and observability documentation.",
};

const guides = [
  { title: "Compute integration", file: "openstack-compute.md", copy: "Understand the configured compute integration and the prerequisites behind instance workflows." },
  { title: "Storage services", file: "storage.md", copy: "Review storage configuration, supported operations and the differences between development and infrastructure-backed storage." },
  { title: "Container registries", file: "container-registries.md", copy: "Learn how registry connections and server-held image credentials support container deployments." },
  { title: "Observability", file: "observability.md", copy: "Understand resource health, telemetry sources and the limits of available operational signals." },
];

export default function DocumentationPage() {
  return <PublicShell>
    <section className={styles.pageHero}><div className={styles.container}>
      <nav className={styles.breadcrumb} aria-label="Breadcrumb"><Link href="/developers">Developer hub</Link><span aria-hidden="true">/</span><span aria-current="page">Documentation</span></nav>
      <p className={styles.kicker}>PLATFORM DOCUMENTATION</p><h1>Understand the system before you build on it.</h1>
      <p>A guided starting point for local setup, service configuration and integration boundaries. Detailed technical guides live alongside the source so you can match them to your checkout.</p>
      <div className={styles.actions}><a className={styles.primary} href="https://github.com/atonixcorp/atonixcorp#readme">Read the setup guide <span aria-hidden="true">&rarr;</span></a><Link className={styles.secondary} href="/developers/api-cli">API & CLI resources</Link></div>
    </div></section>
    <section className={styles.section}><div className={`${styles.container} ${styles.split}`}>
      <div className={styles.splitCopy}><p className={styles.kicker}>YOUR FIRST WORKING ENVIRONMENT</p><h2>Start small. Verify each layer.</h2><p>The root development stack runs the web application and its core services. It does not automatically configure every infrastructure integration or standalone project.</p><p>Keep environment files and credentials private. A healthy application container is not proof that a resource provider is ready to provision workloads.</p><Link className={styles.textLink} href="/platform">Explore platform architecture <span aria-hidden="true">&rarr;</span></Link></div>
      <div className={styles.workflow}><header><span>SETUP READING PATH</span><span>THREE STEPS</span></header>
        <div><span>01</span><div><strong>Check prerequisites</strong><p>Review the repository setup instructions and the dependencies required by your selected services.</p></div></div>
        <div><span>02</span><div><strong>Configure your environment</strong><p>Set independent secrets and the service endpoints for your environment. Do not copy production credentials into a test setup.</p></div></div>
        <div><span>03</span><div><strong>Verify the workflow</strong><p>Check readiness, authentication and project access before testing resource operations.</p></div></div>
      </div>
    </div></section>
    <section className={`${styles.section} ${styles.softSection}`}><div className={styles.container}><header className={styles.sectionHeading}><div><p className={styles.kicker}>SERVICE GUIDES</p><h2>Go deeper where your workload needs it.</h2><p>These links open the maintained technical documents on GitHub. Review the revision that matches your deployed code.</p></div></header>
      <div className={styles.resourceGrid}>{guides.map((guide) => <a className={styles.resourceCard} key={guide.file} href={`https://github.com/atonixcorp/atonixcorp/blob/main/api/docs/${guide.file}`}><h3>{guide.title}</h3><p>{guide.copy}</p><span>Read the guide on GitHub <span aria-hidden="true">&rarr;</span></span></a>)}</div>
    </div></section>
    <PublicCTA title="Missing a guide for your use case?" description="Tell us what you are trying to integrate and where the documentation needs more detail." />
  </PublicShell>;
}
