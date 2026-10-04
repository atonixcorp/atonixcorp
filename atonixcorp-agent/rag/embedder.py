"""Deterministic, dependency-free hashed lexical vectors for document retrieval."""

import hashlib
import math
import re

_TOKEN_PATTERN = re.compile(r"[\w]+", flags=re.UNICODE)


class HashingEmbedder:
    """Map tokens and adjacent-token pairs into normalized signed vectors.

    This local hashing representation enables semantic-style vector ranking
    without downloading a model. It captures lexical overlap, not language
    understanding or synonym relationships.
    """

    def __init__(self, dimensions: int = 512, include_bigrams: bool = True):
        if isinstance(dimensions, bool) or not isinstance(dimensions, int) or dimensions < 16:
            raise ValueError("dimensions must be an integer of at least 16")
        if not isinstance(include_bigrams, bool):
            raise ValueError("include_bigrams must be a boolean")
        self.dimensions = dimensions
        self.include_bigrams = include_bigrams

    @staticmethod
    def _tokens(text: str) -> list[str]:
        if not isinstance(text, str) or not text.strip():
            raise ValueError("Text to embed must be a non-empty string")
        tokens = _TOKEN_PATTERN.findall(text.casefold())
        if not tokens:
            raise ValueError("Text to embed must contain searchable tokens")
        return tokens

    def embed(self, text: str) -> list[float]:
        """Return an L2-normalized signed feature-hashing vector."""
        tokens = self._tokens(text)
        features = [f"u:{token}" for token in tokens]
        if self.include_bigrams:
            features.extend(
                f"b:{left}_{right}" for left, right in zip(tokens, tokens[1:])
            )

        vector = [0.0] * self.dimensions
        for feature in features:
            digest = hashlib.blake2b(feature.encode("utf-8"), digest_size=8).digest()
            value = int.from_bytes(digest, "big")
            index = value % self.dimensions
            vector[index] += 1.0 if value & (1 << 63) else -1.0

        norm = math.sqrt(sum(value * value for value in vector))
        if norm:
            vector = [value / norm for value in vector]
        return vector

    def embed_many(self, texts: list[str]) -> list[list[float]]:
        """Embed a sequence using this embedder's fixed configuration."""
        if not isinstance(texts, list):
            raise ValueError("texts must be a list")
        return [self.embed(text) for text in texts]
