"""Framework-free API router for agent health, models, events, and RAG."""

from __future__ import annotations

import json
from pathlib import Path
from typing import Any
from urllib.parse import parse_qs, urlsplit

from model.core.events import EventBus
from model.core.loader import AtonixLoader
from model.core.metrics import MetricsTracker
from model.core.registry import ModelRegistry
from model.core.supervisor import Supervisor
from rag.pipeline import RAGPipeline

from .middleware import ApiError, BearerTokenAuth


def _positive_limit(value: str | None, default: int, maximum: int = 200) -> int:
    if value is None:
        return default
    try:
        result = int(value)
    except ValueError as error:
        raise ApiError(400, "limit must be an integer") from error
    if not 1 <= result <= maximum:
        raise ApiError(400, f"limit must be between 1 and {maximum}")
    return result


def _required_string(body: dict[str, Any], name: str) -> str:
    value = body.get(name)
    if not isinstance(value, str) or not value.strip():
        raise ApiError(400, f"{name} must be a non-empty string")
    return value.strip()


class ApiRouter:
    """Route validated JSON requests to the implemented agent components."""

    def __init__(
        self,
        agent_root: str | Path,
        registry: ModelRegistry | None = None,
        metrics: MetricsTracker | None = None,
        events: EventBus | None = None,
        supervisor: Supervisor | None = None,
        rag_pipeline: RAGPipeline | None = None,
        api_token: str | None = None,
    ):
        root = Path(agent_root)
        self.registry = registry or ModelRegistry(root / "model" / "registry.json")
        self.model_loader = AtonixLoader(root, registry=self.registry)
        self.metrics = metrics or MetricsTracker(root / "db" / "metrics.json")
        self.events = events or EventBus(root / "db" / "events.json")
        self.supervisor = supervisor or Supervisor(
            registry=self.registry,
            metrics=self.metrics,
            events=self.events,
        )
        self.rag_pipeline = rag_pipeline or RAGPipeline(
            index_path=root / "db" / "rag_index.json"
        )
        self.auth = BearerTokenAuth(api_token)

    def _authorize(self, path: str, headers: dict[str, str] | None) -> None:
        normalized = {key.title(): value for key, value in (headers or {}).items()}
        self.auth.authorize(path, normalized)

    def dispatch(
        self,
        method: str,
        target: str,
        body: Any = None,
        headers: dict[str, str] | None = None,
    ) -> tuple[int, dict[str, Any]]:
        """Dispatch a method/path/body tuple to an API response."""
        if not isinstance(method, str) or not isinstance(target, str):
            raise ApiError(400, "Method and request target must be strings")
        parsed = urlsplit(target)
        path = parsed.path
        query = parse_qs(parsed.query, keep_blank_values=True)
        self._authorize(path, headers)

        if method == "GET" and path == "/healthz":
            return 200, {"status": "ok", "service": "atonixcorp-agent"}
        if method == "GET" and path == "/v1/agent/registry":
            return 200, {"models": self.registry.list()}
        if method == "GET" and path == "/v1/agent/metrics":
            model_names = query.get("model", [])
            if len(model_names) > 1:
                raise ApiError(400, "model may only be specified once")
            runs = self.metrics.list_runs(model_names[0] if model_names else None)
            return 200, {"runs": runs}
        if method == "GET" and path == "/v1/agent/events":
            event_types = query.get("type", [])
            if len(event_types) > 1:
                raise ApiError(400, "type may only be specified once")
            limit = _positive_limit(query.get("limit", [None])[0], 100)
            events = self.events.recent(
                event_types[0] if event_types and event_types[0] else None,
                limit,
            )
            return 200, {"count": len(events), "events": events}
        if method == "POST" and path == "/v1/agent/infer":
            request = self._object_body(body)
            query_text = _required_string(request, "query")
            top_k = self._body_limit(request.get("top_k", 5), 20)
            result = self.rag_pipeline.query(query_text, top_k=top_k)
            return 200, {
                "query": result["query"],
                "context": result["context"],
                "sources": result["sources"],
                "results": result["results"],
                "mode": "retrieval_only",
            }
        if method == "POST" and path == "/v1/agent/predict":
            request = self._object_body(body)
            model_name = _required_string(request, "model")
            model_record = next(
                (
                    item
                    for item in self.registry.list()
                    if item.get("name") == model_name
                ),
                None,
            )
            if model_record is None:
                raise ApiError(404, f"Model {model_name} not found")
            version = model_record.get("production_version")
            if not isinstance(version, str) or not version:
                raise ApiError(409, f"Model {model_name} has no production version")
            entry = self.registry.get(model_name, version)
            if entry is None:
                raise ApiError(409, f"Production version for {model_name} is unavailable")

            family = entry.get("metadata", {}).get("family")
            if family not in {"anomaly", "autoscale", "forecasting", "resource_opt"}:
                raise ApiError(422, f"Model {model_name} has an unsupported ML family")
            try:
                model = self.model_loader.load_ml_model(model_name, version)
                if family == "forecasting":
                    horizon = self._body_limit(request.get("horizon", 1), 100)
                    prediction = model.predict(horizon)
                else:
                    if "observations" not in request:
                        raise ApiError(400, "observations are required for this model")
                    observations = request["observations"]
                    if family == "anomaly":
                        prediction = {
                            "labels": model.predict(observations),
                            "scores": model.score_samples(observations),
                        }
                    elif family == "autoscale":
                        decisions = model.predict(observations)
                        prediction = {
                            "decisions": decisions,
                            "recommendations": [
                                {0: "hold", 1: "scale_up", 2: "scale_down"}[item]
                                for item in decisions
                            ],
                        }
                    else:
                        prediction = model.predict(observations)
            except ApiError:
                raise
            except ValueError as error:
                raise ApiError(422, str(error)) from error
            try:
                json.dumps(prediction, allow_nan=False)
            except (TypeError, ValueError) as error:
                raise ApiError(422, f"Model produced an invalid prediction: {error}") from error
            return 200, {
                "model": model_name,
                "version": version,
                "family": family,
                "prediction": prediction,
                "mode": "advisory",
            }
        if method == "POST" and path == "/v1/agent/retrain":
            request = self._object_body(body)
            model_name = _required_string(request, "model")

            def retrain_model() -> str:
                return self.supervisor.retrain(model_name)

            try:
                new_version = retrain_model()
            except RuntimeError as error:
                raise ApiError(501, str(error)) from error
            except ValueError as error:
                raise ApiError(404, str(error)) from error
            return 202, {"model": model_name, "version": new_version, "status": "retrained"}
        if method == "POST" and path == "/v1/agent/promote":
            request = self._object_body(body)
            model_name = _required_string(request, "model")
            version = request.get("version")
            if version is not None and (
                not isinstance(version, str) or not version.strip()
            ):
                raise ApiError(400, "version must be a non-empty string")
            try:
                entry = self.registry.promote(
                    model_name,
                    stage="production",
                    version=version.strip() if version else None,
                )
            except ValueError as error:
                raise ApiError(404, str(error)) from error
            self.events.model_promoted(model_name, entry["version"])
            return 200, {"model": model_name, "version": entry["version"], "stage": "production"}
        if method == "POST" and path == "/v1/agent/rag/search":
            request = self._object_body(body)
            result = self.rag_pipeline.query(
                _required_string(request, "query"),
                top_k=self._body_limit(request.get("top_k", 5), 20),
            )
            return 200, result
        raise ApiError(404, f"Route not found: {method} {path}")

    @staticmethod
    def _object_body(body: Any) -> dict[str, Any]:
        if not isinstance(body, dict):
            raise ApiError(400, "Request body must be a JSON object")
        return body

    @staticmethod
    def _body_limit(value: Any, maximum: int) -> int:
        if isinstance(value, bool) or not isinstance(value, int):
            raise ApiError(400, "top_k must be an integer")
        if not 1 <= value <= maximum:
            raise ApiError(400, f"top_k must be between 1 and {maximum}")
        return value
