# ADR-47957: Validate Git Subprocess Inputs Against Argument Injection (CWE-88)

**Date**: 2026-07-25
**Status**: Accepted
**Deciders**: GitHub Agentic Workflows maintainers

---

### Context

The remote workflow import feature resolves `owner/repo/path@ref` workflowspec strings and fetches the referenced files by invoking git subprocesses (`git archive`, `git ls-remote`, `git clone`, `git checkout`). The `ref` and `path` components of these specs are supplied by developers in workflow configuration and are user-controlled at the workflowspec level. Prior to this change, both values were passed directly as positional arguments to those subprocesses with no sanitisation and no `--` end-of-options separator. A `ref` value of `--upload-pack=malicious` or a `path` value of `--output=/etc/cron.d/pwned` would be parsed by git as option flags rather than values (CWE-88, argument injection). The primary attack surface was the auth-error fallback path, which is reached without requiring a valid token on token-less or unauthorised executions.

### Decision

We will apply targeted validation at parse time and subprocess boundaries, and use git-specific argument forms that preserve command semantics while preventing option injection:

1. **Centralised input validation** — shared guards `gitutil.ValidateGitRef` and `gitutil.ValidateGitPath` are called at the earliest possible points (workflowspec parse time and at each subprocess call site). `ValidateGitRef` rejects empty refs, leading `-`, NUL bytes, and `..` traversal expressions. `ValidateGitPath` rejects empty paths, leading `-`, absolute paths, and `..` path traversal after normalization.
2. **Use only git argument separators that preserve semantics** — `git archive` keeps `--` before the pathspec, because that command explicitly separates `<tree-ish>` from `<path>`. `git checkout --detach <sha>` is used for full-SHA clone fallbacks so the validated SHA remains a revision argument rather than becoming a pathspec. `git ls-remote` does not receive an extra `--`, because its interface does not define a remote/refs separator there.
3. **Propagate invalid remote-origin refs as errors** — remote-origin parsing now returns an error for unsafe workflowspec refs instead of silently dropping the origin, so callers can distinguish security rejections from non-workflowspec inputs.

### Alternatives Considered

#### Alternative 1: `--` separator only, no validation guards

Add `--` end-of-options separators to every git call without introducing explicit validation functions. This is the minimal fix for commands that define such a separator, but it was not chosen because it provides no early-fail signal and does not apply uniformly across git commands. `git ls-remote` has no supported separator in this position, and `git checkout -- <sha>` changes semantics entirely. Centralised validation makes the invariant visible, testable, and reusable.

#### Alternative 2: Allowlist-based validation only (no command-specific argument hardening)

Reject any ref or path that does not match an explicit allowlist pattern (e.g., alphanumeric characters, slashes, dots, hyphens, underscores). This would be stricter and would block a wider class of unexpected inputs. It was not chosen because an allowlist tight enough to be safe is also tight enough to break legitimate edge-case refs (for example refs with `@`, unicode characters, or non-standard tag formats used in real repositories). The chosen validation blocks the concrete dangerous forms while preserving valid git syntax, and the command-specific git argument changes provide the remaining defence-in-depth where supported.

### Consequences

#### Positive
- Closes the CWE-88 argument injection attack vector across all git fallback paths (`git archive`, `git ls-remote`, `git clone`, `git checkout`) for both ref and path inputs without changing the semantics of SHA checkout or ref resolution.
- `ValidateGitRef` and `ValidateGitPath` are centralised in `pkg/gitutil` and reusable for any future git subprocess additions, ensuring the security invariant is easy to apply consistently.
- Unit tests covering valid inputs, empty values, leading-`-` injection, NUL bytes, absolute paths, and traversal cases provide a regression safety net.
- Invalid remote-origin refs now surface as explicit errors to callers instead of being visible only through debug logging.

#### Negative
- Legitimate refs or paths that start with `-` (an unusual but theoretically valid git ref format), contain NUL bytes, or resolve to absolute/traversing paths will now be rejected. In practice these are not expected in normal workflow import usage, but the restriction is a breaking change for any consumer relying on such values.
- Validation is applied redundantly at multiple layers (parse time and again at each subprocess call site), which adds some code repetition in exchange for defence-in-depth.

#### Neutral
- `git archive` retains a `--` pathspec separator, while other commands rely on validation plus git-native argument forms instead of a one-size-fits-all separator rule.
- The `#nosec G204` annotation on the `git archive` call was retained; its justification (exec.CommandContext with separate args, not shell execution) remains accurate, and the new validation further strengthens the rationale.

---

*Finalized from the draft generated by the adr-writer agent to match the merged implementation in this PR.*
