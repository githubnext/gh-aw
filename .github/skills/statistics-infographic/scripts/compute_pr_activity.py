"""Compare comments and commits per PR between PR-author account cohorts."""

from __future__ import annotations

import argparse
from dataclasses import asdict
import json
from pathlib import Path

from compute_pr_changes import load_records
from render_infographic import Canvas, LayoutError, discover_fonts, draw_activity, save
from statistics_data import DataError, activity_groups


def compute(raw: Path, census: Path, output: Path) -> None:
    repository, snapshot, records = load_records(raw, census)
    activity = activity_groups(records)
    summary = {"repository": repository, "snapshot_date": snapshot.isoformat(), "pr_count": len(records),
               "cohorts": [asdict(group) for group in activity]}
    canvas = Canvas(discover_fonts(None, None), height=1180)
    canvas.text(repository.upper(), 72, 45, 38, bold=True)
    canvas.text("Comments and commits per PR", 72, 110, 72, bold=True)
    draw_activity(canvas, activity, 112, 270, 2336)
    canvas.text(f"Grouped by PR-author account, not session initiator. Unattributed: {activity[2].prs:,} PRs (excluded).",
                112, 890, 34)
    canvas.text("Conversation comments only; excludes inline review comments/reviews. Agent/bot includes non-AI bots.",
                112, 950, 34)
    canvas.text(f"PR census: {snapshot.isoformat()} | All census PR states | Means, not medians.",
                112, 1020, 30)
    canvas.verify()
    output.mkdir(parents=True, exist_ok=True)
    save(canvas, output / "pr-activity.png")
    (output / "pr-activity-summary.json").write_text(json.dumps(summary, indent=2) + "\n", encoding="utf-8")


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--raw-dir", type=Path, required=True)
    parser.add_argument("--census", type=Path, required=True)
    parser.add_argument("--output-dir", type=Path, required=True)
    args = parser.parse_args()
    try:
        compute(args.raw_dir, args.census, args.output_dir)
    except (DataError, LayoutError, OSError, ValueError) as error:
        parser.error(str(error))
    print(f"Recomputed human/agent PR comment and commit comparison in {args.output_dir}; no network access used.")


if __name__ == "__main__":
    main()
