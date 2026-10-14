# ADR-68579: Detect agentic engine execution outside the agent job via compiled-YAML static analysis

**Date**: 2026-10-14
**Status**: Draft
**Deciders**: Unknown (PR #68579, authored by `copilot-swe-agent`)

---

### Context

In `gh-aw`, only the generated `agent` job runs the agentic engine inside the firewall/sandbox configuration derived from `engine:`. Custom steps and custom jobs — including safe-output jobs and imported steps — run **outside** that sandbox with ordinary GitHub Actions security. Nothing previously stopped a workflow author from installing and invoking an engine CLI (Copilot CLI, Claude Code, Codex, Gemini CLI, Agy, Pi) from one of those jobs, which silently bypasses the sandbox, the engine firewall and the MCP/tooling policy that `engine:` is supposed to enforce. The compiler needs a way to catch this at compile time, before a `.lock.yml` is emitted, and it must work even when schema validation is skipped and even for steps that the user never wrote directly (imported or generated).

### Decision

We will add a compiler validation pass, `validateAgenticEngineSteps`, that performs **static pattern analysis over the final compiled workflow YAML** and rejects recognizable agentic engine installations and invocations in every job except `agent`, `detection`, and `evals`. Violations are a hard error in strict mode (no lock file is written) and a compiler warning when `strict: false`. The check is gated by a cheap substring pre-filter (`hasAgenticEngineReference`) so the extra `yaml.Unmarshal` is only paid when an engine name appears in the output. Detection is deliberately *command-position aware*: a hand-written masking pass removes comments, single/double-quoted literals and quoted (data) heredocs before the regexes run, and it recurses into `bash -c '...'` scripts and `$(...)`/backtick substitutions, so that documentation-like text and `command -v copilot` existence checks do not trip the gate.

### Alternatives Considered

#### Alternative 1: Validate the frontmatter/markdown source instead of the compiled YAML

Checking the user-authored workflow markdown would be simpler and would give diagnostics pointing at the exact source line. It was rejected because it cannot see steps contributed by `imports:`, by safe-output job generation, or by built-in job augmentation (e.g. `activation`, `conclusion`) — exactly the places where an injected engine invocation would be hardest to notice. Operating on the final `jobs:` map guarantees that whatever ships in the lock file has been inspected.

#### Alternative 2: Enforce at runtime instead of compile time (sandbox/firewall deny-list)

The engine binaries could be blocked at execution time, e.g. by network policy or by a runner-side guard that fails the step. This gives true enforcement rather than best-effort detection, but it only fails *after* the workflow has been merged and dispatched, costs a run, and cannot cover every install path (vendored binaries, mirrors). Compile-time rejection gives authors immediate, local feedback and keeps the invalid workflow out of the repository; a runtime control could still be added later as defence in depth.

#### Alternative 3: Parse the `run:` script with a real shell parser (e.g. `mvdan/sh`)

A proper shell AST would make command-position detection exact and would remove the need for the bespoke masking/regex machinery in `agentic_engine_steps_validation.go`. It was not chosen in this PR, presumably to avoid a new dependency and to keep the check tolerant of the non-POSIX fragments that appear in Actions `run:` blocks (PowerShell, `&`-prefixed Windows paths). The cost is that correctness now rests on a large, hand-tuned regex/masking layer — which is why the change ships with ~180 lines of table-driven positive/negative tests. *[TODO: verify whether a shell parser was explicitly evaluated and rejected.]*

### Consequences

#### Positive
- The sandbox-bypass path is closed by default: a workflow that installs or runs an engine CLI outside the `agent` job fails to compile in strict mode and produces no lock file.
- Coverage extends to imported steps, safe-output steps and generated job augmentations, not just hand-written custom jobs, because the check runs on the final compiled `jobs:` map.
- Diagnostics name the offending job and step index and point authors at `engine:`, so the remediation is explicit rather than a bare rejection.
- The `strict: false` escape hatch degrades to a warning, so existing workflows are not hard-broken by the new rule.

#### Negative
- Detection is heuristic. The masking + regex approach will inevitably drift: new engines, new install channels (vendored binaries, private registries, container images) and unusual shells are not covered, so this is a speed bump rather than an enforcement boundary.
- False positives are possible for legitimate scripts that happen to sit in a command position (`pi`, `agy` and `codex` are short, generic names), and the only remedy today is `strict: false`, which disables strictness workflow-wide rather than per-step.
- The validation code is intricate (manual quote/heredoc/substitution state machine) and carries a real maintenance burden; a subtle bug there either silently misses bypasses or blocks valid workflows.
- Users can still evade the check trivially (e.g. `C=cop; "${C}ilot" -p ...`, or invoking the engine from a committed script), so it must not be relied on as a security control.

#### Neutral
- `agent`, `detection` and `evals` are hard-coded exemptions via `constants.*JobName`; adding a future built-in job that legitimately runs an engine requires editing this list.
- An extra `yaml.Unmarshal` of the compiled output is now performed when an engine name appears anywhere in the YAML, even when schema validation is skipped; the substring pre-filter keeps the common case free.
- The behaviour is documented in `docs/src/content/docs/reference/steps-jobs.md`, making the restriction part of the public contract for custom steps and jobs.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
