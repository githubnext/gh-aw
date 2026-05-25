#!/usr/bin/env python3
from __future__ import annotations

import argparse
import difflib
import os
import re
import subprocess
import sys
from dataclasses import dataclass, field
from pathlib import Path
from typing import List

TEXT_EXTENSIONS = {
    ".cjs", ".css", ".js", ".json", ".jsx", ".lock", ".md", ".mdx",
    ".mjs", ".py", ".sh", ".sql", ".test", ".toml", ".ts", ".tsx", ".txt",
    ".yaml", ".yml",
}
EXCLUDE_PREFIXES = (
    ".git/",
    ".github/agents/",
    ".changeset/",
)
MANUAL_MARKER = "MANUAL MIGRATION"

@dataclass
class FileResult:
    path: str
    changed: bool = False
    manual_annotations: int = 0
    failures: List[str] = field(default_factory=list)
    diff: str = ""


def list_tracked_files(root: Path) -> List[Path]:
    output = subprocess.check_output(["git", "ls-files"], cwd=root, text=True)
    return [root / line for line in output.splitlines() if line]


def is_text_file(path: Path) -> bool:
    if any(str(path).replace("\\", "/").endswith(prefix) or f"/{prefix}" in str(path).replace("\\", "/") for prefix in []):
        return True
    if path.suffix.lower() in TEXT_EXTENSIONS:
        return True
    if path.name in {"Makefile", "README", "Dockerfile"}:
        return True
    return False


def annotate_manual_model_migrations(text: str) -> tuple[str, int]:
    lines = text.splitlines()
    out: List[str] = []
    count = 0
    pending_annotation = False
    for line in lines:
        if MANUAL_MARKER in line:
            pending_annotation = False
            out.append(line)
            continue
        if re.search(r"(^|[\s\"'])gemini-[A-Za-z0-9._-]+", line) or re.search(r"(^|\s)model:\s*(gemini|gemini-[A-Za-z0-9._-]+)", line):
            if out and MANUAL_MARKER in out[-1]:
                out.append(line)
                pending_annotation = False
                continue
            indent = re.match(r"^\s*", line).group(0)
            comment_prefix = "#" if not line.lstrip().startswith("//") else "//"
            out.append(f"{indent}{comment_prefix} {MANUAL_MARKER}: review former Gemini model mapping for Antigravity.")
            count += 1
        out.append(line)
    return "\n".join(out) + ("\n" if text.endswith("\n") else ""), count


def rewrite_text(text: str) -> tuple[str, int]:
    original = text
    replacements = [
        (r"(?m)^(\s*engine:\s*)gemini(\s*(?:#.*)?)$", r"\1antigravity\2"),
        (r"(?m)^(\s*id:\s*)gemini(\s*(?:#.*)?)$", r"\1antigravity\2"),
        (r"(?m)^(\s*runtime-id:\s*)gemini(\s*(?:#.*)?)$", r"\1antigravity\2"),
        (r'(?m)("engine"\s*:\s*")gemini(")', r'\1antigravity\2'),
        (r'(?m)("id"\s*:\s*")gemini(")', r'\1antigravity\2'),
        (r'(?m)("runtime-id"\s*:\s*")gemini(")', r'\1antigravity\2'),
        (r"\bGEMINI_API_KEY\b", "ANTIGRAVITY_API_KEY"),
        (r"\bGEMINI_MODEL\b", "ANTIGRAVITY_MODEL"),
        (r"\bGEMINI_API_BASE_URL\b", "ANTIGRAVITY_API_BASE_URL"),
        (r"\bGEMINI_CLI_TRUST_WORKSPACE\b", "ANTIGRAVITY_CLI_TRUST_WORKSPACE"),
        (r"\bparse_gemini_log\b", "parse_antigravity_log"),
        (r"\bconvert_gateway_config_gemini\b", "convert_gateway_config_antigravity"),
        (r"\bgemini-client-error\b", "antigravity-client-error"),
        (r"\.gemini/", ".antigravity/"),
        (r"\.gemini\b", ".antigravity"),
        (r"\bGEMINI\.md\b", "ANTIGRAVITY.md"),
        (r"\bGemini CLI\b", "Antigravity CLI"),
        (r"\bGoogle Gemini\b", "Antigravity"),
        (r"\bGemini\b", "Antigravity"),
        (r"\bgemini\b", "antigravity"),
        (r"(?<![@A-Za-z0-9_-])gemini(\s+--|\s*$)", r"agy\1"),
        (r"(?<![@A-Za-z0-9_-])gemini(?=\s+\$\(|\s+['\"]|\s+\w)", "agy"),
        (r"\bInstall Antigravity CLI\b", "Install Antigravity CLI"),
    ]
    for pattern, repl in replacements:
        text = re.sub(pattern, repl, text)
    text, manual_count = annotate_manual_model_migrations(text)
    return text, manual_count


def process_file(path: Path, root: Path, write: bool, show_diff: bool) -> FileResult:
    rel = path.relative_to(root).as_posix()
    result = FileResult(path=rel)
    try:
        raw = path.read_bytes()
    except Exception as exc:
        result.failures.append(str(exc))
        return result
    if b"\x00" in raw:
        return result
    try:
        original = raw.decode("utf-8")
    except UnicodeDecodeError:
        return result
    rewritten, manual_count = rewrite_text(original)
    result.manual_annotations = manual_count
    if rewritten == original:
        return result
    result.changed = True
    if show_diff:
        result.diff = "".join(
            difflib.unified_diff(
                original.splitlines(keepends=True),
                rewritten.splitlines(keepends=True),
                fromfile=rel,
                tofile=rel,
            )
        )
    if write:
        path.write_text(rewritten, encoding="utf-8")
    return result


def main() -> int:
    parser = argparse.ArgumentParser(description="Migrate Gemini engine references to Antigravity.")
    parser.add_argument("--root", default=".", help="Repository root (default: current directory)")
    parser.add_argument("--write", action="store_true", help="Write changes to disk")
    parser.add_argument("--diff", action="store_true", help="Print unified diffs for changed files")
    parser.add_argument("--files-only", action="store_true", help="Print only changed file paths")
    args = parser.parse_args()

    root = Path(args.root).resolve()
    results: List[FileResult] = []
    failures: List[FileResult] = []
    changed: List[FileResult] = []
    for path in list_tracked_files(root):
        rel = path.relative_to(root).as_posix()
        if rel.startswith(EXCLUDE_PREFIXES):
            continue
        if not path.exists() or path.is_dir() or not is_text_file(path):
            continue
        res = process_file(path, root, args.write, args.diff)
        results.append(res)
        if res.failures:
            failures.append(res)
        if res.changed:
            changed.append(res)

    if args.files_only:
        for res in changed:
            print(res.path)
    else:
        for res in changed:
            print(f"{res.path}\tmanual_annotations={res.manual_annotations}")
            if args.diff and res.diff:
                sys.stdout.write(res.diff)

    print()
    print(f"mode={'write' if args.write else 'dry-run'}")
    print(f"changed_files={len(changed)}")
    print(f"manual_annotations={sum(r.manual_annotations for r in changed)}")
    print(f"failures={len(failures)}")
    if failures:
        print("failure_summary:")
        for res in failures:
            for failure in res.failures:
                print(f"- {res.path}: {failure}")
    return 1 if failures else 0


if __name__ == "__main__":
    raise SystemExit(main())
