//go:build !js && !wasm

// Package styles provides centralized style and color definitions for terminal output.
//
// # Adaptive Color System
//
// This package uses compat.AdaptiveColor to automatically adapt colors based on the
// terminal background, ensuring good readability in both light and dark terminal themes.
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
	lipgloss "charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/compat"
)

// Hex color constants for light and dark variants.
// These are used both to build the AdaptiveColor values at runtime and to
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

// Adaptive colors that work well in both light and dark terminal themes.
// Light variants use darker, more saturated colors for visibility on light backgrounds.
// Dark variants use brighter colors (Dracula theme inspired) for dark backgrounds.
var (
	// ColorError is used for error messages and critical issues.
	ColorError = compat.AdaptiveColor{
		Light: lipgloss.Color(hexColorErrorLight), // Darker red for light backgrounds
		Dark:  lipgloss.Color(hexColorErrorDark),  // Bright red for dark backgrounds (Dracula)
	}

	// ColorWarning is used for warning messages and cautionary information.
	ColorWarning = compat.AdaptiveColor{
		Light: lipgloss.Color(hexColorWarningLight), // Darker orange for light backgrounds
		Dark:  lipgloss.Color(hexColorWarningDark),  // Bright orange for dark backgrounds (Dracula)
	}

	// ColorSuccess is used for success messages and confirmations.
	ColorSuccess = compat.AdaptiveColor{
		Light: lipgloss.Color(hexColorSuccessLight), // Darker green for light backgrounds
		Dark:  lipgloss.Color(hexColorSuccessDark),  // Bright green for dark backgrounds (Dracula)
	}

	// ColorInfo is used for informational messages
	ColorInfo = compat.AdaptiveColor{
		Light: lipgloss.Color(hexColorInfoLight), // Darker cyan/blue for light backgrounds
		Dark:  lipgloss.Color(hexColorInfoDark),  // Bright cyan for dark backgrounds (Dracula)
	}

	// ColorPurple is used for file paths, commands, and highlights
	ColorPurple = compat.AdaptiveColor{
		Light: lipgloss.Color(hexColorPurpleLight), // Darker purple for light backgrounds
		Dark:  lipgloss.Color(hexColorPurpleDark),  // Bright purple for dark backgrounds (Dracula)
	}

	// ColorYellow is used for progress messages and attention-grabbing content
	ColorYellow = compat.AdaptiveColor{
		Light: lipgloss.Color(hexColorYellowLight), // Darker yellow/gold for light backgrounds
		Dark:  lipgloss.Color(hexColorYellowDark),  // Bright yellow for dark backgrounds (Dracula)
	}

	// ColorComment is used for secondary/muted information like line numbers
	ColorComment = compat.AdaptiveColor{
		Light: lipgloss.Color(hexColorCommentLight), // Muted gray-blue for light backgrounds
		Dark:  lipgloss.Color(hexColorCommentDark),  // Muted purple-gray for dark backgrounds (Dracula)
	}

	// ColorForeground is used for primary text content
	ColorForeground = compat.AdaptiveColor{
		Light: lipgloss.Color(hexColorForegroundLight), // Dark gray for light backgrounds
		Dark:  lipgloss.Color(hexColorForegroundDark),  // Light gray/white for dark backgrounds (Dracula)
	}

	// ColorBackground is used for highlighted backgrounds
	ColorBackground = compat.AdaptiveColor{
		Light: lipgloss.Color(hexColorBackgroundLight), // Light gray for light backgrounds
		Dark:  lipgloss.Color(hexColorBackgroundDark),  // Dark purple/gray for dark backgrounds (Dracula)
	}

	// ColorBorder is used for table borders and dividers
	ColorBorder = compat.AdaptiveColor{
		Light: lipgloss.Color(hexColorBorderLight), // Light gray border for light backgrounds
		Dark:  lipgloss.Color(hexColorBorderDark),  // Dark purple border for dark backgrounds (Dracula)
	}

	// ColorTableAltRow is used for alternating row backgrounds in tables (zebra striping)
	ColorTableAltRow = compat.AdaptiveColor{
		Light: lipgloss.Color(hexColorTableAltRowLight), // Subtle light gray for light backgrounds
		Dark:  lipgloss.Color(hexColorTableAltRowDark),  // Subtle darker background for dark backgrounds
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
