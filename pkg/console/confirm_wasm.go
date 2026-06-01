//go:build js || wasm

package console

import "errors"

func ConfirmAction(title, affirmative, negative string) (bool, error) {
	return false, errors.New("interactive confirmation not available in Wasm")
}
