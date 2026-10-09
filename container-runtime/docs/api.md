markdown
# AtonixCorp Container Runtime API
## Overview
This document defines the REST API endpoints exposed by the container runtime control plane. These endpoints are consumed by the AtonixCorp dashboard, CLI, and automation systems.

Base URL:
/api/v1/container-runtime


---

## Build Endpoints

### POST `/build`
Triggers a container build from uploaded source code.

**Request:**
```json
{
  "project_id": "my-app",
  "source_path": "/tmp/upload/my-app",
  "runtime": "python"
}
Response:

json
{
  "image_id": "registry.local/my-app:latest",
  "logs": ["build started", "build completed"]
}
Run Endpoints
POST /run
Starts a container using a built image.

Request:

json
{
  "image_id": "registry.local/my-app:latest",
  "cpu": 1,
  "memory": 512
}
Response:

json
{
  "container_id": "ctr-my-app-12345",
  "node": "node-1"
}
Lifecycle Endpoints
POST /stop
json
{
  "container_id": "ctr-my-app-12345"
}
POST /restart
json
{
  "container_id": "ctr-my-app-12345"
}
DELETE /delete
json
{
  "container_id": "ctr-my-app-12345"
}
Logs Endpoint
GET /logs/{container_id}
Streams real‑time logs.

Response (SSE or WebSocket):

Code
2026-10-09T12:30:00Z log line 1
2026-10-09T12:30:01Z log line 2
Metrics Endpoint
GET /metrics/{container_id}
Returns container metrics.

## Response:

json
{
  "cpu_percent": 0.5,
  "memory_mb": 128,
  "network_in": 10,
  "network_out": 5,
  "timestamp": "2026-10-09T12:30:00Z"
}
Scheduler Endpoint
GET /nodes
Returns available nodes.