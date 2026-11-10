import Link from "next/link";
import PublicShell from "../shared/marketing/PublicShell";
import { PublicCTA } from "../shared/marketing/PublicComponents";
import styles from "../shared/marketing/PublicSite.module.css";

export const metadata = { title: "Developer Hub | AtonixCorp", description: "Find AtonixCorp source code, service documentation, API references, CLI tooling and community resources." };
const resources = [
  { title: "Platform documentation", category: "GETTING STARTED", copy: "Follow the setup path and find focused guides for the platform services you need.", href: "/developers/documentation", cta: "Open documentation" },
  { title: "API & CLI resources", category: "BUILD AN INTEGRATION", copy: "Explore API contracts, authentication boundaries and the current command-line tooling.", href: "/developers/api-cli", cta: "Explore interfaces" },
  { title: "Open source", category: "WORK WITH THE CODE", copy: "Find the repository, understand component boundaries and prepare a focused contribution.", href: "/developers/open-source", cta: "Explore the project" },
  { title: "Developer community", category: "CONNECT WITH BUILDERS", copy: "Choose the right place for technical discussion, reproducible issues or private account support.", href: "/developers/community", cta: "Find your community" },
];

export default function DevelopersPage() {
  return <PublicShell><section className={`${styles.pageHero} ${styles.darkHero}`}><div className={`${styles.container} ${styles.split}`}><div><p className={styles.kicker}>DEVELOPER HUB</p><h1>Build with context.<br />Not guesswork.</h1><p>Find the source, understand the interfaces and get familiar with the system behind your workspace.</p><div className={styles.actions}><Link className={styles.primary} href="/developers/documentation">Start with documentation <span aria-hidden="true">&rarr;</span></Link><Link className={styles.secondary} href="/developers/open-source">Explore open source</Link></div></div><div className={styles.codeCard}><div className={styles.codeHeading}><i /><i /><i /><span>DEVELOPER CHECKLIST</span></div><pre><code>{`01  Read the setup instructions\n02  Configure your local environment\n03  Verify service dependencies\n04  Authenticate your requests\n05  Work within your project scope\n06  Test errors, not just success`}</code></pre><p>A practical sequence, not executable commands.</p></div></div></section>
    <section className={styles.section} id="tools"><div className={styles.container}><header className={styles.sectionHeading}><div><p className={styles.kicker}>DOCUMENTATION & TOOLING</p><h2>Get closer to the platform.</h2></div></header><div className={styles.resourceGrid}>{resources.map((resource) => <Link className={styles.resourceCard} href={resource.href} key={resource.title}><span className={styles.kicker}>{resource.category}</span><h3>{resource.title}</h3><p>{resource.copy}</p><span>{resource.cta} <span aria-hidden="true">&rarr;</span></span></Link>)}</div></div></section>
    <section className={`${styles.section} ${styles.softSection}`} id="community"><div className={styles.container}><div className={styles.splitCopy}><p className={styles.kicker}>BUILD IN GOOD COMPANY</p><h2>A dedicated space for the conversation.</h2><p>Find community channels and guidance on sharing technical questions safely.</p><Link className={styles.textLink} href="/developers/community">Visit the community page <span aria-hidden="true">&rarr;</span></Link></div></div></section>
    <PublicCTA title="Building an integration?" description="Tell us about your use case and the interfaces you need." />
  </PublicShell>;
}
