//go:build !js && !wasm

package styles

import (
	"charm.land/huh/v2"
	"charm.land/lipgloss/v2"
)

// HuhTheme is a huh.ThemeFunc that maps the pkg/styles Dracula-inspired
// color palette to huh form fields, giving interactive forms the same visual
// identity as the rest of the CLI output.
var HuhTheme huh.ThemeFunc = func(isDark bool) *huh.Styles {
	t := huh.ThemeBase(isDark)
	lightDark := lipgloss.LightDark(isDark)

	// Map the pkg/styles palette using lipgloss v2's LightDark helper.
	var (
		primary    = lightDark(lipgloss.Color(hexColorPurpleLight), lipgloss.Color(hexColorPurpleDark))
		success    = lightDark(lipgloss.Color(hexColorSuccessLight), lipgloss.Color(hexColorSuccessDark))
		errorColor = lightDark(lipgloss.Color(hexColorErrorLight), lipgloss.Color(hexColorErrorDark))
		warning    = lightDark(lipgloss.Color(hexColorWarningLight), lipgloss.Color(hexColorWarningDark))
		comment    = lightDark(lipgloss.Color(hexColorCommentLight), lipgloss.Color(hexColorCommentDark))
		fg         = lightDark(lipgloss.Color(hexColorForegroundLight), lipgloss.Color(hexColorForegroundDark))
		bg         = lightDark(lipgloss.Color(hexColorBackgroundLight), lipgloss.Color(hexColorBackgroundDark))
		border     = lightDark(lipgloss.Color(hexColorBorderLight), lipgloss.Color(hexColorBorderDark))
	)

	// Focused field styles
	t.Focused.Base = t.Focused.Base.BorderForeground(border)
	t.Focused.Card = t.Focused.Base
	t.Focused.Title = t.Focused.Title.Foreground(primary).Bold(true)
	t.Focused.NoteTitle = t.Focused.NoteTitle.Foreground(primary).Bold(true).MarginBottom(1)
	t.Focused.Directory = t.Focused.Directory.Foreground(primary)
	t.Focused.Description = t.Focused.Description.Foreground(comment)
	t.Focused.ErrorIndicator = t.Focused.ErrorIndicator.Foreground(errorColor)
	t.Focused.ErrorMessage = t.Focused.ErrorMessage.Foreground(errorColor)

	// Select / navigation indicators
	t.Focused.SelectSelector = t.Focused.SelectSelector.Foreground(warning)
	t.Focused.NextIndicator = t.Focused.NextIndicator.Foreground(warning)
	t.Focused.PrevIndicator = t.Focused.PrevIndicator.Foreground(warning)

	// List option styles
	t.Focused.Option = t.Focused.Option.Foreground(fg)
	t.Focused.MultiSelectSelector = t.Focused.MultiSelectSelector.Foreground(warning)
	t.Focused.SelectedOption = t.Focused.SelectedOption.Foreground(success)
	t.Focused.SelectedPrefix = t.Focused.SelectedPrefix.Foreground(success)
	t.Focused.UnselectedOption = t.Focused.UnselectedOption.Foreground(fg)
	t.Focused.UnselectedPrefix = t.Focused.UnselectedPrefix.Foreground(comment)

	// Button styles
	t.Focused.FocusedButton = t.Focused.FocusedButton.Foreground(bg).Background(primary).Bold(true)
	t.Focused.BlurredButton = t.Focused.BlurredButton.Foreground(fg).Background(bg)
	t.Focused.Next = t.Focused.FocusedButton

	// Text input styles
	t.Focused.TextInput.Cursor = t.Focused.TextInput.Cursor.Foreground(warning)
	t.Focused.TextInput.Placeholder = t.Focused.TextInput.Placeholder.Foreground(comment)
	t.Focused.TextInput.Prompt = t.Focused.TextInput.Prompt.Foreground(primary)

	// Blurred styles mirror focused but hide the border
	t.Blurred = t.Focused
	t.Blurred.Base = t.Focused.Base.BorderStyle(lipgloss.HiddenBorder())
	t.Blurred.Card = t.Blurred.Base
	t.Blurred.NextIndicator = lipgloss.NewStyle()
	t.Blurred.PrevIndicator = lipgloss.NewStyle()

	// Group header styles
	t.Group.Title = t.Focused.Title
	t.Group.Description = t.Focused.Description

	return t
}
