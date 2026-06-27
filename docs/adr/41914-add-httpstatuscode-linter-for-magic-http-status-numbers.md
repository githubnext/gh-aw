# ADR-41914: Add `httpstatuscode` Linter to Enforce Named HTTP Status Constants

**Date**: 2026-06-27
**Status**: Draft
**Deciders**: pelikhan, linter-miner automation

---

### Context

The codebase contains Go source files that compare HTTP status codes using raw integer literals (e.g., `200`, `403`, `404`) rather than the named constants from Go's `net/http` package (e.g., `http.StatusOK`, `http.StatusForbidden`). A linter-miner scan (run #51, 2026-06-27) identified 10 real violations in production non-test files, concentrated in `pkg/cli/firewall_log.go` and `pkg/cli/firewall_policy.go`. No existing linter in the suite covers this pattern. Using magic numbers reduces readability and makes the codebase harder to search and audit.

### Decision

We will add a custom `go/analysis` linter, `httpstatuscode`, that reports integer literals in the HTTP status code range (100–599) used in `==` or `!=` comparisons against variables named `status` or `statusCode`, or against selector fields named `StatusCode`. The linter will suggest the corresponding `http.Status*` named constant and will be registered in the multichecker alongside existing analyzers. Violations in test files are excluded; `//nolint:httpstatuscode` suppresses individual occurrences.

### Alternatives Considered

#### Alternative 1: Style Guide + Manual Code Review

Document the convention in the coding style guide and rely on reviewers to catch violations. This requires no tooling investment but provides no automated enforcement — violations slip through during high-velocity code review cycles and must be caught case-by-case. Given that 10 violations already exist undetected in production code, reviewer vigilance alone has proven insufficient.

#### Alternative 2: Configure an Existing General-Purpose Linter (e.g., `revive`, `staticcheck`)

Some general-purpose linters support custom rules or rule plug-ins. However, none of the linters already in the suite expose a rule that narrows to the `status`/`statusCode`/`.StatusCode` naming context. A generic "no magic numbers" rule would generate excessive false positives across the codebase (e.g., build numbers, port numbers, exit codes), requiring significant suppression overhead and offering no targeted diagnostic message.

#### Alternative 3: AST-based Sed/Grep Pre-commit Hook

A pre-commit hook using regex over raw source text could flag bare integer literals near HTTP-related variable names. This approach is fragile (regex cannot accurately parse Go AST), cannot provide file/line-level diagnostics integrated with IDE tooling, and does not compose with the existing `go/analysis` multichecker pipeline.

### Consequences

#### Positive
- Readability improves: `http.StatusForbidden` conveys intent; `403` does not.
- Grep-ability improves: searching by constant name finds all uses reliably; searching for `403` produces false positives.
- Automated enforcement eliminates reliance on reviewer memory and catches regressions at CI time.
- Narrow scope (only `status`, `statusCode`, `.StatusCode` contexts) keeps false-positive rate near zero.

#### Negative
- The `httpStatusNames` map must be manually updated if Go's `net/http` package adds new status codes in future stdlib releases.
- Existing 10 violations in production code must be remediated before the linter can run in blocking mode; until then, they represent technical debt.
- Each new custom analyzer adds surface area to the internal linter suite, increasing maintenance burden for the linters team.

#### Neutral
- The linter applies only to `==` and `!=` binary expressions; other comparison patterns (e.g., `>`, `<`) and switch cases are not in scope.
- Test files are explicitly excluded, consistent with the existing linter convention in this repository.
- The `//nolint:httpstatuscode` escape hatch follows the established nolint pattern already used across the suite.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
