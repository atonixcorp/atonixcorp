import shutil
import tempfile
import unittest
from pathlib import Path

from ml.anomaly.loader import AnomalyModel, load as load_anomaly
from ml.autoscale.loader import AutoscaleModel, load as load_autoscale
from ml.data.csv_dataset import load_csv, load_numeric_csv
from ml.forecasting.loader import ForecastModel, load as load_forecast
from ml.preprocess.numeric import numeric_rows
from ml.resource_opt.loader import ResourceOptimizer, load as load_resource_optimizer
from ml.utils.metrics import (
    accuracy_score,
    mean_absolute_error,
    root_mean_squared_error,
)
from ml.utils.artifacts import artifact_path, save_artifact
from model.core.loader import AtonixLoader
from model.core.registry import ModelRegistry


AGENT_ROOT = Path(__file__).resolve().parents[1]


class DataAndPreprocessingTests(unittest.TestCase):
    def setUp(self):
        self.temporary_directory = tempfile.TemporaryDirectory()
        self.root = Path(self.temporary_directory.name)

    def tearDown(self):
        self.temporary_directory.cleanup()

    def test_csv_loader_converts_numbers_and_selects_numeric_features(self):
        path = self.root / "metrics.csv"
        path.write_text(
            "cpu_percent,memory_percent,service\n55.5,61.0,api\n70,72.5,worker\n",
            encoding="utf-8",
        )

        rows = load_csv(path)
        features, numeric = load_numeric_csv(path, ["cpu_percent", "memory_percent"])

        self.assertEqual(rows[0]["service"], "api")
        self.assertEqual(features, ["cpu_percent", "memory_percent"])
        self.assertEqual(numeric[1]["memory_percent"], 72.5)

    def test_csv_loader_reports_malformed_rows_and_empty_datasets(self):
        path = self.root / "metrics.csv"
        path.write_text("cpu\n1,2\n", encoding="utf-8")
        with self.assertRaisesRegex(ValueError, "extra CSV values"):
            load_csv(path)

        path.write_text("cpu\n", encoding="utf-8")
        with self.assertRaisesRegex(ValueError, "no data rows"):
            load_csv(path)

    def test_numeric_preprocessing_rejects_missing_and_non_finite_values(self):
        with self.assertRaisesRegex(ValueError, "Missing feature"):
            numeric_rows([{"cpu": 1}, {"memory": 2}], ["cpu"])
        with self.assertRaisesRegex(ValueError, "finite number"):
            numeric_rows([{"cpu": float("nan")}])


class ModelFamilyTests(unittest.TestCase):
    def setUp(self):
        self.temporary_directory = tempfile.TemporaryDirectory()
        self.model_dir = Path(self.temporary_directory.name) / "models"

    def tearDown(self):
        self.temporary_directory.cleanup()

    def test_anomaly_detector_fits_predicts_and_round_trips(self):
        rows = [{"cpu_percent": value} for value in (10, 11, 12, 12, 13, 14)]
        model = AnomalyModel().fit(rows)
        self.assertEqual(model.predict([{"cpu_percent": 12}]), [1])
        self.assertEqual(model.predict([{"cpu_percent": 100}]), [-1])

        model.save("v1", self.model_dir)
        reopened = load_anomaly("v1", self.model_dir)
        self.assertEqual(reopened.predict([{"cpu_percent": 100}]), [-1])

    def test_anomaly_detector_handles_constant_training_data(self):
        model = AnomalyModel().fit([{"value": 5}, {"value": 5}, {"value": 5}])
        self.assertEqual(model.predict([{"value": 5}, {"value": 6}]), [1, -1])

    def test_autoscale_policy_obeys_thresholds_and_replica_bounds(self):
        model = AutoscaleModel()
        decisions = model.predict(
            [
                {"cpu_percent": 95, "memory_percent": 50, "replicas": 2},
                {"cpu_percent": 10, "memory_percent": 20, "replicas": 3},
                {"cpu_percent": 95, "memory_percent": 50, "replicas": 10},
                {"cpu_percent": 10, "replicas": 1},
                {"cpu_percent": 10, "queue_depth": 12, "replicas": 2},
                {"cpu_percent": 10, "queue_depth": 12, "replicas": 10},
            ]
        )
        self.assertEqual(decisions, [1, 2, 0, 0, 1, 0])

        model.save("v1", self.model_dir)
        reopened = load_autoscale("v1", self.model_dir)
        self.assertEqual(reopened.predict({"cpu_percent": 90}), [1])

    def test_autoscale_rejects_invalid_signal_and_configuration(self):
        with self.assertRaisesRegex(ValueError, "between 0 and 100"):
            AutoscaleModel().predict({"cpu_percent": 101})
        with self.assertRaisesRegex(ValueError, "Thresholds"):
            AutoscaleModel(scale_up_threshold=20, scale_down_threshold=30)

    def test_forecaster_predicts_multiple_future_steps_and_round_trips(self):
        model = ForecastModel().fit([1, 3, 5, 7])
        self.assertEqual(model.predict(2), [{"value": 9.0}, {"value": 11.0}])

        model.save("v1", self.model_dir)
        reopened = load_forecast("v1", self.model_dir)
        self.assertEqual(reopened.predict(1), [{"value": 9.0}])

    def test_forecaster_fits_multiple_metrics_and_validates_horizon(self):
        model = ForecastModel().fit(
            [
                {"cpu": 10, "memory": 30},
                {"cpu": 20, "memory": 28},
                {"cpu": 30, "memory": 26},
            ]
        )
        self.assertEqual(model.predict(1), [{"cpu": 40.0, "memory": 24.0}])
        with self.assertRaisesRegex(ValueError, "positive integer"):
            model.predict(0)

    def test_resource_optimizer_sizes_at_p95_and_round_trips(self):
        model = ResourceOptimizer(target_utilization=0.5, headroom=1.2).fit(
            [
                {"cpu_cores": 1, "memory_gib": 2},
                {"cpu_cores": 2, "memory_gib": 4},
                {"cpu_cores": 3, "memory_gib": 6},
            ]
        )
        recommendations = model.predict()
        self.assertAlmostEqual(recommendations["cpu_cores"], 7.2)
        self.assertAlmostEqual(recommendations["memory_gib"], 14.4)
        recent = model.predict({"cpu_cores": 2, "memory_gib": 3})
        self.assertAlmostEqual(recent["cpu_cores"], 4.8)
        self.assertAlmostEqual(recent["memory_gib"], 7.2)

        model.save("v1", self.model_dir)
        reopened = load_resource_optimizer("v1", self.model_dir)
        self.assertEqual(reopened.predict(), model.predict())

    def test_resource_optimizer_rejects_negative_usage(self):
        with self.assertRaisesRegex(ValueError, "non-negative"):
            ResourceOptimizer().fit([{"cpu_cores": -1}])

    def test_artifact_versions_cannot_escape_model_directory(self):
        with self.assertRaisesRegex(ValueError, "Invalid model version"):
            AnomalyModel().fit([{"value": 1}, {"value": 2}]).save("../outside", self.model_dir)

    def test_artifact_version_must_match_requested_version(self):
        path = artifact_path("anomaly", "v1", self.model_dir)
        save_artifact(path, {"family": "anomaly", "version": "v2"})
        with self.assertRaisesRegex(ValueError, "does not match requested version"):
            load_anomaly("v1", self.model_dir)

    def test_evaluation_metrics_validate_inputs(self):
        self.assertEqual(accuracy_score([1, 0, 1], [1, 1, 1]), 2 / 3)
        self.assertEqual(mean_absolute_error([1, 3], [2, 1]), 1.5)
        self.assertAlmostEqual(root_mean_squared_error([1, 3], [2, 1]), (2.5) ** 0.5)
        with self.assertRaisesRegex(ValueError, "same non-zero length"):
            mean_absolute_error([1], [])


class ModelLoaderIntegrationTests(unittest.TestCase):
    def setUp(self):
        self.temporary_directory = tempfile.TemporaryDirectory()
        self.root = Path(self.temporary_directory.name)

    def tearDown(self):
        self.temporary_directory.cleanup()

    def test_agent_loader_resolves_registered_family_and_version(self):
        shutil.copytree(
            AGENT_ROOT / "ml",
            self.root / "ml",
            ignore=shutil.ignore_patterns("__pycache__"),
        )
        registry = ModelRegistry(self.root / "model" / "registry.json")
        registry.register("cpu-forecast", "v1", {"family": "forecasting"})

        ForecastModel().fit([1, 2, 3]).save("v1", self.root / "models")
        loaded = AtonixLoader(self.root).load_ml_model("cpu-forecast")

        self.assertEqual(loaded.predict(1), [{"value": 4.0}])


if __name__ == "__main__":
    unittest.main()
