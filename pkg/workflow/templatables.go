// Package workflow – templatable field helpers
//
// A "templatable" field is a safe-output config field that:
//   - Does NOT affect the generated .lock.yml file (i.e. it carries no
//     compile-time information that changes the workflow YAML structure).
//   - CAN be supplied as a literal value (bool/string/int …) OR as a
//     GitHub Actions expression ("${{ inputs.foo }}") that is evaluated at
//     runtime when the env var containing the JSON config is expanded.
//
// # Go side
//
// preprocessBoolFieldAsString must be called before YAML unmarshaling so
// that a struct field typed as *string can store both literal booleans
// ("true"/"false") and GitHub Actions expression strings.  Free-form
// string literals that are not expressions are rejected with an error.
//
// # JS side
//
// parseBoolTemplatable (in templatable.cjs) is the counterpart used by
// safe-output handlers when reading the JSON config at runtime.

package workflow

import (
	"fmt"
	"strings"

	"github.com/github/gh-aw/pkg/logger"
)

// preprocessBoolFieldAsString converts the value of a boolean config field
// to a string before YAML unmarshaling.  This lets struct fields typed as
// *string accept both literal boolean values (true/false) and GitHub Actions
// expression strings (e.g. "${{ inputs.draft-prs }}").
//
// If the value is a bool it is converted to "true" or "false".
// If the value is a string it must be a GitHub Actions expression (starts
// with "${{" and ends with "}}"); any other free-form string is rejected
// and an error is returned.
func preprocessBoolFieldAsString(configData map[string]any, fieldName string, log *logger.Logger) error {
	if configData == nil {
		return nil
	}
	if val, exists := configData[fieldName]; exists {
		switch v := val.(type) {
		case bool:
			if v {
				configData[fieldName] = "true"
			} else {
				configData[fieldName] = "false"
			}
			if log != nil {
				log.Printf("Converted %s bool to string before unmarshaling", fieldName)
			}
		case string:
			if !strings.HasPrefix(v, "${{") || !strings.HasSuffix(v, "}}") {
				return fmt.Errorf("field %q must be a boolean or a GitHub Actions expression (e.g. '${{ inputs.flag }}'), got string %q", fieldName, v)
			}
			// expression string is already in the correct form
		}
	}
	return nil
}

// buildTemplatableBoolEnvVar returns a YAML environment variable entry for a
// templatable boolean field. If value is a GitHub Actions expression it is
// embedded unquoted so that GitHub Actions can evaluate it at runtime;
// otherwise the literal string is quoted. Returns nil if value is nil.
func buildTemplatableBoolEnvVar(envVarName string, value *string) []string {
	if value == nil {
		return nil
	}
	v := *value
	if strings.HasPrefix(v, "${{") {
		return []string{fmt.Sprintf("          %s: %s\n", envVarName, v)}
	}
	return []string{fmt.Sprintf("          %s: %q\n", envVarName, v)}
}

// AddTemplatableBool adds a templatable boolean field to the handler config.
//
// The stored JSON value depends on the content of *value:
//   - "true"  → JSON boolean true   (backward-compatible with existing handlers)
//   - "false" → JSON boolean false
//   - any other string (GitHub Actions expression) → stored as a JSON string so
//     that GitHub Actions can evaluate it at runtime when the env var that
//     contains the JSON config is expanded
//   - nil → field is omitted
func (b *handlerConfigBuilder) AddTemplatableBool(key string, value *string) *handlerConfigBuilder {
	if value == nil {
		return b
	}
	switch *value {
	case "true":
		b.config[key] = true
	case "false":
		b.config[key] = false
	default:
		b.config[key] = *value // expression string – evaluated at runtime
	}
	return b
}
