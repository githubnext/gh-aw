package workflow

import (
	"errors"
	"fmt"
	"strings"

	"github.com/github/gh-aw/pkg/constants"
)

// expandLinearTool converts the first-class tools.linear shorthand into the
// generic remote HTTP MCP representation used by all engines and the gateway.
func expandLinearTool(tools map[string]any) error {
	value, exists := tools["linear"]
	if !exists {
		return nil
	}
	if value == nil {
		value = map[string]any{}
	}
	config, ok := value.(map[string]any)
	if !ok {
		return NewValidationError(
			"tools.linear",
			fmt.Sprintf("%v", value),
			"'tools.linear' must be an object",
			"Example:\n\ntools:\n  linear:\n    token: ${{ secrets.LINEAR_API_KEY }}",
		)
	}

	token, err := validateLinearToolConfig(config)
	if err != nil {
		return err
	}
	linearMCP := map[string]any{
		"type": "http",
		"url":  constants.LinearMCPReadOnlyURL,
		"headers": map[string]any{
			"Authorization": "Bearer " + token,
		},
	}
	if allowed, exists := config["allowed"]; exists {
		linearMCP["allowed"] = allowed
	}
	if required, exists := config["required"]; exists {
		linearMCP["required"] = required
	}
	tools["linear"] = linearMCP
	return nil
}

func validateLinearToolConfig(config map[string]any) (string, error) {
	knownFields := map[string]struct{}{
		"token": {}, "allowed": {}, "required": {},
	}
	for field := range config {
		if _, known := knownFields[field]; !known {
			return "", NewValidationError(
				"tools.linear."+field,
				fmt.Sprintf("%v", config[field]),
				fmt.Sprintf("unknown Linear tool property %q", field),
				"Valid properties are: token, allowed, required.",
			)
		}
	}

	token := constants.LinearMCPDefaultTokenExpr
	if value, exists := config["token"]; exists {
		var ok bool
		token, ok = value.(string)
		if !ok || strings.TrimSpace(token) == "" {
			return "", NewValidationError(
				"tools.linear.token",
				fmt.Sprintf("%v", value),
				"'token' must be a GitHub Actions secret reference",
				"Example:\n\ntools:\n  linear:\n    token: ${{ secrets.CUSTOM_LINEAR_TOKEN }}",
			)
		}
	}
	if match := SecretExpressionPattern.FindString(token); match != token {
		return "", NewValidationError(
			"tools.linear.token",
			"[redacted]",
			"'token' must be a GitHub Actions secret reference so the Linear credential is not embedded in the compiled workflow",
			"Store the credential as a repository secret, then use:\n\ntools:\n  linear:\n    token: ${{ secrets.LINEAR_API_KEY }}",
		)
	}

	if value, exists := config["required"]; exists {
		if _, valid := value.(bool); !valid {
			return "", errors.New("tools.linear.required must be a boolean")
		}
	}
	if err := validateLinearAllowed(config["allowed"]); err != nil {
		return "", err
	}
	return token, nil
}

func validateLinearAllowed(value any) error {
	if value == nil {
		return nil
	}
	allowed, valid := value.([]any)
	if !valid || len(allowed) == 0 {
		return errors.New("tools.linear.allowed must be a non-empty array of tool names")
	}
	for _, tool := range allowed {
		name, valid := tool.(string)
		if !valid || strings.TrimSpace(name) == "" {
			return errors.New("tools.linear.allowed must contain only non-empty tool names")
		}
	}
	return nil
}
