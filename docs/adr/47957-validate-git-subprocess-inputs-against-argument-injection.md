# ADR-47957: Validate Git Subprocess Inputs Against Argument Injection (CWE-88)

**Date**: 2026-07-25
**Status**: Draft
**Deciders**: Unknown (security fix by copilot-swe-agent, see VULN-001)

---

### Context

The remote workflow import feature resolves `owner/repo/path@ref` workflowspec strings and fetches the referenced files by invoking git subprocesses (`git archive`, `git ls-remote`, `git clone`, `git checkout`). The `ref` and `path` components of these specs are supplied by developers in workflow configuration and are user-controlled at the workflowspec level. Prior to this change, both values were passed directly as positional arguments to those subprocesses with no sanitisation and no `--` end-of-options separator. A `ref` value of `--upload-pack=malicious` or a `path` value of `--output=/etc/cron.d/pwned` would be parsed by git as option flags rather than values (CWE-88, argument injection). The primary attack surface was the auth-error fallback path, which is reached without requiring a valid token on token-less or unauthorised executions.

### Decision

We will apply a two-layer defence against git argument injection for all user-supplied `ref` and `path` values passed to git subprocesses:

1. **Centralised input validation** — two new shared guards, `gitutil.ValidateGitRef` and `gitutil.ValidateGitPath`, are called at the earliest possible points (workflowspec parse time and at each subprocess call site) and return an error for empty values, values starting with `-`, and refs containing `..`.
2. **`--` end-of-options separators** — all git subprocess invocations that accept a ref or path as a positional argument are updated to insert `--` before those arguments, ensuring that even if validation were bypassed, git itself would not interpret them as flags (defence-in-depth per the git(1) specification).

### Alternatives Considered

#### Alternative 1: `--` separator only, no validation guards

Add `--` end-of-options separators to every git call without introducing explicit validation functions. This is the minimal fix: it stops git from interpreting leading-`-` values as flags at the subprocess level. It was not chosen because it provides no early-fail signal — an attacker-controlled ref would still reach the subprocess and produce a confusing git error rather than a clear security rejection. It also gives no protection against `..`-based git object traversal expressions, which are not mitigated by `--`. Centralised validation makes the security invariant visible, testable, and reusable for future subprocess additions.

#### Alternative 2: Allowlist-based validation only (no `--` separator)

Reject any ref or path that does not match an explicit allowlist pattern (e.g., alphanumeric characters, slashes, dots, hyphens, underscores). This would be stricter and would block a wider class of unexpected inputs. It was not chosen because an allowlist tight enough to be safe is also tight enough to break legitimate edge-case refs (e.g., refs with `@`, unicode characters, or non-standard tag formats used in real repositories). The denylist approach (`ValidateGitRef`/`ValidateGitPath`) blocks the specific injection vectors (leading `-`, `..`) without rejecting valid refs. Using `--` as defence-in-depth alongside the denylist gives comparable protection without the breakage risk of an allowlist.

### Consequences

#### Positive
- Closes the CWE-88 argument injection attack vector across all git fallback paths (`git archive`, `git ls-remote`, `git clone`, `git checkout`) for both ref and path inputs.
- `ValidateGitRef` and `ValidateGitPath` are centralised in `pkg/gitutil` and reusable for any future git subprocess additions, ensuring the security invariant is easy to apply consistently.
- Unit tests covering valid inputs, empty values, leading-`-` injection, and `..` traversal cases provide a regression safety net.

#### Negative
- Legitimate refs or paths that start with `-` (an unusual but theoretically valid git ref format) or contain `..` (e.g., range expressions used in some tooling) will now be rejected. In practice these are not expected in normal workflow import usage, but the restriction is a breaking change for any consumer relying on such values.
- Validation is applied redundantly at multiple layers (parse time and again at each subprocess call site), which adds some code repetition and means a failed import may report the rejection at the subprocess call rather than at parse time if the parse-time guard is bypassed by future code paths.

#### Neutral
- The `--` separator changes git command signatures (e.g., `git ls-remote <url> -- <ref>` instead of `git ls-remote <url> <ref>`). This is semantically equivalent for all supported git versions and should have no observable behaviour change for valid inputs.
- The `#nosec G204` annotation on the `git archive` call was retained; its justification (exec.CommandContext with separate args, not shell execution) remains accurate, and the new validation further strengthens the rationale.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
