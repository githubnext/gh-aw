# ADR-67819: Provision Native ShellCheck and Report Incomplete MCP Audit Timeouts

**Date**: 2026-10-13
**Status**: Draft
**Deciders**: pelikhan (PR author), gh-aw maintainers

---

### Context

Two failures shared the privileged MCP tool path (issue #67671, clustering #67628/#67630/#65787/#66029). First, the `compile` tool in strict dry-run mode aborted with a missing-ShellCheck error because the `ghcr.io/github/gh-aw` image shipped only `gh`, `git`, `jq`, and `bash`, so the security scanners required by strict validation were simply absent from the container the tool runs in. Second, the `audit` tool used a 5-minute subprocess deadline that was deliberately detached from the gateway's per-tool RPC deadline (`newMCPSubprocessContext`), so a slow audit silently blew past the default 60-second MCP tool deadline and the caller received nothing — no metadata, no partial analysis, no actionable error. Both behaviours are security- and reliability-sensitive: strict validation must not be weakened, and an audit that cannot finish must not be mistaken for a complete report or persisted as one.

### Decision

We will fix both problems by making the environment satisfy the contract rather than relaxing the contract. For ShellCheck, we install the native `shellcheck` package in the compiler image and run `shellcheck --version` at image build time so a missing scanner fails the build, not the user's compile; strict dry-run validation stays mandatory, with no Docker-in-Docker and no downgrade of a missing scanner to a warning. For audit, we bound execution *inside* the 60-second tool deadline: the MCP handler computes a response deadline (`defaultMCPAuditResponseTimeout` 55s, `mcpAuditResponseMargin` 5s), passes a cooperative `--timeout` to the CLI, and keeps a separate hard subprocess deadline. On timeout the CLI emits an explicitly incomplete JSON envelope (`partial: true`, `complete: false`, `stage`, `overview`, `suggestions`) and still exits unsuccessfully, and the MCP layer surfaces it as a tool error with `partial_results`. Incomplete results are never written to the full-audit or run-summary caches.

### Alternatives Considered

#### Alternative 1: Downgrade a missing ShellCheck to a warning in the MCP compile path

This was the first option named in the originating issue ("Make the missing ShellCheck binary a reported warning... or provision the binary"), and it is the cheapest fix: no image change, no build-time verification, works in any environment. It was rejected because strict dry-run compilation is a security gate; turning an absent scanner into a warning means a workflow can pass "strict" validation without ever being scanned, and the failure becomes invisible exactly where it matters most. Provisioning the dependency keeps the gate fail-closed.

#### Alternative 2: Run the scanners via Docker-in-Docker from the compiler image

Keep the image minimal and shell out to containerised scanner images on demand. This preserves a small base image and tracks scanner versions independently of the Alpine package feed. It was rejected because Docker-in-Docker is unavailable in the MCP tool environment, adds a privileged socket dependency, and makes the strict path depend on network pulls at compile time instead of a verified, baked-in binary.

#### Alternative 3: Raise or keep the detached audit subprocess timeout

Leave `newMCPSubprocessContext` detachment in place and simply raise the subprocess budget (or raise the gateway RPC deadline). This is a one-constant change and lets genuinely long audits run to completion. It was rejected because the problem is not that audits are too slow — it is that the caller gets nothing when the RPC deadline fires first. Any subprocess budget larger than the RPC deadline reproduces the silent-timeout bug; the fix must guarantee a response *before* the deadline.

#### Alternative 4: Return the partial envelope as a successful (non-error) result

`buildAuditErrorResult` previously set `IsError: false` so callers treated failures as graceful JSON. Continuing that for timeouts would avoid changing the tool's error contract. It was rejected because an incomplete audit returned as success invites agents and caches to treat it as a finished report; marking it `IsError: true` with `complete: false` makes incompleteness impossible to miss.

### Consequences

#### Positive
- Strict dry-run compilation works in the shipped compiler image without weakening the security gate; a missing scanner now breaks the image build instead of a user's compile.
- MCP `audit` callers always get a response within the 60-second tool deadline, with whatever single-run metadata and analysis were collected.
- Incomplete results are explicitly labelled (`partial`/`complete`/`stage`) and are never persisted as complete audit caches, so a timed-out run cannot poison later reads.
- CLI users gain an independent `gh aw audit --timeout` control with the same incomplete-result semantics.

#### Negative
- The compiler image grows and now depends on the Alpine `shellcheck` package; a feed outage or package removal breaks the image build.
- Scanner version is pinned to whatever the base image provides, so upgrades require an image rebuild rather than a flag.
- Long audits that previously completed under the 5-minute detached budget will now be cut off at ~50 seconds when invoked through MCP, forcing a retry (which reuses downloaded artifacts) or a narrowed `artifacts` set.
- Timeout responses are now tool errors (`IsError: true`), a behaviour change for callers that relied on audit failures being graceful successes.

#### Neutral
- `defaultMCPAuditTimeoutMinutes` is replaced by `defaultMCPAuditResponseTimeout` + `mcpAuditResponseMargin`; the audit tool no longer uses `newMCPSubprocessContext` deadline detachment, while `logs` and `audit-diff` still do.
- The timeout contract is now documented in both `docs/.../gh-aw-as-mcp-server.md` and `docs/.../setup/cli.md`, so the envelope shape is part of the public surface.
- Container tests (`TestDockerfile_Exists`, `TestDockerImage_HasRequiredTools`) now assert ShellCheck presence, and the native strict-compile test skips when ShellCheck is absent locally.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
