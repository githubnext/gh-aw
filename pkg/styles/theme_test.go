package styles

import (
	"testing"

	"github.com/charmbracelet/lipgloss"
)

// TestAdaptiveColorsHaveBothVariants verifies that all adaptive colors
// have both Light and Dark variants defined
func TestAdaptiveColorsHaveBothVariants(t *testing.T) {
	colors := map[string]lipgloss.AdaptiveColor{
		"ColorError":      ColorError,
		"ColorWarning":    ColorWarning,
		"ColorSuccess":    ColorSuccess,
		"ColorInfo":       ColorInfo,
		"ColorPurple":     ColorPurple,
		"ColorYellow":     ColorYellow,
		"ColorComment":    ColorComment,
		"ColorForeground": ColorForeground,
		"ColorBackground": ColorBackground,
		"ColorBorder":     ColorBorder,
	}

	for name, color := range colors {
		t.Run(name, func(t *testing.T) {
			if color.Light == "" {
				t.Errorf("%s has empty Light variant", name)
			}
			if color.Dark == "" {
				t.Errorf("%s has empty Dark variant", name)
			}
			// Ensure Light and Dark are different (otherwise adaptive isn't needed)
			if color.Light == color.Dark {
				t.Errorf("%s has identical Light and Dark variants: %s", name, color.Light)
			}
		})
	}
}

// TestColorFormats verifies all color values are valid hex colors
func TestColorFormats(t *testing.T) {
	colors := map[string]lipgloss.AdaptiveColor{
		"ColorError":      ColorError,
		"ColorWarning":    ColorWarning,
		"ColorSuccess":    ColorSuccess,
		"ColorInfo":       ColorInfo,
		"ColorPurple":     ColorPurple,
		"ColorYellow":     ColorYellow,
		"ColorComment":    ColorComment,
		"ColorForeground": ColorForeground,
		"ColorBackground": ColorBackground,
		"ColorBorder":     ColorBorder,
	}

	isValidHex := func(s string) bool {
		if len(s) != 7 {
			return false
		}
		if s[0] != '#' {
			return false
		}
		for _, c := range s[1:] {
			if !((c >= '0' && c <= '9') || (c >= 'A' && c <= 'F') || (c >= 'a' && c <= 'f')) {
				return false
			}
		}
		return true
	}

	for name, color := range colors {
		t.Run(name+"_Light", func(t *testing.T) {
			if !isValidHex(color.Light) {
				t.Errorf("%s.Light is not a valid hex color: %s", name, color.Light)
			}
		})
		t.Run(name+"_Dark", func(t *testing.T) {
			if !isValidHex(color.Dark) {
				t.Errorf("%s.Dark is not a valid hex color: %s", name, color.Dark)
			}
		})
	}
}

// TestStylesExist verifies that all expected styles are defined
func TestStylesExist(t *testing.T) {
	// This test ensures the styles are properly initialized
	// by checking they don't panic when accessed
	styles := map[string]lipgloss.Style{
		"Error":       Error,
		"Warning":     Warning,
		"Success":     Success,
		"Info":        Info,
		"FilePath":    FilePath,
		"LineNumber":  LineNumber,
		"ContextLine": ContextLine,
		"Highlight":   Highlight,
		"Location":    Location,
		"Command":     Command,
		"Progress":    Progress,
		"Prompt":      Prompt,
		"Count":       Count,
		"Verbose":     Verbose,
		"ListHeader":  ListHeader,
		"ListItem":    ListItem,
		"TableHeader": TableHeader,
		"TableCell":   TableCell,
		"TableTotal":  TableTotal,
		"TableTitle":  TableTitle,
		"TableBorder": TableBorder,
		"ServerName":  ServerName,
		"ServerType":  ServerType,
		"ErrorBox":    ErrorBox,
		"Header":      Header,
	}

	for name, style := range styles {
		t.Run(name, func(t *testing.T) {
			// Render a test string to verify the style works
			result := style.Render("test")
			if result == "" {
				t.Errorf("Style %s rendered empty string", name)
			}
		})
	}
}

// TestStylesRenderNonEmpty verifies that styles can render text
func TestStylesRenderNonEmpty(t *testing.T) {
	testText := "Hello World"

	tests := []struct {
		name  string
		style lipgloss.Style
	}{
		{"Error", Error},
		{"Warning", Warning},
		{"Success", Success},
		{"Info", Info},
		{"FilePath", FilePath},
		{"LineNumber", LineNumber},
		{"ContextLine", ContextLine},
		{"Highlight", Highlight},
		{"Location", Location},
		{"Command", Command},
		{"Progress", Progress},
		{"Prompt", Prompt},
		{"Count", Count},
		{"Verbose", Verbose},
		{"ListHeader", ListHeader},
		{"ListItem", ListItem},
		{"TableHeader", TableHeader},
		{"TableCell", TableCell},
		{"TableTotal", TableTotal},
		{"TableTitle", TableTitle},
		{"TableBorder", TableBorder},
		{"ServerName", ServerName},
		{"ServerType", ServerType},
		{"ErrorBox", ErrorBox},
		{"Header", Header},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.style.Render(testText)
			// The rendered result should contain the original text
			// (styles add ANSI codes but shouldn't remove the text)
			if len(result) < len(testText) {
				t.Errorf("Style %s: rendered length %d is less than input length %d",
					tt.name, len(result), len(testText))
			}
		})
	}
}

// TestDarkColorsAreOriginalDracula verifies that dark variants match the original Dracula theme colors
func TestDarkColorsAreOriginalDracula(t *testing.T) {
	// These are the original colors used in the codebase before the adaptive color migration
	expectedDarkColors := map[string]string{
		"ColorError":   "#FF5555",
		"ColorWarning": "#FFB86C",
		"ColorSuccess": "#50FA7B",
		"ColorInfo":    "#8BE9FD",
		"ColorPurple":  "#BD93F9",
		"ColorYellow":  "#F1FA8C",
		"ColorComment": "#6272A4",
		// ColorForeground and ColorBackground are Dracula theme colors
		"ColorForeground": "#F8F8F2",
		"ColorBackground": "#282A36",
		"ColorBorder":     "#44475A",
	}

	actualColors := map[string]lipgloss.AdaptiveColor{
		"ColorError":      ColorError,
		"ColorWarning":    ColorWarning,
		"ColorSuccess":    ColorSuccess,
		"ColorInfo":       ColorInfo,
		"ColorPurple":     ColorPurple,
		"ColorYellow":     ColorYellow,
		"ColorComment":    ColorComment,
		"ColorForeground": ColorForeground,
		"ColorBackground": ColorBackground,
		"ColorBorder":     ColorBorder,
	}

	for name, expected := range expectedDarkColors {
		t.Run(name, func(t *testing.T) {
			actual := actualColors[name].Dark
			if actual != expected {
				t.Errorf("%s.Dark = %s, want %s (original Dracula color)", name, actual, expected)
			}
		})
	}
}
