//go:build js || wasm

package console

import "fmt"

func ConfirmAction(title, affirmative, negative string) (bool, error) {
	return false, fmt.Errorf("interactive confirmation not available in Wasm")
}
