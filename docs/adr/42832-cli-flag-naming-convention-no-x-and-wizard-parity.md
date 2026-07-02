# ADR-42832: Adopt `--no-X` Flag Convention and Enforce `add-wizard` Flag Parity

**Date**: 2026-07-02
**Status**: Draft
**Deciders**: pelikhan (automated inspection via copilot-swe-agent)

---

### Context

The `gh-aw` CLI had grown inconsistently: most boolean negation flags followed a `--no-X` prefix (e.g., `--no-fix`, `--no-compile`, `--no-actions`), but one flag — `--disable-codemod` — used a `--disable-X` prefix introduced at a different time. This made the CLI surface feel non-uniform to users who relied on tab-completion and `--help` output.

Separately, the `add-wizard` command shared the same underlying workflow installation logic as `add` and `deploy`, but its `AddInteractiveConfig` struct was missing the `Name`, `Force`, `AppendText`, and `DisableSecurityScanner` fields. Those fields were hardcoded to zero values when `add-wizard` called into `AddResolvedWorkflows`, silently ignoring flags a user might expect to work by analogy with `add`/`deploy`.

An automated CLI consistency inspection across 29 files identified these as the two highest-severity issues in the codebase's CLI surface.

### Decision

We will rename `--disable-codemod` to `--no-codemod` in the `fix` and `upgrade` commands, matching the `--no-X` convention used everywhere else. The old flag name is not preserved as an alias (a clean break). We will also expand `AddInteractiveConfig` to include `Name`, `Force`, `AppendText`, and `DisableSecurityScanner`, register those flags on `add-wizard`, and propagate them through to `AddResolvedWorkflows` so the interactive wizard has full parity with the direct `add`/`deploy` commands.

### Alternatives Considered

#### Alternative 1: Keep `--disable-codemod` and add `--no-codemod` as an alias

Both flag names would be accepted. The inconsistency would remain visible in `--help` output and completion lists, but existing scripts using `--disable-codemod` would not break. Rejected because maintaining two names for the same flag adds ongoing documentation burden and extends the period of inconsistency indefinitely.

#### Alternative 2: Keep `--disable-codemod` unchanged and only fix documentation

Leave the flag name as-is and update only prose that mentioned it. This eliminates the breakage risk but permanently encodes the exception into the CLI contract. Rejected because the `--disable-X` outlier will confuse future contributors adding new flags, who will need to decide which convention to follow.

#### Alternative 3: Extend `AddInteractiveConfig` but not register the new flags on `add-wizard`

Propagate the struct fields but do not expose the flags at the CLI layer, deferring the UX decision. Rejected because half-wired flags (struct fields with no CLI entry point) are worse than no fields at all — they create dead code and false impressions that the feature is accessible.

### Consequences

#### Positive
- The CLI flag surface is now internally consistent: all boolean negation flags use `--no-X`.
- `add-wizard` users can now pass `--name`, `--force`, `--append`, and `--no-security-scanner` with the same semantics as `add`/`deploy`.
- The `AddInteractiveConfig` struct accurately reflects the full set of options the wizard supports, making the code easier to audit and extend.

#### Negative
- Renaming `--disable-codemod` to `--no-codemod` is a breaking change for any scripts or CI configurations that used the old flag name. No deprecation alias was added.
- `add-wizard` now accepts more flags, which increases its test surface and the risk of future divergence if `add`/`deploy` flags are added without a corresponding update to `add-wizard`.

#### Neutral
- Downstream documentation (docs/src/content/docs/setup/cli.md) was updated in the same PR to reflect the renamed flag and the corrected `--push` behaviour description.
- The `disable-security-scanner` legacy alias on `add`, `deploy`, and `trial` is preserved as a hidden flag; this PR did not extend that pattern to `add-wizard`'s new `--no-security-scanner` flag.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
