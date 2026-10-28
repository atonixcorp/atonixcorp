# Cloud support tickets

The authenticated Cloud support workspace is available at `/support`. Ticket
creation and history are served through the same-origin web API:

- `POST /api/support/tickets` creates a ticket.
- `GET /api/support/tickets` returns the signed-in user's 50 most recent tickets.

The web API forwards requests to the authenticated API endpoints
`POST /api/v1/support/tickets` and `GET /api/v1/support/tickets`. The API derives
the ticket owner from the verified session token; clients cannot choose a
`user_id`. Cross-origin ticket submissions are rejected, and ticket requests
are rate-limited per authenticated user.

Ticket submissions require `name`, `email`, `category`, `priority`, `subject`,
and `description`. `resource_reference` is optional. Supported categories are
`compute`, `containers`, `storage`, `databases`, `networking`, `security`,
`billing`, `account`, and `other`. Supported priorities are `low`, `normal`,
`high`, and `urgent`.

Tickets are stored in the `support_tickets` table with a generated `ATX-`
reference and initial `open` status. The API attempts to send an SMTP
notification to `SMTP_RECIPIENT`. A saved ticket is still returned if email
delivery fails; the response reports `notification_status: "not_sent"` and the
API logs the delivery error.

The current implementation supports ticket creation and account-scoped
history. Staff assignment, replies, status transitions, and resolution workflows
are not yet available.
