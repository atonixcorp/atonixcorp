"""HTTP server for the AtonixCorp Agent API using the Python standard library."""

import argparse
import json
import os
import sys
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from pathlib import Path
from typing import Any
from urllib.parse import urlsplit

from .middleware import ApiError, require_safe_bind
from .router import ApiRouter

AGENT_ROOT = Path(__file__).resolve().parents[1]
MAX_REQUEST_BYTES = 1024 * 1024


def make_handler(router: ApiRouter) -> type[BaseHTTPRequestHandler]:
    """Create an HTTP request handler bound to one API router."""

    class AgentRequestHandler(BaseHTTPRequestHandler):
        server_version = "AtonixCorpAgentAPI/0.1"
        sys_version = ""

        def log_message(self, format_string: str, *args: Any) -> None:
            sys.stderr.write(
                "%s - %s\n" % (self.address_string(), format_string % args)
            )

        def _write_json(self, status: int, payload: dict[str, Any]) -> None:
            encoded = json.dumps(
                payload,
                ensure_ascii=False,
                allow_nan=False,
                separators=(",", ":"),
            ).encode("utf-8")
            self.send_response(status)
            self.send_header("Content-Type", "application/json; charset=utf-8")
            self.send_header("Content-Length", str(len(encoded)))
            self.send_header("X-Content-Type-Options", "nosniff")
            self.send_header("Cache-Control", "no-store")
            self.end_headers()
            self.wfile.write(encoded)

        def _handle(self) -> None:
            parsed = urlsplit(self.path)
            try:
                router._authorize(parsed.path, dict(self.headers.items()))
                body: Any = None
                if self.command == "POST":
                    content_type = self.headers.get("Content-Type", "")
                    if content_type.split(";", 1)[0].strip().casefold() != "application/json":
                        raise ApiError(415, "Content-Type must be application/json")
                    raw_length = self.headers.get("Content-Length")
                    if raw_length is None:
                        raise ApiError(411, "Content-Length is required")
                    try:
                        length = int(raw_length)
                    except ValueError as error:
                        raise ApiError(400, "Content-Length must be an integer") from error
                    if length < 0:
                        raise ApiError(400, "Content-Length must not be negative")
                    if length > MAX_REQUEST_BYTES:
                        raise ApiError(413, "Request body exceeds the 1 MiB limit")
                    raw_body = self.rfile.read(length)
                    if len(raw_body) != length:
                        raise ApiError(400, "Request body ended before Content-Length")
                    try:
                        body = json.loads(
                            raw_body.decode("utf-8"),
                            parse_constant=lambda value: (_ for _ in ()).throw(
                                ValueError(f"Invalid JSON numeric constant: {value}")
                            ),
                        )
                    except (UnicodeDecodeError, json.JSONDecodeError, ValueError) as error:
                        raise ApiError(400, f"Invalid JSON request body: {error}") from error

                status, result = router.dispatch(
                    self.command,
                    self.path,
                    body=body,
                    headers=dict(self.headers.items()),
                )
                self._write_json(status, result)
            except ApiError as error:
                self._write_json(error.status, {"error": error.message})
            except (BrokenPipeError, ConnectionResetError):
                return
            except Exception as error:
                self._write_json(500, {"error": "Internal server error"})
                sys.stderr.write(
                    f"API request failed: {type(error).__name__}: {error}\n"
                )

        def do_GET(self) -> None:
            self._handle()

        def do_POST(self) -> None:
            self._handle()

        def do_HEAD(self) -> None:
            self._write_json(405, {"error": "Method not allowed"})

        def do_PUT(self) -> None:
            self._write_json(405, {"error": "Method not allowed"})

        def do_PATCH(self) -> None:
            self._write_json(405, {"error": "Method not allowed"})

        def do_DELETE(self) -> None:
            self._write_json(405, {"error": "Method not allowed"})

    return AgentRequestHandler


def create_http_server(
    host: str = "127.0.0.1",
    port: int = 8000,
    api_token: str | None = None,
    agent_root: str | Path = AGENT_ROOT,
) -> ThreadingHTTPServer:
    """Create, but do not start, the authenticated agent API HTTP server."""
    if isinstance(port, bool) or not isinstance(port, int) or not 0 <= port <= 65535:
        raise ValueError("port must be an integer between 0 and 65535")
    require_safe_bind(host, api_token)
    root = Path(agent_root)
    router = ApiRouter(root, api_token=api_token)
    server = ThreadingHTTPServer((host, port), make_handler(router))
    server.daemon_threads = True
    return server


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(description="Run the AtonixCorp Agent HTTP API.")
    parser.add_argument(
        "--host",
        default=os.environ.get("AGENT_API_HOST", "127.0.0.1"),
        help="Bind address (default: 127.0.0.1).",
    )
    parser.add_argument(
        "--port",
        type=int,
        default=int(os.environ.get("AGENT_API_PORT", "8000")),
        help="Bind port (default: 8000).",
    )
    parser.add_argument(
        "--token",
        default=os.environ.get("AGENT_API_TOKEN"),
        help="Bearer token; defaults to AGENT_API_TOKEN.",
    )
    parser.add_argument(
        "--agent-root",
        type=Path,
        default=AGENT_ROOT,
        help="Agent directory containing model/ and db/ state.",
    )
    return parser


def main(argv: list[str] | None = None) -> int:
    args = build_parser().parse_args(argv)
    require_safe_bind(args.host, args.token)
    server = create_http_server(
        host=args.host,
        port=args.port,
        api_token=args.token,
        agent_root=args.agent_root,
    )
    print(f"AtonixCorp Agent API listening on http://{args.host}:{server.server_port}", file=sys.stderr)
    try:
        server.serve_forever()
    except KeyboardInterrupt:
        pass
    finally:
        server.server_close()
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
