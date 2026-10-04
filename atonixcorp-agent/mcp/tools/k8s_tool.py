"""Opt-in, read-only Kubernetes MCP adapter tools."""

from typing import Any


def _backend_method(backend: Any, name: str, **arguments: Any) -> Any:
    method = getattr(backend, name, None)
    if not callable(method):
        raise NotImplementedError(
            f"Kubernetes backend must provide a read-only {name}() method"
        )
    return method(**arguments)


def register_kubernetes_tools(server: Any, backend: Any) -> None:
    """Register read-only cluster inspection if a host injects a backend.

    The adapter must provide ``cluster_status()``, ``list_pods(namespace=None)``,
    and ``list_deployments(namespace=None)``. This module never performs
    Kubernetes writes.
    """
    def cluster_status(arguments: dict[str, Any]) -> Any:
        if arguments:
            raise ValueError("kubernetes.cluster_status does not accept arguments")
        return _backend_method(backend, "cluster_status")

    server.register_tool(
        "kubernetes.cluster_status",
        "Inspect cluster connectivity and read-only cluster status.",
        {"type": "object", "properties": {}, "additionalProperties": False},
        cluster_status,
    )
    for resource in ("pods", "deployments"):
        server.register_tool(
            f"kubernetes.list_{resource}",
            f"List Kubernetes {resource} in an optional namespace.",
            {
                "type": "object",
                "properties": {
                    "namespace": {
                        "type": "string",
                        "description": "Namespace to inspect; backend default if omitted.",
                    }
                },
                "additionalProperties": False,
            },
            lambda arguments, resource_name=resource: _backend_method(
                backend,
                f"list_{resource_name}",
                namespace=arguments.get("namespace"),
            ),
        )
