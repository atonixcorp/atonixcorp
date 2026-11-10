import Link from "next/link";
import PublicShell from "../shared/marketing/PublicShell";
import { PublicCTA } from "../shared/marketing/PublicComponents";
import { solutions } from "../shared/marketing/content";
import styles from "../shared/marketing/PublicSite.module.css";

export const metadata = { title: "Solutions by Workload | AtonixCorp", description: "Explore infrastructure planning for application teams, data-intensive workloads and research." };

export default function SolutionsPage() {
  return <PublicShell><section className={styles.pageHero}><div className={styles.container}><p className={styles.kicker}>SOLUTIONS / START WITH THE WORK</p><h1>Your workload sets the direction.</h1><p>A web application, a data pipeline and a research experiment need different foundations. Start with your requirements, then choose the infrastructure that fits.</p></div></section>
    <section className={styles.section}><div className={styles.container}><div className={styles.solutionGrid}>{solutions.map((solution, index) => <Link className={styles.solutionCard} href={`/solutions/${solution.slug}`} key={solution.slug}><span className={styles.solutionNumber}>0{index + 1} / WORKLOAD GUIDE</span><h3>{solution.name}</h3><p>{solution.description}</p><span className={styles.cardLink}>Explore this approach <span aria-hidden="true">&rarr;</span></span></Link>)}</div></div></section>
    <section className={`${styles.section} ${styles.softSection}`}><div className={styles.container}><header className={styles.sectionHeading}><div><p className={styles.kicker}>A GOOD PLAN STARTS HERE</p><h2>Three questions before deployment.</h2></div></header><div className={styles.features}>{[
      ["What does it need to run?", "Runtime, memory, processing, storage access and networking requirements shape the starting architecture."],
      ["What happens when something fails?", "Recovery expectations, backup requirements and operational ownership must be agreed before going live."],
      ["Who needs access?", "Define workspace and project boundaries, authorized users and responsibility for ongoing operations."],
    ].map(([title, copy], index) => <article key={title}><span>0{index + 1}</span><h3>{title}</h3><p>{copy}</p></article>)}</div></div></section>
    <PublicCTA title="Have a workload that doesn't fit a category?" description="Share your requirements. We can work through the right next step together." />
  </PublicShell>;
}
