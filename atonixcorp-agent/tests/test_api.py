import http.client
import json
import shutil
import tempfile
import threading
import unittest
from pathlib import Path

from api.middleware import ApiError, BearerTokenAuth, require_safe_bind
from api.router import ApiRouter
from api.server import create_http_server
from model.core.events import EventBus
from model.core.metrics import MetricsTracker
from model.core.registry import ModelRegistry
from model.core.supervisor import Supervisor
from ml.autoscale.loader import AutoscaleModel
from rag.pipeline import RAGPipeline

AGENT_ROOT = Path(__file__).resolve().parents[1]


class ApiMiddlewareTests(unittest.TestCase):
    def test_bearer_auth_allows_health_and_checks_protected_routes(self):
        auth = BearerTokenAuth("secret-token")
        auth.authorize("/healthz", {})
        with self.assertRaises(ApiError) as unauthorized:
            auth.authorize("/v1/agent/registry", {})
        self.assertEqual(unauthorized.exception.status, 401)

        with self.assertRaises(ApiError):
            auth.authorize(
                "/v1/agent/registry",
                {"Authorization": "Bearer wrong-token"},
            )
        auth.authorize(
            "/v1/agent/registry",
            {"Authorization": "Bearer secret-token"},
        )

    def test_non_loopback_bind_requires_token(self):
        require_safe_bind("127.0.0.1", None)
        require_safe_bind("localhost", None)
        with self.assertRaisesRegex(ValueError, "AGENT_API_TOKEN"):
            require_safe_bind("0.0.0.0", None)
        require_safe_bind("0.0.0.0", "configured-token")
        with self.assertRaisesRegex(ValueError, "Bind host"):
            require_safe_bind("all-interfaces", "configured-token")


class ApiRouterTests(unittest.TestCase):
    def setUp(self):
        self.temporary_directory = tempfile.TemporaryDirectory()
        self.root = Path(self.temporary_directory.name)
        shutil.copytree(
            AGENT_ROOT / "ml",
            self.root / "ml",
            ignore=shutil.ignore_patterns("__pycache__"),
        )
        self.registry = ModelRegistry(self.root / "model" / "registry.json")
        self.metrics = MetricsTracker(self.root / "db" / "metrics.json")
        self.events = EventBus(self.root / "db" / "events.json")
        self.rag = RAGPipeline(self.root / "db" / "rag_index.json", min_score=0)
        self.supervisor = Supervisor(
            registry=self.registry,
            metrics=self.metrics,
            events=self.events,
        )
        self.router = ApiRouter(
            self.root,
            registry=self.registry,
            metrics=self.metrics,
            events=self.events,
            supervisor=self.supervisor,
            rag_pipeline=self.rag,
            api_token="api-secret",
        )

    def tearDown(self):
        self.temporary_directory.cleanup()

    def request(self, method, target, body=None):
        return self.router.dispatch(
            method,
            target,
            body,
            headers={"Authorization": "Bearer api-secret"},
        )

    def test_health_registry_metrics_and_events_routes(self):
        status, health = self.router.dispatch("GET", "/healthz")
        self.assertEqual(status, 200)
        self.assertEqual(health["status"], "ok")

        self.registry.register("forecast", "v1", {"family": "forecasting"})
        run_id = self.metrics.start_run("forecast", "v1")
        self.metrics.log_metric(run_id, "accuracy", 0.92)
        self.events.log("evaluation", {"run_id": run_id})

        status, registry = self.request("GET", "/v1/agent/registry")
        self.assertEqual(status, 200)
        self.assertEqual(registry["models"][0]["name"], "forecast")

        status, metrics = self.request(
            "GET",
            "/v1/agent/metrics?model=forecast",
        )
        self.assertEqual(status, 200)
        self.assertEqual(metrics["runs"][run_id]["metrics"]["accuracy"], 0.92)

        status, events = self.request(
            "GET",
            "/v1/agent/events?type=evaluation&limit=5",
        )
        self.assertEqual(status, 200)
        self.assertEqual(events["count"], 1)
        self.assertEqual(events["events"][0]["type"], "evaluation")

    def test_rag_inference_returns_retrieval_context_only(self):
        self.rag.ingest_text(
            "When a node fails, Kubernetes schedules affected workloads on healthy nodes.",
            source="recovery.md",
            document_id="recovery",
        )
        status, result = self.request(
            "POST",
            "/v1/agent/infer",
            {"query": "node fails schedule workloads", "top_k": 2},
        )
        self.assertEqual(status, 200)
        self.assertEqual(result["mode"], "retrieval_only")
        self.assertEqual(result["sources"], ["recovery.md"])
        self.assertIn("healthy nodes", result["context"])

    def test_predict_uses_production_model_and_is_advisory_only(self):
        self.registry.register("cluster-autoscaler", "v1", {"family": "autoscale"})
        AutoscaleModel().save("v1", self.root / "models")
        self.registry.promote("cluster-autoscaler", version="v1")

        status, result = self.request(
            "POST",
            "/v1/agent/predict",
            {
                "model": "cluster-autoscaler",
                "observations": [{"cpu_percent": 92, "replicas": 2, "max_replicas": 5}],
            },
        )

        self.assertEqual(status, 200)
        self.assertEqual(result["version"], "v1")
        self.assertEqual(result["family"], "autoscale")
        self.assertEqual(result["prediction"]["recommendations"], ["scale_up"])
        self.assertEqual(result["mode"], "advisory")
        self.assertEqual(self.registry.get("cluster-autoscaler")["stage"], "production")

    def test_predict_rejects_unpromoted_or_missing_models(self):
        self.registry.register("cluster-autoscaler", "v1", {"family": "autoscale"})

        with self.assertRaises(ApiError) as unpromoted:
            self.request(
                "POST",
                "/v1/agent/predict",
                {"model": "cluster-autoscaler", "observations": [{"cpu_percent": 90}]},
            )
        self.assertEqual(unpromoted.exception.status, 409)

        with self.assertRaises(ApiError) as missing:
            self.request(
                "POST",
                "/v1/agent/predict",
                {"model": "missing", "observations": [{"cpu_percent": 90}]},
            )
        self.assertEqual(missing.exception.status, 404)

    def test_predict_rejects_invalid_model_input(self):
        self.registry.register("cluster-autoscaler", "v1", {"family": "autoscale"})
        AutoscaleModel().save("v1", self.root / "models")
        self.registry.promote("cluster-autoscaler", version="v1")

        with self.assertRaises(ApiError) as invalid_input:
            self.request(
                "POST",
                "/v1/agent/predict",
                {"model": "cluster-autoscaler", "observations": [{"cpu_percent": 120}]},
            )
        self.assertEqual(invalid_input.exception.status, 422)

    def test_promote_records_event_and_retrain_without_config_returns_501(self):
        self.registry.register("forecast", "v1", {"family": "forecasting"})
        status, promoted = self.request(
            "POST",
            "/v1/agent/promote",
            {"model": "forecast", "version": "v1"},
        )
        self.assertEqual(status, 200)
        self.assertEqual(promoted["stage"], "production")
        self.assertEqual(self.events.list()[-1]["type"], "model_promoted")

        with self.assertRaises(ApiError) as unavailable:
            self.request("POST", "/v1/agent/retrain", {"model": "forecast"})
        self.assertEqual(unavailable.exception.status, 501)

    def test_retrain_uses_configured_callback(self):
        self.registry.register("forecast", "v1", {"family": "forecasting"})
        self.router.supervisor = Supervisor(
            registry=self.registry,
            metrics=self.metrics,
            events=self.events,
            retrainer=lambda _name, _entry: "v2",
        )
        status, response = self.request(
            "POST",
            "/v1/agent/retrain",
            {"model": "forecast"},
        )
        self.assertEqual(status, 202)
        self.assertEqual(response["version"], "v2")
        self.assertEqual(self.registry.get("forecast")["version"], "v2")

    def test_route_validation_unknown_paths_and_auth(self):
        with self.assertRaises(ApiError) as unauthorized:
            self.router.dispatch("GET", "/v1/agent/registry")
        self.assertEqual(unauthorized.exception.status, 401)

        with self.assertRaises(ApiError) as malformed:
            self.request("POST", "/v1/agent/infer", {"query": " "})
        self.assertEqual(malformed.exception.status, 400)

        with self.assertRaises(ApiError) as bad_limit:
            self.request("GET", "/v1/agent/events?limit=201")
        self.assertEqual(bad_limit.exception.status, 400)

        with self.assertRaises(ApiError) as missing_model:
            self.request("POST", "/v1/agent/promote", {"model": "missing"})
        self.assertEqual(missing_model.exception.status, 404)

        with self.assertRaises(ApiError) as missing_route:
            self.request("GET", "/unknown")
        self.assertEqual(missing_route.exception.status, 404)


class ApiHttpIntegrationTests(unittest.TestCase):
    def test_http_server_enforces_auth_and_content_type(self):
        with tempfile.TemporaryDirectory() as temporary_directory:
            server = create_http_server(
                host="127.0.0.1",
                port=0,
                api_token="http-secret",
                agent_root=temporary_directory,
            )
            thread = threading.Thread(target=server.serve_forever, daemon=True)
            thread.start()
            try:
                connection = http.client.HTTPConnection(
                    "127.0.0.1",
                    server.server_port,
                    timeout=3,
                )
                connection.request("GET", "/healthz")
                health_response = connection.getresponse()
                self.assertEqual(health_response.status, 200)
                self.assertEqual(json.loads(health_response.read())["status"], "ok")

                connection.request("GET", "/v1/agent/registry")
                denied = connection.getresponse()
                self.assertEqual(denied.status, 401)
                self.assertEqual(json.loads(denied.read())["error"], "A valid bearer token is required")

                connection.request(
                    "POST",
                    "/v1/agent/infer",
                    body=json.dumps({"query": "hello"}),
                    headers={
                        "Authorization": "Bearer http-secret",
                        "Content-Type": "text/plain",
                    },
                )
                unsupported = connection.getresponse()
                self.assertEqual(unsupported.status, 415)
                connection.close()
            finally:
                server.shutdown()
                server.server_close()
                thread.join(timeout=3)

    def test_http_server_rejects_unauthenticated_public_bind(self):
        with self.assertRaisesRegex(ValueError, "AGENT_API_TOKEN"):
            create_http_server(host="0.0.0.0", port=0, agent_root="/tmp")


if __name__ == "__main__":
    unittest.main()
