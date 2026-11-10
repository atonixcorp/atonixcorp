import "../styles/ContactPage.css";
import ContactForm from "../contact-form";
import Link from "next/link";
import PublicShell from "../shared/marketing/PublicShell";
import styles from "../shared/marketing/PublicSite.module.css";

export const metadata = { title: "Contact Our Team | AtonixCorp", description: "Talk to AtonixCorp about your cloud workload, platform requirements, research or partnership." };

export default function ContactPage() {
  return (
    <PublicShell><div className="contact-page">

      <section className="contact-page-hero">
        <div className="contact-page-wrap">
          <div className="contact-page-intro">
            <p className="contact-page-eyebrow"><span /> ATONIXCORP / CONTACT</p>
            <h1>
              Let&apos;s talk about
              <span>what you&apos;re building.</span>
            </h1>
            <p className="contact-page-description">
              Tell us what you’re working on. Our team can help with platform
              questions, research, partnerships, and other inquiries.
            </p>

            <div className="contact-page-recipient">
              <span className="contact-page-recipient-icon" aria-hidden="true">@</span>
              <p>
                <small>YOUR MESSAGE WILL BE SENT TO</small>
                <a href="mailto:info@atonixcorp.com">info@atonixcorp.com</a>
              </p>
            </div>
            <p className="contact-page-privacy">
              Your details will only be used to respond to your inquiry.
            </p>
            <div className={styles.contactRoutes}>
              <Link href="/support">Need help with your cloud account? <span aria-hidden="true">&rarr;</span></Link>
              <Link href="/sponsor">Interested in sponsorship? <span aria-hidden="true">&rarr;</span></Link>
            </div>
          </div>

          <div className="contact-page-card">
            <div className="contact-page-card-heading">
              <div>
                <p>GET IN TOUCH</p>
                <h2>Send us a message</h2>
              </div>
              <span aria-hidden="true">↗</span>
            </div>
            <ContactForm />
          </div>
        </div>
      </section>

    </div></PublicShell>
  );
}
