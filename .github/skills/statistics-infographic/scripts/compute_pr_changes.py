"""Recompute an overlapping per-PR diff histogram from cached GitHub records."""

from __future__ import annotations

import argparse
from dataclasses import asdict
from datetime import date
import json
from pathlib import Path

from audit_sources import jsonl, timestamp
from render_infographic import Canvas, LayoutError, discover_fonts, draw_change_histogram, save
from statistics_data import DataError, change_histogram, count, mapping, pr_changes, require, string


def load_records(raw: Path, census: Path) -> tuple[str, date, list[dict[str, object]]]:
    source = mapping(json.loads(census.read_text(encoding="utf-8")), "PR census")
    repository = string(source.get("repository"), "repository")
    snapshot = date.fromisoformat(string(source.get("snapshot_date"), "snapshot_date"))
    total = count(source.get("total_prs"), "total_prs")
    require(total > 0, "PR census must be positive")
    records = jsonl(raw / "pr-statistics-data.jsonl")
    require(len(records) == total, "raw PR population differs from census")
    require(all(timestamp(r.get("createdAt")).date() <= snapshot for r in records),
            "PR creation is after the declared snapshot date")
    require(all(r.get("state") in {"MERGED", "CLOSED", "OPEN"} for r in records), "unknown PR state")
    return repository, snapshot, records


def compute(raw: Path, census: Path, output: Path) -> None:
    repository, snapshot, records = load_records(raw, census)
    changes = pr_changes(records)
    total = len(changes)
    bins = change_histogram(changes)
    summary = {"repository": repository, "snapshot_date": snapshot.isoformat(), "pr_count": total,
               "bins": [asdict(bucket) for bucket in bins]}
    canvas = Canvas(discover_fonts(None, None), height=1180)
    canvas.text(repository.upper(), 72, 45, 38, bold=True)
    canvas.text("Lines added / removed per PR", 72, 110, 72, bold=True)
    draw_change_histogram(canvas, bins, total, 112, 270, 2336)
    canvas.text("GitHub PR additions/deletions across all files; merged, closed-unmerged and open PRs.",
                112, 890, 34)
    canvas.text("Every census PR contributes once to both series, including zeros. Not weekly Git churn.",
                112, 950, 34)
    canvas.text(f"PR census: {snapshot.isoformat()} | Cached GitHub API records; no network access used.",
                112, 1020, 30)
    canvas.verify()
    output.mkdir(parents=True, exist_ok=True)
    save(canvas, output / "pr-change-histogram.png")
    (output / "pr-change-histogram.json").write_text(json.dumps(summary, indent=2) + "\n", encoding="utf-8")


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--raw-dir", type=Path, required=True)
    parser.add_argument("--census", type=Path, required=True, help="Existing PR initiation census summary.")
    parser.add_argument("--output-dir", type=Path, required=True)
    args = parser.parse_args()
    try:
        compute(args.raw_dir, args.census, args.output_dir)
    except (DataError, LayoutError, OSError, ValueError) as error:
        parser.error(str(error))
    print(f"Recomputed overlapping per-PR additions/deletions histogram in {args.output_dir}; no network access used.")


if __name__ == "__main__":
    main()
