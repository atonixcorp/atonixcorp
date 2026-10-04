"""Command-line database migration and legacy JSON import utility."""

import argparse
import json
from pathlib import Path
from typing import Sequence

from .database import DEFAULT_DATABASE_PATH, Database


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(
        description="Initialize or migrate the AtonixCorp Agent SQLite database."
    )
    parser.add_argument(
        "--database",
        type=Path,
        default=DEFAULT_DATABASE_PATH,
        help=f"SQLite file path (default: {DEFAULT_DATABASE_PATH})",
    )
    parser.add_argument(
        "--import-json",
        action="store_true",
        help="Import legacy model/metrics/events/RAG JSON state idempotently.",
    )
    parser.add_argument(
        "--legacy-root",
        type=Path,
        default=Path(__file__).resolve().parents[1],
        help="Agent directory containing the legacy model/ and db/ JSON files.",
    )
    return parser


def main(argv: Sequence[str] | None = None) -> int:
    args = build_parser().parse_args(argv)
    with Database(args.database) as database:
        result = {
            "database": str(database.path),
            "migrations": database.migration_status(),
        }
        if args.import_json:
            result["imported"] = database.import_legacy_json(args.legacy_root)
    print(json.dumps(result, indent=2, allow_nan=False))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
