import "./containers.css";
import Link from "next/link";
import { CloudShell } from "@/app/shared/ui/CloudShell";

export default function ContainersLayout({ children }: Readonly<{ children: React.ReactNode }>) {
  return (
    <CloudShell>
      <div className="container-console">
        <nav className="container-section-nav" aria-label="Container tools">
          <Link href="/containers">Deployments</Link>
          <Link href="/containers/registries">Registries</Link>
        </nav>
        <main className="container-main">{children}</main>
      </div>
    </CloudShell>
  );
}
