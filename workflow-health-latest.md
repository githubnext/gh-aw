# Workflow Health — 2026-06-21T06:09Z

Score: 66/100 (→ stable from 66)
Workflows: 249 | Lock files: 249/249 (100% ✅) | Run: §27895532113

## KEY FINDINGS

### Status (June 21)
- **Compilation:** 249/249 workflows have lock files (100% ✅). Compile-validate clean.
- **Today's runs (early):** PR Sous Chef 1 failure (transient, #40586 filed), CI 1 failure (non-agentic). Smoke cluster continuing.
- **Code Simplifier (Day 15, #39968/#40431/#40577):** Still failing. PR #40578 open (Copilot AI fix, pelikhan review pending). DO NOT RE-FILE.
- **Daily Model Inventory Checker (Day 11+, #39471):** Still failing. session.idle 60s. DO NOT RE-FILE.
- **Daily Safe Outputs Git Simulator:** ✅ RECOVERED — 1/1 success today. Was Day 12+ failure.
- **Daily Safe Output Integrator (Day 12+, #39477):** Still failing. Tool denial. DO NOT RE-FILE.
- **Daily BYOK Ollama Test (Day 12+, #39476/#40417):** Still failing. api-proxy cap. DO NOT RE-FILE.
- **Tool Denial Cluster (systemic, filed Jun 16):** 7+ workflows. DO NOT RE-FILE.
- **Smoke cluster (~75-95% fail, #38998):** Continuing. DO NOT RE-FILE.
- **AIC Budget Crisis (Day 15, #39077):** Root fix still pending. Code Simplifier fix PR open. DO NOT RE-FILE.
- **Daily News: push_repo_memory orphan branch (#40190):** GH013 unsigned commit. DO NOT RE-FILE.
- **Daily Compiler Quality Check (#39724/#39949/#40565):** gpt-5-mini unsupported, Day 4+ new issues filed. DO NOT RE-FILE.
- **LintMonster (#39511):** Alternating pattern continues. DO NOT RE-FILE.

### Positive Changes (Jun 21) ✅
- **Daily Safe Outputs Git Simulator: RECOVERED** — first success after Day 12+ failures.
- **Code Simplifier PR #40578 open** — Copilot AI authored fix for api-proxy issue, pending review.
- **PR Sous Chef failure (#40586):** 1 failure this run — was 100% yesterday, likely transient.

### New Issues Filed By aw-failures (Jun 21)
- #40586 — PR Sous Chef failed
- #40577 — Code Simplifier failed (Day 15)
- #40574 — Smoke Copilot AOAI (apikey) no safe outputs
- #40565 — Daily Compiler Quality Check no safe outputs
- #40563 — Smoke Copilot missing required tool
- #40557 — Smoke Codex missing required data
- #40530 — Smoke Copilot AOAI (Entra) missing required tool

### Actions Taken This Run
- 0 new issues created (all already tracked by aw-failures)
- 1 dashboard comment added to #40569
- Updated workflow-health-latest.md and shared-alerts.md

## Do Not Re-File (Jun 21 state)
- All items from Jun 20 remain active
- Daily Safe Outputs Git Simulator: RECOVERED — remove from do-not-refile when stable 3+ days
