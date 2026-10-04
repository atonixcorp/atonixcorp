"""Opt-in, read-only infrastructure inventory MCP adapter tools."""

from typing import Any


def _backend_method(backend: Any, name: str, **arguments: Any) -> Any:
    method = getattr(backend, name, None)
    if not callable(method):
        raise NotImplementedError(
            f"Infrastructure backend must provide a read-only {name}() method"
        )
    return method(**arguments)


def register_infrastructure_tools(server: Any, backend: Any) -> None:
    """Register instance inventory tools when a host injects an adapter.

    The backend contract is ``list_instances(filters=None)`` and
    ``get_instance(instance_id)``. This adapter does not create, stop, or modify
    infrastructure.
    """
    server.register_tool(
        "infrastructure.list_instances",
        "List infrastructure instances using an optional backend-supported filter object.",
        {
            "type": "object",
            "properties": {"filters": {"type": "object"}},
            "additionalProperties": False,
        },
        lambda arguments: _backend_method(
            backend,
            "list_instances",
            filters=arguments.get("filters"),
        ),
    )
    server.register_tool(
        "infrastructure.get_instance",
        "Read details for one infrastructure instance by ID.",
        {
            "type": "object",
            "properties": {"instance_id": {"type": "string", "minLength": 1}},
            "required": ["instance_id"],
            "additionalProperties": False,
        },
        lambda arguments: _backend_method(
            backend,
            "get_instance",
            instance_id=arguments["instance_id"],
        ),
    )
