# ADR-50991: Add Log-Parser Support for Behavior-Defined Engines

**Date**: 2026-08-07
**Status**: Accepted
**Deciders**: gh-aw maintainers

---

### Context

The gh-aw engine compilation pipeline produces GitHub Actions workflows that include a post-execution log-parsing step to generate step summaries and normalized event data. Built-in engines (e.g., Claude Code) each have a hardcoded JavaScript parser baked into the compiler. Behavior-defined engines — declarative engines whose runtime behavior is described entirely in YAML (crush, opencode, goose, aider) — had no equivalent hook, leaving them unable to produce rich step summaries or normalized event files. This created an observability gap where community-contributed or declarative engines could run tasks but not provide the same post-run analysis as built-in engines.

### Decision

We will add a `log-parser` field to `EngineBehaviorDefinition` that accepts an inline JavaScript snippet containing a `parseLog(logContent)` function. During compilation, `BehaviorDefinedEngine` will override `GetLogParserScriptId()` to return a stable script ID (`<engine-id>_log_parser`) and emit a write step that materializes the inline JS to `${RUNNER_TEMP}/gh-aw/actions/` using the same heredoc-with-delimiter-safety pattern already used for `harness-script`. The raw `parseLog` function is auto-wrapped by `createEngineLogParser` from `log_parser_shared.cjs`, so engine authors provide only the parsing logic and inherit file reading, event enrichment, and step-summary generation for free.

### Alternatives Considered

#### Alternative 1: Hardcode per-engine parsers inside the compiler

Each of the known behavior-defined engines (crush, opencode, goose) could have its parser added directly to the compiler as built-in cases, matching the existing pattern for built-in engines. This would avoid inline JS in YAML and allow parsers to be tested as standalone modules. It was rejected because it does not scale to future community-contributed behavior-defined engines and requires compiler changes for every new engine that wants log parsing — defeating the purpose of the declarative engine model.

#### Alternative 2: Reference an external parser script file per engine

Engine definitions could point to a file path (relative to the repo or a URL) that the compiler fetches and embeds. This would allow parsers to be versioned, tested, and linted independently. It was rejected because it introduces a distribution and versioning problem — the script must be reachable at compile time, and the current engine-definition model is self-contained single-file YAML. Inline JS stays consistent with the existing `harness-script` and `config-adapter` patterns.

### Consequences

#### Positive
- Behavior-defined engines gain parity with built-in engines for log parsing and step summary generation without requiring compiler changes per engine.
- Engine authors write only a single `parseLog(logContent)` function and automatically inherit the full bootstrap (file I/O, event enrichment, step summary) from `createEngineLogParser`.
- The pattern is consistent with existing `harness-script` and `config-adapter` inline-JS fields, so engine authors already familiar with those patterns face minimal learning curve.

#### Negative
- Inline JavaScript embedded in YAML is harder to unit-test and lint than a standalone `.cjs` module file; parser bugs will surface only at runtime in CI.
- Each compiled workflow lock file embeds the full parser source verbatim, increasing lock-file size proportionally with parser complexity and duplicating the code across every job that uses the engine.

#### Neutral
- The `log-parser` field follows `additionalProperties: false` JSON schema validation, so malformed entries are caught at schema validation time rather than at runtime.
- The auto-wrapping via `createEngineLogParser` means the interface contract (`parseLog` must return `{ markdown, logEntries, mcpFailures, maxTurnsHit }`) is enforced by convention rather than by a type system.

---

*ADR reviewed and accepted. See PR [#50991](https://github.com/github/gh-aw/pull/50991) for implementation details.*
