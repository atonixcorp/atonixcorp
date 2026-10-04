"""Read-only search tool for the local RAG index."""

from pathlib import Path
from typing import Any

from rag.pipeline import RAGPipeline


def search_knowledge(
    root: Path,
    query: str,
    top_k: int = 5,
) -> dict[str, Any]:
    """Search the agent-local RAG index and return context with source citations."""
    if not isinstance(query, str) or not query.strip():
        raise ValueError("query must be a non-empty string")
    if isinstance(top_k, bool) or not isinstance(top_k, int) or not 1 <= top_k <= 20:
        raise ValueError("top_k must be between 1 and 20")
    pipeline = RAGPipeline(index_path=root / "db" / "rag_index.json")
    return pipeline.query(query.strip(), top_k=top_k)


def register_rag_tools(server: Any, root: Path) -> None:
    """Register local retrieval without exposing document mutation operations."""
    server.register_tool(
        "rag.search",
        "Search indexed local knowledge and return matching text chunks with sources.",
        {
            "type": "object",
            "properties": {
                "query": {"type": "string", "description": "Question or search terms."},
                "top_k": {"type": "integer", "minimum": 1, "maximum": 20},
            },
            "required": ["query"],
            "additionalProperties": False,
        },
        lambda arguments: search_knowledge(
            root,
            arguments["query"],
            arguments.get("top_k", 5),
        ),
    )
