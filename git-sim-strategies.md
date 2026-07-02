# Git Simulator Strategy Notes

Z3 sweep of 3600 cells (SIZE×HISTORY×FILES×PATCH×BRANCH×COMMIT, COMMIT innermost).
Condensed 06-30 to fit the 10 KB repo-memory budget. **56/3600 tested, all PASS.**

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
- **No payload multiplier for COMMIT=multi (text).** N commits each appending text
  to a file → patch SUM ≈ net diff (~1×), because git emits only newly-appended
  lines per commit. (3× re-emission happens only for true binary blobs / line
  rewrites.) So COMMIT alone never tips the cap.
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
- **Repo-memory budget is tight (10 KB +20% = 12 KB hard).** state.json grows ~95 B
  per tested cell; keep strategies.md lean. Call push_repo_memory to validate.

## Next

Next index: **56** → `tiny-none-few-small-clean-merge_msg` (few-small spans idx
54-62). few-xlarge (~idx 84-89) predicted ~4054 KB → PASS. HISTORY=deep (500) and
FILES=many/batch far ahead — prime untested regions for a first rejection. SIZE
stays tiny (payload small) until idx 720, so no real `rejected` expected before
then unless a PATCH tier is tuned over 4096 KB or a downstream over-count bug fires.
