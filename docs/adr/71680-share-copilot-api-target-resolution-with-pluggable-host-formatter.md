# ADR-71680: Share Copilot API Target Resolution with a Pluggable Host Formatter

**Date**: 2026-10-24
**Status**: Draft
**Deciders**: [TODO: verify] pelikhan, Copilot coding agent

---

### Context

The compiler derives the Copilot API target that is passed to AWF's api-proxy and that is also used for host-only consumers such as firewall allowlists. Resolution follows a precedence chain: explicit `engine.api-target`, then `GITHUB_COPILOT_BASE_URL`, then a literal BYOK `COPILOT_PROVIDER_BASE_URL`. `GetCopilotAPITarget` always used `extractAPITargetHost`, which strips the URL scheme, so a BYOK URL such as `http://host.docker.internal:11434` was emitted to the API proxy config as `host.docker.internal:11434`. That loses the protocol information needed to route to plain-HTTP backends such as Ollama. A version-gated formatter, `extractAPIProxyTargetHost`, already existed that preserves `http://` when the effective AWF version supports HTTP API targets (`awfSupportsHTTPAPITargets`), but the Copilot path did not use it.

### Decision

We will extract the Copilot precedence chain into a single internal function, `resolveCopilotAPITarget(workflowData, extractHost)`, that takes the host-extraction strategy as a function parameter. `GetCopilotAPITarget` keeps its host-only semantics by passing `extractAPITargetHost`, while `BuildAWFConfigJSON` passes a closure over `extractAPIProxyTargetHost` bound to the firewall config so API proxy targets keep the `http://` scheme when AWF supports it. The literal-only gate for `COPILOT_PROVIDER_BASE_URL` (ignoring `${{ ... }}` expressions) is preserved by checking `extractLiteralEngineEnvHost` before delegating to the injected extractor. The main reason is to have one source of truth for target precedence while letting each consumer choose the output format it needs.

### Alternatives Considered

#### Alternative 1: Change `GetCopilotAPITarget` to always preserve the scheme

Making the existing function return `http://host:port` everywhere would have been the smallest diff. It was not chosen because host-only consumers (firewall allowlists, `GetCopilotAllowlistTargets`) expect bare hostnames. Older AWF versions also do not accept scheme-prefixed targets, so this would have broken legacy compatibility.

#### Alternative 2: Duplicate the precedence chain in `BuildAWFConfigJSON`

The API proxy builder could re-implement the explicit-target / `GITHUB_COPILOT_BASE_URL` / BYOK fallback chain inline using `extractAPIProxyTargetHost`. This avoids a function-typed parameter, but the two copies of the precedence rules could drift apart, which is the kind of divergence that caused this bug.

#### Alternative 3: Post-process the host-only result to re-add the scheme

The caller could take the bare host from `GetCopilotAPITarget` and look up which env var it came from to decide whether to prepend `http://`. This was not chosen because it needs the caller to reconstruct which precedence branch matched, which spreads the resolution logic across call sites.

### Consequences

#### Positive
- BYOK HTTP backends (for example Ollama at `http://host.docker.internal:11434`) receive a scheme-preserving target, so the api-proxy can route over plain HTTP.
- Target precedence is defined in one place and shared by host-only and API proxy consumers.
- AWF version gating is reused through `extractAPIProxyTargetHost`, so legacy AWF versions still receive bare hosts.

#### Negative
- `resolveCopilotAPITarget` takes a function parameter, which adds some indirection compared with a direct call.
- The BYOK fallback now calls `extractLiteralEngineEnvHost` only as a gate and then extracts the host again through the injected function. This parses the URL twice and couples the two helpers' notions of a valid URL.
- `GetCopilotAPITarget` and the API proxy config can now return different strings for the same workflow (bare host vs `http://host`). Future callers must pick the right entry point.

#### Neutral
- Compiled lock files that use HTTP BYOK providers change only in the target's scheme (for example `daily-byok-ollama-test.lock.yml`).
- Regression coverage grows to 14 table cases covering HTTPS, HTTP, expressions, explicit precedence, and legacy AWF behavior.
- The ADR does not establish whether this bug caused the 503 failures seen in upstream runs. That remains unproven.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
