package workflow

import (
	"fmt"
	"sort"
)

// GenerateRuntimeSetupSteps creates GitHub Actions steps for runtime setup
func GenerateRuntimeSetupSteps(requirements []RuntimeRequirement) []GitHubActionStep {
	runtimeSetupLog.Printf("Generating runtime setup steps for %d requirements", len(requirements))
	var steps []GitHubActionStep

	for _, req := range requirements {
		steps = append(steps, generateSetupStep(&req))
	}

	return steps
}

// GenerateSerenaLanguageServiceSteps creates installation steps for Serena language services
// This is called after runtime detection to install the language servers needed by Serena
func GenerateSerenaLanguageServiceSteps(tools *ToolsConfig) []GitHubActionStep {
	runtimeSetupLog.Print("Generating Serena language service installation steps")
	var steps []GitHubActionStep

	// Check if Serena is configured
	if tools == nil || tools.Serena == nil {
		return steps
	}

	serenaConfig := tools.Serena

	// Collect all languages from the configuration
	var languages []string

	// Handle short syntax: ["go", "typescript"]
	if len(serenaConfig.ShortSyntax) > 0 {
		languages = serenaConfig.ShortSyntax
	} else if serenaConfig.Languages != nil {
		// Handle object syntax with languages field
		for langName := range serenaConfig.Languages {
			languages = append(languages, langName)
		}
	}

	// Sort languages alphabetically to ensure deterministic order
	sort.Strings(languages)

	runtimeSetupLog.Printf("Found %d Serena languages to install: %v", len(languages), languages)

	// Generate installation steps for each language service
	for _, lang := range languages {
		switch lang {
		case "go":
			// Install gopls for Go language service
			// Check if there's a custom gopls version specified
			goplsVersion := "latest"
			if serenaConfig.Languages != nil {
				if goConfig := serenaConfig.Languages["go"]; goConfig != nil && goConfig.GoplsVersion != "" {
					goplsVersion = goConfig.GoplsVersion
				}
			}
			steps = append(steps, GitHubActionStep{
				"      - name: Install Go language service (gopls)",
				fmt.Sprintf("        run: go install golang.org/x/tools/gopls@%s", goplsVersion),
			})
		case "typescript":
			// Install TypeScript language server
			steps = append(steps, GitHubActionStep{
				"      - name: Install TypeScript language service",
				"        run: npm install -g --silent typescript-language-server typescript",
			})
		case "python":
			// Install Python language server
			steps = append(steps, GitHubActionStep{
				"      - name: Install Python language service",
				"        run: pip install --quiet python-lsp-server",
			})
		case "java":
			// Java language service typically comes with the JDK setup
			// No additional installation needed
			runtimeSetupLog.Print("Java language service (jdtls) typically bundled with JDK, skipping explicit install")
		case "rust":
			// Install rust-analyzer for Rust language service
			steps = append(steps, GitHubActionStep{
				"      - name: Install Rust language service (rust-analyzer)",
				"        run: rustup component add rust-analyzer",
			})
		case "csharp":
			// C# language service typically comes with .NET SDK
			// No additional installation needed
			runtimeSetupLog.Print("C# language service (OmniSharp) typically bundled with .NET SDK, skipping explicit install")
		}
	}

	runtimeSetupLog.Printf("Generated %d Serena language service installation steps", len(steps))
	return steps
}

// generateSetupStep creates a setup step for a given runtime requirement
func generateSetupStep(req *RuntimeRequirement) GitHubActionStep {
	runtime := req.Runtime
	version := req.Version
	runtimeSetupLog.Printf("Generating setup step for runtime: %s, version=%s", runtime.ID, version)
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
		fmt.Sprintf("      - name: Setup %s", runtime.Name),
		fmt.Sprintf("        uses: %s", actionRef),
	}

	// Special handling for Go when go-mod-file is explicitly specified
	if runtime.ID == "go" && req.GoModFile != "" {
		step = append(step, "        with:")
		step = append(step, fmt.Sprintf("          go-version-file: %s", req.GoModFile))
		step = append(step, "          cache: true")
		// Add any extra fields from user's setup step (sorted for stable output)
		var extraKeys []string
		for key := range req.ExtraFields {
			extraKeys = append(extraKeys, key)
		}
		sort.Strings(extraKeys)
		for _, key := range extraKeys {
			valueStr := formatYAMLValue(req.ExtraFields[key])
			step = append(step, fmt.Sprintf("          %s: %s", key, valueStr))
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
	for k, v := range runtime.ExtraWithFields {
		allExtraFields[k] = v
	}

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

// formatYAMLValue formats a value for YAML output
func formatYAMLValue(value any) string {
	switch v := value.(type) {
	case string:
		// Quote strings if they contain special characters or look like non-string types
		if v == "true" || v == "false" || v == "null" {
			return fmt.Sprintf("'%s'", v)
		}
		// Check if it's a number
		if _, err := fmt.Sscanf(v, "%f", new(float64)); err == nil {
			return fmt.Sprintf("'%s'", v)
		}
		// Return as-is for simple strings, quote for complex ones
		return fmt.Sprintf("'%s'", v)
	case bool:
		if v {
			return "true"
		}
		return "false"
	case int:
		return fmt.Sprintf("%d", v)
	case int8:
		return fmt.Sprintf("%d", v)
	case int16:
		return fmt.Sprintf("%d", v)
	case int32:
		return fmt.Sprintf("%d", v)
	case int64:
		return fmt.Sprintf("%d", v)
	case uint:
		return fmt.Sprintf("%d", v)
	case uint8:
		return fmt.Sprintf("%d", v)
	case uint16:
		return fmt.Sprintf("%d", v)
	case uint32:
		return fmt.Sprintf("%d", v)
	case uint64:
		return fmt.Sprintf("%d", v)
	case float32:
		return fmt.Sprintf("%v", v)
	case float64:
		return fmt.Sprintf("%v", v)
	default:
		// For other types, convert to string and quote
		return fmt.Sprintf("'%v'", v)
	}
}
