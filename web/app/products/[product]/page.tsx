import Link from "next/link";
import { notFound } from "next/navigation";
import PublicShell from "@/app/shared/marketing/PublicShell";
import { ProductCards, PublicCTA } from "@/app/shared/marketing/PublicComponents";
import { products } from "@/app/shared/marketing/content";
import { CloudNavIcon } from "@/app/shared/ui/CloudNavIcon";
import styles from "@/app/shared/marketing/PublicSite.module.css";

export function generateStaticParams() { return products.map((product) => ({ product: product.slug })); }

export async function generateMetadata({ params }: { params: Promise<{ product: string }> }) {
  const { product: slug } = await params;
  const product = products.find((item) => item.slug === slug);
  if (!product) notFound();
  return { title: `${product.name} | AtonixCorp`, description: product.summary };
}

export default async function ProductPage({ params }: { params: Promise<{ product: string }> }) {
  const { product: slug } = await params;
  const product = products.find((item) => item.slug === slug);
  if (!product) notFound();
  return <PublicShell>
    <section className={`${styles.pageHero} ${styles.productHero}`}><div className={styles.container}><div>
      <nav className={styles.breadcrumb} aria-label="Breadcrumb"><Link href="/products">Products</Link><span>/</span><span aria-current="page">{product.name}</span></nav>
      <p className={styles.kicker}>{product.name}</p><h1>{product.headline}</h1><p>{product.description}</p>
      <div className={styles.actions}><Link className={styles.primary} href={product.workspace}>Open workspace <span aria-hidden="true">&rarr;</span></Link><Link className={styles.secondary} href="/contact">Talk to the team</Link></div>
    </div><div className={styles.productSculpture} aria-hidden="true"><CloudNavIcon name={product.icon} /><span>{product.category} / ATONIXCORP</span></div></div></section>
    <section className={styles.section}><div className={styles.container}>
      <header className={styles.sectionHeading}><div><p className={styles.kicker}>WORKFLOW FIRST</p><h2>What you can build on.</h2></div></header>
      <div className={styles.features}>{product.features.map((feature, index) => <article key={feature.title}><span>0{index + 1}</span><h3>{feature.title}</h3><p>{feature.description}</p></article>)}</div>
      <section className={styles.availability} aria-labelledby="product-availability"><div><p className={styles.kicker}>BEFORE YOU DEPLOY</p><h2 id="product-availability">Current integration & availability</h2><p>Keep these boundaries in mind when planning your workload.</p></div><ul>{product.considerations.map((item) => <li key={item}>{item}</li>)}</ul></section>
    </div></section>
    <section className={`${styles.section} ${styles.softSection}`}><div className={styles.container}><header className={styles.sectionHeading}><div><p className={styles.kicker}>PART OF THE BIGGER PICTURE</p><h2>Connect the rest of your stack.</h2></div><Link className={styles.textLink} href="/products">All products <span aria-hidden="true">&rarr;</span></Link></header><ProductCards items={products.filter((item) => item.slug !== slug).slice(0, 3)} /></div></section>
    <PublicCTA title={`Plan your ${product.name.toLowerCase()} workflow.`} />
  </PublicShell>;
}
