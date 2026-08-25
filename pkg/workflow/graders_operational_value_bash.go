//go:build !js && !wasm

package workflow

import (
	"errors"
	"os/exec"
	"strings"
)

func validateOperationalValueEvaluatorBash(evaluatorContent string) error {
	cmd := exec.Command("bash", "-n")
	cmd.Stdin = strings.NewReader(evaluatorContent)
	output, err := cmd.CombinedOutput()
	if err == nil {
		return nil
	}
	message := strings.TrimSpace(string(output))
	if message == "" {
		message = err.Error()
	}
	return errors.New(message)
}
