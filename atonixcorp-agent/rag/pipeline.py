"""Document ingestion and retrieval-augmented context assembly."""

import hashlib
import json
import uuid
from collections.abc import Iterable, Mapping
from pathlib import Path
from typing import Any

from .embedder import HashingEmbedder
from .retriever import Retriever
from .vectorstore import DEFAULT_INDEX_PATH, VectorStore

_SUPPORTED_EXTENSIONS = {".md", ".txt"}


class RAGPipeline:
    """Chunk, index, and retrieve local text for grounded downstream answers.

    This pipeline assembles cited context; it does not call an LLM or generate
    answers by itself.
    """

    def __init__(
        self,
        index_path: str | Path = DEFAULT_INDEX_PATH,
        dimensions: int = 512,
        chunk_size: int = 180,
        chunk_overlap: int = 30,
        min_score: float = 0.05,
    ):
        if isinstance(chunk_size, bool) or not isinstance(chunk_size, int) or chunk_size < 1:
            raise ValueError("chunk_size must be a positive integer")
        if (
            isinstance(chunk_overlap, bool)
            or not isinstance(chunk_overlap, int)
            or chunk_overlap < 0
            or chunk_overlap >= chunk_size
        ):
            raise ValueError("chunk_overlap must be a non-negative integer smaller than chunk_size")
        if isinstance(min_score, bool) or not isinstance(min_score, (int, float)):
            raise ValueError("min_score must be a finite number between -1 and 1")
        if not -1 <= float(min_score) <= 1:
            raise ValueError("min_score must be a finite number between -1 and 1")

        self.chunk_size = chunk_size
        self.chunk_overlap = chunk_overlap
        self.min_score = float(min_score)
        self.embedder = HashingEmbedder(dimensions=dimensions)
        self.vectorstore = VectorStore(index_path, dimensions=dimensions)
        self.retriever = Retriever(self.embedder, self.vectorstore)

    def _chunks(self, text: str) -> list[str]:
        if not isinstance(text, str) or not text.strip():
            raise ValueError("Document text must be a non-empty string")
        words = text.split()
        if not words:
            raise ValueError("Document text must contain searchable words")
        chunks = []
        step = self.chunk_size - self.chunk_overlap
        for start in range(0, len(words), step):
            chunk = " ".join(words[start : start + self.chunk_size])
            if chunk:
                chunks.append(chunk)
            if start + self.chunk_size >= len(words):
                break
        return chunks

    def ingest_text(
        self,
        text: str,
        source: str | None = None,
        metadata: Mapping[str, Any] | None = None,
        document_id: str | None = None,
    ) -> list[dict[str, Any]]:
        """Chunk, embed, and persist one text document."""
        if metadata is None:
            document_metadata: dict[str, Any] = {}
        elif isinstance(metadata, Mapping):
            document_metadata = dict(metadata)
        else:
            raise ValueError("metadata must be a mapping")
        source_label = "inline" if source is None else source
        if not isinstance(source_label, str) or not source_label.strip():
            raise ValueError("source must be a non-empty string")
        if document_id is None:
            document_id = uuid.uuid4().hex
        if not isinstance(document_id, str) or not document_id.strip():
            raise ValueError("document_id must be a non-empty string")

        chunks = self._chunks(text)
        try:
            json.dumps(document_metadata, allow_nan=False)
        except (TypeError, ValueError) as error:
            raise ValueError("metadata must contain only finite JSON values") from error
        embedded_chunks = [(chunk, self.embedder.embed(chunk)) for chunk in chunks]
        logical_id = document_id.strip()
        old_ids = {
            document["id"]
            for document in self.vectorstore.list_documents()
            if document["metadata"].get("document_id") == logical_id
        }

        documents = []
        for chunk_index, (chunk, embedding) in enumerate(embedded_chunks):
            chunk_id = hashlib.sha256(
                f"{logical_id}:{chunk_index}:{chunk}".encode("utf-8")
            ).hexdigest()
            chunk_metadata = {
                **document_metadata,
                "source": source_label.strip(),
                "document_id": logical_id,
                "chunk_index": chunk_index,
            }
            documents.append(
                {
                    "id": chunk_id,
                    "text": chunk,
                    "embedding": embedding,
                    "metadata": chunk_metadata,
                }
            )
        stored = self.vectorstore.upsert_many(documents)
        current_ids = {document["id"] for document in stored}
        self.vectorstore.delete_many(old_ids - current_ids)
        return stored

    def ingest_documents(
        self,
        documents: Iterable[Mapping[str, Any]],
    ) -> list[dict[str, Any]]:
        """Ingest mappings containing text and optional source/metadata/id."""
        if isinstance(documents, (str, bytes)):
            raise ValueError("documents must be an iterable of document mappings")
        ingested = []
        for index, document in enumerate(documents):
            if not isinstance(document, Mapping):
                raise ValueError(f"Document at index {index} must be a mapping")
            if "text" not in document:
                raise ValueError(f"Document at index {index} is missing text")
            ingested.extend(
                self.ingest_text(
                    document["text"],
                    source=document.get("source"),
                    metadata=document.get("metadata"),
                    document_id=document.get("id"),
                )
            )
        return ingested

    def ingest_file(
        self,
        path: str | Path,
        metadata: Mapping[str, Any] | None = None,
    ) -> list[dict[str, Any]]:
        """Read and ingest a UTF-8 Markdown or plain-text file."""
        file_path = Path(path)
        if file_path.suffix.lower() not in _SUPPORTED_EXTENSIONS:
            raise ValueError("Only .md and .txt files can be ingested")
        text = file_path.read_text(encoding="utf-8")
        source = str(file_path.resolve())
        document_id = hashlib.sha256(source.encode("utf-8")).hexdigest()
        return self.ingest_text(
            text,
            source=source,
            metadata=metadata,
            document_id=document_id,
        )

    def ingest_directory(
        self,
        path: str | Path,
        metadata: Mapping[str, Any] | None = None,
    ) -> list[dict[str, Any]]:
        """Recursively ingest UTF-8 Markdown and text files in sorted order."""
        directory = Path(path)
        if not directory.is_dir():
            raise NotADirectoryError(f"Document directory does not exist: {directory}")
        ingested = []
        for file_path in sorted(directory.rglob("*")):
            if file_path.is_file() and file_path.suffix.lower() in _SUPPORTED_EXTENSIONS:
                ingested.extend(self.ingest_file(file_path, metadata=metadata))
        return ingested

    def retrieve(self, query: str, top_k: int = 5) -> list[dict[str, Any]]:
        """Retrieve the highest-scoring indexed chunks for a text query."""
        return self.retriever.retrieve(query, top_k=top_k, min_score=self.min_score)

    def query(self, query: str, top_k: int = 5) -> dict[str, Any]:
        """Return retrieved documents and source-labeled context for an LLM."""
        results = self.retrieve(query, top_k=top_k)
        sources = []
        seen_sources: set[str] = set()
        for result in results:
            source = str(result["metadata"].get("source", result["id"]))
            if source not in seen_sources:
                seen_sources.add(source)
                sources.append(source)
        return {
            "query": query,
            "context": self.retriever.format_context(results),
            "sources": sources,
            "results": results,
        }

    def delete_document(self, document_id: str) -> int:
        """Delete all stored chunks belonging to a logical document."""
        matches = {
            document["id"]
            for document in self.vectorstore.list_documents()
            if document["metadata"].get("document_id") == document_id
        }
        return self.vectorstore.delete_many(matches)

    def count(self) -> int:
        """Return the number of indexed chunks."""
        return len(self.vectorstore.list_documents())


def load_pipeline(
    index_path: str | Path = DEFAULT_INDEX_PATH,
    dimensions: int = 512,
    chunk_size: int = 180,
    chunk_overlap: int = 30,
    min_score: float = 0.05,
) -> RAGPipeline:
    """Create the configured local RAG pipeline."""
    return RAGPipeline(
        index_path=index_path,
        dimensions=dimensions,
        chunk_size=chunk_size,
        chunk_overlap=chunk_overlap,
        min_score=min_score,
    )
