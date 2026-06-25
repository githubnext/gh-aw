# Git Simulator Strategy Notes

Compact log of the 3600-cell Z3 sweep (SIZE×HISTORY×FILES×PATCH×BRANCH×COMMIT,
COMMIT innermost). Condensed 2026-06-25 to fit the 10 KB repo-memory budget.

## Results so far (28/3600, all PASS — no failures/rejections yet)

All cells tested live in the `tiny-none-single-*` corner (SIZE=tiny,
HISTORY=none, FILES=single). Enumeration has walked PATCH micro→small→medium and
just entered large; BRANCH clean/ahead/diverged and COMMIT single/multi/merge_msg
fully covered at each PATCH tier reached.

| PATCH tier | cells | branch×commit coverage | patch KB range | result |
|-----------|-------|------------------------|----------------|--------|
| micro (1 KB)   | idx 0-8   | clean/ahead/diverged × all commits | 1.2–4.6   | pass |
| small (50 KB)  | idx 9-17  | clean/ahead/diverged × all commits | 48.7–52.1 | pass |
| medium (200 KB)| idx 18-26 | clean/ahead/diverged × all commits | 198–267*  | pass |
| large (1000 KB)| idx 27    | clean/single only (so far)         | 1013.5    | pass |

*one 06-24 outlier at 267 KB from a base64-inflation bug, since fixed.

## Durable confirmations (hold across every tier tested)

- **No size effects through 1 MB.** create_pull_request patch generation and
  append-only push succeed cleanly at micro→large. format-patch overhead is a
  fixed ~1–2% of payload (header + base64 line-wrap), not super-linear: large
  (1 MB) added only ~1.4% over the 1000 KiB payload. No rejection seen yet.
- **Append-only push = clean fast-forward, always.** Every ahead/diverged cell:
  prior feature tip is ancestor of new tip (`merge-base --is-ancestor` OK); no
  force-push; no `git merge main` on feature; `rev-list --merges feature` empty;
  all parent counts = 1.
- **merge_msg is structurally a normal commit.** A single-parent commit with a
  "Merge branch 'x' into feature" *message* gives empty `rev-list --merges` and
  parent count 1, BUT format-patch names the artifact
  `0001-Merge-branch-x-into-feature.patch`. Cosmetic filename leak only — but the
  clearest standing signal a downstream *message-text* merge heuristic (vs
  `--no-merges`/parent-count) would misfire. Confirmed micro→medium, all branches.
- **diverged over-counts net file diff.** When BRANCH=diverged, `git diff
  main..feature` reports an extra file (history.md) purely from main's
  independent commit (symmetric two-dot artifact), exceeding the declared FILES
  dimension. The `merge-base..feature` diff correctly shows the declared payload.
  Also: append-only push makes actual_commit_count ≥ 2 even when COMMIT=single.
- **bundle < patch, gap widens with size.** `git bundle` (zlib pack) is smaller
  than the .patch (base64-of-binary inflation): ~25% smaller at medium/large.
  The .patch sum is the honest worst-case size metric for any downstream cap.

## Conventions / caveats

- **base64 sizing convention (standardized, fix for 06-24 bug):** truncate the
  base64 STREAM to TARGET*1024 bytes — do NOT base64-encode TARGET bytes (that
  inflates ~4/3). Use non-compressible /dev/urandom base64 so PATCH targets are
  honest near thresholds. Held cleanly on 06-25 (all four within ~2% of target).
- **Runtime note:** the `config-simulator` sub-agent type is NOT registered;
  use `general-purpose` with full self-contained prompts (works fine).

## Hypotheses to validate as coverage grows

- **xlarge (4000 KB) is the prime rejection candidate** — top size tier, first
  reached at idx 36 (clean), band idx 36-44. A 4 MB patch × COMMIT=multi overhead
  is the most likely place to finally hit a size cap.
- **FILES=batch (100 files)** and **HISTORY=deep (500)** remain far ahead in the
  enumeration; candidates for file-count / history-depth limits or timeouts,
  especially combined with xlarge.
- Baseline corner is fully healthy: no boundary effects below 1 MB at
  tiny/none/single.

## Next

Next enumeration index: **28** → `tiny-none-single-large-clean-multi`.
Idx 28-29 finish large/clean (multi, merge_msg); 30-32 large/ahead; 33-35
large/diverged; **idx 36 enters xlarge (4000 KB)/clean** — the strongest
remaining size-limit candidate. Watch COMMIT=multi at large (28) and the whole
xlarge band (36-44).
