# Task Mining Run - 2026-07-28

## Summary
- Discussions scanned: 5 (unprocessed since last run on 2026-07-18)
- Tasks identified: 7
- Issues created: 4
- Duplicates avoided: 3 (already tracked in open issues)

## Created Issues
- Propagate caller context through remote workflow spec fetchers (remote_workflow_spec.go) — from discussion #48389
- Thread context.Context through upgradeExtensionIfOutdated for cancellable release checks — from discussion #48389
- Propagate live context into buildRunsModel / fetchJobStatusesForProcessedRun (logs_orchestrator_filters.go) — from discussion #48389
- Split 117-line generateSetupStepWithArtifactClientCondition into per-mode helpers — from discussion #48508

## Duplicates Skipped
- ExecGH/RunGH/RunGHCombined deprecation — already tracked in an open "[Code Quality]" issue
- ctxbackground linter wrapper-pattern extension — already tracked in an open "[Code Quality]" issue
- syft image-scan context propagation — already tracked in an open "[Code Quality]" issue

## Discussions Reviewed But No New Tasks Extracted
- #48511 (lint-monster) — already self-managing via its own issue-creation workflow (created its own tracking issues)
- #48572 (terminal-stylist) — no action needed; report concludes codebase is already consistent, only optional low-value polish suggested
- #48462 (observability) — findings are about missing telemetry/logs in specific workflow *runs*, not a code-quality defect with a fixable file/line; not actionable as a 1-3 day code task

## Top Patterns Observed
- context.Background() call-chain leakage remains the dominant recurring theme (context propagation gaps across parser, cli, and compiler packages)
- Large/complex functions in compiler YAML generation code continue to surface in daily quality scans
