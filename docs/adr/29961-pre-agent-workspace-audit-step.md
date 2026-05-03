# ADR-29961: Pre-Agent Workspace Audit Step After MCP CLI Mount

**Date**: 2026-05-03
**Status**: Draft
**Deciders**: pelikhan, gh-aw platform team

---

## Part 1 — Narrative (Human-Friendly)

### Context

The gh-aw workflow compiles and executes AI agent runs inside GitHub Actions. Before the AI engine begins, several preparatory steps run: skills are loaded, agent configs are mounted, and MCP servers are wired up as CLI shims. When agents behave unexpectedly—using the wrong skill version, missing an extension, or picking up a stale config—there was no lightweight, reproducible record of exactly what files were present at the moment execution started. Existing artifact collection captures post-run state but does not provide a dedicated pre-execution snapshot of agent-related directories.

### Decision

We will insert a `pre-agent-audit` workflow step immediately after the MCP CLI mount step and before the AI engine executes. This step uses `find` to enumerate files in agent-related directories (`$GITHUB_WORKSPACE/.github/agents/`, `.github/skills/`, `.github/copilot/`, `$HOME/.github/`, `$HOME/.local/share/gh/extensions/`, `$RUNNER_TEMP/gh-aw/`) and writes the listing to a well-known path (`/tmp/gh-aw/pre-agent-audit.txt`). The file is then included in the agent artifact, giving operators a point-in-time snapshot of the agent's configuration landscape for every run.

### Alternatives Considered

#### Alternative 1: Agent-Side Introspection

The AI agent itself could list its environment during execution by invoking shell commands. This was not chosen because it conflates the agent's task scope with infrastructure observability, makes the audit conditional on agent behavior, and cannot capture state that is cleaned up before the agent starts (such as temporary mount scripts).

#### Alternative 2: Rely on Existing Artifact Collection

The workflow already collects a broad set of files into the agent artifact. One could argue that the relevant directories are already implicitly captured. This was not chosen because existing collection targets log files and runtime outputs—not a structured, pre-execution directory listing—making it unreliable for diagnosing setup-phase issues.

#### Alternative 3: Structured JSON Manifest Instead of Plain-Text Listing

A richer audit could emit a JSON manifest with file sizes, mtimes, and checksums. This was not chosen for the initial implementation because the primary goal is human-readable post-mortem investigation; plain text is faster to implement, has no parsing dependencies, and is easier to read in CI logs. A richer format can be adopted later if needed.

### Consequences

#### Positive
- Every agent run now produces a reproducible, point-in-time snapshot of the agent workspace configuration that can be inspected during post-mortem analysis.
- The audit file is attached to the artifact, so the information is available without re-running the job.
- `continue-on-error: true` ensures the step never blocks agent execution, even if a directory is missing or permissions differ between environments.

#### Negative
- Every compiled lock file gains an additional ~40-line step block, increasing generated YAML size across all workflow definitions.
- Failures in the audit step are silently swallowed (`continue-on-error: true`), so a broken audit produces an empty or missing file with no visible signal in the workflow UI.
- The plain-text listing format makes programmatic analysis (e.g., diffing runs) harder than a structured format would.

#### Neutral
- All lock files must be regenerated via `make recompile` and golden test files updated via `make update-wasm-golden` whenever the audit step changes, which is the existing lock-file update contract.
- The audit file path (`/tmp/gh-aw/pre-agent-audit.txt`) is now a named constant (`PreAgentAuditFilePath`) rather than an inline string, consistent with the existing constant-extraction pattern in `pkg/constants/constants.go`.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### Step Placement and Execution

1. The pre-agent audit step **MUST** be placed after all MCP CLI mount steps and before the credential-cleanup and AI-engine-execution steps.
2. The pre-agent audit step **MUST** be declared with `continue-on-error: true` so that directory-not-found errors or permission failures do not block agent execution.
3. The pre-agent audit step **MUST** use the step id `pre-agent-audit`.

### Covered Directories

4. The audit **MUST** enumerate files in at least the following directories:
   - `$GITHUB_WORKSPACE/.github/agents/`
   - `$GITHUB_WORKSPACE/.github/skills/`
   - `$GITHUB_WORKSPACE/.github/copilot/`
   - `$HOME/.github/`
   - `$HOME/.local/share/gh/extensions/`
   - `$RUNNER_TEMP/gh-aw/`
5. The audit **MUST NOT** include files under common cache directories: `node_modules`, `__pycache__`, `.cache`, `vendor`, `.npm`, `.yarn`, `.pnpm-store`, `site-packages`, `.bundle`.

### Output and Artifact Inclusion

6. The audit **MUST** write its listing to the path defined by the `PreAgentAuditFilePath` constant (currently `/tmp/gh-aw/pre-agent-audit.txt`).
7. The audit step **MUST** emit `pre-agent-audit-file` and `pre-agent-audit-line-count` as `GITHUB_OUTPUT` values.
8. The audit file path **MUST** be included in the `collectArtifactPaths` list so it is attached to the agent artifact on every run.
9. The audit file path **MUST** be referenced via the `PreAgentAuditFilePath` constant; implementations **MUST NOT** hardcode the path as a bare string literal.

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Failure to meet any **MUST** or **MUST NOT** requirement constitutes non-conformance.

---

*This is a DRAFT ADR generated by the [Design Decision Gate](https://github.com/github/gh-aw/actions/runs/25284065916) workflow. The PR author must review, complete, and finalize this document before the PR can merge.*
