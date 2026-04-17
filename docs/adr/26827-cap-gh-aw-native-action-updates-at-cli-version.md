# ADR-26827: Cap gh-aw Native Action Updates at Running CLI Version

**Date**: 2026-04-17
**Status**: Draft
**Deciders**: dsyme, Copilot

---

## Part 1 — Narrative (Human-Friendly)

### Context

The `gh aw update` command resolves and writes the latest available version of each action in `actions-lock.json`. Actions in the `github/gh-aw` and `github/gh-aw-actions` repositories are special: they are released in lock-step with the CLI itself — every CLI release produces an identically-tagged action release, and the two are designed to work together as a matched pair. Before this change, `gh aw update` treated these actions identically to third-party actions and updated them to the newest release regardless of the CLI version the user had installed, silently pinning a version that could be ahead of — and incompatible with — the running CLI. This created a correctness gap where a user on CLI `v0.68.3` could have their workflows pinned to action `v0.68.7`, a version the user's CLI had no knowledge of.

### Decision

We will introduce a classification of "gh-aw native actions": any action whose base repository is `github/gh-aw` or `github/gh-aw-actions`. When resolving the target version for a native action during `gh aw update`, if the latest available release exceeds the version of the currently running CLI, the target version is capped at the CLI version instead. The SHA for that capped version tag is fetched and used; if the tag cannot be resolved (e.g., the tag does not yet exist), the native action is skipped entirely rather than updated to a mismatched version. Non-native actions are unaffected by this policy.

### Alternatives Considered

#### Alternative 1: No Version Cap (Prior Behavior)

Allow `gh aw update` to update native actions to the latest available release unconditionally, matching the behavior for all other actions. This is the simplest implementation but is semantically incorrect: it allows users to pin a native action version their CLI cannot interpret, leading to silent mismatches. The prior behavior was specifically identified as a bug, not a feature, so continuing it was not a viable alternative.

#### Alternative 2: Warn but Allow the Update

Emit a warning when a native action's resolved version exceeds the CLI version, but still apply the update. This preserves user agency but offers a poor default: most users would not notice or understand the warning, and the mismatch would silently persist until the CLI was upgraded. Given that the native actions and CLI must be at the same version to function correctly, allowing a mismatched update — even with a warning — is worse than skipping it.

#### Alternative 3: Maintain a Compatibility Manifest

Introduce a checked-in file (e.g., `compatibility.json`) that explicitly maps CLI version ranges to compatible action version ranges. `gh aw update` would consult this manifest to determine the correct action version. This approach is maximally flexible and would support situations where action and CLI versions diverge intentionally, but it introduces a separate artifact to maintain, requires updates on every release, and adds complexity that is unwarranted given the invariant that native action and CLI versions are always released as a matched pair.

### Consequences

#### Positive
- Users on any CLI version will always have their native actions updated to the correct matching version, never to a version the CLI does not understand.
- The behavior is fail-safe: if the capped version's tag cannot be resolved, the action is skipped rather than updated to a mismatched version, preventing silent breakage.
- The fix is transparent to users via log messages that explain when a cap or skip occurs.

#### Negative
- If a hotfix release of the native actions is published (e.g., `v0.68.3-patch.1`) after the CLI `v0.68.3` release, users on `v0.68.3` will not receive that hotfix via `gh aw update` because the cap targets exactly the CLI version tag (`v0.68.3`). They would need to upgrade the CLI to get the hotfix.
- `isGhAwNativeAction` hard-codes two repository prefixes (`github/gh-aw` and `github/gh-aw-actions`). Adding a third native action repository in the future requires a code change to this function.

#### Neutral
- Non-native actions (e.g., `actions/checkout`, third-party actions) are unaffected by this change. Their update logic is identical to the pre-fix behavior.
- The `getActionSHAForTagFn` function, already injectable for testing, is reused to resolve the capped version's SHA, keeping the cap logic covered by the existing test infrastructure.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### Native Action Classification

1. Implementations **MUST** classify an action as a "gh-aw native action" if and only if its base repository (resolved via `gitutil.ExtractBaseRepo`) is `github/gh-aw` or `github/gh-aw-actions`.
2. Implementations **MUST NOT** classify any other action as a gh-aw native action, including actions in forked or similarly named repositories.

### Version Cap Enforcement

1. For each gh-aw native action, implementations **MUST** compare the latest available release version against the version of the currently running CLI before applying any update.
2. If the latest available version is strictly newer than the running CLI version, implementations **MUST** cap the target version to the running CLI version and **MUST** resolve the SHA for that capped version tag.
3. If the capped version tag's SHA cannot be resolved, implementations **MUST** skip the update for that native action entirely and **MUST NOT** apply any version change to it.
4. If the latest available version is equal to or older than the running CLI version, implementations **MUST** proceed with the standard update logic unchanged.
5. Implementations **MUST NOT** apply the version cap to non-native actions; non-native actions **MUST** continue to follow the existing update logic.

### Observability

1. When a native action's update target is capped, implementations **SHOULD** emit a log message identifying the action, the capped target version, and the latest available version that was rejected.
2. When verbose mode is enabled, implementations **MUST** emit a formatted message to stderr explaining the cap or skip decision for each affected native action.

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Failure to meet any **MUST** or **MUST NOT** requirement constitutes non-conformance.

---

*This is a DRAFT ADR generated by the [Design Decision Gate](https://github.com/github/gh-aw/actions/runs/24560253328) workflow. The PR author must review, complete, and finalize this document before the PR can merge.*
