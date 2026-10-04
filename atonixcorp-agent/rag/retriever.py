"""Query embedding and ranked document retrieval."""

from collections.abc import Mapping
from typing import Any

from .embedder import HashingEmbedder
from .vectorstore import VectorStore


class Retriever:
    """Embed text queries and return their nearest indexed document chunks."""

    def __init__(self, embedder: HashingEmbedder, vectorstore: VectorStore):
        if embedder.dimensions != vectorstore.dimensions:
            raise ValueError("Embedder and vector store dimensions must match")
        self.embedder = embedder
        self.vectorstore = vectorstore

    def retrieve(
        self,
        query: str,
        top_k: int = 5,
        min_score: float = -1.0,
    ) -> list[dict[str, Any]]:
        """Retrieve ranked chunks for a non-empty text query."""
        query_vector = self.embedder.embed(query)
        return self.vectorstore.search(query_vector, top_k=top_k, min_score=min_score)

    @staticmethod
    def format_context(results: list[Mapping[str, Any]]) -> str:
        """Format retrieved chunks with source labels for a downstream LLM."""
        sections = []
        for index, result in enumerate(results, start=1):
            metadata = result.get("metadata", {})
            source = metadata.get("source", result.get("id", "unknown"))
            chunk_index = metadata.get("chunk_index")
            label = f"{source} (chunk {chunk_index})" if chunk_index is not None else str(source)
            sections.append(f"[{index}] Source: {label}\n{result['text']}")
        return "\n\n".join(sections)
