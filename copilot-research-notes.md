# Copilot CLI Research Notes (Last 5 runs)

### 2026-06-20 (Run 27860912593) — Latest
- **249 total workflows** (down from 250 — -1 net)
- 122 Copilot (~49%): 36 scalar + 86 object; 63 copilot-sdk
- **🆕 NEW: Pi engine**: 19 workflows `id: pi` with `copilot/gpt-5.4` (was 0)
- **🆕 NEW: tracker-id**: 89 workflows (near-instant mass adoption)
- **✅ STABLE**: sandbox.agent: awf: 16; copilot-sdk: 63
- **⚠️ REGRESSION**: engine.agent custom: 13 → 8 (-5); Total copilot: 133→122 (Pi migration)
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

### 2026-06-15 (Run 27525865107)
- **246 total workflows**; 133 Copilot (54%)
- engine.agent: 34→21 (-13); max-ai-credits: 14→6; sandbox: 20→15
- Discussion: "Copilot CLI Deep Research - 2026-06-15"

---

## Key Persistent Gaps (All Runs)

1. **engine.args** — 22+ consecutive runs ZERO (custom CLI arguments)
2. **engine.api-target** — 22+ consecutive runs ZERO
3. **engine.harness** — Never used as custom override
4. **engine.token-weights** — Never used
5. **max-tool-denials** — 0/63 SDK workflows (9th consecutive run)
6. **MCP timeouts** — session-timeout and tool-timeout: never configured
7. **startup-timeout** — Only 1/249 workflows
8. **blocked-domains** — 0/249 (consistently zero)
9. **engine.version pinning** — 0/122 copilot (none pinned)
10. **max-ai-credits** — Only 18/249 total (7%)

## Trends Summary
- `engine.agent custom`: 5→13→8→21→34→8→24→29→**8** (volatile)
- `sandbox.agent:awf`: ~10→16→16→**16** (stable)
- `copilot-sdk`: 63→63→63→**63** (locked)
- `max-ai-credits`: 5→14→6→18→18→**18** (stable after recovery)
- `min-integrity`: 22→34→**35** (improving)
- `Pi engine`: 0→0→**19** (🆕 launched 2026-06-20)
- `tracker-id`: 0→**89** (🆕 mass adoption 2026-06-20)
