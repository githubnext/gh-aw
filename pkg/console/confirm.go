package console

import (
	"os"

	"github.com/charmbracelet/huh"
)

// isAccessibleMode detects if accessibility mode should be enabled based on environment variables
func isAccessibleMode() bool {
	return os.Getenv("ACCESSIBLE") != "" ||
		os.Getenv("TERM") == "dumb" ||
		os.Getenv("NO_COLOR") != ""
}

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
	).WithAccessible(isAccessibleMode())

	if err := confirmForm.Run(); err != nil {
		return false, err
	}

	return confirmed, nil
}
