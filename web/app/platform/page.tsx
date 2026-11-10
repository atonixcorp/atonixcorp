import Link from "next/link";
import PublicShell from "../shared/marketing/PublicShell";
import { ArchitectureVisual, PublicCTA } from "../shared/marketing/PublicComponents";
import { CloudNavIcon } from "../shared/ui/CloudNavIcon";
import styles from "../shared/marketing/PublicSite.module.css";

export const metadata = { title: "Platform Architecture | AtonixCorp", description: "Understand how AtonixCorp connects workspaces, platform APIs and infrastructure providers without exposing provider dashboards." };

export default function PlatformPage() {
  return <PublicShell>
    <section className={`${styles.pageHero} ${styles.darkHero}`}><div className={`${styles.container} ${styles.split}`}><div><p className={styles.kicker}>THE PLATFORM / HOW IT FITS TOGETHER</p><h1>One workspace.<br />A deliberate architecture.</h1><p>AtonixCorp connects the experience of managing infrastructure with the services that actually run it. Clear interfaces. Explicit ownership. No collection of disconnected provider dashboards.</p><div className={styles.actions}><a className={styles.primary} href="#architecture">Explore the architecture <span aria-hidden="true">&darr;</span></a><Link className={styles.secondary} href="/products">Browse products</Link></div></div><ArchitectureVisual compact /></div></section>
    <section className={styles.section} id="architecture"><div className={styles.container}><header className={styles.sectionHeading}><div><p className={styles.kicker}>FOUR LAYERS. ONE CONNECTED EXPERIENCE.</p><h2>From the console<br />to the infrastructure underneath.</h2></div></header>
      <div className={styles.platformLayers}>{[
        { title: "Your workspace", copy: "The console organizes resources by workspace and project. A resource's ownership remains visible as you move between compute, storage and networking.", icon: "overview" as const },
        { title: "The AtonixCorp API layer", copy: "Authenticated requests use platform resource models and permissions. The browser does not need infrastructure-provider credentials.", icon: "security" as const },
        { title: "Service & provider boundaries", copy: "Service adapters translate infrastructure-specific operations into AtonixCorp workflows. A configured provider is not a promise that every product capability is implemented.", icon: "networking" as const },
        { title: "Resources & reported state", copy: "Provisioning state, operation records and available telemetry return to the console. Missing telemetry remains unknown rather than being presented as healthy.", icon: "observability" as const },
      ].map((layer, index) => <article className={styles.platformLayer} key={layer.title}><span>0{index + 1}</span><div><h3>{layer.title}</h3><p>{layer.copy}</p></div><CloudNavIcon name={layer.icon} /></article>)}</div>
    </div></section>
    <section className={`${styles.section} ${styles.softSection}`} id="principles"><div className={`${styles.container} ${styles.split}`}><div className={styles.splitCopy}><p className={styles.kicker}>ENGINEERING PRINCIPLES</p><h2>Clarity is a platform feature.</h2><p>We are building around consistent ownership and honest operational information, not a screen full of implied guarantees.</p><Link className={styles.textLink} href="/about">Our direction <span aria-hidden="true">&rarr;</span></Link></div><ul className={styles.checkList}><li>Server-side authorization remains the source of truth.</li><li>Submitted requests are not the same as completed operations.</li><li>Physical capacity, quotas and allocations are different measurements.</li><li>Unconnected services are labeled instead of populated with demo resources.</li><li>Provider implementation details stay behind platform interfaces.</li></ul></div></section>
    <PublicCTA title="Understand the foundation before you build." description="Explore the developer resources or talk with us about an integration." />
  </PublicShell>;
}
