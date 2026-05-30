//go:build !integration && !js && !wasm

package styles

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestSpec_Constants_DocumentedSemanticColorHexValues validates that the four
// primary semantic color variables have the exact hex values stated in the
// package README.md specification table.
//
// Specification (README.md adaptive color table):
//
//	ColorError:   Light=#D73737, Dark=#FF5555
//	ColorWarning: Light=#E67E22, Dark=#FFB86C
//	ColorSuccess: Light=#27AE60, Dark=#50FA7B
//	ColorInfo:    Light=#2980B9, Dark=#8BE9FD
func TestSpec_Constants_DocumentedSemanticColorHexValues(t *testing.T) {
	tests := []struct {
		colorName string
		lightHex  string
		darkHex   string
		specLight string
		specDark  string
	}{
		{"ColorError", hexColorErrorLight, hexColorErrorDark, "#D73737", "#FF5555"},
		{"ColorWarning", hexColorWarningLight, hexColorWarningDark, "#E67E22", "#FFB86C"},
		{"ColorSuccess", hexColorSuccessLight, hexColorSuccessDark, "#27AE60", "#50FA7B"},
		{"ColorInfo", hexColorInfoLight, hexColorInfoDark, "#2980B9", "#8BE9FD"},
	}

	for _, tt := range tests {
		t.Run(tt.colorName, func(t *testing.T) {
			assert.Equal(t, tt.specLight, tt.lightHex,
				"%s Light hex should match README.md specification", tt.colorName)
			assert.Equal(t, tt.specDark, tt.darkHex,
				"%s Dark hex should match README.md specification", tt.colorName)
		})
	}
}

// TestSpec_Constants_AllElevenAdaptiveColors validates that all 11 documented
// adaptive color variables are defined with both Light and Dark hex values as
// listed in the package README.md adaptive color table.
//
// Specification: "These variables provide compat.AdaptiveColor values that
// auto-select the correct shade at render time."
func TestSpec_Constants_AllElevenAdaptiveColors(t *testing.T) {
	type colorDef struct {
		name  string
		light string
		dark  string
	}

	colors := []colorDef{
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

	assert.Len(t, colors, 11,
		"README.md documents exactly 11 adaptive color variables")

	for _, c := range colors {
		t.Run(c.name, func(t *testing.T) {
			assert.NotEmpty(t, c.light,
				"%s must have a non-empty Light hex value", c.name)
			assert.NotEmpty(t, c.dark,
				"%s must have a non-empty Dark hex value", c.name)
			assert.NotEqual(t, c.light, c.dark,
				"%s Light and Dark values must differ — adaptive color requires distinct shades", c.name)
		})
	}
}

// TestSpec_Constants_RoundedBorderCornerCharacters validates that RoundedBorder
// uses the documented "╭╮╰╯" rounded corner characters as described in the README.md.
//
// Specification: "RoundedBorder: ╭╮╰╯ rounded corners — Tables, boxes, panels (primary)"
func TestSpec_Constants_RoundedBorderCornerCharacters(t *testing.T) {
	assert.Equal(t, "╭", RoundedBorder.TopLeft,
		"RoundedBorder.TopLeft should be the rounded corner character ╭")
	assert.Equal(t, "╮", RoundedBorder.TopRight,
		"RoundedBorder.TopRight should be the rounded corner character ╮")
	assert.Equal(t, "╰", RoundedBorder.BottomLeft,
		"RoundedBorder.BottomLeft should be the rounded corner character ╰")
	assert.Equal(t, "╯", RoundedBorder.BottomRight,
		"RoundedBorder.BottomRight should be the rounded corner character ╯")
}

// TestSpec_Constants_ThreeBordersAreDistinct validates that RoundedBorder,
// NormalBorder, and ThickBorder produce visually distinct output as documented
// in the README.md border definitions table.
//
// Specification:
//
//	RoundedBorder: "╭╮╰╯ rounded corners"
//	NormalBorder:  "Straight lines"
//	ThickBorder:   "Thick lines"
func TestSpec_Constants_ThreeBordersAreDistinct(t *testing.T) {
	t.Run("RoundedBorder differs from NormalBorder", func(t *testing.T) {
		assert.NotEqual(t, RoundedBorder.TopLeft, NormalBorder.TopLeft,
			"RoundedBorder and NormalBorder TopLeft characters must differ")
	})

	t.Run("RoundedBorder differs from ThickBorder", func(t *testing.T) {
		assert.NotEqual(t, RoundedBorder.Top, ThickBorder.Top,
			"RoundedBorder and ThickBorder Top characters must differ")
	})

	t.Run("NormalBorder differs from ThickBorder", func(t *testing.T) {
		assert.NotEqual(t, NormalBorder.Top, ThickBorder.Top,
			"NormalBorder and ThickBorder Top characters must differ")
	})
}

// TestSpec_Types_StylesAreRenderableValues validates that all documented
// pre-configured lipgloss.Style variables are ready-to-use directly (not
// functions) as stated in the package README.md.
//
// Specification: "These lipgloss.Style values are ready to use directly"
// Design note: "All * styles export pre-configured lipgloss.Style values
// (not functions), so they can be used with method chaining"
func TestSpec_Types_StylesAreRenderableValues(t *testing.T) {
	styleTests := []struct {
		name   string
		render func() string
	}{
		{"Error", func() string { return Error.Render("x") }},
		{"Warning", func() string { return Warning.Render("x") }},
		{"Success", func() string { return Success.Render("x") }},
		{"Info", func() string { return Info.Render("x") }},
		{"FilePath", func() string { return FilePath.Render("x") }},
		{"LineNumber", func() string { return LineNumber.Render("x") }},
		{"ContextLine", func() string { return ContextLine.Render("x") }},
		{"Highlight", func() string { return Highlight.Render("x") }},
		{"Location", func() string { return Location.Render("x") }},
		{"Command", func() string { return Command.Render("x") }},
		{"Progress", func() string { return Progress.Render("x") }},
		{"Prompt", func() string { return Prompt.Render("x") }},
		{"Count", func() string { return Count.Render("x") }},
		{"Verbose", func() string { return Verbose.Render("x") }},
		{"ListHeader", func() string { return ListHeader.Render("x") }},
		{"ListItem", func() string { return ListItem.Render("x") }},
		{"TableHeader", func() string { return TableHeader.Render("x") }},
		{"TableCell", func() string { return TableCell.Render("x") }},
		{"TableTotal", func() string { return TableTotal.Render("x") }},
		{"TableTitle", func() string { return TableTitle.Render("x") }},
		{"TableBorder", func() string { return TableBorder.Render("x") }},
		{"ServerName", func() string { return ServerName.Render("x") }},
		{"ServerType", func() string { return ServerType.Render("x") }},
		{"ErrorBox", func() string { return ErrorBox.Render("x") }},
		{"Header", func() string { return Header.Render("x") }},
		{"TreeEnumerator", func() string { return TreeEnumerator.Render("x") }},
		{"TreeNode", func() string { return TreeNode.Render("x") }},
	}

	for _, s := range styleTests {
		t.Run(s.name, func(t *testing.T) {
			rendered := s.render()
			assert.NotEmpty(t, rendered,
				"%s.Render(\"x\") should return non-empty string — styles are values, not nil", s.name)
		})
	}
}

// TestSpec_Types_HuhTheme validates that HuhTheme is exported and non-nil
// as described in the package README.md.
//
// Specification: "The package also exports HuhTheme — a huh.ThemeFunc that
// applies the same Dracula-inspired color palette to interactive forms."
func TestSpec_Types_HuhTheme(t *testing.T) {
	assert.NotNil(t, HuhTheme,
		"HuhTheme should be exported and non-nil as documented in the README.md")
}
