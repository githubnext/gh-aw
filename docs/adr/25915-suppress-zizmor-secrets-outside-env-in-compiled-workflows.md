# ADR-25915: Suppress zizmor secrets-outside-env Findings in Compiled Workflows via Inline Annotations

**Date**: 2026-04-12
**Status**: Draft
**Deciders**: pelikhan, Copilot

---

## Part 1 — Narrative (Human-Friendly)

### Context

The workflow compiler generates GitHub Actions YAML files (`.lock.yml`) from higher-level workflow definitions. These compiled workflows use GitHub Actions secrets — primarily standard authentication tokens such as `GITHUB_TOKEN`, `GH_AW_GITHUB_TOKEN`, and `COPILOT_GITHUB_TOKEN` — without applying GitHub Actions `environment:` configuration to the enclosing jobs. The zizmor static analysis tool flags this pattern as `secrets-outside-env`, a finding that alerts teams when secrets are used in jobs that lack environment-level protection rules. Across the 187 compiled workflows in this repository, this produced 4,206 findings — nearly all of which are false positives, because environment protection rules (approval gates, deployment restrictions) are not meaningful for the ephemeral authentication tokens the compiler emits. The signal-to-noise ratio of zizmor's output had degraded to the point where genuine security findings were being obscured.

### Decision

We will suppress `secrets-outside-env` findings in compiler-generated YAML by adding `# zizmor: ignore[secrets-outside-env]` inline comments as a post-processing step in the compiler's `generateYAML()` function. The new `addZizmorIgnoreForSecretsOutsideEnv()` post-processor appends the annotation to any line that contains a `${{ secrets.* }}` expression that is not already annotated and is not itself a comment line. This approach follows the existing pattern established for `addZizmorIgnoreForWorkflowRun` and `dangerous-triggers` suppressions, keeping suppression logic co-located with the generator rather than in external configuration files. The annotations self-document the reason for the deviation at the point of use, making intent explicit to both human reviewers and tooling.

### Alternatives Considered

#### Alternative 1: Add GitHub Actions `environment:` Configuration to All Compiled Jobs

Adding an `environment:` block to every compiled job that uses secrets would make zizmor's finding technically accurate — the secrets would be inside an environment. However, named environments require upfront repository configuration (creating the environment in GitHub settings, optionally adding protection rules), and would change the runtime behavior of every compiled workflow by introducing an approval gate or deployment restriction where none was intended. This would be a significant, breaking architectural change to the workflow execution model for the sole purpose of satisfying a linter, and was rejected.

#### Alternative 2: Global Suppression via a zizmor Configuration File

zizmor supports a repository-level configuration file (e.g., `.zizmor.yml`) where entire rule categories can be ignored. Suppressing `secrets-outside-env` globally would eliminate all 4,206 findings but would also silence the finding for any handwritten workflows in the repository that genuinely should use `environment:` protection. This approach sacrifices the ability to detect the real pattern for the sake of eliminating false positives from generated code, which inverts the correct trade-off. It was rejected in favor of narrowly scoped, per-occurrence annotations.

#### Alternative 3: Accept the Noise and Leave Findings Unaddressed

The zizmor output could simply be tolerated, treating the 4,206 findings as known false positives. This was rejected because it degrades the signal-to-noise ratio of the security analysis output to the point where genuine future findings would be overlooked. Accepting persistent false positives undermines the purpose of running static analysis.

### Consequences

#### Positive
- zizmor analysis output becomes meaningful: genuine `secrets-outside-env` findings in handwritten workflows will stand out against a clean baseline, rather than being buried in 4,206 compiler-generated false positives.
- The inline annotation self-documents the security reasoning at the point of use, making it clear to reviewers why each suppression is intentional.
- The implementation follows the established `addZizmorIgnoreForWorkflowRun` / `dangerous-triggers` pattern, keeping the codebase internally consistent.
- The approach is narrowly scoped: only compiler-generated lines with `${{ secrets.* }}` expressions are annotated; handwritten files and comment lines are unaffected.

#### Negative
- The compiler now embeds awareness of a specific external tool's annotation syntax (`zizmor: ignore[...]`). If zizmor changes its inline suppression syntax, all 187 compiled workflow files would need to be regenerated.
- The `addZizmorIgnoreForSecretsOutsideEnv` post-processor operates on raw YAML strings rather than on a parsed AST, which means it relies on line-level pattern matching. Edge cases (multi-line YAML flow scalars, unusual indentation) could result in missed annotations, though the idempotency guard and early-exit optimization mitigate this.
- The suppression is applied unconditionally to all compiled workflows; there is no per-workflow mechanism to opt out if a specific compiled workflow should, in fact, be subject to `secrets-outside-env` enforcement.

#### Neutral
- All 187 `.lock.yml` files are regenerated with 4,174 new annotations; the diff is large but mechanically uniform.
- Wasm golden test files are updated to match the new compiler output.
- The 9 table-driven test cases cover the primary annotation paths, idempotency, comment-line skipping, and edge cases including empty input.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### Compiler Post-Processing

1. Implementations **MUST** apply `addZizmorIgnoreForSecretsOutsideEnv()` (or an equivalent post-processor) to all YAML strings produced by `generateYAML()` before returning the compiled workflow content.
2. The post-processor **MUST** append ` # zizmor: ignore[secrets-outside-env]` to every line that contains both a `${{` token and a `secrets.` reference, except as provided in requirements 3 and 4 below.
3. The post-processor **MUST NOT** modify lines whose trimmed content begins with `#` (comment lines).
4. The post-processor **MUST NOT** modify lines that already contain the string `zizmor: ignore[secrets-outside-env]` (idempotency guarantee).
5. Implementations **SHOULD** short-circuit and return the input string unchanged if it contains no `secrets.` substring, to avoid unnecessary line-by-line processing.

### Annotation Format

1. The inline annotation **MUST** be the exact string ` # zizmor: ignore[secrets-outside-env]` (one leading space, then the comment).
2. Implementations **MUST NOT** alter any other content on an annotated line beyond appending the annotation string.
3. Implementations **MUST** preserve the original line ending convention (the post-processor splits on `\n` and rejoins on `\n`).

### Scope Constraints

1. The post-processor **MUST** be applied only to compiler-generated YAML; implementations **MUST NOT** apply it to handwritten workflow files outside the compiler pipeline.
2. Implementations **MUST NOT** suppress `secrets-outside-env` globally via zizmor configuration; suppression **MUST** be applied inline at the point of each secret expression.

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Failure to meet any **MUST** or **MUST NOT** requirement constitutes non-conformance. In particular: omitting the post-processor from `generateYAML()`, modifying comment lines, producing non-idempotent output, or applying a global zizmor suppression instead of inline annotations are all non-conformant behaviors.

---

*This is a DRAFT ADR generated by the [Design Decision Gate](https://github.com/github/gh-aw/actions/runs/24310778766) workflow. The PR author must review, complete, and finalize this document before the PR can merge.*
