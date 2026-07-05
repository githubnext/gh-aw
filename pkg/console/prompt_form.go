//go:build !js && !wasm

package console

import (
	"charm.land/huh/v2"
	"github.com/github/gh-aw/pkg/styles"
)

// PromptForm creates a huh form with gh-aw's default theme and accessibility mode.
func PromptForm(groups ...*huh.Group) *huh.Form {
	return huh.NewForm(groups...).WithTheme(styles.HuhTheme).WithAccessible(IsAccessibleMode())
}

// PromptInput creates a themed, accessibility-aware single-input form.
func PromptInput(input *huh.Input) *huh.Form {
	return PromptForm(huh.NewGroup(input))
}

// PromptSelect creates a themed, accessibility-aware single-select form.
func PromptSelect[T comparable](selectField *huh.Select[T]) *huh.Form {
	return PromptForm(huh.NewGroup(selectField))
}

// PromptMultiSelect creates a themed, accessibility-aware single-multiselect form.
func PromptMultiSelect[T comparable](multiSelect *huh.MultiSelect[T]) *huh.Form {
	return PromptForm(huh.NewGroup(multiSelect))
}

// PromptText creates a themed, accessibility-aware single-text form.
func PromptText(text *huh.Text) *huh.Form {
	return PromptForm(huh.NewGroup(text))
}

// PromptConfirm creates a themed, accessibility-aware single-confirm form.
func PromptConfirm(confirm *huh.Confirm) *huh.Form {
	return PromptForm(huh.NewGroup(confirm))
}
