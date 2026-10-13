# ADR-68473: Enable function-backed Codex shell tools for GitHub inference

**Date**: 2026-10-13
**Status**: Draft
**Deciders**: Unknown (draft generated from PR #68473)

---

### Context

The Codex engine supports multiple inference providers. When the resolved provider is GitHub (Copilot-backed inference through the compatibility adapter), the compiler previously forced `features.shell_tool = false` in the generated native Codex config, alongside `code_mode = false` and `code_mode_only = false`. That blanket disable was introduced because the Copilot compatibility adapter does not support the Codex code-mode `exec` *custom* tool. However, the pinned Codex release (0.162.1) exposes shell access as ordinary *function* tools (`exec_command`, `write_stdin`), which the adapter passes through normally. The over-broad disable meant workflows that declared `tools.bash: ["*"]` silently lost all shell access: checkout jobs could not read local files, run edits or tests, or invoke CLI-mounted safe-output MCP servers, while the agent still exited successfully. The prompt validator compounded this by emitting provider-wide "native shell is disabled" warnings for every bash and native-read requirement on GitHub-backed Codex. Reported in #68434, #68431, and #68387.

### Decision

We will distinguish the unsupported code-mode `exec` custom tool from the supported function-backed shell tools: for GitHub-backed Codex, `features.shell_tool` is now derived from the workflow's bash policy (`!workflowData.BashDisabled`) while `code_mode` and `code_mode_only` remain hard-disabled. Config-override validation is extracted into `validateCodexGitHubConfig`, which continues to reject user overrides of `model_catalog_json`, `code_mode`, and `code_mode_only`, and additionally rejects an override that re-enables `features.shell_tool` when the compiler's own defaults disabled it (i.e. when `tools.bash` is `false` or `[]`). The provider-wide shell-disabled branch is removed from `prompt_tool_validation.go`, so bash requirements are validated against the normal bash policy, and `native-read` availability is checked inline against engine/provider rather than through the deleted `promptShellDisabledByProvider` helper.

### Alternatives Considered

#### Alternative 1: Keep `shell_tool` disabled and route all shell work through MCP tools

This preserves the status quo and needs no change to the override-validation surface; workflows would be told to replace shell instructions with explicitly configured MCP tools. It was rejected because it is not a workable substitute for checkout workflows: reading repository files, running tests, and invoking CLI-mounted safe outputs all require real process execution, and the restriction was based on a mistaken premise — the adapter's limitation applies to the code-mode custom tool, not to the `exec_command` function tool.

#### Alternative 2: Unconditionally enable `shell_tool` for GitHub inference

The simplest fix — drop the special case entirely and let Codex defaults apply. It was rejected because it would break the explicit denial contract: `tools.bash: false` and `tools.bash: []` must continue to disable shell execution, and user `engine.config` overrides must not be able to re-enable it. Deriving the feature flag from `BashDisabled` and guarding the override path keeps the denial path authoritative.

#### Alternative 3: Gate the change behind a new opt-in engine config flag

An opt-in flag would have limited blast radius while the behaviour is validated on hosted runners. It was rejected because the previous behaviour is a defect rather than a conservative default, and an opt-in would leave the 35 affected workflows broken by default while adding a configuration knob that would need deprecating once the fix is proven.

### Consequences

#### Positive
- GitHub-backed Codex workflows that declare `tools.bash: ["*"]` regain shell execution, restoring checkout, edit, test, and CLI-mounted safe-output flows.
- Misleading provider-wide "native shell is disabled" prompt warnings disappear, so remaining warnings reflect the actual bash policy.
- Override validation is now explicit about *why* each feature is managed, and is isolated in a single testable function (`validateCodexGitHubConfig`).

#### Negative
- The change rewrites 35 generated `.lock.yml` files (`shell_tool: false` → `true`), creating a large diff and a coordination point for any in-flight branches touching the same locks.
- Enabling shell execution on GitHub-backed Codex widens the agent's capability surface on those workflows; the safety boundary now rests entirely on the declared `tools.bash` policy and the override guard.
- Correctness depends on behaviour of pinned Codex 0.162.1 and AWF 0.28.51; a future upstream bump that changes how `exec_command` is surfaced could silently regress this.

#### Neutral
- Code-mode remains disabled for GitHub inference, so the engine stays in direct-tool mode and the code-mode `exec` custom tool is still unavailable.
- `promptShellDisabledByProvider` is deleted; the `native-read` availability check is now expressed inline at its single call site.
- Hosted execution was not verified in this PR (no workflows dispatched, Docker unavailable locally); validation rests on unit and regression tests plus compilation of all 341 workflows.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
