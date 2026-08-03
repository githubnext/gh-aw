# Task Mining Run - 2026-08-03

## Summary
- Discussions scanned: 30 (last 7 days, `Audits`/`General`/`Announcements` categories)
- Discussions with candidate code-quality content: 3 (typist #49984, daily-compiler-quality #49887, plus dedup checks against open issues)
- Tasks identified: 5 new + several already covered by existing open issues
- Issues created: 5
- Duplicates avoided: 2 (compiler_safe_outputs_steps.go refactor and compiler_yaml_prompt.go test gap, both already tracked by open #49971/#49928 and #49972/#49927)

## Created Issues
- Consolidate legacy vs AWF sandbox config duplication (BoundedQueriesConfig/AWFBoundedQueriesConfig, SRTNetworkConfig/AWFNetworkConfig)
- Type CompilerError.Type as an ErrorSeverity enum instead of raw string (re-raised; prior #39816 closed NOT_PLANNED without a code fix, duplication still present)
- Type *TimeoutMinutes int constants as time.Duration
- Deduplicate RepositoryFeatures struct across build-tag files (re-raised; prior #45863 closed without a code fix, duplication still present)
- Replace FrontmatterConfig.RunsOn/Imports/Include any fields with typed sum types

## Top Patterns Observed
- Recurring "legacy vs AWF" config duplication pattern in sandbox/networking config (high-value systemic issue)
- Untyped `any` fields with comment-documented shapes, unapplied `RunsOnValue`/`TemplatableBool` sum-type pattern
- Two prior "closed" issues (#39816, #45863) whose fixes were never actually implemented in code
- Most other daily audit-style discussions this window were descriptive/statistical reports (metrics, telemetry coverage, org health) without distinct actionable code-quality tasks, or had already self-filed their own findings as issues

## Source
Extracted primarily from [Typist - Go Type Consistency Analysis discussion #49984](https://github.com/github/gh-aw/discussions/49984)
