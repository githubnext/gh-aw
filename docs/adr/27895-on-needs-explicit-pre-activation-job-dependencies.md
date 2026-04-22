# ADR-27895: Introduce `on.needs` for Explicit Pre-Activation Job Dependencies

**Date**: 2026-04-22
**Status**: Draft
**Deciders**: pelikhan

---

## Part 1 — Narrative (Human-Friendly)

### Context

Workflow frontmatter supports `on.github-app` credentials that allow the activation job to mint a short-lived GitHub App token. Some security postures require fetching the App ID and private key from an external secret manager at runtime (e.g., HashiCorp Vault, AWS Secrets Manager) via a dedicated job that runs before activation. Prior to this change there was no way to declare such a dependency explicitly: the `pre_activation` and `activation` jobs always ran as the earliest jobs in the graph, making `${{ needs.<job>.outputs.* }}` expressions in `on.github-app` always resolve to empty values at runtime.

### Decision

We will add an `on.needs` array field to the workflow frontmatter `on:` section. Jobs listed in `on.needs` are wired as explicit dependencies of both `pre_activation` and `activation` so that their outputs are available before credential resolution occurs. Jobs in `on.needs` are excluded from the automatic `needs: activation` guard that would normally force custom jobs to run after activation. Validation ensures that only declared custom jobs (not built-in control jobs) can appear in `on.needs`, and that `on.github-app` expression references resolve exclusively to jobs available before activation.

### Alternatives Considered

#### Alternative 1: Auto-detect credential-supply jobs from `on.github-app` expressions

The compiler could parse `${{ needs.<job>.outputs.* }}` expressions in `on.github-app` fields and automatically promote those jobs to pre-activation dependencies without any user declaration. This approach was not chosen because it would make dependency wiring implicit and hard to reason about: a typo or expression change could silently break the dependency graph in non-obvious ways. Explicit declaration (`on.needs`) keeps the dependency contract visible in the frontmatter.

#### Alternative 2: Require credential-supply logic to be inlined as `on.steps`

The existing `on.steps` mechanism allows injecting arbitrary steps into the pre-activation job. Users could fetch credentials there instead of in a separate job. This was not chosen because it conflates credential supply with pre-activation gate logic, prevents parallel execution of credential-fetch and other pre-activation checks, and does not work for teams that already have standalone credential-supply jobs they want to reuse across multiple workflows.

### Consequences

#### Positive
- `on.github-app` expressions can now reference `needs.<job>.outputs.*` values from jobs that run before activation, enabling dynamic credential supply from external secret managers.
- Validation at compile time rejects invalid `on.needs` entries (built-in job names, cycle-prone jobs, undeclared jobs), turning silent runtime failures into clear compiler errors.

#### Negative
- Jobs listed in `on.needs` are exempt from the automatic `needs: activation` safeguard, meaning they run before the activation gate. This widens the surface of pre-activation execution, which must be considered when auditing workflow security.
- Introduces a new top-level frontmatter concept (`on.needs`) that users must learn; documentation and validation errors must be clear enough to avoid confusion with GitHub Actions' job-level `needs:` field.

#### Neutral
- When `on.needs` is non-empty and no other pre-activation checks exist, `pre_activation` is forced to be created (even if it would otherwise be omitted), so that `on.needs` jobs are properly sequenced before `activation`.
- The `activated` output of `pre_activation` is set unconditionally to `"true"` when the job is created solely due to `on.needs` (no permission checks, stop-time, skip-if, etc.), consistent with existing `on.steps`-only behaviour.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### Schema and Parsing

1. Implementations **MUST** accept `on.needs` as an optional array of strings in the workflow frontmatter `on:` section.
2. Each entry in `on.needs` **MUST** match the pattern `^[a-zA-Z_][a-zA-Z0-9_-]*$`.
3. `on.needs` **MUST** contain unique entries (no duplicates).
4. If `on.needs` is absent or `null`, implementations **MUST** treat it as an empty list and **MUST NOT** alter the dependency graph.

### Compiler Dependency Wiring

1. Implementations **MUST** add every job named in `on.needs` to the `needs` list of the `pre_activation` job.
2. Implementations **MUST** add every job named in `on.needs` to the `needs` list of the `activation` job (merged with any existing before-activation dependencies).
3. Implementations **MUST NOT** add an implicit `needs: activation` dependency to any job that is listed in `on.needs`.
4. If `on.needs` is non-empty, implementations **MUST** create a `pre_activation` job even if no other pre-activation checks (permission, stop-time, skip-if, on.steps) are present.
5. When `pre_activation` is created solely because `on.needs` is non-empty (no other checks), the `activated` output **MUST** be set unconditionally to `"true"`.

### Validation

1. Implementations **MUST** reject any `on.needs` entry that references a built-in or compiler-generated job ID (e.g., `pre_activation`, `activation`).
2. Implementations **MUST** reject any `on.needs` entry that references a job that already depends on `pre_activation` or `activation`, to prevent dependency cycles.
3. Implementations **MUST** reject any `on.needs` entry that does not correspond to a job declared in the top-level `jobs:` section.
4. When `on.github-app` fields contain `${{ needs.<job>.outputs.* }}` expressions, implementations **MUST** verify that the referenced job is available before activation (i.e., listed in `on.needs` or otherwise before-activation). Implementations **MUST** emit a compiler error if the referenced job would run after activation.
5. Implementations **SHOULD** emit descriptive error messages that distinguish `on.needs` validation failures from job-level `needs:` validation failures.

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Failure to meet any **MUST** or **MUST NOT** requirement constitutes non-conformance.

---

*This is a DRAFT ADR generated by the [Design Decision Gate](https://github.com/github/gh-aw/actions/runs/24806829131) workflow. The PR author must review, complete, and finalize this document before the PR can merge.*
