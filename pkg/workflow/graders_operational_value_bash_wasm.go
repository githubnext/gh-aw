//go:build js || wasm

package workflow

import "errors"

func validateOperationalValueEvaluatorBash(string) error {
	return errors.New("Bash syntax validation is not available in Wasm")
}
