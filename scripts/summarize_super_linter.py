"""Count complete Super-Linter diagnostics without relying on model excerpts."""

import argparse
import json
import re
from collections import Counter, defaultdict
from pathlib import Path


DIAGNOSTIC = re.compile(
    r"^(?P<file>.+\.go):(?P<line>\d+):(?P<column>\d+): "
    r"(?P<message>.+) \((?P<linter>[^()\s]+)\)$"
)
FIXTURE = re.compile(r"(^|/)pkg/linters/(?:.*/)?testdata/")
ANSI = re.compile(r"\x1b\[[0-?]*[ -/]*[@-~]")
TOOLING_MESSAGE = re.compile(
    r"unknown linters|context loading failed|failed to load packages|"
    r"no go files to analyze|timeout exceeded",
    re.IGNORECASE,
)


def summarize_log(log, outcome):
    if outcome not in ("success", "failure"):
        raise ValueError("Super-linter outcome must be success or failure")
    if not log.strip():
        raise ValueError("Super-linter log must not be empty")

    diagnostics = {}
    tooling_messages = set()
    for raw_line in log.splitlines():
        line = ANSI.sub("", raw_line)
        match = DIAGNOSTIC.fullmatch(line)
        if match:
            diagnostic = match.groupdict()
            # Columns are not part of the workflow's diagnostic identity.
            key = tuple(diagnostic[field] for field in ("file", "line", "linter", "message"))
            diagnostics[key] = diagnostic
        elif TOOLING_MESSAGE.search(line):
            tooling_messages.add(line)

    counts = Counter({"typecheck": 0, "fixtures": 0, "source": 0})
    groups = defaultdict(list)
    for _, diagnostic in sorted(diagnostics.items()):
        if FIXTURE.search(diagnostic["file"]):
            category = "fixtures"
        elif diagnostic["linter"] == "typecheck":
            category = "typecheck"
        else:
            category = "source"
        counts[category] += 1
        groups[(category, diagnostic["linter"])].append(diagnostic)

    coverage_complete = (
        counts["typecheck"] == 0
        and counts["fixtures"] == 0
        and not tooling_messages
        and outcome == "success"
    )
    examples_remaining = 20
    summary_groups = []
    for (category, linter), findings in sorted(groups.items()):
        examples = findings[: min(3, examples_remaining)]
        examples_remaining -= len(examples)
        summary_groups.append(
            {
                "category": category,
                "linter": linter,
                "count": len(findings),
                "examples": examples,
            }
        )
    return {
        "outcome": outcome,
        "diagnostic_count": len(diagnostics),
        "counts": dict(counts),
        "tooling_message_count": len(tooling_messages),
        "coverage_complete": coverage_complete,
        "title_suffix": (
            f"{counts['source']} source findings"
            if coverage_complete
            else "analysis incomplete"
        ),
        "groups": summary_groups,
    }


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--log", required=True, type=Path)
    parser.add_argument("--outcome", required=True, choices=("success", "failure"))
    args = parser.parse_args()
    try:
        summary = summarize_log(args.log.read_text(), args.outcome)
    except (OSError, UnicodeError, ValueError) as error:
        parser.error(str(error))
    print(json.dumps(summary, indent=2))


if __name__ == "__main__":
    main()
