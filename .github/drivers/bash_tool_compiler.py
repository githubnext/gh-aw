#!/usr/bin/env python3
"""
bash_tool_compiler.py

A Python implementation of the bash command parser used by the Copilot SDK
permission checker.  Provides utilities to:

  - Split a shell command text on pipeline operators (&&, ||, |, ;)
  - Extract the executable command name from a shell segment
  - Extract all command names from a complex piped/chained command
  - Check whether a piped command is allowed by a set of shell rules

This module mirrors the logic in bash_command_parser.cjs and
copilot_sdk_driver.cjs so that Python-based Copilot SDK drivers can apply
the same permission-checking semantics as the Node.js driver.

Security invariant:
  When the parser cannot extract command names (empty command text, all
  segments are shell keywords or redirections, etc.) the helpers return
  an empty list/False, ensuring the caller denies the request by default.
"""

from __future__ import annotations

import re
from typing import Optional

# ---------------------------------------------------------------------------
# Shell constants
# ---------------------------------------------------------------------------

# Keywords that may appear as the first word of a segment but are not
# executable commands.
_SHELL_KEYWORDS: frozenset[str] = frozenset(
    ["then", "else", "elif", "fi", "do", "done", "esac", "in", "function", "time", "coproc"]
)

# Regex to detect leading env-var assignment: WORD=anything
_ENV_ASSIGN_RE = re.compile(r"^[A-Za-z_][A-Za-z0-9_]*=\S*")

# Regex to detect redirection operators at the start of a word
_REDIR_RE = re.compile(r"^([<>]|\d+[<>&])")


# ---------------------------------------------------------------------------
# splitOnPipelineOperators
# ---------------------------------------------------------------------------


def split_on_pipeline_operators(command_text: str) -> list[str]:
    """Split a shell command text into individual pipeline segments.

    Splits on the following shell operators: ``&&``, ``||``, ``|``, ``;``

    The split respects:
    - Single-quoted strings (no escaping inside)
    - Double-quoted strings (backslash-escape aware)
    - ``$(...)`` subshell expressions (balanced parentheses)

    Operators embedded inside any of these constructs are NOT treated as
    separators.

    Parameters
    ----------
    command_text:
        Raw bash command text that may contain pipeline operators.

    Returns
    -------
    list[str]
        Non-empty trimmed segments (operators removed).
    """
    if not command_text or not isinstance(command_text, str):
        return []

    segments: list[str] = []
    current: list[str] = []
    i = 0
    n = len(command_text)

    while i < n:
        ch = command_text[i]

        # ── Single-quoted string: no escape sequences ──────────────────────
        if ch == "'":
            current.append(ch)
            i += 1
            while i < n and command_text[i] != "'":
                current.append(command_text[i])
                i += 1
            if i < n:
                current.append(command_text[i])  # closing '
                i += 1
            continue

        # ── Double-quoted string: backslash escapes ────────────────────────
        if ch == '"':
            current.append(ch)
            i += 1
            while i < n and command_text[i] != '"':
                if command_text[i] == "\\" and i + 1 < n:
                    current.append(command_text[i])
                    current.append(command_text[i + 1])
                    i += 2
                else:
                    current.append(command_text[i])
                    i += 1
            if i < n:
                current.append(command_text[i])  # closing "
                i += 1
            continue

        # ── $(...) subshell: balanced parentheses ─────────────────────────
        if ch == "$" and i + 1 < n and command_text[i + 1] == "(":
            current.append(ch)
            i += 1
            depth = 0
            while i < n:
                sc = command_text[i]
                if sc == "(":
                    depth += 1
                elif sc == ")":
                    depth -= 1
                    current.append(sc)
                    i += 1
                    if depth == 0:
                        break
                    continue
                current.append(sc)
                i += 1
            continue

        # ── Pipeline operators ─────────────────────────────────────────────

        # && (AND-then)
        if ch == "&" and i + 1 < n and command_text[i + 1] == "&":
            segments.append("".join(current))
            current = []
            i += 2
            while i < n and command_text[i].isspace():
                i += 1
            continue

        # || (OR-else) — must be checked before lone |
        if ch == "|" and i + 1 < n and command_text[i + 1] == "|":
            segments.append("".join(current))
            current = []
            i += 2
            while i < n and command_text[i].isspace():
                i += 1
            continue

        # | (pipe)
        if ch == "|":
            segments.append("".join(current))
            current = []
            i += 1
            while i < n and command_text[i].isspace():
                i += 1
            continue

        # ; (sequential)
        if ch == ";":
            segments.append("".join(current))
            current = []
            i += 1
            while i < n and command_text[i].isspace():
                i += 1
            continue

        current.append(ch)
        i += 1

    # Push the final segment
    tail = "".join(current).strip()
    if tail:
        segments.append(tail)

    return [s.strip() for s in segments if s.strip()]


# ---------------------------------------------------------------------------
# extractCommandName
# ---------------------------------------------------------------------------


def extract_command_name(segment: str) -> Optional[str]:
    """Extract the executable command name from a single shell command segment.

    Skips:
    - Leading env-var assignments (``VAR=value``, any number)
    - Shell negation operator ``!``
    - Shell grouping braces ``{`` and ``}``
    - Redirection words that begin with ``<``, ``>`` or a digit followed by
      ``<``, ``>`` or ``&``
    - Shell flow-control keywords (``then``, ``else``, ``fi``, ``do``, etc.)

    Parameters
    ----------
    segment:
        A single shell segment containing no pipeline operators.

    Returns
    -------
    str or None
        The command name, or ``None`` if it cannot be determined.
    """
    if not segment or not isinstance(segment, str):
        return None

    remaining = segment.strip()
    if not remaining:
        return None

    # Skip leading env-var assignments
    while remaining:
        m = _ENV_ASSIGN_RE.match(remaining)
        if not m:
            break
        remaining = remaining[m.end():].lstrip()

    if not remaining:
        return None

    # Get the first word
    parts = remaining.split(None, 1)
    if not parts:
        return None

    word = parts[0]

    # Redirection operators
    if _REDIR_RE.match(word):
        return None

    # Shell negation / grouping — recurse on the remainder
    if word in ("!", "{", "}"):
        rest = parts[1].strip() if len(parts) > 1 else ""
        return extract_command_name(rest)

    # Flow-control keywords are not executable commands
    if word in _SHELL_KEYWORDS:
        return None

    return word


# ---------------------------------------------------------------------------
# extractCommandNamesFromPipeline
# ---------------------------------------------------------------------------


def extract_command_names_from_pipeline(command_text: str) -> list[str]:
    """Extract all unique command names from a bash pipeline or command sequence.

    Splits the text on ``&&``, ``||``, ``|``, and ``;`` and extracts the
    executable command name from each resulting segment.  Returns a
    deduplicated list preserving first-occurrence order.

    Returns an empty list when the text is empty, unparseable, or yields no
    recognisable command names.  Callers should treat an empty result as
    "unable to determine commands" and fall back to a safe default (deny).

    Parameters
    ----------
    command_text:
        Raw bash command text (may include pipeline operators).

    Returns
    -------
    list[str]
        Deduplicated list of command names in first-occurrence order.
    """
    if not command_text or not isinstance(command_text, str):
        return []

    text = command_text.strip()
    if not text:
        return []

    segments = split_on_pipeline_operators(text)
    seen: set[str] = set()
    names: list[str] = []

    for segment in segments:
        name = extract_command_name(segment)
        if name and name not in seen:
            seen.add(name)
            names.append(name)

    return names


# ---------------------------------------------------------------------------
# Permission-checking helpers
# ---------------------------------------------------------------------------


def is_identifier_allowed_by_shell_rules(identifier: str, shell_rules: list[str]) -> bool:
    """Check whether a single command identifier is permitted by shell rules.

    Only single-word rules and ``:*`` prefix rules are matched.  Exact
    full-command rules (rules that contain a space) are intentionally skipped
    because they are not meaningful for individual pipeline stages.

    Parameters
    ----------
    identifier:
        A single command name (e.g. ``ls``, ``git``, ``safeoutputs``).
    shell_rules:
        List of shell rule strings extracted from ``shell(...)`` allow-tool
        entries.  Examples: ``"cat"``, ``"git:*"``, ``"git status"``.

    Returns
    -------
    bool
        ``True`` when any rule permits the identifier.
    """
    for rule in shell_rules:
        if rule.endswith(":*"):
            prefix = rule[:-2].strip()
            if prefix and identifier == prefix:
                return True
        elif " " not in rule:
            if identifier == rule:
                return True
    return False


def is_pipeline_allowed(command_text: str, shell_rules: list[str]) -> bool:
    """Check whether a piped / chained bash command is allowed by shell rules.

    Parses the full command text to extract individual stage command names and
    verifies that **every** stage is individually permitted.  Returns ``False``
    when no command names can be extracted (safe default: deny).

    This function is the Python equivalent of the pipeline-aware fallback path
    in the JavaScript ``isAllowed`` function in ``copilot_sdk_driver.cjs``.

    Parameters
    ----------
    command_text:
        Full bash command text (may include ``&&``, ``||``, ``|``, ``;``).
    shell_rules:
        List of shell rule strings.  Examples: ``["cat", "ls", "echo",
        "safeoutputs:*", "gh:*"]``.

    Returns
    -------
    bool
        ``True`` only when ALL extracted command names are individually
        permitted and at least one command name was extracted.
    """
    names = extract_command_names_from_pipeline(command_text)
    if not names:
        return False
    return all(is_identifier_allowed_by_shell_rules(name, shell_rules) for name in names)
