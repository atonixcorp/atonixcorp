"""Per-feature linear trend forecasting with versioned JSON artifacts."""

from collections.abc import Iterable, Mapping
from pathlib import Path
from typing import Any

from ..preprocess.numeric import finite_number, numeric_rows, numeric_series
from ..utils.artifacts import artifact_path, load_artifact, save_artifact


class ForecastModel:
    """Fit least-squares linear trends and forecast the next time steps."""

    def __init__(self):
        self.features: list[str] = []
        self.intercepts: dict[str, float] = {}
        self.slopes: dict[str, float] = {}
        self.observations = 0

    def fit(
        self,
        values: Iterable[Any] | Mapping[str, Any],
    ) -> "ForecastModel":
        if isinstance(values, Mapping):
            features, rows = numeric_rows(values)
        else:
            materialized = list(values) if not isinstance(values, (str, bytes)) else values
            if not isinstance(materialized, (str, bytes)) and materialized and isinstance(
                materialized[0], Mapping
            ):
                features, rows = numeric_rows(materialized)
            else:
                series = numeric_series(materialized, "observation")
                features = ["value"]
                rows = [{"value": value} for value in series]

        if len(rows) < 2:
            raise ValueError("At least two observations are required to fit a forecast")
        self.features = features
        self.observations = len(rows)
        mean_x = (self.observations - 1) / 2
        denominator = sum((index - mean_x) ** 2 for index in range(self.observations))

        for feature in self.features:
            series = [row[feature] for row in rows]
            mean_y = sum(series) / self.observations
            slope = sum(
                (index - mean_x) * (value - mean_y)
                for index, value in enumerate(series)
            ) / denominator
            self.slopes[feature] = slope
            self.intercepts[feature] = mean_y - slope * mean_x
        return self

    def predict(self, horizon: int = 1) -> list[dict[str, float]]:
        """Forecast the next positive number of time steps."""
        if not self.features:
            raise RuntimeError("Fit the forecasting model before prediction")
        if isinstance(horizon, bool) or not isinstance(horizon, int) or horizon < 1:
            raise ValueError("horizon must be a positive integer")
        return [
            {
                feature: self.intercepts[feature]
                + self.slopes[feature] * (self.observations + step)
                for feature in self.features
            }
            for step in range(horizon)
        ]

    def save(self, version: str, model_dir: str | Path | None = None) -> Path:
        if not self.features:
            raise RuntimeError("Fit the forecasting model before saving")
        return save_artifact(
            artifact_path("forecasting", version, model_dir),
            {
                "family": "forecasting",
                "version": version,
                "features": self.features,
                "intercepts": self.intercepts,
                "slopes": self.slopes,
                "observations": self.observations,
            },
        )

    @classmethod
    def from_artifact(cls, payload: dict[str, Any]) -> "ForecastModel":
        model = cls()
        features = payload.get("features")
        intercepts = payload.get("intercepts")
        slopes = payload.get("slopes")
        observations = payload.get("observations")
        if (
            not isinstance(features, list)
            or not features
            or any(not isinstance(feature, str) or not feature for feature in features)
            or len(set(features)) != len(features)
            or not isinstance(intercepts, dict)
            or not isinstance(slopes, dict)
            or set(features) != set(intercepts)
            or set(features) != set(slopes)
            or isinstance(observations, bool)
            or not isinstance(observations, int)
            or observations < 2
        ):
            raise ValueError("Forecast model artifact has invalid fitted parameters")
        model.features = list(features)
        model.intercepts = {
            feature: finite_number(intercepts[feature], f"intercept for {feature}")
            for feature in features
        }
        model.slopes = {
            feature: finite_number(slopes[feature], f"slope for {feature}")
            for feature in features
        }
        model.observations = observations
        return model


def load(version: str, model_dir: str | Path | None = None) -> ForecastModel:
    """Load a forecasting model by version."""
    path = artifact_path("forecasting", version, model_dir)
    return ForecastModel.from_artifact(load_artifact(path, "forecasting", version))
