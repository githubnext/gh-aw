---
name: statistics-infographic
description: Generate a readable seven-panel repository statistics infographic, including overlapping added/removed-line histograms, from validated JSON and pinned Git evidence.
---

# Repository statistics infographic

Use when asked to consolidate repository-statistics slides into one image, regenerate
the infographic with updated data, or verify its calculations. This is a reusable
seven-panel statistical template, not an arbitrary PowerPoint screenshot collage.

The panels cover inferred PR initiation, merge/closed outcomes, start-to-merge
durations, community issue outcomes, Go/JavaScript code composition, and weekly
first-parent code changes, plus overlapping histograms of lines added and removed
for individual PRs. Repository names, dates, counts, proportions, and weeks
come from the input, never from this skill's original dataset.

## Generate

Prefer already collected source evidence. Do not fetch repository data, trigger
workflows, or invent missing statistics. Keep raw data and generated images out of
the repository. Use a temporary or session artifact directory.

Create a Python 3.9+ virtual environment and install the runtime manifest:

```bash
python3 -m venv /tmp/statistics-infographic-venv
/tmp/statistics-infographic-venv/bin/python -m pip install \
  -r .github/skills/statistics-infographic/requirements.txt
```

Supply either one JSON object following
[examples/example.json](examples/example.json), or a directory of existing summaries:

```bash
/tmp/statistics-infographic-venv/bin/python \
  .github/skills/statistics-infographic/scripts/render_infographic.py \
  --input .github/skills/statistics-infographic/examples/example.json \
  --output /tmp/statistics-infographic.png
```

For real data, replace `--input` with `--data-dir /path/to/summaries`.
See [references/data-contract.md](references/data-contract.md) for schemas and
statistical definitions. The example is explicitly synthetic, not a gh-aw report.

Optional arguments: `--title`, `--font-regular`, and `--font-bold`. Supply both font
paths together. Automatic discovery supports Arial on macOS/Windows and Liberation
Sans or DejaVu Sans on Linux. Missing fonts fail explicitly; no bitmap fallback is
used.

The output is a static, RGB, 2560 x 4120 PNG with 300-DPI metadata. A successful
render atomically replaces the explicit output path. Invalid data, unreadable
layout, or invalid fonts leave an existing output intact. Counts too large for
their slots, more than 10 duration bands, or more than 260 weeks require a different
layout; do not silently truncate, rescale text to illegibility, or aggregate data.

### Recompute individual PR change sizes

The seventh panel counts **individual PRs**, never weeks, commits, or files.
Each census PR contributes its GitHub `additions` value to one bin of the added
series and its `deletions` value to one bin of the removed series, including zeros.
The x-axis is lines per PR; the y-axis is number of PRs. Both series use the same
equal-width integer bins; bars overlap with transparency, not stacking or mirroring.
The population includes merged, closed-unmerged, and open census PRs across all
files, unlike the Go/JS-only weekly history panel.

To recompute `pr-change-histogram.json` and a standalone image from cached raw
records containing individual PR `number`, `additions`, `deletions`, `state`,
and `createdAt` fields:

```bash
/tmp/statistics-infographic-venv/bin/python \
  .github/skills/statistics-infographic/scripts/compute_pr_changes.py \
  --raw-dir /path/to/raw-records \
  --census /path/to/summaries/pr-initiation-assumption-summary.json \
  --output-dir /path/to/summaries
```

This writes `pr-change-histogram.json` and `pr-change-histogram.png` (2560 x 1180).
The summary is the required `pr_changes` section in the combined input, or the
sixth canonical file for `--data-dir`. Re-render the complete image with all six
summaries. Missing per-PR additions/deletions fail explicitly; never substitute
weekly Git totals, merge-commit churn, or missing values as zero.
The command never fetches or refreshes API snapshots. If raw PR diff statistics
are unavailable, disclose that real-data recomputation is blocked. Keep any
rendered synthetic example clearly labelled and separate.

## Verify accuracy

Input validation proves internal reconciliation, not correctness of the evidence.
When cached raw API records are available, independently recompute the displayed
statistics before delivery:

```bash
/tmp/statistics-infographic-venv/bin/python \
  .github/skills/statistics-infographic/scripts/audit_sources.py \
  --data-dir /path/to/summaries \
  --raw-dir /path/to/raw-records \
  --repo /path/to/local-checkout
```

The auditor also accepts `--input` instead of `--data-dir`. Omit `--repo` only if
local Git evidence is unavailable, and disclose that code/history were not audited.
For a repository migration, explicitly supply a repeatable
`--repository-alias previous-owner/previous-name` only when issue numbering is
known to be shared.

The Git audit reads immutable objects at the full input commit, not the worktree.
It requires complete local history, disables lazy object fetching, and never
requests credentials or performs network operations. Missing objects or shallow
history fail explicitly. Large repositories may require substantial memory for
the batched blob inventory.

Open the resulting image and inspect all seven panels at normal viewing size:
titles, axes, labels, footnotes, chart proportions, and snapshot metadata. Automated
text-bound and overlap checks complement but do not replace visual inspection.

Deliver the image with its path and disclose missing independent checks or evidence
limitations. In particular, inferred source issues do not prove who initiated an
agent session or which device they used. The no-issue human assumption must be
explicitly authorized for the dataset.

## Maintain

Install the development manifest only for validation work:

```bash
/tmp/statistics-infographic-venv/bin/python -m pip install \
  -r .github/skills/statistics-infographic/requirements-dev.txt
HYPOTHESIS_STORAGE_DIRECTORY=/tmp/statistics-infographic-hypothesis \
  /tmp/statistics-infographic-venv/bin/python -m unittest discover \
  -s .github/skills/statistics-infographic/tests -v
/tmp/statistics-infographic-venv/bin/python -m mypy --strict \
  .github/skills/statistics-infographic/scripts
```

The implementation uses immutable typed records, explicit invariants, exact
rational count ratios with half-up rounding, Decimal-oracle property tests,
source-mutation regressions, deterministic rendering, safe CLI output tests, and
pinned-object Git fixtures. These are rigorous software checks, not a mathematical
proof of the whole program or of causal attribution.
