# Git Simulator Strategy Notes

Z3 sweep of 3600 cells (SIZE×HISTORY×FILES×PATCH×BRANCH×COMMIT, COMMIT innermost).
Condensed 06-30 to fit the 10 KB repo-memory budget. **76/3600 tested, all PASS.**

## Coverage map

- **tiny-none-single (idx 0-44): COMPLETE, all PASS.** PATCH micro→xlarge ×
  {clean,ahead,diverged} × {single,multi,merge_msg}. Patch KB by tier: micro
  1.2-4.6, small ~49-52, medium ~198-210, large ~1013-1116, xlarge ~4053-4060.
- **tiny-none-few-micro (idx 45-53): all PASS.** clean(45-47) 2.28-2.66 KB;
  ahead-single 2.22, ahead-multi 4.05, ahead-merge_msg 3.17, diverged-single 3.02,
  diverged-multi(52) 3.18, diverged-merge_msg(53) 2.42. few+push stays ~2-4 KB.
  merge_msg leak reconfirmed at few-diverged: artifact 0001-Merge-branch-topic-...
  .patch — TOPIC BRANCH NAME leaks into filename; parent count 1, --merges empty.
- **tiny-none-few-small-clean (idx 54-55): all PASS.** single(54) 52.29 KB (bundle
  38.69, -26%); multi(55) 3-commit SUM 52.7 KB ≈ 1.05× payload (net range 51.4 KB)
  — reconfirms COMMIT=multi is ~1× not N× for text. ~0.46 KB framing/file at 50 KB.
- **tiny-none-few-small-diverged (idx 60-62): all PASS.** single(60) 53.80 KB /2
  files /2 commits; multi(61) 4 files /4 commits 56.40 KB ≈1.13× (SHORT-append
  commit split → mild, consistent w/ 1× short-line law); merge_msg(62) 53.76 KB /2
  files, --merges empty + parent=1, filename leak 0001-Merge-branch-topic-...patch
  RECONFIRMED. All carry append-only push follow-up (commit_count≥2), main's
  diverged history.md commit correctly excluded via merge-base.
- **tiny-none-few-medium (idx 63-71): COMPLETE, all PASS.** 5×40960 B base64 (200 KB
  payload), ~5% of cap. clean-single(63) 204.20/1f; clean-multi(64) 204.67/3f 1.023×;
  clean-merge_msg(65) 206.80/1f leak; ahead-single(66) PR 206.78/1f+push 0.89 (FF,2);
  ahead-multi(67) 207.31/3f 1.036×+push 0.89 (FF,4); ahead-merge_msg(68) 204.16/1f
  leak (FF,2); diverged-single(69) 204.15/1f, two-dot delta=0 (diverged main..feature
  already =merge-base..feature; over-count only when feature strictly ahead), FF 2;
  diverged-multi(70) 205.32/3f **1.027×** (disjoint whole-file adds, NOT 2× — see law),
  FF 4; diverged-merge_msg(71) 204.16/1f leak, --merges empty, FF. bundle ~153 KB
  (~25%<patch). All KB. **few tier micro/small/medium DONE — all PASS.**
- **tiny-none-few-large (idx 72-75, partial): all PASS.** 5×204800 B base64 (~1000 KB).
  clean-single(72) 1001.35/1f, bundle 758 (-24%), overhead <0.14%; clean-multi(73)
  1015.13/3f **1.015×** (disjoint 2/2/1 whole-file adds → ~1× not 2×, reconfirms
  same-file-re-touch law), bundle 761; clean-merge_msg(74) 1016/1f leak
  0001-Merge-branch-topic-into-feature.patch, --merges empty parent=1, bundle 764;
  ahead-single(75) PR 1016/1f + push delta 8 KB (FF is-ancestor OK, commit_count=2),
  bundle 768. All ~25% of 4096 cap. large tier 76-80 (ahead-multi/merge_msg,
  diverged) remain.

## The cap (grounded in source)

`max-patch-size` default = **4096 KB** (units KB, per-handler configurable):
compiler_types.go:734, safe_outputs_config.go:482, handler_registry.go:466/528.
Compared (in the ACTION RUNTIME / TS collector, not this Go/JS repo) against the
`git format-patch` output (worst case). `max_patch_files` also handler-emitted,
default ~hundreds (~800?); confirm before FILES=batch(100).

## Durable size laws (hold across every tier tested)

- **patch ≈ payload + ~0.27 KB×files + ~Ncommits×(few-hundred-B From-header).**
  Per-file format-patch header ~270 B (measured idx 45). At micro payload the
  5-file header (~1.35 KB) dominates → total ~2.26× payload. format-patch overhead
  is a fixed ~1-2% of payload at large scale (large 1 MB: +1.4%), NOT super-linear.
- **COMMIT=multi multiplier is LINE-SHAPE dependent (REVISED idx58).** Old "~1×"
  law holds ONLY when appends are many SHORT lines. With FEW VERY LONG lines
  (single-line base64 chunk ~3.4K chars/file), unified-diff 3-line CONTEXT re-emits
  each prior long line every round → measured **~2.06×** at idx58 (PR 3-commit sum
  102.8 KB vs 50 KB payload; added-'+'-bytes still ~1×≈51 KB, the overshoot is all
  context). Toward ~N× as lines lengthen. Still nowhere near cap at small payload,
  but the diff-sizer must assume up-to-~N× for long-line content, not 1×.
  **SHARPENED idx70:** the ~2× only fires when successive commits APPEND to the SAME
  long-line file (re-emitting its context). DISJOINT whole-file base64 adds (each
  commit touches different new files) stay ~1× — idx70 measured 1.027× for 3 commits
  spanning files{1-2},{3-4},{5}. So multiplier = f(same-file re-touch depth), not
  f(commit count). Worst case (every commit appends to one growing long-line file).
- **DIVERGED adds ZERO feature-patch bytes.** main's independent commit (history.md)
  is merge-base-side, EXCLUDED from `merge-base..feature` (the patch the cap
  measures). Two-dot `main..feature` over-counts it (cosmetic only). Append-only
  push also makes actual_commit_count ≥ 2 even when COMMIT=single.
- **merge_msg is structurally a normal commit.** Single-parent, `rev-list --merges`
  empty, BUT format-patch names the artifact `0001-Merge-branch-...patch`. Cosmetic
  filename leak — the standing signal that a downstream *message-text* merge
  heuristic (vs --no-merges/parent-count) would misfire. Confirmed all branches.
- **Append-only push = clean fast-forward, always.** Every ahead/diverged cell:
  old tip is ancestor of new tip (`merge-base --is-ancestor` OK); no force-push; no
  `git merge main` on feature; `rev-list --merges feature` empty; parent counts = 1.
- **bundle < patch** (~25% smaller at medium/large; zlib vs base64). The .patch sum
  is the honest worst-case metric for any downstream cap.

## Rejection-edge analysis (NO real `rejected` seen yet)

xlarge clean-single sits at ~4053 KB = ~99% of 4096 (~43 KB headroom). Adding F
files costs only ~0.27 KB×(F-1): few→~4054, many→~4058, batch→~4080 — ALL under
4096. **FILE-COUNT headers can't breach the cap** (batch lands ~16 KB short).
COMMIT is 1× (text). DIVERGED is 0 bytes. So the first real `rejected` needs
**SIZE>tiny** (stuff.md payload entering the diff) or a PATCH target tuned over
4096 — NOT file-count, commits, or divergence. If a few-tier cell ever rejects,
it's a real bug (over-count, or runtime measuring excluded commits).

## Conventions / caveats

- **base64 sizing:** truncate the base64 STREAM to TARGET*1024 bytes total (do NOT
  base64-encode TARGET bytes — inflates 4/3). Use /dev/urandom (non-compressible)
  so PATCH targets are honest near thresholds.
- **`config-simulator` subagent is TRUNCATED/unregistered** → use `general-purpose`
  with self-contained prompts (works fine).
- **Repo-memory budget is tight (10 KB +20% = 12 KB hard).** 07-03: state.json
  now MINIFIED (no indent) → ~62 B/cell; sat at 12 KB after idx59. **07-04: DONE —
  switched `tested` to FAILURES-ONLY.** All 0-63 pass & contiguous, so `tested={}`
  (249 B); enumeration starts at next_index=64 and [0,64) w/ no entry = pass. Only
  add fail/error/rejected cells going forward. Call push_repo_memory after writing.

## Next

Next index: **76** → `tiny-none-few-large-ahead-multi` (76), then ahead-merge_msg(77),
diverged(78-80) finish the large tier (all ~1 MB → PASS predicted);
few-xlarge (81-89) predicted ~4054 KB → PASS (near 99% cap but header math shows
few lands ~4054, under 4096). That FINISHES FILES=few. FILES=many (idx 90-179) &
batch (180-269) next — many/batch × xlarge is the first place `max_patch_files`
(default ~800?) could bite; confirm that handler count before batch. HISTORY=deep
(500) far ahead. SIZE stays tiny (payload small) until idx 720, so no real `rejected`
expected before then unless a PATCH tier is tuned over 4096 KB or an over-count fires.
