"use client";

import { FormEvent, useState } from "react";

type SubmissionState = "idle" | "submitting" | "success" | "error";

export default function ContactForm() {
  const [state, setState] = useState<SubmissionState>("idle");

  async function submitContact(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setState("submitting");

    const form = event.currentTarget;
    const payload = Object.fromEntries(new FormData(form).entries());

    try {
      const response = await fetch("/api/contact", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(payload),
      });

      if (!response.ok) {
        setState("error");
        return;
      }

      form.reset();
      setState("success");
    } catch {
      setState("error");
    }
  }

  return (
    <form className="contact-form" onSubmit={submitContact}>
      <div className="contact-form-row">
        <label>
          Your name
          <input name="name" required autoComplete="name" maxLength={255} />
        </label>
        <label>
          Email address
          <input name="email" type="email" required autoComplete="email" maxLength={255} />
        </label>
      </div>
      <label>
        Subject
        <input name="subject" required maxLength={255} />
      </label>
      <label>
        Message
        <textarea name="message" required rows={6} maxLength={10000} />
      </label>
      <button type="submit" disabled={state === "submitting"}>
        {state === "submitting" ? "Sending message..." : "Send message"}
        <span aria-hidden="true">→</span>
      </button>
      {state === "success" && (
        <p className="form-status success" role="status">
          Your message was sent to AtonixCorp. We’ll be in touch.
        </p>
      )}
      {state === "error" && (
        <p className="form-status error" role="alert">
          We couldn’t send your message. Please try again or email info@atonixcorp.com directly.
        </p>
      )}
    </form>
  );
}
