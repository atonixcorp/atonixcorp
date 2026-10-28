# Sponsorship requests

The public sponsor form posts to the Next.js endpoint `/api/sponsor`. The site
validates the submitted fields, rejects the honeypot field, and rate-limits
submissions to five per IP address per ten minutes per running web process. It
then forwards valid submissions to the public API endpoint
`/api/v1/contact/sponsorship`.
The deployment proxy should overwrite `x-real-ip` or `x-forwarded-for` with a
trusted client address for the per-IP limit to be effective.

The API validates the request again, stores it as a contact message, sends an
admin notification to `SPONSOR_RECIPIENT`, and sends a confirmation to the
sponsor's email address. `SPONSOR_RECIPIENT` defaults to
`sponsors@atonixcorp.com`. The API still requires the existing SMTP settings
(`SMTP_HOST`, `SMTP_PORT`, `SMTP_USER`, `SMTP_PASSWORD`, and `SMTP_RECIPIENT`)
for mail delivery. Configure `API_URL` in the web app to reach the API.

Hosted payment links are optional and are configured in the web app using
`SPONSOR_PAYFAST_URL`, `SPONSOR_STRIPE_URL`, and `SPONSOR_PAYPAL_URL`. Each
configured value must be an HTTPS URL for an AtonixCorp payment-provider
checkout. Bank-transfer details are shared directly by the sponsorship team;
the site does not collect or store payment card details.

The sponsor page emits `sponsor_page_view`, `sponsor_cta_click`,
`sponsor_package_select`, `sponsor_payment_click`, `sponsor_form_submit`,
`sponsor_form_success`, and `sponsor_form_error` analytics events. Events are
sent to `dataLayer` or `gtag` when available, and are also dispatched as the
`atonix:analytics` browser event. No contact details or sponsorship amount are
included in analytics events.
