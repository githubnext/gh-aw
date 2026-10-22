package cli

import (
	"fmt"
	"sort"
	"strings"

	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/setutil"
	"github.com/github/gh-aw/pkg/sliceutil"
	"github.com/github/gh-aw/pkg/workflow"
	"github.com/github/gh-aw/pkg/workflow/codemod"
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

			newContent, applied, err := codemod.Apply(content, func(f *codemod.Frontmatter) error {
				for _, key := range sliceutil.SortedKeys(unsafeKeys) {
					if err := f.Remove("engine", "env", key); err != nil {
						return err
					}
				}
				return f.PruneEmpty("engine", "env")
			})
			if applied {
				engineEnvSecretsCodemodLog.Printf("Removed unsafe engine.env secret keys: %v", sliceutil.MapKeys(unsafeKeys))
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

func allowedEngineEnvSecretKeys(engineID string) map[string]struct {
} {
	allowed := make(map[string]struct {
	})
	// Keep only required, engine-specific secret names here.
	// We intentionally exclude system secrets (for example GH_AW_GITHUB_TOKEN)
	// and optional secrets so this codemod only
	// preserves strict-mode-safe engine credential overrides.
	for _, req := range getSecretRequirementsForEngine(
		engineID,
		false, // includeSystemSecrets
		false, // includeOptional
	) {
		allowed[req.Name] = struct {
		}{}
	}
	// Also include all secrets returned by the engine's GetRequiredSecretNames so that
	// BYOK credentials (e.g. COPILOT_PROVIDER_API_KEY) are treated the same way as they
	// are during compile-time strict-mode validation and are not removed by this codemod.
	if engineID != "" {
		registry := workflow.GetGlobalEngineRegistry()
		if engine, err := registry.GetEngine(engineID); err == nil {
			// Use a minimal WorkflowData so we get only the engine's unconditional secrets.
			// GetRequiredSecretNames only adds extra secrets when non-nil MCP tools
			// (ParsedTools.GitHub, ParsedTools.Playwright, etc.) are set, or when
			// MCPScripts is populated. By passing empty Tools/ParsedTools we get just the
			// base engine secrets without any optional/conditional ones.
			minimalData := &workflow.WorkflowData{
				Tools:       map[string]any{},
				ParsedTools: &workflow.ToolsConfig{},
			}
			for _, name := range engine.GetRequiredSecretNames(minimalData) {
				allowed[name] = struct {
				}{}
			}
		} else {
			engineEnvSecretsCodemodLog.Printf("Could not look up engine '%s' for allowlist expansion: %v", engineID, err)
		}
	}
	return allowed
}

func findUnsafeEngineEnvSecretKeys(envMap map[string]any, allowed map[string]struct {
}) map[string]struct {
} {
	unsafe := make(map[string]struct {
	})
	for key, value := range envMap {
		if setutil.Contains(allowed, key) {
			continue
		}
		strVal, ok := value.(string)
		if !ok {
			continue
		}
		if len(workflow.ExtractSecretsFromMap(map[string]string{key: strVal})) > 0 {
			unsafe[key] = struct {
			}{}
		}
	}
	engineEnvSecretsCodemodLog.Printf("Found %d unsafe engine.env secret keys out of %d total keys", len(unsafe), len(envMap))
	return unsafe
}

// getTopLevelEnvSecretsGuidedErrorCodemod creates a codemod that emits a guided error
// when secrets are detected in the top-level env: block. Unlike the engine.env codemod
// which can auto-remove unsafe keys, top-level env: secrets cannot be stripped
// automatically because the token may be required by the workflow. Users must move
// the secret into engine-specific secret configuration instead.
func getTopLevelEnvSecretsGuidedErrorCodemod() Codemod {
	return Codemod{
		ID:           "top-level-env-secrets-guided-error",
		Name:         "Detect secrets in top-level env section (manual fix required)",
		Description:  "Detects secrets in the top-level env: block that will be leaked to the agent container in strict mode, and emits a guided error pointing users to engine-specific secret configuration.",
		IntroducedIn: "1.5.0",
		Guided:       true,
		Apply: func(content string, frontmatter map[string]any) (string, bool, error) {
			envValue, hasEnv := frontmatter["env"]
			if !hasEnv {
				return content, false, nil
			}
			envMap, ok := envValue.(map[string]any)
			if !ok {
				return content, false, nil
			}

			envStrings := make(map[string]string)
			for key, value := range envMap {
				if strValue, ok := value.(string); ok {
					envStrings[key] = strValue
				}
			}

			secretExpressions := workflow.ExtractSecretsFromMap(envStrings)
			if len(secretExpressions) == 0 {
				return content, false, nil
			}

			var secretRefs []string
			seenExpressions := make(map[string]struct {
			})
			for _, expr := range secretExpressions {
				if setutil.Contains(seenExpressions, expr) {
					continue
				}
				seenExpressions[expr] = struct {
				}{}
				secretRefs = append(secretRefs, expr)
			}
			sort.Strings(secretRefs)
			engineEnvSecretsCodemodLog.Printf("Found %d secret(s) in top-level env section: %v", len(secretRefs), secretRefs)

			return content, false, fmt.Errorf(
				"top-level env: contains secrets that will be leaked to the agent container. "+
					"Found: %s. "+
					"Manual fix required: move the secret into engine-specific secret configuration. "+
					"See: https://github.github.com/gh-aw/reference/engines/",
				strings.Join(secretRefs, ", "),
			)
		},
	}
}
