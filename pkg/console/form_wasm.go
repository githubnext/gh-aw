//go:build js || wasm

package console

import "fmt"

func RunForm(fields []FormField) error {
	return fmt.Errorf("interactive forms not available in Wasm")
}
