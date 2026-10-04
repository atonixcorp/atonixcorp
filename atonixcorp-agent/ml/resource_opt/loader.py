"""Conservative resource sizing from observed workload usage."""

import math
from collections.abc import Iterable, Mapping
from pathlib import Path
from typing import Any

from ..preprocess.numeric import finite_number, numeric_rows
from ..utils.artifacts import artifact_path, load_artifact, save_artifact


class ResourceOptimizer:
    """Size resource requests from a high usage percentile and safety headroom."""

    def __init__(self, target_utilization: float = 0.70, headroom: float = 1.15):
        self.target_utilization = finite_number(
            target_utilization,
            "target_utilization",
        )
        self.headroom = finite_number(headroom, "headroom")
        if not 0 < self.target_utilization <= 1:
            raise ValueError("target_utilization must be greater than 0 and at most 1")
        if self.headroom < 1:
            raise ValueError("headroom must be at least 1")
        self.features: list[str] = []
        self.recommendations: dict[str, float] = {}
        self.observations = 0

    def _size(
        self,
        rows: Mapping[str, Any] | Iterable[Mapping[str, Any]],
    ) -> dict[str, float]:
        features, normalized = numeric_rows(rows, self.features or None)
        recommendations: dict[str, float] = {}
        for feature in features:
            values = sorted(row[feature] for row in normalized)
            if values[0] < 0:
                raise ValueError(f"Usage values for {feature!r} must be non-negative")
            percentile_index = max(0, math.ceil(0.95 * len(values)) - 1)
            p95 = values[percentile_index]
            recommendations[feature] = (
                p95 * self.headroom / self.target_utilization
            )
        return recommendations

    def fit(
        self,
        usage_rows: Mapping[str, Any] | Iterable[Mapping[str, Any]],
    ) -> "ResourceOptimizer":
        features, normalized = numeric_rows(usage_rows)
        self.features = features
        self.observations = len(normalized)
        self.recommendations = self._size(normalized)
        return self

    def predict(
        self,
        usage_rows: Mapping[str, Any] | Iterable[Mapping[str, Any]] | None = None,
    ) -> dict[str, float]:
        """Return fitted recommendations or size a supplied recent usage window."""
        if not self.features:
            raise RuntimeError("Fit the resource optimizer before prediction")
        if usage_rows is not None:
            return self._size(usage_rows)
        return dict(self.recommendations)

    def save(self, version: str, model_dir: str | Path | None = None) -> Path:
        if not self.features:
            raise RuntimeError("Fit the resource optimizer before saving")
        return save_artifact(
            artifact_path("resource_opt", version, model_dir),
            {
                "family": "resource_opt",
                "version": version,
                "target_utilization": self.target_utilization,
                "headroom": self.headroom,
                "features": self.features,
                "recommendations": self.recommendations,
                "observations": self.observations,
            },
        )

    @classmethod
    def from_artifact(cls, payload: dict[str, Any]) -> "ResourceOptimizer":
        model = cls(
            target_utilization=payload["target_utilization"],
            headroom=payload["headroom"],
        )
        features = payload.get("features")
        recommendations = payload.get("recommendations")
        observations = payload.get("observations")
        if (
            not isinstance(features, list)
            or not features
            or any(not isinstance(feature, str) or not feature for feature in features)
            or len(set(features)) != len(features)
            or not isinstance(recommendations, dict)
            or set(features) != set(recommendations)
            or isinstance(observations, bool)
            or not isinstance(observations, int)
            or observations < 1
        ):
            raise ValueError("Resource optimizer artifact has invalid fitted parameters")
        model.features = list(features)
        model.recommendations = {
            feature: finite_number(
                recommendations[feature],
                f"recommendation for {feature}",
            )
            for feature in features
        }
        if any(value < 0 for value in model.recommendations.values()):
            raise ValueError("Resource recommendations must be non-negative")
        model.observations = observations
        return model


def load(version: str, model_dir: str | Path | None = None) -> ResourceOptimizer:
    """Load a resource optimization model by version."""
    path = artifact_path("resource_opt", version, model_dir)
    return ResourceOptimizer.from_artifact(
        load_artifact(path, "resource_opt", version)
    )
