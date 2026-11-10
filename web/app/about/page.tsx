import Link from "next/link";
import PublicShell from "../shared/marketing/PublicShell";
import { PublicCTA } from "../shared/marketing/PublicComponents";
import styles from "../shared/marketing/PublicSite.module.css";

export const metadata = { title: "About AtonixCorp | Our Direction", description: "Learn about AtonixCorp's platform direction, engineering principles and current development priorities." };

export default function AboutPage() {
  return <PublicShell><section className={styles.pageHero}><div className={styles.container}><p className={styles.kicker}>ABOUT ATONIXCORP</p><h1>Ambitious work needs thoughtful foundations.</h1><p>We are building a connected cloud platform for people working on applications, data, research and complex engineering systems.</p><div className={styles.actions}><Link className={styles.primary} href="/contact">Get in touch <span aria-hidden="true">&rarr;</span></Link><Link className={styles.secondary} href="/sponsor">Support the work</Link></div></div></section>
    <section className={styles.section} id="mission"><div className={`${styles.container} ${styles.split}`}><div className={styles.splitCopy}><p className={styles.kicker}>OUR POINT OF VIEW</p><h2>Infrastructure should bring the work together.</h2><p>Cloud systems are made of many technologies. The experience of using them does not have to be fragmented. AtonixCorp is working toward a consistent platform layer for resources, ownership and operations.</p><p>That means improving the foundations alongside the interface, and being clear about what works today and what still needs to be built.</p></div><blockquote className={styles.aboutStatement}>A clearer relationship between the people building, the workloads running and the infrastructure underneath.</blockquote></div></section>
    <section className={`${styles.section} ${styles.softSection}`} id="roadmap"><div className={styles.container}><div className={styles.splitCopy}><p className={styles.kicker}>OUR DIRECTION</p><h2>Explore the platform roadmap.</h2><p>See connected workflows, integration priorities and future capabilities on their own page. Development direction is not a delivery or availability promise.</p><Link className={styles.textLink} href="/about/roadmap">Open the roadmap <span aria-hidden="true">&rarr;</span></Link></div></div></section>
    <PublicCTA title="Help shape what comes next." description="We welcome technical conversations, research collaborations and thoughtful partnerships." />
  </PublicShell>;
}
