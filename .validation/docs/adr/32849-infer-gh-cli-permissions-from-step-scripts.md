# ADR-32849: Infer gh CLI Permissions from Step Scripts in Activation and Agent Jobs

**Date**: 2026-05-17
**Status**: Draft
**Deciders**: pelikhan, Copilot

---

## Part 1 — Narrative (Human-Friendly)

### Context

The `gh aw compile` command emits an activation job and an agent job with `permissions:` blocks computed from workflow-level features (reaction targets, label events, safe outputs, etc.). Before this change, `buildActivationPermissions` ignored the contents of user-supplied `run` scripts in `jobs.activation.pre-steps`, `jobs.activation.steps`, and `jobs.activation.post-steps`, and the agent job's `buildMainJob` similarly ignored the contents of its `pre-steps`, `steps`, `pre-agent-steps`, and `post-steps`. As a result, a pre-step that called `gh pr diff "$PR_NUMBER" --name-only` would compile to a lock file with no `pull-requests: read` scope, causing the command to fail silently at runtime — Vale (and similar linters) appeared to lint nothing because the `gh` call returned an empty file list and exited non-zero without halting the job. Compounding the problem, an author could accidentally include a write `gh` command (e.g. `gh pr comment`) in a job whose permissions block is intentionally read-only, and the compiler would emit a workflow that fails opaquely at runtime rather than surfacing the misconfiguration at compile time.

### Decision

We will introduce a static, JSON-driven inference layer (`pkg/workflow/data/gh_cli_permissions.json` + `gh_cli_permissions.go`) that scans every `run:` script in the activation and agent jobs' step sections for `gh` subcommands and `gh api` REST path patterns, then merges the inferred minimum `permissions:` scopes into each job's compiled permission map. The scanner is regex-based, derives the `gh <group> <action>` alternation dynamically from the JSON's subcommand-group keys, and recognises two failure modes: (1) any inferred *write* command in either job is a compile-time error directing the author to safe-outputs, because both jobs run with read-only permissions by contract; (2) inferred *read* scopes are merged additively into the existing permission map, never overriding scopes already set by higher-level features. GitHub-App-only scopes (e.g. `codespaces`, `environments`, organisation membership) discovered in activation job scripts are additionally injected into the App token mint step so the minted token covers what the script will call.

### Alternatives Considered

#### Alternative 1: Require Manual `permissions:` Declarations from Workflow Authors

Make the workflow author responsible for declaring `permissions:` in the markdown frontmatter whenever a pre-step calls `gh`. This is the simplest implementation and avoids new parsing infrastructure. It was rejected because workflow authors are frequently non-engineers, and the failure mode (`gh pr diff` silently returning nothing) is sufficiently confusing that it would burn debugging time on every author who hit it. The whole point of compiler-derived permissions is that the lock file should match what the workflow actually does without authors having to keep two truth sources in sync.

#### Alternative 2: Grant Broad Permissions to Any Job with Pre-Step Scripts

Whenever a job has non-empty `pre-steps`, grant `pull-requests: read`, `issues: read`, `actions: read`, and `contents: read` unconditionally. This avoids parsing scripts entirely. It was rejected because it directly violates the principle of least privilege that the rest of the compiler enforces, and because it offers no path to detecting write commands at compile time — write `gh` calls would still fail opaquely at runtime, but now under broader scopes that obscure the misconfiguration.

#### Alternative 3: Full Shell Parser (e.g. `mvdan/sh`)

Use a real POSIX shell parser to extract `gh` invocations precisely, handling pipes, command substitution, conditionals, heredocs, and variable expansion correctly. This would eliminate the regex's known blind spots (e.g. `gh` invoked via `xargs`, indirection through shell variables). It was rejected because it adds a non-trivial dependency for a problem whose payoff is asymmetric — the regex catches the overwhelmingly common straight-line `gh <group> <action>` form, and the failure mode of missing an inference is the same as today (the author adds a manual `permissions:` line). The complexity-to-coverage ratio did not justify the dependency.

#### Alternative 4: Runtime Permission Probing

Have the activation job attempt the operation and escalate permissions on failure. Rejected for the same reasons as in ADR-26535: it requires network round-trips, complicates error handling, and shifts misconfiguration discovery from compile time to runtime. Compile-time inference is the consistent pattern across this compiler.

### Consequences

#### Positive
- Authors can call `gh pr diff`, `gh issue view`, `gh workflow list`, `gh api /repos/.../pulls/...`, etc. in pre-steps and the compiler will mint a lock file with the minimum required scopes — no manual `permissions:` line needed.
- Write `gh` commands in either job are now caught at compile time with an actionable error pointing to safe-outputs, eliminating an entire class of silent-failure misconfigurations.
- The permission-mapping table is data-driven JSON, so adding a new `gh` subcommand group (e.g. a future `gh discussion`) requires no Go code change beyond an entry in `gh_cli_permissions.json`.
- GitHub-App-only scopes (codespaces, environments, organisation members, webhooks) are inferred from `gh api` calls and injected into the App token mint step, closing a gap where App-only scripts previously had to be hand-wired.

#### Negative
- The regex-based scanner has known blind spots: `gh` invocations behind `xargs`, function indirection, dynamic command construction, or non-standard shell quoting will not be detected. Authors hitting these cases must still declare permissions manually, and there is no warning when this happens.
- The new permission inference runs on every compile and adds a small but non-zero amount of work even for workflows that don't use `gh` in pre-steps. Caching in `activationJobBuildContext` mitigates this within a single compile but does not amortise across compiles.
- The JSON file is now a new maintenance surface: when GitHub adds or renames a `gh` subcommand or REST path, `gh_cli_permissions.json` must be updated, and there is no automated check that it stays in sync with the actual `gh` CLI release.
- `pre-agent-steps` is scanned for the agent job but intentionally skipped for the activation job (the section has no meaning there). This asymmetry is documented in code comments but is a latent footgun for anyone adding new job types in the future.

#### Neutral
- The activation job and the agent job now share the same write-command detector and read-permission inferrer, applied through separate call sites. Future changes to inference rules need to be made in one place (`gh_cli_permissions.go`) but verified at two call sites.
- Existing workflows that already declared `permissions:` for `gh` operations will continue to work; the inferred scopes merge additively and do not override explicit declarations.
- The `permissions: {}` opt-out is preserved for the agent job: an author who explicitly zeroed out permissions will not have inferred scopes injected. The activation job has no equivalent opt-out because its permission block is compiler-owned.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### Permission Inference Source of Truth

1. Implementations **MUST** load the `gh` subcommand-to-permission mapping from `pkg/workflow/data/gh_cli_permissions.json` at package init time and **MUST NOT** hardcode subcommand-to-permission mappings elsewhere in the compiler.
2. The JSON schema **MUST** preserve the existing top-level fields: `version`, `description`, `subcommand_groups`, and `api_path_patterns`.
3. Each subcommand group entry **MUST** declare `read_subcommands`, `write_subcommands`, `read_permissions`, `write_permissions`, `app_read_permissions`, and `app_write_permissions`. Empty arrays are permitted.
4. Implementations **MUST** dynamically build the `gh <group>` regex alternation from the JSON's `subcommand_groups` keys, so that adding a new group requires no Go code change.

### Activation Job Inference Scope

1. Implementations **MUST** scan `run:` scripts in `jobs.activation.pre-steps`, `jobs.activation.steps`, and `jobs.activation.post-steps`.
2. Implementations **MUST NOT** scan `jobs.activation.pre-agent-steps`; `pre-agent-steps` is an agent-job-only concept and has no meaning in the activation job.
3. Inferred read permissions **MUST** be merged additively into the activation job's permission map and **MUST NOT** override scopes already set by workflow-level feature derivation.
4. GitHub-App-only scopes inferred from activation job scripts **MUST** be added to the App token mint step's `appPerms` when an App token is being minted.

### Agent Job Inference Scope

1. Implementations **MUST** scan `run:` scripts in the agent job's `pre-steps`, `steps`, `pre-agent-steps`, and `post-steps` sections, drawing from both the top-level fields (`data.PreSteps`, `data.CustomSteps`, `data.PreAgentSteps`, `data.PostSteps`) and the corresponding entries under `jobs.<agent-job-name>`.
2. Implementations **MUST NOT** inject inferred permissions when the workflow frontmatter declares `permissions: {}` (the explicit-empty opt-out), detected by exact-string match against the canonical YAML form.

### Write Command Detection

1. Implementations **MUST** treat any `gh` subcommand whose action is listed in a group's `write_subcommands` array as a write command.
2. When one or more write commands are detected in the activation job's scanned scripts, implementations **MUST** return a compile-time error naming each detected command and **MUST** include a reference to safe-outputs as the supported alternative.
3. When one or more write commands are detected in the agent job's scanned scripts, implementations **MUST** return a compile-time error naming each detected command and **MUST** include a reference to safe-outputs as the supported alternative.
4. Implementations **MUST NOT** silently grant write scopes inferred from script content, regardless of the existing permission map.

### Read Permission Inference

1. When a recognised read subcommand (e.g. `gh pr diff`, `gh issue view`) is found, implementations **MUST** infer the scopes listed in the group's `read_permissions`.
2. When a `gh api` invocation matches a configured `api_path_patterns` regex, implementations **MUST** infer the scopes listed in that pattern's `permissions` (and `app_permissions` for App tokens) field.
3. Implementations **MAY** fall back to the group-level `read_permissions` when an unrecognised action follows a known subcommand group (e.g. `gh pr something-new`); this fallback **SHOULD NOT** be relied on for newly added actions and the JSON **SHOULD** be updated to list new actions explicitly.

### Caching

1. Within a single activation-job build, implementations **MUST** cache the extracted run scripts and inferred permissions on the `activationJobBuildContext` so the scan is not repeated by callers that need the same data (e.g. both `buildActivationPermissions` and `addActivationFeedbackAndValidationSteps`).

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Failure to meet any **MUST** or **MUST NOT** requirement constitutes non-conformance.

---

*This is a DRAFT ADR generated by the [Design Decision Gate](https://github.com/github/gh-aw/actions/runs/26000015148) workflow. The PR author must review, complete, and finalize this document before the PR can merge.*
