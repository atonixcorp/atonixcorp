import Image from "next/image";
import Link from "next/link";
import { products } from "@/app/shared/marketing/content";
import "../../styles/Footer.css";

export default function Footer() {
  return <footer className="public-footer">
    <div className="public-footer-wrap">
      <div className="public-footer-top">
        <div className="public-footer-brand"><Link href="/" aria-label="AtonixCorp home"><Image src="/atonixcorp-icon.png" alt="" width={36} height={36} /><strong>AtonixCorp.</strong></Link><p>Infrastructure for your next idea.<br />Engineering for what comes next.</p><div><a href="https://github.com/atonixcorp/atonixcorp">GitHub</a><a href="https://linkedin.com/company/atonixcorp">LinkedIn</a><a href="https://discord.gg/EGqJ778a55">Discord</a></div></div>
        <nav aria-label="Product links"><h2>Products</h2>{products.map((product) => <Link key={product.slug} href={`/products/${product.slug}`}>{product.name}</Link>)}</nav>
        <nav aria-label="Resource links"><h2>Resources</h2><Link href="/developers">Developer hub</Link><Link href="/platform">Platform architecture</Link><Link href="/developers/documentation">Documentation</Link><Link href="/developers/api-cli">API & CLI resources</Link><Link href="/developers/community">Community</Link><Link href="/solutions">Solutions</Link><Link href="/support">Support center</Link></nav>
        <nav aria-label="Company links"><h2>Company</h2><Link href="/about">About AtonixCorp</Link><Link href="/about/roadmap">Platform roadmap</Link><Link href="/sponsor">Sponsorship</Link><Link href="/contact">Contact us</Link><Link href="/portal">Account portal</Link></nav>
      </div>
      <div className="public-footer-bottom"><span>&copy; {new Date().getFullYear()} AtonixCorp. All rights reserved.</span><div><Link href="/contact">Privacy inquiries</Link><Link href="/contact">Accessibility feedback</Link><a href="#top">Back to top <span aria-hidden="true">&uarr;</span></a></div></div>
    </div>
  </footer>;
}
