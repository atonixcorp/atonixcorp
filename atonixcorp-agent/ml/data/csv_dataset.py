"""CSV dataset loading with explicit numeric conversion helpers."""

import csv
from pathlib import Path
from typing import Any

from ..preprocess.numeric import numeric_rows


def load_csv(path: str | Path) -> list[dict[str, Any]]:
    """Read a CSV file and convert cells that contain numbers to floats."""
    csv_path = Path(path)
    with csv_path.open("r", encoding="utf-8-sig", newline="") as handle:
        reader = csv.DictReader(handle)
        if not reader.fieldnames or any(not name for name in reader.fieldnames):
            raise ValueError(f"CSV file must contain named columns: {csv_path}")

        rows: list[dict[str, Any]] = []
        for line_number, row in enumerate(reader, start=2):
            if None in row:
                raise ValueError(f"Unexpected extra CSV values on line {line_number}")
            converted: dict[str, Any] = {}
            for name, value in row.items():
                if value is None:
                    raise ValueError(f"Missing value for {name!r} on line {line_number}")
                text = value.strip()
                if not text:
                    converted[name] = ""
                    continue
                try:
                    converted[name] = float(text)
                except ValueError:
                    converted[name] = text
            rows.append(converted)
    if not rows:
        raise ValueError(f"CSV file contains no data rows: {csv_path}")
    return rows


def load_numeric_csv(
    path: str | Path,
    features: list[str] | None = None,
) -> tuple[list[str], list[dict[str, float]]]:
    """Load a CSV and validate its selected columns as finite numeric values."""
    rows = load_csv(path)
    return numeric_rows(rows, features)
