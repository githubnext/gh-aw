## Extracted code-quality tasks (2026-08-11 cycle)

1. Fix inverted `strict:` mode documentation in frontmatter reference — filed.
2. Add missing `repository_dispatch` to `user-rate-limit.events` schema enum — filed.
3. README agent-bootstrap block silently defaults Claude Code users to Copilot-oriented artifacts — filed.
4. Consolidate `JobStep`/`JobStepData` identical structs in `pkg/cli` — filed.
5. Give the 4 independent log-entry structs a shared base type — filed.
6. Split `pkg/workflow/compiler_types.go`: separate `CompilerOption` builders from `Compiler` mutators — filed.
7. Investigate 49% agent-job failure rate across 210-run fleet sample (2026-08-10→11) — filed.

Not filed (evidence folded into existing 2026-08-10 process-gate issue instead, to avoid re-filing already-repeatedly-closed lineages):
- Copilot Session Insights conversation-transcript gap (now ~4.5 months per source agent, up from "44+ days" stated 2026-08-10).
- Docs-noob-tester chronic findings (frontmatter terminology, `.lock.yml` duplicate explanation, secret-setup detour) — same chronic non-blocking class as prior cycles.

Not filed (already self-filed by the reporting agent itself this cycle, cross-checked for no gap):
- Sergo: errorfwrapv false positive, ctxbackground enforce-readiness.
- LintMonster: function-length backlog (#50164), dynamic-regexp remediation.
- ESLint Refiner: DI-fallback binding gap (`require-spawn-error-listener`), path-insensitive listener check.

Previous cycle's 2026-08-10 tasks (for reference, all previously filed):
1. Require verified-merged evidence before closing self-filed reliability/doc issues (process gate).
2. Land the DeepReport repo-memory path fix from #51172 for real.
3. Add gh-aw-detection: true to Q and ESLint Monster.
4. Reformat audit-workflows recommendations.json/workflow-trends.json to indent=2.
5. Add native-counterpart doc comments to progress_wasm.go / spinner_wasm.go.
6. Investigate firewall/MCP log retention via upload-side glob/path-depth hypothesis.
