# ADR-68819: Fail closed when the Copilot CLI substitutes the configured model

**Date**: 2026-10-14
**Status**: Accepted
**Deciders**: @pelikhan (PR #68819)

---

### Context

gh-aw lets a workflow select a concrete model or a model alias for the Copilot engine. Two paths could obscure a different selection. First, AWF alias resolution (`actions/setup/js/resolve_model_alias.cjs`) conflated a successfully retrieved catalog with no matching model and a failed catalog retrieval. Second, the native Copilot CLI could report a different startup model than requested while the run still succeeded and its metadata retained the requested model.

[Issue #66894](https://github.com/github/gh-aw/issues/66894) documents an upstream access boundary: a model enabled by organization policy and offered to the standalone CLI may not be offered to the `agentic-workflows` integration. Neither the bundled model catalog nor compilation proves runtime integration access. gh-aw can resolve against the workflow's live catalog and inspect the native CLI's model-selection events; those events report the CLI's selection, not independently verified upstream inference identity.

### Decision

Fail the native main-agent attempt when the CLI reports a concrete model different from the resolved selection. Resolve explicit CLI model arguments before alias and endpoint preprocessing. `actions/setup/js/copilot_model_selection.cjs` reads only newly appended events from the invocation's exact session ID, excludes subagent records, and compares startup selection and main-assistant model identifiers. Continuation retries retain that exact session; fresh retries use a new session. Comparison normalizes Copilot provider prefixes and ignores query parameters that configure reasoning rather than model identity. A mismatch terminates a running CLI or overrides an otherwise successful quick exit, and is not retried. An intentional `auto` selection is not treated as substitution.

For known aliases, distinguish `model_unavailable` from `catalog_unavailable`. After resolution fails, an unavailable-model diagnosis requires that no configured candidate provider has a failed model-catalog fetch; an unrelated provider's populated catalog is insufficient evidence. A failed relevant fetch remains a retrieval failure. A bounded refresh is retained, and an initial established unavailability diagnosis is preserved if the refresh fails. No unresolved known alias is forwarded to the CLI.

Record configured, resolved, and native-observed model identifiers separately in `aw_info.json`, and use observed selection in logs and audit attribution when present. An unavailable model before execution has no effective model. Preserve authorized AWF fallback attribution precedence; native startup substitution is not authorization to use a fallback. Publish a post-agent parser snapshot through the agent artifact and `workflow.info` so conclusion jobs and downloaded logs retain selection status, including an intentionally empty model, instead of restoring the activation snapshot.

The host parser overwrites `runtime/aw_info.json` after agent process termination. Collectors prefer this post-agent publication and do not promote `agent/aw_info.json` to authoritative workflow metadata. The runtime directory is not outside the shared writable sandbox mount; publication and provenance prevent stale activation or untrusted agent-output file preference, not adversarial tampering with the shared source or native session events. Unsandboxed execution retains its existing lack of filesystem isolation.

The compiler emits a warning, not an error, for the `gemini-*` family using Copilot's GitHub integration without an explicit provider base URL. The bare `gemini` engine-name meta-alias maps to `agent` and does not warrant a Gemini-family warning. Runtime expressions and explicit BYOK configurations are not rejected by this check.

Integration access policy remains upstream. This decision does not change the integration ID, bypass access controls, enable Gemini for that integration, or establish the identity of the model serving inference behind Copilot/CAPI.

### Alternatives Considered

#### Alternative 1: Warn on substitution and let the run continue

The observer could log a warning and allow a different native selection to proceed. This would keep runs green, but tolerate a model the workflow did not select. Rejecting an observed mismatch makes the selection failure explicit even though it converts previously passing runs into failures.

#### Alternative 2: Reject unavailable Gemini/Copilot model configurations at compile time

The compiler could prohibit Gemini-family selections for Copilot's GitHub integration. A static catalog cannot establish live access, and a permanent prohibition would reject models later enabled upstream. The check stays advisory; live-catalog resolution and native selection observation provide separate runtime evidence. Explicit BYOK is outside this warning's scope.

#### Alternative 3: Change the integration ID to obtain broader model access

Using the standalone CLI's integration identity could evade the access boundary described in the issue. This is rejected as a policy bypass. Missing integration access must be enabled by Copilot/CAPI, not worked around by gh-aw.

#### Alternative 4: Treat native events as proof of upstream inference identity

The native observer sees CLI session records, not a trusted attestation from the inference service. Reporting its selection as verified upstream identity would overstate the available evidence. Native-observed attribution is retained with this limitation.

### Consequences

#### Positive
- An observed native model mismatch cannot be reported as a successful attempt using the requested model.
- Operators get separate actionable diagnoses for unavailable configured models and incomplete catalog retrieval, with a bounded refresh.
- Metadata and logs distinguish author intent, alias resolution, and native-observed selection instead of conflating them.
- An advisory warning surfaces the Gemini-family integration access concern without permanently prohibiting valid future configurations.

#### Negative
- Workflows that previously completed despite an observed native substitution now fail without retrying that mismatch.
- Detection depends on CLI event formats, locations, and emitted model identifiers. Missing or unrecognized model events do not independently establish upstream identity; a matching native event also cannot exclude substitution behind Copilot/CAPI.
- The post-agent snapshot and session provenance identify the publication path, not an adversarial attestation. The default sandbox exposes the shared temporary directory, and unsandboxed execution has no filesystem isolation; this decision does not establish tamper-proof session or metadata storage.
- Consumers of model metadata must distinguish configured, resolved, and observed identifiers, and retain existing behavior when observation metadata is absent.

#### Neutral
- The observer excludes historical events, subagents, SDK mode, and the threat-detection phase; this guard applies only to native main-agent attempts.
- Lock-file model metadata remains explicitly compile-time configuration and is not rewritten with the observed model.
- Upstream integration access and the separately reported Gemini BYOK `refusal: null` replay failure are unchanged; authorized AWF fallback precedence is preserved.
