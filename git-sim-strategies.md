# Git Simulator Strategy Notes

Compact log of the 3600-cell Z3 sweep (SIZE×HISTORY×FILES×PATCH×BRANCH×COMMIT,
COMMIT innermost). Condensed 2026-06-25 to fit the 10 KB repo-memory budget.

## Results so far (36/3600, all PASS — no failures/rejections yet)

All cells tested live in the `tiny-none-single-*` corner (SIZE=tiny,
HISTORY=none, FILES=single). Enumeration walked PATCH micro→small→medium→large to
COMPLETION; BRANCH clean/ahead/diverged × COMMIT single/multi/merge_msg fully
covered at every tier through large. xlarge is next (idx 36).

| PATCH tier | cells | branch×commit coverage | patch KB range | result |
|-----------|-------|------------------------|----------------|--------|
| micro (1 KB)   | idx 0-8   | full clean/ahead/diverged × all commits | 1.2–4.6   | pass |
| small (50 KB)  | idx 9-17  | full clean/ahead/diverged × all commits | 48.7–52.1 | pass |
| medium (200 KB)| idx 18-26 | full clean/ahead/diverged × all commits | 198–267*  | pass |
| large (1000 KB)| idx 27-35 | full clean/ahead/diverged × all commits | 1013–1116 | pass |

*one 06-24 outlier at 267 KB from a base64-inflation bug, since fixed.
Large tier CLOSED 06-27 (idx 32-35: ahead/merge_msg + diverged×{single,multi,merge_msg}).

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
- **COMMIT=multi patch-SUM inflation is BINARY-only, NOT text-append.**
  CORRECTED 06-27 (idx 34, large-diverged-multi): 3 commits each APPENDING ~333 KB
  of base64 TEXT to the SAME file → patch SUM ≈ 1014 KB ≈ net diff (~1:1, NOT 3×).
  Git's line-based diff emits only the newly appended lines per commit, not the
  cumulative growing file, so multi-commit-to-one-file is CHEAP for text. The ~3×
  the 06-26 note attributed to "append-to-same-file" was WRONG for text; that 3×
  re-emission happens only when git emits a full GIT-binary-patch per revision
  (true binary blob, or prepend/rewrite of existing lines forcing whole-hunk
  re-emit). For our base64-text payloads, patch-SUM tracks net payload at every
  commit count. So a downstream patch-SUM cap trips at ~the same threshold whether
  1 squashed commit or N appends — no hidden multiplier for text. (The 06-26
  idx-28 ~3001 KB figure was a binary-blob / non-append measurement artifact.)
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

Next enumeration index: **36** → `tiny-none-single-xlarge-clean-single`.
Large tier is closed; **idx 36 enters xlarge (4000 KB)/clean** — the strongest
remaining size-limit candidate. xlarge band is idx 36-44 (clean/ahead/diverged ×
single/multi/merge_msg). NOTE the 06-27 correction: for our base64-TEXT payloads,
COMMIT=multi does NOT inflate the patch SUM (~1× not 3×), so the earlier "multi
pushes xlarge SUM toward ~12 MB" hypothesis is RETRACTED — a 4 MB xlarge stays
~4 MB regardless of commit count. The honest worst-case at xlarge is ~4 MB patch
SUM. If a size cap exists it should surface as a flat ~4 MB threshold at idx 36+,
independent of COMMIT. After xlarge, enumeration leaves PATCH and advances FILES
(few/many/batch) then HISTORY (shallow→deep) — batch (100 files) + deep (500) are
the next untested file-count / history-depth limit candidates.
