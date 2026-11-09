import Link from "next/link";
import styles from "../auth-unavailable/status.module.css";

export const metadata = { title: "Access denied | AtonixCorp" };

export default function ForbiddenPage() {
  return (
    <main className={styles.page}>
      <p className={styles.eyebrow}>ATONIXCORP CLOUD / ACCESS CONTROL</p>
      <h1>You do not have permission</h1>
      <p>Your session is authenticated, but this operation is not permitted for your account.</p>
      <div className={styles.actions}><Link href="/dashboard">Return to overview</Link></div>
    </main>
  );
}