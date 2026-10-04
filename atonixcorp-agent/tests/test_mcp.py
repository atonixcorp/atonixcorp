import io
import json
import tempfile
import unittest
from pathlib import Path

from mcp.client import MCPClient, MCPRemoteError
from mcp.protocol import decode_message, encode_message
from mcp.server import MCPServer, build_default_server
from mcp.tools.infra_tool import register_infrastructure_tools
from mcp.tools.k8s_tool import register_kubernetes_tools


class ProtocolTests(unittest.TestCase):
    def test_stdio_frame_round_trip_is_one_json_line(self):
        message = {"jsonrpc": "2.0", "id": 2, "method": "ping"}
        encoded = encode_message(message)
        self.assertTrue(encoded.endswith("\n"))
        self.assertEqual(decode_message(encoded), message)

        with self.assertRaisesRegex(ValueError, "Invalid JSON"):
            decode_message("{bad}\n")
        with self.assertRaisesRegex(ValueError, "constant"):
            decode_message('{"jsonrpc":"2.0","id":1,"params":{"x":NaN}}')
        with self.assertRaisesRegex(ValueError, "must be an object"):
            decode_message("[]")


class MCPServerTests(unittest.TestCase):
    def setUp(self):
        self.temporary_directory = tempfile.TemporaryDirectory()
        self.root = Path(self.temporary_directory.name)
        self.server = build_default_server(self.root)

    def tearDown(self):
        self.temporary_directory.cleanup()

    @staticmethod
    def request(method, params=None, request_id=1):
        message = {"jsonrpc": "2.0", "id": request_id, "method": method}
        if params is not None:
            message["params"] = params
        return message

    def test_initialize_tools_list_and_ping(self):
        initialized = self.server.dispatch(
            self.request(
                "initialize",
                {
                    "protocolVersion": "2024-11-05",
                    "capabilities": {},
                    "clientInfo": {"name": "test", "version": "1"},
                },
            )
        )
        self.assertEqual(initialized["result"]["protocolVersion"], "2024-11-05")
        self.assertIn("tools", initialized["result"]["capabilities"])

        tools = self.server.dispatch(self.request("tools/list"))
        names = {tool["name"] for tool in tools["result"]["tools"]}
        self.assertTrue(
            {
                "agent.status",
                "models.list",
                "events.list",
                "files.list",
                "files.read",
                "rag.search",
            }.issubset(names)
        )
        self.assertFalse(any(name.startswith("kubernetes.") for name in names))
        self.assertFalse(any(name.startswith("infrastructure.") for name in names))
        self.assertTrue(
            all(tool["annotations"]["readOnlyHint"] for tool in tools["result"]["tools"])
        )
        self.assertEqual(self.server.dispatch(self.request("ping"))["result"], {})

    def test_tools_call_returns_json_and_errors_without_server_crash(self):
        status = self.server.dispatch(
            self.request("tools/call", {"name": "agent.status", "arguments": {}})
        )
        self.assertFalse(status["result"]["isError"])
        self.assertEqual(
            json.loads(status["result"]["content"][0]["text"])["registered_models"],
            0,
        )

        bad_arguments = self.server.dispatch(
            self.request(
                "tools/call",
                {"name": "files.read", "arguments": {"path": "../outside.txt"}},
            )
        )
        self.assertTrue(bad_arguments["result"]["isError"])
        self.assertIn("outside", bad_arguments["result"]["content"][0]["text"])

        invalid_schema = self.server.dispatch(
            self.request(
                "tools/call",
                {"name": "events.list", "arguments": {"limit": "many"}},
            )
        )
        self.assertTrue(invalid_schema["result"]["isError"])
        self.assertIn("type integer", invalid_schema["result"]["content"][0]["text"])

        unknown = self.server.dispatch(
            self.request("tools/call", {"name": "missing", "arguments": {}})
        )
        self.assertTrue(unknown["result"]["isError"])

        self.server.register_tool(
            "test.number",
            "Test finite-number validation.",
            {
                "type": "object",
                "properties": {"value": {"type": "number"}},
                "required": ["value"],
                "additionalProperties": False,
            },
            lambda arguments: arguments["value"],
        )
        non_finite = self.server.dispatch(
            self.request(
                "tools/call",
                {"name": "test.number", "arguments": {"value": float("nan")}},
            )
        )
        self.assertTrue(non_finite["result"]["isError"])

    def test_file_tools_are_workspace_confined_and_filter_sensitive_files(self):
        (self.root / "notes.md").write_text("safe text", encoding="utf-8")
        (self.root / "access_token.json").write_text("sensitive", encoding="utf-8")
        (self.root / ".env").write_text("secret", encoding="utf-8")
        outside = self.root.parent / f"{self.root.name}-outside.txt"
        outside.write_text("outside", encoding="utf-8")
        try:
            listed = self.server.dispatch(
                self.request(
                    "tools/call",
                    {"name": "files.list", "arguments": {}},
                )
            )
            entries = json.loads(listed["result"]["content"][0]["text"])["entries"]
            names = {entry["name"] for entry in entries}
            self.assertIn("notes.md", names)
            self.assertNotIn("access_token.json", names)
            self.assertNotIn(".env", names)

            outside_read = self.server.dispatch(
                self.request(
                    "tools/call",
                    {
                        "name": "files.read",
                        "arguments": {"path": f"../{outside.name}"},
                    },
                )
            )
            self.assertTrue(outside_read["result"]["isError"])
            read = self.server.dispatch(
                self.request(
                    "tools/call",
                    {"name": "files.read", "arguments": {"path": "notes.md"}},
                )
            )
            self.assertEqual(
                json.loads(read["result"]["content"][0]["text"])["text"],
                "safe text",
            )
        finally:
            outside.unlink()

    def test_read_only_event_and_rag_tools(self):
        from model.core.events import EventBus
        from rag.pipeline import RAGPipeline

        EventBus(self.root / "db" / "events.json").incident(
            "scheduler",
            "node unavailable",
            "rescheduled workload",
        )
        RAGPipeline(
            self.root / "db" / "rag_index.json",
            min_score=0,
        ).ingest_text(
            "The scheduler reschedules workloads when a node becomes unavailable.",
            source="runbook.md",
            document_id="scheduler-guide",
        )

        events = self.server.dispatch(
            self.request(
                "tools/call",
                {"name": "events.list", "arguments": {"event_type": "incident"}},
            )
        )
        event_payload = json.loads(events["result"]["content"][0]["text"])
        self.assertEqual(event_payload["count"], 1)
        self.assertEqual(event_payload["events"][0]["payload"]["service"], "scheduler")

        rag = self.server.dispatch(
            self.request(
                "tools/call",
                {
                    "name": "rag.search",
                    "arguments": {"query": "scheduler node reschedule"},
                },
            )
        )
        rag_payload = json.loads(rag["result"]["content"][0]["text"])
        self.assertEqual(rag_payload["sources"], ["runbook.md"])

    def test_json_rpc_errors_and_notifications(self):
        invalid = self.server.dispatch({"jsonrpc": "1.0", "id": 1, "method": "ping"})
        self.assertEqual(invalid["error"]["code"], -32600)
        unknown_method = self.server.dispatch(self.request("unknown/method"))
        self.assertEqual(unknown_method["error"]["code"], -32601)
        self.assertIsNone(
            self.server.dispatch(
                {"jsonrpc": "2.0", "method": "notifications/initialized"}
            )
        )

        reader = io.StringIO(
            encode_message(self.request("ping", request_id=8))
            + encode_message(self.request("tools/list", request_id=9))
        )
        writer = io.StringIO()
        self.server.serve(reader, writer)
        responses = [json.loads(line) for line in writer.getvalue().splitlines()]
        self.assertEqual([response["id"] for response in responses], [8, 9])


class OptionalAdapterTests(unittest.TestCase):
    def test_kubernetes_tools_only_appear_with_injected_read_only_backend(self):
        class Backend:
            def cluster_status(self):
                return {"connected": True}

            def list_pods(self, namespace=None):
                return [{"name": "api", "namespace": namespace}]

            def list_deployments(self, namespace=None):
                return [{"name": "api", "namespace": namespace}]

        server = MCPServer()
        register_kubernetes_tools(server, Backend())
        tools = server.dispatch(
            {"jsonrpc": "2.0", "id": 1, "method": "tools/list"}
        )["result"]["tools"]
        self.assertEqual(len(tools), 3)
        result = server.dispatch(
            {
                "jsonrpc": "2.0",
                "id": 2,
                "method": "tools/call",
                "params": {
                    "name": "kubernetes.list_pods",
                    "arguments": {"namespace": "operations"},
                },
            }
        )
        self.assertEqual(
            json.loads(result["result"]["content"][0]["text"])[0]["namespace"],
            "operations",
        )

    def test_infrastructure_tools_require_injected_backend_and_are_read_only(self):
        class Backend:
            def list_instances(self, filters=None):
                return {"filters": filters}

            def get_instance(self, instance_id):
                return {"id": instance_id}

        server = MCPServer()
        register_infrastructure_tools(server, Backend())
        tools = server.dispatch(
            {"jsonrpc": "2.0", "id": 1, "method": "tools/list"}
        )["result"]["tools"]
        self.assertEqual(len(tools), 2)
        self.assertTrue(all(tool["annotations"]["readOnlyHint"] for tool in tools))


class MCPClientTests(unittest.TestCase):
    def test_client_initialize_lists_tools_and_calls_tool(self):
        responses = [
            {
                "jsonrpc": "2.0",
                "id": 1,
                "result": {
                    "protocolVersion": "2024-11-05",
                    "serverInfo": {"name": "test"},
                    "capabilities": {"tools": {}},
                },
            },
            {"jsonrpc": "2.0", "id": 2, "result": {"tools": [{"name": "agent.status"}]}},
            {
                "jsonrpc": "2.0",
                "id": 3,
                "result": {"content": [{"type": "text", "text": "{}"}], "isError": False},
            },
        ]
        reader = io.StringIO("".join(encode_message(item) for item in responses))
        writer = io.StringIO()
        client = MCPClient(reader, writer)

        self.assertEqual(client.initialize()["protocolVersion"], "2024-11-05")
        self.assertEqual(client.server_info["name"], "test")
        self.assertEqual(client.list_tools()[0]["name"], "agent.status")
        self.assertFalse(client.call_tool("agent.status")["isError"])
        sent = [json.loads(line) for line in writer.getvalue().splitlines()]
        self.assertEqual(
            [message.get("method") for message in sent],
            ["initialize", "notifications/initialized", "tools/list", "tools/call"],
        )

    def test_client_raises_for_remote_errors(self):
        reader = io.StringIO(
            encode_message(
                {
                    "jsonrpc": "2.0",
                    "id": 1,
                    "error": {"code": -32601, "message": "missing method"},
                }
            )
        )
        client = MCPClient(reader, io.StringIO())
        with self.assertRaises(MCPRemoteError):
            client.ping()


if __name__ == "__main__":
    unittest.main()
