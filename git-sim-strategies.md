# Git Simulator Strategy Notes

Z3 sweep of 3600 cells (SIZE×HISTORY×FILES×PATCH×BRANCH×COMMIT, COMMIT innermost).
**112/3600 tested, ALL PASS.** No fail/error/rejected ever seen. Enumeration:
`commit=i%3, branch=(i//3)%3, patch=(i//9)%5, files=(i//45)%4, history=(i//180)%4,
size=(i//720)%5`. sizes[tiny,small,medium,large,huge] hist[none,shallow,medium,deep]
files[single,few,many,batch]=[1,5,20,100] patch[micro,small,medium,large,xlarge]=
[1,50,200,1000,4000]KB branch[clean,ahead,diverged] commit[single,multi,merge_msg].

## Coverage map (all PASS)

- **tiny-none-single (idx 0-44): COMPLETE.** micro→xlarge × 3 branches × 3 commits.
- **tiny-none-few (idx 45-89): COMPLETE.** micro/small/medium/large/xlarge all tiers,
  all branches+commits. Representative KB: micro ~2-4, small ~52-56, medium ~204-207,
  large ~1001-1016, xlarge ~4053-4054. Bundle ~75% of patch throughout.
- **tiny-none-many (idx 90-91, OPENED): PASS.** micro-clean-single(90) 5.374 KB /1
  .patch (20 file-diffs in one commit's patch, ~224 B/file framing, header:payload
  ≈4.4:1 at micro); micro-clean-multi(91) 5.473 KB /3 .patch (framing×3, disjoint
  1-7/8-14/15-20). At micro payload the per-file+per-commit framing dominates ~5.5×.
  **idx92-95 (tiny-none-many-micro): PASS.** clean-merge_msg(92) 5.011 KB/1.patch,
  ahead-single(93) 4.98 KB, ahead-multi(94) 5.55 KB/3.patch, ahead-merge_msg(95)
  4.874 KB — all ~5 KB (framing-dominated, ~15 KB short of any concern). All ahead
  cells fast-forward (--is-ancestor=0, no force). merge_msg leak reconfirmed (92,95:
  0001-Merge-branch-topic-into-feature.patch, single-parent, --merges=0). Disjoint
  multi still ~1× payload. many-micro fully consistent w/ prior tiers.
  **idx96-99: PASS.** diverged-single(96) 5.496 KB/1.patch, diverged-multi(97)
  6.024 KB/3.patch (disjoint 7+7+6, ~1×), diverged-merge_msg(98) 5.481 KB/1.patch
  (0001-Merge-branch-topic-into-feature.patch leak reconfirmed, single-parent) — all
  three diverged: main's history.md commit excluded by two-dot, ff push exit 0, no
  merge into feature. **idx99 = FIRST tiny-none-many-small: PASS.** clean-single
  54.466 KB/1.patch for 50.000 KB payload → framing +4.47 KB (~9% at many/small,
  ~0.22 KB/file × 20). many-micro tier (90-98) DONE; small tier open at 99.
  **idx100-103: PASS (tiny-none-many-small continued).** clean-multi(100) 56.19 KB
  /3.patch (disjoint 7/7/6, ~1×); clean-merge_msg(101) 55.65 KB/1.patch (leak
  0001-Merge-branch-topic-into-feature.patch reconfirmed, --merges=0, parent=1);
  ahead-single(102) cpr 55.67→push 56.48 KB/2.patch (ff is-ancestor=0); ahead-multi
  (103) cpr 56.19→push 57.01 KB/4.patch (disjoint ~1×, ff is-ancestor=0). All 20
  files ≤200, ~56 KB ≪4096. many-small framing ~+6 KB (~0.3 KB/file, per-commit
  From-headers add ~0.3 KB/commit). tiny-none-many-small now covers idx99-103.
  **idx104-107: PASS (tiny-none-many-small tier COMPLETE).** ahead-merge_msg(104)
  59.94 KB/2.patch push (ff rc0, leak 0001-Merge-branch-topic-into-feature.patch,
  --merges=0); diverged-single(105) 56.63 KB two-dot excludes main history.md
  (three-dot +0.44 KB), ff rc0; diverged-multi(106) 55.90 KB disjoint ratio 1.118
  (~1×, three-dot +1 patch cosmetic), ff rc0; diverged-merge_msg(107) 56.06 KB leak
  reconfirmed single-parent, ff rc0. GOTCHA: `git diff --name-only main..feature`
  (endpoint) over-lists +1 (main's history.md phantom) under divergence; true set =
  per-commit `log --name-only` or three-dot `diff main...feature`. Cap = TWO-dot.
  **idx108-111: PASS (tiny-none-many-MEDIUM tier opens).** clean-single(108) 207.92
  KB/1.patch (framing ~405 B/file → payload:total 0.962, ~3.9% at medium/10KB-files);
  clean-multi(109) 208.43 KB/3.patch disjoint 7/7/6 ~1×; clean-merge_msg(110) 207.92
  KB/1.patch leak 0001-Merge-branch-topic-into-feature.patch reconfirmed (parent=1,
  --merges empty); ahead-single(111) cpr 207.9→push 218.6 KB/2.patch, ff is-ancestor
  =0 no force. many-medium framing ~+8 KB/20 files (~405 B/file: ~150-270 B header +
  ~135 B of `+`-line prefixes over ~135 base64 lines). All ≪4096. NB per-file framing
  at medium (~405 B) > earlier ~0.27 KB estimate — the `+`-prefix line tax scales with
  payload lines, so framing is ~3.9% of payload here, not a fixed per-file constant.
  **idx112-115: PASS (tiny-none-many-MEDIUM tier COMPLETE).** ahead-multi(112) 3.patch
  208.113 KB disjoint 6/6/8 ~1×, push ff exit0 push_merges=0 (4 commits); ahead-
  merge_msg(113) 1.patch 207.573 KB, leak 0001-Merge-branch-topic-into-feature.patch
  reconfirmed, --merges=0 parent=1, ff exit0; diverged-single(114) 207.568 KB two-dot
  (excl main history.md) vs three-dot 208.050 (+0.48 KB phantom), ff exit0; diverged-
  multi(115) 208.113 two-dot vs 208.591 three-dot (+0.48), disjoint ~1×, ff exit0.
  GOTCHA reconfirmed: `git diff --name-only main..feature` endpoint-lists 21 (main's
  history.md phantom) under divergence; format-patch two-dot = true 20-file cap set.
  many-medium (108-115) DONE. All ~208 KB ≪4096, 20≤100.

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

Next index: **116** → tiny-none-many-LARGE tier (idx117-125 patch=large 1000KB) opens
after idx116 (tiny-none-many-medium-diverged-merge_msg, LAST of medium tier), then
xlarge(126-134). many-medium COMPLETE 108-115. many-small COMPLETE 99-107. FILES=many runs
90-179, batch 180-269. **many-xlarge ~4058 KB still <4096 (safe).** batch-xlarge
~4080 KB <4096 BUT batch=100 files == max-patch-files default 100 (200 here) — watch
the `>` vs `>=` boundary. HISTORY=deep(500) & SIZE>tiny (idx 720+) far ahead — no
real `rejected` expected before then unless a PATCH tier is tuned >4096, batch runs
under a default-100 config, or the 3× same-file-append shape is exercised.
