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

Next enumeration index: **4** → `tiny-none-single-micro-ahead-multi`.
