import hashlib
import importlib
import importlib.util
import sys
from pathlib import Path
from types import ModuleType

from .registry import ModelRegistry

AGENT_ROOT = Path(__file__).resolve().parents[2]


class AtonixLoader:
    """Load available model components from this agent checkout."""

    def __init__(
        self,
        base_path: str | Path | None = None,
        registry: ModelRegistry | None = None,
    ):
        self.base_path = Path(base_path) if base_path is not None else AGENT_ROOT
        self.registry = registry or ModelRegistry(
            self.base_path / "model" / "registry.json"
        )

    @staticmethod
    def _load_module(module_path: Path, module_name: str) -> ModuleType:
        if not module_path.is_file():
            raise FileNotFoundError(f"Component module not found: {module_path}")
        spec = importlib.util.spec_from_file_location(module_name, module_path)
        if spec is None or spec.loader is None:
            raise ImportError(f"Could not load component module: {module_path}")
        module = importlib.util.module_from_spec(spec)
        spec.loader.exec_module(module)
        return module

    def load_ml_model(self, name: str, version: str | None = None):
        entry = self.registry.get(name, version)
        if entry is None:
            raise ValueError(f"ML model '{name}' not found in registry")

        family = entry.get("metadata", {}).get("family")
        if not isinstance(family, str) or not family or Path(family).name != family:
            raise ValueError(f"Invalid ML model family for '{name}': {family!r}")

        ml_root = (self.base_path / "ml").resolve()
        loader_file = (ml_root / family / "loader.py").resolve()
        if not loader_file.is_relative_to(ml_root):
            raise ValueError(f"Invalid ML model family for '{name}': {family!r}")

        package_name = (
            "_atonixcorp_ml_"
            + hashlib.sha256(str(ml_root).encode("utf-8")).hexdigest()[:12]
        )
        if package_name not in sys.modules:
            package_init = ml_root / "__init__.py"
            package_spec = importlib.util.spec_from_file_location(
                package_name,
                package_init,
                submodule_search_locations=[str(ml_root)],
            )
            if package_spec is None or package_spec.loader is None:
                raise ImportError(f"Could not load ML package: {ml_root}")
            package = importlib.util.module_from_spec(package_spec)
            sys.modules[package_name] = package
            package_spec.loader.exec_module(package)

        module = importlib.import_module(f"{package_name}.{family}.loader")
        load = getattr(module, "load", None)
        if not callable(load):
            raise NotImplementedError(f"ML loader for family '{family}' has no load() function")
        return load(entry["version"], model_dir=self.base_path / "models")

    def load_rl_agent(self, name: str):
        entry = self.registry.get(name)
        if entry is None:
            raise ValueError(f"RL agent '{name}' not found in registry")

        module = self._load_module(
            self.base_path / "rl" / "agent_ppo.py",
            "atonixcorp_agent_rl_agent_ppo",
        )
        load_agent = getattr(module, "load_agent", None)
        if not callable(load_agent):
            raise NotImplementedError("The RL agent module does not provide load_agent() yet.")
        return load_agent(entry["version"])

    def load_rag_pipeline(self):
        rag_root = (self.base_path / "rag").resolve()
        package_name = (
            "_atonixcorp_rag_"
            + hashlib.sha256(str(rag_root).encode("utf-8")).hexdigest()[:12]
        )
        if package_name not in sys.modules:
            package_init = rag_root / "__init__.py"
            package_spec = importlib.util.spec_from_file_location(
                package_name,
                package_init,
                submodule_search_locations=[str(rag_root)],
            )
            if package_spec is None or package_spec.loader is None:
                raise ImportError(f"Could not load RAG package: {rag_root}")
            package = importlib.util.module_from_spec(package_spec)
            sys.modules[package_name] = package
            package_spec.loader.exec_module(package)

        module = importlib.import_module(f"{package_name}.pipeline")
        load_pipeline = getattr(module, "load_pipeline", None)
        if not callable(load_pipeline):
            raise NotImplementedError("The RAG pipeline module does not provide load_pipeline() yet.")
        return load_pipeline(index_path=self.base_path / "db" / "rag_index.json")
