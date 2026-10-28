import type { Metadata } from "next";
import Footer from "../sections/footer/Footer";
import GlobalHeader from "../sections/header/GlobalHeader";
import TopHeader from "../sections/header/TopHeader";
import SponsorOptions, { SponsorCallToAction, SponsorCheckoutLink } from "./SponsorOptions";
import "../styles/SponsorPage.css";

export const dynamic = "force-dynamic";

export const metadata: Metadata = {
  title: "Sponsor AtonixCorp | Build What Comes Next",
  description:
    "Support AtonixCorp's cloud infrastructure, artificial intelligence, education technology, open source, research, and community initiatives.",
  alternates: {
    canonical: "https://www.atonixcorp.com/sponsor",
  },
  openGraph: {
    title: "Sponsor AtonixCorp",
    description:
      "Join us in building enterprise-grade cloud, AI, education, and digital transformation solutions that create global impact.",
    url: "https://www.atonixcorp.com/sponsor",
    type: "website",
  },
};

function getCheckoutUrl(envName: string) {
  const configuredUrl = process.env[envName];
  if (!configuredUrl) return null;

  let parsedUrl: URL;
  try {
    parsedUrl = new URL(configuredUrl);
  } catch {
    throw new Error(`${envName} must be a valid HTTPS URL.`);
  }

  if (parsedUrl.protocol !== "https:") {
    throw new Error(`${envName} must be a valid HTTPS URL.`);
  }

  return parsedUrl.toString();
}

const paymentMethods = [
  { id: "payfast", name: "PayFast", url: getCheckoutUrl("SPONSOR_PAYFAST_URL") },
  { id: "stripe", name: "Stripe", url: getCheckoutUrl("SPONSOR_STRIPE_URL") },
  { id: "paypal", name: "PayPal", url: getCheckoutUrl("SPONSOR_PAYPAL_URL") },
];

const initiatives = [
  { number: "01", title: "Innovation", description: "Advance new ideas from early research through practical, real-world solutions." },
  { number: "02", title: "Education", description: "Expand access to technology, learning resources, and hands-on opportunities." },
  { number: "03", title: "Research", description: "Support independent exploration and responsible applied research." },
  { number: "04", title: "Technology Development", description: "Build open, dependable cloud and AI systems for organizations and developers." },
  { number: "05", title: "Community Impact", description: "Invest in programs that help local and global communities thrive." },
];

export default function SponsorPage() {
  return (
    <main className="sponsor-page">
      <TopHeader />
      <GlobalHeader />

      <section className="sponsor-hero" id="top">
        <div className="sponsor-hero-content">
          <p className="sponsor-eyebrow"><span /> AtonixCorp / Sponsorship</p>
          <h1>Sponsor AtonixCorp</h1>
          <p className="sponsor-hero-description">
            Join us in building enterprise-grade cloud, AI, education, and digital transformation solutions that create global impact.
          </p>
          <SponsorCallToAction />
          <div className="sponsor-hero-focus" aria-label="Initiative areas">
            <span>Cloud Infrastructure</span>
            <span>Artificial Intelligence</span>
            <span>Education Technology</span>
            <span>Open Source Development</span>
            <span>Research &amp; Innovation</span>
            <span>Community Programs</span>
          </div>
        </div>
        <div className="sponsor-hero-orbit" aria-hidden="true">
          <div className="sponsor-orbit-core"><span>AC</span></div>
          <span className="orbit-label orbit-label-cloud">CLOUD</span>
          <span className="orbit-label orbit-label-ai">AI</span>
          <span className="orbit-label orbit-label-research">RESEARCH</span>
          <span className="orbit-label orbit-label-community">COMMUNITY</span>
        </div>
      </section>

      <section className="sponsor-section sponsor-why" aria-labelledby="sponsor-why-title">
        <div className="sponsor-content">
          <div className="sponsor-section-heading">
            <p className="sponsor-eyebrow">Why sponsor AtonixCorp</p>
            <h2 id="sponsor-why-title">Invest in progress with lasting impact</h2>
            <p>Your support helps turn ambitious ideas into useful technologies, stronger communities, and more accessible opportunities.</p>
          </div>
          <div className="sponsor-initiative-grid">
            {initiatives.map((initiative) => (
              <article className="sponsor-initiative-card" key={initiative.number}>
                <span>{initiative.number}</span>
                <h3>{initiative.title}</h3>
                <p>{initiative.description}</p>
              </article>
            ))}
          </div>
        </div>
      </section>

      <SponsorOptions />

      <section className="sponsor-section sponsor-payment" aria-labelledby="sponsor-payment-title">
        <div className="sponsor-content sponsor-payment-content">
          <div className="sponsor-section-heading">
            <p className="sponsor-eyebrow">Payment options</p>
            <h2 id="sponsor-payment-title">A straightforward way to contribute</h2>
            <p>Choose a configured secure checkout or request bank-transfer details. We’ll confirm the contribution schedule with you.</p>
          </div>
          <div className="sponsor-payment-grid">
            {paymentMethods.map((method) => (
              <article className="sponsor-payment-card" key={method.id}>
                <div className={`sponsor-provider-mark ${method.id}`} aria-hidden="true">{method.name.slice(0, 1)}</div>
                <h3>{method.name}</h3>
                {method.url ? (
                  <>
                    <p>Continue to AtonixCorp’s secure {method.name} checkout.</p>
                    <SponsorCheckoutLink href={method.url} provider={method.id} newTab>
                      Continue to {method.name}<span aria-hidden="true">↗</span>
                    </SponsorCheckoutLink>
                  </>
                ) : (
                  <>
                    <p>Checkout link available after your sponsorship request.</p>
                    <span className="sponsor-payment-unavailable">Available on request</span>
                  </>
                )}
              </article>
            ))}
            <article className="sponsor-payment-card">
              <div className="sponsor-provider-mark bank" aria-hidden="true">B</div>
              <h3>Bank Transfer</h3>
              <p>Request bank details from our sponsorship team. We’ll share them directly and securely.</p>
              <SponsorCheckoutLink href="mailto:sponsors@atonixcorp.com?subject=Bank%20transfer%20sponsorship%20details" provider="bank-transfer">
                Request bank details <span aria-hidden="true">→</span>
              </SponsorCheckoutLink>
            </article>
          </div>
          <p className="sponsor-payment-note">
            Payment services are provided by their respective providers. AtonixCorp does not collect or store card details on this site.
          </p>
        </div>
      </section>

      <section className="sponsor-contact" aria-labelledby="sponsor-contact-title">
        <div className="sponsor-content sponsor-contact-row">
          <div>
            <p className="sponsor-eyebrow">Let’s talk</p>
            <h2 id="sponsor-contact-title">Have a different way to partner?</h2>
            <p>We welcome individuals, organizations, investors, and mission-aligned partners.</p>
          </div>
          <div className="sponsor-contact-links">
            <a href="mailto:sponsors@atonixcorp.com">sponsors@atonixcorp.com</a>
            <a href="https://www.atonixcorp.com" target="_blank" rel="noopener noreferrer">www.atonixcorp.com ↗</a>
          </div>
        </div>
      </section>

      <Footer />
    </main>
  );
}
