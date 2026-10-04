from __future__ import annotations

import json
import uuid
from copy import deepcopy
from datetime import datetime, timezone
from pathlib import Path
from typing import Any

from .json_store import read_json, write_json

AGENT_ROOT = Path(__file__).resolve().parents[2]
EVENTS_FILE = AGENT_ROOT / "db" / "events.json"


def _utc_now() -> str:
    return datetime.now(timezone.utc).isoformat()


class EventBus:
    """Append-only event log backed by an atomic JSON document."""

    def __init__(self, path: str | Path = EVENTS_FILE):
        self.path = Path(path)
        self.data = read_json(self.path, {"events": []})
        if not isinstance(self.data.get("events"), list):
            raise ValueError(f"Expected an 'events' list in {self.path}")

    def _save(self) -> None:
        write_json(self.path, self.data)

    def log(self, event_type: str, payload: Any) -> dict[str, Any]:
        if not isinstance(event_type, str) or not event_type.strip():
            raise ValueError("Event type must be a non-empty string")
        json.dumps(payload, allow_nan=False)
        event = {
            "id": uuid.uuid4().hex,
            "type": event_type.strip(),
            "payload": deepcopy(payload),
            "timestamp": _utc_now(),
        }
        self.data["events"].append(event)
        self._save()
        return deepcopy(event)

    def list(self, event_type: str | None = None) -> list[dict[str, Any]]:
        events = self.data["events"]
        if event_type is not None:
            events = [event for event in events if event["type"] == event_type]
        return deepcopy(events)

    def recent(
        self,
        event_type: str | None = None,
        limit: int = 10,
    ) -> list[dict[str, Any]]:
        if isinstance(limit, bool) or not isinstance(limit, int) or limit < 0:
            raise ValueError("Event limit must be a non-negative integer")
        if limit == 0:
            return []
        return self.list(event_type)[-limit:]

    def incident(self, service: str, root_cause: str, resolution: str) -> dict[str, Any]:
        return self.log(
            "incident",
            {
                "service": service,
                "root_cause": root_cause,
                "resolution": resolution,
            },
        )

    def model_promoted(self, model_name: str, version: str) -> dict[str, Any]:
        return self.log("model_promoted", {"model": model_name, "version": version})

    def model_retrained(self, model_name: str, version: str) -> dict[str, Any]:
        return self.log("model_retrained", {"model": model_name, "version": version})

    def drift_detected(self, model_name: str, drift_amount: float) -> dict[str, Any]:
        return self.log(
            "drift_detected",
            {"model": model_name, "drift": drift_amount},
        )

    def action(self, action_name: str, details: Any) -> dict[str, Any]:
        return self.log("action_executed", {"action": action_name, "details": details})

    def supervisor(self, signal: str, details: Any = None) -> dict[str, Any]:
        return self.log("supervisor_signal", {"signal": signal, "details": details})
