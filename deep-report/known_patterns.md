## DeepReport Memory (2026-08-11T14:57:00Z)

### Meta: path/persistence fix holding
Repo-memory files written to the corrected one-level path (`deep-report/`) on 2026-08-10 were still present and readable at the start of this cycle (2026-08-11). The workaround is stable. The underlying source-file fix (`.github/workflows/deep-report.md` writing to the wrong two-level path) does not yet appear landed — no merged-PR evidence found this cycle either — so the issue filed 2026-08-10 to fix it for real should stay open until a PR SHA can be cited.

### Headline finding this cycle: fleet-wide agent-job failure rate spiked to 49%
Safe Output Health Monitor (#51935, 2026-08-11) audited all 210 runs in its 24h window and found the `safe_outputs` job itself perfectly healthy (0 failures) but flagged — explicitly out of its own scope — that 103/210 runs (49.0%) had an `agent`-job failure (`Execute Claude Code CLI`/`Execute GitHub Copilot CLI`/`Ingest agent output` steps). This is a large jump from the 2026-08-10 cycle's 10% driver-exit sample (50 runs). Cross-referenced against the same-day Agent Performance Report (#52052): several individual workflows already show high failure rates (PR Sous Chef 84% fail, Issue Monster 80% fail, Contribution Check 67% fail, several at 100% fail with ≥2 runs) plus a known P0 Copilot CLI segfault (#51789) — these could explain a large chunk of the 49% without necessarily being a new fleet-wide regression, but no one has actually done that reconciliation. Filed an issue to investigate/attribute the 49% figure since the monitor that found it explicitly said it's outside its mandate and no owner exists.

### New concrete/actionable findings this cycle (all filed as issues, dedup-checked against open issues first)
1. `strict:` mode docs in `frontmatter.md` describe the opposite of actual behavior (schema + compiler agree `strict` is the stronger security mode; docs imply the reverse) — schema-consistency #51954.
2. `user-rate-limit.events` schema enum omits `repository_dispatch`, which the compiler already treats as a valid inferred trigger — schema-consistency #51954.
3. README's automated agent-bootstrap block never passes `--engine claude`, silently producing Copilot-oriented artifacts for Claude Code users; `CLAUDE_CODE_OAUTH_TOKEN` unsupported-ness is buried in one note — claude-code-user-docs-review #52047.
4. `JobStep`/`JobStepData` are byte-identical duplicate structs in `pkg/cli` requiring manual conversion — typist #52015.
5. `AccessLogEntry`/`FirewallLogEntry`/`AuditLogEntry`/`GatewayLogEntry` are 4 independent structs modeling the same "parsed log line" concept with no shared base — typist #52015.
6. `pkg/workflow/compiler_types.go` (55 top-level functions, 900+ LOC) mixes `CompilerOption` builders with `*Compiler` runtime mutators — two concerns, one file, on a high-churn package — repository-quality #52059.
7. Fleet-wide 49% agent-job failure rate (above) — safe-output-health #51935.

### Chronic lineages — status check this cycle (not re-filed, per prior cycle's process-gate judgment)
- Copilot Session Insights conversation transcripts: still 0 files this cycle (#51985, 2026-08-11) — now stated as "present in every prior analysis... going back to at least 2026-03-20 (~4.5 months)" by the reporting agent itself, a longer-than-previously-stated gap (was "44+ days" on 2026-08-10). Folding into existing process-gate issue evidence rather than re-filing.
- Sergo (#51930) self-filed 2 new issues this cycle (errorfwrapv false positive, ctxbackground enforce-readiness) — not duplicated here.
- LintMonster (#51918) and ESLint Refiner (#51959) both self-filed/updated their own tracking issues this cycle (function-length backlog #50164, dynamic-regexp remediation, DI-fallback binding gap, path-insensitive listener check) — not duplicated here.
- Docs-noob-tester (#51926) re-flagged frontmatter terminology / lock.yml duplication / secret-setup detour — same chronic non-blocking class as before, still not re-filed as new duplicates.

### Fleet health baseline carried forward from 2026-08-10 (agenticworkflows `logs` tool timed out twice this cycle — see flagged_items.md — so this cycle relies on same-day discussion reports instead of raw log pulls)
- Firewall (security-observability #51835, 7d window ending 2026-08-10): 100 firewall-enabled runs (72 with data), 4,514 requests, 4,364 allowed / 150 blocked (3.32% block rate), concentrated on `proxy.golang.org` (126, Go module fetches) and `api.individual.githubcopilot.com` (24, Copilot API variant not allowlisted).
- Daily Issues Report (#52081, 1000-sample): 835 closed / 165 open, 83.5% closure rate, ~13h avg close time, 932/1000 filed by github-actions bot, only 7 unlabeled, 0 stale (30+ days).
- Issues-analyst subagent (500-sample, this cycle): 141 open / 359 closed. Top labels: agentic-workflows (262), automation (137), cookie (113), testing (44), bug (33). No issues open >7 days (all opened within last 4 days — consistent with fast bot-driven churn). Only 3 unlabeled.
