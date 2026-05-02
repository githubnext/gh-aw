# ADR-29668: Inline Sub-Agent Syntax Using H2 Heading Delimiters

**Date**: 2026-05-02
**Status**: Draft
**Deciders**: pelikhan (PR author), gh-aw maintainers

---

## Part 1 — Narrative (Human-Friendly)

### Context

The gh-aw workflow system supports *sub-agents* — secondary AI agents that a parent workflow can invoke by name. Previously, every sub-agent had to be defined in a separate file under `.github/agents/` (for Copilot) or `.agents/agents/` (at runtime). This meant a workflow that needed even a simple helper agent required an additional file, a cross-file import wiring step, and a scattered authoring experience. Workflow authors asked for a way to co-locate small agent definitions with the workflow that uses them, without the overhead of managing separate files.

### Decision

We will use level-2 Markdown headings of the form `## agent: \`name\`` as the delimiter syntax for inline sub-agent blocks embedded directly inside workflow markdown files. Each such block — ending at the next `##` heading or end of file — is extracted at runtime (after `{{#runtime-import}}` macros are resolved) and written to `.agents/agents/<name>.agent.md`, where the Copilot CLI discovers and invokes it by name. At compile time the same extraction logic strips sub-agent sections from the effective markdown so they do not influence include expansion, workflow-name detection, or prompt generation.

### Alternatives Considered

#### Alternative 1: Separate agent files only (status quo)

Workflow authors continue creating individual `.github/agents/<name>.md` files for every sub-agent. This approach is already supported and requires no new parser logic. It was not chosen for the inline use case because it fragments the workflow definition across multiple files, increasing cognitive overhead for simple, tightly-coupled agents that have no reuse value outside a single workflow.

#### Alternative 2: XML/HTML comment fences as block delimiters

Sub-agent blocks could be delimited with XML-style markers such as `<!-- agent: name -->` … `<!-- /agent -->` or custom HTML elements. This avoids introducing new Markdown heading semantics. It was not chosen because Markdown renderers (GitHub, VS Code) surface `##` headings as visible, navigable sections in the table of contents, giving inline agent definitions first-class visibility in the document. HTML comments are invisible in rendered Markdown, making the sub-agent definitions harder to discover during code review.

#### Alternative 3: Frontmatter-level sub-agent declarations

Sub-agents could be declared inside the parent workflow's YAML frontmatter as a structured key (e.g., `agents: [{name: "summarizer", model: "..."}]`). This would keep the agent metadata in a structured, machine-readable block. It was not chosen because frontmatter is parsed once at compile time and does not naturally support multi-line free-form prompt text; supporting rich prompt bodies in YAML would require multi-line strings and introduce indentation-sensitive quoting complexity.

### Consequences

#### Positive
- Workflow authors can define simple, single-use sub-agents in the same file as the parent workflow, reducing file count and improving discoverability during review.
- Rendered Markdown automatically includes sub-agent sections in the document's heading outline, making agent definitions visible and navigable on GitHub and in IDEs.
- The extraction step is transparent to the rest of the pipeline: compile-time and runtime both strip sub-agent sections before any include expansion or prompt interpolation, so existing functionality is not affected.

#### Negative
- Workflow files grow longer as sub-agent definitions accumulate. A workflow with several sub-agents can become difficult to scan.
- Inline sub-agents share revision history with the parent workflow; there is no independent per-agent commit history, making it harder to understand how an individual agent evolved.
- The extraction logic must be maintained in two places: `pkg/parser/sub_agent_extractor.go` (Go, compile-time) and `actions/setup/js/extract_inline_sub_agents.cjs` (JavaScript, runtime). Any syntax change must be applied to both.

#### Neutral
- Sub-agents defined inline do **not** accept an `engine` field; they inherit the parent workflow's engine. This is a deliberate simplification but may be unexpected to authors familiar with standalone agent files.
- The `## agent: \`name\`` heading appears literally in the rendered workflow document. Workflow documentation and reference pages (`inline-sub-agents.md`) must clarify this dual role.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### Syntax

1. An inline sub-agent block **MUST** begin with a level-2 Markdown heading matching the pattern `## agent: \`name\`` where `name` starts with a lowercase letter (`a–z`) and contains only lowercase letters, digits, hyphens (`-`), and underscores (`_`).
2. An inline sub-agent block **MUST NOT** use `engine` as a frontmatter key; sub-agents **SHALL** inherit the parent workflow's engine.
3. An inline sub-agent block **MUST** end at the next level-2 Markdown heading (`##`) or end of file — whichever comes first. No explicit closing marker is required or permitted.
4. Agent names within a single workflow file **MUST** be unique. Duplicate names **SHALL** be treated as a parse error.
5. Agent names **MUST NOT** start with a digit, contain uppercase letters, spaces, or path-separator characters (`/`, `\`).

### Extraction Ordering

1. Inline sub-agent extraction **MUST** occur after all `{{#runtime-import}}` macros in the workflow file have been fully resolved, so that any import macros inside a sub-agent block are inlined before the block is written to disk.
2. The compile-time extractor (`pkg/parser/sub_agent_extractor.go`) **MUST** strip sub-agent sections from the effective markdown before include expansion, workflow-name extraction, and prompt generation.
3. The runtime extractor (`actions/setup/js/extract_inline_sub_agents.cjs`) **MUST** write each extracted block to `.agents/agents/<name>.agent.md` within the GitHub Actions workspace directory.

### Output Files

1. Each extracted sub-agent **MUST** be written to `.agents/agents/<name>.agent.md` (relative to the workspace root), where `<name>` is the identifier from the `## agent:` heading.
2. Each written file **MUST** end with a newline character.
3. Implementations **SHOULD** create the `.agents/agents/` directory recursively if it does not already exist rather than failing with an error.

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Failure to meet any **MUST** or **MUST NOT** requirement constitutes non-conformance.

---

*This is a DRAFT ADR generated by the [Design Decision Gate](https://github.com/github/gh-aw/actions/runs/25249034519) workflow. The PR author must review, complete, and finalize this document before the PR can merge.*
