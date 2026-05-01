# ADR-29588: Targeted Credential Cleanup for Known Credential-Leaking GitHub Actions

**Date**: 2026-05-01
**Status**: Draft
**Deciders**: pelikhan

---

## Part 1 — Narrative (Human-Friendly)

### Context

Certain well-known GitHub Actions authenticate to cloud providers or container registries and leave credentials on disk after executing. Actions such as `google-github-actions/auth`, `aws-actions/configure-aws-credentials`, `azure/login`, `docker/login-action`, and `actions/checkout` (with deploy keys) write credentials to well-known filesystem locations (e.g., `~/.aws/credentials`, `~/.azure/`, `~/.docker/config.json`, `~/.ssh/`). When the gh-aw agentic engine executes inside the same GitHub Actions job, it operates in the same runner environment and can read — and potentially exfiltrate — those credentials. This is a security boundary violation: users configure cloud credentials for their workflow steps, not for the AI agent.

### Decision

We will implement a **compile-time detection, runtime cleanup** strategy: the workflow compiler statically scans all merged step collections for known credential-leaking actions, records which providers were detected, and injects a targeted cleanup step immediately before the agentic engine executes. Cleanup is driven by `GH_AW_CLEAN_*` environment variables so that only the credential locations belonging to detected actions are removed — no undetected providers are touched. This approach was chosen because it is surgical (avoids accidentally removing credentials a non-agentic step may still need), deterministic (compile-time detection means no runtime surprises), and self-documenting (the emitted YAML step makes the cleanup visible in the compiled lock file).

### Alternatives Considered

#### Alternative 1: Blanket Credential Sweep (Unconditional Removal)

Run a single cleanup step that removes all known credential locations unconditionally, regardless of which actions are present in the workflow. This is simpler to implement — no detection logic needed — but would destructively remove credentials even when no credential-leaking action is present, potentially breaking workflows that store non-agent credentials in those locations. It also provides no auditability of which specific providers were cleaned.

#### Alternative 2: Policy-Only / Documentation

Document the credential exposure risk and require workflow authors to add their own cleanup steps before agent execution. This requires zero platform changes and preserves maximum flexibility, but it relies entirely on user awareness and compliance. Given that the agentic engine runs in a context where most users may not anticipate credential exposure, a passive documentation-only approach was deemed insufficient to meet the security bar.

#### Alternative 3: Block Workflows That Use Credential-Leaking Actions

Reject compilation of any workflow that includes a known credential-leaking action. This is maximally secure but is too restrictive: it would break the common and legitimate use case of authenticating to a cloud provider to perform pre-agent setup steps (e.g., pulling a container image, fetching secrets for the agent's environment). Users should be able to use these actions — just not have their credentials available to the agent.

### Consequences

#### Positive
- Prevents the agentic engine from accessing cloud-provider credentials that were intended only for pre-agent workflow steps.
- Cleanup is surgical and targeted: only credential locations for detected actions are removed, reducing the risk of breaking legitimate non-agent usage.
- The cleanup step is visible in the compiled lock YAML, making the security measure auditable and transparent to workflow authors.
- No cleanup step is emitted when no known credential-leaking actions are present, keeping the compiled output minimal.

#### Negative
- The detection logic relies on a static, compile-time allowlist (`knownCredentialLeakingActions`). Novel or organization-internal credential-leaking actions not on the list will not be cleaned up; the allowlist must be maintained as new actions emerge.
- Detection matches only on action prefix (strips `@version`), so a fork of a known action at a different org path would not be detected.
- The feature cannot protect against credentials written to non-standard or undocumented paths by known actions (e.g., if a future version of `aws-actions/configure-aws-credentials` changes its credential path).

#### Neutral
- The cleanup step uses `continue-on-error: true`, meaning failures in credential removal do not abort the job. This is consistent with the existing `clean_git_credentials.sh` step but means a silent failure leaves credentials in place.
- The compiled lock files for existing workflows that use known credential-leaking actions will gain a new step on the next recompile (a backward-compatible, additive change).
- Detection runs across all three step collections (`customSteps`, `preSteps`, `preAgentSteps`), which future step collection additions would require updating.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### Detection

1. The workflow compiler **MUST** scan all merged step collections (custom steps, pre-steps, and pre-agent-steps) for known credential-leaking actions at compile time, before generating the final YAML output.
2. Detection **MUST** match on the action name prefix (the portion before `@version`), stripping inline comments (` # …`) before comparison.
3. The set of known credential-leaking actions **MUST** include at minimum: `google-github-actions/auth`, `aws-actions/configure-aws-credentials`, `azure/login`, `docker/login-action`, and `actions/checkout`.
4. Detection results **MUST** be stored on the `WorkflowData` struct so that downstream compilation steps can consume them without re-scanning.

### Cleanup Step Injection

1. When one or more known credential-leaking actions are detected, the compiler **MUST** inject a cleanup step immediately before the agentic engine execution step.
2. The injected cleanup step **MUST** set `continue-on-error: true` to prevent a cleanup failure from blocking agent execution.
3. The injected cleanup step **MUST** include only the `GH_AW_CLEAN_*` environment variables corresponding to detected actions; variables for undetected providers **MUST NOT** be included.
4. The environment variables in the injected step **MUST** be emitted in a stable, deterministic order (matching the canonical order defined in `knownCredentialLeakingActions`).
5. When no known credential-leaking actions are detected, the compiler **MUST NOT** emit a cleanup step.

### Cleanup Script Behavior

1. The cleanup script (`clean_known_action_credentials.sh`) **MUST** perform cleanup for a provider only when the corresponding `GH_AW_CLEAN_*` variable is set to `"true"`.
2. The script **MUST** exit with code `0` on success (including when there are no credentials to clean).
3. The script **MUST NOT** remove credential files for providers whose `GH_AW_CLEAN_*` variable is not set.
4. The script **SHOULD** log each removed file or directory to standard output to aid in debugging.

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Failure to meet any **MUST** or **MUST NOT** requirement constitutes non-conformance.

---

*This is a DRAFT ADR generated by the [Design Decision Gate](https://github.com/github/gh-aw/actions/runs/25226791761) workflow. The PR author must review, complete, and finalize this document before the PR can merge.*
