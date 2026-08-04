# ADR-50259: Enforce Host Allowlist in Workflowspec Parser and Redact Sensitive Env Vars in MCP Inspect

**Date**: 2026-08-04
**Status**: Proposed — pending maintainer acceptance on merge
**Deciders**: pelikhan, Copilot SWE Agent

---

### Context

The `gh aw` CLI resolves remote workflow dependencies via workflowspec strings of the form `host/owner/repo/path@ref`. The parser entry point (`pkg/cli/spec.go`) enforced an `isGitHubHost` allowlist (`github.com`, `*.ghe.com`, `*.github.com`) and validated `owner`/`repo` via `IsValidGitHubIdentifier`/`IsValidGitHubRepositoryName`. However, the nested-import resolution path (`pkg/parser/remote_workflow_spec.go` → `parseWorkflowSpecParts`) had **no equivalent gate**: any string containing a dot was accepted as a host and passed to the authenticated go-gh REST client and git fallback URL builders without validation. This asymmetry created an SSRF vector: a third-party workflow with a nested import `evil.com/owner/repo/path.md@ref` would cause `gh aw` to fire an authenticated outbound request to the attacker's host at compile time, and would attach enterprise tokens (`GH_ENTERPRISE_TOKEN`) when set.

Separately, `gh aw mcp inspect` printed the raw MCP server environment map — including `GITHUB_PERSONAL_ACCESS_TOKEN` — verbatim to stderr via `fmt.Fprintf(os.Stderr, "  Environment Variables: %v\n", config.Env)`. This print is not gated by `--verbose` and applies no redaction, so GitHub PATs land in CI job logs (readable by anyone with Actions read access) and in local terminal scrollback.

### Decision

We will enforce two security boundaries at the code layer:

1. **Parser-side host allowlist**: Add `IsGitHubHost` (`pkg/parser/github_urls.go`) — mirroring the CLI allowlist — and call it in `parseWorkflowSpecParts` before returning `host`, `owner`, or `repo`. Any workflowspec host that is not `github.com`, `raw.githubusercontent.com`, `*.ghe.com`, or `*.github.com` is rejected with an error before any outbound request is made.

2. **Env-var redaction in MCP inspect**: Add `redactSensitiveEnvValues` that masks values for any env key whose lowercase name contains `token`, `secret`, `key`, `password`, `credential`, or `auth`, replacing non-empty values with `***redacted***`. Apply this before printing env vars to stderr in `spawnMCPInspector`.

As part of enforcing (1), `IsValidGitHubRepositoryName` also accepts dots so real repositories such as `github/.github` are not rejected, while the relative path segments `.` and `..` remain invalid, and `raw.githubusercontent.com` is normalized to `github.com` in the parser (matching `pkg/cli/spec.go`) so API and git-remote URLs target the correct host.

Both fixes are applied at the point where untrusted data is consumed, consistent with the existing defense-in-depth approach already visible in the codebase (git subprocess arg sanitization, tar/zip traversal guards, path validation).

### Alternatives Considered

#### Alternative 1: Remove host-prefixed workflowspec support entirely

Drop the `host/owner/repo/path[@ref]` import format so that no parser-side host handling is needed. This eliminates the SSRF surface by removal rather than validation.

Not chosen because this format is the primary mechanism for referencing workflows on GitHub Enterprise Server (`*.ghe.com`) instances. Removing it would break a supported, documented use case for enterprise users. The allowlist approach preserves the feature for legitimate hosts while closing the SSRF gap.

#### Alternative 2: Reject the env dump unless `--verbose` is passed

Gate the `Environment Variables:` output block behind the existing `verbose` flag in `spawnMCPInspector`, so secrets are not printed by default.

Not chosen because this reduces but does not eliminate the exposure: users debugging MCP server issues routinely run with `--verbose` and would still see raw PAT values. Redaction preserves diagnostic utility (key names remain visible, non-sensitive values remain visible) while making the fix unconditional.

#### Alternative 3: Print only env key names (suppress values entirely)

Log `Environment Variables: [GITHUB_PERSONAL_ACCESS_TOKEN GH_TOKEN MY_VAR]` with no values at all.

Not chosen because non-sensitive variables (e.g., `LANG`, `PATH`, `NO_PROXY`) are genuinely useful for debugging MCP server misconfiguration. Suppressing all values is more aggressive than necessary; the heuristic pattern match retains non-sensitive values.

#### Alternative 4: Network-level egress filtering

Block outbound requests to non-GitHub hosts at the network layer (firewall, container policy) rather than in application code.

Not chosen because `gh aw` is a developer CLI that runs on uncontrolled user machines and in CI runners with varying network policies. Application-layer validation is the only enforcement mechanism that applies universally across all deployment contexts.

### Consequences

#### Positive
- SSRF attack surface for nested workflowspec imports is eliminated regardless of the caller's network environment.
- Enterprise tokens (`GH_ENTERPRISE_TOKEN`, `GITHUB_ENTERPRISE_TOKEN`) are no longer attached to requests to attacker-controlled hosts.
- GitHub PATs are no longer printed to stderr or CI logs during `gh aw mcp inspect` runs.
- `IsGitHubHost` is a named, tested function that unifies host validation logic previously duplicated between CLI and parser layers.

#### Negative
- Workflowspecs that reference a non-GitHub host (e.g., a private Gitea or Bitbucket instance) will now fail with an explicit validation error. This is by design — such hosts were never a documented supported case — but may surprise users who assembled such specs inadvertently.
- The sensitive-key heuristic (`token`, `key`, `auth`, etc.) may redact env vars that are not actually secret (e.g., `KEYBASE_USER`, `AUTHOR_NAME`). The cost is reduced diagnostic verbosity for those keys, not data loss.

#### Neutral
- Unit tests for both `IsGitHubHost`/`parseWorkflowSpecParts` host validation and `redactSensitiveEnvValues` are added alongside the production changes, increasing test surface in `pkg/parser` and `pkg/cli`.
- The pattern list in `sensitiveEnvKeyPatterns` is a package-level var, making it straightforward to extend if new sensitive key patterns are identified.

---

*ADR created for PR #50259 to satisfy the Design Decision Gate before human review and merge. Status becomes Accepted when the PR is merged.*
