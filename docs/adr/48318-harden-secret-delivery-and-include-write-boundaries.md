# ADR-48318: Harden Secret Delivery via Stdin and Enforce Include Write Boundaries

**Date**: 2026-07-27
**Status**: Draft
**Deciders**: Unknown (generated from PR #48318 diff)

---

### Context

VulnHunter flagged two credible security issues in the `gh-aw` CLI tool. First, calls to `gh secret set` passed secret values as a command-line argument via `--body <value>`, making credential material visible in process listings, shell history, and debug logs on the host machine. Second, the `fetchAndSaveRemoteIncludes` function allowed remote `@include` directives to specify paths that, after resolution, could escape the intended write directories (`.github/workflows` and `.github/shared`), enabling a path traversal attack that could overwrite arbitrary files on the local filesystem.

Both issues affect the `pkg/cli` package and arise from missing or incorrect input validation at security boundaries: process argument handling and filesystem write operations.

### Decision

We will pass secret values to `gh secret set` exclusively via stdin using `RunGHInputContext` with `--body -`, eliminating argv exposure. We will also add a pre-write boundary check in `fetchAndSaveRemoteIncludes` using `fileutil.ValidatePathWithinBase` to reject any resolved target path that falls outside the allowed base directory for that include type (workflows directory for relative includes, shared directory for shared/workflowspec includes).

### Alternatives Considered

#### Alternative 1: Sanitize or Redact Secret Values Before Passing via Argv

Secret values could be escaped, base64-encoded, or otherwise transformed before being passed as a command-line argument. This avoids changing the call interface but still places the transformed material in the process argument list, which remains readable in `/proc/<pid>/cmdline` and system audit logs during the brief execution window. It does not eliminate the exposure — it only obfuscates it — and any transformation must be reversed by the receiving process, adding complexity without a real security gain.

#### Alternative 2: Write Secrets to a Temporary File and Pass the File Path

The secret value could be written to a `mktemp`-created file, passed to `gh secret set` via `--body @<file>`, and then deleted. This keeps the secret out of argv but introduces new risks: the temp file may be readable by other users, the deletion may fail leaving the file on disk, and the lifecycle management adds code complexity. Stdin is simpler, ephemeral by nature, and the accepted standard for passing sensitive material to subprocesses.

#### Alternative 3: Reject All Relative Include Paths (Only Allow Workflowspec Format)

Relative `@include` paths (e.g., `@include helper.md`) could be entirely disallowed, requiring all includes to use fully-qualified workflowspec format. This eliminates the traversal surface for relative paths but breaks the existing documented feature of including workflow fragments co-located with the main workflow file, which is a common and legitimate use case. Restricting to a boundary check preserves the feature while blocking traversal.

#### Alternative 4: Strip Path Traversal Sequences from the Include Path Before Resolution

Normalize the include path by removing `..` components before computing the target path. Allowlist-style normalization is error-prone: character-encoding tricks (e.g., `%2e%2e`), symlink chains, or double-slash sequences can bypass naive stripping. Validating the fully-resolved target path against the base directory after all joins are computed is the canonical defense (TOCTOU-safe) and is already implemented by `fileutil.ValidatePathWithinBase`.

### Consequences

#### Positive
- Secret values no longer appear in process argv (`/proc/<pid>/cmdline`), shell history for the parent process, or system audit logs at the point of the `gh` subprocess invocation.
- Malicious or misconfigured remote `@include` paths can no longer cause writes outside `.github/workflows` or `.github/shared`, blocking the path traversal attack class.
- Both fixes are covered by targeted regression tests that verify the security property directly (args log inspection for stdin delivery; `NoFileExists` assertion for traversal rejection).

#### Negative
- All `gh secret set` call sites must use the `RunGHInputContext` API variant, which requires a `context.Context` parameter; call sites without an available context must use `context.Background()` as a fallback, which is slightly less cancellation-friendly.
- The write boundary check may reject edge-case include paths that were previously silently accepted (e.g., a path that resolves to a sibling of the target directory); authors of such includes will need to rewrite the path in a supported format.

#### Neutral
- The `fetchIncludeFromSource` package-level variable introduced to allow test injection is a dependency-inversion seam, consistent with how `downloadRemoteImportFile` is already structured; this is a test-enabling pattern, not a production-behavior change.
- Both changes are backward-compatible with the external `gh` CLI interface — only the Go call sites inside `pkg/cli` are affected.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
