# ADR-70356: Encode Historical zizmor Findings as In-Process AST Lint Rules

**Date**: 2026-10-20
**Status**: Draft
**Deciders**: Not recorded (PR #70356 author: pelikhan)

---

### Context

Credit for the upstream audit definitions, names, security research, and remediation guidance belongs to the [zizmor maintainers and contributors](https://github.com/zizmorcore/zizmor). zizmor is [MIT-licensed](https://github.com/zizmorcore/zizmor/blob/main/LICENSE); its [audit documentation](https://docs.zizmor.sh/audits/) describes the original checks. The repository fixes below provide gh-aw regression evidence, not the origin of those audits. This decision introduces independent Go implementations of selected checks, not embedded zizmor code, equivalent audit coverage, or upstream endorsement.

The repository has fixed the same kinds of zizmor findings several times. These include persisted checkout credentials (#2991, #14449, #25873), dependency-cache restoration in release workflows (#5214, #5872, #19865), workflow-wide write permissions (#5229), literal-only obfuscation expressions (#4398, #14257), and template injection into executable code. Each time, an external scanner found the problem. The in-process Actions/AW linter (`pkg/workflow/lint`) could not catch a regression of these patterns. The linter already has a default rule registry (`Default()` in `lint.go`) with rules such as `action-pin`, `template-injection`, and `permission-scope`. It runs over both typed/raw Actions ASTs and AW documents and must also build for WASM (`GOOS=js GOARCH=wasm`). Because of that WASM constraint, a new check cannot depend on optional native parsers or network lookups.

### Decision

We will turn the recurring zizmor findings into first-class, selectable rules in the existing in-process AST lint registry, without calling zizmor or fetching action metadata. Four new default rules are registered: `artipacked`, `cache-poisoning`, `excessive-permissions` (Actions only, since AW root permissions scope the agent job) and `obfuscation`. They emit warnings in both strict and non-strict mode. The existing `template-injection` rule is extended to cover `actions/github-script` inputs without a JavaScript parser. The existing `action-pin` and shell-injection checks are reused, so no duplicate rules are added. Shared AW execution traversal (`aw/execution.go`) is extended to include `safe-outputs.steps`. When a compiled Actions graph is supplied, it takes precedence over the companion AW source.

### Alternatives Considered

#### Alternative 1: Keep relying on the external zizmor scanner

Keep zizmor as the only detector for these patterns. This avoids adding and maintaining rule code. However, regressions surface only where and when zizmor runs, not in `gh aw lint` or compile-time checks, and not in WASM/in-browser use. The repository's history of repeated fixes shows this was not enough to stop regressions. It was rejected as the sole mechanism, but it stays complementary: the docs state that the new rules are "not complete zizmor parity."

#### Alternative 2: Embed or invoke zizmor from the linter

Run zizmor in-process or as a subprocess and map its findings to lint diagnostics. This would give broader parity, including creator verification. However, it adds an external binary or runtime dependency, is incompatible with the WASM build, and would bring zizmor's ignore-comment semantics and network or metadata needs into the linter. It was rejected in favour of self-contained AST checks.

#### Alternative 3: One aggregate "zizmor" rule or new duplicate rules for every finding

Register a single umbrella rule, or one new rule per historical finding, including pinning and shell injection. This would be simpler to describe. However, it would duplicate the existing `action-pin`/`template-injection` coverage and cause double-reporting, and a single umbrella rule could not be selected independently. It was rejected in favour of narrowly scoped rules plus extending existing ones.

### Consequences

#### Positive
- Regressions of the historically fixed patterns are caught by the default in-process linter, including in CLI and WASM builds, with the original fix PRs recorded in `workflow-linters.md`.
- Rules can be selected individually (`Default().Select("artipacked", ...)`) and reuse shared helpers (`actionName`, `actionInputBoolean`, `executionPolicyBlocks`), so no new rules duplicate existing ones.
- `safe-outputs.steps` are now included in shared execution traversal, so every execution-scoped rule covers them, not just the new rules.

#### Negative
- The rules are heuristic review warnings, not proof of compromise. For example, `artipacked` warns when `persist-credentials` is omitted even if later Git operations need credentials. Existing workflows may start producing new warnings: the existing CLI test had to add `persist-credentials: false`.
- Cache checks hard-code knowledge of specific actions and version defaults (pre-v4 `setup-go`, pre-v5 `setup-node`). This knowledge must be maintained as upstream action behaviour changes. SHA-pinned versions with unknown defaults need an explicit opt-out.
- Existing zizmor suppressions do not suppress these independent rules, so authors may need to handle the same finding in two places.

#### Neutral
- Coverage stops at static AST evidence. Imports are not recursively loaded, generated defaults are not reconstructed, and creator verification (#50015) remains out of scope.
- `excessive-permissions` applies only to the Actions AST, while the existing `permission-scope` rule continues to cover blanket grants for both AW and Actions.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
