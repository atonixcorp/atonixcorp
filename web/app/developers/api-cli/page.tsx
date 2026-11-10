import Link from "next/link";
import PublicShell from "../../shared/marketing/PublicShell";
import { PublicCTA } from "../../shared/marketing/PublicComponents";
import styles from "../../shared/marketing/PublicSite.module.css";

export const metadata = {
  title: "API & CLI Resources | AtonixCorp",
  description: "Explore AtonixCorp API specifications, integration principles and the current command-line tooling without assuming unsupported service coverage.",
};

export default function APIResourcesPage() {
  return <PublicShell>
    <section className={`${styles.pageHero} ${styles.darkHero}`}><div className={`${styles.container} ${styles.split}`}>
      <div><p className={styles.kicker}>DEVELOPERS / API & CLI</p><h1>Build integrations around clear contracts.</h1><p>Explore the platform interfaces, understand authorization and follow operations through to their result. Choose the tooling that matches your deployed services.</p><div className={styles.actions}><a className={styles.primary} href="https://github.com/atonixcorp/atonixcorp/blob/main/api/docs/swagger.yaml">Open the API specification <span aria-hidden="true">&rarr;</span></a><Link className={styles.secondary} href="/developers/documentation">Read the documentation</Link></div></div>
      <div className={styles.codeCard}><div className={styles.codeHeading}><i /><i /><i /><span>INTEGRATION LIFECYCLE</span></div><pre><code>{`Identify the service contract\n  -> authenticate the caller\n  -> select an authorized scope\n  -> validate the request\n  -> follow operation status\n  -> handle failure explicitly`}</code></pre><p>A conceptual workflow, not an executable API request.</p></div>
    </div></section>
    <section className={styles.section}><div className={styles.container}><header className={styles.sectionHeading}><div><p className={styles.kicker}>TWO WAYS INTO THE CODE</p><h2>Choose your integration starting point.</h2></div></header><div className={styles.resourceGrid}>
      <article className={styles.resourceCard}><span className={styles.kicker}>HTTP INTERFACES</span><h3>API reference</h3><p>The repository includes an API specification and service-specific guides. Check routes, request fields and authentication against the implementation you deploy; the specification is not a promise that every provider operation is connected.</p><a className={styles.textLink} href="https://github.com/atonixcorp/atonixcorp/tree/main/api/docs">Browse API reference files <span aria-hidden="true">&rarr;</span></a></article>
      <article className={styles.resourceCard}><span className={styles.kicker}>COMMAND-LINE TOOLING</span><h3>Explore the CLI</h3><p>The CLI is a separate repository component with evolving command implementations. Review its source and configuration before using it. Do not assume it supports every console workflow or that every command is production-ready.</p><a className={styles.textLink} href="https://github.com/atonixcorp/atonixcorp/tree/main/cli">Inspect CLI source <span aria-hidden="true">&rarr;</span></a></article>
    </div></div></section>
    <section className={`${styles.section} ${styles.softSection}`}><div className={styles.container}><header className={styles.sectionHeading}><div><p className={styles.kicker}>DESIGN FOR REAL OPERATIONS</p><h2>The response is only part of the contract.</h2></div></header><div className={styles.features}>
      <article><span>01 / AUTHORIZATION</span><h3>Keep ownership explicit</h3><p>Authentication does not grant access to every workspace or project. Handle access-denied responses and validate the scope of each resource request.</p></article>
      <article><span>02 / OPERATIONS</span><h3>Track completion</h3><p>A submitted provisioning request is not a running resource. Follow the service&apos;s operation state and distinguish completion from failure or an unavailable dependency.</p></article>
      <article><span>03 / DIAGNOSTICS</span><h3>Share context, not secrets</h3><p>Record relevant status and request identifiers when available. Remove tokens, credentials and private workload data before sharing diagnostics.</p></article>
    </div></div></section>
    <PublicCTA title="Need a specific integration?" description="Share the service, operation and workflow you need so we can discuss the current interface and its limits." />
  </PublicShell>;
}
