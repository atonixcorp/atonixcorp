import Link from "next/link";
import PublicShell from "../shared/marketing/PublicShell";
import styles from "../shared/marketing/PublicSite.module.css";

export const metadata = { title: "Account Portal | AtonixCorp", description: "Sign in to your AtonixCorp cloud workspace, create an account, or find help with your next step." };

export default function PortalLanding() {
  return <PublicShell><section className={`${styles.pageHero} ${styles.portalHero}`}><div className={styles.container}><p className={styles.kicker}>YOUR ATONIXCORP STARTING POINT</p><h1>Welcome.<br />Where do you want to go?</h1><p>Your cloud workspace, account access and support. Choose the next step that is right for you.</p></div></section>
    <section className={styles.section}><div className={styles.container}><div className={styles.entryGrid}>
      <article><span className={styles.kicker}>CLOUD CONSOLE</span><h2>Back to your workspace.</h2><p>Manage your authorized resources, review activity and pick up where you left off.</p><Link className={styles.primary} href="/dashboard">Open cloud console <span aria-hidden="true">&rarr;</span></Link></article>
      <article><span className={styles.kicker}>NEW TO ATONIXCORP?</span><h2>Start with an account.</h2><p>Create your account, then establish a workspace or get access from your team.</p><Link className={styles.secondary} href="/onboard">Create an account</Link></article>
      <article><span className={styles.kicker}>SUPPORT</span><h2>Find the right help.</h2><p>Track account-linked support tickets or contact us about a general question.</p><Link className={styles.secondary} href="/support">Visit support</Link></article>
    </div></div></section>
    <section className={`${styles.section} ${styles.softSection}`}><div className={styles.container}><header className={styles.sectionHeading}><div><p className={styles.kicker}>BEFORE YOU SIGN IN</p><h2>A few useful things to know.</h2></div></header><div className={styles.faq}>
      <details><summary>What is the difference between an account and a workspace?</summary><p>Your account identifies you. A workspace groups resources and access, while projects organize resources within it. Access is checked by platform services.</p></details>
      <details><summary>Does creating an account provision infrastructure?</summary><p>No. Account creation is separate from creating cloud resources. Availability and provisioning depend on the infrastructure configured for your workspace.</p></details>
      <details><summary>Where should I ask a product or partnership question?</summary><p>Use the <Link href="/contact">contact page</Link> for general inquiries. Use the support workspace for private, account-related issues.</p></details>
    </div></div></section>
  </PublicShell>;
}
