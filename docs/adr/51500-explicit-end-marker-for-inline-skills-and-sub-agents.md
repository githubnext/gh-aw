# ADR-51500: Explicit End Marker for Inline Skills and Sub-Agents

**Date**: 2026-08-08
**Status**: Accepted
**Deciders**: pelikhan, app/copilot-swe-agent

---

## Part 1 — Narrative (Human-Friendly)

### Context

Inline skills (`## skill: \`name\``, see [ADR-34874](34874-inline-skill-syntax-mirroring-inline-sub-agents.md)) and inline sub-agents (`## agent: \`name\``, see [ADR-29668](29668-inline-sub-agent-syntax-using-h2-heading-delimiters.md)) both close their block at the next level-2 Markdown heading (`##`) or end of file, with no explicit closing marker. This works well when the block is the last thing in a file, but breaks down when a block is embedded in the *middle* of a document — most notably when a small, reusable snippet is spliced into a workflow via an `imports:` entry. Because the block greedily consumes everything up to the next `##` heading anywhere in the merged document, any unrelated content that happens to follow the import point (which may be far away from the snippet itself) is silently swallowed into the block, or — worse — never reappears anywhere in the compiled output. This was discovered while trying to convert `.github/workflows/shared/reporting.md` (a short, heading-free reporting-guidelines snippet imported into ~100 workflows) into an inline skill: since the snippet has no heading of its own, the resulting skill block would extend all the way to the next `##` heading in *whatever* workflow imports it, consuming content the snippet author never intended to touch.

### Decision

We will add an optional, explicit end marker to both inline skill and inline sub-agent syntax: `## end skill: \`name\`` and `## end agent: \`name\``, respectively. The end marker mirrors the start marker's heading level, keyword ordering, and name rules (lowercase identifier in backticks) so that authors and LLMs learn a single, symmetric open/close pattern. When a start marker's block is closed by a matching end marker, extraction uses that marker as the exact boundary — regardless of any `##` headings inside the block — and any content following the end marker (up to the next start marker or end of file) is preserved as ordinary document content rather than discarded. When no matching end marker is present, behavior is unchanged from before this ADR: the block ends at the next `##` heading or EOF. An end marker whose name does not match any known start marker (a typo, or a marker used before its corresponding open marker) is treated as an authoring mistake and rejected with a parse error naming the orphaned marker.

### Alternatives Considered

#### Alternative 1: HTML/XML comment closing marker (e.g. `<!-- end skill: name -->`)

This would avoid adding new heading semantics and could not be mistaken for a real section title. It was rejected for the same reason inline sub-agents rejected HTML comments as the *opening* marker in ADR-29668: comments are invisible in rendered Markdown, undermining the discoverability that motivated using headings in the first place, and would make the open/close pair visually asymmetric (visible heading paired with an invisible comment).

#### Alternative 2: Indentation or fenced-code-block style closing (e.g. a horizontal rule `---` or a fenced block wrapping the whole section)

Horizontal rules (`---`) already have meaning as YAML frontmatter delimiters inside these blocks (skill/sub-agent frontmatter itself uses `---`), so reusing them as a section-closing marker would be ambiguous and error-prone to parse. Wrapping entire sections in triple-backtick fences would break third-party Markdown tooling that expects fenced blocks to represent code, not structural boundaries, and would visually nest awkwardly with a block's own internal code samples.

#### Alternative 3: Generic unnamed end marker (`## end`)

A single generic `## end` heading (without repeating the keyword and name) is shorter to type, but is ambiguous when multiple inline blocks are being closed in sequence or nested near each other, and produces duplicate heading text across a document, which several Markdown linters (e.g. markdownlint's MD024, "no duplicate headings") flag by default. Requiring the keyword and name in the end marker keeps every heading in the document unique and makes the specific block being closed unambiguous to both humans and LLMs skimming the file.

### Consequences

#### Positive
- Inline skills and sub-agents can now be safely embedded in the middle of a document (for example via an import) without swallowing unrelated content that follows them.
- The end marker syntax is a normal ATX heading, so it renders visibly in GitHub/VS Code previews and is markdownlint-friendly (no duplicate heading text, since names differ per block).
- Blocks closed by an explicit end marker may contain their own nested `##` headings, since the boundary is no longer "the next H2 anywhere" — useful for skills/agents with structured, multi-section instructions.
- The feature is fully backward compatible: files with no end markers behave exactly as before.

#### Negative
- Authors must now remember two related but distinct syntaxes (implicit next-H2/EOF closing vs. explicit named end marker), which adds a small amount of conceptual surface area.
- Mismatched or orphaned end markers are hard parse errors, which could surprise authors who make a name typo — though this is intentional, since a silently-ignored end marker would reintroduce the exact swallowing bug the feature is meant to fix.
- The extraction algorithm is now more complex (tracking a cursor and end-marker search windows) than the original "next H2 or EOF" scan, in both the Go compiler and the JS runtime extractor.

#### Neutral
- The end marker's search window is bounded by the next start marker of the same type (or EOF), so a start marker followed later by an unrelated, differently-named end marker for a *different* block is still treated as an orphan rather than silently matched to the wrong block.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### Syntax

1. An inline skill block **MAY** be closed by a level-2 Markdown heading matching `` ## end skill: `name` `` where `name` is identical to the name used in the corresponding `` ## skill: `name` `` start marker.
2. An inline sub-agent block **MAY** be closed by a level-2 Markdown heading matching `` ## end agent: `name` `` where `name` is identical to the name used in the corresponding `` ## agent: `name` `` start marker.
3. If a matching end marker is present after a start marker and before the next start marker of the same type (or EOF), the block's content **MUST** be exactly the text between the start marker's line and the end marker's line, regardless of any `##` headings contained within.
4. If no matching end marker is found, the block **MUST** end at the next level-2 Markdown heading (`##`) or end of file, preserving prior (pre-ADR-51500) behavior.
5. Text following a matching end marker, up to the next start marker of the same type or end of file, **MUST** be preserved as ordinary document content (not discarded).
6. An end marker whose name does not correspond to any start marker of the same type within its search window **MUST** be treated as a parse error identifying the orphaned marker's name.

### Extraction Ordering

1. End marker resolution **MUST** occur using the same extraction pass as start marker resolution, in both the compile-time Go extractors (`pkg/parser/inline_skill_extractor.go`, `pkg/parser/sub_agent_extractor.go`, sharing logic in `pkg/parser/inline_section_helpers.go`) and the runtime JS extractors (`actions/setup/js/extract_inline_skills.cjs`, `actions/setup/js/extract_inline_sub_agents.cjs`).
2. Both extraction layers **MUST** implement identical marker syntax, name rules, and closing semantics.

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Failure to meet any **MUST** or **MUST NOT** requirement constitutes non-conformance.

---

*ADR created by [adr-writer agent]. Status: Accepted.*
