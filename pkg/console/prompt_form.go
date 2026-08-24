//go:build !js && !wasm

package console

import (
	"context"
	"errors"
	"fmt"
	"io"

	"charm.land/huh/v2"
	"github.com/github/gh-aw/pkg/styles"
	"github.com/github/gh-aw/pkg/tty"
)

const (
	ansiSaveCursor       = "\0337"
	ansiRestoreCursor    = "\0338"
	ansiClearScreenBelow = "\033[J"
)

// PromptForm wraps a huh form so completed questions are removed before the
// caller prints the decision result.
type PromptForm struct {
	*huh.Form
	out        io.Writer
	clearOnRun bool
}

// NewForm creates a huh form with gh-aw's default theme and accessibility mode.
func NewForm(groups ...*huh.Group) *PromptForm {
	accessible := IsAccessibleMode()
	return &PromptForm{
		Form:       huh.NewForm(groups...).WithTheme(styles.HuhTheme).WithAccessible(accessible),
		out:        stderrWriter(),
		clearOnRun: tty.IsStderrTerminal() && !accessible,
	}
}

// NewInputForm creates a themed, accessibility-aware single-input form.
func NewInputForm(input *huh.Input) *PromptForm {
	return NewForm(huh.NewGroup(input))
}

// NewSelectForm creates a themed, accessibility-aware single-select form.
func NewSelectForm[T comparable](selectField *huh.Select[T]) *PromptForm {
	return NewForm(huh.NewGroup(selectField))
}

// NewConfirmForm creates a themed, accessibility-aware single-confirm form.
func NewConfirmForm(confirm *huh.Confirm) *PromptForm {
	return NewForm(huh.NewGroup(confirm))
}

// Run runs the form and removes its rendered question when it exits.
func (f *PromptForm) Run() error {
	return f.run(func() error { return f.Form.Run() })
}

// RunWithContext runs the form with a context and removes its rendered question when it exits.
func (f *PromptForm) RunWithContext(ctx context.Context) error {
	return f.run(func() error { return f.Form.RunWithContext(ctx) })
}

func (f *PromptForm) run(runForm func() error) error {
	if !f.clearOnRun {
		fmt.Fprintln(f.out)
		return runForm()
	}
	fmt.Fprint(f.out, ansiSaveCursor)
	fmt.Fprintln(f.out)
	defer fmt.Fprint(f.out, ansiRestoreCursor, ansiClearScreenBelow)
	return runForm()
}

// IsCancelled reports whether err represents a deliberate user cancellation
// (Ctrl-C / Esc before form submission, i.e. huh.ErrUserAborted).
// Use this to distinguish graceful cancellation from genuine failures.
func IsCancelled(err error) bool {
	return errors.Is(err, huh.ErrUserAborted)
}
