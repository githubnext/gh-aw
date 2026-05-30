//go:build js || wasm

package console

import "fmt"

func ShowInteractiveList(title string, items []ListItem) (string, error) {
	return "", fmt.Errorf("interactive list not available in Wasm")
}
