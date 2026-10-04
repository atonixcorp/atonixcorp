"""Robust median/MAD anomaly detection with versioned JSON artifacts."""

from collections.abc import Iterable, Mapping, Sequence
from pathlib import Path
from typing import Any

from ..preprocess.numeric import finite_number, numeric_rows
from ..utils.artifacts import artifact_path, load_artifact, save_artifact


def _median(values: Sequence[float]) -> float:
    ordered = sorted(values)
    middle = len(ordered) // 2
    if len(ordered) % 2:
        return ordered[middle]
    return (ordered[middle - 1] + ordered[middle]) / 2


def _quantile(values: Sequence[float], fraction: float) -> float:
    ordered = sorted(values)
    position = (len(ordered) - 1) * fraction
    lower = int(position)
    upper = min(lower + 1, len(ordered) - 1)
    weight = position - lower
    return ordered[lower] * (1 - weight) + ordered[upper] * weight


class AnomalyModel:
    """Unsupervised robust outlier detector; predictions use -1/1 labels."""

    def __init__(self, threshold: float = 3.5):
        self.threshold = finite_number(threshold, "threshold")
        if self.threshold <= 0:
            raise ValueError("threshold must be greater than zero")
        self.features: list[str] = []
        self.centers: dict[str, float] = {}
        self.scales: dict[str, float] = {}

    def fit(
        self,
        rows: Mapping[str, Any] | Iterable[Mapping[str, Any]],
        features: Sequence[str] | None = None,
    ) -> "AnomalyModel":
        self.features, normalized = numeric_rows(rows, features)
        if len(normalized) < 2:
            raise ValueError("At least two rows are required to fit anomaly detection")

        for feature in self.features:
            values = [row[feature] for row in normalized]
            center = _median(values)
            deviations = [abs(value - center) for value in values]
            scale = 1.4826 * _median(deviations)
            if scale == 0:
                scale = (_quantile(values, 0.75) - _quantile(values, 0.25)) / 1.349
            self.centers[feature] = center
            self.scales[feature] = scale
        return self

    def _scores(
        self,
        rows: Mapping[str, Any] | Iterable[Mapping[str, Any]],
    ) -> list[float]:
        if not self.features:
            raise RuntimeError("Fit the anomaly model before prediction")
        _, normalized = numeric_rows(rows, self.features)
        scores: list[float] = []
        for row in normalized:
            feature_scores = []
            for feature in self.features:
                distance = abs(row[feature] - self.centers[feature])
                scale = self.scales[feature]
                feature_scores.append(
                    distance / scale if scale > 0 else (0.0 if distance == 0 else float("inf"))
                )
            scores.append(max(feature_scores))
        return scores

    def score_samples(
        self,
        rows: Mapping[str, Any] | Iterable[Mapping[str, Any]],
    ) -> list[float]:
        """Return the maximum robust z-score for each input row."""
        return self._scores(rows)

    def predict(
        self,
        rows: Mapping[str, Any] | Iterable[Mapping[str, Any]],
    ) -> list[int]:
        """Return -1 for an anomaly and 1 for an inlier."""
        return [-1 if score > self.threshold else 1 for score in self._scores(rows)]

    def save(self, version: str, model_dir: str | Path | None = None) -> Path:
        if not self.features:
            raise RuntimeError("Fit the anomaly model before saving")
        path = artifact_path("anomaly", version, model_dir)
        return save_artifact(
            path,
            {
                "family": "anomaly",
                "version": version,
                "threshold": self.threshold,
                "features": self.features,
                "centers": self.centers,
                "scales": self.scales,
            },
        )

    @classmethod
    def from_artifact(cls, payload: dict[str, Any]) -> "AnomalyModel":
        model = cls(payload["threshold"])
        features = payload.get("features")
        centers = payload.get("centers")
        scales = payload.get("scales")
        if (
            not isinstance(features, list)
            or not features
            or any(not isinstance(feature, str) or not feature for feature in features)
            or len(set(features)) != len(features)
            or not isinstance(centers, dict)
            or not isinstance(scales, dict)
            or set(features) != set(centers)
            or set(features) != set(scales)
        ):
            raise ValueError("Anomaly model artifact has invalid fitted parameters")
        model.features = list(features)
        model.centers = {
            feature: finite_number(centers[feature], f"center for {feature}")
            for feature in features
        }
        model.scales = {
            feature: finite_number(scales[feature], f"scale for {feature}")
            for feature in features
        }
        if any(scale < 0 for scale in model.scales.values()):
            raise ValueError("Anomaly model scales must be non-negative")
        return model


def load(version: str, model_dir: str | Path | None = None) -> AnomalyModel:
    """Load a trained anomaly detector by version."""
    path = artifact_path("anomaly", version, model_dir)
    return AnomalyModel.from_artifact(load_artifact(path, "anomaly", version))
