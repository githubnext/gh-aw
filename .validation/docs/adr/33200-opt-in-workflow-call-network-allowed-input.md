# ADR-33200: Opt-in `workflow_call` `network_allowed` Input for Reusable Workflows

**Date**: 2026-05-19
**Status**: Draft
**Deciders**: Unknown

---

## Part 1 — Narrative (Human-Friendly)

### Context

Reusable `.lock.yml` workflows currently bake the source workflow's `network.allowed` allowlist directly into the compiled AWF config. Consumer repositories that need to add a domain or ecosystem to that allowlist cannot do so without forking the source workflow and recompiling the lock file, which defeats the purpose of centrally hosted reusable workers. The source author still needs to retain a static minimum floor for the allowlist so callers cannot weaken the security posture set by the publisher. The change must apply consistently across all AWF-backed engines (Claude, Codex, Copilot, Crush, Gemini, OpenCode, Pi) so the reusable contract behaves the same regardless of which engine is configured.

### Decision

We will introduce an opt-in `network.allowed-input: true` frontmatter flag on `workflow_call` workflows. When enabled, the compiler injects a new `network_allowed: string` input into the compiled `workflow_call` interface and emits a runtime step that unions the caller-supplied value (comma-separated ecosystem identifiers and/or domains) into `awf-config.json`'s `network.allowDomains` before AWF starts. The compiled workflow's static `network.allowed` continues to act as the immutable floor; the caller input is additive and deduplicated. The runtime input is forwarded into the AWF-backed engine step via the `GH_AW_WORKFLOW_CALL_NETWORK_ALLOWED` environment variable rather than inline `${{ }}` expansion, to avoid shell-injection surface area.

### Alternatives Considered

#### Alternative 1: Always expose `network_allowed` on every `workflow_call` workflow

Make the input ubiquitous so consumers never have to coordinate with the source author. Rejected because it silently changes the contract for every reusable workflow currently published from this repo, gives any caller the ability to extend the allowlist whether or not the publisher intends to delegate that authority, and breaks the principle that network allowlists are a security-relevant setting controlled by the source workflow author.

#### Alternative 2: Allow the caller input to fully replace the source allowlist

Treat `inputs.network_allowed` as an override rather than a union, letting callers narrow or completely rewrite the allowlist. Rejected because it removes the publisher's ability to enforce a minimum security floor — a caller could remove `defaults` or drop required ecosystem domains, breaking the source workflow in subtle ways or weakening its sandbox. Union-with-floor preserves publisher control while still permitting per-caller extension.

#### Alternative 3: Recompile the lock file per consumer

Encourage consumers to fork or run `gh-aw compile` against the source workflow with their own additional `network.allowed` entries. Rejected because it defeats the central-hosting model (every consumer maintains a fork), creates version-drift and supply-chain risk, and pushes a compile-time toolchain requirement onto every downstream repo that just wants to call a shared worker.

### Consequences

#### Positive
- Centrally hosted reusable workflows can now serve callers with varying network needs without per-consumer forks or recompilation.
- The source author retains an enforced floor on the allowlist; callers can only add domains, never remove them.
- Opt-in semantics keep behavior unchanged for every existing reusable workflow that does not set `allowed-input: true`.
- Environment-variable plumbing (`GH_AW_WORKFLOW_CALL_NETWORK_ALLOWED`) keeps the caller-supplied value out of inline shell expansion, reducing template-injection risk.

#### Negative
- The effective network allowlist is no longer statically derivable from the lock file alone — auditors must also inspect the caller's `with:` block to know what domains were actually permitted at runtime.
- Introduces a runtime Python step that reads, mutates, and rewrites `awf-config.json` between config generation and AWF startup, adding a new failure surface (file I/O, JSON parsing) on the agent's critical path.
- The compiler must embed the full ecosystem-to-domain expansion map as JSON inside the generated Python script, duplicating logic that already exists in Go (`getEcosystemDomains`) and creating a maintenance pairing.

#### Neutral
- Adds a new public env var name `GH_AW_WORKFLOW_CALL_NETWORK_ALLOWED` to the workflow-engine contract.
- Adds a new top-level constant `NetworkAllowedInputName = "network_allowed"` that becomes part of the reusable-workflow input vocabulary alongside `aw_context`.
- Refactors the previously aw-context-specific `injectAwContextIntoTrigger` helper into a generic `injectInputIntoTrigger` so the same machinery serves both internal inputs.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### Frontmatter Schema

1. The compiler **MUST** accept an optional boolean `network.allowed-input` field in workflow frontmatter.
2. The compiler **MUST** treat `network.allowed-input` as `false` when the field is omitted, preserving prior compilation output for existing workflows.
3. The compiler **MUST NOT** inject the `network_allowed` input into the compiled workflow when `network.allowed-input` is `false` or absent.
4. The compiler **MUST NOT** inject the `network_allowed` input into the compiled workflow when the source workflow does not declare a `workflow_call` trigger, regardless of the value of `network.allowed-input`.

### Compiled `workflow_call` Interface

1. When `network.allowed-input` is `true` and the workflow declares a `workflow_call` trigger, the compiled `.lock.yml` **MUST** expose a `network_allowed` input under `on.workflow_call.inputs` typed as `string`, with `required: false` and a `default` of the empty string.
2. The injected `network_allowed` input description **MUST** document that the value is a comma-separated list of ecosystem identifiers and/or domains that will be unioned with the static allowlist at runtime.
3. The input-injection routine **MUST** be idempotent: running it twice on the same `on:` section **MUST** produce the same output and **MUST NOT** duplicate the `network_allowed` key.

### Runtime AWF Config Merge

1. When the feature is enabled, the runtime config-setup step **MUST** preserve the compiled workflow's static `network.allowed` entries as the baseline `network.allowDomains` of `awf-config.json`.
2. The runtime merge **MUST** union the tokens parsed from the caller-supplied `network_allowed` value into `network.allowDomains` before AWF starts.
3. Tokens **MUST** be split on `,`, trimmed of surrounding whitespace, and empty tokens **MUST** be discarded.
4. Ecosystem identifiers (e.g. `rust`, `python`) **MUST** be expanded to their concrete domain sets using the same ecosystem-to-domain mapping the compiler uses for static `network.allowed` entries; non-ecosystem tokens **MUST** be treated as literal domains.
5. The merged domain list **MUST** be deduplicated while preserving the order of first appearance.
6. The runtime step **MUST** fail with a clear diagnostic message when `awf-config.json` is missing, unparseable, unreadable, or unwritable, rather than silently continuing with an incomplete allowlist.

### Engine Wiring

1. AWF-backed engines (Claude, Codex, Copilot, Crush, Gemini, OpenCode, Pi) **MUST** pass the caller-supplied `network_allowed` value to the runtime config-setup step via the `GH_AW_WORKFLOW_CALL_NETWORK_ALLOWED` environment variable.
2. The caller-supplied value **MUST NOT** be inlined into shell scripts via `${{ inputs.network_allowed }}` expression interpolation at any site other than the engine-step `env:` block.
3. The environment variable **MUST NOT** be set when `network.allowed-input` is disabled for the workflow, so consuming steps can rely on its presence as a feature signal.

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Failure to meet any **MUST** or **MUST NOT** requirement constitutes non-conformance.

---

*This is a DRAFT ADR generated by the [Design Decision Gate](https://github.com/github/gh-aw/actions/runs/26077668160) workflow. The PR author must review, complete, and finalize this document before the PR can merge.*
