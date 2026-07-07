# ADR-43909: Follow Symlinks in Sparse Checkout for .github Subdirectories

**Date**: 2026-07-07
**Status**: Draft
**Deciders**: Unknown

---

### Context

The workflow compiler builds sparse-checkout configurations for `.github` subdirectories (`.github/agents`, `.github/skills`, `.github/prompts`) when generating activation jobs. Some repositories use git symlinks at these paths to point to directories elsewhere in the repo (e.g. `.github/agents → ../.ai/agents`). Previously, the compiler added only the symlink blob path to the sparse-checkout manifest; at runtime this produced a dangling symlink, causing any `{{#runtime-import .github/agents/<agent>.md}}` expression to fail silently. No mechanism existed to detect this case or include the symlink's target directory in the checkout.

### Decision

We will add a `resolveSymlinkExtraPaths` function that inspects `.github/agents`, `.github/skills`, and `.github/prompts` at compile time using `os.Lstat` and `os.Readlink`. When a candidate path is a symlink, the resolved target (relative to the repository root via `filepath.Clean(filepath.Join(dir, target))`) is appended to `extraPaths` before the sparse-checkout step is generated. Symlink targets whose resolved path begins with `..` are silently rejected to prevent path traversal. Paths already present in `extraPaths` are deduplicated via a map.

### Alternatives Considered

#### Alternative 1: Require explicit user configuration

Repository owners could be required to enumerate symlink targets explicitly in a configuration file or in their sparse-checkout definition. This pushes the responsibility to every affected repository and is error-prone: users must know the internal structure of the compiled sparse-checkout and keep their configuration in sync as symlink targets change. Given that the compiler already resolves extra paths automatically for other cases, silent auto-detection is the lower-friction approach.

#### Alternative 2: Resolve all symlinks generically across extraPaths

Instead of inspecting a fixed set of three candidates, the compiler could recursively follow symlinks for every entry in `extraPaths`. This would cover any future well-known paths without code changes. However, unbounded symlink traversal increases the risk of accidentally including unintended directories, makes the behavior harder to reason about, and complicates path-escape validation. Limiting resolution to the three known subdirectories keeps the scope narrow and the security surface small.

### Consequences

#### Positive
- `runtime-import` expressions that reference `.github/agents`, `.github/skills`, or `.github/prompts` now work correctly when those paths are git symlinks.
- Path traversal is explicitly blocked: symlink targets resolving outside the repository root are ignored with a log message, preserving security guarantees.
- Deduplication ensures the generated sparse-checkout manifest does not contain redundant entries when a target is already present.

#### Negative
- `resolveSymlinkExtraPaths` uses relative paths (`os.Lstat(".github/agents")`) and implicitly assumes the compiler's working directory is the repository root at the time of compilation. If the working directory differs, all three `os.Lstat` calls fail silently and no symlink targets are added — the bug reappears without any error surfaced to the caller.
- Only the three hard-coded candidate paths are inspected. Other symlinked `.github` subdirectories (e.g. `.github/workflows` or custom subdirectories) are not covered and would require separate changes.

#### Neutral
- The function is append-only: it does not modify or remove existing `extraPaths` entries, preserving backward compatibility with all callers.
- The `resolveSymlinkExtraPaths` call is placed immediately before both `GenerateGitHubFolderCheckoutStep` invocations via the single call-site in `generateCheckoutGitHubFolderForActivation`, so all callers inherit the fix without further changes.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
