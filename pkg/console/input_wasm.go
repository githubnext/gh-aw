//go:build js || wasm

package console

import "errors"

func PromptInput(title, description, placeholder string) (string, error) {
	return "", errors.New("interactive input not available in Wasm")
}

func PromptSecretInput(title, description string) (string, error) {
	return "", errors.New("interactive input not available in Wasm")
}

func PromptInputWithValidation(title, description, placeholder string, validate func(string) error) (string, error) {
	return "", errors.New("interactive input not available in Wasm")
}
