import tempfile
import unittest
from pathlib import Path

from model.core.loader import AtonixLoader
from rag.embedder import HashingEmbedder
from rag.pipeline import RAGPipeline
from rag.vectorstore import VectorStore


AGENT_ROOT = Path(__file__).resolve().parents[1]


class EmbedderTests(unittest.TestCase):
    def test_embedding_is_deterministic_normalized_and_dimensioned(self):
        embedder = HashingEmbedder(dimensions=128)
        first = embedder.embed("GPU cluster capacity")
        second = embedder.embed("GPU cluster capacity")

        self.assertEqual(first, second)
        self.assertEqual(len(first), 128)
        self.assertAlmostEqual(sum(value * value for value in first), 1.0)
        self.assertNotEqual(first, embedder.embed("database backups"))

    def test_embedder_rejects_invalid_configuration_and_text(self):
        with self.assertRaisesRegex(ValueError, "at least 16"):
            HashingEmbedder(dimensions=8)
        with self.assertRaisesRegex(ValueError, "searchable tokens"):
            HashingEmbedder().embed("!!!")


class VectorStoreTests(unittest.TestCase):
    def setUp(self):
        self.temporary_directory = tempfile.TemporaryDirectory()
        self.index_path = Path(self.temporary_directory.name) / "index.json"

    def tearDown(self):
        self.temporary_directory.cleanup()

    def test_persistence_upsert_delete_and_isolated_values(self):
        store = VectorStore(self.index_path, dimensions=16)
        stored = store.upsert("doc-1", "cluster capacity", [1.0] + [0.0] * 15, {"tag": "ops"})
        stored["metadata"]["tag"] = "mutated"
        reopened = VectorStore(self.index_path, dimensions=16)

        self.assertEqual(reopened.get("doc-1")["metadata"]["tag"], "ops")
        self.assertEqual(len(reopened.list_documents()), 1)
        self.assertTrue(reopened.delete("doc-1"))
        self.assertFalse(reopened.delete("doc-1"))
        self.assertEqual(VectorStore(self.index_path, dimensions=16).list_documents(), [])

    def test_search_ranks_by_cosine_and_validates_queries(self):
        store = VectorStore(self.index_path, dimensions=16)
        store.upsert("match", "match", [1.0] + [0.0] * 15)
        store.upsert("orthogonal", "other", [0.0, 1.0] + [0.0] * 14)

        results = store.search([2.0] + [0.0] * 15, top_k=2, min_score=0)
        self.assertEqual([item["id"] for item in results], ["match", "orthogonal"])
        self.assertEqual(results[0]["score"], 1.0)
        with self.assertRaisesRegex(ValueError, "non-zero magnitude"):
            store.search([0.0] * 16)
        with self.assertRaisesRegex(ValueError, "positive integer"):
            store.search([1.0] + [0.0] * 15, top_k=0)

    def test_store_rejects_dimension_mismatch_and_corrupt_json(self):
        store = VectorStore(self.index_path, dimensions=16)
        store.upsert("doc", "data", [1.0] + [0.0] * 15)
        with self.assertRaisesRegex(ValueError, "Invalid RAG index structure"):
            VectorStore(self.index_path, dimensions=32)

        self.index_path.write_text("{bad", encoding="utf-8")
        with self.assertRaisesRegex(ValueError, "Invalid RAG index JSON"):
            VectorStore(self.index_path, dimensions=16)


class RAGPipelineTests(unittest.TestCase):
    def setUp(self):
        self.temporary_directory = tempfile.TemporaryDirectory()
        self.root = Path(self.temporary_directory.name)
        self.pipeline = RAGPipeline(
            self.root / "index.json",
            dimensions=128,
            chunk_size=8,
            chunk_overlap=2,
            min_score=0,
        )

    def tearDown(self):
        self.temporary_directory.cleanup()

    def test_ingest_search_citations_and_reload(self):
        chunks = self.pipeline.ingest_text(
            "Kubernetes nodes provide compute capacity. GPU nodes schedule machine learning workloads.",
            source="runbook.md",
            metadata={"category": "compute"},
            document_id="compute-runbook",
        )
        self.assertGreater(len(chunks), 1)
        self.assertEqual(self.pipeline.count(), len(chunks))

        result = self.pipeline.query("GPU workloads on Kubernetes nodes", top_k=2)
        self.assertEqual(result["query"], "GPU workloads on Kubernetes nodes")
        self.assertTrue(result["results"])
        self.assertIn("Source: runbook.md", result["context"])
        self.assertEqual(result["sources"], ["runbook.md"])

        reloaded = RAGPipeline(
            self.root / "index.json",
            dimensions=128,
            chunk_size=8,
            chunk_overlap=2,
            min_score=0,
        )
        self.assertEqual(reloaded.count(), len(chunks))
        self.assertEqual(reloaded.query("GPU workloads")["sources"], ["runbook.md"])

    def test_ingest_same_document_replaces_old_chunks(self):
        self.pipeline.ingest_text(
            "old content " * 18,
            source="notes.txt",
            document_id="replace-me",
        )
        self.assertGreater(self.pipeline.count(), 1)

        replacement = self.pipeline.ingest_text(
            "new replacement content",
            source="notes.txt",
            document_id="replace-me",
        )
        self.assertEqual(self.pipeline.count(), 1)
        self.assertEqual(self.pipeline.vectorstore.list_documents()[0]["text"], replacement[0]["text"])

    def test_file_and_directory_ingestion_support_text_formats(self):
        docs = self.root / "docs"
        docs.mkdir()
        (docs / "one.md").write_text("Operational incident response guide.", encoding="utf-8")
        (docs / "two.txt").write_text("Cloud networking troubleshooting notes.", encoding="utf-8")
        (docs / "skip.csv").write_text("not indexed", encoding="utf-8")

        indexed = self.pipeline.ingest_directory(docs)
        self.assertEqual(len(indexed), 2)
        self.assertEqual(
            {item["metadata"]["source"].split("/")[-1] for item in indexed},
            {"one.md", "two.txt"},
        )
        with self.assertRaisesRegex(ValueError, "Only .md and .txt"):
            self.pipeline.ingest_file(docs / "skip.csv")

    def test_input_validation_and_logical_document_deletion(self):
        with self.assertRaisesRegex(ValueError, "chunk_overlap"):
            RAGPipeline(self.root / "other.json", chunk_size=10, chunk_overlap=10)
        with self.assertRaisesRegex(ValueError, "Document text"):
            self.pipeline.ingest_text(" ")

        self.pipeline.ingest_text("system incident details", document_id="incident-1")
        self.assertEqual(self.pipeline.delete_document("incident-1"), 1)
        self.assertEqual(self.pipeline.count(), 0)

    def test_unified_loader_uses_agent_local_rag_index(self):
        loaded = AtonixLoader(AGENT_ROOT).load_rag_pipeline()
        self.assertEqual(loaded.__class__.__name__, RAGPipeline.__name__)
        self.assertTrue(callable(loaded.query))
        self.assertEqual(loaded.vectorstore.path, AGENT_ROOT / "db" / "rag_index.json")


if __name__ == "__main__":
    unittest.main()
