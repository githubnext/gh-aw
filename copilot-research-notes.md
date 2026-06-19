# Copilot CLI Research Notes (Last 9 runs)

### 2026-06-19 (Run 27807178037) — This Run
- **250 total workflows** (same as 06-18)
- 133 Copilot (~53%): 36 scalar + 96 object; 63 copilot-sdk
- **✅ IMPROVEMENTS vs last run:**
  - `engine.agent`: 24 → 29 (+5, +21% — continuing recovery trend)
  - `engine.agent awf`: 13 → 16 (+3 — AWF migration continuing)
- **⚠️ REGRESSIONS vs last run:**
  - Copilot scalar form: 40 → 36 (-4 — some removed/converted to object form)
  - Total copilot: 136 → 133 (-3 net)
- **engine.agent breakdown**: 16 AWF + ~13 custom agents + 5 blank values
- **max-continuations**: 7 (stable)
- **copilot-sdk**: 63 (stable — 47% of copilot workflows)
- **copilot-sdk-driver**: 3 (stable)
- **cache-memory**: 72 (stable)
- **network configured**: 132 (stable)
- **sandbox**: 23 (17% — stable)
- **min-integrity**: 34 (stable)
- **max-ai-credits**: 18/133 (14% — 115/133 still missing!)
- **max-tool-denials**: 0/63 SDK (PERSISTENT GAP — 8th consecutive run)
- **experiments**: 41 (stable)
- **append-only-comments**: 8 (stable — smoke workflows only)
- **mcp.session-timeout**: 0 (PERSISTENT GAP)
- **mcp.tool-timeout**: 0 (PERSISTENT GAP)
- **blocked-domains**: 0 (consistently zero)
- **startup-timeout**: 1/250 (barely used)
- **engine.args**: 0 in engine block (PERSISTENT GAP, 21st consecutive run)
- **engine.api-target**: 0 (PERSISTENT GAP, 21st consecutive run)
- **engine.harness**: 0 (never used as custom override)
- **engine.token-weights**: 0 (never used)
- **Unused agent files**: grumpy-reviewer, interactive-agent-designer, w3c-specification-writer, create-safe-output-type, custom-engine-implementation (5 files — unchanged)
- **Model overrides**: 63 workflows (54 small, 6 large, others specific models)
- Discussion created: "Copilot CLI Deep Research - 2026-06-19"

### 2026-06-18 (Run 27738311748)
- **250 total workflows** (up from 249 on 06-16 — +1 net)
- 136 Copilot (~54%): 40 scalar form + 96 object form; 63 copilot-sdk
- **✅ IMPROVEMENTS vs last run:**
  - `engine.agent`: 8 → 24 (+200% — major recovery! exceeds prior 21 state, still below 34 peak)
    - 13 using `agent: awf` (AWF firewall mode migration pattern)
    - 11 using custom agent files
  - `append-only-comments`: 0 → 8 (new feature with strong adoption)
  - `total workflows`: 249 → 250 (+1)
- **⚠️ REGRESSIONS vs last run:**
  - `blocked-domains`: 1 → 0 (the one user removed it — now 0/250!)
- **max-continuations: 7** (stable — 5% of copilot workflows)
- **copilot-sdk: 63** (stable — 46% of all copilot workflows)
- **engine.args**: 0 (PERSISTENT GAP, 20th consecutive run)
- **engine.api-target**: 0 (PERSISTENT GAP, 20th consecutive run)
- Discussion created: "Copilot CLI Deep Research - 2026-06-18"

### 2026-06-16 (Run 27596173580)
- **249 total workflows** (up from 246 on 06-15 — +3 net)
- 136 Copilot (~55%): 40 scalar form + 96 object form; 63 copilot-sdk
- **✅ IMPROVEMENTS vs last run:**
  - `max-ai-credits`: 6 → 18 (+200% — budget guardrails finally spreading!)
  - `min-integrity`: 22 → 34 (+12, improving security posture)
  - `total workflows`: 246 → 249 (+3)
  - `strict: true`: 79 → 151 (likely counting all engines now)
- **⚠️ REGRESSIONS vs last run:**
  - `engine.agent`: 21 → 8 (further decline from 34 peak on 06-10)
- **max-continuations: 7** (stable — 5% of copilot workflows)
- **copilot-sdk: 63** (stable — 46% of all copilot workflows)
- **engine.args**: 0 (PERSISTENT GAP, 19th consecutive run)
- Discussion created: "Copilot CLI Deep Research - 2026-06-16"

### 2026-06-15 (Run 27525865107)
- **246 total workflows** (up from 245 on 06-10 — +1 net)
- 133 Copilot (54%), 47 Claude, 10 Codex
- **⚠️ REGRESSIONS vs last run:**
  - `engine.agent`: 34 → 21 (-13 workflows removed agent references!)
  - `max-ai-credits`: 14 → 6 (-8 workflows lost budget guard)
  - `sandbox`: 20 → 15 (-5 workflows lost sandbox)
- **max-continuations: 7** (up from 6 — slight growth)
- **copilot-sdk: 63** (stable — 47% of all copilot workflows)
- **engine.args**: 0 (PERSISTENT GAP, 18th consecutive run)
- Discussion created: "Copilot CLI Deep Research - 2026-06-15"

### 2026-06-10 (Run 27254548925)
- **245 total workflows** (down from 340 on 06-08 — repo cleanup/consolidation removed ~95 workflows)
- 132 Copilot (54%), 64 Claude, 16 Codex
- **engine.agent: 34 workflows** (up from 13! — +161% growth — major improvement, then reverted)
- **copilot-sdk-driver: 5** (up from 3 — 2 new custom drivers added)
- **max-ai-credits: 14** (up from 5 — +180% improvement, but 118/132 still missing)
- **engine.args**: 0 (PERSISTENT GAP, 17th consecutive run)
- Discussion created: "Copilot CLI Deep Research - 2026-06-10"

### 2026-06-08 (Run 27117423076)
- **340 total workflows** (from 236 on 05-31, +104 or +44% in 8 days)
- 132 Copilot (39%), 64 Claude, 16 Codex
- **copilot-sdk: 63 workflows (48%)** — MASSIVE adoption
- **BYOK: 2 workflows** (new: Azure OpenAI smoke test added)
- Discussion created: "Copilot CLI Deep Research - 2026-06-08"

### 2026-05-31 (Run 26703913319)
- 236 total workflows; 97 Copilot (41%), 51 Claude, 9 Codex
- **max-continuations**: 5 workflows (unchanged)
- **cache-memory**: 116 workflows (significant growth)
- **sandbox AWF**: 23 workflows
- Discussion created: "Copilot CLI Deep Research - 2026-05-31"

### 2026-05-27 (Run 26491933777)
- 236 total workflows; 125 Copilot (53%)
- Discussion created: "Copilot CLI Deep Research - 2026-05-27"

### 2026-05-21 (Run 26206481620)
- 233 total MD workflows; 100 Copilot (43%)
- Discussion created: "Copilot CLI Deep Research - 2026-05-21"

---

## Key Persistent Gaps (Tracked Across All Runs)

1. **engine.args** — 21+ consecutive runs with ZERO usage (custom CLI arguments)
2. **engine.api-target** — 21+ consecutive runs with ZERO usage (custom API endpoints)
3. **engine.harness** — Never used as custom override (always uses built-in copilot_harness.cjs)
4. **engine.token-weights** — Never used
5. **max-continuations** — Only 7/133 copilot workflows (5%) use autopilot mode
6. **MCP session/tool timeout** — Never configured (engine.mcp.session-timeout, engine.mcp.tool-timeout)
7. **max-tool-denials** — 0/63 SDK workflows (should pair with copilot-sdk: true)
8. **startup-timeout** — Only 1/250 workflows
9. **blocked-domains** — 0/133 copilot workflows (consistently zero)
10. **max-ai-credits** — Only 18/133 copilot workflows (14%) — 115 without budget guardrails

## Trends Summary

- `engine.agent` adoption: 25→25→13→34→21→8→24→**29** (volatile but recovering; AWF migration drives jump)
- `copilot-sdk`: 0 → 63 (stabilized at 63 since June 2026)
- `copilot-sdk-driver`: 0 → 3 → 3 → 3 (stable small usage)
- `max-ai-credits`: 0→5→14→6→18→18→**18** (stable after recovery)
- `min-integrity`: 22→34→**34** (stable)
- `experiments`: 41 (stable)
- `max-continuations` adoption: 5→5→6→6→7→7→**7** (very slow growth)
- `append-only-comments`: 0→8→**8** (stable at smoke workflows)
- `blocked-domains`: 0→1→0→**0** (consistently zero)
- `engine.args`: 0 throughout (21+ consecutive runs — structurally unused)
