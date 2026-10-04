"""Standard-library MCP server using newline-delimited JSON-RPC over stdio."""

import json
import math
import sys
from dataclasses import dataclass
from pathlib import Path
from typing import Any, Callable, TextIO

from .protocol import (
    JSONRPC_VERSION,
    MCP_PROTOCOL_VERSION,
    JsonRpcError,
    decode_message,
    encode_message,
)
from .tools.file_tool import register_file_tools
from .tools.agent_tool import register_agent_tools
from .tools.infra_tool import register_infrastructure_tools
from .tools.k8s_tool import register_kubernetes_tools
from .tools.logs_tool import register_event_tools
from .tools.rag_tool import register_rag_tools

AGENT_ROOT = Path(__file__).resolve().parents[1]
ToolHandler = Callable[[dict[str, Any]], Any]


@dataclass(frozen=True)
class Tool:
    name: str
    description: str
    input_schema: dict[str, Any]
    handler: ToolHandler

    def definition(self) -> dict[str, Any]:
        return {
            "name": self.name,
            "description": self.description,
            "inputSchema": self.input_schema,
            "annotations": {
                "readOnlyHint": True,
                "destructiveHint": False,
                "idempotentHint": True,
                "openWorldHint": False,
            },
        }


class MCPServer:
    """A small MCP tools server with strict JSON-RPC request validation."""

    def __init__(self, name: str = "atonixcorp-agent", version: str = "0.1.0"):
        self.name = name
        self.version = version
        self._tools: dict[str, Tool] = {}

    def register_tool(
        self,
        name: str,
        description: str,
        input_schema: dict[str, Any],
        handler: ToolHandler,
    ) -> None:
        if not isinstance(name, str) or not name.strip():
            raise ValueError("Tool name must be a non-empty string")
        if name in self._tools:
            raise ValueError(f"Tool {name!r} is already registered")
        if not isinstance(description, str) or not description.strip():
            raise ValueError("Tool description must be a non-empty string")
        if not isinstance(input_schema, dict) or input_schema.get("type") != "object":
            raise ValueError("Tool input schema must be a JSON object schema")
        if not callable(handler):
            raise ValueError("Tool handler must be callable")
        json.dumps(input_schema, allow_nan=False)
        self._tools[name] = Tool(name, description, input_schema, handler)

    def _call_tool(self, params: Any) -> dict[str, Any]:
        if not isinstance(params, dict):
            raise JsonRpcError(-32602, "tools/call params must be an object")
        name = params.get("name")
        arguments = params.get("arguments", {})
        if not isinstance(name, str) or not name:
            raise JsonRpcError(-32602, "tools/call requires a non-empty tool name")
        if not isinstance(arguments, dict):
            raise JsonRpcError(-32602, "Tool arguments must be an object")
        tool = self._tools.get(name)
        if tool is None:
            return {
                "content": [{"type": "text", "text": f"Unknown tool: {name}"}],
                "isError": True,
            }
        try:
            self._validate_arguments(tool.input_schema, arguments)
            result = tool.handler(arguments)
            text = json.dumps(result, ensure_ascii=False, allow_nan=False, indent=2)
        except Exception as error:
            text = f"{type(error).__name__}: {error}"
            return {"content": [{"type": "text", "text": text}], "isError": True}
        return {"content": [{"type": "text", "text": text}], "isError": False}

    @staticmethod
    def _validate_arguments(schema: dict[str, Any], arguments: dict[str, Any]) -> None:
        properties = schema.get("properties", {})
        required = schema.get("required", [])
        missing = [name for name in required if name not in arguments]
        if missing:
            raise ValueError(f"Missing required argument(s): {', '.join(missing)}")
        if schema.get("additionalProperties") is False:
            extra = sorted(set(arguments) - set(properties))
            if extra:
                raise ValueError(f"Unknown argument(s): {', '.join(extra)}")
        types = {
            "object": dict,
            "string": str,
            "array": list,
            "boolean": bool,
        }
        for name, value in arguments.items():
            specification = properties.get(name)
            if not isinstance(specification, dict):
                continue
            expected_type = specification.get("type")
            if expected_type == "integer":
                valid = isinstance(value, int) and not isinstance(value, bool)
            elif expected_type == "number":
                valid = (
                    isinstance(value, (int, float))
                    and not isinstance(value, bool)
                    and math.isfinite(value)
                )
            else:
                valid_type = types.get(expected_type)
                valid = valid_type is None or isinstance(value, valid_type)
            if not valid:
                raise ValueError(f"Argument {name!r} must have type {expected_type}")
            if isinstance(value, str) and "minLength" in specification:
                if len(value) < specification["minLength"]:
                    raise ValueError(
                        f"Argument {name!r} must contain at least "
                        f"{specification['minLength']} characters"
                    )
            if "enum" in specification and value not in specification["enum"]:
                raise ValueError(
                    f"Argument {name!r} must be one of: "
                    f"{', '.join(map(str, specification['enum']))}"
                )
            if isinstance(value, (int, float)) and not isinstance(value, bool):
                if "minimum" in specification and value < specification["minimum"]:
                    raise ValueError(
                        f"Argument {name!r} must be at least {specification['minimum']}"
                    )
                if "maximum" in specification and value > specification["maximum"]:
                    raise ValueError(
                        f"Argument {name!r} must be at most {specification['maximum']}"
                    )

    def dispatch(self, message: dict[str, Any]) -> dict[str, Any] | None:
        """Dispatch one JSON-RPC request or notification."""
        if not isinstance(message, dict) or message.get("jsonrpc") != JSONRPC_VERSION:
            return JsonRpcError(-32600, "Invalid JSON-RPC request").to_response(
                message.get("id") if isinstance(message, dict) else None
            )
        method = message.get("method")
        if not isinstance(method, str):
            return JsonRpcError(-32600, "Request method must be a string").to_response(
                message.get("id")
            )

        is_notification = "id" not in message
        request_id = message.get("id")
        if not is_notification and (
            isinstance(request_id, bool)
            or (
                request_id is not None
                and not isinstance(request_id, (int, str))
            )
        ):
            return JsonRpcError(-32600, "Request id must be a string or integer").to_response(
                request_id
            )
        params = message.get("params", {})

        if method == "notifications/initialized":
            return None
        if is_notification:
            return None

        try:
            if method == "initialize":
                if not isinstance(params, dict):
                    raise JsonRpcError(-32602, "initialize params must be an object")
                requested_version = params.get("protocolVersion", MCP_PROTOCOL_VERSION)
                if not isinstance(requested_version, str):
                    raise JsonRpcError(-32602, "protocolVersion must be a string")
                result = {
                    "protocolVersion": (
                        requested_version
                        if requested_version == MCP_PROTOCOL_VERSION
                        else MCP_PROTOCOL_VERSION
                    ),
                    "capabilities": {"tools": {"listChanged": False}},
                    "serverInfo": {"name": self.name, "version": self.version},
                    "instructions": (
                        "This server exposes read-only agent and local workspace tools. "
                        "Kubernetes and infrastructure adapters are disabled unless "
                        "explicitly configured by the host."
                    ),
                }
            elif method == "ping":
                result = {}
            elif method == "tools/list":
                if not isinstance(params, dict):
                    raise JsonRpcError(-32602, "tools/list params must be an object")
                result = {"tools": [tool.definition() for tool in self._tools.values()]}
            elif method == "tools/call":
                result = self._call_tool(params)
            else:
                raise JsonRpcError(-32601, f"Method not found: {method}")
        except JsonRpcError as error:
            return error.to_response(request_id)

        return {"jsonrpc": JSONRPC_VERSION, "id": request_id, "result": result}

    def serve(self, reader: TextIO, writer: TextIO) -> None:
        """Serve requests until input EOF; protocol diagnostics stay off stdout."""
        while True:
            line = reader.readline()
            if line == "":
                return
            try:
                if not line.strip():
                    raise ValueError("Empty JSON-RPC frame")
                message = decode_message(line)
                response = self.dispatch(message)
            except ValueError as error:
                response = JsonRpcError(-32700, str(error)).to_response(None)
            if response is not None:
                writer.write(encode_message(response))
                writer.flush()


def build_default_server(
    agent_root: str | Path = AGENT_ROOT,
    file_root: str | Path | None = None,
    kubernetes_backend: Any = None,
    infrastructure_backend: Any = None,
) -> MCPServer:
    """Create the safe default server and optionally attach read-only adapters."""
    root = Path(agent_root).resolve()
    workspace = Path(file_root).resolve() if file_root is not None else root
    server = MCPServer()
    register_file_tools(server, workspace)
    register_event_tools(server, root)
    register_agent_tools(server, root)
    register_rag_tools(server, root)
    if kubernetes_backend is not None:
        register_kubernetes_tools(server, kubernetes_backend)
    if infrastructure_backend is not None:
        register_infrastructure_tools(server, infrastructure_backend)
    return server


def main() -> None:
    """Run the default stdio MCP server."""
    build_default_server().serve(sys.stdin, sys.stdout)


if __name__ == "__main__":
    main()
