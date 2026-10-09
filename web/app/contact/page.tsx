import "../styles/ContactPage.css";
import ContactForm from "../contact-form";
import TopHeader from "../sections/header/TopHeader";
import GlobalHeader from "../sections/header/GlobalHeader";
import Footer from "../sections/footer/Footer";

export default function ContactPage() {
  return (
    <main className="contact-page">
      <TopHeader />
      <GlobalHeader />

      <section className="contact-page-hero">
        <div className="contact-page-wrap">
          <div className="contact-page-intro">
            <p className="contact-page-eyebrow"><span /> ATONIXCORP / CONTACT</p>
            <h1>
              Let’s start a
              <span>conversation.</span>
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

      <Footer />
    </main>
  );
}
