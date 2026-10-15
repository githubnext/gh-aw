"""Recompute code summaries and an overlapping weekly histogram from local Git."""

from __future__ import annotations

import argparse
from dataclasses import asdict
import json
from pathlib import Path

from audit_sources import collect_git
from render_infographic import Canvas, LayoutError, discover_fonts, draw_change_histogram, save
from statistics_data import DataError, change_histogram


def compute(repo: Path, repository: str, commit: str, output: Path) -> None:
    groups, weeks, _ = collect_git(repo, commit)
    bins = change_histogram(weeks)
    summaries = {
        "repo-structure-statistics.json": {
            "repository": repository, "commit": commit,
            "included_files": sum(g.files for g in groups),
            "nonblank_lines": sum(g.lines for g in groups),
            "groups": [{"language": g.language, "role": g.role, "files": g.files,
                        "nonblank_lines": g.lines} for g in groups],
        },
        "weekly-code-history.json": {
            "repository": repository, "commit": commit, "week_count": len(weeks),
            "total_added": sum(w.added for w in weeks),
            "total_deleted": sum(w.deleted for w in weeks),
            "weeks": [{"week_start": w.start.isoformat(), "added": w.added, "deleted": w.deleted}
                      for w in weeks],
        },
        "weekly-change-histogram.json": {
            "repository": repository, "commit": commit, "weeks_per_series": len(weeks),
            "bounds": "lower inclusive, upper exclusive",
            "bins": [asdict(bucket) for bucket in bins],
        },
    }
    canvas = Canvas(discover_fonts(None, None), height=1180)
    canvas.text(repository.upper(), 72, 45, 38, bold=True)
    canvas.text("Weekly lines added / removed", 72, 110, 72, bold=True)
    draw_change_histogram(canvas, bins, len(weeks), 112, 270, 2336)
    canvas.text("First-parent Go / JS diff lines, including blanks; shared equal-width bins; counts of weeks.",
                112, 890, 34)
    canvas.text("Zeros included. Renames count delete/add; boundary weeks may be partial. Local Git only.",
                112, 950, 34)
    canvas.text(f"Code: {commit}", 112, 1020, 30)
    canvas.verify()
    output.mkdir(parents=True, exist_ok=True)
    save(canvas, output / "weekly-change-histogram.png")
    for name, summary in summaries.items():
        (output / name).write_text(json.dumps(summary, indent=2) + "\n", encoding="utf-8")


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--repo", type=Path, required=True)
    parser.add_argument("--repository", required=True, help="Repository owner/name for snapshot metadata.")
    parser.add_argument("--commit", required=True, help="Full immutable commit object ID; no refs or fetching.")
    parser.add_argument("--output-dir", type=Path, required=True)
    args = parser.parse_args()
    try:
        compute(args.repo, args.repository, args.commit, args.output_dir)
    except (DataError, LayoutError, OSError, ValueError) as error:
        parser.error(str(error))
    print(f"Recomputed code map, weekly history and overlapping histogram in {args.output_dir}; no network access used.")


if __name__ == "__main__":
    main()
