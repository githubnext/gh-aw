# styles Package

The `styles` package provides centralized color constants, adaptive color variables, border definitions, and pre-configured `lipgloss` styles for consistent terminal output across the codebase.

## Overview

All colors use `compat.AdaptiveColor` to automatically choose between light and dark variants based on the terminal's background. The dark palette is inspired by the [Dracula theme](https://draculatheme.com/); the light palette uses darker, more saturated colors for good contrast on light backgrounds.

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
| `FilePath` | Purple | File paths |
| `LineNumber` | Comment/muted | Line numbers in diffs |
| `ContextLine` | Foreground | Context lines in diffs |
| `Highlight` | Yellow, bold | Highlighted text |
| `Location` | Purple, bold | Location references |
| `Command` | Purple | CLI commands |
| `Progress` | Yellow | Progress indicators |
| `Prompt` | Cyan | Interactive prompts |
| `Count` | Yellow, bold | Numeric counts |
| `Verbose` | Comment/muted | Verbose/debug output |
| `ListHeader` | Purple, bold | List section headers |
| `ListItem` | Foreground | List items |
| `TableHeader` | Purple, bold | Table column headers |
| `TableCell` | Foreground | Table cell content |
| `TableTotal` | Yellow, bold | Table total/summary rows |
| `TableTitle` | Purple, bold | Table titles |
| `TableBorder` | Border color | Table border lines |
| `ServerName` | Purple, bold | MCP server names |
| `ServerType` | Comment/muted | MCP server type labels |
| `ErrorBox` | Error color, rounded border | Error message boxes |
| `Header` | Foreground, bold, border | Section headers |
| `TreeEnumerator` | Comment/muted | Tree branch characters |
| `TreeNode` | Foreground | Tree node text |

## Usage

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

## Design Notes

- Colors are defined with both light and dark hex constants (`hexColor*Light`, `hexColor*Dark`) so tests can assert exact color values without depending on the `lipgloss` type system.
- The package uses `charm.land/lipgloss/v2` and `charm.land/lipgloss/v2/compat` for adaptive color support.
- For visual examples and detailed usage guidelines, see `scratchpad/styles-guide.md`.
- All `*` styles export pre-configured `lipgloss.Style` values (not functions), so they can be used with method chaining: `styles.Error.Copy().Underline(true)`.
