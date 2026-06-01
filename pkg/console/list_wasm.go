//go:build js || wasm

package console

import "errors"

func ShowInteractiveList(title string, items []ListItem) (string, error) {
	return "", errors.New("interactive list not available in Wasm")
}
