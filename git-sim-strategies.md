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

### 2026-06-21 (run 27895339281) — cells 8-11

Completed index 8 (last `tiny-none-single-micro-*` cell) and entered the first
three PATCH=small cells. All **passed**. First exercise of a non-trivial patch
size (~50 KB) at SIZE=tiny/HISTORY=none.

| config_id | outcome | patch files | patch KB | commits | merges | push |
|-----------|---------|-------------|----------|---------|--------|------|
| tiny-none-single-micro-diverged-merge_msg | pass (both) | 2 | 2.32 | 2 | 0 | +1 append → 2 |
| tiny-none-single-small-clean-single | pass | 1 | 51.3 | 1 | 0 | none |
| tiny-none-single-small-clean-multi | pass | 3 | 51.6 | 3 | 0 | none |
| tiny-none-single-small-clean-merge_msg | pass | 1 | 51.4 | 1 | 0 | none |

New confirmations this run:
- **PATCH=small (50 KB) is well within limits** for create_pull_request at
  tiny/none. Measured patch sizes (51.3–51.6 KB) track the 50 KB target closely;
  format-patch header overhead is negligible at this scale (~1.4–1.6 KB even
  with COMMIT=multi spreading 3 email headers). No size-limit effects.
- **High-entropy content matters for measurement**: agents used base64 random
  text so the diff did not deflate trivially. `git bundle` cross-checks were
  ~38 KB (zlib gain) vs ~51 KB raw patch — the 50 KB target is the *patch* size,
  not the compressed bundle. Keep using non-compressible content so PATCH targets
  are honest near future thresholds.
- **merge_msg re-confirmed at 50 KB**: `git rev-list --merges` empty; single
  parent; format-patch names the file from the "Merge branch ..." subject but
  processes it as a normal commit. Consistent across micro and small.
- **diverged append-only still safe at this scale**: base advanced
  independently (modifying history.md), feature got an append-only commit, no
  `git merge` on feature. Net `main..feature` diff surfaced history.md as a
  second changed file purely from base divergence — an artifact of the diverged
  setup, not the declared single-file payload. Worth noting: the *net diff* file
  count can exceed the declared FILES dimension when BRANCH=diverged touches the
  same base files differently.

### 2026-06-22 (run 27933136829) — cells 12-15

Tested indices 12-15: the PATCH=small / BRANCH={ahead,diverged} cells at
SIZE=tiny / HISTORY=none. All **passed**. First exercise of append-only push at
the ~50 KB patch scale across all three commit structures.

| config_id | outcome | patch files | patch KB | commits (final) | merges | push |
|-----------|---------|-------------|----------|-----------------|--------|------|
| tiny-none-single-small-ahead-single | pass (both) | 1 | 49.9 | 2 | 0 | +1 append → 2 |
| tiny-none-single-small-ahead-multi | pass (both) | 1 | 50.1 | 4 | 0 | +1 append → 4 |
| tiny-none-single-small-ahead-merge_msg | pass (both) | 1 | 48.7 | 2 | 0 | +1 append → 2 |
| tiny-none-single-small-diverged-single | pass (both) | 2 | 49.1 | 2 | 0 | +1 append → 2 |

New confirmations this run:
- **Append-only push is fast-forward at 50 KB**: `merge-base --is-ancestor`
  confirmed the appended commit was a clean fast-forward (no force-push, no
  `git merge`) on every ahead/diverged cell. Patch sizes tracked the 50 KB
  target within ~3% (48.7–50.1 KB) using base64 /dev/urandom content.
- **merge_msg filename leak re-confirmed at 50 KB**: the "Merge branch 'x' into
  feature" single-parent commit yields an EMPTY `git rev-list --merges` and
  `parent` count = 1, but `format-patch` names the artifact
  `0001-Merge-branch-x-into-feature.patch`. The misleading *filename* derives
  from the subject even though the commit is structurally normal — cosmetic
  artifact, not a rejection, but the clearest signal yet that any message-text
  merge heuristic downstream would misfire here.
- **COMMIT=multi header overhead stays negligible at 50 KB**: 3-commit split
  added ~30 metadata lines vs ~636 payload lines; total still landed at 50.1 KB
  (essentially exact). Header overhead is not a size-limit factor at this scale.
- **diverged inflates net file count again**: declared 1-file payload, but
  `git diff --stat main..feature` reported 3 net files (payload.bin, plus
  stuff.md from the append commit, plus history.md showing as a deletion purely
  because main advanced independently). Consistent with the 06-21 observation —
  the net diff over-counts vs the declared FILES dimension whenever
  BRANCH=diverged. Also: the append-only push step necessarily makes
  actual_commit_count ≥ 2 even when COMMIT=single declares 1.

### 2026-06-23 (run 28004973333) — cells 16-19

Tested indices 16-19: the last two PATCH=small / BRANCH=diverged cells, then the
first two PATCH=medium (200 KB) / BRANCH=clean cells. All **passed**. First
exercise of the 200 KB patch tier — 4× the previous max.

| config_id | outcome | patch files | patch KB | commits | merges | push |
|-----------|---------|-------------|----------|---------|--------|------|
| tiny-none-single-small-diverged-multi | pass (both) | 3 | 52.09 | 3 | 0 | +1 append → 4 |
| tiny-none-single-small-diverged-merge_msg | pass (both) | 1 | 50.62 | 1 | 0 | +1 append → 2 |
| tiny-none-single-medium-clean-single | pass | 1 | 205.8 | 1 | 0 | none |
| tiny-none-single-medium-clean-multi | pass | 3 | 202.0 | 3 | 0 | none |

New confirmations this run:
- **PATCH=medium (200 KB) is comfortably within limits** for create_pull_request
  at tiny/none. Measured patch sizes (202–206 KB) track the 200 KB target
  closely with non-compressible base64 content. No size-limit effects at 4× the
  prior tier — overhead stays proportionally constant, not growing with size.
- **format-patch overhead is fixed-per-commit and immaterial at 200 KB**:
  COMMIT=single added ~3.2 KB (~1.6%); COMMIT=multi (3 commits) added ~4.6 KB
  (~2.3%), i.e. ~1.5 KB per commit. The per-commit header cost is constant in
  absolute terms, so its *relative* weight shrinks as PATCH grows — the
  COMMIT=multi-near-threshold risk matters most at micro PATCH, least at large.
- **bundle < patch confirmed again**: diverged-multi bundle was 38.57 KB vs
  52.09 KB raw patch (pack vs base64-in-patch). The .patch sum stays the honest
  size metric for thresholds.
- **merge_msg filename leak re-confirmed at 50 KB / diverged**: single-parent
  "Merge branch x into feature" commit → empty `git rev-list --merges`, parent
  count 1, but format-patch names it `0001-Merge-branch-x-into-feature.patch`.
  Cosmetic; a downstream message-text merge heuristic would still misfire.
- **diverged append-only still a clean fast-forward at 50 KB**: `merge-base
  --is-ancestor` confirmed prior tip is ancestor of new tip on both diverged
  cells; no force-push, no `git merge` on feature.

### 2026-06-24 (run 28077852611) — cells 20-23

Tested indices 20-23: the last PATCH=medium / BRANCH=clean cell, then all three
PATCH=medium / BRANCH=ahead cells at SIZE=tiny / HISTORY=none. All **passed**.
First exercise of append-only push at the 200 KB patch tier.

| config_id | outcome | patch files | patch KB | commits (final) | merges | push |
|-----------|---------|-------------|----------|-----------------|--------|------|
| tiny-none-single-medium-clean-merge_msg | pass | 1 | 198.4 | 1 | 0 | none |
| tiny-none-single-medium-ahead-single | pass (both) | 1 | 198 | 2 | 0 | +1 append → 2 |
| tiny-none-single-medium-ahead-multi | pass (both) | 3 | 202.4 | 4 | 0 | +1 append → 4 |
| tiny-none-single-medium-ahead-merge_msg | pass (both) | 1 | 267.8 | 2 | 0 | +1 append → 2 |

New confirmations this run:
- **PATCH=medium (200 KB) append-only push is a clean fast-forward**:
  `merge-base --is-ancestor` confirmed prior tip is ancestor of new tip on all
  three ahead cells; no force-push, no `git merge` on feature. The 200 KB tier
  behaves identically to micro/small for the push path.
- **merge_msg filename leak re-confirmed at 200 KB (clean + ahead)**: single-
  parent "Merge branch topic into feature" → empty `git rev-list --merges`,
  parent count 1, but format-patch names it
  `0001-Merge-branch-topic-into-feature.patch`. Cosmetic; consistent across
  micro/small/medium and across clean/ahead/diverged. The clearest standing
  signal that a downstream *message-text* merge heuristic (vs `--no-merges` /
  parent-count) would misfire here.
- **COMMIT=multi at 200 KB stays proportional**: 3 patches (68941/69154/69156 B)
  = 207251 B (~202.4 KB); per-commit header overhead ~150-200 B per patch — i.e.
  the per-commit cost did not grow with PATCH size, confirming the 06-23
  "fixed-per-commit" finding at the ahead/push path too.
- **MEASUREMENT CAVEAT — base64 sizing inconsistency**: the ahead-merge_msg
  agent produced a 267.8 KB patch (vs ~198-202 KB for the other three) because
  it base64-encoded the full 200000 random *bytes* (which inflates ~4/3 → ~270
  KB on disk) instead of truncating the base64 stream to ~200000 bytes. Both are
  non-compressible and well under any cap so outcome is unaffected, but agents
  are NOT uniform about whether `patch_target_kb` means pre- or post-base64
  bytes. This matters near a real threshold: standardize on "truncate the base64
  output to TARGET_KB" so PATCH targets are honest. Watch at PATCH=large/xlarge.

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

Next enumeration index: **24** → `tiny-none-single-medium-diverged-single`.

Indices 24-26 finish PATCH=medium / BRANCH=diverged (24 = diverged-single,
25 = diverged-multi, 26 = diverged-merge_msg); index 27 enters PATCH=large
(1000 KB) at BRANCH=clean. The medium tier is now fully benign on size across
clean/ahead. The first genuinely interesting size jump is **PATCH=large (1000
KB)** starting at index 27, then **PATCH=xlarge (4000 KB)** at indices ~36-44 —
these remain the priority candidates for the first size-limit rejection. Also
prioritize standardizing the base64 truncation convention before reaching
large/xlarge so PATCH targets are measured consistently (see 06-24 caveat).
FILES=batch (100 files) and HISTORY=deep are still far ahead in the enumeration.
