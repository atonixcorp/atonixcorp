"use client";

import { FormEvent, useEffect, useState } from "react";
import { trackSponsorEvent } from "../shared/lib/sponsor-analytics";

const packages = [
  {
    id: "bronze",
    name: "Bronze Sponsor",
    amount: 25,
    recognition: "Community recognition",
    benefits: ["Name listed on the sponsor page", "Quarterly impact updates"],
  },
  {
    id: "silver",
    name: "Silver Sponsor",
    amount: 100,
    recognition: "Featured recognition",
    benefits: ["All Bronze benefits", "Featured sponsor listing", "Early project briefings"],
  },
  {
    id: "gold",
    name: "Gold Sponsor",
    amount: 500,
    recognition: "Premium recognition",
    benefits: ["All Silver benefits", "Recognition in an annual impact report", "Priority partner briefings"],
  },
  {
    id: "platinum",
    name: "Platinum Sponsor",
    amount: 1000,
    recognition: "Strategic partner recognition",
    benefits: ["All Gold benefits", "Strategic collaboration opportunities", "Dedicated partner contact"],
  },
];

type SubmissionState = "idle" | "submitting" | "success" | "error";

export function SponsorCallToAction() {
  return (
    <a
      className="sponsor-primary-button"
      href="#sponsor-packages"
      onClick={() => trackSponsorEvent("sponsor_cta_click")}
    >
      Become a Sponsor <span aria-hidden="true">↗</span>
    </a>
  );
}

export function SponsorCheckoutLink({
  href,
  provider,
  children,
  newTab = false,
}: {
  href: string;
  provider: string;
  children: React.ReactNode;
  newTab?: boolean;
}) {
  return (
    <a
      href={href}
      target={newTab ? "_blank" : undefined}
      rel={newTab ? "noopener noreferrer" : undefined}
      onClick={() => trackSponsorEvent("sponsor_payment_click", { payment_provider: provider })}
    >
      {children}
    </a>
  );
}

export default function SponsorOptions() {
  const [selectedPackage, setSelectedPackage] = useState("");
  const [amount, setAmount] = useState("");
  const [state, setState] = useState<SubmissionState>("idle");

  useEffect(() => {
    trackSponsorEvent("sponsor_page_view");
  }, []);

  function choosePackage(packageId: string) {
    const sponsorshipPackage = packages.find((item) => item.id === packageId);
    if (!sponsorshipPackage) return;

    setSelectedPackage(packageId);
    setAmount(String(sponsorshipPackage.amount));
    trackSponsorEvent("sponsor_package_select", { package: packageId });
    document.getElementById("sponsor-form")?.scrollIntoView({ behavior: "smooth" });
  }

  function updatePackage(packageId: string) {
    setSelectedPackage(packageId);
    const sponsorshipPackage = packages.find((item) => item.id === packageId);
    setAmount(sponsorshipPackage ? String(sponsorshipPackage.amount) : "");
  }

  async function submitRequest(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setState("submitting");

    const form = event.currentTarget;
    const formData = new FormData(form);
    const packageId = String(formData.get("package") ?? "");
    const paymentProvider = String(formData.get("paymentProvider") ?? "not-sure");

    trackSponsorEvent("sponsor_form_submit", {
      package: packageId || "custom",
      payment_provider: paymentProvider,
    });

    try {
      const response = await fetch("/api/sponsor", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          name: formData.get("name"),
          company: formData.get("company"),
          email: formData.get("email"),
          phone: formData.get("phone"),
          country: formData.get("country"),
          amount: Number(formData.get("amount")),
          package: packageId,
          paymentProvider,
          message: formData.get("message"),
          website: formData.get("website"),
        }),
      });

      if (!response.ok) {
        setState("error");
        trackSponsorEvent("sponsor_form_error");
        return;
      }

      form.reset();
      setSelectedPackage("");
      setAmount("");
      setState("success");
      trackSponsorEvent("sponsor_form_success", {
        package: packageId || "custom",
        payment_provider: paymentProvider,
      });
    } catch {
      setState("error");
      trackSponsorEvent("sponsor_form_error");
    }
  }

  return (
    <>
      <section className="sponsor-section sponsor-packages" id="sponsor-packages" aria-labelledby="sponsor-packages-title">
        <div className="sponsor-content">
          <div className="sponsor-section-heading">
            <p className="sponsor-eyebrow">Sponsorship packages</p>
            <h2 id="sponsor-packages-title">Choose how you want to make an impact</h2>
            <p>Monthly USD contributions are suggested starting levels. We’ll confirm recognition and benefits with you directly.</p>
          </div>

          <div className="sponsor-package-grid">
            {packages.map((sponsorshipPackage) => (
              <article className="sponsor-package-card" key={sponsorshipPackage.id}>
                <p className="sponsor-package-level">{sponsorshipPackage.recognition}</p>
                <h3>{sponsorshipPackage.name}</h3>
                <p className="sponsor-package-price">
                  ${sponsorshipPackage.amount.toLocaleString("en-US")}
                  <span> / month</span>
                </p>
                <ul>
                  {sponsorshipPackage.benefits.map((benefit) => (
                    <li key={benefit}><span aria-hidden="true">✓</span>{benefit}</li>
                  ))}
                </ul>
                <button type="button" onClick={() => choosePackage(sponsorshipPackage.id)}>
                  Select {sponsorshipPackage.name.replace(" Sponsor", "")}
                  <span aria-hidden="true">→</span>
                </button>
              </article>
            ))}
          </div>
        </div>
      </section>

      <section className="sponsor-section sponsor-form-section" id="sponsor-form" aria-labelledby="sponsor-form-title">
        <div className="sponsor-form-layout sponsor-content">
          <div className="sponsor-form-intro">
            <p className="sponsor-eyebrow">Custom sponsorship</p>
            <h2 id="sponsor-form-title">Let’s build something meaningful together.</h2>
            <p>Tell us about your organization and the level of support you have in mind. Our team will follow up to discuss next steps.</p>
            <a href="mailto:sponsors@atonixcorp.com">sponsors@atonixcorp.com</a>
          </div>

          <form className="sponsor-form" onSubmit={submitRequest}>
            <div className="sponsor-field-grid">
              <label>
                Full Name <span aria-hidden="true">*</span>
                <input name="name" required autoComplete="name" maxLength={255} />
              </label>
              <label>
                Company Name
                <input name="company" autoComplete="organization" maxLength={255} />
              </label>
              <label>
                Email Address <span aria-hidden="true">*</span>
                <input name="email" type="email" required autoComplete="email" maxLength={255} />
              </label>
              <label>
                Phone Number
                <input name="phone" type="tel" autoComplete="tel" maxLength={50} />
              </label>
              <label>
                Country <span aria-hidden="true">*</span>
                <input name="country" required autoComplete="country-name" maxLength={100} />
              </label>
              <label>
                Sponsorship Package
                <select name="package" value={selectedPackage} onChange={(event) => updatePackage(event.target.value)}>
                  <option value="">Custom sponsorship</option>
                  {packages.map((item) => (
                    <option value={item.id} key={item.id}>{item.name}</option>
                  ))}
                </select>
              </label>
              <label className="sponsor-amount-field">
                Sponsorship Amount (USD / month) <span aria-hidden="true">*</span>
                <input
                  name="amount"
                  type="number"
                  inputMode="decimal"
                  min="1"
                  max="10000000"
                  step="0.01"
                  required
                  value={amount}
                  onChange={(event) => setAmount(event.target.value)}
                />
              </label>
              <label className="sponsor-payment-field">
                Preferred Payment Method
                <select name="paymentProvider" defaultValue="not-sure">
                  <option value="not-sure">I’d like to discuss options</option>
                  <option value="payfast">PayFast</option>
                  <option value="stripe">Stripe</option>
                  <option value="paypal">PayPal</option>
                  <option value="bank-transfer">Bank Transfer</option>
                </select>
              </label>
            </div>
            <label>
              Message
              <textarea name="message" rows={5} maxLength={10000} placeholder="Tell us about the initiatives you’d like to support." />
            </label>
            <div className="sponsor-honeypot" aria-hidden="true">
              <label htmlFor="sponsor-website">Leave this field empty</label>
              <input id="sponsor-website" name="website" tabIndex={-1} autoComplete="off" />
            </div>
            <button className="sponsor-submit-button" type="submit" disabled={state === "submitting"}>
              {state === "submitting" ? "Submitting request..." : "Submit Sponsorship Request"}
              <span aria-hidden="true">→</span>
            </button>
            {state === "success" && (
              <p className="sponsor-form-status success" role="status">
                Thank you. Your sponsorship request has been received, and a confirmation email is on its way.
              </p>
            )}
            {state === "error" && (
              <p className="sponsor-form-status error" role="alert">
                We couldn’t submit your request. Please try again or email sponsors@atonixcorp.com.
              </p>
            )}
          </form>
        </div>
      </section>
    </>
  );
}
