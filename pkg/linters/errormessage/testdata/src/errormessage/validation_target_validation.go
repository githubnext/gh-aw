package errormessage

import (
	"errors"
	fmtalias "fmt"
)

func NewValidationError(field, value, reason, suggestion string) error {
	return errors.New(reason)
}

func badValidationFormat() error {
	return fmtalias.Errorf("invalid input") // want `use NewValidationError\(\.\.\.\) instead of fmt\.Errorf\(\.\.\.\) in validation files` `error message uses negative language without constructive guidance; include expected/requires/should/example details`
}

func badValidationWrap(err error) error {
	return fmtalias.Errorf("failed to parse config: %w", err) // want `use NewValidationError\(\.\.\.\) instead of fmt\.Errorf\(\.\.\.\) in validation files` `avoid generic 'failed to \.\.\.: %w' wrapping; add specific recovery guidance` `error message uses negative language without constructive guidance; include expected/requires/should/example details`
}

func badValidationErrorObject() error {
	return NewValidationError("tools.github", "", "invalid mode", "") // want `NewValidationError\(\.\.\.\) suggestion must not be empty`
}

func badValidationSuggestionNoExample() error {
	return NewValidationError("tools.github", "remote", "unsupported", "Set mode to local") // want `NewValidationError\(\.\.\.\) suggestion should include an example \(for example: YAML snippet\)`
}

func badValidationFormatAliased() error {
	return fmtalias.Errorf("invalid value") // want `use NewValidationError\(\.\.\.\) instead of fmt\.Errorf\(\.\.\.\) in validation files` `error message uses negative language without constructive guidance; include expected/requires/should/example details`
}

type formatter struct{}

func (formatter) Errorf(msg string, _ ...any) error { return errors.New(msg) }

func localShadowNotFmtPackage() error {
	fmt := formatter{}
	return fmt.Errorf("all good")
}
