# ADR-25837: Prevent Template Injection by Moving `${{ }}` Expressions to `env:` Blocks

**Date**: 2026-04-11
**Status**: Draft
**Deciders**: Unknown (automated Copilot PR)

---

## Part 1 — Narrative (Human-Friendly)

### Context

The static analysis tool `zizmor` flagged 16 compiled GitHub Actions workflows where `${{ }}` expression syntax appeared directly inside `run:` blocks (typically inside heredocs used to write JSON config files). When GitHub Actions evaluates a workflow, it substitutes `${{ expr }}` expressions *before* the shell sees the `run:` block content. If the expression contains user-controlled input (e.g., PR titles, branch names, step outputs derived from issue bodies), an attacker can break out of the heredoc string and inject arbitrary shell commands. The compiler was generating these unsafe patterns from three distinct code paths: the Safe Outputs config generator, the guard policy builder, and the OTEL endpoint renderer.

### Decision

We will move all `${{ }}` expressions emitted inside `run:` blocks into the step's `env:` block and reference them using shell variable syntax (`${VAR_NAME}`) in the heredoc body. This is the canonical GitHub Actions hardening pattern: GitHub Actions substitutes expressions in `env:` values but the shell receives the result as an environment variable, treating it as data rather than code. We also generalize the existing `ExtractSecretsFromValue` function to `ExtractAllExpressionsFromValue` so that *any* expression type (github context, step outputs, secrets) is uniformly extracted and bound to a deterministically named env var.

### Alternatives Considered

#### Alternative 1: Quote/Escape Expressions at the Point of Use

Wrap every `${{ }}` expression with `${{ toJSON(expr) }}` or similar encoding so the substituted value cannot contain shell metacharacters. This was not chosen because the approach must be applied consistently to each call site individually, is easy to miss during future maintenance, and requires runtime JSON-encoding even when the value is not a JSON structure (e.g., a plain endpoint URL).

#### Alternative 2: Use `github-script` or Dedicated Steps Instead of Heredocs

Replace the heredoc-based config generation with a `github-script` step that constructs the JSON object programmatically, where expression values are bound as JS variables rather than interpolated into strings. This was not chosen because it would require a larger refactor of the compiler's config-writing logic, introduce a Node.js dependency into steps that currently have none, and add latency to workflow execution.

#### Alternative 3: Single-quoted Heredoc (Prevent Shell Expansion Entirely)

Use a single-quoted heredoc delimiter (`<< 'EOF'`) so the shell does not expand `${VAR}` references at all, and rely solely on GitHub Actions pre-substitution of `${{ }}`. This was not chosen because the guard policy and OTEL paths need the shell to expand `${VAR}` references that were *already* moved to `env:` in earlier hardening work; single-quoting would break that existing indirection.

### Consequences

#### Positive
- Eliminates all 16 zizmor `template-injection` findings in a systematic, compiler-level fix rather than per-file patches.
- Establishes a uniform env-var naming convention (`GH_AW_GITHUB_*`, `GH_AW_STEP_*`, `GH_AW_GUARD_*`) for compiler-generated env vars, making generated workflows easier to audit.
- `ExtractAllExpressionsFromValue` is reusable for any future config path that needs the same pattern.

#### Negative
- Generated workflows now include more `env:` entries per step, increasing YAML verbosity and slightly raising the character count of compiled workflow files.
- The guard env vars (`GH_AW_GUARD_BLOCKED_USERS`, etc.) must be explicitly excluded from Docker container env passthrough; this is handled by the `addedEnvVars` allowlist but represents a new invariant that future contributors must maintain.
- The regex `guardExprRE` now matches both `${{ ... }}` and `${VAR}` forms; if a future expression style is added, the regex must be extended again.

#### Neutral
- All 187 existing compiled workflows are regenerated; the only semantic change is the env-var indirection for the affected 16 workflows.
- Tests must cover the new `ExtractAllExpressionsFromValue` naming rules for github-context, step-output, and general expression categories.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### Expression Handling in Compiled Workflows

1. The compiler **MUST NOT** emit `${{ }}` GitHub Actions expressions directly inside the body of a `run:` block (including heredoc bodies and inline shell commands).
2. Any `${{ }}` expression that must appear in a `run:` block value **MUST** be lifted to the step's `env:` block and referenced as `${ENV_VAR_NAME}` inside the `run:` block.
3. The compiler **MUST** use `ExtractAllExpressionsFromValue` (or a functionally equivalent routine) to identify and bind all `${{ }}` expressions before writing a `run:` block.

### Environment Variable Naming Convention

1. Compiler-generated env vars for `${{ secrets.X }}` expressions **MUST** use the secret name `X` directly as the env var name (e.g., `DD_API_KEY`).
2. Compiler-generated env vars for `${{ github.X }}` expressions **MUST** use the form `GH_AW_GITHUB_<SANITIZED_PATH>` (e.g., `GH_AW_GITHUB_WORKFLOW`).
3. Compiler-generated env vars for `${{ steps.X.outputs.Y }}` expressions **MUST** use the form `GH_AW_STEP_<SANITIZED_STEP_ID>_<SANITIZED_OUTPUT>`.
4. Compiler-generated env vars for all other expressions **MUST** use the form `GH_AW_<SANITIZED_EXPRESSION>`.
5. Sanitized names **MUST** be uppercased and contain only `[A-Z0-9_]` characters; any other character **MUST** be replaced with `_`.

### Guard Expression Rendering

1. The `guardExprRE` regular expression **MUST** match both `${{ ... }}` and `${VAR_NAME}` sentinel-prefixed forms so that existing and future guard policy expressions are correctly un-quoted during rendering.
2. Guard policy env vars (`GH_AW_GUARD_BLOCKED_USERS`, `GH_AW_GUARD_TRUSTED_USERS`, `GH_AW_GUARD_APPROVAL_LABELS`) **MUST** be set in the step's `env:` block and **MUST NOT** be forwarded into Docker container environments via the MCP env passthrough mechanism.

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. In particular: no `${{ }}` expression **MUST** appear in any compiled `run:` block body, all compiler-generated env var names **MUST** follow the naming convention in the "Environment Variable Naming Convention" section, and guard env vars **MUST NOT** be passed through to Docker containers. Failure to meet any **MUST** or **MUST NOT** requirement constitutes non-conformance.

---

*This is a DRAFT ADR generated by the [Design Decision Gate](https://github.com/github/gh-aw/actions/runs/24291612020) workflow. The PR author must review, complete, and finalize this document before the PR can merge.*
