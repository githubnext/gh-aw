package cli

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/githubnext/gh-aw/pkg/console"
	"github.com/githubnext/gh-aw/pkg/logger"
)

var jqLog = logger.New("cli:jq")

// ApplyJqFilter applies a jq filter to JSON input
func ApplyJqFilter(jsonInput string, jqFilter string) (string, error) {
	jqLog.Printf("Applying jq filter: %s (input size: %d bytes)", jqFilter, len(jsonInput))

	// Check if jq is available
	jqPath, err := exec.LookPath("jq")
	if err != nil {
		jqLog.Printf("jq not found in PATH")
		fmt.Fprintln(os.Stderr, console.FormatErrorWithSuggestions(
			"jq not found in PATH",
			[]string{
				"Install jq to use filter functionality",
				"On macOS: brew install jq",
				"On Ubuntu/Debian: sudo apt-get install jq",
			},
		))
		return "", errors.New("jq not found in PATH")
	}
	jqLog.Printf("Found jq at: %s", jqPath)

	// Pipe through jq
	cmd := exec.Command(jqPath, jqFilter)
	cmd.Stdin = strings.NewReader(jsonInput)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		jqLog.Printf("jq filter failed: %v, stderr: %s", err, stderr.String())
		fmt.Fprintln(os.Stderr, console.FormatErrorMessage(fmt.Sprintf("jq filter failed: %v, stderr: %s", err, stderr.String())))
		return "", err
	}

	jqLog.Printf("jq filter succeeded (output size: %d bytes)", stdout.Len())
	return stdout.String(), nil
}
