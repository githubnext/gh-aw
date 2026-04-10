//go:build js || wasm

// This file provides WASM/JS no-op stubs for Docker validation functions.
// The canonical (non-WASM) implementations live in docker_validation.go.
// If any function signatures change in docker_validation.go, this file must be updated to match.

package workflow

func validateDockerImage(image string, verbose bool) error {
	return nil
}
