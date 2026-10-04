"""SQLite persistence and repositories for agent state."""

from __future__ import annotations

import json
import math
import sqlite3
import threading
import uuid
from contextlib import contextmanager
from datetime import datetime, timezone
from pathlib import Path
from typing import Any, Generator

from .schema import MIGRATIONS_DIR, migrate

AGENT_ROOT = Path(__file__).resolve().parents[1]
DEFAULT_DATABASE_PATH = AGENT_ROOT / "db" / "atonixcorp-agent.sqlite3"


def _utc_now() -> str:
    return datetime.now(timezone.utc).isoformat()


def _json(value: Any, field: str) -> str:
    try:
        return json.dumps(value, allow_nan=False, separators=(",", ":"))
    except (TypeError, ValueError) as error:
        raise ValueError(f"{field} must contain only finite JSON values") from error


def _decode(value: str, field: str) -> Any:
    try:
        return json.loads(value)
    except json.JSONDecodeError as error:
        raise ValueError(f"Invalid stored JSON in {field}") from error


class Database:
    """SQLite connection with migrations and transaction-aware domain operations.

    The object is safe for serialized use across threads in one process.
    Applications that need multi-process deployments should provide their own
    operational backups, authorization, and database lifecycle controls.
    """

    def __init__(
        self,
        path: str | Path = DEFAULT_DATABASE_PATH,
        migrations_dir: str | Path = MIGRATIONS_DIR,
        timeout: float = 5.0,
    ):
        if isinstance(timeout, bool) or not isinstance(timeout, (int, float)):
            raise ValueError("timeout must be a positive finite number")
        if not math.isfinite(timeout) or timeout <= 0:
            raise ValueError("timeout must be a positive finite number")
        self.path = Path(path) if str(path) != ":memory:" else Path(":memory:")
        if str(path) != ":memory:":
            self.path.parent.mkdir(parents=True, exist_ok=True)
            connection_path = str(self.path)
        else:
            connection_path = ":memory:"
        self._lock = threading.RLock()
        self._local = threading.local()
        self.connection = sqlite3.connect(
            connection_path,
            timeout=float(timeout),
            isolation_level=None,
            check_same_thread=False,
        )
        self.connection.row_factory = sqlite3.Row
        self.connection.execute("PRAGMA foreign_keys = ON")
        self.connection.execute(f"PRAGMA busy_timeout = {max(1, int(timeout * 1000))}")
        if connection_path != ":memory:":
            self.connection.execute("PRAGMA journal_mode = WAL")
            self.connection.execute("PRAGMA synchronous = FULL")
        try:
            migrate(self.connection, migrations_dir)
        except Exception:
            self.connection.close()
            raise

    def close(self) -> None:
        """Close the underlying SQLite connection."""
        with self._lock:
            self.connection.close()

    def __enter__(self) -> "Database":
        return self

    def __exit__(self, _exc_type: Any, _exc_value: Any, _traceback: Any) -> None:
        del _exc_type, _exc_value, _traceback
        self.close()

    @contextmanager
    def transaction(self) -> Generator[sqlite3.Connection, None, None]:
        """Open an atomic transaction, nesting with savepoints as needed."""
        with self._lock:
            depth = getattr(self._local, "transaction_depth", 0)
            savepoint = f"agent_sp_{depth}"
            if depth == 0:
                self.connection.execute("BEGIN IMMEDIATE")
            else:
                self.connection.execute(f"SAVEPOINT {savepoint}")
            self._local.transaction_depth = depth + 1
            try:
                yield self.connection
            except Exception:
                if depth == 0:
                    self.connection.rollback()
                else:
                    self.connection.execute(f"ROLLBACK TO SAVEPOINT {savepoint}")
                    self.connection.execute(f"RELEASE SAVEPOINT {savepoint}")
                raise
            else:
                if depth == 0:
                    self.connection.commit()
                else:
                    self.connection.execute(f"RELEASE SAVEPOINT {savepoint}")
            finally:
                self._local.transaction_depth = depth

    def migration_status(self) -> list[dict[str, Any]]:
        """Return applied schema migrations."""
        rows = self.connection.execute(
            "SELECT version, name, checksum, applied_at "
            "FROM schema_migrations ORDER BY version"
        ).fetchall()
        return [dict(row) for row in rows]

    def register_model(
        self,
        name: str,
        version: str,
        metadata: dict[str, Any] | None = None,
        stage: str = "staging",
        created_at: str | None = None,
    ) -> dict[str, Any]:
        """Register one model version and mark it current."""
        if not isinstance(name, str) or not name.strip():
            raise ValueError("Model name must be a non-empty string")
        if not isinstance(version, str) or not version.strip():
            raise ValueError("Model version must be a non-empty string")
        if stage not in {"staging", "production"}:
            raise ValueError("stage must be 'staging' or 'production'")
        if metadata is None:
            metadata = {}
        if not isinstance(metadata, dict):
            raise ValueError("metadata must be a dictionary")
        metadata_json = _json(metadata, "metadata")
        name, version = name.strip(), version.strip()
        now = created_at or _utc_now()

        with self.transaction() as connection:
            connection.execute(
                """
                INSERT INTO models(name, current_version, production_version, created_at, updated_at)
                VALUES (?, ?, ?, ?, ?)
                ON CONFLICT(name) DO NOTHING
                """,
                (name, version, version if stage == "production" else None, now, now),
            )
            connection.execute(
                """
                INSERT INTO model_versions(
                    model_name, version, stage, metadata_json, created_at,
                    promotion_history_json
                ) VALUES (?, ?, ?, ?, ?, '[]')
                """,
                (name, version, stage, metadata_json, now),
            )
            connection.execute(
                "UPDATE models SET current_version = ?, updated_at = ?, "
                "production_version = CASE WHEN ? = 'production' "
                "THEN ? ELSE production_version END WHERE name = ?",
                (version, now, stage, version, name),
            )
        return self.get_model(name, version)  # type: ignore[return-value]

    @staticmethod
    def _model_version_row(row: sqlite3.Row) -> dict[str, Any]:
        return {
            "name": row["model_name"],
            "version": row["version"],
            "stage": row["stage"],
            "metadata": _decode(row["metadata_json"], "model metadata"),
            "created_at": row["created_at"],
            "promotion_history": _decode(
                row["promotion_history_json"],
                "model promotion history",
            ),
        }

    def get_model(self, name: str, version: str | None = None) -> dict[str, Any] | None:
        """Fetch a registered model version, defaulting to the current version."""
        if version is None:
            current = self.connection.execute(
                "SELECT current_version FROM models WHERE name = ?",
                (name,),
            ).fetchone()
            if current is None:
                return None
            version = current["current_version"]
        row = self.connection.execute(
            """
            SELECT model_name, version, stage, metadata_json, created_at,
                   promotion_history_json
            FROM model_versions WHERE model_name = ? AND version = ?
            """,
            (name, version),
        ).fetchone()
        return self._model_version_row(row) if row is not None else None

    def list_models(self) -> list[dict[str, Any]]:
        """List model records with current version and all version details."""
        rows = self.connection.execute(
            """
            SELECT m.name, m.current_version, m.production_version,
                   m.created_at, m.updated_at
            FROM models AS m ORDER BY m.name
            """
        ).fetchall()
        result = []
        for model in rows:
            versions = self.connection.execute(
                """
                SELECT model_name, version, stage, metadata_json, created_at,
                       promotion_history_json
                FROM model_versions WHERE model_name = ? ORDER BY created_at, version
                """,
                (model["name"],),
            ).fetchall()
            result.append(
                {
                    "name": model["name"],
                    "version": model["current_version"],
                    "production_version": model["production_version"],
                    "created_at": model["created_at"],
                    "updated_at": model["updated_at"],
                    "versions": {
                        row["version"]: self._model_version_row(row)
                        for row in versions
                    },
                }
            )
        return result

    def promote_model(
        self,
        name: str,
        version: str | None = None,
        stage: str = "production",
    ) -> dict[str, Any]:
        """Change one registered version's stage and record promotion history."""
        if stage not in {"staging", "production"}:
            raise ValueError("stage must be 'staging' or 'production'")
        with self.transaction() as connection:
            if version is None:
                model = connection.execute(
                    "SELECT current_version FROM models WHERE name = ?",
                    (name,),
                ).fetchone()
                if model is None:
                    raise ValueError(f"Model {name!r} not found")
                version = model["current_version"]
            row = connection.execute(
                """
                SELECT stage, promotion_history_json FROM model_versions
                WHERE model_name = ? AND version = ?
                """,
                (name, version),
            ).fetchone()
            if row is None:
                raise ValueError(f"Model {name!r} version {version!r} not found")
            history = _decode(row["promotion_history_json"], "model promotion history")
            history.append(
                {"from": row["stage"], "to": stage, "at": _utc_now()}
            )
            connection.execute(
                """
                UPDATE model_versions
                SET stage = ?, promotion_history_json = ?
                WHERE model_name = ? AND version = ?
                """,
                (stage, _json(history, "promotion history"), name, version),
            )
            if stage == "production":
                connection.execute(
                    "UPDATE models SET production_version = ?, updated_at = ? WHERE name = ?",
                    (version, _utc_now(), name),
                )
        return self.get_model(name, version)  # type: ignore[return-value]

    def start_run(
        self,
        model_name: str,
        version: str,
        run_id: str | None = None,
        created_at: str | None = None,
    ) -> str:
        """Create a metric run linked to a registered model version."""
        if not isinstance(model_name, str) or not model_name.strip():
            raise ValueError("Model name must be a non-empty string")
        if not isinstance(version, str) or not version.strip():
            raise ValueError("Model version must be a non-empty string")
        model_name, version = model_name.strip(), version.strip()
        run_id = run_id or f"{model_name}-{version}-{uuid.uuid4().hex}"
        now = created_at or _utc_now()
        with self.transaction() as connection:
            exists = connection.execute(
                "SELECT 1 FROM model_versions WHERE model_name = ? AND version = ?",
                (model_name, version),
            ).fetchone()
            if exists is None:
                connection.execute(
                    """
                    INSERT INTO models(name, current_version, created_at, updated_at)
                    VALUES (?, ?, ?, ?)
                    ON CONFLICT(name) DO NOTHING
                    """,
                    (model_name, version, now, now),
                )
                connection.execute(
                    """
                    INSERT INTO model_versions(
                        model_name, version, stage, metadata_json, created_at,
                        promotion_history_json
                    ) VALUES (?, ?, 'staging', '{}', ?, '[]')
                    """,
                    (model_name, version, now),
                )
                connection.execute(
                    "UPDATE models SET current_version = ?, updated_at = ? WHERE name = ?",
                    (version, now, model_name),
                )
            connection.execute(
                """
                INSERT INTO metric_runs(run_id, model_name, model_version, created_at, updated_at)
                VALUES (?, ?, ?, ?, ?)
                """,
                (run_id, model_name, version, now, now),
            )
        return run_id

    def log_metric(self, run_id: str, name: str, value: int | float) -> None:
        """Insert or update one finite numeric metric on a run."""
        if not isinstance(name, str) or not name.strip():
            raise ValueError("Metric name must be a non-empty string")
        if isinstance(value, bool) or not isinstance(value, (int, float)):
            raise ValueError("Metric value must be a finite number")
        if not math.isfinite(value):
            raise ValueError("Metric value must be a finite number")
        with self.transaction() as connection:
            exists = connection.execute(
                "SELECT 1 FROM metric_runs WHERE run_id = ?",
                (run_id,),
            ).fetchone()
            if exists is None:
                raise ValueError(f"Run {run_id!r} not found")
            connection.execute(
                """
                INSERT INTO metrics(run_id, name, value) VALUES (?, ?, ?)
                ON CONFLICT(run_id, name) DO UPDATE SET value = excluded.value
                """,
                (run_id, name.strip(), float(value)),
            )
            connection.execute(
                "UPDATE metric_runs SET updated_at = ? WHERE run_id = ?",
                (_utc_now(), run_id),
            )

    def get_recent_metrics(
        self,
        model_name: str,
        version: str | None = None,
    ) -> dict[str, Any] | None:
        """Return metric values from the latest run for a model/version."""
        if version is None:
            row = self.connection.execute(
                """
                SELECT run_id FROM metric_runs WHERE model_name = ?
                ORDER BY updated_at DESC, created_at DESC LIMIT 1
                """,
                (model_name,),
            ).fetchone()
        else:
            row = self.connection.execute(
                """
                SELECT run_id FROM metric_runs
                WHERE model_name = ? AND model_version = ?
                ORDER BY updated_at DESC, created_at DESC LIMIT 1
                """,
                (model_name, version),
            ).fetchone()
        if row is None:
            return None
        metrics = self.connection.execute(
            "SELECT name, value FROM metrics WHERE run_id = ? ORDER BY name",
            (row["run_id"],),
        ).fetchall()
        return {metric["name"]: metric["value"] for metric in metrics}

    def list_runs(self, model_name: str | None = None) -> dict[str, dict[str, Any]]:
        """Return metric runs and their metric values."""
        if model_name is None:
            rows = self.connection.execute(
                "SELECT * FROM metric_runs ORDER BY created_at, run_id"
            ).fetchall()
        else:
            rows = self.connection.execute(
                "SELECT * FROM metric_runs WHERE model_name = ? ORDER BY created_at, run_id",
                (model_name,),
            ).fetchall()
        result = {}
        for row in rows:
            metrics = self.connection.execute(
                "SELECT name, value FROM metrics WHERE run_id = ? ORDER BY name",
                (row["run_id"],),
            ).fetchall()
            result[row["run_id"]] = {
                "model": row["model_name"],
                "version": row["model_version"],
                "metrics": {metric["name"]: metric["value"] for metric in metrics},
                "created_at": row["created_at"],
                "updated_at": row["updated_at"],
            }
        return result

    def log_event(
        self,
        event_type: str,
        payload: Any,
        event_id: str | None = None,
        created_at: str | None = None,
    ) -> dict[str, Any]:
        """Persist a structured agent event."""
        if not isinstance(event_type, str) or not event_type.strip():
            raise ValueError("Event type must be a non-empty string")
        event_id = event_id or uuid.uuid4().hex
        timestamp = created_at or _utc_now()
        payload_json = _json(payload, "event payload")
        with self.transaction() as connection:
            connection.execute(
                "INSERT INTO events(id, event_type, payload_json, created_at) VALUES (?, ?, ?, ?)",
                (event_id, event_type.strip(), payload_json, timestamp),
            )
        return {
            "id": event_id,
            "type": event_type.strip(),
            "payload": json.loads(payload_json),
            "timestamp": timestamp,
        }

    def list_events(
        self,
        event_type: str | None = None,
        limit: int = 100,
    ) -> list[dict[str, Any]]:
        """Return newest events first with an optional exact type filter."""
        if isinstance(limit, bool) or not isinstance(limit, int) or not 0 <= limit <= 1000:
            raise ValueError("limit must be an integer between 0 and 1000")
        if limit == 0:
            return []
        if event_type is None:
            rows = self.connection.execute(
                "SELECT id, event_type, payload_json, created_at FROM events "
                "ORDER BY created_at DESC, rowid DESC LIMIT ?",
                (limit,),
            ).fetchall()
        else:
            rows = self.connection.execute(
                "SELECT id, event_type, payload_json, created_at FROM events "
                "WHERE event_type = ? ORDER BY created_at DESC, rowid DESC LIMIT ?",
                (event_type, limit),
            ).fetchall()
        return [
            {
                "id": row["id"],
                "type": row["event_type"],
                "payload": _decode(row["payload_json"], "event payload"),
                "timestamp": row["created_at"],
            }
            for row in rows
        ]

    def record_incident(
        self,
        service: str,
        root_cause: str,
        resolution: str,
        details: dict[str, Any] | None = None,
        incident_id: str | None = None,
        created_at: str | None = None,
    ) -> dict[str, Any]:
        """Persist incident knowledge and an associated event atomically."""
        for field, value in (
            ("service", service),
            ("root_cause", root_cause),
            ("resolution", resolution),
        ):
            if not isinstance(value, str) or not value.strip():
                raise ValueError(f"{field} must be a non-empty string")
        if details is None:
            details = {}
        if not isinstance(details, dict):
            raise ValueError("details must be a dictionary")
        incident_id = incident_id or uuid.uuid4().hex
        timestamp = created_at or _utc_now()
        payload = {
            **details,
            "service": service.strip(),
            "root_cause": root_cause.strip(),
            "resolution": resolution.strip(),
        }
        with self.transaction() as connection:
            connection.execute(
                """
                INSERT INTO incidents(id, service, root_cause, resolution, details_json, created_at)
                VALUES (?, ?, ?, ?, ?, ?)
                """,
                (
                    incident_id,
                    service.strip(),
                    root_cause.strip(),
                    resolution.strip(),
                    _json(details, "incident details"),
                    timestamp,
                ),
            )
            connection.execute(
                "INSERT INTO events(id, event_type, payload_json, created_at) "
                "VALUES (?, 'incident', ?, ?)",
                (incident_id, _json(payload, "incident payload"), timestamp),
            )
        return {"id": incident_id, **payload, "created_at": timestamp}

    def list_incidents(self, service: str | None = None, limit: int = 100) -> list[dict[str, Any]]:
        """List incident records, optionally filtered by service."""
        if isinstance(limit, bool) or not isinstance(limit, int) or not 0 <= limit <= 1000:
            raise ValueError("limit must be an integer between 0 and 1000")
        if limit == 0:
            return []
        if service is None:
            rows = self.connection.execute(
                "SELECT * FROM incidents ORDER BY created_at DESC, rowid DESC LIMIT ?",
                (limit,),
            ).fetchall()
        else:
            rows = self.connection.execute(
                "SELECT * FROM incidents WHERE service = ? "
                "ORDER BY created_at DESC, rowid DESC LIMIT ?",
                (service, limit),
            ).fetchall()
        return [
            {
                "id": row["id"],
                "service": row["service"],
                "root_cause": row["root_cause"],
                "resolution": row["resolution"],
                "details": _decode(row["details_json"], "incident details"),
                "created_at": row["created_at"],
            }
            for row in rows
        ]

    def record_action(
        self,
        action_name: str,
        details: Any,
        status: str = "recorded",
        action_id: str | None = None,
        created_at: str | None = None,
    ) -> dict[str, Any]:
        """Persist an action audit record and its associated event."""
        if not isinstance(action_name, str) or not action_name.strip():
            raise ValueError("action_name must be a non-empty string")
        if not isinstance(status, str) or not status.strip():
            raise ValueError("status must be a non-empty string")
        action_id = action_id or uuid.uuid4().hex
        timestamp = created_at or _utc_now()
        details_json = _json(details, "action details")
        with self.transaction() as connection:
            connection.execute(
                """
                INSERT INTO actions(id, action_name, details_json, status, created_at)
                VALUES (?, ?, ?, ?, ?)
                """,
                (action_id, action_name.strip(), details_json, status.strip(), timestamp),
            )
            connection.execute(
                "INSERT INTO events(id, event_type, payload_json, created_at) "
                "VALUES (?, 'action_executed', ?, ?)",
                (
                    action_id,
                    _json(
                        {"action": action_name.strip(), "details": details, "status": status.strip()},
                        "action event",
                    ),
                    timestamp,
                ),
            )
        return {
            "id": action_id,
            "action": action_name.strip(),
            "details": json.loads(details_json),
            "status": status.strip(),
            "created_at": timestamp,
        }

    def list_actions(self, action_name: str | None = None, limit: int = 100) -> list[dict[str, Any]]:
        """List recent action audit records."""
        if isinstance(limit, bool) or not isinstance(limit, int) or not 0 <= limit <= 1000:
            raise ValueError("limit must be an integer between 0 and 1000")
        if limit == 0:
            return []
        if action_name is None:
            rows = self.connection.execute(
                "SELECT * FROM actions ORDER BY created_at DESC, rowid DESC LIMIT ?",
                (limit,),
            ).fetchall()
        else:
            rows = self.connection.execute(
                "SELECT * FROM actions WHERE action_name = ? "
                "ORDER BY created_at DESC, rowid DESC LIMIT ?",
                (action_name, limit),
            ).fetchall()
        return [
            {
                "id": row["id"],
                "action": row["action_name"],
                "details": _decode(row["details_json"], "action details"),
                "status": row["status"],
                "created_at": row["created_at"],
            }
            for row in rows
        ]

    def upsert_rag_document(
        self,
        chunk_id: str,
        document_id: str,
        source: str,
        chunk_index: int,
        text: str,
        embedding: list[float],
        metadata: dict[str, Any] | None = None,
        created_at: str | None = None,
    ) -> None:
        """Insert or update a RAG chunk and its vector representation."""
        if not all(isinstance(value, str) and value.strip() for value in (chunk_id, document_id, source, text)):
            raise ValueError("RAG chunk IDs, source, and text must be non-empty strings")
        if isinstance(chunk_index, bool) or not isinstance(chunk_index, int) or chunk_index < 0:
            raise ValueError("chunk_index must be a non-negative integer")
        if not isinstance(embedding, list) or not embedding:
            raise ValueError("embedding must be a non-empty numeric list")
        if any(
            isinstance(value, bool)
            or not isinstance(value, (int, float))
            or not math.isfinite(value)
            for value in embedding
        ):
            raise ValueError("embedding must contain only finite numbers")
        if metadata is None:
            metadata = {}
        if not isinstance(metadata, dict):
            raise ValueError("metadata must be a dictionary")
        now = created_at or _utc_now()
        with self.transaction() as connection:
            connection.execute(
                """
                INSERT INTO rag_documents(
                    chunk_id, document_id, source, chunk_index, text,
                    embedding_json, metadata_json, created_at
                ) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
                ON CONFLICT(chunk_id) DO UPDATE SET
                    document_id = excluded.document_id,
                    source = excluded.source,
                    chunk_index = excluded.chunk_index,
                    text = excluded.text,
                    embedding_json = excluded.embedding_json,
                    metadata_json = excluded.metadata_json,
                    created_at = excluded.created_at
                """,
                (
                    chunk_id,
                    document_id,
                    source,
                    chunk_index,
                    text,
                    _json(embedding, "embedding"),
                    _json(metadata, "RAG metadata"),
                    now,
                ),
            )

    def list_rag_documents(self, document_id: str | None = None) -> list[dict[str, Any]]:
        """List RAG chunks, optionally for one logical document."""
        if document_id is None:
            rows = self.connection.execute(
                "SELECT * FROM rag_documents ORDER BY document_id, chunk_index"
            ).fetchall()
        else:
            rows = self.connection.execute(
                "SELECT * FROM rag_documents WHERE document_id = ? ORDER BY chunk_index",
                (document_id,),
            ).fetchall()
        return [
            {
                "id": row["chunk_id"],
                "document_id": row["document_id"],
                "source": row["source"],
                "chunk_index": row["chunk_index"],
                "text": row["text"],
                "embedding": _decode(row["embedding_json"], "RAG embedding"),
                "metadata": _decode(row["metadata_json"], "RAG metadata"),
                "created_at": row["created_at"],
            }
            for row in rows
        ]

    def delete_rag_document(self, document_id: str) -> int:
        """Delete all indexed chunks for one logical document."""
        with self.transaction() as connection:
            cursor = connection.execute(
                "DELETE FROM rag_documents WHERE document_id = ?",
                (document_id,),
            )
        return cursor.rowcount

    def import_legacy_json(self, root: str | Path = AGENT_ROOT) -> dict[str, int]:
        """Import existing JSON registry, metrics, events, and RAG index once.

        Existing records are preserved using INSERT OR IGNORE. This is safe to
        rerun; import counts report only records inserted by this invocation.
        """
        root_path = Path(root)
        registry_path = root_path / "model" / "registry.json"
        metrics_path = root_path / "db" / "metrics.json"
        events_path = root_path / "db" / "events.json"
        rag_path = root_path / "db" / "rag_index.json"

        def read(path: Path, default: dict[str, Any]) -> dict[str, Any]:
            if not path.exists():
                return default
            try:
                with path.open("r", encoding="utf-8") as handle:
                    payload = json.load(handle)
            except json.JSONDecodeError as error:
                raise ValueError(f"Invalid legacy JSON in {path}") from error
            if not isinstance(payload, dict):
                raise ValueError(f"Expected a JSON object in {path}")
            return payload

        registry = read(registry_path, {"models": {}})
        metrics = read(metrics_path, {"runs": {}})
        events = read(events_path, {"events": []})
        rag = read(rag_path, {"documents": []})
        models = registry.get("models", {})
        runs = metrics.get("runs", {})
        event_list = events.get("events", [])
        documents = rag.get("documents", [])
        if not isinstance(models, dict) or not isinstance(runs, dict):
            raise ValueError("Legacy registry and metrics JSON have invalid structure")
        if not isinstance(event_list, list) or not isinstance(documents, list):
            raise ValueError("Legacy event and RAG JSON have invalid structure")
        counts = {
            "models": 0,
            "model_versions": 0,
            "runs": 0,
            "metrics": 0,
            "events": 0,
            "rag_chunks": 0,
        }

        with self.transaction() as connection:
            for model_name, model in models.items():
                if not isinstance(model, dict):
                    raise ValueError(f"Invalid legacy model record: {model_name}")
                versions = model.get("versions")
                if not isinstance(versions, dict):
                    version = model.get("version")
                    if not isinstance(version, str) or not version:
                        raise ValueError(f"Invalid legacy model version: {model_name}")
                    versions = {version: model}
                if not versions or any(
                    not isinstance(version, str) or not version
                    for version in versions
                ):
                    raise ValueError(f"Invalid legacy model versions: {model_name}")
                current_version = model.get("version") or next(reversed(versions))
                if current_version not in versions:
                    raise ValueError(
                        f"Current version {current_version!r} is missing for model {model_name}"
                    )
                created_at = model.get("created_at") or _utc_now()
                production_version = model.get("production_version")
                connection.execute(
                    """
                    INSERT OR IGNORE INTO models(
                        name, current_version, production_version, created_at, updated_at
                    ) VALUES (?, ?, ?, ?, ?)
                    """,
                    (model_name, current_version, production_version, created_at, created_at),
                )
                counts["models"] += connection.execute("SELECT changes()").fetchone()[0]
                for version, entry in versions.items():
                    if not isinstance(entry, dict):
                        raise ValueError(f"Invalid legacy model version entry: {model_name}/{version}")
                    stage = entry.get("stage", "staging")
                    if stage not in {"staging", "production"}:
                        stage = "staging"
                    entry_metadata = entry.get("metadata", {})
                    if not isinstance(entry_metadata, dict):
                        raise ValueError(f"Invalid metadata for {model_name}/{version}")
                    connection.execute(
                        """
                        INSERT OR IGNORE INTO model_versions(
                            model_name, version, stage, metadata_json, created_at,
                            promotion_history_json
                        ) VALUES (?, ?, ?, ?, ?, ?)
                        """,
                        (
                            model_name,
                            version,
                            stage,
                            _json(entry_metadata, "model metadata"),
                            entry.get("created_at", created_at),
                            _json(entry.get("promotion_history", []), "promotion history"),
                        ),
                    )
                    counts["model_versions"] += connection.execute(
                        "SELECT changes()"
                    ).fetchone()[0]
                connection.execute(
                    """
                    UPDATE models SET current_version = ?,
                        production_version = COALESCE(production_version, ?)
                    WHERE name = ?
                    """,
                    (current_version, production_version, model_name),
                )

            for run_id, run in runs.items():
                if not isinstance(run, dict):
                    raise ValueError(f"Invalid legacy metric run: {run_id}")
                model_name = run.get("model")
                version = run.get("version")
                if not isinstance(model_name, str) or not isinstance(version, str):
                    raise ValueError(f"Metric run {run_id} has no model/version")
                timestamp = run.get("created_at") or _utc_now()
                connection.execute(
                    """
                    INSERT OR IGNORE INTO models(name, current_version, created_at, updated_at)
                    VALUES (?, ?, ?, ?)
                    """,
                    (model_name, version, timestamp, timestamp),
                )
                counts["models"] += connection.execute("SELECT changes()").fetchone()[0]
                connection.execute(
                    """
                    INSERT OR IGNORE INTO model_versions(
                        model_name, version, stage, metadata_json, created_at,
                        promotion_history_json
                    ) VALUES (?, ?, 'staging', '{}', ?, '[]')
                    """,
                    (model_name, version, timestamp),
                )
                connection.execute(
                    "INSERT OR IGNORE INTO metric_runs VALUES (?, ?, ?, ?, ?)",
                    (
                        run_id,
                        model_name,
                        version,
                        timestamp,
                        run.get("updated_at") or timestamp,
                    ),
                )
                counts["runs"] += connection.execute("SELECT changes()").fetchone()[0]
                run_metrics = run.get("metrics", {})
                if not isinstance(run_metrics, dict):
                    raise ValueError(f"Invalid metrics for run {run_id}")
                for metric_name, value in run_metrics.items():
                    if (
                        isinstance(value, bool)
                        or not isinstance(value, (int, float))
                        or not math.isfinite(value)
                    ):
                        raise ValueError(f"Invalid metric {metric_name!r} for run {run_id}")
                    connection.execute(
                        "INSERT OR IGNORE INTO metrics(run_id, name, value) VALUES (?, ?, ?)",
                        (run_id, metric_name, value),
                    )
                    counts["metrics"] += connection.execute(
                        "SELECT changes()"
                    ).fetchone()[0]

            for event in event_list:
                if not isinstance(event, dict):
                    raise ValueError("Invalid legacy event entry")
                event_id = event.get("id")
                event_type = event.get("type")
                timestamp = event.get("timestamp") or _utc_now()
                if not isinstance(event_id, str) or not isinstance(event_type, str):
                    raise ValueError("Legacy event is missing its id or type")
                payload_json = _json(event.get("payload"), "event payload")
                connection.execute(
                    "INSERT OR IGNORE INTO events VALUES (?, ?, ?, ?)",
                    (event_id, event_type, payload_json, timestamp),
                )
                counts["events"] += connection.execute("SELECT changes()").fetchone()[0]
                payload = event.get("payload")
                if event_type == "incident" and isinstance(payload, dict):
                    service = payload.get("service", "unknown")
                    connection.execute(
                        """
                        INSERT OR IGNORE INTO incidents(
                            id, service, root_cause, resolution, details_json, created_at
                        ) VALUES (?, ?, ?, ?, '{}', ?)
                        """,
                        (
                            event_id,
                            str(service),
                            str(payload.get("root_cause", "")),
                            str(payload.get("resolution", "")),
                            timestamp,
                        ),
                    )
                elif event_type == "action_executed" and isinstance(payload, dict):
                    connection.execute(
                        """
                        INSERT OR IGNORE INTO actions(
                            id, action_name, details_json, status, created_at
                        ) VALUES (?, ?, ?, 'recorded', ?)
                        """,
                        (
                            event_id,
                            str(payload.get("action", "unknown")),
                            _json(payload.get("details"), "action details"),
                            timestamp,
                        ),
                    )

            for document in documents:
                if not isinstance(document, dict):
                    raise ValueError("Invalid legacy RAG document entry")
                doc_id = document.get("id")
                text = document.get("text")
                metadata = document.get("metadata", {})
                embedding = document.get("embedding")
                if not isinstance(doc_id, str) or not isinstance(text, str):
                    raise ValueError("Legacy RAG document is missing id or text")
                if not isinstance(metadata, dict):
                    raise ValueError(f"Invalid metadata in RAG document {doc_id}")
                document_id = metadata.get("document_id", doc_id)
                source = metadata.get("source", "unknown")
                chunk_index = metadata.get("chunk_index", 0)
                if (
                    not isinstance(document_id, str)
                    or not isinstance(source, str)
                    or isinstance(chunk_index, bool)
                    or not isinstance(chunk_index, int)
                    or chunk_index < 0
                    or not isinstance(embedding, list)
                    or not embedding
                    or any(
                        isinstance(value, bool)
                        or not isinstance(value, (int, float))
                        or not math.isfinite(value)
                        for value in embedding
                    )
                ):
                    raise ValueError(f"Invalid vector or metadata in RAG document {doc_id}")
                connection.execute(
                    """
                    INSERT OR IGNORE INTO rag_documents(
                        chunk_id, document_id, source, chunk_index, text,
                        embedding_json, metadata_json, created_at
                    ) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
                    """,
                    (
                        doc_id,
                        document_id,
                        source,
                        chunk_index,
                        text,
                        _json(embedding, "RAG embedding"),
                        _json(metadata, "RAG metadata"),
                        _utc_now(),
                    ),
                )
                counts["rag_chunks"] += connection.execute(
                    "SELECT changes()"
                ).fetchone()[0]
        return counts
