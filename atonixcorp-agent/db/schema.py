"""SQLite schema versions and transactional SQL migration runner."""

import hashlib
import re
import sqlite3
from dataclasses import dataclass
from pathlib import Path

MIGRATIONS_DIR = Path(__file__).resolve().parent / "migrations"
_MIGRATION_NAME = re.compile(r"^(\d{4})_[a-z0-9_]+\.sql$")
_MIGRATIONS_TABLE_SQL = """
CREATE TABLE IF NOT EXISTS schema_migrations (
    version INTEGER PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    checksum TEXT NOT NULL,
    applied_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
)
"""


@dataclass(frozen=True)
class Migration:
    version: int
    name: str
    sql: str
    checksum: str


def discover_migrations(directory: str | Path = MIGRATIONS_DIR) -> list[Migration]:
    """Load migration scripts in order and validate their unique version numbers."""
    migration_dir = Path(directory)
    if not migration_dir.is_dir():
        raise FileNotFoundError(f"Migration directory not found: {migration_dir}")

    migrations = []
    for path in sorted(migration_dir.iterdir()):
        if not path.is_file() or not path.name.endswith(".sql"):
            continue
        match = _MIGRATION_NAME.fullmatch(path.name)
        if match is None:
            raise ValueError(f"Invalid migration filename: {path.name}")
        sql = path.read_text(encoding="utf-8")
        if not sql.strip():
            raise ValueError(f"Migration is empty: {path}")
        migrations.append(
            Migration(
                version=int(match.group(1)),
                name=path.name,
                sql=sql,
                checksum=hashlib.sha256(sql.encode("utf-8")).hexdigest(),
            )
        )
    versions = [migration.version for migration in migrations]
    if len(versions) != len(set(versions)):
        raise ValueError("Migration version numbers must be unique")
    return migrations


def _execute_script_transactionally(connection: sqlite3.Connection, sql: str) -> None:
    """Execute a multi-statement SQL script without executescript's implicit commit."""
    statement = ""
    for line in sql.splitlines(keepends=True):
        statement += line
        if sqlite3.complete_statement(statement):
            if statement.strip():
                connection.execute(statement)
            statement = ""
    if statement.strip():
        raise ValueError("Migration contains an incomplete SQL statement")


def applied_migrations(connection: sqlite3.Connection) -> list[dict[str, object]]:
    """Return migration rows, or an empty list before schema setup."""
    exists = connection.execute(
        "SELECT 1 FROM sqlite_master WHERE type = 'table' AND name = 'schema_migrations'"
    ).fetchone()
    if exists is None:
        return []
    rows = connection.execute(
        "SELECT version, name, checksum, applied_at "
        "FROM schema_migrations ORDER BY version"
    ).fetchall()
    return [
        {
            "version": row["version"],
            "name": row["name"],
            "checksum": row["checksum"],
            "applied_at": row["applied_at"],
        }
        for row in rows
    ]


def migrate(
    connection: sqlite3.Connection,
    directory: str | Path = MIGRATIONS_DIR,
) -> list[str]:
    """Apply pending migrations, rejecting modified or unknown applied scripts."""
    migrations = discover_migrations(directory)
    connection.execute(_MIGRATIONS_TABLE_SQL)
    known = {migration.version: migration for migration in migrations}
    existing = applied_migrations(connection)
    applied_by_version = {int(row["version"]): row for row in existing}

    unknown = sorted(set(applied_by_version) - set(known))
    if unknown:
        raise ValueError(
            f"Database contains migration versions absent from source: {unknown}"
        )
    for version, row in applied_by_version.items():
        migration = known[version]
        if row["name"] != migration.name or row["checksum"] != migration.checksum:
            raise ValueError(
                f"Applied migration {migration.name} has changed; "
                "create a new migration instead of editing it"
            )

    applied_names = []
    for migration in migrations:
        if migration.version in applied_by_version:
            continue
        connection.execute("BEGIN IMMEDIATE")
        try:
            _execute_script_transactionally(connection, migration.sql)
            connection.execute(
                "INSERT INTO schema_migrations(version, name, checksum) VALUES (?, ?, ?)",
                (migration.version, migration.name, migration.checksum),
            )
            connection.commit()
        except Exception:
            connection.rollback()
            raise
        applied_names.append(migration.name)
    return applied_names
