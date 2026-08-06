# ADR-50850: Migrate CLI Table Rendering from tabwriter to console.RenderTable

**Date**: 2026-08-06
**Status**: Draft
**Deciders**: pelikhan (PR author), copilot-swe-agent (implementation)

---

### Context

`pkg/cli/logs_format_compact.go` rendered its two `[runs]` tables using `text/tabwriter`, writing tab-delimited lines directly to the output writer. Every other table in the CLI used the centralized `console.RenderTable` / `console.TableConfig` API, which provides lipgloss-based borders, automatic TTY detection, and consistent plain-text degradation on non-TTY writers (pipes, CI logs). The divergence meant the compact logs formatter had different visual output, bypassed the project's shared styling machinery, and required manual flush-error handling (`tw.Flush()`) not present elsewhere.

### Decision

We will migrate both tabwriter-backed tables in `renderLogsCompactToWriter` and `renderLogsCompactVerboseToWriter` to `console.RenderTable(console.TableConfig{...})`. All column headers, ordering, fallback values (`-` for empty fields), filtering of `skipped`/`cancelled` runs, and numeric formatting are preserved; only the alignment mechanism changes from tab-padding to lipgloss borders.

### Alternatives Considered

#### Alternative 1: Keep text/tabwriter

The existing tabwriter approach is simple and has no runtime dependencies beyond the standard library. However, it produces plain tab-aligned output that does not adapt to TTY vs. non-TTY contexts the same way `console.RenderTable` does, creates visible inconsistency with every other CLI table, and requires manual flush-error handling. Accepting this inconsistency indefinitely is a maintenance burden and makes the codebase harder to reason about.

#### Alternative 2: Write a Custom Formatter Matching console.RenderTable Behavior

A custom formatter could in theory produce identical output without the `console` package dependency. However, this would duplicate the TTY-detection and border-rendering logic already centralized in `console.RenderTable`, creating two sources of truth for the same behavior and increasing the risk of divergence over time. No material benefit justifies the extra code.

### Consequences

#### Positive
- Compact logs tables now use the same bordered rendering style as all other CLI tables, eliminating a visible inconsistency in the tool's output.
- TTY-detection and plain-text degradation are handled automatically by `console.RenderTable`; no manual flush-or-error pattern is needed.
- The `text/tabwriter` import and both `tw.Flush()` error-logging branches are removed, reducing code surface area.

#### Negative
- Output format changes from whitespace-aligned columns to lipgloss box-drawing characters (`╭─┬─╮`, etc.), which is a breaking change for any downstream consumer parsing the raw text of these tables.
- Box-drawing border glyphs increase per-row character count, raising token density when this output is consumed by LLMs or agents that read compact logs. The PR body explicitly calls this out as a deliberate tradeoff worth monitoring.

#### Neutral
- A new test file `logs_format_compact_test.go` is added to cover bordered rendering, non-TTY ANSI-escape degradation, and `skipped`/`cancelled` run exclusion — behavior that was previously untested.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
