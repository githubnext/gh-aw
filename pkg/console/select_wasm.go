//go:build js || wasm

package console

import "fmt"

func PromptSelect(title, description string, options []SelectOption) (string, error) {
	return "", fmt.Errorf("interactive selection not available in Wasm")
}

func PromptMultiSelect(title, description string, options []SelectOption, limit int) ([]string, error) {
	return nil, fmt.Errorf("interactive selection not available in Wasm")
}
