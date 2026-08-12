# ADR-52180: Structured Validation Error Messages with Three-Part Format

**Date**: 2026-08-12
**Status**: Draft
**Deciders**: pelikhan

---

### Context

The `errormessage` linter (defined in `.github/skills/error-messages/SKILL.md`) enforces that validation error messages follow a three-part structure: `[what's wrong]. [what's expected]. [example of correct usage]`. An audit flagged five files in `pkg/workflow` as non-compliant (13–48% compliance): `checkout_config_parser.go`, `safe_outputs_data_schema.go`, `stop_after.go`, `model_identifier.go`, and `engine_driver_validation.go`. These files produced bare negative phrases (`"invalid checkout configuration"`, `"must be a boolean"`) with no guidance on what correct input looks like, leaving users unable to self-correct from the error message alone.

The codebase already has two error-producing conventions in this package: the `NewValidationError(field, value, reason, suggestion)` structured type used in `*_validation.go` files, and inline `fmt.Errorf`/`errors.New` calls everywhere else.

### Decision

We will bring all error messages in the five flagged files into linter compliance using a two-tier approach:
- In `*_validation.go` files, convert `fmt.Errorf` calls to `NewValidationError(field, value, reason, suggestion)`, which enforces the structured format at the type level.
- In all other files, extend each `fmt.Errorf`/`errors.New` call inline to append `. Expected …. Example:\n<yaml-snippet>` text.

No changes to control flow, validation logic, or error types — only message text is updated.

### Alternatives Considered

#### Alternative 1: Loosen or suppress the linter rule for these files

The `errormessage` linter could be configured to exempt the flagged files, silencing the CI failure without changing any messages. This was rejected because the compliance gap is real: users encountering a validation error with no actionable guidance must consult documentation or source code. Suppressing the rule trades short-term velocity for long-term UX debt.

#### Alternative 2: Unify all validation errors under `NewValidationError`

All five files could be converted to use `NewValidationError` regardless of file type, giving a uniform structured API across the package. This was rejected because (a) `checkout_config_parser.go`, `safe_outputs_data_schema.go`, `stop_after.go`, and `model_identifier.go` are not `*_validation.go` files and the convention is deliberately file-type-scoped, (b) wrapping `errors.New` calls in the structured type would require converting all call sites across these files, and (c) the inline text approach achieves identical user-visible output with surgical diffs that are easy to review.

### Consequences

#### Positive
- Users receive self-contained error messages that state the problem, describe the expected input format, and show a concrete YAML/identifier example — enabling self-correction without consulting docs.
- All five flagged files reach 100% linter compliance, unblocking the CI check introduced in the `errormessage` workflow.

#### Negative
- Error message strings grow significantly (roughly doubling in length for most messages), increasing log and CLI output verbosity when validation fails.
- Existing test assertions that matched against fixed substrings (e.g., `"safe basename"`, `"unsupported extension"`, `"empty path segments"`) must be verified or updated when the surrounding message text changes; the PR preserves these substrings in the `reason` field of `NewValidationError`, but future refactors must keep this in mind.

#### Neutral
- The two-tier convention (`NewValidationError` for `*_validation.go`, inline text elsewhere) becomes implicitly established by this PR. Future validation files should follow the same file-type-scoped rule.
- The `stop_after.go` and other non-validation files remain on `fmt.Errorf`/`errors.New` — they are not migrated to the structured type, which is consistent with the existing convention but leaves the package with two error-production patterns.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
