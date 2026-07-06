//go:build !js && !wasm

package console

import (
	"charm.land/huh/v2"
	"github.com/github/gh-aw/pkg/styles"
)

// NewForm creates a huh form with gh-aw's default theme and accessibility mode.
func NewForm(groups ...*huh.Group) *huh.Form {
	return huh.NewForm(groups...).WithTheme(styles.HuhTheme).WithAccessible(IsAccessibleMode())
}

// NewInputForm creates a themed, accessibility-aware single-input form.
func NewInputForm(input *huh.Input) *huh.Form {
	return NewForm(huh.NewGroup(input))
}

// NewSelectForm creates a themed, accessibility-aware single-select form.
func NewSelectForm[T comparable](selectField *huh.Select[T]) *huh.Form {
	return NewForm(huh.NewGroup(selectField))
}

// NewMultiSelectForm creates a themed, accessibility-aware single multi-select form.
func NewMultiSelectForm[T comparable](multiSelect *huh.MultiSelect[T]) *huh.Form {
	return NewForm(huh.NewGroup(multiSelect))
}

// NewTextForm creates a themed, accessibility-aware single-text form.
func NewTextForm(text *huh.Text) *huh.Form {
	return NewForm(huh.NewGroup(text))
}

// NewConfirmForm creates a themed, accessibility-aware single-confirm form.
func NewConfirmForm(confirm *huh.Confirm) *huh.Form {
	return NewForm(huh.NewGroup(confirm))
}
