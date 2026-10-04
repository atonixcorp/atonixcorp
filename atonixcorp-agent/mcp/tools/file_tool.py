"""Read-only workspace file tools with path and size restrictions."""

import os
from pathlib import Path
from typing import Any

_READABLE_SUFFIXES = {
    ".csv",
    ".json",
    ".log",
    ".md",
    ".py",
    ".rst",
    ".toml",
    ".txt",
    ".yaml",
    ".yml",
}
_SENSITIVE_NAMES = {".env", "credentials", "id_rsa", "id_ed25519"}
_SENSITIVE_MARKERS = ("secret", "credential", "private_key", "api_key", "token")
_MAX_READ_BYTES = 256 * 1024
_MAX_LIST_ITEMS = 200


def _resolve_inside(root: Path, relative_path: str) -> Path:
    if not isinstance(relative_path, str) or not relative_path:
        raise ValueError("path must be a non-empty relative path")
    candidate = (root / relative_path).resolve()
    try:
        candidate.relative_to(root)
    except ValueError as error:
        raise ValueError("Requested path is outside the configured workspace") from error
    return candidate


def _is_readable(path: Path) -> bool:
    lower_name = path.name.lower()
    return (
        not path.name.startswith(".")
        and lower_name not in _SENSITIVE_NAMES
        and not any(marker in lower_name for marker in _SENSITIVE_MARKERS)
        and path.suffix.lower() in _READABLE_SUFFIXES
    )


def list_files(root: Path, relative_path: str = ".", limit: int = 100) -> dict[str, Any]:
    """List visible, readable files directly below a workspace directory."""
    if isinstance(limit, bool) or not isinstance(limit, int) or not 1 <= limit <= _MAX_LIST_ITEMS:
        raise ValueError(f"limit must be between 1 and {_MAX_LIST_ITEMS}")
    directory = _resolve_inside(root, relative_path)
    if not directory.is_dir():
        raise NotADirectoryError(f"Directory not found: {relative_path}")

    entries = []
    for entry in sorted(os.scandir(directory), key=lambda item: item.name.casefold()):
        if entry.name.startswith(".") or entry.is_symlink():
            continue
        entry_path = Path(entry.path)
        if entry.is_dir(follow_symlinks=False):
            entries.append({"name": entry.name, "type": "directory"})
        elif entry.is_file(follow_symlinks=False) and _is_readable(entry_path):
            entries.append(
                {
                    "name": entry.name,
                    "type": "file",
                    "size_bytes": entry.stat(follow_symlinks=False).st_size,
                }
            )
        if len(entries) >= limit:
            break
    return {"path": relative_path, "entries": entries}


def read_file(root: Path, relative_path: str) -> dict[str, Any]:
    """Read a small UTF-8 text file under the configured workspace root."""
    path = _resolve_inside(root, relative_path)
    if not path.is_file():
        raise FileNotFoundError(f"File not found: {relative_path}")
    if not _is_readable(path):
        raise ValueError("This file type or name is not available through the read tool")
    size = path.stat().st_size
    if size > _MAX_READ_BYTES:
        raise ValueError(f"File exceeds the {_MAX_READ_BYTES}-byte read limit")
    try:
        text = path.read_text(encoding="utf-8")
    except UnicodeDecodeError as error:
        raise ValueError("File is not valid UTF-8 text") from error
    return {"path": relative_path, "size_bytes": size, "text": text}


def register_file_tools(server: Any, root: Path) -> None:
    """Register safe workspace listing and read operations."""
    server.register_tool(
        "files.list",
        "List visible directories and supported text files under the configured workspace.",
        {
            "type": "object",
            "properties": {
                "path": {"type": "string", "description": "Workspace-relative directory."},
                "limit": {"type": "integer", "minimum": 1, "maximum": _MAX_LIST_ITEMS},
            },
            "additionalProperties": False,
        },
        lambda arguments: list_files(
            root,
            arguments.get("path", "."),
            arguments.get("limit", 100),
        ),
    )
    server.register_tool(
        "files.read",
        "Read a UTF-8 text file inside the configured workspace (256 KiB maximum).",
        {
            "type": "object",
            "properties": {
                "path": {"type": "string", "description": "Workspace-relative file path."}
            },
            "required": ["path"],
            "additionalProperties": False,
        },
        lambda arguments: read_file(root, arguments["path"]),
    )
