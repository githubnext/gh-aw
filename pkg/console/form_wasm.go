//go:build js || wasm

package console

import "errors"

func RunForm(fields []FormField) error {
	return errors.New("interactive forms not available in Wasm")
}
