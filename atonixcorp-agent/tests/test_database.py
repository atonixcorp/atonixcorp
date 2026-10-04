import json
import shutil
import tempfile
import unittest
from pathlib import Path

from db.database import Database
from db.schema import discover_migrations


AGENT_ROOT = Path(__file__).resolve().parents[1]
MIGRATIONS = AGENT_ROOT / "db" / "migrations"


class DatabaseMigrationTests(unittest.TestCase):
    def setUp(self):
        self.temporary_directory = tempfile.TemporaryDirectory()
        self.root = Path(self.temporary_directory.name)

    def tearDown(self):
        self.temporary_directory.cleanup()

    def test_all_migrations_apply_once_and_survive_reopen(self):
        database_path = self.root / "agent.sqlite3"
        with Database(database_path) as database:
            first = database.migration_status()
            self.assertEqual(
                [item["version"] for item in first],
                [1, 2, 3, 4],
            )
            self.assertEqual(database.list_models(), [])

        with Database(database_path) as reopened:
            self.assertEqual(reopened.migration_status(), first)

    def test_migration_failure_rolls_back_schema_changes(self):
        migration_dir = self.root / "migrations"
        migration_dir.mkdir()
        (migration_dir / "0001_partial.sql").write_text(
            "CREATE TABLE should_rollback (id INTEGER PRIMARY KEY);\n"
            "THIS IS NOT VALID SQL;\n",
            encoding="utf-8",
        )
        database_path = self.root / "failure.sqlite3"
        with self.assertRaises(Exception):
            Database(database_path, migrations_dir=migration_dir)

        import sqlite3

        connection = sqlite3.connect(database_path)
        try:
            table = connection.execute(
                "SELECT 1 FROM sqlite_master WHERE type='table' AND name='should_rollback'"
            ).fetchone()
            self.assertIsNone(table)
            self.assertEqual(
                connection.execute("SELECT COUNT(*) FROM schema_migrations").fetchone()[0],
                0,
            )
        finally:
            connection.close()

    def test_changed_applied_migration_is_rejected(self):
        migration_dir = self.root / "migrations"
        shutil.copytree(MIGRATIONS, migration_dir)
        database_path = self.root / "checksum.sqlite3"
        with Database(database_path, migrations_dir=migration_dir):
            pass

        migration = migration_dir / "0001_init.sql"
        migration.write_text(
            migration.read_text(encoding="utf-8") + "\n-- changed\n",
            encoding="utf-8",
        )
        with self.assertRaisesRegex(ValueError, "has changed"):
            Database(database_path, migrations_dir=migration_dir)

    def test_discovery_rejects_invalid_migration_names(self):
        migration_dir = self.root / "migrations"
        migration_dir.mkdir()
        (migration_dir / "oops.sql").write_text("SELECT 1;", encoding="utf-8")
        with self.assertRaisesRegex(ValueError, "Invalid migration filename"):
            discover_migrations(migration_dir)


class DatabaseRepositoryTests(unittest.TestCase):
    def setUp(self):
        self.temporary_directory = tempfile.TemporaryDirectory()
        self.path = Path(self.temporary_directory.name) / "agent.sqlite3"
        self.database = Database(self.path)

    def tearDown(self):
        self.database.close()
        self.temporary_directory.cleanup()

    def test_models_versions_promotions_and_duplicate_protection(self):
        first = self.database.register_model(
            "forecast",
            "1.0",
            {"family": "forecasting"},
            created_at="2025-01-01T00:00:00Z",
        )
        self.database.register_model(
            "forecast",
            "2.0",
            {"family": "forecasting"},
            created_at="2025-01-02T00:00:00Z",
        )
        promoted = self.database.promote_model("forecast", "1.0")

        self.assertEqual(first["stage"], "staging")
        self.assertEqual(promoted["stage"], "production")
        self.assertEqual(promoted["promotion_history"][-1]["to"], "production")
        listed = self.database.list_models()[0]
        self.assertEqual(listed["version"], "2.0")
        self.assertEqual(listed["production_version"], "1.0")
        with self.assertRaises(Exception):
            self.database.register_model("forecast", "1.0", {"family": "changed"})

    def test_metrics_are_linked_to_model_versions_and_kept_per_run(self):
        self.database.register_model("forecast", "v1")
        run_id = self.database.start_run("forecast", "v1", run_id="run-1")
        self.database.log_metric(run_id, "baseline_accuracy", 0.8)
        self.database.log_metric(run_id, "current_accuracy", 0.9)

        self.assertEqual(
            self.database.get_recent_metrics("forecast", "v1"),
            {"baseline_accuracy": 0.8, "current_accuracy": 0.9},
        )
        self.assertEqual(
            self.database.list_runs()["run-1"]["version"],
            "v1",
        )
        with self.assertRaisesRegex(ValueError, "finite number"):
            self.database.log_metric(run_id, "broken", float("nan"))
        with self.assertRaises(Exception):
            self.database.start_run("forecast", "missing", run_id="run-1")

    def test_events_incidents_and_actions_persist_audit_records(self):
        event = self.database.log_event("drift_detected", {"model": "forecast"})
        incident = self.database.record_incident(
            "scheduler",
            "node unavailable",
            "workload rescheduled",
            details={"cluster": "test"},
            incident_id="incident-1",
        )
        action = self.database.record_action(
            "restart",
            {"service": "api"},
            status="completed",
            action_id="action-1",
        )
        self.assertEqual(self.database.list_events(limit=10)[0]["id"], "action-1")
        self.assertEqual(self.database.list_events("incident")[0]["id"], incident["id"])
        self.assertEqual(self.database.list_incidents("scheduler")[0]["details"]["cluster"], "test")
        self.assertEqual(self.database.list_actions()[0]["status"], "completed")
        self.assertEqual(event["type"], "drift_detected")
        self.assertEqual(action["action"], "restart")

    def test_rag_chunks_upsert_list_and_delete(self):
        self.database.upsert_rag_document(
            "chunk-1",
            "doc-1",
            "runbook.md",
            0,
            "node failure response",
            [0.1, 0.2, 0.3],
            {"kind": "runbook"},
        )
        listed = self.database.list_rag_documents("doc-1")
        self.assertEqual(listed[0]["embedding"], [0.1, 0.2, 0.3])
        self.assertEqual(listed[0]["metadata"]["kind"], "runbook")
        self.assertEqual(self.database.delete_rag_document("doc-1"), 1)
        self.assertEqual(self.database.list_rag_documents(), [])

    def test_nested_transactions_use_savepoints(self):
        with self.database.transaction() as connection:
            connection.execute(
                "INSERT INTO events(id, event_type, payload_json, created_at) "
                "VALUES ('outer', 'test', '{}', 'now')"
            )
            try:
                with self.database.transaction() as nested:
                    nested.execute(
                        "INSERT INTO events(id, event_type, payload_json, created_at) "
                        "VALUES ('inner', 'test', '{}', 'now')"
                    )
                    raise RuntimeError("rollback nested work")
            except RuntimeError:
                pass
        self.assertEqual([item["id"] for item in self.database.list_events()], ["outer"])

    def test_json_import_is_atomic_and_idempotent(self):
        legacy_root = self.path.parent / "legacy"
        (legacy_root / "model").mkdir(parents=True)
        (legacy_root / "db").mkdir()
        (legacy_root / "model" / "registry.json").write_text(
            json.dumps(
                {
                    "models": {
                        "forecast": {
                            "version": "v1",
                            "created_at": "2025-01-01T00:00:00Z",
                            "stage": "production",
                            "metadata": {"family": "forecasting"},
                            "production_version": "v1",
                            "versions": {
                                "v1": {
                                    "version": "v1",
                                    "created_at": "2025-01-01T00:00:00Z",
                                    "stage": "production",
                                    "metadata": {"family": "forecasting"},
                                    "promotion_history": [],
                                }
                            },
                        }
                    }
                }
            ),
            encoding="utf-8",
        )
        (legacy_root / "db" / "metrics.json").write_text(
            json.dumps(
                {
                    "runs": {
                        "run-legacy": {
                            "model": "forecast",
                            "version": "v1",
                            "metrics": {"accuracy": 0.91},
                            "created_at": "2025-01-02T00:00:00Z",
                            "updated_at": "2025-01-02T00:00:00Z",
                        }
                    }
                }
            ),
            encoding="utf-8",
        )
        (legacy_root / "db" / "events.json").write_text(
            json.dumps(
                {
                    "events": [
                        {
                            "id": "event-legacy",
                            "type": "incident",
                            "payload": {
                                "service": "scheduler",
                                "root_cause": "node lost",
                                "resolution": "rescheduled",
                            },
                            "timestamp": "2025-01-03T00:00:00Z",
                        }
                    ]
                }
            ),
            encoding="utf-8",
        )
        (legacy_root / "db" / "rag_index.json").write_text(
            json.dumps(
                {
                    "documents": [
                        {
                            "id": "chunk-legacy",
                            "text": "runbook content",
                            "embedding": [0.1, 0.2],
                            "metadata": {
                                "source": "runbook.md",
                                "document_id": "doc-legacy",
                                "chunk_index": 0,
                            },
                        }
                    ]
                }
            ),
            encoding="utf-8",
        )

        first = self.database.import_legacy_json(legacy_root)
        second = self.database.import_legacy_json(legacy_root)
        self.assertEqual(
            first,
            {
                "models": 1,
                "model_versions": 1,
                "runs": 1,
                "metrics": 1,
                "events": 1,
                "rag_chunks": 1,
            },
        )
        self.assertEqual(
            second,
            {
                "models": 0,
                "model_versions": 0,
                "runs": 0,
                "metrics": 0,
                "events": 0,
                "rag_chunks": 0,
            },
        )
        self.assertEqual(self.database.get_recent_metrics("forecast", "v1"), {"accuracy": 0.91})
        self.assertEqual(self.database.list_incidents()[0]["service"], "scheduler")
        self.assertEqual(self.database.list_rag_documents()[0]["id"], "chunk-legacy")

    def test_json_import_rolls_back_everything_on_invalid_record(self):
        legacy_root = self.path.parent / "legacy"
        (legacy_root / "model").mkdir(parents=True)
        (legacy_root / "db").mkdir()
        (legacy_root / "model" / "registry.json").write_text(
            json.dumps(
                {
                    "models": {
                        "valid": {
                            "version": "1",
                            "metadata": {},
                            "versions": {
                                "1": {"stage": "staging", "metadata": {}}
                            },
                        }
                    }
                }
            ),
            encoding="utf-8",
        )
        (legacy_root / "db" / "metrics.json").write_text(
            json.dumps(
                {
                    "runs": {
                        "bad": {
                            "model": "valid",
                            "version": "1",
                            "metrics": {"accuracy": float("nan")},
                        }
                    }
                }
            ),
            encoding="utf-8",
        )
        with self.assertRaisesRegex(ValueError, "Invalid metric"):
            self.database.import_legacy_json(legacy_root)
        self.assertEqual(self.database.list_models(), [])
        self.assertEqual(self.database.list_runs(), {})


if __name__ == "__main__":
    unittest.main()
