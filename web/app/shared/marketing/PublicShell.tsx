import type { ReactNode } from "react";
import TopHeader from "@/app/sections/header/TopHeader";
import GlobalHeader from "@/app/sections/header/GlobalHeader";
import Footer from "@/app/sections/footer/Footer";
import styles from "./PublicSite.module.css";

export default function PublicShell({ children }: { children: ReactNode }) {
  return <div className={styles.site} id="top">
    <a className="skip-link" href="#public-content">Skip to main content</a>
    <TopHeader />
    <GlobalHeader />
    <main id="public-content" tabIndex={-1}>{children}</main>
    <Footer />
  </div>;
}
