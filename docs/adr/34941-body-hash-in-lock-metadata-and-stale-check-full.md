# ADR-34941: Body Hash in Lock Metadata and `stale-check: full` Runtime Verification

**Date**: 2026-05-26
**Status**: Draft
**Deciders**: Unknown

---

## Part 1 — Narrative (Human-Friendly)

### Context

Compiled `.lock.yml` files previously embedded a `frontmatter_hash` in their `gh-aw-metadata` header but did not hash the markdown prompt body or any transitively imported files. As a result, an author could edit the prompt body — or any file pulled in via an `@import` directive — without changing the frontmatter, and the lock file would still be considered "fresh" by both the compiler skip-write optimization and the runtime staleness check. There was no detection signal for body-only drift, so production workflows could silently run against a body that no longer matched what was compiled into the lock. The skip-write optimization (which preserves the lock file timestamp when content is unchanged) further masked the issue, because body-only edits did not flip any metadata field and so produced no rewrite. We need a way to detect body drift at compile time and, on an opt-in basis, at activation time on the runner.

### Decision

We will add a `body_hash` field to lock metadata under a new schema version `v4`, covering the SHA-256 of the normalized body texts (main body and all transitively imported bodies, sorted and joined with `\n---\n` delimiter) as a single opaque hash string, and introduce a workflow-level `on.stale-check: full` frontmatter option that injects `GH_AW_STALE_CHECK_FULL=true` into the activation step to enable runtime body-hash verification on both the GitHub API and local filesystem fallback paths. The body hash is computed alongside the existing frontmatter hash at compile time via a shared `computeWorkflowHash` closure; failure to compute the body hash is non-fatal (we record the frontmatter hash and continue). Pre-v4 lock files without `body_hash` remain valid because the field is `omitempty` and the runtime comparator skips body comparison when the lock metadata does not carry one. The lock schema version is bumped to `v4` and `GenerateLockMetadata` now takes a `LockHashInfo` struct instead of positional hash parameters.

### Alternatives Considered

#### Alternative 1: Combine frontmatter and body into a single workflow hash

We could have replaced `frontmatter_hash` with a single composite hash covering frontmatter + body + imports, eliminating the need for a new field. Rejected for two reasons: (1) it would have been a breaking metadata change requiring all existing lock files to be regenerated before runtime checks would pass, with no graceful fallback; and (2) keeping the hashes separate preserves the existing fast-path frontmatter-only check (cheap) and lets the more expensive body check (which must read every imported file) run only when explicitly opted in via `stale-check: full`.

#### Alternative 2: Always verify the body hash at runtime (no opt-in)

We could have made body verification unconditional whenever a `body_hash` field is present in the lock. Rejected because body verification on the local filesystem fallback path requires reading every transitively imported file on every workflow activation, which is non-trivial I/O for workflows with deep import graphs. Making it opt-in via `stale-check: full` lets workflow authors choose between fast activation (frontmatter-only) and strict drift detection (full body), and avoids regressing activation latency for the majority of workflows that don't need body-level guarantees.

#### Alternative 3: Detect body drift only at compile time (no runtime signal)

We could have stopped at recording the body hash in metadata, relying on developers to recompile before committing. Rejected because the existing failure mode is precisely that compiled lock files are checked into the repo while the source workflow has been edited downstream of the last compile — a CI-only signal would not catch a workflow being activated on a runner with stale lock content. A runtime check is the only way to fail closed when source and lock disagree at activation time.

### Consequences

#### Positive
- Body-only edits to a workflow or any of its imported files now produce a different lock file (body hash changes), so the compiler's skip-write optimization correctly triggers a rewrite. The new `TestCompilerWritesWhenBodyContentChanged` test pins this behavior.
- Workflows can opt into full drift detection at runtime by adding `on.stale-check: full` to their frontmatter (under the `on:` section), fail-closing if the lock body hash diverges from the live source's body hash on either the GitHub API path or the local filesystem fallback path.
- Backwards compatible: pre-v4 lock files without `body_hash` continue to validate (the field is `omitempty` on the Go side; the JS runtime gracefully skips body comparison when the field is absent).
- The lock metadata schema is now explicitly versioned at `v4`, giving us a clean discriminator for future metadata extensions.

#### Negative
- Compilation now reads every transitively imported file in addition to the main workflow, adding I/O to compile time proportional to the import graph depth. Workflows with large or deep imports will see compile latency rise.
- The new `LockHashInfo` struct is a breaking change to the internal `GenerateLockMetadata` signature; any out-of-tree caller or test that builds lock metadata directly must migrate from positional hash arguments to the struct form.
- Runtime activation now has a second hash to verify when `stale-check: full` is set, adding file reads or API calls during the "Check workflow lock file" step. Worst case (full mode + many imports + local fallback path) requires reading every imported file at activation time.
- The `stale-check` frontmatter field now accepts both `boolean` and the literal string `"full"`, adding a small amount of schema and parser complexity.

#### Neutral
- The body-hash failure mode at compile time is non-fatal: if body-hash computation fails (e.g., a transient I/O error on an imported file), the compile continues with only the frontmatter hash recorded. This is a deliberate "best-effort" choice; downstream code must tolerate a v4 lock with no `body_hash`.
- The `body_hash` is a single opaque SHA-256 hex string computed by concatenating the normalized body text and all normalized imported body texts (sorted, joined with `\n---\n`) and hashing the result directly — no JSON envelope wrapper.
- The `computeWorkflowHash` closure deduplicates the parsed-content-first-with-file-fallback pattern shared between frontmatter and body hashing; future hash computations (if added) should reuse this helper rather than reimplementing the fallback.
- Imported file bodies are sorted before being joined, so the body hash is independent of the order in which imports were resolved.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### Lock Metadata Schema

1. The lock metadata header **MUST** declare `schema_version: "v4"` when a `body_hash` field is present.
2. When the compiler successfully computes a body hash, it **MUST** include the `body_hash` field in the lock metadata.
3. The `body_hash` field, when present, **MUST** be the lowercase hex SHA-256 of the concatenation of the normalized main body text and the normalized bodies of all transitively imported files (sorted, joined with `\n---\n` delimiter), treated as a single opaque string.
4. The lock metadata header **MUST** retain the existing `frontmatter_hash` field independently of `body_hash`.
5. Consumers of lock metadata **MUST** treat the absence of `body_hash` as a valid pre-v4 lock and **MUST NOT** fail validation solely because the field is missing.
6. The compiler **MUST NOT** abort lock generation if body-hash computation fails; it **MUST** instead omit `body_hash` and continue with the frontmatter hash alone.

### Compile-Time Body Hash Computation

1. The compiler **MUST** include the bodies of all transitively imported files when computing the body hash.
2. Imported file bodies **MUST** be sorted before being joined, so that ordering of imports does not affect the hash.
3. Body-hash computation **SHOULD** prefer already-parsed in-memory content over re-reading from disk when both are available; the shared `computeWorkflowHash` helper **MUST** implement the parsed-content-first, file-fallback pattern.
4. The internal `GenerateLockMetadata` function **MUST** accept hash inputs as a `LockHashInfo` struct rather than as positional parameters.

### `stale-check` Frontmatter

1. The `on.stale-check` frontmatter field (nested under the `on:` section) **MUST** accept the values `true`, `false`, and the literal string `"full"` (per `main_workflow_schema.json`).
2. When `on.stale-check: full` is set on a workflow, the compiler **MUST** set the internal `StaleCheckFull` flag and **MUST** inject `GH_AW_STALE_CHECK_FULL=true` into the environment of the activation job's "Check workflow lock file" step.
3. The compiler **MUST NOT** inject `GH_AW_STALE_CHECK_FULL` when `on.stale-check` is unset, `false`, or `true` (the boolean form continues to behave as before).

### Runtime Body Hash Verification

1. When `GH_AW_STALE_CHECK_FULL=true`, the runtime checker **MUST** verify the body hash in addition to the existing frontmatter hash, on both the GitHub API path and the local filesystem fallback path.
2. The runtime **MUST** compute the live body hash using the same concatenation and sorting rules used at compile time, so that compile-time and runtime hashes are byte-identical for matching content.
3. The runtime **MUST** perform body hash verification only after the frontmatter hash check passes; it **MUST NOT** report a body-hash mismatch when the frontmatter hash already mismatches.
4. The runtime **MUST** gracefully skip body comparison when the lock file is pre-v4 or otherwise does not contain a `body_hash` field, even if `GH_AW_STALE_CHECK_FULL=true`.
5. The runtime body-hash comparison helper **MUST** receive its file-reading dependency by destructured injection (e.g. `{ fileReader }`) rather than by global reference, so the comparator remains testable in isolation.
6. When `GH_AW_STALE_CHECK_FULL` is unset or not `true`, the runtime **MUST** preserve prior behavior and **MUST NOT** verify the body hash.

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Failure to meet any **MUST** or **MUST NOT** requirement constitutes non-conformance.

---

*This is a DRAFT ADR generated by the [Design Decision Gate](https://github.com/github/gh-aw/actions/runs/26458153802) workflow. The PR author must review, complete, and finalize this document before the PR can merge.*
