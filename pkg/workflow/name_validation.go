//go:build !js && !wasm

// This file provides package and image name validation utilities for agentic workflows.
//
// # Name Validation
//
// Package and image names passed to external tools (npm, pip, uv, docker) must not
// start with '-'. A leading '-' would be interpreted as a command-line flag by the
// downstream tool, causing unintended argument injection.
//
// Note: exec.Command uses argv directly (not sh -c), so this is argument injection,
// not shell injection. The risk is low — compilation runs on the developer's local
// machine with the developer's own privileges.

package workflow

import (
	"fmt"
	"strings"
)

// rejectHyphenPrefixPackages returns a ValidationError if any of the provided
// names starts with '-'. The kind parameter (e.g. "npx", "pip", "uv") is used
// in the error messages.
//
// Names starting with '-' would be interpreted as flags by the downstream CLI
// tool, constituting argument injection into the exec.Command call.
func rejectHyphenPrefixPackages(names []string, kind string) error {
	var invalid []string
	for _, name := range names {
		if strings.HasPrefix(name, "-") {
			invalid = append(invalid, fmt.Sprintf("%s package name '%s' is invalid: names must not start with '-'", kind, name))
		}
	}
	if len(invalid) == 0 {
		return nil
	}
	return NewValidationError(
		kind+".packages",
		fmt.Sprintf("%d invalid package names", len(invalid)),
		kind+" package names must not start with '-'",
		"Fix invalid package names:\n\n"+strings.Join(invalid, "\n"),
	)
}
