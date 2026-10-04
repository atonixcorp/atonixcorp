"""Read-only model registry and agent health tools."""

import json
from pathlib import Path
from typing import Any


def _read_json(path: Path, default: dict[str, Any]) -> dict[str, Any]:
    if not path.exists():
        return default
    try:
        with path.open("r", encoding="utf-8") as handle:
            value = json.load(handle)
    except json.JSONDecodeError as error:
        raise ValueError(f"Invalid agent state JSON in {path}") from error
    if not isinstance(value, dict):
        raise ValueError(f"Expected a JSON object in {path}")
    return value


def list_models(root: Path) -> dict[str, Any]:
    """Read registered models without triggering registry migrations or writes."""
    registry = _read_json(root / "model" / "registry.json", {"models": {}})
    models = registry.get("models")
    if not isinstance(models, dict):
        raise ValueError("Model registry must contain a models object")
    return {"count": len(models), "models": list(models.values())}


def agent_status(root: Path) -> dict[str, Any]:
    """Return local agent component and persistence availability."""
    registry = list_models(root)
    return {
        "agent": "atonixcorp-agent",
        "status": "ready",
        "registered_models": registry["count"],
        "components": {
            "model_registry": (root / "model" / "registry.json").exists(),
            "event_store": (root / "db" / "events.json").exists(),
            "rag_index": (root / "db" / "rag_index.json").exists(),
        },
    }


def register_agent_tools(server: Any, root: Path) -> None:
    """Register read-only agent status and model registry tools."""
    def status_tool(arguments: dict[str, Any]) -> dict[str, Any]:
        if arguments:
            raise ValueError("agent.status does not accept arguments")
        return agent_status(root)

    def models_tool(arguments: dict[str, Any]) -> dict[str, Any]:
        if arguments:
            raise ValueError("models.list does not accept arguments")
        return list_models(root)

    server.register_tool(
        "agent.status",
        "Report local agent readiness and whether state stores exist.",
        {"type": "object", "properties": {}, "additionalProperties": False},
        status_tool,
    )
    server.register_tool(
        "models.list",
        "List locally registered ML/RL model records without changing registry state.",
        {"type": "object", "properties": {}, "additionalProperties": False},
        models_tool,
    )
