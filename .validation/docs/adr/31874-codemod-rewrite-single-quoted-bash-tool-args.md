# ADR-31874: Codemod to Rewrite Single-Quoted `tools.bash` Args to Double-Quoted Form

**Date**: 2026-05-13
**Status**: Draft
**Deciders**: Unknown

---

## Part 1 — Narrative (Human-Friendly)

### Context

`tools.bash` allowlist entries containing single-quoted shell arguments (for example, `grep -rn 'pattern' --include='*.lua'`) are downstream consumed by the Copilot shell allow-tool generator, which performs prefix-matching on commands. The presence of single quotes caused those entries to be truncated to a prefix during prefix-match generation, silently degrading command precision for users. The single-quoted form is semantically equivalent in most cases to a properly-escaped double-quoted form, so an automated rewrite is feasible. Users had no automated migration path and were expected to manually rewrite entries across potentially many workflow files; this PR closes that gap inside the `gh aw fix` codemod pipeline.

### Decision

We will add a new codemod `bash-single-quoted-args-rewrite` to the `GetAllCodemods()` registry that scans `tools.bash` list entries in workflow frontmatter and rewrites safely parseable single-quoted segments into double-quoted segments while escaping `\`, `"`, `$`, and `` ` `` to preserve literal semantics. When a command contains unmatched single quotes the rewrite is skipped (the original entry is left unchanged) and a warning diagnostic is emitted, on the principle that safety beats best-effort rewriting for tool allowlists. The codemod is integrated into the existing codemod ordering and registry-presence tests so that `gh aw fix --write` applies it automatically.

### Alternatives Considered

#### Alternative 1: Documentation-Only Migration Guidance

Document the truncation issue and ask users to manually rewrite their `tools.bash` entries. This was rejected because the prefix-match truncation is silent and most users won't notice until command precision degrades in production; a manual approach does not scale across many workflows and repositories, and users have no automated way to discover the problem.

#### Alternative 2: Fix the Downstream Copilot Shell Allow-Tool Generator

Change the downstream prefix-match generator so that single-quoted entries survive without truncation. This would address the root cause in one place, but it requires changes coordinated with Copilot's allow-tool generation logic and does not retroactively help workflows that already exist with the legacy single-quoted form. The codemod approach is local to `gh aw fix`, ships independently, and migrates existing workflows in a single pass.

#### Alternative 3: Full Shell-Aware Tokenizer

Use a complete POSIX-compliant shell tokenizer (e.g., the `shlex` or `mvdan/sh` libraries) to fully parse and re-emit the command. This was rejected as over-engineered for the problem: the codemod only needs to transform balanced single-quoted segments into double-quoted equivalents while escaping a small set of metacharacters, and a small hand-written scan keeps the dependency surface narrow and the behavior easy to audit.

### Consequences

#### Positive

- Users can run `gh aw fix --write` to migrate all affected `tools.bash` entries automatically, eliminating manual rewrites across potentially many workflow files.
- Command precision is preserved end-to-end: rewritten entries survive Copilot shell prefix-match generation intact, restoring user intent.
- The rewrite is safety-first: unmatched single quotes are explicitly left alone with a warning, so the codemod never produces a semantically broken command.

#### Negative

- The rewrite is a heuristic, not a full shell parser; commands using single quotes in semantically-significant ways beyond literal segments (e.g., partially-quoted strings, complex compositions) may not round-trip in pathological cases. The unmatched-quote guard is the main defense, but other edge cases remain possible.
- Double-quoted semantics differ from single-quoted semantics in shells: variables, command substitutions, and backticks are interpreted inside double quotes. The escaping logic (`\$`, `` \` ``) is intended to preserve literal meaning, but a future change to the escape set could silently introduce expansion.
- The codemod adds another step to the pipeline; the codemod count, order, and registry assertions must be kept in sync.

#### Neutral

- The codemod is placed immediately after `getBashAnonymousRemovalCodemod()` in the registry order, grouping bash-related codemods together. The corresponding test assertions in `pkg/cli/fix_codemods_test.go` were updated to match.
- The codemod carries an `IntroducedIn: "0.39.0"` field that records the version in which the rewrite became available.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### Codemod Registration

1. The `bash-single-quoted-args-rewrite` codemod **MUST** be registered in `GetAllCodemods()` so it runs as part of `gh aw fix --write`.
2. The codemod **MUST** be implemented as a `Codemod` struct value with non-empty `ID`, `Name`, `Description`, and `IntroducedIn` fields.
3. The codemod `ID` **MUST** be `bash-single-quoted-args-rewrite` and **MUST** appear in the expected codemod order list in `pkg/cli/fix_codemods_test.go`.

### Codemod Behavior

1. The codemod **MUST** return `(content, false, nil)` and leave the file unchanged when no `tools.bash` list is present in the frontmatter.
2. The codemod **MUST** return `(content, false, nil)` when no entry in `tools.bash` contains a single quote.
3. For each `tools.bash` entry containing balanced single-quoted segments, the codemod **MUST** rewrite each segment of the form `'...'` to `"..."` with the following escaping applied inside the segment: `\`, `"`, `$`, and `` ` `` **MUST** be prefixed with a backslash.
4. The codemod **MUST NOT** modify entries that contain unmatched single quotes; such entries **MUST** be left literally unchanged in the output.
5. For each entry with unmatched single quotes, the codemod **MUST** emit a warning diagnostic identifying the entry that could not be safely rewritten.
6. The codemod **MUST** preserve all content outside `tools.bash` entries — including other frontmatter keys, markdown body content, and the YAML frontmatter delimiters.
7. The codemod **MUST NOT** alter `tools.bash` entries that are not strings (e.g., map or list values), and **MUST** return `(content, false, nil)` when `tools` or `tools.bash` exist but are not of the expected `map[string]any` / `[]any` types.

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Failure to meet any **MUST** or **MUST NOT** requirement constitutes non-conformance.

---

*This is a DRAFT ADR generated by the [Design Decision Gate](https://github.com/github/gh-aw/actions/runs/25781277297) workflow. The PR author must review, complete, and finalize this document before the PR can merge.*
