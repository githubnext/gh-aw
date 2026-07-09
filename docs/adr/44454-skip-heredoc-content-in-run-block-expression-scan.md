# ADR-44454: Skip Heredoc Content in Run-Block Expression Scan

**Date**: 2026-07-09
**Status**: Draft
**Deciders**: Unknown

---

### Context

`CompileSimpleWorkflow` regressed ~3x in execution time (5.67ms → 14.6ms) with ~12x more allocations after a change that introduced MCP tool support. Workflows using MCP tools generate `run:` steps that pipe JSON configuration through shell heredocs using a unique compiler-generated delimiter (e.g., `cat << GH_AW_MCP_CONFIG_01641c1bd0fd81fa_EOF | "$GH_AW_NODE" ...`). These heredoc bodies contain `${{ toJSON(...) }}` expressions that are not present in `allowedRunScriptExpressionRegex`. The text-scanning path (Path B) of `validateTemplateInjection` — `walkRunBlockLines` / `scanRunContentExpressions` — visited every line in multiline run blocks without awareness of heredoc boundaries, causing it to flag heredoc-embedded expressions as disallowed on every compilation. This set `hasDisallowed=true` unconditionally, triggering a full `yaml.Unmarshal` (~434 MB of allocations per 100-iteration benchmark run) even though the parsed path (Path A, `validateNoGitHubExpressionsInRunScriptsFromParsed`) already calls `removeHeredocContent` and never raises an error for these expressions.

### Decision

We will add heredoc-state tracking to `walkRunBlockLines` so that lines inside heredocs are skipped during expression scanning on Path B. This is implemented by adding `detectHeredocDelimiter` — a heuristic function that extracts the closing delimiter from a heredoc-opening line (handling unquoted, `<<-`, single-quoted, and double-quoted forms) — and threading `inHeredoc` / `heredocDelimiter` state into the walk loop. The heredoc-opening line itself is still visited (for context), but all body lines until the closing delimiter are skipped. This aligns Path B's scanning semantics with Path A's `removeHeredocContent` behavior.

### Alternatives Considered

#### Alternative 1: Expand `allowedRunScriptExpressionRegex` to include `toJSON(...)` patterns

The immediate trigger was `toJSON(steps.determine-automatic-lockdown.outputs.visibility)` not matching the allowlist. Adding this specific pattern (or a broader `toJSON(...)` rule) to the regex would have eliminated the false positive for this case. However, this is a band-aid: any future expression pattern generated inside heredoc content that is not in the allowlist would re-trigger the regression. It also conflates "this expression is safe in heredoc context" with "this expression is allowed in general shell context," which are distinct security questions. Rejected because it does not fix the structural root cause.

#### Alternative 2: Cache `yaml.Unmarshal` results per workflow content hash

The expensive `yaml.Unmarshal` call on the `hasDisallowed` fallback path could be memoized using the raw YAML content as a cache key. This would avoid repeated parsing across multiple compilations of the same workflow. However, it introduces state (cache invalidation, memory pressure), does not fix the false-positive detection that triggers the fallback, and masks the underlying issue rather than correcting it. Rejected because caching a path that should not be taken is worse than avoiding the path entirely.

### Consequences

#### Positive
- `CompileSimpleWorkflow` performance is restored to baseline: ~11ms/op → ~3.8ms/op (below the historical 5.67ms/op average), and allocations drop from 71,945/op to 6,230/op
- Path B scan semantics now match Path A: both skip heredoc content, eliminating a class of false-positive expression detections for heredoc-embedded `${{ ... }}` in compiler-generated run steps
- The `detectHeredocDelimiter` function is independently testable and covered by 7 unit test cases

#### Negative
- `detectHeredocDelimiter` is a heuristic text parser: it detects `<<` as a heredoc opener, which can produce false positives for shell comparison operators (e.g., `if [ $x < 5 ]`). The test suite covers this case and it does not trigger a false `inHeredoc` flag (since `< 5` has a space before the `<`, but `<<` does not match a single `<`), but unusual shell constructs that happen to contain `<<` in non-heredoc positions could theoretically be misidentified
- The walk loop becomes stateful (two additional variables per invocation), adding modest complexity to a previously stateless function

#### Neutral
- The fix is localized entirely to `walkRunBlockLines` and `detectHeredocDelimiter` in `template_injection_validation.go`; no changes to the parsed-path validators or the `CompileSimpleWorkflow` API
- Tests are added for both the heredoc-skipping behavior (`TestScanRunContentExpressionsHeredoc`, 6 cases) and the delimiter extractor (`TestDetectHeredocDelimiter`, 7 cases)

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
