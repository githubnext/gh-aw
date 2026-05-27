# ADR-35091: Reorganize Workflow and Parser Packages by Semantic Clustering

**Date**: 2026-05-27
**Status**: Draft
**Deciders**: pelikhan, Copilot

---

## Part 1 — Narrative (Human-Friendly)

### Context

Semantic clustering of 858 non-test Go files surfaced four organizational issues that violate the file-naming and placement conventions established in ADR-27325 and extended by ADR-28282 and ADR-29336. First, `pkg/workflow/compiler_safe_outputs.go` contained six functions whose semantic domains (trigger parsing, default tools, sandbox state) did not match the file's name. Second, the `pkg/workflow` directory had a split-brain naming pattern where some files used the `compiler_safe_outputs_*` prefix and others used `safe_outputs_*` for the same conceptual area, including three thin files with one or fewer functions each. Third, MCP rendering file names mixed `mcp_rendering.go` (a renderer factory) with `mcp_config_builtin.go` (which actually contained renderer functions, not config). Fourth, `pkg/parser/include_processor.go` defined `containsExpression(v any) bool` — a recursive walker over `map[string]any`/`[]any` — that collided in name with `containsExpression(s string) bool` in `pkg/workflow/expression_patterns.go`, which performs a single-string match with different semantics.

### Decision

We will reorganize files in `pkg/workflow` and rename the colliding function in `pkg/parser` so that file names accurately reflect contents and exported/unexported names do not collide across packages with divergent semantics. Specifically: (1) move `parseOnSection`, `mergeCommandOtherEvents`, `mergeEventConfig`, and `parseEventTypes` from `compiler_safe_outputs.go` to `trigger_parser.go`; move `applyDefaultTools` to `tools.go`; move `isSandboxEnabled` to `sandbox.go`. (2) Merge the `compiler_safe_outputs_env.go`, `compiler_safe_outputs_config.go`, `compiler_safe_outputs_core.go`, and `compiler_safe_outputs_handlers.go` files into their `safe_outputs_*` counterparts and delete the source files; retain `compiler_safe_outputs_builder.go` as-is. (3) Rename `mcp_rendering.go` to `mcp_renderer_factory.go` and move `renderSafeOutputsMCPConfigWithOptions` and `renderAgenticWorkflowsMCPConfigWithOptions` into a new `mcp_renderer_builtin.go`, leaving `mcp_config_builtin.go` reserved for future type/constant declarations. (4) Rename `containsExpression(v any) bool` in `pkg/parser/include_processor.go` to `frontmatterValueContainsExpression` to make the recursive-over-frontmatter intent explicit and eliminate the cross-package name collision. All changes are intra-package (or rename-only); no public API behavior is altered.

### Alternatives Considered

#### Alternative 1: Leave Files As-Is and Document Drift via Comments

We could leave the files in place and add file-header comments explaining that each file's actual scope diverges from its name. This was rejected because documentation without structural enforcement degrades the same way it did in the cases addressed by ADR-27325, ADR-28282, and ADR-29336 — contributors continue adding new functions to the nearest convenient file, and the gap between file name and contents widens over time. The split-brain `compiler_safe_outputs_*` vs. `safe_outputs_*` naming is exactly the outcome of letting earlier drift accumulate.

#### Alternative 2: Rename `compiler_safe_outputs.go` to Match Its Contents

Rather than moving functions out, we could rename `compiler_safe_outputs.go` to a name that matches whatever six-function bag it contains. This was rejected because the functions are not semantically related to each other — they cover trigger parsing, default tools, sandbox state, and safe-jobs merging — so no single accurate name exists. Splitting by domain into existing files (`trigger_parser.go`, `tools.go`, `sandbox.go`) better serves discoverability and matches the precedent set by ADR-29336.

#### Alternative 3: Keep `containsExpression` Names and Disambiguate by Package Prefix at Call Sites

We could keep the colliding `containsExpression` names and require callers to use package-qualified references (`parser.containsExpression` vs. `workflow.containsExpression`) for clarity. This was rejected because both functions are unexported and therefore can never be referenced cross-package — the qualification mechanism does not help. The collision only surfaces when reading the code or running cross-package searches, where two same-named functions with different signatures and semantics are actively misleading. A rename is the only way to make the divergent semantics visible at the name level.

### Consequences

#### Positive
- `compiler_safe_outputs.go` is reduced to two functions (`mergeSafeJobsFromIncludedConfigs`, `needsGitCommands`), tightly scoped to safe-output-specific compiler glue.
- Trigger parsing, default-tools logic, and sandbox state checks are co-located with the rest of their respective domains (`trigger_parser.go`, `tools.go`, `sandbox.go`), improving discoverability.
- The `compiler_safe_outputs_*` vs. `safe_outputs_*` split-brain is eliminated; safe-outputs code is reachable under a single naming prefix.
- `mcp_renderer_factory.go` and `mcp_renderer_builtin.go` accurately describe their renderer contents; `mcp_config_builtin.go` is reserved for future config types and no longer misleads readers.
- Cross-package `grep containsExpression` now returns one definition per distinct semantic, making static analysis and code review more reliable.
- Reinforces the semantic file-organization convention from ADR-27325, ADR-28282, and ADR-29336.

#### Negative
- `git blame` on the moved functions points at the relocation commit rather than original authorship without `--follow`, increasing the friction of historical investigations.
- Open pull requests touching `compiler_safe_outputs_env.go`, `compiler_safe_outputs_config.go`, `compiler_safe_outputs_core.go`, or `compiler_safe_outputs_handlers.go` will incur merge conflicts that require rebasing onto the new file layout.
- Reviewers must read the diff with both rename and move semantics in mind; large patches with `additions ≈ deletions` are harder to scan than equivalent renames in a single git operation.

#### Neutral
- No public API surface changes; all moved or renamed identifiers retain their signatures.
- `pkg/workflow` file count changes (net reduction from merging three thin files; net increase of one from splitting `mcp_renderer_builtin.go` out of `mcp_config_builtin.go`).
- The unexported `containsExpression` symbol in `pkg/parser` is gone; any future references must use `frontmatterValueContainsExpression`.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### Trigger Parsing (`pkg/workflow`)

1. `parseOnSection`, `mergeCommandOtherEvents`, `mergeEventConfig`, and `parseEventTypes` **MUST** reside in `pkg/workflow/trigger_parser.go`.
2. New functions that parse, normalize, or merge the workflow `on:` section or its sub-events **MUST** be added to `trigger_parser.go` and **MUST NOT** be placed in `compiler_safe_outputs.go`.

### Default Tools and Sandbox State (`pkg/workflow`)

1. `applyDefaultTools` **MUST** reside in `pkg/workflow/tools.go`, adjacent to its caller `applyDefaults`.
2. `isSandboxEnabled` **MUST** reside in `pkg/workflow/sandbox.go`, adjacent to its callers in `firewall.go` and `strict_mode_permissions_validation.go`.
3. New default-tool-application functions **MUST** be added to `tools.go`; new sandbox state predicates **MUST** be added to `sandbox.go`. Neither **SHALL** be placed in `compiler_safe_outputs.go`.

### Safe-Outputs File Naming (`pkg/workflow`)

1. New files whose primary contents pertain to safe-outputs configuration, env wiring, core types, or handler registration **MUST** use the `safe_outputs_*` prefix.
2. New files **MUST NOT** use the `compiler_safe_outputs_*` prefix, except for `compiler_safe_outputs.go` and `compiler_safe_outputs_builder.go`, which are retained for backwards compatibility with their current scope.
3. The deleted files `compiler_safe_outputs_env.go`, `compiler_safe_outputs_config.go`, `compiler_safe_outputs_core.go`, and `compiler_safe_outputs_handlers.go` **MUST NOT** be reintroduced; their previous contents now reside in the corresponding `safe_outputs_*` files.

### MCP Rendering File Naming (`pkg/workflow`)

1. `mcp_renderer_factory.go` **MUST** contain the renderer factory previously located in `mcp_rendering.go`; the file name `mcp_rendering.go` **MUST NOT** be reintroduced.
2. `renderSafeOutputsMCPConfigWithOptions` and `renderAgenticWorkflowsMCPConfigWithOptions` **MUST** reside in `mcp_renderer_builtin.go`.
3. `mcp_config_builtin.go` **MUST NOT** contain renderer functions; its scope **MUST** be limited to type or constant declarations for built-in MCP configuration.

### Cross-Package Function Naming (`pkg/parser` and `pkg/workflow`)

1. The recursive frontmatter walker in `pkg/parser/include_processor.go` **MUST** be named `frontmatterValueContainsExpression`.
2. New unexported helper functions **MUST NOT** share an identifier with an unexported function in another `gh-aw` package when the two functions have divergent signatures or semantics, where this would mislead cross-package code search or review.

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Failure to meet any **MUST** or **MUST NOT** requirement — in particular, reintroducing the deleted `compiler_safe_outputs_*` files, placing trigger-parsing or sandbox-state functions back into `compiler_safe_outputs.go`, restoring the `mcp_rendering.go` filename, or restoring `containsExpression` as the name of the parser-side recursive walker — constitutes non-conformance.

---

*This is a DRAFT ADR generated by the [Design Decision Gate](https://github.com/github/gh-aw/actions/runs/26492354588) workflow. The PR author must review, complete, and finalize this document before the PR can merge.*
