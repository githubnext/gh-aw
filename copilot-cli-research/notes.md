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
