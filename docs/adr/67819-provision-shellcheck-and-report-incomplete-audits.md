# ADR-67819: Provision Native ShellCheck and Report Incomplete MCP Audit Timeouts

**Date**: 2026-10-11
**Status**: Proposed
**Proposed deciders**: pelikhan (PR author), gh-aw maintainers

---

### Context

Two failures shared the privileged MCP tool path (issue #67671, clustering #67628/#67630/#65787/#66029). First, the `compile` tool in strict dry-run mode aborted with a missing-ShellCheck error because the `ghcr.io/github/gh-aw` image did not include ShellCheck, a native scanner required by strict validation. Second, the `audit` tool used a 5-minute subprocess deadline that was deliberately detached from the gateway's per-tool RPC deadline (`newMCPSubprocessContext`), so a slow audit could outlast the default 60-second MCP tool deadline without returning results to the caller. Both behaviours are security- and reliability-sensitive: strict validation must not be weakened, and an audit that cannot finish must not be mistaken for a complete report or persisted as one.

### Decision

The proposed decision satisfies the strict-validation and audit-deadline contracts rather than relaxing them. For ShellCheck, the compiler image installs the native `shellcheck` package and runs `shellcheck --version` at image build time so a missing scanner fails the build, not the user's compile; strict dry-run validation stays mandatory, with no Docker-in-Docker and no downgrade of a missing scanner to a warning. For audit, the MCP handler computes a response deadline of at most 55s, or 5s before an earlier caller deadline, and keeps that as the hard subprocess deadline. After subprocess admission, it passes the remaining budget minus another 5s as a cooperative CLI `--timeout` (normally about 50s). On cooperative single-run timeout with JSON output, the CLI emits an explicitly incomplete envelope (`partial: true`, `complete: false`, `stage`, `overview`, `suggestions`) and still exits unsuccessfully. The MCP layer marks timeout responses as tool errors, including `partial_results` when an incomplete CLI envelope is available; a hard timeout can return an error without partial results. Incomplete envelopes are never written to the full-audit or run-summary caches.

### Alternatives Considered

These alternatives describe design trade-offs inferred from the implementation and originating issue; they do not record maintainer approval or rejection.

#### Alternative 1: Downgrade a missing ShellCheck to a warning in the MCP compile path

This option was named in the originating issue ("Make the missing ShellCheck binary a reported warning... or provision the binary"). It avoids an image change but weakens the strict dry-run security gate: turning an absent scanner into a warning permits "strict" validation without that scan. Provisioning the dependency keeps the gate fail-closed.

#### Alternative 2: Run the scanners via Docker-in-Docker from the compiler image

Keep the image minimal and shell out to containerised scanner images on demand. This tracks scanner versions independently of the Alpine package feed, but requires a container runtime and potentially network pulls at compile time instead of a verified, baked-in binary. The native package avoids those additional runtime dependencies.

#### Alternative 3: Raise or keep the detached audit subprocess timeout

Leave `newMCPSubprocessContext` detachment in place and raise the subprocess budget or gateway RPC deadline. A larger subprocess budget alone does not address the caller losing results when the RPC deadline fires first. Any subprocess budget larger than the RPC deadline risks reproducing the silent-timeout bug; the fix reserves time to respond *before* the deadline.

#### Alternative 4: Return the partial envelope as a successful (non-error) result

`buildAuditErrorResult` previously set `IsError: false` so callers treated failures as graceful JSON. Continuing that for timeouts would avoid changing the tool's error contract, but could lead callers to treat an incomplete audit as a finished report. Timeout responses now use `IsError: true` and `complete: false`; other failures retain the existing non-error JSON envelope.

### Consequences

#### Positive
- Strict dry-run compilation works in the shipped compiler image without weakening the security gate; a missing scanner now breaks the image build instead of a user's compile.
- MCP `audit` execution reserves time to return a timeout response before the default 60-second tool deadline; partial single-run results are included when the CLI emits an incomplete envelope.
- Incomplete CLI results are explicitly labelled (`partial`/`complete`/`stage`) and are never persisted as complete audit caches.
- CLI users gain an independent `gh aw audit --timeout` control with the same incomplete-result semantics.

#### Negative
- The compiler image grows and now depends on the Alpine `shellcheck` package; a feed outage or package removal breaks the image build.
- The scanner version comes from the Alpine package repository at build time, without an explicit package-version pin; upgrades require an image rebuild rather than a flag.
- Long audits that previously completed under the 5-minute detached budget will now be cut off at ~50 seconds when invoked through MCP, forcing a retry (which reuses downloaded artifacts) or a narrowed `artifacts` set.
- Timeout responses are now tool errors (`IsError: true`), a behaviour change for callers that relied on audit failures being graceful successes.

#### Neutral
- `defaultMCPAuditTimeoutMinutes` is replaced by `defaultMCPAuditResponseTimeout` + `mcpAuditResponseMargin`; the audit tool no longer uses `newMCPSubprocessContext` deadline detachment, while `logs` and `audit-diff` still do.
- The timeout contract is now documented in both `docs/.../gh-aw-as-mcp-server.md` and `docs/.../setup/cli.md`, so the envelope shape is part of the public surface.
- Container tests (`TestDockerfile_Exists`, `TestDockerImage_HasRequiredTools`) now assert ShellCheck presence, and the native strict-compile test skips when ShellCheck is absent locally.

---

*Generated ADR reviewed against the implementation and proposed for maintainer consideration; architecture approval is not implied.*
