import Link from "next/link";
import PublicShell from "../shared/marketing/PublicShell";
import { ProductCards, PublicCTA } from "../shared/marketing/PublicComponents";
import styles from "../shared/marketing/PublicSite.module.css";

export const metadata = { title: "Cloud Products | AtonixCorp", description: "Explore compute, storage, networking, containers, database infrastructure and observability in AtonixCorp Cloud." };

export default function ProductsPage() {
  return <PublicShell>
    <section className={styles.pageHero}><div className={styles.container}><p className={styles.kicker}>PRODUCTS / YOUR CLOUD TOOLKIT</p><h1>The right building blocks.<br />A connected foundation.</h1><p>Explore what each service does, how it fits into your workload, and which capabilities are connected today.</p><div className={styles.actions}><Link className={styles.primary} href="/onboard">Create an account <span aria-hidden="true">&rarr;</span></Link><Link className={styles.secondary} href="/platform">Understand the architecture</Link></div></div></section>
    <section className={styles.section}><div className={styles.container}><ProductCards /><div className={styles.availability}><div><p className={styles.kicker}>MAKE AN INFORMED CHOICE</p><h2>Capabilities, not assumptions.</h2><p>Product pages distinguish connected workflows from work that is still in development.</p></div><div><ul><li>Resource availability depends on your workspace and configured infrastructure.</li><li>Published pricing, regional capacity and service-level commitments are not currently available.</li><li>Talk to the team before planning a production or regulated workload.</li></ul><div className={styles.actions}><Link className={styles.textLink} href="/contact">Discuss your requirements <span aria-hidden="true">&rarr;</span></Link></div></div></div></div></section>
    <PublicCTA title="Find the right starting point for your workload." />
  </PublicShell>;
}
