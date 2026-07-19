//go:build !js && !wasm

// Package styles provides centralized style and color definitions for terminal output.
//
// # Adaptive Color System
//
// This package defines an adaptiveColor type that automatically selects between
// light and dark color variants based on the terminal background, ensuring good
// readability in both light and dark terminal themes.
// Each color constant includes both Light and Dark variants that are automatically
// selected based on the user's terminal configuration.
//
// # Design Philosophy
//
// Light Mode Strategy:
//   - Uses darker, more saturated colors for visibility on light backgrounds
//   - Ensures high contrast ratios for accessibility
//   - Colors are muted to reduce visual fatigue
//
// Dark Mode Strategy:
//   - Inspired by the Dracula color theme (https://draculatheme.com/)
//   - Uses bright, vibrant colors optimized for dark backgrounds
//   - Maintains consistency with popular dark terminal themes
//
// # Color Palette Overview
//
// The palette includes semantic colors for common CLI use cases:
//   - Status colors: Error (red), Warning (orange), Success (green), Info (cyan)
//   - Highlight colors: Purple (commands, file paths), Yellow (progress, attention)
//   - Structural colors: Comment (muted text), Foreground (primary text), Background, Border
//
// Each color constant is documented with its light/dark hex values and semantic usage.
// For visual examples and usage guidelines, see scratchpad/styles-guide.md
//
// # Usage Example
//
//	import "github.com/github/gh-aw/pkg/styles"
//
//	// Using pre-configured styles
//	fmt.Println(styles.Error.Render("Something went wrong"))
//	fmt.Println(styles.Success.Render("Operation completed"))
//
//	// Using color constants for custom styles
//	customStyle := lipgloss.NewStyle().
//		Foreground(styles.ColorInfo).
//		Bold(true)
//	fmt.Println(customStyle.Render("Custom styled text"))
package styles

import (
	"image/color"
	"os"
	"runtime"

	lipgloss "charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/term"
)

// hasDarkBackground tracks whether the terminal has a dark background.
// Default is true (dark), which suits the vast majority of modern terminals.
// On non-Windows platforms it is updated at startup by configureHasDarkBackground.
// On Windows the probe is skipped entirely: the lipgloss background-color query
// can crash (STATUS_DLL_INIT_FAILED) or hang under ConPTY and other
// pseudo-terminal environments, so the safe default is always used instead.
var hasDarkBackground = true

// adaptiveColor selects between a light and a dark color variant based on the
// terminal background detected at startup.
// Use this for shared package-level colors that should follow gh-aw's startup
// probe; for per-render selection in one-off UI code, prefer lipgloss.LightDark
// (see huh_theme.go).
type adaptiveColor struct {
	Light color.Color
	Dark  color.Color
}

// RGBA satisfies the color.Color interface.
func (c adaptiveColor) RGBA() (uint32, uint32, uint32, uint32) {
	if hasDarkBackground {
		return c.Dark.RGBA()
	}
	return c.Light.RGBA()
}

type backgroundDetector func(term.File, term.File) bool

func configureHasDarkBackground(detector backgroundDetector) {
	hasDarkBackground = detector(os.Stdin, os.Stderr)
}

func shouldProbeTerminalBackground(goos string) bool {
	// On Windows, the lipgloss background-color query can crash
	// (STATUS_DLL_INIT_FAILED) or hang under ConPTY and other pseudo-terminal
	// environments. Skip the probe entirely and use the default (dark background).
	// A Windows-safe background detector can be added later without changing the
	// startup-safety contract in this package.
	if goos == "windows" {
		return false
	}
	return true
}

func init() {
	if shouldProbeTerminalBackground(runtime.GOOS) {
		configureHasDarkBackground(lipgloss.HasDarkBackground)
	}
}

// Hex color constants for light and dark variants.
// These are used both to build the color.Color values at runtime and to
// enable straightforward assertions in tests (same package).
const (
	hexColorErrorLight       = "#D73737"
	hexColorErrorDark        = "#FF5555"
	hexColorWarningLight     = "#E67E22"
	hexColorWarningDark      = "#FFB86C"
	hexColorSuccessLight     = "#27AE60"
	hexColorSuccessDark      = "#50FA7B"
	hexColorInfoLight        = "#2980B9"
	hexColorInfoDark         = "#8BE9FD"
	hexColorPurpleLight      = "#8E44AD"
	hexColorPurpleDark       = "#BD93F9"
	hexColorYellowLight      = "#B7950B"
	hexColorYellowDark       = "#F1FA8C"
	hexColorCommentLight     = "#6C7A89"
	hexColorCommentDark      = "#6272A4"
	hexColorForegroundLight  = "#2C3E50"
	hexColorForegroundDark   = "#F8F8F2"
	hexColorBackgroundLight  = "#ECF0F1"
	hexColorBackgroundDark   = "#282A36"
	hexColorBorderLight      = "#BDC3C7"
	hexColorBorderDark       = "#44475A"
	hexColorTableAltRowLight = "#F5F5F5"
	hexColorTableAltRowDark  = "#1A1A1A"
)

// Package-level color.Color values parsed once from the hex constants above.
// Both the adaptiveColor vars below (startup-probe path) and huh_theme.go
// (per-render LightDark path) reference these, so the hex parsing happens
// only once and there is a single source of truth for the palette.
var (
	colorErrorLight       = lipgloss.Color(hexColorErrorLight)
	colorErrorDark        = lipgloss.Color(hexColorErrorDark)
	colorWarningLight     = lipgloss.Color(hexColorWarningLight)
	colorWarningDark      = lipgloss.Color(hexColorWarningDark)
	colorSuccessLight     = lipgloss.Color(hexColorSuccessLight)
	colorSuccessDark      = lipgloss.Color(hexColorSuccessDark)
	colorInfoLight        = lipgloss.Color(hexColorInfoLight)
	colorInfoDark         = lipgloss.Color(hexColorInfoDark)
	colorPurpleLight      = lipgloss.Color(hexColorPurpleLight)
	colorPurpleDark       = lipgloss.Color(hexColorPurpleDark)
	colorYellowLight      = lipgloss.Color(hexColorYellowLight)
	colorYellowDark       = lipgloss.Color(hexColorYellowDark)
	colorCommentLight     = lipgloss.Color(hexColorCommentLight)
	colorCommentDark      = lipgloss.Color(hexColorCommentDark)
	colorForegroundLight  = lipgloss.Color(hexColorForegroundLight)
	colorForegroundDark   = lipgloss.Color(hexColorForegroundDark)
	colorBackgroundLight  = lipgloss.Color(hexColorBackgroundLight)
	colorBackgroundDark   = lipgloss.Color(hexColorBackgroundDark)
	colorBorderLight      = lipgloss.Color(hexColorBorderLight)
	colorBorderDark       = lipgloss.Color(hexColorBorderDark)
	colorTableAltRowLight = lipgloss.Color(hexColorTableAltRowLight)
	colorTableAltRowDark  = lipgloss.Color(hexColorTableAltRowDark)
)

// Adaptive colors that work well in both light and dark terminal themes.
// Light variants use darker, more saturated colors for visibility on light backgrounds.
// Dark variants use brighter colors (Dracula theme inspired) for dark backgrounds.
// Note: huh_theme.go uses the same color* vars above via lipgloss.LightDark for
// per-render selection; both paths share the same palette constants.
var (
	// ColorError is used for error messages and critical issues.
	ColorError = adaptiveColor{
		Light: colorErrorLight, // Darker red for light backgrounds
		Dark:  colorErrorDark,  // Bright red for dark backgrounds (Dracula)
	}

	// ColorWarning is used for warning messages and cautionary information.
	ColorWarning = adaptiveColor{
		Light: colorWarningLight, // Darker orange for light backgrounds
		Dark:  colorWarningDark,  // Bright orange for dark backgrounds (Dracula)
	}

	// ColorSuccess is used for success messages and confirmations.
	ColorSuccess = adaptiveColor{
		Light: colorSuccessLight, // Darker green for light backgrounds
		Dark:  colorSuccessDark,  // Bright green for dark backgrounds (Dracula)
	}

	// ColorInfo is used for informational messages
	ColorInfo = adaptiveColor{
		Light: colorInfoLight, // Darker cyan/blue for light backgrounds
		Dark:  colorInfoDark,  // Bright cyan for dark backgrounds (Dracula)
	}

	// ColorPurple is used for file paths, commands, and highlights
	ColorPurple = adaptiveColor{
		Light: colorPurpleLight, // Darker purple for light backgrounds
		Dark:  colorPurpleDark,  // Bright purple for dark backgrounds (Dracula)
	}

	// ColorYellow is used for progress messages and attention-grabbing content
	ColorYellow = adaptiveColor{
		Light: colorYellowLight, // Darker yellow/gold for light backgrounds
		Dark:  colorYellowDark,  // Bright yellow for dark backgrounds (Dracula)
	}

	// ColorComment is used for secondary/muted information like line numbers
	ColorComment = adaptiveColor{
		Light: colorCommentLight, // Muted gray-blue for light backgrounds
		Dark:  colorCommentDark,  // Muted purple-gray for dark backgrounds (Dracula)
	}

	// ColorForeground is used for primary text content
	ColorForeground = adaptiveColor{
		Light: colorForegroundLight, // Dark gray for light backgrounds
		Dark:  colorForegroundDark,  // Light gray/white for dark backgrounds (Dracula)
	}

	// ColorBackground is used for highlighted backgrounds
	ColorBackground = adaptiveColor{
		Light: colorBackgroundLight, // Light gray for light backgrounds
		Dark:  colorBackgroundDark,  // Dark purple/gray for dark backgrounds (Dracula)
	}

	// ColorBorder is used for table borders and dividers
	ColorBorder = adaptiveColor{
		Light: colorBorderLight, // Light gray border for light backgrounds
		Dark:  colorBorderDark,  // Dark purple border for dark backgrounds (Dracula)
	}

	// ColorTableAltRow is used for alternating row backgrounds in tables (zebra striping)
	ColorTableAltRow = adaptiveColor{
		Light: colorTableAltRowLight, // Subtle light gray for light backgrounds
		Dark:  colorTableAltRowDark,  // Subtle darker background for dark backgrounds
	}
)

// Border definitions for consistent styling across CLI output.
// These provide a centralized set of border styles that adapt based on context.
var (
	// RoundedBorder is the primary border style for most boxes and tables.
	// It provides a softer, more polished appearance with rounded corners (╭╮╰╯).
	// Used for: tables, error boxes, emphasis boxes, and informational panels.
	RoundedBorder = lipgloss.RoundedBorder()

	// NormalBorder is used for subtle borders and section dividers.
	// It provides clean, simple straight lines suitable for left-side emphasis.
	// Used for: info sections with left border, subtle dividers.
	NormalBorder = lipgloss.NormalBorder()

	// ThickBorder is available for special cases requiring extra visual weight.
	// Use sparingly - RoundedBorder with bold/color is usually sufficient for emphasis.
	// Reserved for: future use cases requiring maximum visual impact.
	ThickBorder = lipgloss.ThickBorder()
)

// Pre-configured styles for common use cases

// Error style for error messages - bold red
var Error = lipgloss.NewStyle().
	Bold(true).
	Foreground(ColorError)

// Warning style for warning messages - bold orange
var Warning = lipgloss.NewStyle().
	Bold(true).
	Foreground(ColorWarning)

// Success style for success messages - bold green
var Success = lipgloss.NewStyle().
	Bold(true).
	Foreground(ColorSuccess)

// Info style for informational messages - bold cyan
var Info = lipgloss.NewStyle().
	Bold(true).
	Foreground(ColorInfo)

// FilePath style for file paths and locations - bold purple
var FilePath = lipgloss.NewStyle().
	Bold(true).
	Foreground(ColorPurple)

// LineNumber style for line numbers in error context - muted
var LineNumber = lipgloss.NewStyle().
	Foreground(ColorComment)

// ContextLine style for source code context lines
var ContextLine = lipgloss.NewStyle().
	Foreground(ColorForeground)

// Highlight style for error highlighting - inverted colors
var Highlight = lipgloss.NewStyle().
	Background(ColorError).
	Foreground(ColorBackground)

// Location style for directory/file location messages - bold orange
var Location = lipgloss.NewStyle().
	Bold(true).
	Foreground(ColorWarning)

// Command style for command execution messages - bold purple
var Command = lipgloss.NewStyle().
	Bold(true).
	Foreground(ColorPurple)

// Progress style for progress/activity messages - yellow
var Progress = lipgloss.NewStyle().
	Foreground(ColorYellow)

// Prompt style for user prompt messages - bold green
var Prompt = lipgloss.NewStyle().
	Bold(true).
	Foreground(ColorSuccess)

// Count style for count/numeric status messages - bold cyan
var Count = lipgloss.NewStyle().
	Bold(true).
	Foreground(ColorInfo)

// Verbose style for verbose debugging output - italic muted
var Verbose = lipgloss.NewStyle().
	Italic(true).
	Foreground(ColorComment)

// ListHeader style for section headers in lists - bold underline green
var ListHeader = lipgloss.NewStyle().
	Bold(true).
	Underline(true).
	Foreground(ColorSuccess)

// ListItem style for items in lists
var ListItem = lipgloss.NewStyle().
	Foreground(ColorForeground)

// Table styles

// TableHeader style for table headers - bold muted
var TableHeader = lipgloss.NewStyle().
	Bold(true).
	Foreground(ColorComment)

// TableCell style for regular table cells
var TableCell = lipgloss.NewStyle().
	Foreground(ColorForeground)

// TableTotal style for total/summary rows - bold green
var TableTotal = lipgloss.NewStyle().
	Bold(true).
	Foreground(ColorSuccess)

// TableTitle style for table titles - bold green
var TableTitle = lipgloss.NewStyle().
	Bold(true).
	Foreground(ColorSuccess)

// TableBorder style for table borders
var TableBorder = lipgloss.NewStyle().
	Foreground(ColorBorder)

// MCP inspection styles

// ServerName style for MCP server names - bold purple
var ServerName = lipgloss.NewStyle().
	Bold(true).
	Foreground(ColorPurple)

// ServerType style for MCP server type information - cyan
var ServerType = lipgloss.NewStyle().
	Foreground(ColorInfo)

// ErrorBox style for error boxes with rounded borders
var ErrorBox = lipgloss.NewStyle().
	Border(RoundedBorder).
	BorderForeground(ColorError).
	Padding(1).
	Margin(1)

// Header style for section headers with margin - bold green
var Header = lipgloss.NewStyle().
	Bold(true).
	Foreground(ColorSuccess).
	MarginBottom(1)

// Tree styles for hierarchical output

// TreeEnumerator style for tree branch characters (├── └──)
var TreeEnumerator = lipgloss.NewStyle().
	Foreground(ColorBorder)

// TreeNode style for tree node content
var TreeNode = lipgloss.NewStyle().
	Foreground(ColorForeground)

// Schedule calendar intensity styles for the heatmap renderer

// ScheduleCalendarEmpty style for zero-trigger slots - muted
var ScheduleCalendarEmpty = lipgloss.NewStyle().
	Foreground(ColorComment)

// ScheduleCalendarLow style for low-intensity calendar slots - cyan
var ScheduleCalendarLow = lipgloss.NewStyle().
	Foreground(ColorInfo)

// ScheduleCalendarMedium style for medium-intensity calendar slots - green
var ScheduleCalendarMedium = lipgloss.NewStyle().
	Foreground(ColorSuccess)

// ScheduleCalendarHigh style for high-intensity calendar slots - orange
var ScheduleCalendarHigh = lipgloss.NewStyle().
	Foreground(ColorWarning)

// ScheduleCalendarCritical style for critical-intensity calendar slots - bold red
var ScheduleCalendarCritical = lipgloss.NewStyle().
	Bold(true).
	Foreground(ColorError)
