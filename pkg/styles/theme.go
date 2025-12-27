// Package styles provides centralized style and color definitions for terminal output.
// It uses lipgloss.AdaptiveColor to automatically adapt colors based on the terminal background,
// ensuring good readability in both light and dark terminal themes.
package styles

import "github.com/charmbracelet/lipgloss"

// Adaptive colors that work well in both light and dark terminal themes.
// Light variants use darker, more saturated colors for visibility on light backgrounds.
// Dark variants use brighter colors (Dracula theme inspired) for dark backgrounds.
var (
	// ColorError is used for error messages and critical issues.
	ColorError = lipgloss.AdaptiveColor{
		Light: "#D73737", // Darker red for light backgrounds
		Dark:  "#FF5555", // Bright red for dark backgrounds (Dracula)
	}

	// ColorWarning is used for warning messages and cautionary information.
	ColorWarning = lipgloss.AdaptiveColor{
		Light: "#E67E22", // Darker orange for light backgrounds
		Dark:  "#FFB86C", // Bright orange for dark backgrounds (Dracula)
	}

	// ColorSuccess is used for success messages and confirmations.
	ColorSuccess = lipgloss.AdaptiveColor{
		Light: "#27AE60", // Darker green for light backgrounds
		Dark:  "#50FA7B", // Bright green for dark backgrounds (Dracula)
	}

	// ColorInfo is used for informational messages
	ColorInfo = lipgloss.AdaptiveColor{
		Light: "#2980B9", // Darker cyan/blue for light backgrounds
		Dark:  "#8BE9FD", // Bright cyan for dark backgrounds (Dracula)
	}

	// ColorPurple is used for file paths, commands, and highlights
	ColorPurple = lipgloss.AdaptiveColor{
		Light: "#8E44AD", // Darker purple for light backgrounds
		Dark:  "#BD93F9", // Bright purple for dark backgrounds (Dracula)
	}

	// ColorYellow is used for progress messages and attention-grabbing content
	ColorYellow = lipgloss.AdaptiveColor{
		Light: "#B7950B", // Darker yellow/gold for light backgrounds
		Dark:  "#F1FA8C", // Bright yellow for dark backgrounds (Dracula)
	}

	// ColorComment is used for secondary/muted information like line numbers
	ColorComment = lipgloss.AdaptiveColor{
		Light: "#6C7A89", // Muted gray-blue for light backgrounds
		Dark:  "#6272A4", // Muted purple-gray for dark backgrounds (Dracula)
	}

	// ColorForeground is used for primary text content
	ColorForeground = lipgloss.AdaptiveColor{
		Light: "#2C3E50", // Dark gray for light backgrounds
		Dark:  "#F8F8F2", // Light gray/white for dark backgrounds (Dracula)
	}

	// ColorBackground is used for highlighted backgrounds
	ColorBackground = lipgloss.AdaptiveColor{
		Light: "#ECF0F1", // Light gray for light backgrounds
		Dark:  "#282A36", // Dark purple/gray for dark backgrounds (Dracula)
	}

	// ColorBorder is used for table borders and dividers
	ColorBorder = lipgloss.AdaptiveColor{
		Light: "#BDC3C7", // Light gray border for light backgrounds
		Dark:  "#44475A", // Dark purple border for dark backgrounds (Dracula)
	}
)

// Border definitions for consistent styling across CLI output.
// These provide a centralized set of border styles that adapt based on context.
var (
	// RoundedBorder is used for error boxes, emphasis boxes, and informational panels.
	// It provides a softer, more polished appearance with rounded corners.
	RoundedBorder = lipgloss.RoundedBorder()

	// NormalBorder is used for standard tables and section dividers.
	// It provides clean, simple borders suitable for data presentation.
	NormalBorder = lipgloss.NormalBorder()

	// ThickBorder is used for high-emphasis boxes and critical information.
	// It draws more attention with heavier visual weight.
	ThickBorder = lipgloss.ThickBorder()

	// ASCIIBorder is used for tables in CI/CD logs or environments with uncertain unicode support.
	// It uses only ASCII characters (+, -, |) which work in all environments.
	// This is the recommended default for table rendering.
	ASCIIBorder = lipgloss.ASCIIBorder()

	// MarkdownBorder is used for tables that will be copied to markdown (docs, GitHub issues).
	// It uses markdown table syntax (|, -, :) for native rendering in markdown viewers.
	// Note: Should typically be used with BorderTop(false) and BorderBottom(false).
	MarkdownBorder = lipgloss.MarkdownBorder()
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
