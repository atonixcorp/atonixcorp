# AtonixCorp Agent

The **AtonixCorp Agent** is the autonomous intelligence engine for DevOps, cloud
operations, and governance across the AtonixCorp infrastructure stack. It brings
together model management, operational signals, incident knowledge, and
infrastructure tools to support AtonixCorp's ModelOps, AIOps, and Cloud
Intelligence systems.

Its intended capabilities include self-learning, drift detection, retraining,
model promotion, incident learning, and tool-driven cloud automation.

## Contents

- [Core capabilities](#core-capabilities)
- [Architecture](#architecture)
- [Key components](#key-components)
- [ML model families](#ml-model-families)
- [Reinforcement learning](#reinforcement-learning)
- [Retrieval-augmented generation](#retrieval-augmented-generation)
- [MCP tools](#mcp-tools)
- [API layer](#api-layer)
- [Learning lifecycle](#learning-lifecycle)
- [Deployment](#deployment)
- [Roadmap](#roadmap)

## Core capabilities

The implemented foundation includes a local JSON-backed model registry, metrics
tracker, event log, signal-evaluating supervisor, dependency-free baseline
models for the four ML families below, and a local RAG document-ingestion and
retrieval pipeline. The API, RL updates, and infrastructure integrations remain
future work; these components are development foundations, not production-ready
managed training or autonomous operations.

### Unified ModelOps engine

The agent is designed to manage ML, RL, and RAG components through a shared
registry and loader:

- Model registration and versioning
- Promotion between staging and production
- Artifact management
- Performance tracking
- Drift detection and retraining

### Self-learning supervisor

The supervisor monitors model metrics and operational signals to support:

- Accuracy drift and performance-degradation detection
- Retraining and model-promotion workflows
- Event logging
- Incident learning
- RAG embedding updates and RL reward updates

### Event-driven intelligence

The event bus provides a central place to record:

- Incidents and drift alerts
- Model and retraining events
- DevOps and cloud-operations actions
- Supervisor decisions

These events are intended to inform retrieval, policy updates, and future
training cycles.

### Multi-domain AI

The agent brings together components for:

- Anomaly detection
- Autoscaling prediction
- Time-series forecasting
- Resource optimization
- DevOps reinforcement-learning agents
- RAG-based knowledge retrieval
- Reasoning and summarization skills
- Infrastructure tools exposed through MCP

## Architecture

```text
atonixcorp-agent/
├── model/
│   ├── core/           # Registry, loader, supervisor, metrics, and events
│   └── registry.json   # Model registry state
├── ml/                 # Anomaly, autoscaling, forecasting, and optimization
├── rl/                 # Reinforcement-learning agents and environments
├── rag/                # Retrieval-augmented generation pipeline
├── skills/             # Built-in reasoning skills
├── mcp/                # Kubernetes, logs, infrastructure, and file tools
├── db/                 # SQLite database, repositories, schema, and migrations
└── api/                # Agent API and middleware
```

## Key components

| Component | Responsibility |
| --- | --- |
| **Model registry** (`model/core/registry.py`) | Registers models and records their versions, metadata, stages, and promotion state. |
| **Model loader** (`model/core/loader.py`) | Loads registered ML model versions and the local RAG pipeline. RL, skill, and MCP loading are not implemented yet. |
| **Supervisor** (`model/core/supervisor.py`) | Evaluates registered models for drift and performance drops, records events, and provides explicit promotion and configured-retrainer hooks. Its monitoring loop does not run retraining or promotion automatically. |
| **Metrics tracker** (`model/core/metrics.py`) | Records model runs and metrics, and calculates improvement from baseline values. |
| **Event bus** (`model/core/events.py`) | Persists model, incident, action, and supervisor events. |

## ML model families

### Anomaly detection

`ml/anomaly/loader.py` implements an unsupervised median/MAD detector for
finite numeric features. Fit it on representative normal observations; it
returns `-1` for an outlier and `1` for an inlier.

### Autoscaling prediction

`ml/autoscale/loader.py` provides a configurable, threshold-based recommendation
policy: `0` means no change, `1` scale up, and `2` scale down. It respects
replica limits and does not execute infrastructure actions.

### Forecasting

`ml/forecasting/loader.py` fits per-feature least-squares linear trends and
forecasts future steps for numeric time series such as CPU, memory, disk,
network, and traffic usage.

### Resource optimization

`ml/resource_opt/loader.py` recommends resource requests from the observed
95th-percentile usage, a target utilization, and explicit safety headroom.

### Data, preprocessing, artifacts, and metrics

- `ml/data/csv_dataset.py` reads CSV rows and converts numeric cells.
- `ml/preprocess/numeric.py` validates selected features and rejects missing,
  non-numeric, boolean, and non-finite model inputs.
- `ml/utils/artifacts.py` saves and loads versioned JSON artifacts atomically.
  Artifacts are written under `models/<family>/`; old pickle artifacts are not
  loaded.
- `ml/utils/metrics.py` supplies accuracy, mean absolute error, and root mean
  squared error.

The models are standard-library implementations intended as transparent
starting points. They do not include automated hyperparameter selection,
distributed training, model-quality gates, or production deployment controls.
The autoscaling component produces recommendations only.

## Reinforcement learning

The DevOps PPO agent is intended to learn remediation policies using incident
and alert history, prior actions, and supervisor feedback. The RL components
include reward shaping, policy updates, and environment simulation.

## Retrieval-augmented generation

The RAG pipeline in `rag/` can ingest Markdown and plain-text documents,
split them into overlapping chunks, persist them in a local JSON vector index,
and retrieve ranked context with source citations. It can index incidents,
resolutions, runbooks, and exported logs.

`HashingEmbedder` maps tokens and adjacent token pairs into deterministic
normalized vectors without external dependencies. This captures lexical overlap
only; it is not a pretrained semantic embedding model and will not reliably
match synonyms. The pipeline assembles context for a downstream LLM but does
not call an LLM or generate answers itself. The index is intended for local,
single-process development, not multi-process or production deployments.

## MCP tools

The `mcp/` package implements a standard-library MCP server and synchronous
stdio client using newline-delimited JSON-RPC. Run the server from this
directory with:

```sh
python -m mcp.server
```

The default server advertises read-only tools:

| Tool | Purpose |
| --- | --- |
| `agent.status` | Report local state-store availability and model count. |
| `models.list` | Read registered model records. |
| `events.list` | Filter and inspect recent agent events. |
| `files.list`, `files.read` | List and read supported text files within the agent directory. |
| `rag.search` | Retrieve ranked, source-labeled context from the local RAG index. |

File access is confined to the configured workspace root; reads are limited to
supported text formats and 256 KiB per file. Hidden files and filenames
containing common secret markers are excluded. The server does not expose
write, shell, or arbitrary tool-execution operations.

Kubernetes inspection (`cluster_status`, pod listing, and deployment listing)
and infrastructure inventory (instance listing and lookup) are opt-in. A host
must pass an adapter to `build_default_server`; the adapter must implement the
documented read-only methods. No Kubernetes or infrastructure backend is
installed or configured by default, and no mutating infrastructure tools are
registered.

The stdio server writes only protocol messages to stdout. Host applications
should send diagnostics to stderr and keep the process alive for the MCP
session. This is a local development integration; authentication and
multi-tenant authorization must be provided by a remote host if one is added.

## API layer

The `api/` package exposes an HTTP/JSON API using only the Python standard
library. Start it locally from this directory with:

```sh
python -m api.server
```

By default it binds to `127.0.0.1:8000`. For authenticated remote binding,
configure a token and bind address:

```sh
AGENT_API_TOKEN='replace-with-a-long-random-token' \
AGENT_API_HOST=0.0.0.0 \
AGENT_API_PORT=8000 \
python -m api.server
```

Every endpoint except `GET /healthz` requires
`Authorization: Bearer <token>` when a token is configured. A non-loopback bind
is refused without `AGENT_API_TOKEN`. POST bodies must use JSON and are limited
to 1 MiB. Responses are JSON and are marked non-cacheable.

Implemented endpoints:

| Endpoint | Intended use |
| --- | --- |
| `GET /healthz` | Unauthenticated process health check. |
| `GET /v1/agent/registry` | List locally registered model versions. |
| `GET /v1/agent/metrics?model=<name>` | List metric runs, optionally filtered by model. |
| `GET /v1/agent/events?type=<type>&limit=<n>` | Read recent events. |
| `POST /v1/agent/predict` | Run a production-promoted ML model in advisory-only mode. |
| `POST /v1/agent/infer` | Retrieve RAG context for `{ "query": "...", "top_k": 5 }`; does not call an LLM. |
| `POST /v1/agent/rag/search` | Search the local RAG index and return cited chunks. |
| `POST /v1/agent/promote` | Explicitly promote `{ "model": "...", "version": "..." }` to production. |
| `POST /v1/agent/retrain` | Invoke a configured retrainer; returns 501 when none is configured. |

The API currently reads and writes the same local JSON-backed core stores used
by the registry, metrics tracker, event bus, and RAG pipeline. It does not
automatically switch these endpoints to SQLite. Authentication is a shared
bearer token, not per-user authorization; use a trusted network boundary or
reverse proxy for production.

`/v1/agent/predict` accepts a registered model name and live observations:

```json
{
  "model": "cluster-autoscaler",
  "observations": [
    { "cpu_percent": 82, "memory_percent": 76, "replicas": 2, "max_replicas": 5 }
  ]
}
```

Only the model version explicitly promoted to production is loaded. Forecasting
models accept a positive `horizon` instead of observations. The endpoint returns
predictions as recommendations only; it does not apply scaling, provisioning,
deployment, or other infrastructure changes. Integrating a telemetry source
must send real observations in the feature names and units expected by the
registered model. Demo values and placeholder Cloud metrics must not be used.

### Dashboard connections

The Cloud Next.js dashboard proxies `GET /v1/agent/registry` and
`POST /v1/agent/predict` through same-origin `/api/agent/...` routes. Configure
`ATONIX_AGENT_API_URL` and `ATONIX_AGENT_API_TOKEN` in its server environment;
do not use `NEXT_PUBLIC_` variables for the token.

The DevSecOps runner exposes matching same-origin proxy routes at
`/agent/registry` and `/agent/predict`. Configure the same two environment
variables on the runner process. The dashboards report model readiness, but
predictions remain unavailable until the products provide real telemetry and
production-promoted models. In particular, current DevSecOps runner data is
pipeline/job status rather than cluster metrics, and Cloud's database metrics
endpoint currently returns placeholder values.

## Learning lifecycle

1. **Detect** drift, performance drops, anomalies, or incidents.
2. **Retrain** applicable ML models or RL agents and refresh RAG embeddings.
3. **Promote** improved model versions to production.
4. **Store** events, metrics, and incident records.
5. **Improve** policies, retrieval knowledge, and supervisor thresholds using
   accumulated feedback.

## Deployment

The agent is intended to run in AtonixCorp Cloud, OrcaCloud HPC nodes, DevOps
clusters, and the governance control plane. Planned deployment characteristics
include:

- Containerized operation
- CPU and GPU inference
- Multi-tenant isolation
- Secure tool routing

The optional SQLite layer in `db/` provides versioned migrations and repositories
for models, metrics, events, incidents, action audit records, and RAG chunks.
The existing core registry, metrics tracker, event bus, and RAG vector store
still use local JSON by default; SQLite is available to consumers that opt into
`db.Database`. Both approaches are development foundations. Production
authentication, authorization, backups, and deployment controls are not
implemented.

### Docker

Build and run the API container from this directory. Set a strong token first;
the API refuses to bind to the container's non-loopback interface without it:

```sh
export AGENT_API_TOKEN='replace-with-a-long-random-token'
docker compose up --build -d
```

The Compose configuration publishes the API only on host loopback at
`http://127.0.0.1:8000`, stores runtime state in the `agent-data` named volume,
and runs the process as an unprivileged user with a read-only container
filesystem. Use `docker compose logs -f` to inspect logs and
`docker compose down` to stop the service. Removing persisted state is a
separate operation and is not required to stop the container.

For a one-off image build:

```sh
docker build -t atonixcorp-agent:local .
docker run --rm \
  -p 127.0.0.1:8000:8000 \
  -e AGENT_API_TOKEN \
  -v atonixcorp-agent-data:/data \
  atonixcorp-agent:local
```

## Development and tests

The core, baseline ML, local RAG, and SQLite modules use only the Python
standard library. From this directory, initialize the SQLite database with:

```sh
python -m db.migrate
```

To import existing JSON registry, metrics, events, and RAG-index data into
SQLite, use the explicit idempotent import option:

```sh
python -m db.migrate --import-json
```

This creates `db/atonixcorp-agent.sqlite3`; override it with
`--database /path/to/agent.sqlite3`. JSON files are left unchanged.

Use the database API directly:

```python
from db import Database

with Database() as database:
    database.register_model("forecast", "1.0", {"family": "forecasting"})
    run_id = database.start_run("forecast", "1.0")
    database.log_metric(run_id, "accuracy", 0.93)
    database.log_event("evaluation_completed", {"run_id": run_id})
```

Database migrations are checksum-verified and each migration is applied
transactionally. `Database.import_legacy_json()` can be rerun safely; existing
records are not overwritten. The SQLite layer uses foreign keys, WAL for file
databases, and nested savepoints on a single connection.

Run the test suite:

```sh
python -m unittest discover -s tests -v
```

Fit, persist, and reload an anomaly detector:

```python
from ml.anomaly.loader import AnomalyModel, load

detector = AnomalyModel().fit(
    [{"cpu_percent": value} for value in (38, 40, 41, 42, 43)]
)
detector.save("1.0.0")

detector = load("1.0.0")
print(detector.predict([{"cpu_percent": 95}]))  # [-1] means anomaly
```

Forecast a numeric series:

```python
from ml.forecasting.loader import ForecastModel

forecaster = ForecastModel().fit([10, 12, 15, 17])
print(forecaster.predict(horizon=2))
```

Index local text and retrieve cited context:

```python
from model.core.loader import AtonixLoader

rag = AtonixLoader().load_rag_pipeline()
rag.ingest_file("runbooks/incident-response.md", metadata={"kind": "runbook"})
result = rag.query("How do I respond to a node failure?", top_k=4)
print(result["context"])
print(result["sources"])
```

The default index is stored in `db/rag_index.json`; override `index_path` when
creating `RAGPipeline` to isolate an index or use a different storage location.
Use `ingest_directory(path)` to index `.md` and `.txt` files recursively.

Register an artifact for loading through the unified model loader by storing its
family in registry metadata, for example `{"family": "forecasting"}`. The
artifact version must match the registry version. Models trained and registered
by this core are not automatically promoted; promotion remains explicit.

Run one supervision cycle and inspect its signals:

```python
from model.core.supervisor import Supervisor

supervisor = Supervisor()
print(supervisor.run_once())
```

The supervisor records drift and performance signals when they first appear,
then records a clear event when a signal recovers. Retraining requires an
explicitly supplied retrainer callback. The baseline models do not provide that
callback automatically; automatic RAG updates from the supervisor, RL incident
updates, and API routes are not implemented yet.

## Roadmap

- Distributed supervisor
- Multi-agent coordination
- Cross-cluster intelligence
- Autonomous remediation
- Governance AI integration
