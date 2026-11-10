import Link from "next/link";
import { notFound } from "next/navigation";
import PublicShell from "@/app/shared/marketing/PublicShell";
import { ProductCards, PublicCTA } from "@/app/shared/marketing/PublicComponents";
import { products, solutions } from "@/app/shared/marketing/content";
import styles from "@/app/shared/marketing/PublicSite.module.css";

export function generateStaticParams() { return solutions.map((solution) => ({ solution: solution.slug })); }

export async function generateMetadata({ params }: { params: Promise<{ solution: string }> }) {
  const { solution: slug } = await params;
  const solution = solutions.find((item) => item.slug === slug);
  if (!solution) notFound();
  return { title: `${solution.name} | AtonixCorp Solutions`, description: solution.summary };
}

export default async function SolutionPage({ params }: { params: Promise<{ solution: string }> }) {
  const { solution: slug } = await params;
  const solution = solutions.find((item) => item.slug === slug);
  if (!solution) notFound();
  return <PublicShell>
    <section className={`${styles.pageHero} ${styles.darkHero}`}><div className={styles.container}><p className={styles.kicker}>SOLUTIONS / {solution.name}</p><h1>{solution.headline}</h1><p>{solution.description}</p><div className={styles.actions}><Link className={styles.primary} href="/contact">Discuss your workload <span aria-hidden="true">&rarr;</span></Link><Link className={styles.secondary} href="/solutions">All solutions</Link></div></div></section>
    <section className={styles.section}><div className={styles.container}><header className={styles.sectionHeading}><div><p className={styles.kicker}>YOUR PLANNING GUIDE</p><h2>Make the important decisions first.</h2></div></header><div className={styles.features}>{solution.priorities.map((priority, index) => <article key={priority.title}><span>0{index + 1}</span><h3>{priority.title}</h3><p>{priority.description}</p></article>)}</div><div className={styles.availability}><div><p className={styles.kicker}>SCOPE & EXPECTATIONS</p><h2>Plan against real capabilities.</h2></div><p>{solution.note}</p></div></div></section>
    <section className={`${styles.section} ${styles.softSection}`}><div className={styles.container}><header className={styles.sectionHeading}><div><p className={styles.kicker}>EXPLORE THE BUILDING BLOCKS</p><h2>Services to evaluate for this workload.</h2></div></header><ProductCards items={solution.productSlugs.flatMap((slug) => products.filter((product) => product.slug === slug))} /></div></section>
    <PublicCTA title="Let's work through your requirements." />
  </PublicShell>;
}
