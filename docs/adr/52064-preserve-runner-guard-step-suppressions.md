# ADR-52064: Preserve Runner Guard Step Suppressions Through Workflow Compilation

**Date**: 2026-08-11
**Status**: Draft
**Deciders**: pelikhan, copilot-swe-agent

---

### Context

The gh-aw workflow compiler reads human-authored workflow definitions (`.md` frontmatter + body) and generates locked YAML files (`.lock.yml`). Runner Guard security rule RGS-012 flags outbound HTTP requests made during CI as potential secret exfiltration. Operators suppress known-safe requests by placing `# runner-guard:ignore RGS-012 -- <justification>` comments directly above the relevant step in the frontmatter YAML. However, these comments were silently discarded during compilation, leaving the generated `.lock.yml` without the suppressions and causing legitimate, read-only model inventory requests (to Anthropic, Google, OpenAI, and models.dev) to be flagged on every run.

### Decision

We will add a text-based post-processing step (`preserveRunnerGuardStepSuppressions`) to the workflow compiler that copies `runner-guard:ignore` directive comments from the frontmatter YAML to the matching step in the generated YAML output. Both supported directive forms are recognised: a standalone comment on the line above a step, and an inline trailing comment on the step's first line. Each step is identified by its name when it has one, and by its first line's content otherwise, so unnamed `- uses:`/`- run:` steps are covered as well. Block scalar payloads (`run: |`, `run: >`) are excluded from the scan in both documents, so directives and `- name:` lines embedded in shell scripts are never treated as structural YAML. To prevent unsafe propagation, suppressions are only injected when the step identity appears exactly once in both the frontmatter and the generated output, and directives that annotate an ambiguous (repeated) step identity are dropped rather than attributed to an arbitrary step.

### Alternatives Considered

#### Alternative 1: Manually patch the generated lock file

Operators could edit `.lock.yml` directly after each compilation to add suppression comments. This was rejected because lock files are marked "DO NOT EDIT" and are regenerated on every compiler run, causing any manual additions to be overwritten.

#### Alternative 2: Global workflow-level Runner Guard exception

Runner Guard could be configured with a blanket exception for the entire model inventory workflow, bypassing per-step suppression. This was rejected because a global exception removes targeted justification-per-request and reduces security visibility, making it harder to audit which specific requests are considered safe.

### Consequences

#### Positive
- Suppressions are authored once in the readable workflow source (`.md` frontmatter) and automatically survive compilation — no manual post-processing of lock files required.
- Ambiguous cases (duplicate step names, runner-guard comments embedded inside shell scripts) are detected and silently skipped, preventing incorrect suppression propagation.

#### Negative
- The propagation mechanism is identity-based: renaming a step (or changing its first line) in the frontmatter without updating the corresponding suppression comment will silently drop the suppression from the next generated lock file.
- The post-processing step inspects and rewrites the generated YAML as raw text rather than parsing it structurally, making it sensitive to formatting changes in the compiler's YAML output.

#### Neutral
- Lock files must be regenerated whenever frontmatter suppression comments are added, removed, or modified.
- The feature ships with unit tests covering propagation for named, unnamed and inline-annotated steps, plus script-embedded comments, duplicate step names and ambiguous directives being ignored.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
