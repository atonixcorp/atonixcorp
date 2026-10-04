CREATE TABLE rag_documents (
    chunk_id TEXT PRIMARY KEY,
    document_id TEXT NOT NULL,
    source TEXT NOT NULL,
    chunk_index INTEGER NOT NULL CHECK (chunk_index >= 0),
    text TEXT NOT NULL,
    embedding_json TEXT NOT NULL,
    metadata_json TEXT NOT NULL DEFAULT '{}',
    created_at TEXT NOT NULL,
    UNIQUE (document_id, chunk_index)
);

CREATE INDEX idx_rag_documents_document
    ON rag_documents(document_id, chunk_index);
