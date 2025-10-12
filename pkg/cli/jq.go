package cli

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

// ApplyJqFilter applies a jq filter to JSON input
func ApplyJqFilter(jsonInput string, jqFilter string) (string, error) {
	// Check if jq is available
	jqPath, err := exec.LookPath("jq")
	if err != nil {
		return "", fmt.Errorf("jq not found in PATH")
	}

	// Pipe through jq
	cmd := exec.Command(jqPath, jqFilter)
	cmd.Stdin = strings.NewReader(jsonInput)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("jq filter failed: %w, stderr: %s", err, stderr.String())
	}

	return stdout.String(), nil
}
