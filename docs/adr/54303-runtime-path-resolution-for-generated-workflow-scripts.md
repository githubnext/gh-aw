# ADR-54303: Runtime Path Resolution for Generated Workflow Scripts

**Date**: 2026-08-20
**Status**: Draft
**Deciders**: pelikhan, copilot-swe-agent

---

### Context

The `gh aw compile` command generates `.lock.yml` workflows that previously embedded GitHub expression syntax (`${{ runner.temp }}`) directly inside executable `actions/github-script` JavaScript bodies and shell `run:` commands. CodeQL flags this pattern as a potential code injection vulnerability (`actions/code-injection/medium`) because GitHub expressions are resolved before script execution, making it possible for externally-controlled values to influence the executable body. The project needed a generation strategy that satisfies CodeQL's static analysis rules without requiring manual patches to generated artifacts.

### Decision

We will change the code generator to resolve runner temp paths at runtime using environment variables instead of embedding GitHub expression syntax in executable bodies. For JavaScript, generated code will use `path.join(process.env.RUNNER_TEMP, ...)` for path construction. For shell commands, generated scripts will use quoted `${RUNNER_TEMP}` variable references. We will also add a compile-time validation step that rejects any generated executable body still containing the unsafe `${{ runner.temp }}/gh-aw/actions` pattern.

### Alternatives Considered

#### Alternative 1: Suppress CodeQL Alerts for Generated Files

Add CodeQL suppression annotations or exclude generated lock files from CodeQL scans so the existing generation pattern continues unchanged.

This was not chosen because suppressing alerts does not eliminate the underlying security risk — the actual code-injection vulnerability remains present, and suppression could hide genuine future regressions. It also couples the project's security posture to alert-suppression bookkeeping rather than safe code generation.

#### Alternative 2: Avoid Using Runner Temp Entirely

Use a different directory strategy (e.g., workspace-relative paths, pre-installed action locations) instead of `runner.temp` for action resolution at all.

This was not chosen because `runner.temp` is the established contract between `gh aw compile` and the generated workflows for action placement. Changing that contract would require broader changes across the toolchain and all existing generated workflows, which is a significantly higher-risk refactor than adjusting how paths are constructed within scripts.

### Consequences

#### Positive
- Generated lock workflows no longer trigger CodeQL `actions/code-injection/medium` alerts, removing a source of security noise for downstream consumers.
- The compile-time validation guard ensures the unsafe pattern cannot be silently reintroduced by future compiler changes.
- Runtime environment variable resolution is a safer and more idiomatic pattern for referencing runner paths inside executed script bodies.

#### Negative
- Generated JavaScript bodies become slightly more verbose, requiring a `path.join` call for each action `require()` instead of a single string interpolation.
- The added validation logic increases compiler complexity and may produce opaque errors if the detection heuristic needs updating.

#### Neutral
- All existing checked-in lock files and WASM golden fixtures must be recompiled as part of this change; this is a one-time migration cost with no ongoing maintenance burden.
- The change does not affect the runtime behavior of generated workflows — only how paths are represented in the generated source.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
