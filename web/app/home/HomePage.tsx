import Link from "next/link";
import PublicShell from "../shared/marketing/PublicShell";
import { ArchitectureVisual, ProductCards, PublicCTA } from "../shared/marketing/PublicComponents";
import { solutions } from "../shared/marketing/content";
import { CloudNavIcon } from "../shared/ui/CloudNavIcon";
import styles from "../shared/marketing/PublicSite.module.css";

export default function HomePage() {
  return <PublicShell>
    <section className={styles.homeHero}>
      <div className={styles.container}>
        <div className={styles.heroCopy}>
          <Link className={styles.heroPill} href="/platform"><span /> YOUR WORK. ONE CONNECTED CLOUD. <b aria-hidden="true">&rarr;</b></Link>
          <h1>Your next idea.<br /><em>Built on solid ground.</em></h1>
          <p className={styles.heroLead}>Compute, storage and networking, together in one cloud workspace. Give your applications a foundation you can understand and build on.</p>
          <div className={styles.actions}><Link className={styles.primary} href="/onboard">Start building <span aria-hidden="true">&rarr;</span></Link><Link className={styles.secondary} href="/products">Explore products</Link></div>
          <div className={styles.heroFootnote}><span>Project-focused control</span><span>Open engineering</span><span>Connected infrastructure</span></div>
        </div>
        <ArchitectureVisual />
      </div>
    </section>
    <div className={styles.principleStrip}><div className={styles.container}><strong>LESS FRAGMENTATION. MORE FOCUS.</strong>
      <span><CloudNavIcon name="compute" />Build your applications</span><span><CloudNavIcon name="storage" />Connect your data</span><span><CloudNavIcon name="observability" />Understand your infrastructure</span>
    </div></div>
    <section className={styles.section} id="capabilities"><div className={styles.container}>
      <header className={styles.sectionHeading}><div><p className={styles.kicker}>THE BUILDING BLOCKS</p><h2>One platform.<br />Room for your next big thing.</h2><p>Choose the infrastructure your work needs. Keep it together in a workspace built around your team.</p></div><Link className={styles.textLink} href="/products">Meet the products <span aria-hidden="true">&rarr;</span></Link></header>
      <ProductCards />
    </div></section>
    <section className={`${styles.section} ${styles.softSection}`}><div className={`${styles.container} ${styles.split}`}>
      <div className={styles.splitCopy}><p className={styles.kicker}>YOUR CLOUD, IN CONTEXT</p><h2>Less switching.<br />A clearer picture.</h2><p>Infrastructure should not feel like a collection of disconnected dashboards. AtonixCorp brings your resources, their ownership and their reported state into the same conversation.</p>
        <ul className={styles.checkList}><li>Workspaces and projects that reflect how you work.</li><li>Resource information from platform APIs, not estimates.</li><li>Clear boundaries between available and planned capabilities.</li></ul>
        <Link className={styles.textLink} href="/platform">See how it fits together <span aria-hidden="true">&rarr;</span></Link>
      </div>
      <div className={styles.workflow}><header><span>FROM IDEA TO INFRASTRUCTURE</span><span>WORKFLOW GUIDE</span></header>
        {[
          ["01", "Create your workspace", "Give your team and resources a clear ownership boundary."],
          ["02", "Choose your building blocks", "Start with the configured compute, storage and network options."],
          ["03", "Deploy with context", "Follow provisioning state instead of guessing what is happening."],
          ["04", "Stay close to your workloads", "Review resource inventory and the telemetry your services report."],
        ].map(([step, title, copy]) => <div key={step}><span>{step}</span><div><strong>{title}</strong><p>{copy}</p></div></div>)}
      </div>
    </div></section>
    <section className={styles.section}><div className={styles.container}>
      <header className={styles.sectionHeading}><div><p className={styles.kicker}>BUILT AROUND THE WORK</p><h2>Different ambitions.<br />A thoughtful starting point.</h2></div><Link className={styles.textLink} href="/solutions">Explore solutions <span aria-hidden="true">&rarr;</span></Link></header>
      <div className={styles.solutionGrid}>{solutions.map((solution, index) => <Link className={styles.solutionCard} href={`/solutions/${solution.slug}`} key={solution.slug}>
        <span className={styles.solutionNumber}>0{index + 1} / SOLUTIONS</span><h3>{solution.name}</h3><p>{solution.summary}</p><span className={styles.cardLink}>Find your starting point <span aria-hidden="true">&rarr;</span></span>
      </Link>)}</div>
    </div></section>
    <section className={`${styles.section} ${styles.developerBand}`}><div className={`${styles.container} ${styles.split}`}>
      <div className={styles.splitCopy}><p className={styles.kicker}>FOR PEOPLE WHO BUILD</p><h2>Not a black box.<br />A platform to understand.</h2><p>Explore the source, read the service documentation and understand the interfaces behind your workspace. Good infrastructure starts with clear engineering.</p><div className={styles.actions}><Link className={styles.primary} href="/developers">Developer hub <span aria-hidden="true">&rarr;</span></Link><a className={styles.secondary} href="https://github.com/atonixcorp/atonixcorp">Explore the source</a></div></div>
      <div className={styles.codeCard}><div className={styles.codeHeading}><i /><i /><i /><span>PLATFORM / PROJECT STRUCTURE</span></div><pre><code>{`atonixcorp/\n  api/          Platform API layer\n  cloud/        Infrastructure services\n  cli/          Command-line tooling\n  web/          Cloud console\n  security/     Security components`}</code></pre><p>A view of the project, not a simulated terminal.</p></div>
    </div></section>
    <PublicCTA />
  </PublicShell>;
}
