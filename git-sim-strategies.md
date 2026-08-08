# Git Simulator Strategy Notes

Z3 sweep of 3600 cells (SIZE×HISTORY×FILES×PATCH×BRANCH×COMMIT, COMMIT innermost).
**164/3600 tested, ALL PASS.** No fail/error/rejected ever seen. Enumeration:
`commit=i%3, branch=(i//3)%3, patch=(i//9)%5, files=(i//45)%4, history=(i//180)%4,
size=(i//720)%5`. sizes[tiny,small,medium,large,huge] hist[none,shallow,medium,deep]
files[single,few,many,batch]=[1,5,20,100] patch[micro,small,medium,large,xlarge]=
[1,50,200,1000,4000]KB branch[clean,ahead,diverged] commit[single,multi,merge_msg].

## Coverage map (all PASS)

- **tiny-none-single (idx 0-44): COMPLETE.** micro→xlarge × 3 branches × 3 commits.
- **tiny-none-few (idx 45-89): COMPLETE.** micro/small/medium/large/xlarge all tiers,
  all branches+commits. Representative KB: micro ~2-4, small ~52-56, medium ~204-207,
  large ~1001-1016, xlarge ~4053-4054. Bundle ~75% of patch throughout.
- **tiny-none-many-MICRO (idx 90-98): COMPLETE, all PASS.** ~5-6 KB/cell, framing-
  dominated (header:payload ≈4.4-5.5× at 1 KB payload; ~224 B/file). All branches+
  commits. merge_msg leak, disjoint multi ~1×, ff push rc0 all reconfirmed.
- **tiny-none-many-SMALL (idx 99-107): COMPLETE, all PASS.** ~54-60 KB/cell for 50 KB
  payload → framing +~6 KB (~0.3 KB/file, ~9%). ahead/diverged ff rc0, two-dot excludes
  main history.md (three-dot +0.44 KB cosmetic).
- **tiny-none-many-MEDIUM (idx 108-116): COMPLETE, all PASS.** ~207-208 KB/cell for
  200 KB payload → framing ~+8 KB (~405 B/file: header + `+`-prefix line tax scaling
  with payload lines → ~3.9% at 10 KB files). idx116 diverged-merge_msg two-dot 207.92
  vs three-dot 208.34 (+431 B phantom). All ff rc0, merge_msg leak reconfirmed.
- **tiny-none-many-LARGE (idx 117-123): PASS.** ahead(120-122)/diverged(123) ~1004-
  1018 KB two-dot, all clean ff (is-ancestor rc0), disjoint-multi(121) 1.019×, merge_msg
  leak(122) reconfirmed. idx123 CLARIFIES the phantom: `git diff --name-only main..feature`
  =21 (TWO-dot endpoint-tree compare pulls main's divergent history.md) vs three-dot=20
  (merge-base→feature, clean) — OPPOSITE polarity from format-patch, where two-dot is the
  clean 20-file/1018.42 KB cap set and three-dot adds the +0.47 KB phantom artifact. So:
  format-patch→use TWO-dot; git-diff --name-only→use THREE-dot for the honest feature set.
  clean-single(117) 1.patch
  1018.30 KB for 1000 KB payload → framing +18.30 KB (~0.915 KB/file, payload:total
  0.982, ~1.8%); bundle 762.24 KB = 25.2% smaller. clean-multi(118) 3.patch 1018.81 KB
  disjoint 7/7/6 → ~1.02× (per-commit From-header ~0.87-0.97 KB, ~2.8 KB/3 commits, NOT
  2×/3×). clean-merge_msg(119) 1.patch 1018.31 KB, leak reconfirmed, structural clean
  (parent=1, --merges=0, --no-merges counts=1 → message-substring 'Merge branch'
  heuristic would misfire). All ~1018 KB = ~24.9% of 4096 (3078 KB headroom), 20≤200.
  **KEY: framing % DROPS as file payload grows (micro ~5× → small ~9% → medium ~3.9%
  → large ~1.8%): fixed ~150-270 B header amortizes over larger base64 body.**
  GOTCHA reconfirmed (idx116): endpoint `git diff --name-only main..feature` lists 21
  (main history.md phantom under divergence) but format-patch two-dot = true 20 files.
- **tiny-none-many-large-DIVERGED (idx124-125): COMPLETE, all PASS.** diverged-multi(124)
  1018.92 KB, diverged-merge_msg(125) 1018.43 KB. Endpoint `diff --name-only` = 21 (main
  history.md phantom) but true feature = 20; ~3077 KB headroom. ff rc0, merges=0, merge_msg
  single-parent (PARENTS=2) + topic-name filename leak both reconfirmed. many-large DONE 117-125.
- **tiny-none-many-XLARGE (idx126-127): PASS, TIGHTEST CELLS YET.** clean-single(126)
  4057.41 KB, clean-multi(127) 4057.91 KB (3 disjoint patches, ~1×). **Only ~38 KB headroom
  to 4096** — framing +57.4 KB over 4000 payload (~1.4%). 20≤200 files. This is the closest
  any tested cell has come to max-patch-size; xlarge-clean confirmed SAFE but marginal. Bundle
  unmeasurable this run (git bundle produced no file for main..feature).

## THE CAPS (grounded)

- **max-patch-size default = 4096 KB** (KB units, per-handler configurable):
  compiler_types.go:734, safe_outputs_config.go:482, handler_registry.go:466/528.
  Measured vs `git format-patch` two-dot output in the ACTION RUNTIME (TS collector).
- **max-patch-files default = 100** (unique files in a create-pull-request patch;
  per-handler override). Source: .github/aw/safe-outputs-runtime.md:204 &
  safe-outputs-content.md:204; enforced in push_signed_commits.test.cjs:2143
  ("exceeds max-patch-files"). **THIS WORKFLOW SETS max-patch-files: 200**
  (daily-safeoutputs-git-simulator.md:46). CORRECTION: earlier notes guessed ~800 —
  it is 100 default. **FILES=batch = 100 files** sits EXACTLY at the default cap →
  under default it could reject if the check is `>=`; under THIS workflow's 200 it
  is safe. many(20) always safe. Confirm the `>` vs `>=` boundary when batch runs.

## Durable size laws (hold across every tier tested)

- **patch ≈ payload + ~0.27 KB×files + Ncommits×(few-hundred-B From-header).**
  Per-file format-patch header ~224-270 B. At MICRO payload the framing dominates
  (many-micro: ~224 B/file × 20 + per-commit ≈ 5.4-5.5 KB total for ~1 KB payload,
  ratio ~4.4-5.5×). At LARGE scale framing is a fixed ~1.3-1.4% of payload (xlarge
  4000 KB → +~54 KB, i.e. 4053-4054 KB). NOT super-linear.
- **COMMIT=multi multiplier = f(same-file re-touch depth), NOT commit count.**
  DISJOINT whole-file adds (each commit touches different new files) stay ~1×
  (measured 1.013-1.027× across medium/large/xlarge). The ~2× only fires when
  successive commits APPEND to the SAME long-line file (unified-diff 3-line context
  re-emits the prior long line). WORST CASE (idx82): single long UNWRAPPED line, 3
  commits each appending ~1/3 to a 4 MB file → format-patch **~3.0×** (each append
  rewrites the whole line: remove-all+add-all) → at 4000 KB payload this BREACHES
  4096 by ~3×. So diff-sizer must assume up-to-~3× for single-long-line same-file
  re-append; realistic few/many-disjoint is ~1.01×.
- **DIVERGED adds ZERO feature-patch bytes (TWO-dot).** main's independent commit
  (history.md) is merge-base-side, EXCLUDED from `main..feature` (the cap metric).
  Two-dot is the honest cap set. Three-dot `main...feature` = SYMMETRIC diff — ALSO
  emits main's divergent patch as an extra artifact (over-count +0.4-0.5 KB,
  cosmetic). Confirmed idx87 (+0.44), idx88 (+0.43), idx89 (+427 B). Append-only
  push makes actual_commit_count ≥ 2 even when COMMIT=single.
- **merge_msg is structurally a normal commit.** Single-parent, `rev-list --merges`
  empty, BUT format-patch names the artifact `0001-Merge-branch-topic-into-
  feature.patch` — the TOPIC BRANCH NAME leaks into the filename. Cosmetic filename
  leak; standing signal that a downstream *message-text* merge heuristic (vs
  --no-merges/parent-count) would misfire. Reconfirmed every branch incl idx89.
- **Append-only push = clean fast-forward, always.** Every ahead/diverged cell:
  old tip is-ancestor of new tip (exit 0); no force-push; no `git merge main` on
  feature; `rev-list --merges` of pushed range empty; parent counts = 1.
- **bundle < patch** (~25% smaller medium/large/xlarge; zlib vs base64). The .patch
  two-dot sum is the honest worst-case metric for any downstream cap.

## Rejection-edge analysis (NO real `rejected` seen yet)

xlarge clean-single ~4053 KB = ~99% of 4096 (~43 KB headroom). Adding F files costs
only ~0.27 KB×(F-1): few→~4054, many→~4058, batch→~4080 — ALL under 4096. **FILE-
COUNT headers can't breach max-patch-size** (batch lands ~16 KB short). But batch=100
files DOES hit max-patch-files=100 default (200 here). COMMIT-multi is 1× (disjoint).
DIVERGED is 0 bytes two-dot. So the first real max-patch-SIZE `rejected` needs
**SIZE>tiny** (stuff.md payload entering the diff, first at idx 720) OR the 3×
single-long-line same-file-append shape (that WOULD be a real over-cap finding). The
first max-patch-FILES `rejected` needs batch under a default-100 config.

## Conventions / caveats

- **base64 sizing:** truncate the base64 STREAM to TARGET*1024 bytes total (do NOT
  base64-encode TARGET bytes — inflates 4/3). Use /dev/urandom (non-compressible).
- **`config-simulator` subagent is TRUNCATED/unregistered** → use `general-purpose`
  with self-contained prompts (works fine; 4 parallel cells ~20-55s each).
- **Repo-memory budget tight (10 KB +20% = 12 KB hard).** state.json MINIFIED,
  `tested` is FAILURES-ONLY: [0,next_index) with no entry = pass, so tested={}
  (~250 B). Only add fail/error/rejected going forward. Call push_repo_memory after.

## Next

Next index: **152** = batch-MEDIUM (200 KB payload) clean-single. batch-small tier is
now DONE (144-151, 9/9 pass). batch-medium/large/xlarge = idx152-179 remain to finish
FILES=batch under tiny-none. HISTORY=deep(500) & SIZE>tiny (idx720+) far ahead.
**BATCH boundary confirmed (idx135):** 100 files == default max-patch-files 100 →
workflow sets 200 so safe; measure file-count from PATCH SET not tree diff. **Reminder
for idx152+:** use the base64-text truncation convention (not raw /dev/urandom binary)
for baseline comparability — see METHODOLOGY GOTCHA note above.

FILES=many COMPLETE (90-134), all PASS. **batch-micro COMPLETE (idx135-143), all PASS:**
100 files/cell, ~1 KB payload → framing DOMINATES ~256 B/file → ~22-26 KB patch (~24×;
size ~0.5% of 4096, only max-patch-files is live). clean-single(135) 24 KB, clean-multi
(136) 21.83 KB/3 disjoint ~1×, clean-merge_msg(137) 21.57 KB filename leak. ahead-single
(138) 24.40 KB push +0.50 KB, ahead-multi(139) 24.745 KB push +1.011 KB FF rc0. ahead-
merge_msg(140) 26.23 KB/2c FF rc0. diverged-single(141) two-dot 26.153 KB vs three-dot
+0.428 KB phantom(main history.md). diverged-multi(142) 24.0 KB/4 patches disjoint ~1.0×
phantom +143 B. diverged-merge_msg(143) 26.23 KB three-dot phantom +440 B. ALL: merges=0
/parent=1, leak `0001-Merge-branch-topic-into-feature.patch`, msg-substring heuristic
misfire, ff is-ancestor rc0. **Diverged GOTCHA:** `diff --name-only main..feature`=102
(100+followup+phantom history.md tree-revert) but format-patch commit-range=true set.
First real `rejected` still needs: PATCH tier >4096, batch under default-100, or 3×
single-long-line same-file-append. batch-xlarge (~4080 KB, 100f, idx~170s) — watch both.

**batch-SMALL clean tier COMPLETE (idx144-147), all PASS:** 100f, 50 KB payload → framing
+~22 KB (~220 B/file header) → patch ~72 KB (~1.44×, ~1.8% of 4096; 100≤200, only
max-patch-files live). clean-single(144) 71.97/1c, clean-multi(145) 72.51/3 disjoint ~1×,
clean-merge_msg(146) 71.96/1c filename leak+parent=1, ahead-single(147) 71.96 push delta
+1.01 KB FF is-ancestor rc0 (2c). batch-100 framing % tracks file-count: micro ~24× →
small ~1.44× → converges ~1.0× as payload grows (large/xlarge remain the cap watch).

**batch-SMALL tier now FULLY COMPLETE (idx144-151), all 9/9 PASS** (ahead×3, diverged×3,
clean×3 all covered). ahead-multi(148) 93.96 KB/101f/4c, ahead-merge_msg(149) 93.36 KB/101f/2c
(filename leak `0001-Merge-branch-topic-into-feature.patch` reconfirmed, parent=1,
`rev-list --merges` empty), diverged-single(150) two-dot 93.79 KB vs three-dot 94.28 KB
(+509 B phantom, format-patch polarity confirmed), diverged-multi(151) two-dot 93.23 KB
(4 patches) vs three-dot 93.73 KB/5 patches (+509 B, +1 phantom file) — disjoint-multi
still ~1× (no same-file re-touch). All: feature-tip-vs-feature-tip FF check (not
main-vs-feature) is the correct ancestor check for push_to_pull_request_branch under
divergence — confirmed passes even when `git merge-base --is-ancestor main feature` fails.

**⚠ METHODOLOGY GOTCHA — RAW BINARY vs BASE64-TEXT PAYLOAD CHANGES PATCH SIZE ~1.3×:**
idx148-151 sub-agents built file payloads from raw `/dev/urandom` bytes (genuinely
binary), NOT base64-text-encoded per the "Conventions" section below. Result: git
detects these as binary files and uses "GIT binary patch" base85 encoding + binary-diff
headers, inflating measured patch size to ~1.83-1.87× the raw payload (vs the ~1.44×
measured at idx144-147 using base64-text content for the SAME nominal 50 KB/100-file
cell). Root cause: base85 binary-patch encoding + zlib-then-encode framing costs more
per file than a plain text unified diff of equivalent byte count. **This is not a
regression** — both are valid, non-cheating fills — but it means prior "framing %"
laws (~1.3-1.4% at large/xlarge, "drops as payload grows") were derived under the
base64-text convention and may UNDERSTATE real-world cost for repos with genuinely
binary content (images, compiled artifacts). **Action for future runs:** keep using
the base64-text convention for apples-to-apples comparison across the sweep, but flag
that a true binary-payload variant of large/xlarge cells could plausibly breach caps
~1.3× sooner than the text-based baseline suggests — worth a dedicated one-off check
near idx126-127 territory (xlarge, only ~38 KB headroom under text convention; binary
content there could realistically breach 4096 KB).

## Run 2026-07-30: idx152-155 (batch-small tier closed + batch-medium clean tier)

- **idx152 batch-small-diverged-merge_msg: PASS.** 76.74 KB/100f/2c (two-dot authoritative).
  Three-dot phantom +447B/+1 file reconfirmed. **batch-small tier now TRULY 9/9 complete**
  (prior run's "9/9" claim was off-by-one; this was the missing cell). ff-on-feature-tip
  holds under main/feature divergence; merge_msg filename leak + single-parent/no-merges
  reconfirmed again.
- **idx153/154/155 batch-medium clean (single/multi/merge_msg): ALL PASS.** 200 KB payload,
  100 files → 221.95 / 230.72 / 230.17 KB (~11-15% framing, ~223 B/file header cost).
  Confirms law: framing % keeps dropping as per-file payload grows (batch-small ~44% →
  batch-medium ~11-15% → large/xlarge tiers ~1-2%). disjoint-multi(154) vs single-baseline
  in-same-run ratio ~1.002x (per-commit overhead only ~285 B, lower than earlier ~0.87-0.97 KB
  estimate — likely author/date line length variance, not a contradiction). merge_msg(155)
  filename leak + single-parent reconfirmed at this payload size too. All far under caps
  (~5.6% of 4096 KB, 100/200 files).
## Run 2026-07-31: idx156-159 (batch-medium ahead tier + diverged-single)

- **⚠ CAP CORRECTION:** this workflow's YAML (daily-safeoutputs-git-simulator.md:45,57)
  sets `max-patch-size: 5120` KB for BOTH create-pull-request and push-to-pull-request-branch
  (NOT the framework default 4096 cited earlier — that default was never actually the live
  cap here). All prior "% of 4096" headroom calcs in this file understate real headroom by
  ~25%; re-baseline future large/xlarge-tier analysis against 5120. max-patch-files=200
  confirmed unchanged. Also confirmed `allowed-files: [sim/**, stuff.md, history.md]` on both
  handlers — from idx156 onward, patch payload files are placed under `sim/batch/...` (not
  bare `batch/...`) for allow-list fidelity; negligible effect on measured sizes (path-string
  length only).
- **idx156 ahead-single: PASS.** 101 files/226-228KB (2 format-patch commits, two-dot==three-dot
  since no main divergence). ff rc0, merges=0/parent=1, sim/** compliant.
- **idx157 ahead-multi: PASS.** 3 disjoint batch commits (34/33/33) + 1 followup = 4 commits,
  101 files, 226.74 KB — confirms disjoint-multi stays ~1x (no multiplication) even at
  FILES=batch scale. ff rc0, merges=0, sim/** compliant.
- **idx158 ahead-merge_msg: PASS.** Filename leak reconfirmed at batch scale:
  `0001-Merge-branch-topic-into-feature.patch`. merges=0/parent=1 (structurally normal despite
  message). 101 files/226.19 KB, ff rc0.
- **idx159 diverged-single: PASS.** New precise numbers: two-dot format-patch=2 files/226.19KB
  (feature-only, authoritative for caps); three-dot=3 files/226.71KB (+0.51KB main phantom
  commit leaked in). **Confirms format-patch vs `diff --name-only` phantom INVERSION exactly**:
  `diff --name-only main..feature`=102 (contaminated by main's divergent history.md) vs
  `main...feature`=101 (clean) — opposite polarity from format-patch where two-dot is clean.
  Correct ff check for push_to_pull_request_branch under divergence = feature-tip-vs-feature-tip
  (rc0), NOT main-vs-feature (rc1, correctly fails — divergence is real).
## Run 2026-08-01: idx160-163 (batch-medium tier CLOSED + batch-large clean tier opened)

- **idx160 diverged-multi: PASS.** 101f/234.13KB/4c (two-dot log-range, correct). `git diff
  --stat main..feature` (TREE diff, not log range) showed 102f w/ 2 phantom deletions from
  main's divergent commit — confirms the two-dot GOTCHA generalizes beyond `--name-only` to
  plain `diff --stat` too: only `format-patch`/`log` two-dot (commit-range semantics) is
  phantom-free; any tree-comparison form (`diff`, `diff --stat`, `diff --name-only`) leaks
  main's divergent commit content when using two dots. Three-dot format-patch added a 5th
  phantom patch file (240.3KB) for main's commit — reconfirms two-dot is correct for both
  create-pull-request and push-to-pull-request-branch patch generation.
- **idx161 diverged-merge_msg: PASS.** 100f/224.94KB/2c. Same tree-diff-vs-log-range GOTCHA
  reconfirmed independently. Literal 3-dot format-patch leaked main's commit as an extra
  patch file named for its own subject (not just phantom bytes) — a real file-level leak, not
  just a byte-count artifact. merge_msg filename leak (`0001-Merge-branch-topic-into-...`)
  + single-parent/no-merges structural check reconfirmed at this tier too.
- **batch-medium tier now FULLY CLOSED (idx153-161, 9/9 pass).**
- **idx162 batch-large-clean-single: PASS.** 100f/1043.77KB (1000KB payload, 1c). Bundle
  766.97KB (~27% smaller, consistent w/ zlib-vs-base64 law). Headroom to REAL cap (5120KB,
  not the old assumed 4096) = 4076KB (~20.4% utilized) — much safer margin than earlier notes
  implied before the 5120 correction (idx156) was made.
- **idx163 batch-large-clean-multi: PASS.** 100f/1054.67KB/3c disjoint. Single-commit baseline
  built for direct A/B: 1079391B vs multi 1079982B = 1.0005x — tightest confirmation yet that
  disjoint-multi commits add negligible (~590B total) overhead vs same-file re-append's ~3x.
- **Zero real fail/error/rejected across 164 cells.** Rejection candidate still SIZE>tiny
  (idx720+, first real stuff.md/history.md payload in the diff) or same-file re-append shape.

## Run 2026-08-02: idx164-167 (batch-large clean tier CLOSED + batch-large ahead tier CLOSED)

- **idx164 clean-merge_msg: PASS.** 100f/1043.21KB/1c. Filename leak reconfirmed
  (`0001-Merge-branch-topic-xyz-into-feature.patch`), parent=1/merges=empty. **batch-large
  clean tier now fully closed (162-164, 3/3 pass).**
- **idx165 ahead-single: PASS.** 101f/1045.71KB/2c. Isolated follow-up-push delta measured
  separately: 1 patch/2.54KB — incremental push cost stays tiny regardless of the large
  initial payload already on the branch. ff rc0, merges=0/parent=1.
- **idx166 ahead-multi: PASS.** 4 commits (3 disjoint initial + 1 followup)/1046.33KB. Same-run
  single-commit baseline 1043.17KB → disjoint-multi ratio 1.003x, reconfirming disjoint ~1x
  law at ahead+batch+large scale. ff rc0, merges=0.
- **idx167 ahead-merge_msg: PASS.** 101f/1045.78KB/2c. Filename leak + ff rc0 + parent=1
  reconfirmed together (merge-worded initial commit, then a normal followup push).
- **batch-large ahead tier now fully closed (165-167, 3/3 pass).** batch-large tier overall:
  clean+ahead closed (6/9); diverged sub-tier (3 cells) remains.
- **Zero real fail/error/rejected across 168 cells.**
- **Next index: 168** = tiny-none-batch-large-diverged-single (closes batch-large tier fully),
  then batch-xlarge (idx171+) under the corrected 5120KB cap.

## Run 2026-08-03: idx168-171 (batch-large-diverged tier CLOSED + batch-xlarge-clean opened)

- **idx168 diverged-single: PASS.** 1030.10KB/1c/1f(patch). Two-dot format-patch clean
  (1030.10KB); three-dot leaks main's commit as phantom 2nd patch (+529B). Tree `diff
  --name-only main..feature`=101 (phantom history.md) vs true feature=100 — same
  inversion law reconfirmed. ff-fail main-vs-feature rc1 correct (real divergence).
- **idx169 diverged-multi: PASS.** 101f/1055.14KB/4c (3 disjoint batches 34/33/33 +1
  followup). Same-run single-commit baseline 1054.29KB → multi ratio 1.0008x, tightest
  disjoint-multi confirmation yet. ff rc0 old→new feature tip.
- **idx170 diverged-merge_msg: PASS.** 101f/1026.69KB/2c. Filename leak reconfirmed
  (`0001-Merge-branch-topic-into-feature.patch`), parent=1/merges=empty structurally
  clean despite message text.
- **batch-large tier now FULLY CLOSED (idx162-170, 9/9 pass).**
- **idx171 batch-xlarge-clean-single: PASS — first xlarge cell under corrected 5120KB
  cap.** 100f/4026.18KB/1c for 4000KB payload → only ~0.66% framing overhead (lower
  than prior ~1.3-1.4% law; large per-file payload amortizes header cost well).
  Headroom to cap = 1093.82KB (~21.4% free) — comfortable, not marginal like the old
  4096KB-cap analysis assumed. Bundle 3033.91KB (24.65% smaller, zlib-vs-base64 law
  holds). batch-xlarge tier: clean-single done, ahead/merge_msg/diverged remain (172-179).
- **Zero real fail/error/rejected across 172 cells.**

## Run 2026-08-04: idx172-175 (batch-xlarge clean+ahead tiers CLOSED)

- **idx172 clean-multi: PASS.** 3 disjoint commits (34/33/33f)/4083.07KB/100f, headroom
  ~1037KB. Pairwise `comm -12` confirmed zero filename overlap across commits.
- **idx173 clean-merge_msg: PASS.** 1c/100f/4082.52KB. Filename leak reconfirmed
  (`0001-Merge-branch-topic-xyz-into-feature.patch`), parent=1/merges=empty.
- **idx174 ahead-single: PASS.** Initial 1c/100f/4082.54KB + followup push 1c/1f/3.64KB
  = 101f/4086.18KB/2c total. ff rc0 (OLD_TIP→feature ancestor).
- **idx175 ahead-multi: PASS.** Initial 3 disjoint commits/100f/4037.24KB + followup
  push 1c/1f/4.59KB = 101f/4041.83KB/4c. Same-run single-baseline A/B ratio 1.0001x
  (4134133/4133528B) — tightest disjoint-multi≈1x confirmation yet, even at xlarge+batch.
- **batch-xlarge tier: clean+ahead CLOSED (9/9... actually 6/9; diverged×3 remain).**
  All 4 well under 5120KB cap (~20-21% utilized) and 200-file cap (100-101 touched).
  No new laws contradicted; disjoint-multi≈1x and merge_msg filename-leak hold at
  the largest payload/file-count tier tested yet.
- **Zero real fail/error/rejected across 176 cells.**
- **Next index: 176** = tiny-none-batch-xlarge-diverged-single (closes batch-xlarge
  tier fully; last cell before HISTORY moves off "none", idx180+).

## Run 2026-08-05: idx176-179 (batch-xlarge tier CLOSED; entire tiny-none block CLOSED)

- **idx176 ahead-merge_msg: PASS.** 101f/4027.25KB/2c. merge_msg filename leak
  reconfirmed (`0001-Merge-branch-topic-xyz-into-feature.patch`), parent=1/merges=empty,
  ff rc0. Closes batch-xlarge ahead tier (174-176, 3/3).
- **idx177 diverged-single: PASS.** two-dot 2f/4084.08KB vs three-dot 3f/4084.52KB
  (+0.44KB main phantom). ff(old→new feature tip) rc0; ff(main→feature) rc1 (real
  divergence) — both checks reconfirmed at xlarge scale.
- **idx178 diverged-multi: PASS.** two-dot 4f/4038.89KB (3 disjoint + followup) vs
  three-dot 5f/4039.37KB. Disjoint-multi ratio vs single-baseline = 1.0001x — tightest
  confirmation yet that disjoint commits add ~0 overhead even at xlarge+batch+diverged.
- **idx179 diverged-merge_msg: PASS.** two-dot 2f/4083.08KB vs three-dot 3f/4083.53KB.
  Filename leak + parent=1/merges=empty + ff-rc0/rc1 pair all reconfirmed together.
- **batch-xlarge tier now FULLY CLOSED (171-179, 9/9 pass).**
- **ENTIRE tiny-none BLOCK NOW CLOSED (idx0-179, 180/180, zero fail/error/rejected).**
  All FILES×PATCH×BRANCH×COMMIT combos pass under SIZE=tiny/HISTORY=none. Largest
  cells (~4027-4084 KB) sit at ~79-80% of the 5120KB cap — consistent margin, no
  cliff found yet.
- **Next index: 180** = tiny-shallow-single-micro-clean-single — first cell with
  HISTORY=shallow (5 entries in history.md). This begins probing whether history.md
  entry count (vs raw patch bytes) has any independent cost effect, since history.md
  content itself is tiny/negligible-sized regardless of entry count in most designs.

## Run 2026-08-06: idx180-183 (HISTORY moves off "none" — tiny-shallow-single-micro tier opened)

- **idx180 clean-single: PASS.** 1f/3KB (1KB payload target, 5 history.md entries,
  0 stuff.md entries). Framing dominates at this scale as expected (micro-tier law
  holds). 1c/1 parent/0 merges.
- **idx181 clean-multi: PASS.** 1f/3KB/3c (2 empty follow-up commits per multi
  convention at file_count=1 — no additional file content, so disjoint-multi ratio
  law doesn't apply distinctly here; only the single real file diff counts).
- **idx182 clean-merge_msg: PASS.** 1f/3KB/1c. Filename/message convention holds
  (single-parent, 0 merges, message text only).
- **idx183 ahead-single: PASS.** Initial 1f/3KB + followup push 1f/~0KB (tiny extra
  file) = 2f/3KB/2c total. ff rc0 (old tip → new tip ancestor), confirms append-only
  clean fast-forward law still holds at HISTORY=shallow.
- **HISTORY=shallow (5 entries) has ZERO measurable cost impact** vs HISTORY=none at
  same SIZE/FILES/PATCH tier — history.md content itself stays tiny regardless of
  entry count in this range; patch sizes (~3KB) match the tiny-none-single-micro
  baseline (idx0) almost exactly. Confirms history.md entry count is NOT itself a
  cost driver (only stuff.md/patch payload sizes drive measured KB).
- **Zero real fail/error/rejected across 184 cells.**
- **Next index: 184** = tiny-shallow-single-micro-diverged-single (closes the
  shallow-single-micro tier; ahead/diverged/clean × single/multi/merge_msg = 9 cells,
  184-188 remain... actually clean×3 + ahead-single done = 4/9, remaining 5).

## Run 2026-08-07: idx184-187 (shallow-single-micro tier nearly closed: ahead+diverged)

- **idx184 ahead-multi: PASS.** 4c/3.28KB, 2 unique files (payload.txt+history.md) but
  4 diff-git entries (history.md touched in 3 separate commits) — file_count metric
  should use unique-files not diff-git-line-count. ff rc0, merges=0/parent=1.
- **idx185 ahead-merge_msg: PASS.** 2c/2.09KB. Filename leak reconfirmed at shallow-
  history tier: `0001-Merge-branch-topic-xyz-into-feature.patch`, merges=0/parent=1.
- **idx186 diverged-single: PASS.** two-dot 2f/2.54KB vs three-dot 3f/3150B (+545B/+1f
  phantom, format-patch polarity confirmed again). NEW: tree-diff shows OPPOSITE
  polarity — `diff --name-only main..feature` (two-dot) itself leaks history.md from
  main's divergence while two-dot format-patch stays clean; three-dot tree-diff
  (merge-base-relative) is the clean one. ff(OLD_TIP→NEW_TIP) rc0; ff(main↔feature)
  BOTH directions rc1 (genuine bidirectional divergence, neither is ancestor of other).
- **idx187 diverged-multi: PASS.** two-dot 4f/3.15KB vs three-dot 5f/3751B (+525B
  phantom). Disjoint-multi ratio: single-baseline 1618B vs 3-commit initial set 2655B
  = ~1.64x (small history.md-append commits carry ~500B/commit header overhead at
  THIS micro scale — not the ~1.0x seen at larger payload tiers; law refinement:
  disjoint-multi ratio approaches 1.0x only once real payload dominates fixed
  per-commit header cost, at micro scale fixed overhead is comparatively material).
  ff rc0, merges=0/parent=1.
- **HISTORY=shallow ahead/diverged sub-cases confirm all core laws hold** (ff checks,
  merge_msg filename leak, two-dot/three-dot polarity for both format-patch and
  tree-diff) at HISTORY=5-entries same as HISTORY=none.
- **Zero real fail/error/rejected across 188 cells.**
- **Next index: 188** = tiny-shallow-single-micro-diverged-merge_msg (closes the
  shallow-single-micro tier fully, 9/9). Then HISTORY=shallow moves to FILES=few
  (idx189+).

## Run 2026-08-08: idx188-191 (shallow-single-micro CLOSED; shallow-single-small opened)

- **idx188 diverged-merge_msg: PASS.** Closes shallow-single-micro tier 9/9. two-dot
  1f/1.566KB vs three-dot 2f/2.21KB (+0.644KB phantom). ff(old→new tip) rc0; both-
  direction main↔feature rc1 (genuine divergence). parent=1/merges=empty, filename
  leak `0001-Merge-branch-topic-xyz-into-feature.patch` reconfirmed.
- **⚠ ENUMERATION CORRECTION:** prior note assumed idx189+ moves to FILES=few — WRONG.
  `files=(i//45)%4` only advances every 45 indices; FILES=single spans the FULL
  180-224 range within each history block (180=start of shallow block, +44=224).
  idx189-191 are still FILES=single, with PATCH advancing micro(180-188)→small
  (189-197). Confirmed by direct recompute: idx189 commit=0,branch=0,patch=1,files=0,
  history=1,size=0 = clean-single, small, single, shallow, tiny.
- **idx189 clean-single (small,single): PASS.** 1f/51.22KB (50KB payload), bundle
  38.64KB (~24.6% smaller). parent=1/merges=empty.
- **idx190 clean-multi (small,single, same-file 3-commit append): PASS.** Newline-
  wrapped (fold -w100) chunks → ratio 1.028x vs single-commit baseline (53.69 vs
  52.23 KB) — CONFIRMS append-shape law: wrapped/multi-line append stays ~1x; only
  a genuinely single unwrapped long line reflows to ~3x. Shape matters more than
  raw size for the multi-commit same-file multiplier risk.
- **idx191 clean-merge_msg (small,single): PASS.** 1f/50.05KB, filename leak
  `0001-Merge-branch-topic-xyz-into-feature.patch` reconfirmed, parent=1/merges=empty.
- **Zero real fail/error/rejected across 192 cells.**
- **Next index: 192** = tiny-shallow-single-small-ahead-single (continues
  shallow-single-small tier: ahead×3 + diverged×3 remain, idx192-197).
