"""Threshold-based autoscaling decisions with versioned JSON artifacts."""

from collections.abc import Iterable, Mapping
from pathlib import Path
from typing import Any

from ..preprocess.numeric import finite_number, mapping_rows
from ..utils.artifacts import artifact_path, load_artifact, save_artifact


class AutoscaleModel:
    """Recommend scale-up, no-change, or scale-down from live resource signals."""

    def __init__(
        self,
        scale_up_threshold: float = 80.0,
        scale_down_threshold: float = 30.0,
        queue_depth_threshold: float = 10.0,
    ):
        self.scale_up_threshold = finite_number(
            scale_up_threshold,
            "scale_up_threshold",
        )
        self.scale_down_threshold = finite_number(
            scale_down_threshold,
            "scale_down_threshold",
        )
        self.queue_depth_threshold = finite_number(
            queue_depth_threshold,
            "queue_depth_threshold",
        )
        if not 0 <= self.scale_down_threshold < self.scale_up_threshold <= 100:
            raise ValueError(
                "Thresholds must satisfy 0 <= scale_down < scale_up <= 100"
            )
        if self.queue_depth_threshold <= 0:
            raise ValueError("queue_depth_threshold must be greater than zero")

    @staticmethod
    def _integer(row: Mapping[str, Any], field: str, default: int) -> int:
        value = finite_number(row.get(field, default), field)
        if not value.is_integer():
            raise ValueError(f"{field} must be an integer")
        return int(value)

    def predict(
        self,
        rows: Mapping[str, Any] | Iterable[Mapping[str, Any]],
    ) -> list[int]:
        """Return 0=no change, 1=scale up, or 2=scale down for each row."""
        decisions: list[int] = []
        for index, row in enumerate(mapping_rows(rows)):
            signals: list[float] = []
            for field in ("cpu_percent", "memory_percent"):
                if field in row:
                    signal = finite_number(row[field], f"{field} in row {index}")
                    if not 0 <= signal <= 100:
                        raise ValueError(f"{field} must be between 0 and 100")
                    signals.append(signal)
            if not signals:
                raise ValueError(
                    f"Row {index} must include cpu_percent or memory_percent"
                )

            current = self._integer(row, "replicas", 1)
            minimum = self._integer(row, "min_replicas", 1)
            maximum = self._integer(row, "max_replicas", 10)
            if minimum < 0 or maximum < minimum or not minimum <= current <= maximum:
                raise ValueError(
                    f"Invalid replica bounds in row {index}: "
                    "require 0 <= min_replicas <= replicas <= max_replicas"
                )
            queue_depth = finite_number(
                row.get("queue_depth", 0),
                f"queue_depth in row {index}",
            )
            if queue_depth < 0:
                raise ValueError("queue_depth must be non-negative")

            scale_up = (
                max(signals) >= self.scale_up_threshold
                or queue_depth >= self.queue_depth_threshold
            )
            scale_down = all(
                signal <= self.scale_down_threshold for signal in signals
            )
            if scale_up:
                decisions.append(1 if current < maximum else 0)
            elif scale_down and current > minimum:
                decisions.append(2)
            else:
                decisions.append(0)
        return decisions

    def save(self, version: str, model_dir: str | Path | None = None) -> Path:
        return save_artifact(
            artifact_path("autoscale", version, model_dir),
            {
                "family": "autoscale",
                "version": version,
                "scale_up_threshold": self.scale_up_threshold,
                "scale_down_threshold": self.scale_down_threshold,
                "queue_depth_threshold": self.queue_depth_threshold,
            },
        )

    @classmethod
    def from_artifact(cls, payload: dict[str, Any]) -> "AutoscaleModel":
        return cls(
            scale_up_threshold=payload.get("scale_up_threshold"),
            scale_down_threshold=payload.get("scale_down_threshold"),
            queue_depth_threshold=payload.get("queue_depth_threshold"),
        )


def load(version: str, model_dir: str | Path | None = None) -> AutoscaleModel:
    """Load the autoscaling policy configuration by version."""
    path = artifact_path("autoscale", version, model_dir)
    return AutoscaleModel.from_artifact(load_artifact(path, "autoscale", version))
