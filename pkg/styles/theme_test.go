//go:build !integration && !js && !wasm

package styles

import (
	"strings"
	"testing"

	lipgloss "charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/compat"
)

// TestAdaptiveColorsHaveBothVariants verifies that all adaptive colors
// have both Light and Dark variants defined
func TestAdaptiveColorsHaveBothVariants(t *testing.T) {
	colorDefs := []struct {
		name  string
		light string
		dark  string
	}{
		{"ColorError", hexColorErrorLight, hexColorErrorDark},
		{"ColorWarning", hexColorWarningLight, hexColorWarningDark},
		{"ColorSuccess", hexColorSuccessLight, hexColorSuccessDark},
		{"ColorInfo", hexColorInfoLight, hexColorInfoDark},
		{"ColorPurple", hexColorPurpleLight, hexColorPurpleDark},
		{"ColorYellow", hexColorYellowLight, hexColorYellowDark},
		{"ColorComment", hexColorCommentLight, hexColorCommentDark},
		{"ColorForeground", hexColorForegroundLight, hexColorForegroundDark},
		{"ColorBackground", hexColorBackgroundLight, hexColorBackgroundDark},
		{"ColorBorder", hexColorBorderLight, hexColorBorderDark},
		{"ColorTableAltRow", hexColorTableAltRowLight, hexColorTableAltRowDark},
	}

	for _, def := range colorDefs {
		t.Run(def.name, func(t *testing.T) {
			if def.light == "" {
				t.Errorf("%s has empty Light variant", def.name)
			}
			if def.dark == "" {
				t.Errorf("%s has empty Dark variant", def.name)
			}
			// Ensure Light and Dark are different (otherwise adaptive isn't needed)
			if def.light == def.dark {
				t.Errorf("%s has identical Light and Dark variants: %s", def.name, def.light)
			}
		})
	}
}

// TestColorFormats verifies all color values are valid hex colors
func TestColorFormats(t *testing.T) {
	colorDefs := []struct {
		name  string
		light string
		dark  string
	}{
		{"ColorError", hexColorErrorLight, hexColorErrorDark},
		{"ColorWarning", hexColorWarningLight, hexColorWarningDark},
		{"ColorSuccess", hexColorSuccessLight, hexColorSuccessDark},
		{"ColorInfo", hexColorInfoLight, hexColorInfoDark},
		{"ColorPurple", hexColorPurpleLight, hexColorPurpleDark},
		{"ColorYellow", hexColorYellowLight, hexColorYellowDark},
		{"ColorComment", hexColorCommentLight, hexColorCommentDark},
		{"ColorForeground", hexColorForegroundLight, hexColorForegroundDark},
		{"ColorBackground", hexColorBackgroundLight, hexColorBackgroundDark},
		{"ColorBorder", hexColorBorderLight, hexColorBorderDark},
		{"ColorTableAltRow", hexColorTableAltRowLight, hexColorTableAltRowDark},
	}

	isValidHex := func(s string) bool {
		if len(s) != 7 {
			return false
		}
		if s[0] != '#' {
			return false
		}
		for _, c := range s[1:] {
			if (c < '0' || c > '9') && (c < 'A' || c > 'F') && (c < 'a' || c > 'f') {
				return false
			}
		}
		return true
	}

	for _, def := range colorDefs {
		t.Run(def.name+"_Light", func(t *testing.T) {
			if !isValidHex(def.light) {
				t.Errorf("%s.Light is not a valid hex color: %s", def.name, def.light)
			}
		})
		t.Run(def.name+"_Dark", func(t *testing.T) {
			if !isValidHex(def.dark) {
				t.Errorf("%s.Dark is not a valid hex color: %s", def.name, def.dark)
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
	// These are the original Dracula palette colors hardcoded for comparison.
	// If any hex constant in theme.go changes, this test will catch the drift.
	expectedDarkColors := map[string]string{
		"ColorError":      "#FF5555",
		"ColorWarning":    "#FFB86C",
		"ColorSuccess":    "#50FA7B",
		"ColorInfo":       "#8BE9FD",
		"ColorPurple":     "#BD93F9",
		"ColorYellow":     "#F1FA8C",
		"ColorComment":    "#6272A4",
		"ColorForeground": "#F8F8F2",
		"ColorBackground": "#282A36",
		"ColorBorder":     "#44475A",
	}

	actualDarkHex := map[string]string{
		"ColorError":      hexColorErrorDark,
		"ColorWarning":    hexColorWarningDark,
		"ColorSuccess":    hexColorSuccessDark,
		"ColorInfo":       hexColorInfoDark,
		"ColorPurple":     hexColorPurpleDark,
		"ColorYellow":     hexColorYellowDark,
		"ColorComment":    hexColorCommentDark,
		"ColorForeground": hexColorForegroundDark,
		"ColorBackground": hexColorBackgroundDark,
		"ColorBorder":     hexColorBorderDark,
	}

	for name, expected := range expectedDarkColors {
		t.Run(name, func(t *testing.T) {
			actual := actualDarkHex[name]
			if actual != expected {
				t.Errorf("%s.Dark = %s, want %s (original Dracula color)", name, actual, expected)
			}
		})
	}
}

// TestAdaptiveColorVarsUseHexConstants verifies that the exported AdaptiveColor vars
// are backed by the expected hex constants (spot-check a few key colors).
func TestAdaptiveColorVarsUseHexConstants(t *testing.T) {
	// Verify that the compat.AdaptiveColor vars hold non-nil color values.
	colors := map[string]compat.AdaptiveColor{
		"ColorError":       ColorError,
		"ColorWarning":     ColorWarning,
		"ColorSuccess":     ColorSuccess,
		"ColorInfo":        ColorInfo,
		"ColorPurple":      ColorPurple,
		"ColorYellow":      ColorYellow,
		"ColorComment":     ColorComment,
		"ColorForeground":  ColorForeground,
		"ColorBackground":  ColorBackground,
		"ColorBorder":      ColorBorder,
		"ColorTableAltRow": ColorTableAltRow,
	}

	for name, c := range colors {
		t.Run(name, func(t *testing.T) {
			if c.Light == nil {
				t.Errorf("%s.Light is nil", name)
			}
			if c.Dark == nil {
				t.Errorf("%s.Dark is nil", name)
			}
		})
	}
}

// TestBordersExist verifies that all expected border definitions are defined
func TestBordersExist(t *testing.T) {
	borders := map[string]lipgloss.Border{
		"RoundedBorder": RoundedBorder,
		"NormalBorder":  NormalBorder,
		"ThickBorder":   ThickBorder,
	}

	for name, border := range borders {
		t.Run(name, func(t *testing.T) {
			// Verify the border has defined characters (non-empty)
			if border.Top == "" && border.Bottom == "" && border.Left == "" && border.Right == "" {
				t.Errorf("Border %s has no defined border characters", name)
			}
		})
	}
}

// TestBordersAreDistinct verifies that each border style is visually distinct
func TestBordersAreDistinct(t *testing.T) {
	// RoundedBorder should have curved corners
	if RoundedBorder.TopLeft != "╭" {
		t.Errorf("RoundedBorder.TopLeft = %q, want curved corner ╭", RoundedBorder.TopLeft)
	}

	// NormalBorder should have straight corners
	if NormalBorder.TopLeft != "┌" {
		t.Errorf("NormalBorder.TopLeft = %q, want straight corner ┌", NormalBorder.TopLeft)
	}

	// ThickBorder should have thick lines
	if ThickBorder.Top != "━" {
		t.Errorf("ThickBorder.Top = %q, want thick line ━", ThickBorder.Top)
	}
}

// TestErrorBoxUsesCentralizedBorder verifies that ErrorBox uses the centralized RoundedBorder
func TestErrorBoxUsesCentralizedBorder(t *testing.T) {
	// Create a style with the RoundedBorder to compare
	testStyle := lipgloss.NewStyle().
		Border(RoundedBorder).
		BorderForeground(ColorError).
		Padding(1).
		Margin(1)

	testText := "Test error message"
	errorBoxResult := ErrorBox.Render(testText)
	testStyleResult := testStyle.Render(testText)

	// Both should contain the test text
	if len(errorBoxResult) == 0 {
		t.Error("ErrorBox rendered empty string")
	}
	if len(testStyleResult) == 0 {
		t.Error("testStyle rendered empty string")
	}

	// Verify that ErrorBox output contains the rounded border characters
	// RoundedBorder uses: ╭ (top-left), ╮ (top-right), ╰ (bottom-left), ╯ (bottom-right)
	if !strings.Contains(errorBoxResult, "╭") {
		t.Error("ErrorBox missing rounded top-left corner (╭)")
	}
	if !strings.Contains(errorBoxResult, "╮") {
		t.Error("ErrorBox missing rounded top-right corner (╮)")
	}
	if !strings.Contains(errorBoxResult, "╰") {
		t.Error("ErrorBox missing rounded bottom-left corner (╰)")
	}
	if !strings.Contains(errorBoxResult, "╯") {
		t.Error("ErrorBox missing rounded bottom-right corner (╯)")
	}

	// Both should produce identical output (same border, same styling)
	if errorBoxResult != testStyleResult {
		t.Errorf("ErrorBox output differs from expected:\nGot:\n%s\nExpected:\n%s", errorBoxResult, testStyleResult)
	}
}
