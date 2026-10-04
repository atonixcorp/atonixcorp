"""Atomic JSON vector store for the agent's local RAG index."""

import json
import math
import os
import tempfile
import threading
from copy import deepcopy
from collections.abc import Iterable, Mapping
from pathlib import Path
from typing import Any

AGENT_ROOT = Path(__file__).resolve().parents[1]
DEFAULT_INDEX_PATH = AGENT_ROOT / "db" / "rag_index.json"


def _finite_vector(value: Any, dimensions: int, field: str) -> list[float]:
    if not isinstance(value, list) or len(value) != dimensions:
        raise ValueError(f"{field} must be a vector with {dimensions} values")
    vector: list[float] = []
    for item in value:
        if isinstance(item, bool) or not isinstance(item, (int, float)):
            raise ValueError(f"{field} values must be finite numbers")
        number = float(item)
        if not math.isfinite(number):
            raise ValueError(f"{field} values must be finite numbers")
        vector.append(number)
    return vector


class VectorStore:
    """Small single-process vector store persisted as an atomic JSON document."""

    def __init__(self, path: str | Path = DEFAULT_INDEX_PATH, dimensions: int = 512):
        if isinstance(dimensions, bool) or not isinstance(dimensions, int) or dimensions < 1:
            raise ValueError("dimensions must be a positive integer")
        self.path = Path(path)
        self.dimensions = dimensions
        self._lock = threading.RLock()
        self._documents: dict[str, dict[str, Any]] = {}
        self._load()

    def _load(self) -> None:
        if not self.path.exists():
            return
        try:
            with self.path.open("r", encoding="utf-8") as handle:
                payload = json.load(handle)
        except json.JSONDecodeError as error:
            raise ValueError(f"Invalid RAG index JSON in {self.path}") from error

        if (
            not isinstance(payload, dict)
            or payload.get("schema_version") != 1
            or payload.get("dimensions") != self.dimensions
            or not isinstance(payload.get("documents"), list)
        ):
            raise ValueError(f"Invalid RAG index structure in {self.path}")

        documents: dict[str, dict[str, Any]] = {}
        for item in payload["documents"]:
            if not isinstance(item, dict):
                raise ValueError(f"Invalid document entry in {self.path}")
            doc_id = item.get("id")
            text = item.get("text")
            metadata = item.get("metadata")
            if (
                not isinstance(doc_id, str)
                or not doc_id
                or doc_id in documents
                or not isinstance(text, str)
                or not text.strip()
                or not isinstance(metadata, dict)
            ):
                raise ValueError(f"Invalid document fields in {self.path}")
            try:
                json.dumps(metadata, allow_nan=False)
            except (TypeError, ValueError) as error:
                raise ValueError(f"Invalid document metadata in {self.path}") from error
            documents[doc_id] = {
                "id": doc_id,
                "text": text,
                "metadata": deepcopy(metadata),
                "embedding": _finite_vector(
                    item.get("embedding"),
                    self.dimensions,
                    f"embedding for {doc_id}",
                ),
            }
        self._documents = documents

    def _save(self) -> None:
        self.path.parent.mkdir(parents=True, exist_ok=True)
        payload = {
            "schema_version": 1,
            "dimensions": self.dimensions,
            "documents": list(self._documents.values()),
        }
        serialized = json.dumps(payload, allow_nan=False, separators=(",", ":"))
        temporary_path: Path | None = None
        try:
            with tempfile.NamedTemporaryFile(
                mode="w",
                encoding="utf-8",
                dir=self.path.parent,
                prefix=f".{self.path.name}.",
                suffix=".tmp",
                delete=False,
            ) as handle:
                temporary_path = Path(handle.name)
                handle.write(serialized)
                handle.flush()
                os.fsync(handle.fileno())
            os.replace(temporary_path, self.path)
        except OSError:
            if temporary_path is not None:
                temporary_path.unlink(missing_ok=True)
            raise

    def upsert(
        self,
        doc_id: str,
        text: str,
        embedding: list[float],
        metadata: dict[str, Any] | None = None,
    ) -> dict[str, Any]:
        """Insert or replace a document and persist the updated index."""
        return self.upsert_many(
            [
                {
                    "id": doc_id,
                    "text": text,
                    "embedding": embedding,
                    "metadata": metadata if metadata is not None else {},
                }
            ]
        )[0]

    def upsert_many(
        self,
        documents: Iterable[Mapping[str, Any]],
    ) -> list[dict[str, Any]]:
        """Atomically insert or replace several documents in one index write."""
        prepared: list[dict[str, Any]] = []
        for index, item in enumerate(documents):
            if not isinstance(item, Mapping):
                raise ValueError(f"Document at index {index} must be a mapping")
            doc_id = item.get("id")
            text = item.get("text")
            metadata = item.get("metadata", {})
            if not isinstance(doc_id, str) or not doc_id.strip():
                raise ValueError(f"doc_id at index {index} must be a non-empty string")
            if not isinstance(text, str) or not text.strip():
                raise ValueError(f"text at index {index} must be a non-empty string")
            if not isinstance(metadata, dict):
                raise ValueError(f"metadata at index {index} must be a dictionary")
            try:
                json.dumps(metadata, allow_nan=False)
            except (TypeError, ValueError) as error:
                raise ValueError(
                    f"metadata at index {index} must contain only finite JSON values"
                ) from error
            prepared.append(
                {
                    "id": doc_id.strip(),
                    "text": text,
                    "metadata": deepcopy(metadata),
                    "embedding": _finite_vector(
                        item.get("embedding"),
                        self.dimensions,
                        f"embedding at index {index}",
                    ),
                }
            )
        if not prepared:
            return []

        with self._lock:
            previous = {
                document["id"]: self._documents.get(document["id"])
                for document in prepared
            }
            for document in prepared:
                self._documents[document["id"]] = document
            try:
                self._save()
            except OSError:
                for doc_id, old_document in previous.items():
                    if old_document is None:
                        self._documents.pop(doc_id, None)
                    else:
                        self._documents[doc_id] = old_document
                raise
        return [deepcopy(document) for document in prepared]

    def delete(self, doc_id: str) -> bool:
        """Delete a document by ID, returning whether it existed."""
        return bool(self.delete_many([doc_id]))

    def delete_many(self, doc_ids: Iterable[str]) -> int:
        """Atomically delete multiple document IDs in one index write."""
        with self._lock:
            ids = set(doc_ids)
            previous = {
                doc_id: self._documents[doc_id]
                for doc_id in ids
                if doc_id in self._documents
            }
            if not previous:
                return 0
            for doc_id in previous:
                del self._documents[doc_id]
            try:
                self._save()
            except OSError:
                self._documents.update(previous)
                raise
            return len(previous)

    def get(self, doc_id: str) -> dict[str, Any] | None:
        """Return an isolated copy of a stored document."""
        with self._lock:
            document = self._documents.get(doc_id)
            return deepcopy(document) if document is not None else None

    def list_documents(self) -> list[dict[str, Any]]:
        """Return all documents without exposing mutable internal state."""
        with self._lock:
            return [deepcopy(document) for document in self._documents.values()]

    def search(
        self,
        query_embedding: list[float],
        top_k: int = 5,
        min_score: float = -1.0,
    ) -> list[dict[str, Any]]:
        """Return cosine-ranked documents, highest score first."""
        if isinstance(top_k, bool) or not isinstance(top_k, int) or top_k < 1:
            raise ValueError("top_k must be a positive integer")
        if isinstance(min_score, bool) or not isinstance(min_score, (int, float)):
            raise ValueError("min_score must be a finite number between -1 and 1")
        threshold = float(min_score)
        if not math.isfinite(threshold) or not -1 <= threshold <= 1:
            raise ValueError("min_score must be a finite number between -1 and 1")
        query = _finite_vector(query_embedding, self.dimensions, "query_embedding")
        query_norm = math.sqrt(sum(value * value for value in query))
        if query_norm == 0:
            raise ValueError("query_embedding must have a non-zero magnitude")

        with self._lock:
            scored = []
            for document in self._documents.values():
                vector = document["embedding"]
                vector_norm = math.sqrt(sum(value * value for value in vector))
                score = (
                    sum(left * right for left, right in zip(query, vector))
                    / (query_norm * vector_norm)
                    if vector_norm
                    else 0.0
                )
                if score >= threshold:
                    result = deepcopy(document)
                    result["score"] = score
                    scored.append(result)
        scored.sort(key=lambda document: (-document["score"], document["id"]))
        return scored[:top_k]
