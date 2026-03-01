//go:build !js && !wasm

package console

import (
	"errors"

	"github.com/charmbracelet/huh"
	"github.com/github/gh-aw/pkg/tty"
)

// PromptSecretInput shows an interactive password input prompt with masking
// The input is masked for security and includes validation
// Returns the entered secret value or an error
func PromptSecretInput(title, description string) (string, error) {
	// Check if stdin is a TTY - if not, we can't show interactive forms
	if !tty.IsStderrTerminal() {
		return "", errors.New("interactive input not available (not a TTY)")
	}

	var value string

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title(title).
				Description(description).
				EchoMode(huh.EchoModePassword). // Masks input for security
				Validate(func(s string) error {
					if len(s) == 0 {
						return errors.New("value cannot be empty")
					}
					return nil
				}).
				Value(&value),
		),
	).WithAccessible(IsAccessibleMode())

	if err := form.Run(); err != nil {
		return "", err
	}

	return value, nil
}
