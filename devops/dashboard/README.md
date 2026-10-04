# AtonixCorp DevSecOps Dashboard

This is a standalone frontend application. Its root route (`/`) opens the DevSecOps
dashboard directly; there is no marketing or product landing page.

## Run the dashboard

```sh
cd devops/dashboard
npm install
npm run dev
```

Open `http://localhost:3002`. The dashboard starts in interactive demo mode with
sample data. Demo pipeline runs are stored in the current page session and do not
execute jobs or create cloud resources.

Use the theme button in the header to switch between light and dark mode. The
selected theme is saved in this browser. The AtonixCorp logo is used for the
dashboard's browser-tab icon.

## Connect the DevOps runner

In another terminal, run the runner service from the Go module:

```sh
cd devops
ORCASTACK_RUNNER_WORKSPACE="$PWD" go run ./cmd/orcastack-runner
```

The runner listens on port `8086` by default. In the dashboard, select
**Connect runner** to load pipeline/job data from its API. Set
`VITE_RUNNER_API_URL` in `devops/dashboard/.env.local` if the runner uses a
different URL. The runner currently accepts the `api-test` workflow, which runs
`go test ./...` in `ORCASTACK_RUNNER_WORKSPACE`; only start this workflow when
you intend to execute those tests.

## Open it from Cloud

The Cloud dashboard’s **DevSecOps** navigation entry opens this standalone app.
For local development it points to `http://localhost:3002`. Set
`NEXT_PUBLIC_DEVSECOPS_DASHBOARD_URL` when building the Cloud frontend to point to
the deployed dashboard URL.

Dashboard metrics and platform/service-health panels are illustrative until
connected to their respective live services. Runner connection status and pipeline
runs are live when the runner API is connected.
