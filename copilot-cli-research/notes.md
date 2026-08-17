# Copilot CLI Research Notes

## Run 30785495408 (2026-08-03)
- Orphaned custom agent files unchanged since prior run: create-safe-output-type, custom-engine-implementation, grumpy-reviewer, interactive-agent-designer, w3c-specification-writer (0 workflows reference them).
- `--share` flag: still only used in this research workflow itself.
- `max-continuations` (autopilot): only 11/98 copilot-engine-block workflows use it.
- copilot-sdk: true adoption continues to grow (68 workflows).

## Run 30878540517 (2026-08-04)
- max-tool-denials: RESOLVED — jumped from 0 to 66 workflows using it (was 17+ run persistent gap).
- --share flag: still only 1 workflow (this research workflow itself) — persistent gap.
- engine.args / engine.env: still 0 usage across all copilot workflows — no custom CLI args or env overrides anywhere in repo.
- engine.model override for copilot: still 0.
- cache-memory: grew to 75 (from 94 reported previously — recheck methodology, likely counting differs).
- 5 orphaned custom agent files unchanged again this run (0 references): create-safe-output-type, custom-engine-implementation, grumpy-reviewer, interactive-agent-designer, w3c-specification-writer.
- Total workflows: 270 (up from 269).

## 2026-08-05 (run 30975991815)
- Re-ran full inventory; workflow count grew 270->275, copilot-engine workflows flat at 38.
- Custom agents: 9 defined in .github/agents/, only 4 referenced (3 orphaned confirmed unchanged: create-safe-output-type, interactive-agent-designer, grumpy-reviewer, custom-engine-implementation, w3c-specification-writer - 5 orphaned).
- No change in --share adoption (still 1), engine.args/model overrides still 0.
- cache-memory adoption rose 75->98 workflows - positive trend, worth highlighting as a success story vs recommending further adoption.

## Run 31072088452 (2026-08-06)
- Total workflows 275 (flat), copilot-engine workflows flat at 38.
- engine.args usage rose 0->3 — small positive movement, first non-zero reading in tracked history.
- --share flag: still stuck at 1 (this research workflow only) — persistent gap across 5+ runs.
- 5 orphaned custom agents unchanged again (create-safe-output-type, interactive-agent-designer, grumpy-reviewer, custom-engine-implementation, w3c-specification-writer).
- cache-memory adoption flat at 98 (no growth since last run) — plateaued after earlier rise from 75.
- max-tool-denials flat at 66, max-continuations flat at 11, network config flat at 152.

## Run 31147540181 (2026-08-07)
- Total workflows grew 275->276; copilot-engine detection refined (132 files match "engine: copilot$" or "id: copilot$" combined, up from prior narrower 38-count metric which undercounted shorthand `engine: copilot` usage — methodology note, not real growth).
- --share flag: still stuck at 1 (this research workflow only) — persistent gap across 6+ runs now.
- engine.args usage: reverted to 0 (previous run showed 3; recheck shows those were false-positive proximity matches, not true engine.args blocks) — treat prior "3" reading as measurement noise, true value is 0 across all runs.
- engine.agent flag: newly measured this run — 11 workflows use engine.agent (daily-credit-limit-test, smoke-service-ports, agent-performance-analyzer, archie, contribution-check, daily-assign-issue-to-user, daily-file-diet, daily-max-ai-credits-test, glossary-maintainer, technical-doc-writer, workflow-generator).
- Custom agents: 11 defined (up from 9), 6 referenced (adr-writer, ci-cleaner, contribution-checker, technical-doc-writer x2, agentic-workflows x2, interactive-agent-designer referenced via .agent.md? recheck) - same 5 orphaned agents persist unchanged for 6+ runs: create-safe-output-type, interactive-agent-designer, grumpy-reviewer, custom-engine-implementation, w3c-specification-writer.
- cache-memory adoption flat at 99 (plateaued, same as last run).
- max-tool-denials flat at 67, max-continuations flat at 11, network config flat at 152.
- No new Copilot CLI features detected in codebase since last run.

## Run 31238532089 (2026-08-08)
- Total workflows 283 (up from 276); copilot-engine-block workflows: 185 broad match ("engine: copilot" or "id: copilot"), 97 with explicit `id: copilot:` extended form.
- --share flag: still stuck at 1 (this research workflow only) — persistent gap across 7+ runs now. RECOMMEND: stop re-flagging every run, treat as accepted low-priority/non-issue since no workflow currently needs conversation-sharing.
- engine.args: confirmed still 0 across all copilot workflows (re-verified with direct grep under engine: blocks) — persistent gap, 3+ runs confirmed as real zero.
- add-dir custom usage: 0 real workflows use it (only doc references in this research workflow itself).
- disable-builtin-mcps: 1 real usage (auto-triage-issues.md) besides this research workflow's doc text.
- Custom agents: 11 defined, orphaned set UNCHANGED for 7+ runs: create-safe-output-type, custom-engine-implementation, grumpy-reviewer, w3c-specification-writer (4 confirmed zero-reference). interactive-agent-designer now has 1 reference (previously orphaned) — resolved.
- cache-memory adoption: 78 (recheck; prior runs reported ~99, methodology drift again on this metric — needs a stable single regex going forward: `cache-memory:` at start of line).
- max-tool-denials: 65 (flat), network config: 159 (flat/slight rise), sandbox: 254 (majority default), model overrides: 102 (higher than earlier readings — methodology likely counting `model:` in MCP/tool configs, not just engine.model).
- No new Copilot CLI Go features detected since last run (same file set: copilot_engine.go, copilot_engine_execution.go, copilot_engine_tools.go, copilot_mcp.go, etc.)
- RECOMMENDATION SHIFT: metrics for cache-memory and model-override have inconsistent regex methodology across runs, causing false "trend" signals. Future runs should use a fixed canonical grep pattern documented once, rather than re-deriving each time.

## Run 31293794264 (2026-08-09)
- Total workflows 284 (up from 283); copilot-engine canonical count 132 (stable methodology: `^engine: copilot$` or `^id: copilot$`).
- --share flag: still stuck at 1 (this research workflow only) — persistent gap across 8+ runs. Confirmed as accepted low-priority non-issue.
- engine.args: confirmed still 0 — persistent, stable zero across 4+ direct re-verifications.
- add-dir custom usage: 0 real workflows (only this research workflow's own doc references).
- disable-builtin-mcps: 1 real usage (auto-triage-issues.md), unchanged.
- Orphaned custom agents UNCHANGED for 8+ runs: create-safe-output-type, custom-engine-implementation, grumpy-reviewer, interactive-agent-designer, w3c-specification-writer (5 total, 0 references each).
- cache-memory adoption: 78 (flat vs last run using same canonical regex — plateaued, no methodology drift this time).
- max-tool-denials: 65 (flat), max-continuations: 11 (flat), network config: 159 (flat), sandbox: 255 (flat/slight rise with new workflows).
- model overrides for copilot engine: 0 confirmed with corrected regex (prior "102" reading was measurement noise from counting MCP/tool `model:` fields, not engine.model — resolved as false positive).
- No new Copilot CLI Go features detected in codebase since last run (same file set unchanged).
- RECOMMENDATION: The 5 orphaned custom agents have now been stable/unreferenced for 8 consecutive runs — recommend either wiring them into a workflow or removing them, rather than continuing to flag every run.

## Run 31354820498 (2026-08-10)
- 9th consecutive run confirming same persistent gaps, no new movement:
  - --share flag: still 1 (this workflow only), 9+ runs.
  - engine.args: still 0, stable.
  - add-dir custom: still 0 real workflows.
  - disable-builtin-mcps: still 1 real usage (auto-triage-issues.md).
  - Orphaned agents unchanged for 9 runs: create-safe-output-type, custom-engine-implementation, grumpy-reviewer, w3c-specification-writer (interactive-agent-designer now has 1 ref, resolved off orphan list last run and confirmed still referenced this run).
- SKIPPING full issue creation this run per prior recommendation: findings are fully stable/duplicated across 8+ prior issues already filed. Filing a noop instead of a new near-duplicate issue to avoid issue-tracker noise.
- Recommend: next research run should only file a new issue if (a) a new Copilot CLI feature appears in the Go codebase, or (b) any of the 4 orphaned agents get removed/wired-in, or (c) --share/engine.args adoption changes from 0/1.

## Run 31562860555 (2026-08-12)
- Total workflows grew 284->285. Copilot combined (id: copilot + engine: copilot) count: 139.
- --share flag: still stuck at 1 (this research workflow only) — 11th consecutive run confirming persistent gap.
- engine.args: reconfirmed true value is 0 (no false positives this run).
- Orphaned custom agents unchanged 11th consecutive run: create-safe-output-type, custom-engine-implementation, grumpy-reviewer, interactive-agent-designer, w3c-specification-writer (5 of 11 agent files).
- cache-memory adoption continues climbing: 126 (up from ~100-101 recent runs) — durable growth trend.
- copilot-sdk: true adoption: 71 (up from 68) — steady growth.
- max-continuations flat at 11, max-tool-denials stable ~65.
- No new Copilot CLI features detected in codebase since last run.
- Filed issue with full findings and recommended action items (orphaned agent triage + CI lint check for orphan detection).

## Run 31666761449 (2026-08-13)
- NEW FEATURE DETECTED: PR #52377 (merged 2026-08-12) introduced top-level `model` field replacing deprecated `engine.model`, with `gh aw fix` codemod `engine-model-to-top-level` for auto-migration.
- Adoption already underway: 32/138 copilot workflows (23%) use the new top-level `model:` field; ~147 workflows repo-wide still reference legacy nested `engine.model` and are candidates for the codemod.
- Total workflows 285 (flat), copilot combined count 138 (flat vs 139 last run - within noise of regex/methodology).
- --share flag: still 1 (this workflow only) — 12th consecutive run, confirmed persistent non-issue.
- engine.args: still 0, stable across 5+ direct re-verifications.
- disable-builtin-mcps: 2 real usages (auto-triage-issues.md + this research workflow).
- add-dir: 0 real workflow usages (doc references only).
- Orphaned custom agents: create-safe-output-type, custom-engine-implementation, grumpy-reviewer, w3c-specification-writer unchanged 12 consecutive runs (interactive-agent-designer resolved off list, still 1 reference). RECOMMEND: file a one-time cleanup PR to either remove or reference these 4 files rather than continuing to flag every run.
- Filed this run's issue to spotlight the new top-level model field migration opportunity as primary actionable finding.

## Run 31769511486 (2026-08-14)
- Total workflows flat at 285. Narrow `engine: copilot` count: 38 (this run used stricter grep than prior runs' 138/139 combined metric - methodology varies run to run, treat combined count as noisy).
- MAJOR MOVEMENT: top-level `model:` field adoption jumped from 32 (run 31666761449, 2026-08-13) to 98 workflows this run. Legacy nested `engine.model:` usage dropped correspondingly from ~147 to 95. The `gh aw fix` codemod (engine-model-to-top-level) migration appears to be actively rolling out repo-wide - this is the most significant trend since tracking began.
- --share flag: 0 real workflow usages this run (previously counted as 1, but that count was this research workflow's own doc/example text, not real usage) - confirmed still a non-issue, no workflow needs conversation sharing.
- engine.args: confirmed still 0 - stable across 6+ direct re-verifications, persistent non-issue.
- Orphaned custom agents UNCHANGED for 13th consecutive run: create-safe-output-type, custom-engine-implementation, grumpy-reviewer, w3c-specification-writer (interactive-agent-designer remains referenced/resolved).
- No new Copilot CLI Go features detected in codebase since last run (same file set: copilot_engine.go, copilot_engine_execution.go, copilot_engine_tools.go, copilot_mcp.go, copilot_installer.go, copilot_inline_driver.go, copilot_logs.go).
- DECISION: Filing noop this run - the only real news (top-level model migration) was already the headline finding of the immediately preceding issue (run 31666761449, filed 2026-08-12/13), and its continued progress doesn't yet warrant a fresh issue. Orphaned agents remain the same unaddressed low-priority cleanup item flagged for 13 runs. No new actionable finding distinct from what's already tracked in open issues.

## Run 31862974294 (2026-08-15)
- Total workflows flat at 285. Narrow `engine: copilot` count flat at 38.
- --share flag: still 1 (this research workflow only) — 13th+ consecutive run confirming persistent non-issue.
- engine.args: reconfirmed still 0 — the "args:" hits in eslint-miner.md/jsweep.md/smoke-copilot.md are LSP server config args (`lsp.typescript.args: ["--stdio"]`), not engine.args; true engine.args usage remains 0.
- Orphaned custom agents unchanged again: create-safe-output-type, custom-engine-implementation, grumpy-reviewer, w3c-specification-writer (4, interactive-agent-designer remains resolved with 1 ref).
- disable-builtin-mcps: flat at 2 real usages (auto-triage-issues.md + this research workflow).
- CODEBASE CHANGE DETECTED: commit c35faf4 "feat!: remove Firecracker support" (#52774) removed Firecracker sandbox support entirely from the repo (no remaining references in pkg/workflow or docs). This shrinks the sandbox feature surface (AWF/SRT remain) but is not a Copilot CLI usage gap — no action needed, just noted for inventory accuracy.
- No other new Copilot CLI Go features detected since last run.
- DECISION: Filed noop per established policy (run 31354820498+) — only file new issue when (a) new Copilot CLI feature appears, (b) orphaned agent status changes, or (c) --share/engine.args adoption changes from 0/1. None of these conditions met this run.

## Run 31925380594 (2026-08-16)
- Total workflows flat at 285. Narrow `engine: copilot` count flat at 38.
- --share flag: still 1 (this research workflow only) — 14th+ consecutive run confirming persistent non-issue (no workflow needs conversation sharing).
- engine.args: reconfirmed still 0 — stable across 7+ direct re-verifications.
- Orphaned custom agents unchanged: create-safe-output-type, custom-engine-implementation, grumpy-reviewer, w3c-specification-writer (4, interactive-agent-designer remains resolved with 1 ref via archie.md-style agent: usage).
- disable-builtin-mcps: flat at 2 real usages (auto-triage-issues.md + this research workflow's own doc example).
- copilot-sdk: true adoption: 71 (flat vs last run).
- No new Copilot CLI Go features detected in codebase since last run (same 8-file module set unchanged).
- DECISION: Filed noop per established policy (run 31354820498+) — only file new issue when (a) new Copilot CLI feature appears, (b) orphaned agent status changes, or (c) --share/engine.args adoption changes from 0/1. None of these conditions met this run.

## Run 31992795476 (2026-08-17)
- Total workflows 284 (was 285 last run; minor count drift, not investigated further as not material to Copilot CLI usage). Narrow `engine: copilot` count flat at 38.
- --share flag: real usage still 0 (only appears in this research workflow's own doc/example text) — 15th+ consecutive run confirming persistent non-issue.
- engine.args: reconfirmed still 0 real usages — the 103 `args:` hits across workflows are non-engine config (LSP server args like `lsp.typescript.args: ["--stdio"]`), not `engine.args`.
- Orphaned custom agents unchanged (15th+ consecutive run): create-safe-output-type, custom-engine-implementation, grumpy-reviewer, w3c-specification-writer (0 references each).
- disable-builtin-mcps: flat at 2 real usages (auto-triage-issues.md + this research workflow's own doc example).
- copilot-sdk: true adoption: 70 (flat, -1 noise vs last run's 71).
- max-tool-denials: 64, max-continuations: 11, network config: 158, sandbox: 157 — all flat/consistent with recent runs.
- No new Copilot CLI Go features detected in codebase since last run (same module set: copilot_engine.go, copilot_engine_execution.go, copilot_engine_tools.go, copilot_mcp.go, copilot_installer.go, copilot_engine_installation.go, copilot_inline_driver.go, copilot_logs.go, pkg/cli/copilot_*.go).
- DECISION: Filed noop per established policy (run 31354820498+) — only file new issue when (a) new Copilot CLI feature appears, (b) orphaned agent status changes, or (c) --share/engine.args adoption changes from 0/1. None of these conditions met this run.
