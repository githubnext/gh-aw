# ADR-68579: Detect agentic engine execution outside the agent job via compiled-YAML static analysis

**Date**: 2026-10-14
**Status**: Draft
**Deciders**: Unknown (PR #68579, authored by `copilot-swe-agent`)

---

### Context

In `gh-aw`, only the generated `agent` job runs the agentic engine inside the firewall/sandbox configuration derived from `engine:`. Custom steps and custom jobs — including safe-output jobs and imported steps — run **outside** that sandbox with ordinary GitHub Actions security. Nothing previously stopped a workflow author from installing and invoking an engine CLI (Copilot CLI, Claude Code, Codex, Gemini CLI, Agy, Pi) from one of those jobs, which silently bypasses the sandbox, the engine firewall and the MCP/tooling policy that `engine:` is supposed to enforce. The compiler needs a way to catch this at compile time, before a `.lock.yml` is emitted, and it must work even when schema validation is skipped and even for steps that the user never wrote directly (imported or generated).

### Decision

The compiler validation pass, `validateAgenticEngineSteps`, checks **secret references in the final compiled workflow YAML** against the required secret names supplied by registered engines. References outside `agent`, `detection`, and `evals` suggest engine execution outside the sandbox. They produce a hard error in strict mode (no lock file is written) and a compiler warning when `strict: false`. Generated activation credential-validation steps are exempt. A pre-filter checks for matching secret expressions before requiring an extra `yaml.Unmarshal`. Shell commands, package installations, and action names are not inspected.

### Alternatives Considered

#### Alternative 1: Validate the frontmatter/markdown source instead of the compiled YAML

Checking the user-authored workflow markdown would be simpler and would give diagnostics pointing at the exact source line. It was rejected because it cannot see steps contributed by `imports:`, by safe-output job generation, or by built-in job augmentation (e.g. `activation`, `conclusion`) — exactly the places where an injected engine invocation would be hardest to notice. Operating on the final `jobs:` map guarantees that whatever ships in the lock file has been inspected.

#### Alternative 2: Enforce at runtime instead of compile time (sandbox/firewall deny-list)

The engine binaries could be blocked at execution time, e.g. by network policy or by a runner-side guard that fails the step. This gives true enforcement rather than best-effort detection, but it only fails *after* the workflow has been merged and dispatched, costs a run, and cannot cover every install path (vendored binaries, mirrors). Compile-time rejection gives authors immediate, local feedback and keeps the invalid workflow out of the repository; a runtime control could still be added later as defence in depth.

#### Alternative 3: Parse the `run:` script with a real shell parser (e.g. `mvdan/sh`)

A shell AST parser was implemented during review, then removed at the maintainer's request along with its dependency. Secret-only detection avoids shell-specific parsing and its maintenance burden, but deliberately does not detect execution that uses other credentials.

### Consequences

#### Positive
- Recognized engine-secret references outside designated jobs fail to compile in strict mode and produce no lock file.
- Coverage extends to imported steps, safe-output steps and generated job augmentations, not just hand-written custom jobs, because the check runs on the final compiled `jobs:` map.
- Diagnostics name the offending job and step index and point authors at `engine:`, so the remediation is explicit rather than a bare rejection.
- The `strict: false` escape hatch degrades to a warning, so existing workflows are not hard-broken by the new rule.

#### Negative
- Detection is heuristic: engines using renamed secrets, other credentials, or dynamic secret lookups are not detected, so this is not an enforcement boundary.
- Provider secrets may also be used without running an engine, such as model-list API requests. Those workflows require `strict: false`, which changes strictness workflow-wide rather than per-step.

#### Neutral
- `agent`, `detection` and `evals` are hard-coded exemptions via `constants.*JobName`; adding a future built-in job that legitimately runs an engine requires editing this list.
- An extra `yaml.Unmarshal` of the compiled output is performed when a recognized engine-secret expression appears in the YAML, even when schema validation is skipped; ordinary engine names do not trigger parsing.
- The behaviour is documented in `docs/src/content/docs/reference/steps-jobs.md`, making the restriction part of the public contract for custom steps and jobs.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
