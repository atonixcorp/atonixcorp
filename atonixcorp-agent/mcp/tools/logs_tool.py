"""Read-only access to the agent's local structured event log."""

from pathlib import Path
from typing import Any

from model.core.events import EventBus


def list_events(
    root: Path,
    event_type: str | None = None,
    limit: int = 50,
) -> dict[str, Any]:
    """Return the most recent locally persisted events."""
    if event_type is not None and (
        not isinstance(event_type, str) or not event_type.strip()
    ):
        raise ValueError("event_type must be a non-empty string when provided")
    if isinstance(limit, bool) or not isinstance(limit, int) or not 1 <= limit <= 200:
        raise ValueError("limit must be between 1 and 200")
    events = EventBus(root / "db" / "events.json").recent(event_type, limit)
    return {"count": len(events), "events": events}


def register_event_tools(server: Any, root: Path) -> None:
    """Register structured event-log search."""
    server.register_tool(
        "events.list",
        "List recent persisted agent events, optionally filtering by exact event type.",
        {
            "type": "object",
            "properties": {
                "event_type": {"type": "string", "description": "Exact event type filter."},
                "limit": {"type": "integer", "minimum": 1, "maximum": 200},
            },
            "additionalProperties": False,
        },
        lambda arguments: list_events(
            root,
            arguments.get("event_type"),
            arguments.get("limit", 50),
        ),
    )
