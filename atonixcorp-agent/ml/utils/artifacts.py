"""Safe JSON persistence for versioned model artifacts."""

import json
import os
import re
import tempfile
from pathlib import Path
from typing import Any

AGENT_ROOT = Path(__file__).resolve().parents[2]
_SAFE_COMPONENT = re.compile(r"^[A-Za-z0-9][A-Za-z0-9_.-]*$")


def artifact_path(
    family: str,
    version: str,
    model_dir: str | Path | None = None,
) -> Path:
    """Build a confined path for a JSON model artifact."""
    for label, value in (("family", family), ("version", version)):
        if (
            not isinstance(value, str)
            or not _SAFE_COMPONENT.fullmatch(value)
            or value in {".", ".."}
        ):
            raise ValueError(f"Invalid model {label}: {value!r}")
    root = Path(model_dir) if model_dir is not None else AGENT_ROOT / "models"
    return root / family / f"{family}_{version}.json"


def save_artifact(path: str | Path, payload: dict[str, Any]) -> Path:
    """Atomically write a JSON artifact and reject non-JSON numeric values."""
    artifact = Path(path)
    artifact.parent.mkdir(parents=True, exist_ok=True)
    serialized = json.dumps(payload, allow_nan=False, separators=(",", ":"))
    temporary_path: Path | None = None
    try:
        with tempfile.NamedTemporaryFile(
            mode="w",
            encoding="utf-8",
            dir=artifact.parent,
            prefix=f".{artifact.name}.",
            suffix=".tmp",
            delete=False,
        ) as handle:
            temporary_path = Path(handle.name)
            handle.write(serialized)
            handle.flush()
            os.fsync(handle.fileno())
        os.replace(temporary_path, artifact)
    except OSError:
        if temporary_path is not None:
            temporary_path.unlink(missing_ok=True)
        raise
    return artifact


def load_artifact(
    path: str | Path,
    expected_family: str,
    expected_version: str | None = None,
) -> dict[str, Any]:
    """Load and validate a JSON artifact for one model family."""
    artifact = Path(path)
    try:
        with artifact.open("r", encoding="utf-8") as handle:
            payload = json.load(handle)
    except json.JSONDecodeError as error:
        raise ValueError(f"Invalid model artifact JSON in {artifact}") from error
    if not isinstance(payload, dict) or payload.get("family") != expected_family:
        raise ValueError(f"Invalid {expected_family} model artifact: {artifact}")
    if expected_version is not None and payload.get("version") != expected_version:
        raise ValueError(
            f"Model artifact version does not match requested version {expected_version!r}"
        )
    return payload
