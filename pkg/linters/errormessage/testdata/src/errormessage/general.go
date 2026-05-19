package errormessage

import "fmt"

func badNegativeLanguage() error {
	return fmt.Errorf("cannot run task") // want `error message uses negative language without constructive guidance; include expected/requires/should/example details`
}

func badInvalidLanguage() error {
	return fmt.Errorf("invalid input") // want `error message uses negative language without constructive guidance; include expected/requires/should/example details`
}

func goodConstructiveLanguage() error {
	return fmt.Errorf("invalid repo format — expected owner/repo format, for example: github/gh-aw")
}

func notAnErrorCall() string {
	return fmt.Sprintf("cannot run task")
}
