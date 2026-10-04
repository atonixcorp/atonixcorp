import math
import threading
from collections.abc import Callable
from typing import Any

from .events import EventBus
from .metrics import MetricsTracker
from .registry import ModelRegistry


class Supervisor:
    """Evaluate registered models and record signals without unsafe auto-actions."""

    def __init__(
        self,
        registry: ModelRegistry | None = None,
        metrics: MetricsTracker | None = None,
        events: EventBus | None = None,
        retrainer: Callable[[str, dict[str, Any]], str] | None = None,
        drift_threshold: float = 0.15,
        performance_drop_threshold: float = 0.20,
        auto_promote_threshold: float = 0.10,
    ):
        self.registry = registry or ModelRegistry()
        self.metrics = metrics or MetricsTracker()
        self.events = events or EventBus()
        self.retrainer = retrainer
        self.drift_threshold = self._validate_threshold("drift_threshold", drift_threshold)
        self.performance_drop_threshold = self._validate_threshold(
            "performance_drop_threshold",
            performance_drop_threshold,
        )
        self.auto_promote_threshold = self._validate_threshold(
            "auto_promote_threshold",
            auto_promote_threshold,
        )
        self._signal_state: dict[str, tuple[bool, bool]] = {}

    @staticmethod
    def _validate_threshold(name: str, value: float) -> float:
        if isinstance(value, bool) or not isinstance(value, (int, float)):
            raise ValueError(f"{name} must be a finite non-negative number")
        if not Supervisor._is_finite(value) or value < 0:
            raise ValueError(f"{name} must be a finite non-negative number")
        return float(value)

    @staticmethod
    def _is_finite(value: int | float) -> bool:
        try:
            return math.isfinite(value)
        except OverflowError:
            return False

    def _accuracy_metrics(self, model_name: str) -> tuple[float, float] | None:
        entry = self.registry.get(model_name)
        if entry is None:
            return None
        metrics = self.metrics.get_recent(model_name, entry["version"])
        if not metrics:
            return None
        baseline = metrics.get("baseline_accuracy")
        current = metrics.get("current_accuracy")
        if baseline is None or current is None:
            return None
        for name, value in (("baseline_accuracy", baseline), ("current_accuracy", current)):
            if isinstance(value, bool) or not isinstance(value, (int, float)):
                raise ValueError(f"{name} for model {model_name} must be numeric")
            if not self._is_finite(value):
                raise ValueError(f"{name} for model {model_name} must be finite")
        return float(baseline), float(current)

    def detect_drift(self, model_name: str) -> bool:
        values = self._accuracy_metrics(model_name)
        if values is None:
            return False
        baseline, current = values
        return baseline - current > self.drift_threshold

    def performance_drop(self, model_name: str) -> bool:
        values = self._accuracy_metrics(model_name)
        if values is None:
            return False
        baseline, current = values
        return baseline - current > self.performance_drop_threshold

    def evaluate_model(self, model_name: str) -> dict[str, Any]:
        values = self._accuracy_metrics(model_name)
        result = {"model": model_name, "drift": False, "performance_drop": False}
        if values is None:
            result["status"] = "insufficient_metrics"
            return result

        baseline, current = values
        difference = baseline - current
        result["drift"] = difference > self.drift_threshold
        result["performance_drop"] = difference > self.performance_drop_threshold
        result["status"] = "signals_detected" if any(
            (result["drift"], result["performance_drop"])
        ) else "healthy"

        previous_drift, previous_drop = self._signal_state.get(model_name, (False, False))
        if result["drift"] and not previous_drift:
            self.events.drift_detected(model_name, difference)
        elif previous_drift and not result["drift"]:
            self.events.supervisor("drift_cleared", {"model": model_name})
        if result["performance_drop"] and not previous_drop:
            self.events.supervisor(
                "performance_drop",
                {
                    "model": model_name,
                    "baseline_accuracy": baseline,
                    "current_accuracy": current,
                    "drop": difference,
                },
            )
        elif previous_drop and not result["performance_drop"]:
            self.events.supervisor("performance_drop_cleared", {"model": model_name})
        self._signal_state[model_name] = (result["drift"], result["performance_drop"])
        return result

    def run_once(self) -> list[dict[str, Any]]:
        return [self.evaluate_model(model["name"]) for model in self.registry.list()]

    def retrain(self, model_name: str) -> str:
        if self.retrainer is None:
            raise RuntimeError(
                "No retrainer is configured; automatic model training is not available."
            )
        entry = self.registry.get(model_name)
        if entry is None:
            raise ValueError(f"Model {model_name} not found")

        self.events.log("retrain_started", {"model": model_name, "version": entry["version"]})
        new_version = self.retrainer(model_name, entry)
        if not isinstance(new_version, str) or not new_version.strip():
            raise ValueError("Configured retrainer must return a non-empty version string")
        new_version = new_version.strip()
        self.registry.register(model_name, new_version, entry.get("metadata", {}))
        self.events.model_retrained(model_name, new_version)
        return new_version

    def auto_promote(self, model_name: str) -> bool:
        entry = self.registry.get(model_name)
        if entry is None:
            raise ValueError(f"Model {model_name} not found")
        improvement = self.metrics.compute_improvement(model_name, entry["version"])
        if improvement <= self.auto_promote_threshold:
            return False
        self.registry.promote(model_name, stage="production", version=entry["version"])
        self.events.model_promoted(model_name, entry["version"])
        return True

    def learn_from_incident(self, incident: dict[str, Any]) -> dict[str, Any]:
        if not isinstance(incident, dict):
            raise ValueError("Incident must be a dictionary")
        return self.events.log("incident", incident)

    def run(
        self,
        interval_seconds: float = 10,
        stop_event: threading.Event | None = None,
    ) -> None:
        if isinstance(interval_seconds, bool) or not isinstance(interval_seconds, (int, float)):
            raise ValueError("interval_seconds must be a finite positive number")
        if not self._is_finite(interval_seconds) or interval_seconds <= 0:
            raise ValueError("interval_seconds must be a finite positive number")

        stop_event = stop_event or threading.Event()
        while not stop_event.is_set():
            self.run_once()
            if stop_event.wait(interval_seconds):
                break
