# ADR-43302: Section-Aware Secret Validation Messages in Workflow Compiler

**Date**: 2026-07-04
**Status**: Draft
**Deciders**: Unknown

---

### Context

The workflow compiler's `validateEnvSecretsSection` function validates secrets across multiple
env sections (`env`, `engine.env`) and emits warnings or errors when `${{ secrets.* }}`
references are detected. Historically it used a single generic message — "will be leaked to the
agent container" — regardless of which section contained the secret reference.

However, `engine.env` secrets are handled differently from top-level `env` secrets: the
`ComputeAWFExcludeEnvVarNames` function auto-detects any `engine.env` value that contains a
`${{ secrets.* }}` reference and adds the corresponding variable name to AWF's `--exclude-env`
list. This means `engine.env` secrets never reach the agent sandbox process, making the
"will be leaked" warning factually incorrect for that section and a source of user confusion.

### Decision

We will differentiate the validation messages emitted by `validateEnvSecretsSection` based on
which section is being validated, rather than using a single generic message for all sections.
For `engine.env`, messages will accurately reflect that secrets are auto-excluded from the
agent sandbox via AWF `--exclude-env`; for all other sections (e.g., top-level `env`), the
existing "will be leaked" language is retained because those secrets genuinely reach the
agent container. Strict mode continues to treat `engine.env` secrets as an error (encouraging
engine-specific secret configuration), but with accurate, non-misleading error text.

### Alternatives Considered

#### Alternative 1: Remove the `engine.env` warning entirely

Since `engine.env` secrets are auto-excluded and never reach the agent process, one option is
to suppress the warning entirely for that section — no message, no error. This eliminates the
false positive and simplifies the code path. It was rejected because users should still be
nudged toward engine-specific secret configuration (the more secure, explicit pattern), and
removing all feedback would silently permit a configuration style that the platform discourages.

#### Alternative 2: Extract a separate validation function for `engine.env`

Rather than branching inside `validateEnvSecretsSection`, a dedicated
`validateEngineEnvSecretsSection` function could own all `engine.env` secret logic. This would
keep each function single-purpose and avoid section-name string comparisons. It was rejected
for this fix because it would duplicate the secret-detection regex and collection logic that
lives in `validateEnvSecretsSection`, increasing maintenance surface for a change whose primary
goal is correcting a misleading string — not restructuring validation logic.

### Consequences

#### Positive
- Users see accurate feedback: `engine.env` secret warnings now correctly describe sandbox
  exclusion rather than falsely claiming leakage to the agent container.
- Strict mode error messages for `engine.env` remain actionable and no longer contradict the
  platform's actual security behavior, reducing support burden and user confusion.

#### Negative
- `validateEnvSecretsSection` now contains section-specific branching (`if sectionName == "engine.env"`),
  slightly increasing cyclomatic complexity and making the function less generic.
- Any future env section added to the validation path must be evaluated for whether it also
  requires a custom message, creating an implicit maintenance contract.

#### Neutral
- Test assertions for `engine.env` strict-mode cases were updated to match the new message
  text; this is a mechanical change with no behavioral impact on test coverage.
- The `awf_helpers_test.go` additions verify the existing `ComputeAWFExcludeEnvVarNames`
  behavior that this fix depends on for correctness, improving confidence in the invariant.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
