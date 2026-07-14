package workflow

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/github/gh-aw/pkg/parser"
	"github.com/github/gh-aw/pkg/sliceutil"
	"github.com/github/gh-aw/pkg/stringutil"
)

// effectiveStrictMode computes the effective strict mode for a workflow.
// Priority: CLI flag (c.strictMode) > frontmatter strict field > default (true).
// This should be used when emitting metadata/env vars to correctly reflect the
// workflow's strictness as inferred from the source (frontmatter).
func (c *Compiler) effectiveStrictMode(frontmatter map[string]any) bool {
	if c.strictMode {
		// CLI flag takes precedence
		return true
	}
	if strictVal, exists := frontmatter["strict"]; exists {
		if strictBool, ok := strictVal.(bool); ok {
			return strictBool
		}
	}
	// Default: strict mode is on when no explicit setting
	return true
}

// effectiveSafeUpdate returns true when safe update mode should be enforced for
// the given workflow. Safe update mode is equivalent to strict mode: it is
// enabled whenever strict mode is active (CLI --strict flag, frontmatter
// strict: true, or the default). It can be disabled via the CLI --approve flag
// to approve all changes.
func (c *Compiler) effectiveSafeUpdate(data *WorkflowData) bool {
	if c.approve {
		return false
	}
	return c.effectiveStrictMode(data.RawFrontmatter)
}

// generateWorkflowHeader generates the YAML header section including comments
// for description, source, imports/includes, frontmatter-hash, stop-time, and manual-approval.
// All ANSI escape codes are stripped from the output.
// The gh-aw-metadata line is placed first for easy machine parsing.
func (c *Compiler) generateWorkflowHeader(yaml *strings.Builder, data *WorkflowData, frontmatterHash string, bodyHash string, secrets []string, actions []string) {
	// Skip the ASCII art banner in wasm/editor mode — it takes up too much space
	if c.skipHeader {
		return
	}

	// Add lock metadata as the very first line for easy machine parsing.
	// Single-line JSON format to minimize merge conflicts.
	if frontmatterHash != "" {
		agentInfo := AgentMetadataInfo{}
		// Agent ID: prefer EngineConfig.ID, fall back to legacy AI field
		if data.EngineConfig != nil && data.EngineConfig.ID != "" {
			agentInfo.AgentID = data.EngineConfig.ID
		} else if data.AI != "" {
			agentInfo.AgentID = data.AI
		}
		// Agent model: only include if statically configured
		if data.EngineConfig != nil && data.EngineConfig.Model != "" {
			agentInfo.AgentModel = data.EngineConfig.Model
		}
		// Detection agent info: only if threat detection has its own engine config
		if data.SafeOutputs != nil && data.SafeOutputs.ThreatDetection != nil && data.SafeOutputs.ThreatDetection.EngineConfig != nil {
			agentInfo.DetectionAgentID = data.SafeOutputs.ThreatDetection.EngineConfig.ID
			agentInfo.DetectionAgentModel = data.SafeOutputs.ThreatDetection.EngineConfig.Model
		}
		agentInfo.EngineVersions = collectEngineVersionsForMetadata(data)
		agentInfo.AgentImageRunner = resolveAgentImageRunnerIdentifier(data.RawFrontmatter)
		metadata := GenerateLockMetadata(LockHashInfo{FrontmatterHash: frontmatterHash, BodyHash: bodyHash}, data.StopTime, c.effectiveStrictMode(data.RawFrontmatter), agentInfo)
		if metadata.CompilerVersion == "" && c.GetActionTag() != "" {
			metadata.CompilerVersion = c.GetVersion()
		}
		metadataJSON, err := metadata.ToJSON()
		if err != nil {
			// Fallback to legacy format if JSON serialization fails
			fmt.Fprintf(yaml, "# frontmatter-hash: %s\n", frontmatterHash)
		} else {
			fmt.Fprintf(yaml, "# gh-aw-metadata: %s\n", metadataJSON)
		}
	}

	// Embed the gh-aw-manifest immediately after gh-aw-metadata for easy machine parsing.
	// The manifest records all secrets, external actions, container images, and frontmatter
	// skills detected at compile time so that subsequent compilations can perform safe update
	// enforcement.
	manifest := NewGHAWManifest(secrets, actions, data.ActionResolutionFailures, data.DockerImagePins, data.Redirect, data.Skills, data.RawFrontmatter["on"])
	if manifestJSON, err := manifest.ToJSON(); err == nil {
		fmt.Fprintf(yaml, "# gh-aw-manifest: %s\n", manifestJSON)
	} else {
		compilerYamlLog.Printf("Failed to serialize gh-aw-manifest: %v. Safe update mode will not be available for future compilations of this workflow.", err)
	}

	// Add workflow header with logo and instructions
	sourceFile := "the corresponding .md file"
	if data.Source != "" {
		sourceFile = data.Source
	}
	header := GenerateWorkflowHeader(sourceFile, "gh-aw", "")
	yaml.WriteString(header)

	// Add description comment if provided
	if data.Description != "" {
		cleanDescription := stringutil.StripANSI(data.Description)
		// Split description into lines and prefix each with "# "
		descriptionLines := strings.SplitSeq(strings.TrimSpace(cleanDescription), "\n")
		for line := range descriptionLines {
			fmt.Fprintf(yaml, "# %s\n", strings.TrimSpace(line))
		}
	}

	// Add source comment if provided
	if data.Source != "" {
		yaml.WriteString("#\n")
		cleanSource := stringutil.StripANSI(data.Source)
		// Normalize to Unix paths (forward slashes) for cross-platform compatibility
		cleanSource = filepath.ToSlash(cleanSource)
		fmt.Fprintf(yaml, "# Source: %s\n", cleanSource)
	}

	// Add manifest of imported/included files if any exist
	// Build a user-visible imports list by filtering out internal builtin engine paths
	// (e.g. "@builtin:engines/copilot.md") which are implementation details.
	var visibleImports []string
	for _, file := range data.ImportedFiles {
		if !strings.HasPrefix(file, parser.BuiltinPathPrefix) {
			visibleImports = append(visibleImports, file)
		}
	}

	if len(visibleImports) > 0 || len(data.IncludedFiles) > 0 {
		yaml.WriteString("#\n")
		yaml.WriteString("# Resolved workflow manifest:\n")

		if len(visibleImports) > 0 {
			yaml.WriteString("#   Imports:\n")
			for _, file := range visibleImports {
				cleanFile := stringutil.StripANSI(file)
				// Normalize to Unix paths (forward slashes) for cross-platform compatibility
				cleanFile = filepath.ToSlash(cleanFile)
				fmt.Fprintf(yaml, "#     - %s\n", cleanFile)
			}
		}

		if len(data.IncludedFiles) > 0 {
			yaml.WriteString("#   Includes:\n")
			for _, file := range data.IncludedFiles {
				cleanFile := stringutil.StripANSI(file)
				// Normalize to Unix paths (forward slashes) for cross-platform compatibility
				cleanFile = filepath.ToSlash(cleanFile)
				fmt.Fprintf(yaml, "#     - %s\n", cleanFile)
			}
		}
	}

	// Add inlined-imports comment to indicate the field was used at compile time
	if data.InlinedImports {
		yaml.WriteString("#\n")
		yaml.WriteString("# inlined-imports: true\n")
	}

	// Add frontmatter-declared env vars with source attribution.
	// Note: programmatically injected env vars (e.g. OTEL_* from OTLP config) are not listed here.
	if len(data.EnvSources) > 0 {
		yaml.WriteString("#\n")
		yaml.WriteString("# Frontmatter env variables:\n")
		// Sort keys for deterministic output
		keys := sliceutil.SortedKeys(data.EnvSources)
		for _, k := range keys {
			fmt.Fprintf(yaml, "#   - %s: %s\n", k, data.EnvSources[k])
		}
	}

	// Add list of secrets referenced in the workflow
	if len(secrets) > 0 {
		yaml.WriteString("#\n")
		yaml.WriteString("# Secrets used:\n")
		for _, s := range secrets {
			fmt.Fprintf(yaml, "#   - %s\n", s)
		}
	}

	// Add list of external custom actions referenced in the workflow
	if len(actions) > 0 {
		yaml.WriteString("#\n")
		yaml.WriteString("# Custom actions used:\n")
		for _, a := range actions {
			fmt.Fprintf(yaml, "#   - %s\n", a)
		}
	}

	// Add list of container images used in the workflow
	if len(data.DockerImages) > 0 {
		yaml.WriteString("#\n")
		yaml.WriteString("# Container images used:\n")
		for _, img := range data.DockerImages {
			fmt.Fprintf(yaml, "#   - %s\n", img)
		}
	}

	// Add stop-time comment if configured
	if data.StopTime != "" {
		yaml.WriteString("#\n")
		cleanStopTime := stringutil.StripANSI(data.StopTime)
		fmt.Fprintf(yaml, "# Effective stop-time: %s\n", cleanStopTime)
	}

	// Add manual-approval comment if configured
	if data.ManualApproval != "" {
		yaml.WriteString("#\n")
		cleanManualApproval := stringutil.StripANSI(data.ManualApproval)
		fmt.Fprintf(yaml, "# Manual approval required: environment '%s'\n", cleanManualApproval)
	}

	yaml.WriteString("\n")
}
