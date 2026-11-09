import Link from "next/link";
import styles from "./status.module.css";

export const metadata = { title: "Authentication unavailable | AtonixCorp" };

export default async function AuthenticationUnavailablePage({
  searchParams,
}: {
  searchParams: Promise<{ next?: string }>;
}) {
  const { next } = await searchParams;
  const destination = typeof next === "string" && next.startsWith("/") && !next.startsWith("//")
    ? next
    : "/dashboard";

  return (
    <main className={styles.page}>
      <p className={styles.eyebrow}>ATONIXCORP CLOUD / AUTHENTICATION</p>
      <h1>Authentication service unavailable</h1>
      <p>The platform could not verify your session. No protected workspace data was loaded.</p>
      <div className={styles.actions}>
        <Link href={`/login?next=${encodeURIComponent(destination)}`}>Return to sign in</Link>
        <Link href={destination}>Retry connection</Link>
      </div>
    </main>
  );
}