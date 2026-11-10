import Link from "next/link";
import PublicShell from "../../shared/marketing/PublicShell";
import { PublicCTA } from "../../shared/marketing/PublicComponents";
import styles from "../../shared/marketing/PublicSite.module.css";

export const metadata = {
  title: "Developer Community | AtonixCorp",
  description: "Connect with AtonixCorp builders, discuss technical ideas, report reproducible issues and find the right channel for private support.",
};

export default function CommunityPage() {
  return <PublicShell>
    <section className={`${styles.pageHero} ${styles.portalHero}`}><div className={styles.container}><p className={styles.kicker}>DEVELOPER COMMUNITY</p><h1>Better questions. Better systems. Together.</h1><p>Exchange ideas with other builders, share what you are learning and help make the platform easier to understand. Choose a channel that fits your question.</p><div className={styles.actions}><a className={styles.primary} href="https://discord.gg/EGqJ778a55">Join the Discord community <span aria-hidden="true">&rarr;</span></a><Link className={styles.secondary} href="/developers/open-source">Contribute to the project</Link></div></div></section>
    <section className={styles.section}><div className={styles.container}><header className={styles.sectionHeading}><div><p className={styles.kicker}>FIND THE RIGHT CONVERSATION</p><h2>Ideas in public. Account details in private.</h2></div></header><div className={styles.resourceGrid}>
      <a className={styles.resourceCard} href="https://discord.gg/EGqJ778a55"><span className={styles.kicker}>DISCUSS & LEARN</span><h3>Community conversations</h3><p>Talk through architecture, development questions and lessons from working with the project. Community channels are not a private support or emergency-response service.</p><span>Open Discord <span aria-hidden="true">&rarr;</span></span></a>
      <a className={styles.resourceCard} href="https://github.com/atonixcorp/atonixcorp/issues"><span className={styles.kicker}>REPRODUCIBLE FEEDBACK</span><h3>Project issues</h3><p>Search existing issues before reporting a public, non-sensitive bug or improvement. Include the relevant version, reproduction steps and expected behavior.</p><span>Browse GitHub issues <span aria-hidden="true">&rarr;</span></span></a>
      <Link className={styles.resourceCard} href="/support"><span className={styles.kicker}>PRIVATE ACCOUNT HELP</span><h3>Support workspace</h3><p>Sign in to submit and track an account-linked support ticket. Use this route for account or infrastructure questions that should not be discussed publicly.</p><span>Open support <span aria-hidden="true">&rarr;</span></span></Link>
      <Link className={styles.resourceCard} href="/contact"><span className={styles.kicker}>PARTNERSHIPS & QUESTIONS</span><h3>Contact the team</h3><p>Discuss collaboration, platform requirements or where to direct a sensitive concern. Do not include passwords, API keys or other secrets in your message.</p><span>Contact AtonixCorp <span aria-hidden="true">&rarr;</span></span></Link>
    </div></div></section>
    <section className={`${styles.section} ${styles.softSection}`}><div className={`${styles.container} ${styles.split}`}><div className={styles.splitCopy}><p className={styles.kicker}>MAKE FEEDBACK USEFUL</p><h2>Give people enough context to help.</h2><p>A focused example is easier to investigate than a full dump of an environment. Explain what you expected, what happened and how someone else can reproduce it.</p></div><div className={styles.resourceCard}><span className={styles.kicker}>BEFORE YOU POST</span><h3>Protect your workspace.</h3><ul className={styles.checkList}><li>Remove credentials, tokens and personal or customer data.</li><li>Share only the smallest relevant, sanitized log excerpt.</li><li>Keep the discussion respectful and specific.</li><li>Use private channels for sensitive account concerns.</li></ul></div></div></section>
    <PublicCTA title="Have a collaboration in mind?" description="Tell us about the research, application or infrastructure challenge you are working on." />
  </PublicShell>;
}
