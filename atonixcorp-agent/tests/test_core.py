import json
import tempfile
import threading
import unittest
from pathlib import Path

from model.core.events import EventBus
from model.core.metrics import MetricsTracker
from model.core.registry import ModelRegistry
from model.core.supervisor import Supervisor


class CorePersistenceTests(unittest.TestCase):
    def setUp(self):
        self.temporary_directory = tempfile.TemporaryDirectory()
        self.root = Path(self.temporary_directory.name)

    def tearDown(self):
        self.temporary_directory.cleanup()

    def test_registry_keeps_versions_and_promotion_history(self):
        registry = ModelRegistry(self.root / "model" / "registry.json")
        registry.register("forecast", "1.0", {"family": "forecasting"})
        registry.register("forecast", "2.0", {"family": "forecasting"})
        registry.promote("forecast", version="1.0")

        reopened = ModelRegistry(self.root / "model" / "registry.json")
        self.assertEqual(reopened.get("forecast")["version"], "2.0")
        self.assertEqual(reopened.get("forecast", "1.0")["stage"], "production")
        self.assertEqual(
            reopened.get("forecast", "1.0")["promotion_history"][-1]["to"],
            "production",
        )
        self.assertEqual(reopened.list()[0]["production_version"], "1.0")

    def test_registry_migrates_existing_single_version_records(self):
        path = self.root / "registry.json"
        path.write_text(
            json.dumps(
                {
                    "models": {
                        "legacy": {
                            "name": "legacy",
                            "version": "1",
                            "created_at": "2025-01-01T00:00:00+00:00",
                            "stage": "production",
                            "metadata": {},
                        }
                    }
                }
            ),
            encoding="utf-8",
        )

        registry = ModelRegistry(path)
        self.assertEqual(registry.get("legacy", "1")["stage"], "production")
        migrated = json.loads(path.read_text(encoding="utf-8"))
        self.assertIn("versions", migrated["models"]["legacy"])

    def test_registry_rejects_duplicate_versions_without_overwriting(self):
        registry = ModelRegistry(self.root / "registry.json")
        registry.register("model", "1", {"purpose": "test"})

        with self.assertRaisesRegex(ValueError, "already registered"):
            registry.register("model", "1", {"purpose": "changed"})

        self.assertEqual(registry.get("model")["metadata"]["purpose"], "test")

    def test_metrics_are_persisted_and_reject_invalid_values(self):
        path = self.root / "db" / "metrics.json"
        tracker = MetricsTracker(path)
        run_id = tracker.start_run("forecast", "1")
        tracker.log_metric(run_id, "baseline_accuracy", 0.9)
        tracker.log_metric(run_id, "current_accuracy", 0.95)

        with self.assertRaisesRegex(ValueError, "finite number"):
            tracker.log_metric(run_id, "invalid", float("nan"))

        reopened = MetricsTracker(path)
        self.assertAlmostEqual(reopened.compute_improvement("forecast"), 0.05)
        self.assertEqual(
            reopened.list_runs("forecast")[run_id]["metrics"]["current_accuracy"],
            0.95,
        )

    def test_event_bus_persists_filters_and_returns_isolated_payloads(self):
        path = self.root / "db" / "events.json"
        bus = EventBus(path)
        payload = {"service": "api"}
        event = bus.log("incident", payload)
        payload["service"] = "mutated"
        event["payload"]["service"] = "also-mutated"
        bus.log("model_promoted", {"model": "forecast"})

        reopened = EventBus(path)
        self.assertEqual(reopened.list("incident")[0]["payload"]["service"], "api")
        self.assertEqual(len(reopened.recent(limit=1)), 1)
        self.assertEqual(reopened.recent(limit=0), [])
        with self.assertRaisesRegex(ValueError, "non-negative integer"):
            reopened.recent(limit=-1)

    def test_corrupt_persistence_is_reported(self):
        path = self.root / "events.json"
        path.write_text("{invalid", encoding="utf-8")

        with self.assertRaisesRegex(ValueError, "Invalid JSON"):
            EventBus(path)


class SupervisorTests(unittest.TestCase):
    def setUp(self):
        self.temporary_directory = tempfile.TemporaryDirectory()
        root = Path(self.temporary_directory.name)
        self.registry = ModelRegistry(root / "registry.json")
        self.registry.register("forecast", "1", {"family": "forecasting"})
        self.metrics = MetricsTracker(root / "metrics.json")
        run_id = self.metrics.start_run("forecast", "1")
        self.metrics.log_metric(run_id, "baseline_accuracy", 0.95)
        self.metrics.log_metric(run_id, "current_accuracy", 0.70)
        self.events = EventBus(root / "events.json")
        self.supervisor = Supervisor(
            registry=self.registry,
            metrics=self.metrics,
            events=self.events,
        )

    def tearDown(self):
        self.temporary_directory.cleanup()

    def test_evaluation_emits_drift_and_performance_signals_only(self):
        results = self.supervisor.run_once()

        self.assertEqual(
            results,
            [
                {
                    "model": "forecast",
                    "drift": True,
                    "performance_drop": True,
                    "status": "signals_detected",
                }
            ],
        )
        self.assertEqual(
            [event["type"] for event in self.events.list()],
            ["drift_detected", "supervisor_signal"],
        )
        self.supervisor.run_once()
        self.assertEqual(len(self.events.list()), 2)
        self.assertEqual(self.registry.get("forecast")["version"], "1")

    def test_retraining_requires_explicit_configured_trainer(self):
        with self.assertRaisesRegex(RuntimeError, "No retrainer is configured"):
            self.supervisor.retrain("forecast")
        self.assertEqual(self.registry.get("forecast")["version"], "1")

    def test_configured_retrainer_registers_new_version(self):
        supervisor = Supervisor(
            registry=self.registry,
            metrics=self.metrics,
            events=self.events,
            retrainer=lambda name, entry: "2",
        )

        self.assertEqual(supervisor.retrain("forecast"), "2")
        self.assertEqual(self.registry.get("forecast")["version"], "2")
        self.assertEqual(self.events.list()[-1]["type"], "model_retrained")

    def test_auto_promotion_uses_measured_improvement(self):
        run_id = self.metrics.start_run("forecast", "2")
        self.metrics.log_metric(run_id, "baseline_accuracy", 0.7)
        self.metrics.log_metric(run_id, "current_accuracy", 0.9)
        self.registry.register("forecast", "2", {"family": "forecasting"})

        self.assertTrue(self.supervisor.auto_promote("forecast"))
        self.assertEqual(self.registry.get("forecast")["stage"], "production")
        self.assertEqual(self.events.list()[-1]["type"], "model_promoted")

    def test_auto_promotion_does_not_use_metrics_from_an_older_version(self):
        self.registry.register("forecast", "2", {"family": "forecasting"})

        self.assertFalse(self.supervisor.auto_promote("forecast"))
        self.assertEqual(self.registry.get("forecast")["stage"], "staging")
        self.assertEqual(self.supervisor.evaluate_model("forecast")["status"], "insufficient_metrics")

    def test_supervisor_run_can_stop_without_starting_a_cycle(self):
        stop_event = threading.Event()
        stop_event.set()

        self.supervisor.run(stop_event=stop_event)
        self.assertEqual(self.events.list(), [])


if __name__ == "__main__":
    unittest.main()
