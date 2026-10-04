"""JSON-RPC 2.0 message helpers and MCP protocol constants."""

import json
from typing import Any, TextIO

JSONRPC_VERSION = "2.0"
MCP_PROTOCOL_VERSION = "2024-11-05"


class JsonRpcError(Exception):
    """JSON-RPC error with a protocol-defined integer code."""

    def __init__(self, code: int, message: str, data: Any = None):
        super().__init__(message)
        self.code = code
        self.message = message
        self.data = data

    def to_response(self, request_id: Any) -> dict[str, Any]:
        error: dict[str, Any] = {"code": self.code, "message": self.message}
        if self.data is not None:
            error["data"] = self.data
        return {"jsonrpc": JSONRPC_VERSION, "id": request_id, "error": error}


def encode_message(message: dict[str, Any]) -> str:
    """Encode one JSON-RPC message as the MCP stdio newline frame."""
    if not isinstance(message, dict):
        raise ValueError("JSON-RPC message must be an object")
    return json.dumps(message, ensure_ascii=False, allow_nan=False, separators=(",", ":")) + "\n"


def decode_message(line: str) -> dict[str, Any]:
    """Decode and validate the JSON object carried by one stdio frame."""
    def reject_non_finite(value: str) -> None:
        raise ValueError(f"Invalid JSON-RPC JSON constant: {value}")

    try:
        message = json.loads(line, parse_constant=reject_non_finite)
    except json.JSONDecodeError as error:
        raise ValueError(f"Invalid JSON-RPC JSON: {error.msg}") from error
    if not isinstance(message, dict):
        raise ValueError("JSON-RPC message must be an object")
    return message


def write_message(writer: TextIO, message: dict[str, Any]) -> None:
    """Write and flush one newline-delimited JSON-RPC message."""
    writer.write(encode_message(message))
    writer.flush()


def read_message(reader: TextIO) -> dict[str, Any] | None:
    """Read the next stdio frame, returning None at clean end-of-stream."""
    line = reader.readline()
    if line == "":
        return None
    if not line.strip():
        raise ValueError("Empty JSON-RPC frame")
    return decode_message(line)
