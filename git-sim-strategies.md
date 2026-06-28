# Git Simulator Strategy Notes

Compact log of the 3600-cell Z3 sweep (SIZE×HISTORY×FILES×PATCH×BRANCH×COMMIT,
COMMIT innermost). Condensed 2026-06-25 to fit the 10 KB repo-memory budget.

## Results so far (40/3600, all PASS — no failures/rejections yet)

All cells in the `tiny-none-single-*` corner. PATCH walked micro→large to
COMPLETION; xlarge now PARTIAL (clean×{sgl,multi,mrg}+ahead-sgl, 06-28).

| PATCH tier | cells | branch×commit coverage | patch KB range | result |
|-----------|-------|------------------------|----------------|--------|
| micro (1 KB)   | idx 0-8   | full clean/ahead/diverged × all commits | 1.2–4.6   | pass |
| small (50 KB)  | idx 9-17  | full clean/ahead/diverged × all commits | 48.7–52.1 | pass |
| medium (200 KB)| idx 18-26 | full clean/ahead/diverged × all commits | 198–267*  | pass |
| large (1000 KB)| idx 27-35 | full clean/ahead/diverged × all commits | 1013–1116 | pass |
| xlarge (4000KB)| idx 36-39 | clean×{sgl,multi,mrg} + ahead-sgl ONLY   | 4052.9–4054.7 | pass |

*one 06-24 outlier at 267 KB from a base64-inflation bug, since fixed.
Large CLOSED 06-27. xlarge OPENED 06-28; remaining idx 40-44 (ahead-
multi/merge_msg + diverged×all).

## *** HEADLINE 06-28: xlarge rides ~99% of the REAL default cap ***

GROUNDED IN SOURCE (not a guess): the create-PR / push-to-branch patch cap is
**`max-patch-size` default = 4096 KB** (compiler_types.go:734 "in KB (defaults to
4096)"; safe_outputs_config.go:482 sets 4096 when unset; handler_registry.go:466,
528 emit `max_patch_size: 4096`). Units = KB, per-handler configurable, NOT ~1 MB.
A 06-28 sub-agent guessed ~1 MB and predicted "rejected" for idx 36 → RETRACTED;
corrected vs source → PASS (4052.9 < 4096).
- PATCH=xlarge (4000 KB payload) → **~4053–4055 KB** actual format-patch (~1.3–1.4%
  base64-in-patch overhead). UNDER 4096 but only **~41–43 KB (~1%) headroom** —
  the whole band rides the cap.
- PREDICTION: first real `rejected` appears at xlarge × (diverged OR multi-file OR
  bigger SIZE), where extra hunk/index/file headers tip net patch past 4096. Esp.
  BRANCH=diverged (symmetric-diff adds history.md hunk), FILES few/many/batch.
- Cap COMPARISON logic lives in the ACTION RUNTIME (TS collector), not this repo's
  Go/JS (pkg only sets the value). Unconfirmed whether it measures payload bytes
  or full format-patch bytes; if the latter (worst case), ~1% headroom is operative.

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

- **xlarge × extra-overhead = the rejection edge.** Now grounded: cap = 4096 KB,
  xlarge clean-single already at 4053 KB (~99%). The remaining xlarge cells
  (40-44) and any future xlarge with diverged/multi-file/bigger-SIZE are the
  prime spots to finally cross 4096 → real `rejected`. COMMIT does NOT inflate
  (1× for text), so COMMIT alone won't tip it; FILE-COUNT and DIVERGED will.
- **FILES=batch (100 files)** and **HISTORY=deep (500)** far ahead in enumeration;
  also note FILES>single adds per-file diff/index headers — at xlarge that header
  overhead is exactly what could breach 4096. Plus possible `max_patch_files` cap
  (handler emits `max_patch_files`, default seen ~800; confirm before batch).
- Baseline corner healthy: no boundary effects below 1 MB at tiny/none/single.

## Next

Next enumeration index: **40** → `tiny-none-single-xlarge-ahead-multi`.
xlarge clean-band (36-38) + ahead-single (39) CLOSED 06-28, all PASS at ~4053 KB
(~99% of the 4096 KB cap). Band idx 40-44 = ahead×{multi,merge_msg} +
diverged×{single,multi,merge_msg}. WATCH idx 42-44 (diverged): the symmetric-diff
history.md hunk adds bytes on top of an already-4053 KB patch — first plausible
breach of 4096 → real `rejected`. After xlarge fully closes (idx 45), enumeration
leaves PATCH and advances FILES (few/many/batch) then HISTORY (shallow→deep);
multi-file at xlarge (header overhead) and batch/deep are next limit candidates.
REMINDER: `config-simulator` subagent unregistered → use `general-purpose`.
