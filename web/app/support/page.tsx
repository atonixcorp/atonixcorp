"use client";

import { FormEvent, useCallback, useEffect, useState } from "react";
import Link from "next/link";
import { CloudShell } from "@/app/shared/ui/CloudShell";
import "./support.css";

type Ticket = {
  reference: string;
  category: string;
  priority: string;
  subject: string;
  status: string;
  created_at: string;
  updated_at: string;
};

type TicketResponse = {
  tickets?: Ticket[];
  ticket?: Ticket;
  notification_status?: "sent" | "not_sent";
  error?: string;
};

const categories = [
  ["compute", "Compute"],
  ["containers", "Containers"],
  ["storage", "Storage"],
  ["databases", "Databases"],
  ["networking", "Networking"],
  ["security", "Security"],
  ["billing", "Billing"],
  ["account", "Account access"],
  ["other", "Other"],
];

function formatTime(value: string): string {
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? "—" : date.toLocaleString();
}

async function requestTickets(): Promise<{ response: Response; payload: TicketResponse }> {
  const response = await fetch("/api/support/tickets", { cache: "no-store" });
  const payload = await response.json() as TicketResponse;
  return { response, payload };
}

export default function SupportPage() {
  const [tickets, setTickets] = useState<Ticket[]>([]);
  const [loading, setLoading] = useState(true);
  const [authenticated, setAuthenticated] = useState(true);
  const [loadError, setLoadError] = useState("");
  const [submitError, setSubmitError] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [submitted, setSubmitted] = useState<{ ticket: Ticket; notificationSent: boolean } | null>(null);

  const loadTickets = useCallback(async () => {
    setLoading(true);
    try {
      const { response, payload } = await requestTickets();
      if (response.status === 401) {
        setAuthenticated(false);
        setTickets([]);
        setLoadError("");
        return;
      }
      if (!response.ok) throw new Error(payload.error || "Your support tickets could not be loaded.");
      setAuthenticated(true);
      setTickets(payload.tickets ?? []);
      setLoadError("");
    } catch (error) {
      setLoadError(error instanceof Error ? error.message : "Your support tickets could not be loaded.");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    let active = true;
    void requestTickets().then(({ response, payload }) => {
      if (!active) return;
      if (response.status === 401) {
        setAuthenticated(false);
        setTickets([]);
        setLoadError("");
      } else if (!response.ok) {
        throw new Error(payload.error || "Your support tickets could not be loaded.");
      } else {
        setAuthenticated(true);
        setTickets(payload.tickets ?? []);
        setLoadError("");
      }
    }).catch((error: unknown) => {
      if (active) setLoadError(error instanceof Error ? error.message : "Your support tickets could not be loaded.");
    }).finally(() => {
      if (active) setLoading(false);
    });
    return () => { active = false; };
  }, []);

  async function submitTicket(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setSubmitting(true);
    setSubmitError("");
    setSubmitted(null);
    const form = event.currentTarget;
    const data = new FormData(form);
    const payload = Object.fromEntries(data.entries());

    try {
      const response = await fetch("/api/support/tickets", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(payload),
      });
      const result = await response.json() as TicketResponse;
      if (response.status === 401) {
        setAuthenticated(false);
        throw new Error("Sign in to submit and track a support ticket.");
      }
      if (!response.ok || !result.ticket) {
        throw new Error(result.error || "Your support request could not be submitted.");
      }
      setSubmitted({
        ticket: result.ticket,
        notificationSent: result.notification_status === "sent",
      });
      form.reset();
      await loadTickets();
    } catch (error) {
      setSubmitError(error instanceof Error ? error.message : "Your support request could not be submitted.");
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <CloudShell>
      <main className="support-page">
        <header className="support-heading">
          <div>
            <p className="support-eyebrow">ATONIXCORP CLOUD / SUPPORT</p>
            <h1>How can we help?</h1>
            <p>Get help with your cloud workspace, services, and account. Submitted requests are linked to your signed-in account.</p>
          </div>
          <Link className="support-contact-link" href="/contact">General contact <span aria-hidden="true">→</span></Link>
        </header>

        {!authenticated && (
          <section className="support-notice" role="status">
            <div><strong>Sign in to submit and track support tickets.</strong><p>Ticket history is private to your account.</p></div>
            <Link href="/login">Sign in <span aria-hidden="true">→</span></Link>
          </section>
        )}

        <div className="support-grid">
          <section className="support-card support-form-card">
            <header><p className="support-eyebrow">SUPPORT REQUEST</p><h2>Submit a ticket</h2><p>Include enough detail to help us understand the issue. Do not include passwords, API keys, or other secrets.</p></header>
            <form className="support-form" onSubmit={submitTicket}>
              <div className="support-form-row">
                <label>Name<input name="name" required maxLength={255} autoComplete="name" disabled={!authenticated || submitting} /></label>
                <label>Email<input name="email" type="email" required maxLength={255} autoComplete="email" disabled={!authenticated || submitting} /></label>
              </div>
              <div className="support-form-row">
                <label>Category
                  <select name="category" required defaultValue="compute" disabled={!authenticated || submitting}>
                    {categories.map(([value, label]) => <option key={value} value={value}>{label}</option>)}
                  </select>
                </label>
                <label>Priority
                  <select name="priority" required defaultValue="normal" disabled={!authenticated || submitting}>
                    <option value="low">Low</option>
                    <option value="normal">Normal</option>
                    <option value="high">High</option>
                    <option value="urgent">Urgent</option>
                  </select>
                </label>
              </div>
              <label>Subject<input name="subject" required maxLength={255} disabled={!authenticated || submitting} /></label>
              <label>Resource or instance ID <span className="support-optional">(optional)</span><input name="resource_reference" maxLength={255} placeholder="For example, an instance or volume ID" disabled={!authenticated || submitting} /></label>
              <label>What happened?<textarea name="description" rows={6} required maxLength={10000} disabled={!authenticated || submitting} placeholder="Describe what you expected, what happened, and any relevant error message." /></label>
              <button type="submit" disabled={!authenticated || submitting}>{submitting ? "Submitting ticket…" : "Submit support ticket"} <span aria-hidden="true">→</span></button>
              {submitError && <p className="support-error" role="alert">{submitError}</p>}
              {submitted && (
                <div className="support-success" role="status">
                  <strong>Ticket {submitted.ticket.reference} was created.</strong>
                  <p>Status: {submitted.ticket.status}. {submitted.notificationSent
                    ? "The support team was notified."
                    : "The ticket is saved, but email notification could not be delivered. Contact support if this is urgent."}</p>
                </div>
              )}
            </form>
          </section>

          <aside className="support-side">
            <section className="support-card support-guides">
              <header><p className="support-eyebrow">QUICK LINKS</p><h2>Platform help</h2></header>
              <Link href="/compute">Compute and instances <span aria-hidden="true">→</span></Link>
              <Link href="/containers">Containers <span aria-hidden="true">→</span></Link>
              <Link href="/storage">Storage <span aria-hidden="true">→</span></Link>
              <Link href="/cloud/databases">Databases <span aria-hidden="true">→</span></Link>
              <Link href="/networking">Networking <span aria-hidden="true">→</span></Link>
              <Link href="/observability">Observability <span aria-hidden="true">→</span></Link>
            </section>

            <section className="support-card support-ticket-card">
              <header><p className="support-eyebrow">YOUR REQUESTS</p><h2>Recent tickets</h2><p>Ticket history is private to your signed-in account.</p></header>
              {loading ? <p className="support-muted" role="status">Loading tickets…</p>
                : loadError ? <div className="support-inline-error" role="alert"><p>{loadError}</p><button type="button" onClick={() => { setLoading(true); void loadTickets(); }}>Retry</button></div>
                  : !authenticated ? <p className="support-muted">Sign in to see your tickets.</p>
                    : tickets.length ? <div className="support-ticket-list">{tickets.map((ticket) => (
                      <article key={ticket.reference}>
                        <div className="support-ticket-meta"><strong>{ticket.reference}</strong><span className={`support-priority support-priority-${ticket.priority}`}>{ticket.priority}</span></div>
                        <h3>{ticket.subject}</h3>
                        <p>{ticket.category} · Updated {formatTime(ticket.updated_at)}</p>
                        <span className="support-ticket-status">{ticket.status}</span>
                      </article>
                    ))}</div>
                      : <div className="support-empty"><strong>No support tickets yet</strong><p>Submitted requests will appear here with their reference and current status.</p></div>}
            </section>
          </aside>
        </div>

        <p className="support-footnote">For general business, partnership, or media inquiries, use the <Link href="/contact">contact page</Link>.</p>
      </main>
    </CloudShell>
  );
}
