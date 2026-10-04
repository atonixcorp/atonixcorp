"""Minimal synchronous MCP client for newline-delimited stdio streams."""

import threading
from typing import Any, TextIO

from .protocol import JSONRPC_VERSION, MCP_PROTOCOL_VERSION, decode_message, encode_message


class MCPRemoteError(RuntimeError):
    """Remote JSON-RPC error returned by an MCP server."""

    def __init__(self, code: int, message: str, data: Any = None):
        super().__init__(message)
        self.code = code
        self.data = data


class MCPClient:
    """Request/response MCP client over already-open text reader/writer streams."""

    def __init__(self, reader: TextIO, writer: TextIO):
        self.reader = reader
        self.writer = writer
        self._next_id = 1
        self._lock = threading.Lock()
        self.server_info: dict[str, Any] | None = None
        self.protocol_version: str | None = None

    def _send(self, message: dict[str, Any]) -> None:
        self.writer.write(encode_message(message))
        self.writer.flush()

    def _request(self, method: str, params: dict[str, Any] | None = None) -> Any:
        with self._lock:
            request_id = self._next_id
            self._next_id += 1
            request: dict[str, Any] = {
                "jsonrpc": JSONRPC_VERSION,
                "id": request_id,
                "method": method,
            }
            if params is not None:
                request["params"] = params
            self._send(request)

            while True:
                line = self.reader.readline()
                if line == "":
                    raise EOFError(f"MCP server closed before replying to {method}")
                if not line.strip():
                    continue
                response = decode_message(line)
                if response.get("jsonrpc") != JSONRPC_VERSION:
                    raise RuntimeError("MCP server returned an invalid JSON-RPC version")
                if "id" not in response:
                    continue
                if response.get("id") != request_id:
                    raise RuntimeError(
                        f"MCP server response ID does not match request {request_id}"
                    )
                if "error" in response:
                    error = response["error"]
                    if not isinstance(error, dict):
                        raise RuntimeError("MCP server returned an invalid JSON-RPC error")
                    raise MCPRemoteError(
                        error.get("code", -32603),
                        error.get("message", "Unknown MCP server error"),
                        error.get("data"),
                    )
                if "result" not in response:
                    raise RuntimeError("MCP server response is missing result")
                return response["result"]

    def initialize(self) -> dict[str, Any]:
        """Initialize the MCP session and send the initialized notification."""
        result = self._request(
            "initialize",
            {
                "protocolVersion": MCP_PROTOCOL_VERSION,
                "capabilities": {},
                "clientInfo": {"name": "atonixcorp-agent-client", "version": "0.1.0"},
            },
        )
        if not isinstance(result, dict):
            raise RuntimeError("MCP initialize result must be an object")
        self.protocol_version = result.get("protocolVersion")
        server_info = result.get("serverInfo")
        self.server_info = server_info if isinstance(server_info, dict) else {}
        self._send(
            {
                "jsonrpc": JSONRPC_VERSION,
                "method": "notifications/initialized",
            }
        )
        return result

    def ping(self) -> dict[str, Any]:
        """Send the MCP ping request."""
        result = self._request("ping")
        if not isinstance(result, dict):
            raise RuntimeError("MCP ping result must be an object")
        return result

    def list_tools(self) -> list[dict[str, Any]]:
        """List tools advertised by the connected MCP server."""
        result = self._request("tools/list")
        if not isinstance(result, dict) or not isinstance(result.get("tools"), list):
            raise RuntimeError("MCP tools/list response has an invalid shape")
        return result["tools"]

    def call_tool(
        self,
        name: str,
        arguments: dict[str, Any] | None = None,
    ) -> dict[str, Any]:
        """Call one MCP tool and return its content envelope."""
        if not isinstance(name, str) or not name:
            raise ValueError("Tool name must be a non-empty string")
        if arguments is not None and not isinstance(arguments, dict):
            raise ValueError("Tool arguments must be an object")
        result = self._request(
            "tools/call",
            {"name": name, "arguments": arguments if arguments is not None else {}},
        )
        if not isinstance(result, dict) or not isinstance(result.get("content"), list):
            raise RuntimeError("MCP tools/call response has an invalid shape")
        return result
