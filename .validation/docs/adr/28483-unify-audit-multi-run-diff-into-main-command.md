# ADR-28483: Unify Multi-Run Diff Mode into the Main `audit` Command

**Date**: 2026-04-25
**Status**: Draft
**Deciders**: pelikhan, Copilot

---

## Part 1 — Narrative (Human-Friendly)

### Context

The `gh aw audit` command previously accepted exactly one run ID or URL and produced a single-run audit report. Comparing two runs required invoking a separate subcommand, `audit diff`, which users had to discover independently. The same limitation existed in the MCP tool wrapper, which exposed `audit` only via a single `run_id_or_url` string field. This two-entry-point design created a discoverability gap: agents and users performing regression detection had to know that multi-run comparison was a distinct subcommand rather than a natural extension of `audit`.

### Decision

We will unify multi-run diff mode into the main `audit` command by changing its argument signature from `ExactArgs(1)` to `MinimumNArgs(1)`. When exactly one argument is provided the command behaves as before; when two or more are provided the first is treated as the base run and the remaining arguments are compared against it, delegating to the existing `RunAuditDiff` implementation. The `audit diff` subcommand will be hidden (`Hidden: true`) and retained only for backward compatibility. In the MCP tool wrapper, we will add a new `run_ids_or_urls []string` field as the preferred input, while keeping the old `run_id_or_url string` field as a deprecated fallback.

### Alternatives Considered

#### Alternative 1: Keep `audit diff` as the Primary Interface, Improve Documentation

The status quo could be preserved and discoverability improved through documentation updates and help text alone. This was rejected because documentation cannot help agents that parse command output programmatically, and it would not simplify the MCP tool schema. The fundamental UX problem — that comparison requires a different command — would remain.

#### Alternative 2: Add a `--compare` Flag to `audit`

A flag-based approach (e.g., `gh aw audit 12345 --compare 12346`) would keep the argument list unambiguous (first positional arg is always the base). This was rejected because it is more verbose and less natural when comparing against multiple runs. Positional arguments are consistent with how `audit diff` already worked, so migration is straightforward for existing users and scripts.

### Consequences

#### Positive
- Users and agents have a single entry point for all audit use cases; no need to remember `audit diff`.
- The MCP tool schema gains a typed `run_ids_or_urls` array that makes multi-run diff intent explicit.
- Validation logic (self-comparison rejection, duplicate ID rejection, invalid ID rejection) is shared between the subcommand and the new path.

#### Negative
- The `audit diff` subcommand must be kept indefinitely as a hidden backward-compatibility alias, adding maintenance surface.
- The `--parse` flag silently becomes a no-op in multi-run mode, which is a subtle inconsistency that may surprise users who upgrade from single-run workflows.
- Agent instruction files and documentation required a sweep to replace `audit diff <id1> <id2>` with `audit <id1> <id2>`.

#### Neutral
- The error envelope returned by the MCP tool was updated to use `run_ids_or_urls` (array) instead of `run_id_or_url` (string), which is a breaking change for any consumer that inspects the error structure. Callers relying on the old field name will need to update.
- Test coverage for the new `runAuditMulti` function and MCP tool multi-run path was added in the same PR.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### CLI Command Signature

1. The `audit` command **MUST** accept one or more positional arguments, each being a numeric run ID or a supported GitHub Actions URL format.
2. When exactly one argument is provided, the command **MUST** produce a single-run audit report identical in structure to the previous behavior.
3. When two or more arguments are provided, the command **MUST** treat the first argument as the base run and all subsequent arguments as comparison runs, delegating to the multi-run diff implementation.
4. The command **MUST NOT** accept a self-comparison (base run ID equal to any comparison run ID) and **MUST** return a descriptive error in that case.
5. The command **MUST NOT** accept duplicate comparison run IDs and **MUST** return a descriptive error in that case.
6. The `--parse` flag **MUST** be accepted in multi-run mode but **SHOULD** be documented as a no-op; implementations **MUST NOT** fail if `--parse` is passed alongside multiple run IDs.

### `audit diff` Subcommand

1. The `audit diff` subcommand **MUST** remain present in the CLI binary and **MUST** continue to function as before.
2. The `audit diff` subcommand **MUST** be hidden from help output (`Hidden: true`) to discourage new usage.
3. The `audit diff` subcommand **MUST NOT** be removed in any release that does not provide a documented migration path.

### MCP Tool Schema

1. The MCP `audit` tool **MUST** accept a `run_ids_or_urls` field of type `[]string` as the primary input.
2. The MCP `audit` tool **MUST** accept the deprecated `run_id_or_url` field of type `string` as a fallback when `run_ids_or_urls` is absent or empty.
3. When both fields are provided, `run_ids_or_urls` **MUST** take precedence.
4. The tool **MUST** return an MCP `InvalidParams` error when neither field provides at least one run ID.
5. Error envelopes returned by the tool **MUST** include a `run_ids_or_urls` array field (not `run_id_or_url`) reflecting the resolved list of run IDs that were attempted.

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Failure to meet any **MUST** or **MUST NOT** requirement constitutes non-conformance.

---

*This is a DRAFT ADR generated by the [Design Decision Gate](https://github.com/github/gh-aw/actions/runs/24936666718) workflow. The PR author must review, complete, and finalize this document before the PR can merge.*
