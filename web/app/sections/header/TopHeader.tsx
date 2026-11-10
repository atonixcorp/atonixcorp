import Link from "next/link";
import "../../styles/TopHeader.css";

export default function TopHeader() {
  return <div className="top-header"><div className="top-header-wrap">
    <Link className="top-announcement" href="/platform"><span>BUILD WITH ATONIXCORP</span> A connected foundation for your next idea <b aria-hidden="true">&rarr;</b></Link>
    <nav aria-label="Utility navigation"><a href="https://github.com/atonixcorp/atonixcorp">GitHub</a><Link href="/support">Support</Link><Link href="/contact">Contact us</Link></nav>
  </div></div>;
}
