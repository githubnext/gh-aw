#!/usr/bin/env python3
"""Deterministic precompute for Agentic Workflow Portfolio Yield."""

from __future__ import annotations

import argparse
import json
import math
import os
import re
import sys
from collections import Counter, defaultdict
from pathlib import Path, PureWindowsPath
from typing import Any

LAMBDA = 0.25
OVERLAP_THRESHOLD = 0.70
ALLOWED_RECOMMENDATIONS = ("Keep", "Revise", "Merge", "Instrument", "Retire")
STOPWORDS = {
    "a",
    "about",
    "after",
    "all",
    "also",
    "an",
    "and",
    "any",
    "are",
    "as",
    "at",
    "be",
    "been",
    "before",
    "being",
    "by",
    "can",
    "do",
    "for",
    "from",
    "get",
    "github",
    "have",
    "if",
    "in",
    "into",
    "is",
    "it",
    "its",
    "job",
    "of",
    "on",
    "or",
    "repo",
    "repository",
    "that",
    "the",
    "their",
    "this",
    "to",
    "use",
    "using",
    "workflow",
    "workflows",
    "with",
    "you",
    "your",
}
RISKY_PERMISSION_LEVELS = {"write", "admin"}
TELEMETRY_KEYS = {
    "input_tokens",
    "output_tokens",
    "runtime_duration",
    "tool_calls",
    "retries",
    "success_rate",
    "safe_output_success",
    "workflow_invocation_count",
    "user_interaction_count",
    "reviewer_interaction_count",
    "accepted_outputs",
    "outputs_acted_upon",
    "actionable_comments",
    "pr_impact",
    "issues_resolved",
    "bugs_found",
    "manual_minutes_saved",
}


class InputError(ValueError):
    """Raised when an input document cannot be processed safely."""


def clamp(value: Any, lower: float = 0.0, upper: float = 1.0) -> float:
    try:
        numeric = float(value)
    except (TypeError, ValueError):
        numeric = lower
    if math.isnan(numeric) or math.isinf(numeric):
        numeric = lower
    return max(lower, min(upper, numeric))


def round_score(value: Any) -> float:
    return round(clamp(value), 4)


def normalize_text(value: Any) -> str:
    if value is None:
        return ""
    if isinstance(value, str):
        return value.strip()
    return str(value).strip()


def coerce_bool(value: Any, default: bool = False) -> bool:
    if value is None:
        return default
    if isinstance(value, bool):
        return value
    if isinstance(value, (int, float)):
        return value != 0
    if isinstance(value, str):
        lowered = value.strip().lower()
        if lowered in {"true", "yes", "1", "observed", "validated"}:
            return True
        if lowered in {"false", "no", "0", "missing", "absent"}:
            return False
    return default


def split_frontmatter(text: str) -> tuple[str, str]:
    if not text.startswith("---"):
        return "", text
    lines = text.splitlines()
    end_index = None
    for index in range(1, len(lines)):
        if lines[index].strip() == "---":
            end_index = index
            break
    if end_index is None:
        return "", text
    frontmatter = "\n".join(lines[1:end_index])
    body = "\n".join(lines[end_index + 1 :])
    return frontmatter, body


def _split_inline_items(text: str) -> list[str]:
    items: list[str] = []
    current: list[str] = []
    depth = 0
    quote: str | None = None
    for char in text:
        if quote:
            current.append(char)
            if char == quote:
                quote = None
            continue
        if char in {"'", '"'}:
            quote = char
            current.append(char)
            continue
        if char in "[{":
            depth += 1
            current.append(char)
            continue
        if char in "]}":
            depth = max(0, depth - 1)
            current.append(char)
            continue
        if char == "," and depth == 0:
            items.append("".join(current).strip())
            current = []
            continue
        current.append(char)
    if current:
        items.append("".join(current).strip())
    return [item for item in items if item != ""]


def parse_scalar(value: str) -> Any:
    value = value.strip()
    if value == "":
        return ""
    lower = value.lower()
    if lower == "true":
        return True
    if lower == "false":
        return False
    if lower in {"null", "none", "~"}:
        return None
    if value.startswith("[") and value.endswith("]"):
        inner = value[1:-1].strip()
        if not inner:
            return []
        return [parse_scalar(item) for item in _split_inline_items(inner)]
    if value.startswith("{") and value.endswith("}"):
        inner = value[1:-1].strip()
        if not inner:
            return {}
        parsed: dict[str, Any] = {}
        for item in _split_inline_items(inner):
            key, raw = split_key_value(item)
            parsed[key] = parse_scalar(raw or "")
        return parsed
    if (value.startswith('"') and value.endswith('"')) or (value.startswith("'") and value.endswith("'")):
        return value[1:-1]
    if re.fullmatch(r"-?\d+", value):
        try:
            return int(value)
        except ValueError:
            return value
    if re.fullmatch(r"-?\d+\.\d+", value):
        try:
            return float(value)
        except ValueError:
            return value
    return value


def split_key_value(text: str) -> tuple[str, str | None]:
    depth = 0
    quote: str | None = None
    for index, char in enumerate(text):
        if quote:
            if char == quote:
                quote = None
            continue
        if char in {"'", '"'}:
            quote = char
            continue
        if char in "[{":
            depth += 1
            continue
        if char in "]}":
            depth = max(0, depth - 1)
            continue
        if char == ":" and depth == 0:
            key = text[:index].strip()
            rest = text[index + 1 :].strip()
            return key, rest if rest != "" else None
    raise InputError(f"Invalid frontmatter line: {text}")


def maybe_split_mapping(text: str) -> tuple[str, str | None] | None:
    try:
        key, rest = split_key_value(text)
    except InputError:
        return None
    if not re.fullmatch(r"[A-Za-z0-9_-]+", key):
        return None
    return key, rest


def _next_significant(lines: list[str], start: int) -> int:
    index = start
    while index < len(lines):
        stripped = lines[index].strip()
        if stripped and not stripped.startswith("#"):
            break
        index += 1
    return index


def parse_block_scalar(lines: list[str], start: int, indent: int) -> tuple[str, int]:
    chunks: list[str] = []
    index = start
    while index < len(lines):
        raw = lines[index]
        stripped = raw.strip()
        current_indent = len(raw) - len(raw.lstrip(" "))
        if stripped and current_indent < indent:
            break
        if stripped == "":
            chunks.append("")
            index += 1
            continue
        if current_indent < indent:
            break
        chunks.append(raw[indent:])
        index += 1
    return "\n".join(chunks).rstrip(), index


def parse_yaml_block(lines: list[str], start: int = 0, indent: int = 0) -> tuple[Any, int]:
    start = _next_significant(lines, start)
    if start >= len(lines):
        return {}, start
    line = lines[start]
    current_indent = len(line) - len(line.lstrip(" "))
    if current_indent < indent:
        return {}, start
    indent = current_indent
    is_list = line.lstrip().startswith("- ")
    if is_list:
        items: list[Any] = []
        index = start
        while index < len(lines):
            index = _next_significant(lines, index)
            if index >= len(lines):
                break
            raw = lines[index]
            item_indent = len(raw) - len(raw.lstrip(" "))
            if item_indent < indent:
                break
            stripped = raw[item_indent:]
            if item_indent != indent or not stripped.startswith("- "):
                break
            payload = stripped[2:].strip()
            index += 1
            if payload == "":
                child, index = parse_yaml_block(lines, index, indent + 2)
                items.append(child)
                continue
            mapping = maybe_split_mapping(payload)
            if mapping is not None:
                key, rest = mapping
                item: dict[str, Any] = {}
                if rest in {"|", ">", "|-", ">-"}:
                    child, index = parse_block_scalar(lines, index, indent + 4)
                    item[key] = child
                elif rest is None:
                    child, index = parse_yaml_block(lines, index, indent + 2)
                    item[key] = child
                else:
                    item[key] = parse_scalar(rest)
                while True:
                    lookahead = _next_significant(lines, index)
                    if lookahead >= len(lines):
                        break
                    next_raw = lines[lookahead]
                    next_indent = len(next_raw) - len(next_raw.lstrip(" "))
                    if next_indent < indent + 2:
                        break
                    if next_indent == indent and next_raw.lstrip().startswith("- "):
                        break
                    if next_indent > indent + 2:
                        break
                    extra_key, extra_rest = split_key_value(next_raw.strip())
                    index = lookahead + 1
                    if extra_rest in {"|", ">", "|-", ">-"}:
                        child, index = parse_block_scalar(lines, index, indent + 4)
                        item[extra_key] = child
                    elif extra_rest is None:
                        child, index = parse_yaml_block(lines, index, indent + 4)
                        item[extra_key] = child
                    else:
                        item[extra_key] = parse_scalar(extra_rest)
                items.append(item)
                continue
            items.append(parse_scalar(payload))
        return items, index
    mapping: dict[str, Any] = {}
    index = start
    while index < len(lines):
        index = _next_significant(lines, index)
        if index >= len(lines):
            break
        raw = lines[index]
        current_indent = len(raw) - len(raw.lstrip(" "))
        if current_indent < indent:
            break
        if current_indent > indent:
            break
        stripped = raw.strip()
        if stripped.startswith("- "):
            break
        key, rest = split_key_value(stripped)
        index += 1
        if rest in {"|", ">", "|-", ">-"}:
            child, index = parse_block_scalar(lines, index, indent + 2)
            mapping[key] = child
        elif rest is None:
            if index < len(lines) and _next_significant(lines, index) < len(lines):
                child, index = parse_yaml_block(lines, index, indent + 2)
                mapping[key] = child
            else:
                mapping[key] = {}
        else:
            mapping[key] = parse_scalar(rest)
    return mapping, index


def parse_frontmatter_text(frontmatter: str) -> dict[str, Any]:
    if not frontmatter.strip():
        return {}
    parsed, _ = parse_yaml_block(frontmatter.splitlines())
    if not isinstance(parsed, dict):
        raise InputError("Workflow frontmatter must parse to an object")
    return parsed


def read_workflow(path: Path) -> tuple[dict[str, Any], str]:
    text = path.read_text(encoding="utf-8")
    frontmatter_text, body = split_frontmatter(text)
    return parse_frontmatter_text(frontmatter_text), body


def discover_workflow_files(workflows_root: Path) -> list[Path]:
    files = []
    for path in workflows_root.rglob("*.md"):
        if "shared" in path.relative_to(workflows_root).parts:
            continue
        files.append(path)
    return sorted(files)


def as_list(value: Any) -> list[Any]:
    if value is None:
        return []
    if isinstance(value, list):
        return value
    return [value]


def _get_workflows_root(workflow_path: Path) -> Path | None:
    resolved = workflow_path.resolve()
    for candidate in (resolved.parent, *resolved.parents):
        if candidate.parent == candidate:
            break
        if candidate.name == "workflows" and candidate.parent.name == ".github":
            return candidate
    return None


def _path_is_within(path: Path, root: Path) -> bool:
    try:
        path.resolve().relative_to(root.resolve())
        return True
    except ValueError:
        return False


def _is_absolute_import_path(raw: str) -> bool:
    return Path(raw).is_absolute() or PureWindowsPath(raw).is_absolute()


def normalize_import_paths(workflow_path: Path, frontmatter: dict[str, Any]) -> list[Path]:
    workflows_root = _get_workflows_root(workflow_path)
    imports = []
    for item in as_list(frontmatter.get("imports")):
        raw: str | None = None
        if isinstance(item, str):
            raw = item
        elif isinstance(item, dict):
            raw = normalize_text(item.get("uses")) or normalize_text(item.get("path"))
        if not raw or "@" in raw or "/" in raw and not raw.startswith("shared/") and not raw.startswith("."):
            if raw and (raw.startswith(("shared/", "./", "../", "/")) or _is_absolute_import_path(raw)):
                pass
            else:
                continue
        if not workflows_root:
            continue
        if _is_absolute_import_path(raw):
            continue
        if raw.startswith("shared/"):
            import_path = (workflows_root / raw).resolve()
            if _path_is_within(import_path, workflows_root / "shared"):
                imports.append(import_path)
        elif raw.startswith("./") or raw.startswith("../"):
            import_path = (workflow_path.parent / raw).resolve()
            if _path_is_within(import_path, workflows_root):
                imports.append(import_path)
    return imports


def has_observability_config(frontmatter: dict[str, Any]) -> bool:
    observability = frontmatter.get("observability")
    return isinstance(observability, dict) and isinstance(observability.get("otlp"), dict)


def has_imported_observability(workflow_path: Path, frontmatter: dict[str, Any]) -> bool:
    for import_path in normalize_import_paths(workflow_path, frontmatter):
        if not import_path.exists():
            continue
        try:
            imported_frontmatter, _ = read_workflow(import_path)
        except Exception:
            continue
        if has_observability_config(imported_frontmatter):
            return True
        if normalize_text(import_path.name).lower().find("otel") >= 0:
            return True
        mcp_servers = imported_frontmatter.get("mcp-servers")
        if isinstance(mcp_servers, dict) and "otel" in mcp_servers:
            return True
    return False


def get_matching_lockfile(workflow_path: Path) -> Path:
    return workflow_path.with_suffix(".lock.yml")


def detect_lockfile_status(workflow_path: Path) -> tuple[bool, bool]:
    lockfile = get_matching_lockfile(workflow_path)
    if not lockfile.exists():
        return False, False
    try:
        workflow_mtime = workflow_path.stat().st_mtime
        lockfile_mtime = lockfile.stat().st_mtime
    except OSError:
        return True, False
    return True, lockfile_mtime < workflow_mtime


def count_steps(value: Any) -> int:
    if isinstance(value, list):
        return len(value)
    if isinstance(value, dict) and value:
        return 1
    return 0


def collect_step_text(value: Any) -> str:
    return json.dumps(value, sort_keys=True, ensure_ascii=False) if value else ""


def infer_timeout_minutes(value: Any) -> int | None:
    if isinstance(value, (int, float)):
        return int(value)
    if isinstance(value, str):
        match = re.search(r"\d+", value)
        if match:
            return int(match.group(0))
    return None


def permissions_risk(permissions: Any) -> float:
    if not isinstance(permissions, dict) or not permissions:
        return 0.45
    read_scopes = 0
    elevated = 0
    id_token = 0
    for scope, level in permissions.items():
        normalized_scope = normalize_text(scope).lower()
        normalized_level = normalize_text(level).lower()
        if normalized_scope == "id-token" and normalized_level in RISKY_PERMISSION_LEVELS:
            id_token += 1
        if normalized_level == "read":
            read_scopes += 1
        elif normalized_level in RISKY_PERMISSION_LEVELS:
            elevated += 1
        elif normalized_level == "none":
            continue
    breadth = clamp(read_scopes / 6.0)
    return round_score(0.2 + breadth * 0.35 + elevated * 0.45 + id_token * 0.1)


def count_tools(frontmatter: dict[str, Any]) -> int:
    tool_count = len(frontmatter.get("tools", {}) or {})
    mcp_servers = frontmatter.get("mcp-servers") or {}
    if isinstance(mcp_servers, dict):
        tool_count += len(mcp_servers)
    return tool_count


def tokenize(text: str) -> list[str]:
    return [token for token in re.findall(r"[a-z0-9]+", text.lower()) if token not in STOPWORDS and len(token) > 2]


def extract_trigger_tokens(on_value: Any) -> list[str]:
    if isinstance(on_value, str):
        return tokenize(on_value)
    if isinstance(on_value, list):
        tokens: list[str] = []
        for item in on_value:
            tokens.extend(tokenize(normalize_text(item)))
        return tokens
    if isinstance(on_value, dict):
        tokens = []
        for key, value in on_value.items():
            tokens.extend(tokenize(key))
            tokens.extend(tokenize(json.dumps(value, sort_keys=True)))
        return tokens
    return []


def extract_headings(body: str) -> list[str]:
    return [match.group(1).strip() for match in re.finditer(r"^#+\s+(.+)$", body, re.MULTILINE)]


def build_intent_text(path: Path, frontmatter: dict[str, Any], body: str) -> str:
    parts = [
        path.stem.replace("-", " "),
        normalize_text(frontmatter.get("name")),
        normalize_text(frontmatter.get("description")),
        " ".join(extract_trigger_tokens(frontmatter.get("on"))),
        " ".join((frontmatter.get("safe-outputs") or {}).keys()) if isinstance(frontmatter.get("safe-outputs"), dict) else "",
        " ".join((frontmatter.get("tools") or {}).keys()) if isinstance(frontmatter.get("tools"), dict) else "",
        " ".join(extract_headings(body)),
        re.sub(r"\s+", " ", body)[:1500],
    ]
    return " ".join(part for part in parts if part).strip()


def estimate_agentic_fraction(frontmatter: dict[str, Any], body: str) -> tuple[float, float]:
    pre_text = collect_step_text(frontmatter.get("pre-agent-steps"))
    post_text = collect_step_text(frontmatter.get("post-steps"))
    body_words = len(body.split())
    pre_weight = count_steps(frontmatter.get("pre-agent-steps")) * 1.3
    post_weight = count_steps(frontmatter.get("post-steps")) * 1.1
    pre_weight += 0.8 * len(re.findall(r"\b(python3?|jq|grep|awk|sed|sort|uniq|cat|find)\b", pre_text))
    post_weight += 0.8 * len(re.findall(r"\b(python3?|jq|grep|awk|sed|sort|uniq|cat|find)\b", post_text))
    tool_weight = count_tools(frontmatter) * 0.15
    agent_weight = max(0.25, body_words / 220.0 + tool_weight)
    total = pre_weight + post_weight + agent_weight
    if total <= 0:
        return 0.5, 0.5
    agentic_fraction = round_score(agent_weight / total)
    deterministic_fraction = round_score(1.0 - agentic_fraction)
    return agentic_fraction, deterministic_fraction


def score_observability(has_direct: bool, has_imported: bool, telemetry_metrics: dict[str, Any]) -> float:
    score = 0.0
    if has_direct:
        score += 0.6
    if has_imported:
        score += 0.3
    if telemetry_metrics:
        score += 0.4
    return round_score(score)


def score_safe_outputs(safe_outputs: Any) -> float:
    if not isinstance(safe_outputs, dict) or not safe_outputs:
        return 0.0
    score = 0.3
    if "create-issue" in safe_outputs:
        score += 0.3
    if any(key in safe_outputs for key in ("mentions", "allowed-github-references", "max-bot-mentions")):
        score += 0.2
    for key, value in safe_outputs.items():
        if isinstance(value, dict) and value.get("max") is not None:
            score += 0.2
            break
    return round_score(score)


def score_cost(frontmatter: dict[str, Any], body: str, telemetry_metrics: dict[str, Any], agentic_fraction: float) -> float:
    timeout = infer_timeout_minutes(frontmatter.get("timeout-minutes")) or 20
    base = 0.15 + clamp(timeout / 60.0) * 0.2 + agentic_fraction * 0.25 + clamp(count_tools(frontmatter) / 8.0) * 0.15
    base += clamp(len(body) / 6000.0) * 0.1
    if telemetry_metrics:
        base += clamp((telemetry_metrics.get("input_tokens", 0) + telemetry_metrics.get("output_tokens", 0)) / 250000.0) * 0.3
        base += clamp(telemetry_metrics.get("runtime_duration", 0) / 1800.0) * 0.25
        base += clamp(telemetry_metrics.get("tool_calls", 0) / 150.0) * 0.2
        base += clamp(telemetry_metrics.get("retries", 0) / 6.0) * 0.2
    return round_score(base)


def score_trust(
    strict: bool,
    timeout_minutes: int | None,
    has_lockfile: bool,
    lockfile_stale: bool,
    safe_output_score: float,
    observability_score: float,
    telemetry_metrics: dict[str, Any],
) -> float:
    score = 0.2
    if strict:
        score += 0.2
    if timeout_minutes:
        score += 0.1
    if has_lockfile and not lockfile_stale:
        score += 0.15
    score += safe_output_score * 0.2
    score += observability_score * 0.1
    if telemetry_metrics:
        score += clamp(telemetry_metrics.get("success_rate", 0.5)) * 0.35
        score += clamp(1.0 - telemetry_metrics.get("retries", 0) / 6.0) * 0.15
        score += clamp(telemetry_metrics.get("safe_output_success", 0.0)) * 0.2
    return round_score(score)


def score_usefulness(
    frontmatter: dict[str, Any],
    body: str,
    safe_output_score: float,
    telemetry_metrics: dict[str, Any],
) -> float:
    score = 0.1 + safe_output_score * 0.25 + clamp(len(extract_headings(body)) / 8.0) * 0.1
    triggers = extract_trigger_tokens(frontmatter.get("on"))
    if triggers:
        score += 0.1
    if telemetry_metrics:
        score += clamp(telemetry_metrics.get("outputs_acted_upon", telemetry_metrics.get("accepted_outputs", 0.0))) * 0.35
        score += clamp(telemetry_metrics.get("issues_resolved", 0) / 10.0) * 0.15
        score += clamp(telemetry_metrics.get("manual_minutes_saved", 0) / 180.0) * 0.2
        score += clamp(telemetry_metrics.get("actionable_comments", 0) / 10.0) * 0.1
    else:
        score += 0.15 if safe_output_score > 0 else 0.0
    return round_score(score)


def score_adoption(frontmatter: dict[str, Any], telemetry_metrics: dict[str, Any]) -> float:
    score = 0.05
    on_value = frontmatter.get("on")
    if isinstance(on_value, dict) and "workflow_dispatch" in on_value:
        score += 0.1
    if isinstance(on_value, dict) and "schedule" in on_value:
        score += 0.1
    imports = as_list(frontmatter.get("imports"))
    score += clamp(len(imports) / 4.0) * 0.1
    if telemetry_metrics:
        score += clamp(telemetry_metrics.get("workflow_invocation_count", 0) / 50.0) * 0.45
        interactions = telemetry_metrics.get("user_interaction_count", 0) + telemetry_metrics.get("reviewer_interaction_count", 0)
        score += clamp(interactions / 25.0) * 0.2
    return round_score(score)


def score_maintenance(
    frontmatter: dict[str, Any],
    body: str,
    overlap_hint: float,
    agentic_fraction: float,
    has_precompute: bool,
    has_postcompute: bool,
) -> float:
    body_lines = len(body.splitlines())
    imports = len(as_list(frontmatter.get("imports")))
    tool_count = count_tools(frontmatter)
    score = 0.1 + clamp(body_lines / 250.0) * 0.2 + clamp(tool_count / 8.0) * 0.15 + clamp(imports / 5.0) * 0.1
    score += overlap_hint * 0.2 + agentic_fraction * 0.2
    if not has_precompute:
        score += 0.1
    if not has_postcompute:
        score += 0.1
    return round_score(score)


def score_risk(
    permission_score: float,
    strict: bool,
    timeout_minutes: int | None,
    has_lockfile: bool,
    lockfile_stale: bool,
    safe_output_score: float,
    observability_score: float,
    network: Any,
    agentic_fraction: float,
) -> float:
    score = permission_score * 0.35 + agentic_fraction * 0.15
    if not strict:
        score += 0.2
    if timeout_minutes is None:
        score += 0.15
    if not has_lockfile:
        score += 0.15
    if lockfile_stale:
        score += 0.15
    if safe_output_score == 0.0:
        score += 0.2
    if observability_score < 0.4:
        score += 0.15
    if not isinstance(network, dict) or "allowed" not in network:
        score += 0.1
    return round_score(score)


def evidence_quality_for_workflow(
    observability_declared: bool,
    telemetry_observed: bool,
    telemetry_validated: bool,
    telemetry_metrics: dict[str, Any],
) -> str:
    if observability_declared and telemetry_validated and len(telemetry_metrics) >= 3:
        return "high"
    if telemetry_validated or (telemetry_observed and len(telemetry_metrics) >= 2):
        return "medium"
    return "low"


def normalize_telemetry_entry(entry: dict[str, Any]) -> dict[str, Any]:
    normalized: dict[str, Any] = {}
    aliases = {
        "runtime_duration": ("runtime_duration", "duration_seconds", "duration", "runtime_seconds"),
        "tool_calls": ("tool_calls", "tool_call_count"),
        "retries": ("retries", "retry_count"),
        "success_rate": ("success_rate",),
        "safe_output_success": ("safe_output_success", "safe_output_success_rate"),
        "workflow_invocation_count": ("workflow_invocation_count", "invocation_count", "runs"),
        "user_interaction_count": ("user_interaction_count", "user_interactions"),
        "reviewer_interaction_count": ("reviewer_interaction_count", "reviewer_interactions"),
        "input_tokens": ("input_tokens",),
        "output_tokens": ("output_tokens",),
        "accepted_outputs": ("accepted_outputs",),
        "outputs_acted_upon": ("outputs_acted_upon", "acted_upon_rate"),
        "actionable_comments": ("actionable_comments",),
        "pr_impact": ("pr_impact",),
        "issues_resolved": ("issues_resolved",),
        "bugs_found": ("bugs_found",),
        "manual_minutes_saved": ("manual_minutes_saved", "minutes_saved"),
    }
    for target, keys in aliases.items():
        for key in keys:
            if key in entry:
                normalized[target] = entry[key]
                break
    metrics = {key: value for key, value in normalized.items() if key in TELEMETRY_KEYS}
    observed = coerce_bool(
        entry.get("telemetry_observed", entry.get("observed")),
        default=bool(metrics),
    )
    validated = coerce_bool(
        entry.get("telemetry_validated", entry.get("validated")),
        default=False,
    )
    return {
        "metrics": metrics,
        "observed": observed and bool(metrics),
        "validated": validated and bool(metrics),
        "source": normalize_text(entry.get("source")),
    }


def load_otel_summary(path: str | None) -> dict[str, dict[str, Any]]:
    if not path:
        return {}
    summary_path = Path(path)
    if not summary_path.exists():
        return {}
    try:
        payload = json.loads(summary_path.read_text(encoding="utf-8"))
    except json.JSONDecodeError:
        return {}
    entries: list[dict[str, Any]] = []
    if isinstance(payload, dict):
        if isinstance(payload.get("workflows"), list):
            entries = [entry for entry in payload["workflows"] if isinstance(entry, dict)]
        elif isinstance(payload.get("workflow_metrics"), dict):
            for key, value in payload["workflow_metrics"].items():
                if isinstance(value, dict):
                    item = dict(value)
                    item.setdefault("name", key)
                    entries.append(item)
        else:
            for key, value in payload.items():
                if isinstance(value, dict):
                    item = dict(value)
                    item.setdefault("name", key)
                    entries.append(item)
    elif isinstance(payload, list):
        entries = [entry for entry in payload if isinstance(entry, dict)]
    index: dict[str, dict[str, Any]] = {}
    for entry in entries:
        normalized = normalize_telemetry_entry(entry)
        if not normalized.get("metrics"):
            continue
        keys = {
            normalize_text(entry.get("path")),
            normalize_text(entry.get("workflow_path")),
            normalize_text(entry.get("name")),
            normalize_text(entry.get("workflow")),
            normalize_text(entry.get("workflow_name")),
        }
        for key in list(keys):
            if key:
                index[key] = normalized
                index[Path(key).stem] = normalized
    return index


def telemetry_for_workflow(
    workflow_path: Path,
    relative_path: str,
    frontmatter: dict[str, Any],
    telemetry_index: dict[str, dict[str, Any]],
) -> dict[str, Any]:
    candidates = [
        normalize_text(relative_path),
        workflow_path.name,
        normalize_text(frontmatter.get("name")),
    ]
    for key in candidates:
        if key in telemetry_index:
            return dict(telemetry_index[key])
    return {"metrics": {}, "observed": False, "validated": False, "source": ""}


def build_workflow_record(workflow_path: Path, workflows_root: Path, telemetry_index: dict[str, dict[str, Any]]) -> dict[str, Any]:
    frontmatter, body = read_workflow(workflow_path)
    relative_path = workflow_path.relative_to(workflows_root.parent.parent).as_posix()
    has_lockfile, lockfile_stale = detect_lockfile_status(workflow_path)
    strict = bool(frontmatter.get("strict", False))
    timeout_minutes = infer_timeout_minutes(frontmatter.get("timeout-minutes"))
    safe_outputs = frontmatter.get("safe-outputs") or {}
    has_safe_outputs = isinstance(safe_outputs, dict) and bool(safe_outputs)
    telemetry_entry = telemetry_for_workflow(workflow_path, relative_path, frontmatter, telemetry_index)
    telemetry_metrics = telemetry_entry.get("metrics", {})
    telemetry_observed = coerce_bool(telemetry_entry.get("observed"), default=bool(telemetry_metrics)) and bool(telemetry_metrics)
    telemetry_validated = coerce_bool(telemetry_entry.get("validated"), default=False) and bool(telemetry_metrics)
    has_direct_observability = has_observability_config(frontmatter)
    has_imported = has_imported_observability(workflow_path, frontmatter)
    observability_declared = has_direct_observability or has_imported
    observability_score = score_observability(has_direct_observability, has_imported, telemetry_metrics)
    safe_output_score = score_safe_outputs(safe_outputs)
    agentic_fraction, deterministic_fraction = estimate_agentic_fraction(frontmatter, body)
    permission_score = permissions_risk(frontmatter.get("permissions"))
    notes: list[str] = []
    if not has_lockfile:
        notes.append("missing lockfile")
    elif lockfile_stale:
        notes.append("stale lockfile")
    if not strict:
        notes.append("strict mode disabled")
    if timeout_minutes is None:
        notes.append("missing timeout")
    if not has_safe_outputs:
        notes.append("missing safe outputs")
    if not observability_declared:
        notes.append("observability not declared")
    elif not telemetry_observed:
        notes.append("telemetry not observed")
    elif not telemetry_validated:
        notes.append("telemetry not validated")
    network = frontmatter.get("network")
    usefulness = score_usefulness(frontmatter, body, safe_output_score, telemetry_metrics)
    adoption = score_adoption(frontmatter, telemetry_metrics)
    trust = score_trust(strict, timeout_minutes, has_lockfile, lockfile_stale, safe_output_score, observability_score, telemetry_metrics)
    cost = score_cost(frontmatter, body, telemetry_metrics, agentic_fraction)
    maintenance_drag = score_maintenance(
        frontmatter,
        body,
        overlap_hint=0.0,
        agentic_fraction=agentic_fraction,
        has_precompute=count_steps(frontmatter.get("pre-agent-steps")) > 0,
        has_postcompute=count_steps(frontmatter.get("post-steps")) > 0,
    )
    risk = score_risk(
        permission_score,
        strict,
        timeout_minutes,
        has_lockfile,
        lockfile_stale,
        safe_output_score,
        observability_score,
        network,
        agentic_fraction,
    )
    return {
        "path": relative_path,
        "name": normalize_text(frontmatter.get("name")) or workflow_path.stem,
        "description": normalize_text(frontmatter.get("description")),
        "has_lockfile": has_lockfile,
        "lockfile_stale": lockfile_stale,
        "has_safe_outputs": has_safe_outputs,
        "has_observability": has_direct_observability,
        "has_imported_observability": has_imported,
        "observability_declared": observability_declared,
        "telemetry_observed": telemetry_observed,
        "telemetry_validated": telemetry_validated,
        "telemetry_source": normalize_text(telemetry_entry.get("source")),
        "strict": strict,
        "timeout_minutes": timeout_minutes,
        "permissions_risk": permission_score,
        "tool_count": count_tools(frontmatter),
        "pre_agent_steps_count": count_steps(frontmatter.get("pre-agent-steps")),
        "post_steps_count": count_steps(frontmatter.get("post-steps")),
        "agentic_fraction": agentic_fraction,
        "deterministic_fraction": deterministic_fraction,
        "usefulness": usefulness,
        "adoption": adoption,
        "trust": trust,
        "cost": cost,
        "risk": risk,
        "maintenance_drag": maintenance_drag,
        "overlap_drag": 0.0,
        "yield": 0.0,
        "intent_text": build_intent_text(workflow_path, frontmatter, body),
        "recommendation_seed": "Instrument" if observability_score < 0.4 else "Revise",
        "evidence_quality": evidence_quality_for_workflow(
            observability_declared,
            telemetry_observed,
            telemetry_validated,
            telemetry_metrics,
        ),
        "notes": notes,
        "telemetry_metrics": telemetry_metrics,
    }


def compute_similarity_matrix(workflows: list[dict[str, Any]]) -> tuple[dict[tuple[str, str], float], dict[str, Counter[str]]]:
    documents: dict[str, Counter[str]] = {}
    doc_frequency: Counter[str] = Counter()
    for workflow in workflows:
        counts = Counter(tokenize(workflow.get("intent_text", "")))
        documents[workflow["path"]] = counts
        for token in counts:
            doc_frequency[token] += 1
    total_docs = max(1, len(workflows))
    tfidf_vectors: dict[str, dict[str, float]] = {}
    norms: dict[str, float] = {}
    for path, counts in documents.items():
        vector: dict[str, float] = {}
        for token, frequency in counts.items():
            idf = math.log((1.0 + total_docs) / (1.0 + doc_frequency[token])) + 1.0
            vector[token] = frequency * idf
        tfidf_vectors[path] = vector
        norms[path] = math.sqrt(sum(value * value for value in vector.values())) or 1.0
    similarities: dict[tuple[str, str], float] = {}
    paths = [workflow["path"] for workflow in workflows]
    for index, left in enumerate(paths):
        for right in paths[index + 1 :]:
            dot = 0.0
            shared = set(tfidf_vectors[left]).intersection(tfidf_vectors[right])
            for token in shared:
                dot += tfidf_vectors[left][token] * tfidf_vectors[right][token]
            similarity = clamp(dot / (norms[left] * norms[right]))
            similarities[(left, right)] = round(similarity, 4)
    return similarities, documents


def build_overlap_clusters(workflows: list[dict[str, Any]], similarities: dict[tuple[str, str], float], documents: dict[str, Counter[str]]) -> list[dict[str, Any]]:
    adjacency: dict[str, set[str]] = defaultdict(set)
    for (left, right), similarity in similarities.items():
        if similarity >= OVERLAP_THRESHOLD:
            adjacency[left].add(right)
            adjacency[right].add(left)
    seen: set[str] = set()
    clusters: list[dict[str, Any]] = []
    for workflow in workflows:
        path = workflow["path"]
        if path in seen or path not in adjacency:
            continue
        stack = [path]
        members: list[str] = []
        while stack:
            current = stack.pop()
            if current in seen:
                continue
            seen.add(current)
            members.append(current)
            stack.extend(sorted(adjacency[current] - seen))
        members.sort()
        if len(members) < 2:
            continue
        max_overlap = max(
            similarities.get((left, right), similarities.get((right, left), 0.0))
            for index, left in enumerate(members)
            for right in members[index + 1 :]
        )
        token_counts = Counter()
        for member in members:
            token_counts.update(documents.get(member, Counter()))
        reason = ", ".join(token for token, _ in token_counts.most_common(4)) or "shared operational intent"
        clusters.append({"workflows": members, "max_overlap": round(max_overlap, 4), "reason": reason})
    return clusters


def portfolio_overlap_drag(similarities: dict[tuple[str, str], float]) -> float:
    drag = 0.0
    for similarity in similarities.values():
        drag += similarity * similarity * 2.0
    return round(drag, 4)


def compute_workflow_yield(
    usefulness: float,
    adoption: float,
    trust: float,
    cost: float,
    risk: float,
    maintenance_drag: float,
    overlap_drag: float,
) -> float:
    denominator = 1.0 + cost + risk + maintenance_drag + overlap_drag
    if denominator <= 0:
        return 0.0
    return round((usefulness * adoption * trust) / denominator, 4)


def assign_recommendation(workflow: dict[str, Any], clustered_paths: set[str]) -> str:
    if not workflow.get("has_observability") and not workflow.get("has_imported_observability"):
        return "Instrument"
    if workflow["yield"] < 0.08 and workflow["trust"] < 0.45 and (
        workflow["risk"] > 0.55 or workflow["cost"] > 0.55 or workflow["maintenance_drag"] > 0.55
    ):
        return "Retire"
    if workflow["overlap_drag"] >= 0.45 or workflow["path"] in clustered_paths:
        return "Merge"
    if workflow["usefulness"] >= 0.35 and (
        workflow["cost"] > 0.55
        or workflow["risk"] > 0.55
        or workflow["maintenance_drag"] > 0.55
        or workflow["agentic_fraction"] > 0.65
    ):
        return "Revise"
    return "Keep"


def compute_episode_metrics(workflows: list[dict[str, Any]], similarities: dict[tuple[str, str], float]) -> list[dict[str, Any]]:
    buckets: dict[str, list[dict[str, Any]]] = defaultdict(list)
    for workflow in workflows:
        text = workflow.get("intent_text", "").lower()
        if "pull request" in text or "pr" in text or "review" in text:
            buckets["pr-pipeline"].append(workflow)
        elif "issue" in text or "triage" in text:
            buckets["issue-pipeline"].append(workflow)
        elif "release" in text or "deploy" in text:
            buckets["release-pipeline"].append(workflow)
        elif "incident" in text or "security" in text:
            buckets["incident-pipeline"].append(workflow)
    episodes: list[dict[str, Any]] = []
    for label, members in buckets.items():
        if len(members) < 2:
            continue
        paths = [member["path"] for member in members]
        pair_scores = []
        for index, left in enumerate(paths):
            for right in paths[index + 1 :]:
                pair_scores.append(similarities.get((left, right), similarities.get((right, left), 0.0)))
        if not pair_scores:
            continue
        avg_overlap = sum(pair_scores) / len(pair_scores)
        avg_cost = sum(member["cost"] for member in members) / len(members)
        avg_yield = sum(member["yield"] for member in members) / len(members)
        coordination_drag = round_score(avg_overlap * (0.5 + avg_cost))
        episode_yield = round(max(0.0, avg_yield / (1.0 + coordination_drag)), 4)
        if avg_overlap < 0.2 and coordination_drag < 0.2:
            continue
        episodes.append(
            {
                "episode": label,
                "workflows": paths,
                "coordination_drag": coordination_drag,
                "episode_yield": episode_yield,
                "evidence_quality": "medium" if avg_overlap >= 0.35 else "low",
            }
        )
    return episodes


def compute_organizational_health(workflows: list[dict[str, Any]], overlap_drag_value: float) -> dict[str, Any]:
    if not workflows:
        return {"fragmentation": 0.0, "reuse": 0.0, "trust_concentration": 0.0, "governance_drag": 0.0, "notes": []}
    reuse = round_score(
        sum(1.0 for workflow in workflows if workflow.get("has_imported_observability") or workflow.get("tool_count", 0) > 0) / len(workflows)
    )
    average_trust = sum(workflow["trust"] for workflow in workflows) / len(workflows)
    trust_concentration = round_score(max(workflow["trust"] for workflow in workflows) - average_trust)
    governance_drag = round_score(
        sum(workflow["risk"] + workflow["agentic_fraction"] * 0.25 for workflow in workflows) / len(workflows)
    )
    fragmentation = round_score(clamp(overlap_drag_value / max(1.0, len(workflows)) * 0.7 + (1.0 - reuse) * 0.3))
    notes: list[str] = []
    if fragmentation > 0.6 and average_trust < 0.5:
        notes.append("High overlap plus uneven trust suggests organizational fragmentation.")
    if reuse > 0.55 and fragmentation < 0.45 and average_trust > 0.55:
        notes.append("Shared imports and higher trust indicate improving operational coherence.")
    if governance_drag > 0.55:
        notes.append("Governance drag is elevated by broad scope, missing telemetry, or high agentic fractions.")
    return {
        "fragmentation": fragmentation,
        "reuse": reuse,
        "trust_concentration": trust_concentration,
        "governance_drag": governance_drag,
        "notes": notes,
    }


def portfolio_evidence_quality(
    workflows: list[dict[str, Any]],
    telemetry_observed_coverage: float,
    telemetry_validated_coverage: float,
) -> str:
    if telemetry_validated_coverage >= 0.75 and all(workflow["evidence_quality"] != "low" for workflow in workflows):
        return "high"
    if telemetry_validated_coverage >= 0.35 or (telemetry_observed_coverage >= 0.5 and telemetry_validated_coverage > 0.0):
        return "medium"
    return "low"


def build_recommendation_seed(workflows: list[dict[str, Any]]) -> dict[str, list[str]]:
    buckets = {key.lower(): [] for key in ALLOWED_RECOMMENDATIONS}
    for workflow in workflows:
        buckets[workflow["recommendation_seed"].lower()].append(workflow["path"])
    return buckets


def compute_portfolio_metrics(workflows: list[dict[str, Any]], overlap_drag_value: float) -> dict[str, Any]:
    if not workflows:
        return {
            "workflow_count": 0,
            "portfolio_yield": 0.0,
            "portfolio_overlap_drag": 0.0,
            "portfolio_cost": 0.0,
            "portfolio_risk": 0.0,
            "portfolio_maintenance_drag": 0.0,
            "average_agentic_fraction": 0.0,
            "average_deterministic_fraction": 0.0,
            "observability_declared_coverage": 0.0,
            "telemetry_observed_coverage": 0.0,
            "telemetry_validated_coverage": 0.0,
            "telemetry_coverage": 0.0,
            "evidence_quality": "low",
        }
    average_yield = sum(workflow["yield"] for workflow in workflows) / len(workflows)
    observability_declared_coverage = sum(1.0 for workflow in workflows if workflow.get("observability_declared")) / len(workflows)
    telemetry_observed_coverage = sum(1.0 for workflow in workflows if workflow.get("telemetry_observed")) / len(workflows)
    telemetry_validated_coverage = sum(1.0 for workflow in workflows if workflow.get("telemetry_validated")) / len(workflows)
    return {
        "workflow_count": len(workflows),
        "portfolio_yield": round(average_yield - LAMBDA * overlap_drag_value, 4),
        "portfolio_overlap_drag": round(overlap_drag_value, 4),
        "portfolio_cost": round(sum(workflow["cost"] for workflow in workflows) / len(workflows), 4),
        "portfolio_risk": round(sum(workflow["risk"] for workflow in workflows) / len(workflows), 4),
        "portfolio_maintenance_drag": round(sum(workflow["maintenance_drag"] for workflow in workflows) / len(workflows), 4),
        "average_agentic_fraction": round(sum(workflow["agentic_fraction"] for workflow in workflows) / len(workflows), 4),
        "average_deterministic_fraction": round(sum(workflow["deterministic_fraction"] for workflow in workflows) / len(workflows), 4),
        "observability_declared_coverage": round(observability_declared_coverage, 4),
        "telemetry_observed_coverage": round(telemetry_observed_coverage, 4),
        "telemetry_validated_coverage": round(telemetry_validated_coverage, 4),
        "telemetry_coverage": round(telemetry_validated_coverage, 4),
        "evidence_quality": portfolio_evidence_quality(workflows, telemetry_observed_coverage, telemetry_validated_coverage),
    }


def precompute(workflows_root: Path, otel_summary_path: str | None = None) -> dict[str, Any]:
    telemetry_index = load_otel_summary(otel_summary_path)
    workflow_files = discover_workflow_files(workflows_root)
    workflows = [build_workflow_record(path, workflows_root, telemetry_index) for path in workflow_files]
    similarities, documents = compute_similarity_matrix(workflows)
    overlap_by_path: dict[str, float] = defaultdict(float)
    overlap_peers: dict[str, dict[str, float]] = defaultdict(dict)
    for (left, right), similarity in similarities.items():
        squared = similarity * similarity
        overlap_by_path[left] += squared
        overlap_by_path[right] += squared
        overlap_peers[left][right] = similarity
        overlap_peers[right][left] = similarity
    for workflow in workflows:
        workflow["overlap_drag"] = round_score(overlap_by_path.get(workflow["path"], 0.0))
        workflow["maintenance_drag"] = round_score(workflow["maintenance_drag"] + workflow["overlap_drag"] * 0.2)
        workflow["yield"] = compute_workflow_yield(
            workflow["usefulness"],
            workflow["adoption"],
            workflow["trust"],
            workflow["cost"],
            workflow["risk"],
            workflow["maintenance_drag"],
            workflow["overlap_drag"],
        )
        workflow["overlap_peers"] = overlap_peers.get(workflow["path"], {})
    overlap_clusters = build_overlap_clusters(workflows, similarities, documents)
    clustered_paths = {path for cluster in overlap_clusters for path in cluster["workflows"]}
    for workflow in workflows:
        workflow["recommendation_seed"] = assign_recommendation(workflow, clustered_paths)
    overlap_drag_value = portfolio_overlap_drag(similarities)
    portfolio_metrics = compute_portfolio_metrics(workflows, overlap_drag_value)
    telemetry_coverage = {
        "observability_declared_coverage": portfolio_metrics["observability_declared_coverage"],
        "telemetry_observed_coverage": portfolio_metrics["telemetry_observed_coverage"],
        "telemetry_validated_coverage": portfolio_metrics["telemetry_validated_coverage"],
        "observability_declared_workflows": [workflow["path"] for workflow in workflows if workflow.get("observability_declared")],
        "observed_workflows": [workflow["path"] for workflow in workflows if workflow.get("telemetry_observed")],
        "validated_workflows": [workflow["path"] for workflow in workflows if workflow.get("telemetry_validated")],
        "declared_without_observation": [
            workflow["path"]
            for workflow in workflows
            if workflow.get("observability_declared") and not workflow.get("telemetry_observed")
        ],
        "observed_without_validation": [
            workflow["path"]
            for workflow in workflows
            if workflow.get("telemetry_observed") and not workflow.get("telemetry_validated")
        ],
        "missing_workflows": [
            workflow["path"]
            for workflow in workflows
            if not workflow.get("observability_declared")
        ],
    }
    episode_metrics = compute_episode_metrics(workflows, similarities)
    organizational_health = compute_organizational_health(workflows, overlap_drag_value)
    return {
        "portfolio_metrics": portfolio_metrics,
        "workflows": workflows,
        "overlap_clusters": overlap_clusters,
        "telemetry_coverage": telemetry_coverage,
        "episode_metrics": episode_metrics,
        "organizational_health_signals": organizational_health,
        "recommendations_seed": build_recommendation_seed(workflows),
        "overlap_pairs": [
            {"left": left, "right": right, "score": score}
            for (left, right), score in sorted(similarities.items())
        ],
    }


def parse_args(argv: list[str] | None = None) -> argparse.Namespace:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--workflows", required=True, help="Path to the .github/workflows directory")
    parser.add_argument("--out", required=True, help="Output JSON path")
    return parser.parse_args(argv)


def main(argv: list[str] | None = None) -> int:
    args = parse_args(argv)
    workflows_root = Path(args.workflows)
    payload = precompute(workflows_root, os.environ.get("AWY_OTEL_SUMMARY_JSON"))
    output_path = Path(args.out)
    output_path.parent.mkdir(parents=True, exist_ok=True)
    output_path.write_text(json.dumps(payload, indent=2, sort_keys=True) + "\n", encoding="utf-8")
    return 0


if __name__ == "__main__":
    sys.exit(main())
