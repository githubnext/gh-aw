# styles Package

> Centralized Dracula-inspired terminal color palette, adaptive color variables, border definitions, and pre-configured lipgloss styles for consistent CLI output.

The `styles` package provides centralized color constants, adaptive color variables, border definitions, and pre-configured `lipgloss` styles for consistent terminal output across the codebase.

## Overview

All colors use `compat.AdaptiveColor` to automatically choose between light and dark variants based on the terminal's background. The dark palette is inspired by the [Dracula theme](https://draculatheme.com/); the light palette uses darker, more saturated colors for good contrast on light backgrounds.

## Public API

The `styles` package exports the following:

| Category | Exports |
|----------|---------|
| Adaptive colors | `ColorError`, `ColorWarning`, `ColorSuccess`, `ColorInfo`, `ColorPurple`, `ColorYellow`, `ColorComment`, `ColorForeground`, `ColorBackground`, `ColorBorder`, `ColorTableAltRow` |
| Border styles | `RoundedBorder`, `NormalBorder`, `ThickBorder` |
| Pre-configured `lipgloss.Style` | `Error`, `Warning`, `Success`, `Info`, `FilePath`, `LineNumber`, `ContextLine`, `Highlight`, `Location`, `Command`, `Progress`, `Prompt`, `Count`, `Verbose`, `ListHeader`, `ListItem`, `Header`, `TableHeader`, `TableCell`, `TableTotal`, `TableTitle`, `TableBorder`, `ErrorBox`, `ServerName`, `ServerType`, `TreeEnumerator`, `TreeNode`, `ScheduleCalendarEmpty`, `ScheduleCalendarLow`, `ScheduleCalendarMedium`, `ScheduleCalendarHigh`, `ScheduleCalendarCritical` |
| Huh theme | `HuhTheme` — `huh.ThemeFunc` for Dracula-inspired interactive forms |

## Adaptive Color Variables

These variables provide `compat.AdaptiveColor` values that auto-select the correct shade at render time:

| Variable | Semantic use | Light | Dark |
|----------|-------------|-------|------|
| `ColorError` | Error messages, critical issues | `#D73737` | `#FF5555` |
| `ColorWarning` | Warnings, cautionary information | `#E67E22` | `#FFB86C` |
| `ColorSuccess` | Success messages, confirmations | `#27AE60` | `#50FA7B` |
| `ColorInfo` | Informational messages | `#2980B9` | `#8BE9FD` |
| `ColorPurple` | File paths, commands, highlights | `#8E44AD` | `#BD93F9` |
| `ColorYellow` | Progress, attention-grabbing content | `#B7950B` | `#F1FA8C` |
| `ColorComment` | Secondary/muted information, line numbers | `#6C7A89` | `#6272A4` |
| `ColorForeground` | Primary text content | `#2C3E50` | `#F8F8F2` |
| `ColorBackground` | Highlighted backgrounds | `#ECF0F1` | `#282A36` |
| `ColorBorder` | Table borders and dividers | `#BDC3C7` | `#44475A` |
| `ColorTableAltRow` | Alternating table row backgrounds | `#F5F5F5` | `#1A1A1A` |

## Border Definitions

| Variable | Style | Usage |
|----------|-------|-------|
| `RoundedBorder` | `╭╮╰╯` rounded corners | Tables, boxes, panels (primary) |
| `NormalBorder` | Straight lines | Left-side emphasis, subtle dividers |
| `ThickBorder` | Thick lines | Reserved for maximum visual emphasis |

## Pre-configured Styles

These `lipgloss.Style` values are ready to use directly:

| Variable | Color | Usage |
|----------|-------|-------|
| `Error` | Red, bold | Error messages |
| `Warning` | Orange, bold | Warning messages |
| `Success` | Green, bold | Success confirmations |
| `Info` | Cyan, bold | Informational messages |
| `FilePath` | Purple, bold | File paths |
| `LineNumber` | Comment/muted | Line numbers in diffs |
| `ContextLine` | Foreground | Context lines in diffs |
| `Highlight` | Error bg, background fg (inverted) | Highlighted error text |
| `Location` | Warning/orange, bold | Location references |
| `Command` | Purple, bold | CLI commands |
| `Progress` | Yellow | Progress indicators |
| `Prompt` | Success/green, bold | Interactive prompts |
| `Count` | Info/cyan, bold | Numeric counts |
| `Verbose` | Comment/muted, italic | Verbose/debug output |
| `ListHeader` | Success/green, bold, underline | List section headers |
| `ListItem` | Foreground | List items |
| `TableHeader` | Comment/muted, bold | Table column headers |
| `TableCell` | Foreground | Table cell content |
| `TableTotal` | Success/green, bold | Table total/summary rows |
| `TableTitle` | Success/green, bold | Table titles |
| `TableBorder` | Border color | Table border lines |
| `ServerName` | Purple, bold | MCP server names |
| `ServerType` | Info/cyan | MCP server type labels |
| `ErrorBox` | Error color, rounded border | Error message boxes |
| `Header` | Success/green, bold, bottom margin | Section headers |
| `TreeEnumerator` | Border color | Tree branch characters |
| `TreeNode` | Foreground | Tree node text |
| `ScheduleCalendarEmpty` | Comment/muted | Heatmap: zero-trigger slots |
| `ScheduleCalendarLow` | Info/cyan | Heatmap: low-intensity slots |
| `ScheduleCalendarMedium` | Success/green | Heatmap: medium-intensity slots |
| `ScheduleCalendarHigh` | Warning/orange | Heatmap: high-intensity slots |
| `ScheduleCalendarCritical` | Error/red, bold | Heatmap: critical-intensity slots |

## Usage Examples

```go
import "github.com/github/gh-aw/pkg/styles"

// Use pre-configured styles
fmt.Println(styles.Error.Render("Something went wrong"))
fmt.Println(styles.Success.Render("Operation completed"))
fmt.Println(styles.Command.Render("gh aw compile"))

// Use adaptive colors for custom styles
customStyle := lipgloss.NewStyle().
    Foreground(styles.ColorInfo).
    Bold(true)
fmt.Println(customStyle.Render("Custom styled text"))
```

## Huh Theme

The package also exports `HuhTheme` — a `huh.ThemeFunc` that applies the same Dracula-inspired color palette to interactive forms rendered with the [huh](https://github.com/charmbracelet/huh) library.

```go
import "github.com/github/gh-aw/pkg/styles"

form := huh.NewForm(...).WithTheme(styles.HuhTheme)
```

## Wasm Build Variants

Under the `js || wasm` build constraint, `theme_wasm.go` replaces all `lipgloss` types with lightweight no-op stubs that return text unchanged (no ANSI escape codes emitted):

| Type | Implements | Behavior |
|------|-----------|---------|
| `WasmStyle` | `.Render(...string) string` | Returns input text concatenated, unchanged |
| `WasmColor` | `color.Color` via `.RGBA()` | Returns transparent black `(0,0,0,0)` |
| `WasmBorder` | — | No-op placeholder |

All adaptive color variables, border variables, and style variables are re-declared as `WasmColor{}`, `WasmBorder{}`, or `WasmStyle{}` values in the Wasm build.

## Dependencies

**External**:
- `charm.land/lipgloss/v2` — terminal style rendering (non-Wasm builds)
- `charm.land/lipgloss/v2/compat` — adaptive color support (`compat.AdaptiveColor`)
- `charm.land/huh/v2` — form library themed by `HuhTheme` (non-Wasm builds)

## Design Decisions

- Colors are defined with both light and dark hex constants (`hexColor*Light`, `hexColor*Dark`) so tests can assert exact color values without depending on the `lipgloss` type system.
- The package uses `charm.land/lipgloss/v2` and `charm.land/lipgloss/v2/compat` for adaptive color support.
- For visual examples and detailed usage guidelines, see `scratchpad/styles-guide.md`.
- All `*` styles export pre-configured `lipgloss.Style` values (not functions), so they can be used with method chaining: `styles.Error.Copy().Underline(true)`.

## Thread Safety

All exported variables are package-level `lipgloss.Style` values initialized at program startup. They are safe to read concurrently. Creating derived styles via method chaining (e.g., `styles.Error.Copy().Underline(true)`) MUST be done per call site — `lipgloss.Style` values SHOULD NOT be mutated after initialization.

---

*This specification is automatically maintained by the [spec-extractor](../../.github/workflows/spec-extractor.md) workflow.*
