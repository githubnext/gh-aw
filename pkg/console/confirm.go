//go:build !js && !wasm

package console

import (
	"charm.land/huh/v2"
	"github.com/github/gh-aw/pkg/styles"
)

// ConfirmAction shows an interactive confirmation dialog using Bubble Tea (huh)
// Returns true if the user confirms, false if they cancel or an error occurs
func ConfirmAction(title, affirmative, negative string) (bool, error) {
	var confirmed bool

	confirmForm := huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title(title).
				Affirmative(affirmative).
				Negative(negative).
				Value(&confirmed),
		),
	).WithTheme(styles.HuhTheme()).WithAccessible(IsAccessibleMode())

	if err := confirmForm.Run(); err != nil {
		return false, err
	}

	return confirmed, nil
}
