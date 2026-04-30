# ADR-29170: Stdin Input Mode for Logs and Audit Commands

**Date**: 2026-04-29
**Status**: Draft
**Deciders**: pelikhan, Copilot

---

## Part 1 — Narrative (Human-Friendly)

### Context

The `gh aw logs` and `gh aw audit` commands discover workflow runs by querying the GitHub API based on filters (workflow name, count, date range). This API-based discovery is unsuitable when a user already knows the exact run IDs they want to process — for example, when scripting batch analyses, piping output from another tool, or replaying a saved list of run IDs. There was no way to bypass discovery and supply run IDs directly without using positional arguments, which do not integrate naturally with shell pipelines.

### Decision

We will add a `--stdin` flag to both `gh aw logs` and `gh aw audit` that reads workflow run IDs or URLs from standard input (one per line), bypassing the GitHub API run-discovery step entirely. This approach follows Unix pipeline conventions and allows users to compose `gh aw logs` and `gh aw audit` with other shell tools. The `--stdin` flag is mutually exclusive with positional arguments on both commands.

### Alternatives Considered

#### Alternative 1: Positional Arguments Only (Status Quo)

Users can already supply one or more run IDs as positional arguments (e.g., `gh aw audit 1234 5678`). This works for a small, known set of runs typed interactively but does not support piping from other commands or reading from files without shell substitution (`$(cat ids.txt)`). Shell substitution has argument-count limits and breaks easily with large lists.

#### Alternative 2: File-Path Flag (`--file path/to/ids.txt`)

A `--file` flag could accept a path to a text file containing run IDs. This is more explicit and reproducible (the file path can be version-controlled), but it is less composable in shell pipelines and requires writing intermediate files. Stdin is more idiomatic for Unix-style tools and is already the conventional mechanism for streaming data into CLI commands.

### Consequences

#### Positive
- Enables Unix-style composition: users can pipe output from other `gh` or shell commands directly into `gh aw logs` and `gh aw audit`.
- Bypasses GitHub API run-discovery quota, making batch processing of known run IDs cheaper and faster.
- The stdin parsing helper (`readRunIDsFromStdin`) is a small, fully-tested utility reused by both commands.

#### Negative
- A parallel orchestration path (`DownloadWorkflowLogsFromStdin`) largely replicates the filtering and rendering logic of `DownloadWorkflowLogs`, increasing the maintenance surface.
- Some time-based and count-based flags (`--after`, `--count`, `--date`, workflow-name filtering) are silently ignored in stdin mode; this could surprise users who supply them alongside `--stdin`.
- Numeric-only run IDs require an explicit `--repo owner/repo` flag in stdin mode, because there is no workflow-name context from which to infer the repository.

#### Neutral
- The `cobra.MinimumNArgs(1)` constraint on `audit` is replaced with `cobra.ArbitraryArgs` plus manual validation; the effective behavior is unchanged for positional-args usage.
- Blank lines and `#`-prefixed comment lines in stdin input are silently skipped, which is consistent with common Unix text-file conventions.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### Stdin Flag Behaviour

1. Both `gh aw logs` and `gh aw audit` **MUST** accept a `--stdin` boolean flag that, when set, reads workflow run IDs or URLs from standard input instead of discovering runs via the GitHub API.
2. Each line read from stdin **MUST** be trimmed of leading and trailing whitespace before processing.
3. Blank lines and lines whose first non-whitespace character is `#` **MUST** be silently ignored.
4. The `--stdin` flag and positional run-ID arguments **MUST NOT** be used together; implementations **MUST** return an error if both are supplied simultaneously.
5. If stdin produces zero valid entries after filtering, the command **SHOULD** emit a warning to stderr and exit successfully (status 0) rather than treating empty input as an error.

### Input Format

1. Stdin **MUST** accept both numeric run IDs (e.g., `1234567890`) and full GitHub Actions run URLs (e.g., `https://github.com/owner/repo/actions/runs/1234567890`).
2. When a numeric-only run ID is supplied and no owner/repo is encoded in the input, implementations **MUST** require the `--repo owner/repo` flag and **MUST** return an error if it is absent.
3. Implementations **SHOULD** accept GHES run URLs in addition to github.com URLs, consistent with existing positional-argument handling.

### Flag Interactions

1. Content-filtering flags (`--engine`, `--firewall`, `--no-firewall`, `--safe-output`, `--filtered-integrity`, `--no-staged`) **MUST** apply to runs supplied via stdin in the same way they apply to runs discovered via the GitHub API.
2. Discovery-scoping flags that are meaningless without API discovery (`--count`, `--date`, `--after`, workflow-name positional argument) **SHOULD NOT** silently take effect in stdin mode; implementations **SHOULD** document that these flags are ignored when `--stdin` is set.

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Failure to meet any **MUST** or **MUST NOT** requirement constitutes non-conformance.

---

*This is a DRAFT ADR generated by the [Design Decision Gate](https://github.com/github/gh-aw/actions/runs/25131761595) workflow. The PR author must review, complete, and finalize this document before the PR can merge.*
