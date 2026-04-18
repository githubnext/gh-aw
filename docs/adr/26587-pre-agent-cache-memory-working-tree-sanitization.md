# ADR-26587: Pre-Agent Cache-Memory Working-Tree Sanitization

**Date**: 2026-04-16
**Status**: Draft
**Deciders**: pelikhan, Copilot

---

## Part 1 — Narrative (Human-Friendly)

### Context

The cache-memory system restores files from a prior agent run into a working directory before the current agent executes. With `integrity: none` (the default configuration), this restoration happens before any threat-detection gate fires. A compromised prior run could therefore plant executable scripts (e.g. `helper.sh`) or unexpected file types in the cache, which the next agent would encounter without any validation. The system needed a deterministic security gate that runs *after* the cache is restored but *before* the agent can access the working tree.

### Decision

We will extend the existing `setup_cache_memory_git.sh` shell script—which already runs in the correct position (after cache restore, before agent execution)—to perform two sanitization steps: (1) unconditionally strip execute bits from every working-tree file so planted executables cannot run, and (2) when `GH_AW_ALLOWED_EXTENSIONS` is configured (a colon-separated list passed by the Go code in `generateCacheMemoryGitSetupStep`), remove any file whose extension is not on the allowed list. This approach adds a zero-new-step security gate without restructuring workflow templates.

### Alternatives Considered

#### Alternative 1: Separate Dedicated "Sanitize Cache" Workflow Step

A standalone step could be inserted into every generated workflow immediately after the cache-restore step. This would be visible in the YAML and independently auditable. However, it requires patching every generated workflow template, increases generated-YAML complexity, and creates a step-ordering dependency that is harder to enforce correctly across all cache configurations. Centralising the logic in the existing git-setup script avoids these problems.

#### Alternative 2: Go-Level File System Sanitization

The sanitization could be implemented in Go within `cache.go` at workflow-generation time or injected as a Go binary invoked from the step. This would keep all business logic in a single language and allow richer error handling. It was rejected because the shell script already executes in the exact lifecycle position required, and introducing a compiled Go binary just for chmod/rm operations would add build and distribution overhead with no material benefit over shell primitives.

#### Alternative 3: Content-Based Threat Scanning

Rather than filtering by permission bits or file extension, the gate could scan file content for known-malicious patterns (YARA rules, signature checks, etc.). This provides stronger detection for obfuscated threats but introduces false-positive risk, brittle pattern maintenance, and significant latency. The permission-strip + extension-filter approach provides meaningful defence-in-depth with deterministic, auditable behaviour and no false positives.

#### Alternative 4: Rely on Existing Threat Detection

The existing integrity and threat-detection pipeline could be trusted to catch malicious content. This was rejected because threat detection fires concurrently with or after agent execution—not unconditionally before it—so a planted executable could be invoked before detection fires. The pre-agent sanitization gate closes this race condition.

### Consequences

#### Positive
- Eliminates a class of supply-chain-style attacks where a compromised prior run plants executable files in cache-memory.
- The execute-bit strip provides unconditional defence even when no `allowed-extensions` list is configured.
- Extension filtering allows teams to explicitly declare a minimal allowed surface area for cached files.
- No additional workflow step is emitted; existing generated workflows are not structurally changed.
- Unit tests in `cache_integrity_test.go` cover the env-var emission contract, making the behaviour regression-safe.

#### Negative
- `chmod a-x` runs on every cache-memory-enabled workflow unconditionally, even when no security concern exists—a small overhead on each run.
- Any legitimate workflow that relies on executable bits being preserved in cached files (unlikely given cache-memory is data-only by design) would break silently—files are de-executed without notification.
- The extension filtering uses a colon-separated string passed through an env var, which is a slightly brittle interface compared to a structured config file; misconfigured extension strings (e.g. missing leading dot) would silently allow all files.

#### Neutral
- The sanitization is implemented in Bash within the existing setup script rather than as a new architectural component, keeping the footprint minimal.
- The `GH_AW_ALLOWED_EXTENSIONS` env var is only emitted when `AllowedExtensions` is non-empty; the default (all extensions allowed) requires no configuration change.
- The design naturally separates concerns: Go code owns the configuration contract, the shell script owns the filesystem operations.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### Pre-Agent Sanitization Execution Order

1. The sanitization gate **MUST** execute after the cache-restore step and before any agent step can access the cache working tree.
2. The sanitization gate **MUST NOT** be skipped or made conditional on the `GH_AW_MIN_INTEGRITY` level; it **MUST** run for all integrity configurations, including `none`.
3. The sanitization logic **MUST** be implemented in `setup_cache_memory_git.sh` or a script invoked by it, so that it runs in the same step as the git-setup work.

### Execute-Bit Stripping

1. Implementations **MUST** strip execute permissions (`chmod a-x`) from every file in the cache working tree, excluding files inside `.git/`.
2. Implementations **MUST NOT** require any configuration to enable execute-bit stripping; it **MUST** be active for all cache-memory workflows.
3. Implementations **SHOULD** log a confirmation message after stripping execute permissions so that audit logs confirm the step ran.

### Extension-Based File Filtering

1. When the `GH_AW_ALLOWED_EXTENSIONS` environment variable is set and non-empty, implementations **MUST** remove any working-tree file (excluding `.git/`) whose extension is not present in the colon-separated allowed list.
2. Extensions in `GH_AW_ALLOWED_EXTENSIONS` **MUST** include the leading dot (e.g. `.json`, `.md`); files with no extension are treated as if their extension is an empty string.
3. When `GH_AW_ALLOWED_EXTENSIONS` is absent or empty, implementations **MUST NOT** remove any files based on extension (all extensions are allowed).
4. Implementations **SHOULD** log each removed file path and its extension, and **SHOULD** log a final count of removed files.

### Go-to-Shell Configuration Contract

1. The Go code in `generateCacheMemoryGitSetupStep` **MUST** emit `GH_AW_ALLOWED_EXTENSIONS` as an environment variable to the git-setup step when `cache.AllowedExtensions` is non-empty.
2. The env-var value **MUST** be the `AllowedExtensions` slice joined with `:` as separator.
3. The Go code **MUST NOT** emit `GH_AW_ALLOWED_EXTENSIONS` when `AllowedExtensions` is nil or empty; the absence of the variable is the signal for "allow all extensions."

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Failure to meet any **MUST** or **MUST NOT** requirement constitutes non-conformance.

---

*This is a DRAFT ADR generated by the [Design Decision Gate](https://github.com/github/gh-aw/actions/runs/24514620900) workflow. The PR author must review, complete, and finalize this document before the PR can merge.*
