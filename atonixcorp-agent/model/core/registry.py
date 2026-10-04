import json
from copy import deepcopy
from datetime import datetime, timezone
from pathlib import Path
from typing import Any

from .json_store import read_json, write_json

AGENT_ROOT = Path(__file__).resolve().parents[2]
REGISTRY_FILE = AGENT_ROOT / "model" / "registry.json"
VALID_STAGES = {"staging", "production"}


def _utc_now() -> str:
    return datetime.now(timezone.utc).isoformat()


class ModelRegistry:
    """Persistent model/version registry backed by an atomic JSON file."""

    def __init__(self, path: str | Path = REGISTRY_FILE):
        self.path = Path(path)
        self.registry = read_json(self.path, {"models": {}})
        if not isinstance(self.registry.get("models"), dict):
            raise ValueError(f"Expected a 'models' object in {self.path}")
        self._migrate_legacy_entries()

    def _migrate_legacy_entries(self) -> None:
        changed = False
        for name, model in self.registry["models"].items():
            if "versions" in model:
                continue
            version = model.get("version")
            if not isinstance(version, str) or not version:
                raise ValueError(f"Model {name} has no valid registered version")
            entry = {
                "name": name,
                "version": version,
                "created_at": model.get("created_at", _utc_now()),
                "stage": model.get("stage", "staging"),
                "metadata": deepcopy(model.get("metadata", {})),
                "promotion_history": [],
            }
            model["versions"] = {version: entry}
            changed = True
        if changed:
            self._save()

    def _save(self) -> None:
        write_json(self.path, self.registry)

    def register(
        self,
        name: str,
        version: str,
        metadata: dict[str, Any] | None = None,
    ) -> dict[str, Any]:
        if not isinstance(name, str) or not name.strip():
            raise ValueError("Model name must be a non-empty string")
        if not isinstance(version, str) or not version.strip():
            raise ValueError("Model version must be a non-empty string")
        if metadata is not None and not isinstance(metadata, dict):
            raise ValueError("Model metadata must be a dictionary")
        json.dumps(metadata or {}, allow_nan=False)

        name = name.strip()
        version = version.strip()
        model = self.registry["models"].get(name)
        if model is None:
            model = {"name": name, "versions": {}}
            self.registry["models"][name] = model

        versions = model.setdefault("versions", {})
        if version in versions:
            raise ValueError(f"Model {name} version {version} is already registered")

        now = _utc_now()
        entry = {
            "name": name,
            "version": version,
            "created_at": now,
            "stage": "staging",
            "metadata": deepcopy(metadata or {}),
            "promotion_history": [],
        }
        versions[version] = entry
        model.update(
            {
                "version": version,
                "created_at": now,
                "stage": "staging",
                "metadata": deepcopy(entry["metadata"]),
            }
        )
        self._save()
        return deepcopy(entry)

    def promote(
        self,
        name: str,
        stage: str = "production",
        version: str | None = None,
    ) -> dict[str, Any]:
        if stage not in VALID_STAGES:
            raise ValueError(f"Stage must be one of: {', '.join(sorted(VALID_STAGES))}")

        model = self.registry["models"].get(name)
        if model is None:
            raise ValueError(f"Model {name} not found")

        selected_version = version or model["version"]
        entry = model.get("versions", {}).get(selected_version)
        if entry is None:
            if selected_version == model.get("version"):
                entry = {
                    "name": name,
                    "version": model["version"],
                    "created_at": model["created_at"],
                    "stage": model.get("stage", "staging"),
                    "metadata": model.get("metadata", {}),
                    "promotion_history": [],
                }
                model.setdefault("versions", {})[selected_version] = entry
            else:
                raise ValueError(f"Model {name} version {selected_version} not found")

        previous_stage = entry["stage"]
        entry["stage"] = stage
        entry.setdefault("promotion_history", []).append(
            {"from": previous_stage, "to": stage, "at": _utc_now()}
        )
        if selected_version == model["version"]:
            model["stage"] = stage
        if stage == "production":
            model["production_version"] = selected_version

        self._save()
        return deepcopy(entry)

    def get(self, name: str, version: str | None = None) -> dict[str, Any] | None:
        model = self.registry["models"].get(name)
        if model is None:
            return None
        if version is None:
            version = model["version"]
        entry = model.get("versions", {}).get(version)
        if entry is not None:
            return deepcopy(entry)
        if version == model.get("version"):
            return deepcopy(model)
        return None

    def list(self) -> list[dict[str, Any]]:
        return [deepcopy(model) for model in self.registry["models"].values()]
