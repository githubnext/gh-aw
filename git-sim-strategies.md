# Git Simulator Strategy Notes

Z3 sweep of 3600 cells (SIZE×HISTORY×FILES×PATCH×BRANCH×COMMIT, COMMIT innermost).
**136/3600 tested, ALL PASS.** No fail/error/rejected ever seen. Enumeration:
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

Next index: **148** = batch-SMALL ahead-multi. batch = idx135-179 (files=(i//45)%4==3);
batch-small/medium/large/xlarge = idx144-179 finish FILES=batch. HISTORY=deep(500) &
SIZE>tiny (idx720+) far ahead. **BATCH boundary confirmed (idx135):** 100 files ==
default max-patch-files 100 → workflow sets 200 so safe; measure file-count from PATCH
SET not tree diff.

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
