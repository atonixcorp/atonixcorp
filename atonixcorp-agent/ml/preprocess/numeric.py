"""Validation helpers shared by the dependency-free baseline models."""

import math
from collections.abc import Iterable, Mapping, Sequence
from typing import Any


def finite_number(value: Any, field: str) -> float:
    """Convert a value to a finite float, rejecting booleans and missing data."""
    if isinstance(value, bool) or value is None:
        raise ValueError(f"{field} must be a finite number")
    try:
        number = float(value)
    except (TypeError, ValueError) as error:
        raise ValueError(f"{field} must be a finite number") from error
    if not math.isfinite(number):
        raise ValueError(f"{field} must be a finite number")
    return number


def mapping_rows(
    rows: Mapping[str, Any] | Iterable[Mapping[str, Any]],
) -> list[Mapping[str, Any]]:
    """Normalize one mapping or a non-empty iterable of mappings."""
    if isinstance(rows, Mapping):
        normalized = [rows]
    elif isinstance(rows, (str, bytes)):
        raise ValueError("Rows must be mappings, not text")
    else:
        normalized = list(rows)
    if not normalized:
        raise ValueError("At least one row is required")
    if any(not isinstance(row, Mapping) for row in normalized):
        raise ValueError("Every row must be a mapping")
    return normalized


def numeric_rows(
    rows: Mapping[str, Any] | Iterable[Mapping[str, Any]],
    features: Sequence[str] | None = None,
) -> tuple[list[str], list[dict[str, float]]]:
    """Return finite numeric rows with a stable, validated feature order."""
    source_rows = mapping_rows(rows)
    if features is None:
        selected_features = list(source_rows[0])
    else:
        selected_features = list(features)
    if not selected_features or any(
        not isinstance(feature, str) or not feature.strip()
        for feature in selected_features
    ):
        raise ValueError("At least one non-empty feature name is required")
    if len(set(selected_features)) != len(selected_features):
        raise ValueError("Feature names must be unique")

    normalized: list[dict[str, float]] = []
    for row_index, row in enumerate(source_rows):
        normalized_row: dict[str, float] = {}
        for feature in selected_features:
            if feature not in row:
                raise ValueError(f"Missing feature {feature!r} in row {row_index}")
            normalized_row[feature] = finite_number(
                row[feature],
                f"{feature!r} in row {row_index}",
            )
        normalized.append(normalized_row)
    return selected_features, normalized


def numeric_series(values: Iterable[Any], field: str = "value") -> list[float]:
    """Convert a non-empty series to finite floats."""
    if isinstance(values, (str, bytes)):
        raise ValueError("Values must be an iterable of numbers, not text")
    result = [finite_number(value, field) for value in values]
    if not result:
        raise ValueError("At least one value is required")
    return result
