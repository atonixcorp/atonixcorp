"use client";
import "../../styles/ContactSection.css";

export default function ContactSection() {
  return (
    <section className="contact-section" id="contact">
      <div className="contact-section-wrap">
        <p className="section-index">06 / Contact</p>

        <h2>Let’s build what comes next.</h2>

        <p>
          Talk with our team about infrastructure, research, partnerships, or developer programs.
        </p>

        <div className="contact-actions">
          <a className="contact-action-primary" href="/contact">
            Contact AtonixCorp <span aria-hidden="true">→</span>
          </a>
          <a href="mailto:dev@atonixcorp.com">Developer support</a>
          <a href="mailto:partners@atonixcorp.com">Partnerships</a>
        </div>
      </div>
    </section>
  );
}
