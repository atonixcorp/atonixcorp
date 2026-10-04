import math
import uuid
from copy import deepcopy
from datetime import datetime, timezone
from pathlib import Path
from typing import Any

from .json_store import read_json, write_json

AGENT_ROOT = Path(__file__).resolve().parents[2]
METRICS_FILE = AGENT_ROOT / "db" / "metrics.json"


def _is_finite(value: int | float) -> bool:
    try:
        return math.isfinite(value)
    except OverflowError:
        return False


def _utc_now() -> str:
    return datetime.now(timezone.utc).isoformat()


class MetricsTracker:
    """Persist model runs and numeric metrics in an atomic JSON document."""

    def __init__(self, path: str | Path = METRICS_FILE):
        self.path = Path(path)
        self.data = read_json(self.path, {"runs": {}})
        if not isinstance(self.data.get("runs"), dict):
            raise ValueError(f"Expected a 'runs' object in {self.path}")

    def _save(self) -> None:
        write_json(self.path, self.data)

    def start_run(self, model_name: str, version: str) -> str:
        if not isinstance(model_name, str) or not model_name.strip():
            raise ValueError("Model name must be a non-empty string")
        if not isinstance(version, str) or not version.strip():
            raise ValueError("Model version must be a non-empty string")

        run_id = f"{model_name.strip()}-{version.strip()}-{uuid.uuid4().hex}"
        now = _utc_now()
        self.data["runs"][run_id] = {
            "model": model_name.strip(),
            "version": version.strip(),
            "metrics": {},
            "created_at": now,
            "updated_at": now,
        }
        self._save()
        return run_id

    def log_metric(self, run_id: str, key: str, value: int | float) -> None:
        run = self.data["runs"].get(run_id)
        if run is None:
            raise ValueError(f"Run {run_id} not found")
        if not isinstance(key, str) or not key.strip():
            raise ValueError("Metric key must be a non-empty string")
        if isinstance(value, bool) or not isinstance(value, (int, float)):
            raise ValueError("Metric value must be a finite number")
        if not _is_finite(value):
            raise ValueError("Metric value must be a finite number")

        run["metrics"][key.strip()] = value
        run["updated_at"] = _utc_now()
        self._save()

    def get_recent(
        self,
        model_name: str,
        version: str | None = None,
    ) -> dict[str, Any] | None:
        runs = [
            run for run in self.data["runs"].values()
            if run["model"] == model_name and (version is None or run["version"] == version)
        ]
        if not runs:
            return None
        latest = max(
            runs,
            key=lambda run: run.get("updated_at") or run["created_at"],
        )
        return deepcopy(latest["metrics"])

    def compute_improvement(self, model_name: str, version: str | None = None) -> float:
        metrics = self.get_recent(model_name, version)
        if not metrics:
            return 0.0
        baseline = metrics.get("baseline_accuracy", 1.0)
        current = metrics.get("current_accuracy", 1.0)
        if not isinstance(baseline, (int, float)) or isinstance(baseline, bool):
            raise ValueError("baseline_accuracy must be numeric")
        if not isinstance(current, (int, float)) or isinstance(current, bool):
            raise ValueError("current_accuracy must be numeric")
        if not _is_finite(baseline) or not _is_finite(current):
            raise ValueError("accuracy metrics must be finite")
        return current - baseline

    def list_runs(self, model_name: str | None = None) -> dict[str, dict[str, Any]]:
        runs = self.data["runs"]
        if model_name is not None:
            runs = {
                run_id: run for run_id, run in runs.items()
                if run["model"] == model_name
            }
        return deepcopy(runs)
