import Link from "next/link";
import PublicShell from "../../shared/marketing/PublicShell";
import { PublicCTA } from "../../shared/marketing/PublicComponents";
import styles from "../../shared/marketing/PublicSite.module.css";

export const metadata = {
  title: "Platform Roadmap | AtonixCorp",
  description: "Explore AtonixCorp's connected workflows, integration priorities and future development areas without confusing plans with available services.",
};

export default function RoadmapPage() {
  return <PublicShell>
    <section className={styles.pageHero}><div className={styles.container}><nav className={styles.breadcrumb} aria-label="Breadcrumb"><Link href="/about">About AtonixCorp</Link><span aria-hidden="true">/</span><span aria-current="page">Platform roadmap</span></nav><p className={styles.kicker}>DIRECTION, NOT DELIVERY PROMISES</p><h1>Build the foundation. Prove the next layer.</h1><p>A clear view of the workflows we are improving, the integrations that need more depth and the capabilities we are working toward. These are development areas, not scheduled releases or service commitments.</p><div className={styles.actions}><Link className={styles.primary} href="/products">Explore current products <span aria-hidden="true">&rarr;</span></Link><Link className={styles.secondary} href="/contact">Discuss your requirements</Link></div></div></section>
    <section className={styles.section}><div className={styles.container}><header className={styles.sectionHeading}><div><p className={styles.kicker}>THREE DEVELOPMENT HORIZONS</p><h2>Progress with explicit boundaries.</h2></div></header><div className={styles.timeline}>
      <article><span className={styles.kicker}>CONNECTED WORKFLOWS</span><h3>The cloud foundation</h3><p>Continue improving workspace-scoped compute, block and object storage, networking, container deployments and operational visibility. Availability depends on configured services and project authorization.</p></article>
      <article><span className={styles.kicker}>INTEGRATION WORK</span><h3>Depth before breadth</h3><p>Strengthen lifecycle operations, validation, permissions, telemetry, recovery and synchronization between service records and infrastructure. Provisioning state must remain distinct from workload readiness.</p></article>
      <article><span className={styles.kicker}>FUTURE CAPABILITIES</span><h3>More demanding workloads</h3><p>Develop the foundations for integrated HPC, cluster management, advanced data services and inference without presenting unfinished services as available.</p></article>
    </div><div className={styles.availability}><div><p className={styles.kicker}>PLAN WITH EVIDENCE</p><h2>A roadmap is not an availability guarantee.</h2></div><ul><li>No release dates, uptime commitments or capacity guarantees are implied.</li><li>Check product limitations and your configured integrations before making deployment plans.</li><li>Managed backups, general cluster administration and other unconnected features remain unavailable until their integrations are delivered and validated.</li></ul></div></div></section>
    <section className={`${styles.section} ${styles.softSection}`}><div className={styles.container}><div className={styles.resourceGrid}>
      <Link className={styles.resourceCard} href="/developers/open-source"><span className={styles.kicker}>FOLLOW THE IMPLEMENTATION</span><h3>Look beyond the roadmap</h3><p>Explore the source, development guidance and project issues to understand the implementation behind a capability.</p><span>Explore open source <span aria-hidden="true">&rarr;</span></span></Link>
      <Link className={styles.resourceCard} href="/developers/community"><span className={styles.kicker}>SHARE YOUR CONTEXT</span><h3>Explain the workload</h3><p>Help us understand the problem, dependencies and operational constraints behind the feature you need.</p><span>Find a conversation <span aria-hidden="true">&rarr;</span></span></Link>
    </div></div></section>
    <PublicCTA title="Help shape what comes next." description="We welcome technical conversations, research collaborations and thoughtful partnerships." />
  </PublicShell>;
}
