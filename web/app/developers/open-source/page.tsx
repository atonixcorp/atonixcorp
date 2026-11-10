import Link from "next/link";
import PublicShell from "../../shared/marketing/PublicShell";
import { PublicCTA } from "../../shared/marketing/PublicComponents";
import styles from "../../shared/marketing/PublicSite.module.css";

export const metadata = {
  title: "Open Source & Contributions | AtonixCorp",
  description: "Explore the AtonixCorp repository, understand component boundaries and prepare focused, verifiable contributions.",
};

export default function OpenSourcePage() {
  return <PublicShell>
    <section className={styles.pageHero}><div className={styles.container}><nav className={styles.breadcrumb} aria-label="Breadcrumb"><Link href="/developers">Developer hub</Link><span aria-hidden="true">/</span><span aria-current="page">Open source</span></nav><p className={styles.kicker}>SOURCE & CONTRIBUTIONS</p><h1>Understand the code. Improve the foundation.</h1><p>Explore how the platform is assembled, investigate a focused problem and share a change that others can understand and verify.</p><div className={styles.actions}><a className={styles.primary} href="https://github.com/atonixcorp/atonixcorp">Explore the repository <span aria-hidden="true">&rarr;</span></a><a className={styles.secondary} href="https://github.com/atonixcorp/atonixcorp/issues">Browse project issues</a></div></div></section>
    <section className={styles.section}><div className={styles.container}><header className={styles.sectionHeading}><div><p className={styles.kicker}>A FOCUSED CONTRIBUTION</p><h2>Make the change easy to follow.</h2><p>Different components have different dependencies and integration boundaries. Start with the documentation and existing tests for the area you plan to change.</p></div></header><div className={styles.timeline}>
      <article><span className={styles.kicker}>01 / UNDERSTAND</span><h3>Read the surrounding system</h3><p>Check the component instructions, related issues and existing implementation. Separate a missing provider integration from a bug in an existing workflow.</p></article>
      <article><span className={styles.kicker}>02 / IMPLEMENT</span><h3>Keep the scope clear</h3><p>Use established patterns, preserve authorization boundaries and update directly related documentation. Keep unrelated refactors out of a focused change.</p></article>
      <article><span className={styles.kicker}>03 / VERIFY</span><h3>Show what you tested</h3><p>Run the relevant checks, describe the result and state any integration limits. A successful build alone does not demonstrate live infrastructure behavior.</p></article>
    </div></div></section>
    <section className={`${styles.section} ${styles.softSection}`}><div className={styles.container}><div className={styles.resourceGrid}>
      <Link className={styles.resourceCard} href="/developers/documentation"><span className={styles.kicker}>LOCAL DEVELOPMENT</span><h3>Prepare your environment</h3><p>Find setup instructions and service-specific guides before running the components you want to work on.</p><span>Open documentation <span aria-hidden="true">&rarr;</span></span></Link>
      <a className={styles.resourceCard} href="https://github.com/atonixcorp/atonixcorp/blob/main/LICENSE.md"><span className={styles.kicker}>TERMS & ATTRIBUTION</span><h3>Read the applicable license</h3><p>Review the repository license and any component-specific terms before using or distributing code. Preserve required notices and do not assume every dependency has the same terms.</p><span>Read the repository license <span aria-hidden="true">&rarr;</span></span></a>
    </div><div className={styles.availability}><div><p className={styles.kicker}>SHARE RESPONSIBLY</p><h2>Keep secrets out of contributions.</h2></div><p>Do not commit environment secrets, credentials or private customer data. Use sanitized reproduction steps, and contact the team privately before sharing sensitive concerns publicly.</p></div></div></section>
    <PublicCTA title="Want to contribute a larger idea?" description="Discuss the use case and boundaries before investing in a broad implementation." />
  </PublicShell>;
}
