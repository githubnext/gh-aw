//go:build js || wasm

package console

import "errors"

func PromptSelect(title, description string, options []SelectOption) (string, error) {
	return "", errors.New("interactive selection not available in Wasm")
}

func PromptMultiSelect(title, description string, options []SelectOption, limit int) ([]string, error) {
	return nil, errors.New("interactive selection not available in Wasm")
}
