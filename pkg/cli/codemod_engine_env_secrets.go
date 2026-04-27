package cli

import (
	"strings"

	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/workflow"
)

var engineEnvSecretsCodemodLog = logger.New("cli:codemod_engine_env_secrets")

// getEngineEnvSecretsCodemod creates a codemod that removes unsafe secret-bearing entries
// from engine.env while preserving allowed engine-required secret overrides.
func getEngineEnvSecretsCodemod() Codemod {
	return Codemod{
		ID:           "engine-env-secrets-to-engine-config",
		Name:         "Remove unsafe secrets from engine.env",
		Description:  "Removes secret-bearing engine.env entries that are not required engine secret overrides, preventing strict-mode leaks.",
		IntroducedIn: "0.26.0",
		Apply: func(content string, frontmatter map[string]any) (string, bool, error) {
			engineValue, hasEngine := frontmatter["engine"]
			if !hasEngine {
				return content, false, nil
			}

			engineMap, ok := engineValue.(map[string]any)
			if !ok {
				return content, false, nil
			}

			envAny, hasEnv := engineMap["env"]
			if !hasEnv {
				return content, false, nil
			}

			envMap, ok := envAny.(map[string]any)
			if !ok {
				return content, false, nil
			}

			engineID := extractEngineIDForCodemod(frontmatter, engineMap)
			allowed := allowedEngineEnvSecretKeys(engineID)
			unsafeKeys := findUnsafeEngineEnvSecretKeys(envMap, allowed)
			if len(unsafeKeys) == 0 {
				return content, false, nil
			}

			newContent, applied, err := applyFrontmatterLineTransform(content, func(lines []string) ([]string, bool) {
				updated, modified := removeUnsafeEngineEnvKeys(lines, unsafeKeys)
				if !modified {
					return lines, false
				}
				cleaned := removeEmptyEngineEnvBlock(updated)
				return cleaned, true
			})
			if applied {
				engineEnvSecretsCodemodLog.Printf("Removed unsafe engine.env secret keys: %v", mapKeys(unsafeKeys))
			}
			return newContent, applied, err
		},
	}
}

func extractEngineIDForCodemod(frontmatter map[string]any, engineMap map[string]any) string {
	if id, ok := engineMap["id"].(string); ok && id != "" {
		engineEnvSecretsCodemodLog.Printf("Extracted engine ID from engine.id: %s", id)
		return id
	}
	if runtimeAny, hasRuntime := engineMap["runtime"]; hasRuntime {
		if runtimeMap, ok := runtimeAny.(map[string]any); ok {
			if id, ok := runtimeMap["id"].(string); ok && id != "" {
				engineEnvSecretsCodemodLog.Printf("Extracted engine ID from engine.runtime.id: %s", id)
				return id
			}
		}
	}
	if id, ok := frontmatter["engine"].(string); ok && id != "" {
		engineEnvSecretsCodemodLog.Printf("Extracted engine ID from frontmatter engine field: %s", id)
		return id
	}
	engineEnvSecretsCodemodLog.Print("No engine ID found, using empty string for secret key lookup")
	return ""
}

func allowedEngineEnvSecretKeys(engineID string) map[string]bool {
	allowed := make(map[string]bool)
	// Keep only required, engine-specific secret names here.
	// We intentionally exclude system secrets (for example GH_AW_GITHUB_TOKEN)
	// and optional secrets so this codemod only
	// preserves strict-mode-safe engine credential overrides.
	for _, req := range getSecretRequirementsForEngine(
		engineID,
		false, // includeSystemSecrets
		false, // includeOptional
	) {
		allowed[req.Name] = true
	}
	return allowed
}

func findUnsafeEngineEnvSecretKeys(envMap map[string]any, allowed map[string]bool) map[string]bool {
	unsafe := make(map[string]bool)
	for key, value := range envMap {
		if allowed[key] {
			continue
		}
		strVal, ok := value.(string)
		if !ok {
			continue
		}
		if len(workflow.ExtractSecretsFromMap(map[string]string{key: strVal})) > 0 {
			unsafe[key] = true
		}
	}
	engineEnvSecretsCodemodLog.Printf("Found %d unsafe engine.env secret keys out of %d total keys", len(unsafe), len(envMap))
	return unsafe
}

func removeUnsafeEngineEnvKeys(lines []string, unsafeKeys map[string]bool) ([]string, bool) {
	result := make([]string, 0, len(lines))
	modified := false

	inEngine := false
	engineIndent := ""
	inEnv := false
	envIndent := ""
	removingKey := false
	removingKeyIndent := ""

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		indent := getIndentation(line)

		if isTopLevelKey(line) && strings.HasPrefix(trimmed, "engine:") {
			inEngine = true
			engineIndent = indent
			inEnv = false
			removingKey = false
			result = append(result, line)
			continue
		}

		if inEngine && len(trimmed) > 0 && !strings.HasPrefix(trimmed, "#") && len(indent) <= len(engineIndent) {
			inEngine = false
			inEnv = false
			removingKey = false
		}

		if inEngine && !inEnv && strings.HasPrefix(trimmed, "env:") && strings.TrimSpace(strings.TrimPrefix(trimmed, "env:")) == "" {
			inEnv = true
			envIndent = indent
			removingKey = false
			result = append(result, line)
			continue
		}

		if inEnv && len(trimmed) > 0 && !strings.HasPrefix(trimmed, "#") && len(indent) <= len(envIndent) {
			inEnv = false
			removingKey = false
		}

		if inEnv && removingKey {
			if len(trimmed) == 0 {
				continue
			}
			if strings.HasPrefix(trimmed, "#") && len(indent) > len(removingKeyIndent) {
				continue
			}
			if len(indent) > len(removingKeyIndent) {
				continue
			}
			removingKey = false
		}

		if inEnv && !removingKey && len(trimmed) > 0 && !strings.HasPrefix(trimmed, "#") && len(indent) > len(envIndent) {
			key := parseYAMLMapKey(trimmed)
			if key != "" && unsafeKeys[key] {
				modified = true
				removingKey = true
				removingKeyIndent = indent
				continue
			}
		}

		result = append(result, line)
	}

	return result, modified
}

func removeEmptyEngineEnvBlock(lines []string) []string {
	result := make([]string, 0, len(lines))
	for i := range lines {
		line := lines[i]
		trimmed := strings.TrimSpace(line)
		if trimmed == "env:" {
			envIndent := getIndentation(line)
			hasValues := false
			j := i + 1
			for ; j < len(lines); j++ {
				t := strings.TrimSpace(lines[j])
				if len(t) == 0 {
					continue
				}
				if len(getIndentation(lines[j])) <= len(envIndent) {
					break
				}
				hasValues = true
				break
			}
			if !hasValues {
				continue
			}
		}
		result = append(result, line)
	}
	return result
}

func mapKeys(m map[string]bool) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
