# Copilot CLI Research Notes (Last 5 runs)

### 2026-06-22 (Run 27931509105) — Latest
- **249 total workflows** (stable)
- 123 Copilot (~49%): 40 scalar + 86 object; 63 copilot-sdk
- Pi engine: 19 (stable); Claude: **56** (+8 vs 2026-06-20); Codex: **16** (+6)
- **🆕 DISCOVERY**: `max-daily-ai-credits` NOW TRACKED: **73 workflows** (was missed in all prior runs!)
- **✅ IMPROVEMENTS**: engine.agent custom: 8→13 (+5 recovery); min-integrity: 35→43 (+8)
- **✅ IMPROVEMENTS**: Claude +17% (48→56); Codex +60% (10→16) in 2 days
- **max-tool-denials**: 0/63 SDK (PERSISTENT GAP — 10th consecutive run)
- **engine.args**: 0 in engine block (PERSISTENT GAP, 23rd consecutive run)
- engine.env/BYOK: 3 (daily-byok-ollama-test, smoke-copilot-aoai-apikey, smoke-copilot-aoai-entra)
- tracker-id: 89 (stable); cache-memory: 72 (stable); startup-timeout: 1 (ruflo-backed-task)
- Discussion: "Copilot CLI Deep Research - 2026-06-22"

### 2026-06-20 (Run 27860912593)
- **249 total workflows**
- 122 Copilot (~49%): 36 scalar + 86 object; 63 copilot-sdk
- **🆕 NEW: Pi engine**: 19 workflows `id: pi` with `copilot/gpt-5.4` (was 0)
- **🆕 NEW: tracker-id**: 89 workflows (near-instant mass adoption)
- **✅ STABLE**: sandbox.agent: awf: 16; copilot-sdk: 63
- **⚠️ REGRESSION**: engine.agent custom: 13 → 8 (-5)
- max-continuations: 7 (stable); cache-memory: 72 (stable)
- **max-tool-denials**: 0/63 SDK (PERSISTENT GAP — 9th consecutive run)
- **engine.args**: 0 in engine block (PERSISTENT GAP, 22nd consecutive run)
- Discussion: "Copilot CLI Deep Research - 2026-06-20"

### 2026-06-19 (Run 27807178037)
- **250 total workflows**; 133 Copilot (53%): 36 scalar + 96 object; 63 copilot-sdk
- **✅ IMPROVEMENTS**: engine.agent: 24→29 (+5); agent awf: 13→16 (+3 AWF migration)
- **⚠️ REGRESSIONS**: copilot scalar: 40→36; total copilot: 136→133 (-3)
- engine.agent: 29 (awf: 16, custom: 13); max-continuations: 7
- Discussion: "Copilot CLI Deep Research - 2026-06-19"

### 2026-06-18 (Run 27738311748)
- **250 total workflows**; 136 Copilot (~54%): 40 scalar + 96 object; 63 copilot-sdk
- **✅ IMPROVEMENTS**: engine.agent: 8→24 (+200%); append-only-comments: 0→8
- **⚠️ REGRESSIONS**: blocked-domains: 1→0
- Discussion: "Copilot CLI Deep Research - 2026-06-18"

### 2026-06-16 (Run 27596173580)
- **249 total workflows**; 136 Copilot (~55%)
- **✅ IMPROVEMENTS**: max-ai-credits: 6→18 (+200%); min-integrity: 22→34 (+12)
- **⚠️ REGRESSIONS**: engine.agent: 21→8
- Discussion: "Copilot CLI Deep Research - 2026-06-16"

---

## Key Persistent Gaps (All Runs)

1. **engine.args** — 23+ consecutive runs ZERO (custom CLI arguments to copilot binary)
2. **engine.api-target** — all runs ZERO (custom API endpoint for GHEC/GHES)
3. **engine.harness** — Never used as custom override (default copilot_harness.cjs always active)
4. **engine.token-weights** — Never used (custom AI credit cost multipliers)
5. **max-tool-denials** — 0/63 SDK workflows (10th consecutive run, default is 5)
6. **MCP timeouts** — session-timeout and tool-timeout: never configured
7. **startup-timeout** — Only 1/249 workflows (ruflo-backed-task: 300s)
8. **blocked-domains** — 0/249 configured (consistently zero, only mentioned in prompt text)
9. **engine.version pinning** — 0/123 copilot (none pinned to specific version)
10. **max-ai-credits** — Only 18/249 total (7%) — but 73 use max-daily-ai-credits (29%)

## Trends Summary
- `engine.agent custom`: 5→13→8→21→34→8→24→29→8→**13** (volatile, recovering)
- `sandbox.agent:awf`: ~10→16→16→16→**16** (stable)
- `copilot-sdk`: 63→63→63→**63** (locked)
- `min-integrity`: 22→34→35→**43** (improving steadily)
- `Pi engine`: 0→19→**19** (stable experimental)
- `tracker-id`: 0→89→**89** (stable since launch)
- `claude_workflows`: 47→48→**56** (accelerating growth)
- `codex_workflows`: 10→**16** (fast growth)
- `max-daily-ai-credits`: NEWLY TRACKED → **73** (significant existing adoption)
