# ADR-69813: In-Process AST Lint Rule Registry with Secret-Leak Taint Analysis

**Date**: 2026-05-14
**Status**: Draft
**Deciders**: pelikhan, Unknown (review required)

---

### Context

Agentic workflows can expose secrets in several ways. Values can flow through GitHub Actions expressions, `env` and step/job outputs, Bash/sh `run` scripts, and the AW prompt sent to the agent. Before this PR, `gh aw lint` ran only the external actionlint on compiled `.lock.yml` files. actionlint does not trace data flow from secret sources to disclosure sinks such as logs, step summaries, outputs, artifacts, network calls, or prompts. One example is the Gemini model inventory workflow, which put an API key in a URL. The repository already has a typed Actions workflow AST (`pkg/workflow/actions`, see ADR-69755) and an AW document parser (`pkg/workflow/aw`). These give a structured base for new checks. More security and structural checks are expected later, so the first checker should not be hard-wired into the CLI.

### Decision

We will add an in-process, extensible lint framework in `pkg/workflow/lint`. Rules register with a kebab-case ID, a description, a default severity, and a `Check(context.Context, Input)` callback. Each callback receives the Actions AST and an optional AW document, and returns deterministic, value-free diagnostics. Alongside it, we will build a reusable source/sink/filter taint engine in `pkg/workflow/taint`. The engine tracks symbolic provenance through expressions, environment and output propagation, and Bash/sh scripts (parsed with the newly added `mvdan.cc/sh/v3` dependency). It never evaluates expressions or retrieves secret values. The first built-in rule, `secret-leak`, sits on top of this engine. By default, `gh aw lint` runs all registered AST rules alongside actionlint. Both run even if one fails, and a failure in either one makes the command exit with a nonzero status. `--rules`, `--list-rules`, and `--json` let users select rules, list them, and get structured output.

### Alternatives Considered

#### Alternative 1: Extend actionlint or rely on external scanners (zizmor/poutine)

Custom checks could be added through actionlint's rule hooks, or the existing external security scanners could handle the problem. This option was considered because the tools are already part of the compile pipeline. It was rejected for three reasons: these tools do not know about AW prompts or companion `.md` sources; gh-aw cannot control their taint models; and every new check would depend on external release cycles. An in-process registry lets gh-aw define its own source/sink semantics for agentic workflows.

#### Alternative 2: A single hard-coded secret-leak checker without a rule registry

A standalone secret-leak pass invoked directly from `gh aw lint` would have meant less code up front. It was rejected because the PR explicitly expects more rules later. A registry gives every future checker the same validation, selection, deterministic sorting/deduplication, failure isolation, and CLI integration. Without one, each new check would need its own wiring.

#### Alternative 3: Regex/pattern matching on lock-file text instead of AST taint analysis

Grepping for `secrets.*` near `echo`, `curl`, and similar commands is simpler. It cannot follow values through `env`, step outputs, or shell variable assignments, so it would produce both false positives and false negatives. It could also pull secret-bearing text into diagnostics. The AST-based approach keeps diagnostics free of values and records symbolic provenance.

### Consequences

#### Positive
- New structural and data-flow checks can be added by registering a `lint.Rule` in `lint.Default()`. Taint-based rules can reuse `taint.Rule` (`IsSource`/`IsSink`/`IsFilter`/`ActionSources`).
- Diagnostics are deterministic and contain no sensitive values, which makes them safe to print, export as JSON, and use in CI. A failing rule does not stop the other rules from reporting.
- A regression test applies all rules to the 344 compiled workflows in the repository. The new analysis has already caught and fixed a real exposure (Gemini API key in a URL, now sent in an auth header).

#### Negative
- The PR adds a new third-party dependency, `mvdan.cc/sh/v3` (BSD-3-Clause), and about 1,100 lines of new taint/shell analysis code to maintain.
- Coverage is partial. Unmodeled external commands, unsupported shells, external action internals, filesystem operations, and dynamic code are reported only as coverage warnings (364 in the current repository scan). A clean result is not proof that a workflow is safe.
- `gh aw lint` now does more by default. It can fail on new AST rule errors even when actionlint passes, which changes behavior for existing users.

#### Neutral
- Explicit `--rules` selection runs only the AST rules and skips actionlint. Compilation (`gh aw compile`) keeps its existing behavior and does not run the new rules.
- Authentication arguments are treated as intentional credential use. Endpoint trust is not inferred, so the `secret-leak` rule may need policy changes as usage patterns evolve.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
