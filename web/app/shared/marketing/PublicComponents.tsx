import Link from "next/link";
import { CloudNavIcon } from "../ui/CloudNavIcon";
import { products, type Product } from "./content";
import styles from "./PublicSite.module.css";

export function ProductCards({ items = products }: { items?: Product[] }) {
  return <div className={styles.productGrid}>{items.map((product) => <Link href={`/products/${product.slug}`} className={styles.productCard} key={product.slug}>
    <span className={styles.productIcon}><CloudNavIcon name={product.icon} /></span>
    <span className={styles.kicker}>{product.category}</span>
    <h3>{product.name}</h3><p>{product.summary}</p><span className={styles.cardLink}>Explore product <span aria-hidden="true">&rarr;</span></span>
  </Link>)}</div>;
}

export function PublicCTA({ title = "Your next build starts with a conversation.", description = "Tell us what you want to run. Let's find the right place to start." }: { title?: string; description?: string }) {
  return <section className={styles.cta}><div className={styles.container}>
    <div><p className={styles.kicker}>LET&apos;S BUILD WHAT&apos;S NEXT</p><h2>{title}</h2><p>{description}</p></div>
    <div className={styles.actions}><Link className={styles.primary} href="/contact">Talk to our team <span aria-hidden="true">&rarr;</span></Link><Link className={styles.secondary} href="/onboard">Create an account</Link></div>
  </div></section>;
}

export function ArchitectureVisual({ compact = false }: { compact?: boolean }) {
  return <figure className={`${styles.architecture} ${compact ? styles.compact : ""}`} aria-label="Conceptual architecture: one workspace connects compute, containers, data and networking. Not live infrastructure.">
    <div className={styles.visualTop}><span><i /> ATONIXCORP CLOUD</span><span>ARCHITECTURE VIEW</span></div>
    <div className={styles.workspaceNode}><span className={styles.nodeIcon}><CloudNavIcon name="overview" /></span><div><small>YOUR WORKSPACE</small><strong>One place. Connected infrastructure.</strong></div><span className={styles.nodePlus} aria-hidden="true">+</span></div>
    <div className={styles.connectors} aria-hidden="true"><i /><i /><i /></div>
    <div className={styles.serviceNodes}>{[
      { icon: "compute" as const, label: "Compute", detail: "Run your workloads" },
      { icon: "containers" as const, label: "Containers", detail: "Deploy your applications" },
      { icon: "storage" as const, label: "Data", detail: "Keep your work connected" },
    ].map((node) => <div key={node.label}><CloudNavIcon name={node.icon} /><strong>{node.label}</strong><small>{node.detail}</small></div>)}</div>
    <div className={styles.networkLayer}><CloudNavIcon name="networking" /><span>Networks & connectivity</span><span className={styles.layerLine} aria-hidden="true" /></div>
    <div className={styles.foundation}><span>Identity & access</span><span>Platform APIs</span><span>Observability</span></div>
    <figcaption>YOUR IDEAS. YOUR WORKLOADS. ONE CONNECTED FOUNDATION.</figcaption>
  </figure>;
}
