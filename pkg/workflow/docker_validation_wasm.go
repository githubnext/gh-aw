//go:build js || wasm

package workflow

func validateDockerImage(image string, verbose bool, requireDocker bool) error {
	return nil
}
