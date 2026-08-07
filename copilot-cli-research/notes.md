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
