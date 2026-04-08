package workflow

import (
	"fmt"
	"maps"
	"sort"
	"strings"

	"github.com/github/gh-aw/pkg/logger"
)

var runtimeStepGeneratorLog = logger.New("workflow:runtime_step_generator")

// GenerateRuntimeSetupSteps creates GitHub Actions steps for runtime setup
func GenerateRuntimeSetupSteps(requirements []RuntimeRequirement) []GitHubActionStep {
	runtimeStepGeneratorLog.Printf("Generating runtime setup steps: requirement_count=%d", len(requirements))
	runtimeSetupLog.Printf("Generating runtime setup steps for %d requirements", len(requirements))
	var steps []GitHubActionStep

	for _, req := range requirements {
		steps = append(steps, generateSetupStep(&req))

		// Add environment variable capture steps after setup actions for AWF chroot mode.
		// Most env vars are inherited via AWF_HOST_PATH, but Go is special.
		switch req.Runtime.ID {
		case "go":
			// GitHub Actions uses "trimmed" Go binaries that require GOROOT to be explicitly set.
			// Unlike other runtimes where PATH is sufficient, Go's trimmed binaries need GOROOT
			// for /proc/self/exe resolution. actions/setup-go does NOT export GOROOT to the
			// environment, so we must capture it explicitly.
			runtimeStepGeneratorLog.Print("Adding GOROOT capture step for chroot mode compatibility")
			steps = append(steps, generateEnvCaptureStep("GOROOT", "go env GOROOT"))
		}
		// Note: Java and .NET don't need capture steps anymore because:
		// - AWF_HOST_PATH captures the complete host PATH including $JAVA_HOME/bin and $DOTNET_ROOT
		// - AWF's entrypoint.sh exports PATH="${AWF_HOST_PATH}" which preserves all setup-* additions
	}

	runtimeStepGeneratorLog.Printf("Generated %d runtime setup steps", len(steps))
	return steps
}

// generateEnvCaptureStep creates a step to capture an environment variable and export it.
// This is required because some setup actions don't export env vars, but AWF chroot mode
// needs them to be set in the environment to pass them to the container.
func generateEnvCaptureStep(envVar string, captureCmd string) GitHubActionStep {
	return GitHubActionStep{
		fmt.Sprintf("      - name: Capture %s for AWF chroot mode", envVar),
		fmt.Sprintf("        run: echo \"%s=$(%s)\" >> \"$GITHUB_ENV\"", envVar, captureCmd),
	}
}

// generateSetupStep creates a setup step for a given runtime requirement
func generateSetupStep(req *RuntimeRequirement) GitHubActionStep {
	runtime := req.Runtime
	version := req.Version
	runtimeStepGeneratorLog.Printf("Generating setup step for runtime: %s, version=%s, if=%s", runtime.ID, version, req.IfCondition)
	runtimeSetupLog.Printf("Generating setup step for runtime: %s, version=%s, if=%s", runtime.ID, version, req.IfCondition)
	// Use default version if none specified
	if version == "" {
		version = runtime.DefaultVersion
	}

	// Use SHA-pinned action reference for security if available
	actionRef := GetActionPin(runtime.ActionRepo)

	// If no pin exists (custom action repo), use the action repo with its version
	if actionRef == "" {
		if runtime.ActionVersion != "" {
			actionRef = fmt.Sprintf("%s@%s", runtime.ActionRepo, runtime.ActionVersion)
		} else {
			// Fallback to just the repo name (shouldn't happen in practice)
			actionRef = runtime.ActionRepo
		}
	}

	step := GitHubActionStep{
		"      - name: Setup " + runtime.Name,
		"        uses: " + actionRef,
	}

	// Add if condition if specified
	if req.IfCondition != "" {
		step = append(step, "        if: "+req.IfCondition)
	}

	// Special handling for Go when go-mod-file is explicitly specified
	if runtime.ID == "go" && req.GoModFile != "" {
		step = append(step, "        with:")
		step = append(step, "          go-version-file: "+req.GoModFile)
		// Merge extra fields from runtime configuration and user's setup step
		allGoModExtraFields := make(map[string]string)
		maps.Copy(allGoModExtraFields, runtime.ExtraWithFields)
		for k, v := range req.ExtraFields {
			allGoModExtraFields[k] = formatYAMLValue(v)
		}
		var extraKeys []string
		for key := range allGoModExtraFields {
			extraKeys = append(extraKeys, key)
		}
		sort.Strings(extraKeys)
		for _, key := range extraKeys {
			step = append(step, fmt.Sprintf("          %s: %s", key, allGoModExtraFields[key]))
		}
		return step
	}

	// Add version field if we have a version
	if version != "" {
		step = append(step, "        with:")
		step = append(step, fmt.Sprintf("          %s: '%s'", runtime.VersionField, version))
	} else if runtime.ID == "uv" {
		// For uv without version, no with block needed (unless there are extra fields)
		if len(req.ExtraFields) == 0 {
			return step
		}
		step = append(step, "        with:")
	}

	// Merge extra fields from runtime configuration and user's setup step
	// User fields take precedence over runtime fields
	// Note: runtime.ExtraWithFields are pre-formatted strings, req.ExtraFields need formatting
	allExtraFields := make(map[string]string)

	// Add runtime extra fields (already formatted)
	maps.Copy(allExtraFields, runtime.ExtraWithFields)

	// Add user extra fields (need formatting), these override runtime fields
	for k, v := range req.ExtraFields {
		allExtraFields[k] = formatYAMLValue(v)
	}

	// Output merged extra fields in sorted key order for stable output
	var allKeys []string
	for key := range allExtraFields {
		allKeys = append(allKeys, key)
	}
	sort.Strings(allKeys)
	for _, key := range allKeys {
		step = append(step, fmt.Sprintf("          %s: %s", key, allExtraFields[key]))
		log.Printf("  Added extra field to runtime setup: %s = %s", key, allExtraFields[key])
	}

	return step
}

// GenerateMultiSecretValidationStep creates a GitHub Actions step that validates at least one
// of multiple secrets is available.
// secretNames: slice of secret names to validate (e.g., []string{"CODEX_API_KEY", "OPENAI_API_KEY"})
// engineName: the display name of the engine (e.g., "Codex")
// docsURL: URL to the documentation page for setting up the secret
// envOverrides: optional map of env var key to expression override (from engine.env); when set,
// the overridden expression is used instead of the default "${{ secrets.KEY }}" so the
// validation step checks the user-provided secret reference rather than the default one.
func GenerateMultiSecretValidationStep(secretNames []string, engineName, docsURL string, envOverrides map[string]string) GitHubActionStep {
	if len(secretNames) == 0 {
		// This is a programming error - engine configurations should always provide secrets
		// Log the error and return empty step to avoid breaking compilation
		runtimeStepGeneratorLog.Printf("ERROR: GenerateMultiSecretValidationStep called with empty secretNames for engine %s", engineName)
		return GitHubActionStep{}
	}

	// Build the step name
	stepName := fmt.Sprintf("      - name: Validate %s secret", strings.Join(secretNames, " or "))

	// Build the command to call the validation script
	// The script expects: SECRET_NAME1 [SECRET_NAME2 ...] ENGINE_NAME DOCS_URL
	// Use shellJoinArgs to properly escape multi-word engine names and special characters
	scriptArgs := append(secretNames, engineName, docsURL)
	scriptArgsStr := shellJoinArgs(scriptArgs)

	stepLines := []string{
		stepName,
		"        id: validate-secret",
		"        run: bash \"${RUNNER_TEMP}/gh-aw/actions/validate_multi_secret.sh\" " + scriptArgsStr,
		"        env:",
	}

	// Add env section with all secrets. When engine.env provides an override for a key,
	// use that expression (e.g. "${{ secrets.MY_ORG_TOKEN }}") so the validation step
	// validates the user-supplied secret instead of the default one.
	for _, secretName := range secretNames {
		expr := fmt.Sprintf("${{ secrets.%s }}", secretName)
		if envOverrides != nil {
			if override, ok := envOverrides[secretName]; ok {
				expr = override
			}
		}
		stepLines = append(stepLines, fmt.Sprintf("          %s: %s", secretName, expr))
	}

	return GitHubActionStep(stepLines)
}
