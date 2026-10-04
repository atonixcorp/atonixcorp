"""Small, dependency-free evaluation metrics."""

import math
from collections.abc import Sequence

from ..preprocess.numeric import finite_number


def _paired_values(
    actual: Sequence[float],
    predicted: Sequence[float],
) -> tuple[list[float], list[float]]:
    if len(actual) != len(predicted) or not actual:
        raise ValueError("Actual and predicted values must have the same non-zero length")
    return (
        [finite_number(value, "actual value") for value in actual],
        [finite_number(value, "predicted value") for value in predicted],
    )


def accuracy_score(actual: Sequence[int], predicted: Sequence[int]) -> float:
    """Return the fraction of labels predicted correctly."""
    actual_values, predicted_values = _paired_values(actual, predicted)
    return sum(left == right for left, right in zip(actual_values, predicted_values)) / len(
        actual_values
    )


def mean_absolute_error(actual: Sequence[float], predicted: Sequence[float]) -> float:
    """Return mean absolute prediction error."""
    actual_values, predicted_values = _paired_values(actual, predicted)
    return sum(abs(left - right) for left, right in zip(actual_values, predicted_values)) / len(
        actual_values
    )


def root_mean_squared_error(
    actual: Sequence[float],
    predicted: Sequence[float],
) -> float:
    """Return root mean squared prediction error."""
    actual_values, predicted_values = _paired_values(actual, predicted)
    return math.sqrt(
        sum((left - right) ** 2 for left, right in zip(actual_values, predicted_values))
        / len(actual_values)
    )
