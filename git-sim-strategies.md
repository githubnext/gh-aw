# Git Simulator Strategy Notes

Observations and patterns discovered across runs. Used to prioritize high-risk
configuration cells in the 3600-cell space.

## Run Log

### 2026-06-19 (run 27808553938) — cells 0-3

Tested the first 4 cells of the deterministic enumeration (all in the
`tiny-none-single-micro-*` corner). All **passed**.

| config_id | outcome | patch files | patch KB | commits |
|-----------|---------|-------------|----------|---------|
| tiny-none-single-micro-clean-single | pass | 1 | 1.19 | 1 |
| tiny-none-single-micro-clean-multi | pass | 3 | 3.10 | 3 |
| tiny-none-single-micro-clean-merge_msg | pass | 1 | 1.52 | 1 |
| tiny-none-single-micro-ahead-single | pass (both) | 2 | 2.58 | 2 |

### 2026-06-20 (run 27861930502) — cells 4-7

Tested the next 4 cells, completing the `tiny-none-single-micro-{ahead,diverged}`
sub-corner. All **passed**. Exercised `ahead`/`diverged` branch states and the
`merge_msg` commit structure under append-only push.

| config_id | outcome | patch files | patch KB | commits | merges | push |
|-----------|---------|-------------|----------|---------|--------|------|
| tiny-none-single-micro-ahead-multi | pass (both) | 3 | 4.58 | 3 | 0 | +1 append → 4 |
| tiny-none-single-micro-ahead-merge_msg | pass (both) | 1 | 1.61 | 1 | 0 | +1 append → 2 |
| tiny-none-single-micro-diverged-single | pass (both) | 1 | 1.54 | 1 | 0 | +1 append → 2 |
| tiny-none-single-micro-diverged-multi | pass (both) | 3 | 4.58 | 3 | 0 | +1 append → 4 |

New confirmations this run:
- **merge_msg verified at the rev-list level**: `git rev-list --merges` returns 0
  for the single-parent "Merge branch ..." commit — confirms the earlier
  hypothesis that a merge-style *message* is structurally a normal commit and
  format-patch / append-only push handle it without misclassification.
- **diverged push is safe when append-only**: simulated the base branch advancing
  independently (a divergent commit on `main` not merged into `feature`), then
  pushed by appending one commit to `feature`. Append-only push is unaffected by
  base divergence because no `git merge main` is performed on the PR branch —
  consistent with the documented guidance to use rebase, never merge, into a PR
  branch. A real rejection would only be expected if a workflow merged base in
  (creating a 2-parent commit) — not yet exercised.

## Patterns / Hypotheses (to validate as coverage grows)

- **Baseline corner is healthy**: the smallest configs (tiny/none/single/micro)
  pass cleanly for create_pull_request, multi-commit, merge-style messages, and
  append-only push. No boundary effects observed yet at this scale.
- **format-patch overhead is per-commit**: COMMIT=multi inflated total patch
  size to ~3.1 KB for ~1 KB of content (3 commits × per-commit email/diff
  headers). Watch this when PATCH is near a size threshold and COMMIT=multi —
  the header overhead could push a borderline patch over a limit.
- **"Merge branch ..." message is NOT a merge commit**: a single-parent commit
  with a merge-style message is processed normally by format-patch. A naive
  message-text heuristic (vs. parent-count / `--no-merges`) could misclassify
  and wrongly reject. Flag this if a real rejection appears on a merge_msg cell.
- **Likely high-risk cells to prioritize later**: PATCH=xlarge (4000 KB) and
  FILES=batch (100 files), especially combined with COMMIT=multi (header
  overhead) and HISTORY=deep. These are the candidates for size-limit rejections
  or timeouts. Not yet reached by the enumeration.

## Next

Next enumeration index: **8** → `tiny-none-single-micro-diverged-merge_msg`.

This exhausts the `tiny-none-single-micro-*` corner after index 8; index 9 moves
to `tiny-none-single-small-clean-single` (first PATCH=small cell).
