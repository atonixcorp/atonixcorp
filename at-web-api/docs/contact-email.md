# Contact email delivery

Website inquiries submitted at `/contact` are forwarded to the API at
`/api/v1/contact/submit`. The API saves each inquiry and sends an SMTP notification
to the configured `SMTP_RECIPIENT`.

Set `SMTP_RECIPIENT=info@atonixcorp.com` in the API's runtime environment. SMTP
credentials must be provided through `SMTP_HOST`, `SMTP_PORT`, `SMTP_USER`, and
`SMTP_PASSWORD`; keep those credentials out of source control.

The website reports success only when the API confirms delivery. Configure
`API_URL` for the Next.js application to point to the reachable API service.
