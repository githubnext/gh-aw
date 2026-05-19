package cli

import (
	"fmt"
	"slices"
	"strings"

	"github.com/github/gh-aw/pkg/logger"
)

var llmGatewayPortCodemodLog = logger.New("cli:codemod_llm_gateway_port")

const defaultLLMGatewayPort = 8080

// getSupportsLLMGatewayToLLMGatewayPortCodemod migrates supportsLLMGateway: true
// to llmGatewayPort: 8080.
func getSupportsLLMGatewayToLLMGatewayPortCodemod() Codemod {
	return Codemod{
		ID:           "supports-llm-gateway-to-llm-gateway-port",
		Name:         "Replace supportsLLMGateway with llmGatewayPort",
		Description:  "Replaces deprecated 'supportsLLMGateway: true' with 'llmGatewayPort: 8080'.",
		IntroducedIn: "0.15.0",
		Apply: func(content string, frontmatter map[string]any) (string, bool, error) {
			if !hasSupportsLLMGatewayTrue(frontmatter) {
				return content, false, nil
			}
			newContent, applied, err := applyFrontmatterLineTransform(content, replaceSupportsLLMGatewayTrue)
			if applied {
				llmGatewayPortCodemodLog.Print("Replaced supportsLLMGateway with llmGatewayPort")
			}
			return newContent, applied, err
		},
	}
}

func hasSupportsLLMGatewayTrue(value any) bool {
	switch v := value.(type) {
	case map[string]any:
		for key, child := range v {
			if key == "supportsLLMGateway" {
				if enabled, ok := child.(bool); ok && enabled {
					return true
				}
			}
			if hasSupportsLLMGatewayTrue(child) {
				return true
			}
		}
	case []any:
		return slices.ContainsFunc(v, hasSupportsLLMGatewayTrue)
	}
	return false
}

func replaceSupportsLLMGatewayTrue(lines []string) ([]string, bool) {
	result := make([]string, 0, len(lines))
	modified := false

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if valuePartRaw, ok := strings.CutPrefix(trimmed, "supportsLLMGateway:"); ok {
			valuePart := strings.TrimSpace(valuePartRaw)
			comment := ""
			if idx := strings.Index(valuePart, "#"); idx >= 0 {
				comment = strings.TrimSpace(valuePart[idx:])
				valuePart = strings.TrimSpace(valuePart[:idx])
			}

			if valuePart == "true" {
				newLine := fmt.Sprintf("%sllmGatewayPort: %d", getIndentation(line), defaultLLMGatewayPort)
				if comment != "" {
					newLine += " " + comment
				}
				result = append(result, newLine)
				modified = true
				llmGatewayPortCodemodLog.Printf("Replaced supportsLLMGateway on line %d", i+1)
				continue
			}
		}

		result = append(result, line)
	}

	return result, modified
}
