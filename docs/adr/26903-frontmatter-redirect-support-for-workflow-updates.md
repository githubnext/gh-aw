# ADR-26903: Frontmatter Redirect Support for Workflow Updates

**Date**: 2026-04-17
**Status**: Draft
**Deciders**: pelikhan, Copilot

---

## Part 1 — Narrative (Human-Friendly)

### Context

Agentic workflows installed via `gh aw update` track their upstream source using a `source` frontmatter field. When a workflow is renamed, relocated, or replaced by a newer version in the upstream repository, consumers have no automated way to discover the new location — the next `gh aw update` continues fetching the old (potentially stale or deleted) path. The system needs a lightweight mechanism for upstream workflow authors to declare that a workflow has moved, and for the update command to transparently follow that declaration without requiring manual consumer intervention.

### Decision

We will introduce a `redirect` frontmatter field in the workflow schema that points to the new canonical location of a workflow. When `gh aw update` encounters a workflow whose upstream content declares a `redirect`, it will follow the redirect chain (up to a depth of 20), rewrite the local `source` field to the resolved destination, and disable 3-way merge for the hop to avoid spurious conflicts. A `--no-redirect` flag is added so operators who require explicit control over source changes can refuse any update that would silently relocate a workflow.

### Alternatives Considered

#### Alternative 1: Out-of-band redirect registry

Maintain a central registry (e.g., a JSON file in the upstream repo or a service endpoint) that maps old workflow paths to new ones. The update command would consult this registry before fetching. This was not chosen because it requires additional infrastructure, creates a coupling to a registry format that all upstream repos must adopt, and does not compose well with forks or private repositories.

#### Alternative 2: HTTP-style redirect at the content-hosting layer

Rely on GitHub repository redirects (e.g., redirecting the raw content URL via a GitHub redirect when a file is moved). This was not chosen because GitHub does not provide automatic content-level HTTP redirects for individual file paths within a repository, and even if it did, the redirect would lose the semantic information needed to update the `source` field in the local workflow file.

#### Alternative 3: Require manual consumer re-pinning

Document that when a workflow moves, consumers must manually run `gh aw add` with the new location and delete the old file. This was not chosen because it places the burden on every consumer rather than the upstream author, and silent staleness is a worse outcome than a transparent automated redirect.

### Consequences

#### Positive
- Upstream workflow authors can declare a move once, and all consumers transparently follow it on the next `gh aw update` run.
- The `source` field in consumer files is automatically rewritten to the resolved canonical location, keeping provenance accurate.
- Redirect loops (A → B → A) are detected and reported rather than spinning indefinitely.
- The `--no-redirect` flag gives security- or stability-conscious operators an explicit opt-out with a clear error message.
- The compiler emits an informational message when compiling a workflow that has a `redirect` configured, making the stub status visible during local development.

#### Negative
- Redirect chains are followed silently at update time; consumers may not notice that the source of their workflow has changed unless they inspect the diff.
- Disabling 3-way merge on redirect hops means local customizations to a redirected workflow will be overwritten on the first update after a redirect is followed.
- The maximum redirect depth (20) is an arbitrary constant; very long chains will fail rather than succeed.
- Adding `noRedirect bool` to the already-long parameter list of `UpdateWorkflows` / `updateWorkflow` / `RunUpdateWorkflows` increases function arity and slightly complicates call sites.

#### Neutral
- The `redirect` field is defined in the JSON schema for workflow frontmatter, so existing schema validation tooling will recognize it without special-casing.
- The `FrontmatterConfig` struct gains a `Redirect` field that is serialized/deserialized symmetrically with the existing `Source` field.
- No changes are required to lock file format or compilation output; the redirect field is used only at update time and during compiler validation messages.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### Redirect Field

1. The `redirect` field **MUST** be a string when present in workflow frontmatter; non-string values **MUST** cause the update to fail with a descriptive error.
2. The `redirect` field value **MUST** be either a workflow source spec (`owner/repo/path@ref`) or a GitHub blob URL (`https://github.com/owner/repo/blob/ref/path`); other formats **MUST** be rejected with a parse error.
3. The `redirect` field **MUST NOT** point to a local path or a non-remote location; redirect targets **MUST** resolve to a remote repository slug.

### Redirect Chain Resolution

1. Implementations **MUST** resolve redirect chains iteratively, following each `redirect` field until a workflow without a redirect is reached.
2. Implementations **MUST** detect redirect loops using a visited-location set and **MUST** return an error when a previously visited location is encountered.
3. Implementations **MUST NOT** follow more than 20 redirect hops; exceeding this limit **MUST** result in an error.
4. Implementations **MUST** rewrite the local `source` field to the fully resolved final location after following a redirect chain.
5. When a redirect is followed, implementations **MUST** disable 3-way merge for that update and override the local file with the redirected content.
6. Implementations **SHOULD** emit a warning message to stderr for each redirect hop followed, naming the source and destination locations.

### `--no-redirect` Flag

1. When `--no-redirect` is specified, implementations **MUST** refuse any update where the upstream workflow declares a `redirect` field, and **MUST** return a non-zero exit code with an explanatory error message.
2. The error message **MUST** identify the workflow name, the current upstream location, and the redirect target so the operator understands what redirect was refused.
3. When `--no-redirect` is not specified, redirect following **MUST** be the default behavior.

### Compiler Behavior

1. The compiler **MUST** emit an informational message to stderr when compiling a workflow whose frontmatter contains a non-empty `redirect` field.
2. The informational message **MUST** include the redirect target value so developers are aware the compiled file is a redirect stub.
3. The presence of a `redirect` field **MUST NOT** cause compilation to fail; it is advisory metadata only.

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Failure to meet any **MUST** or **MUST NOT** requirement constitutes non-conformance.

---

*This is a DRAFT ADR generated by the [Design Decision Gate](https://github.com/github/gh-aw/actions/runs/24575079707) workflow. The PR author must review, complete, and finalize this document before the PR can merge.*
