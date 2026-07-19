package workflow

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/parser"
	"github.com/github/gh-aw/pkg/sliceutil"
	"github.com/github/gh-aw/pkg/stringutil"
)

var compilerYamlHeaderLog = logger.New("workflow:compiler_yaml:header")

// generateWorkflowHeader generates the YAML header section including comments
// for description, source, imports/includes, frontmatter-hash, stop-time, and manual-approval.
// All ANSI escape codes are stripped from the output.
// The gh-aw-metadata line is placed first for easy machine parsing.
func (c *Compiler) generateWorkflowHeader(yaml *strings.Builder, data *WorkflowData, frontmatterHash string, bodyHash string, secrets []string, actions []string) {
	// Skip the ASCII art banner in wasm/editor mode — it takes up too much space
	if c.skipHeader {
		return
	}

	c.writeHeaderMetadata(yaml, data, frontmatterHash, bodyHash)
	writeHeaderManifest(yaml, data, secrets, actions)
	writeGeneratedWorkflowBanner(yaml, data)
	writeDescriptionAndSourceComments(yaml, data)
	writeResolvedWorkflowManifest(yaml, data)
	writeInlinedImportsComment(yaml, data)
	writeEnvSourcesComment(yaml, data)
	writeListComment(yaml, "Secrets used:", secrets)
	writeListComment(yaml, "Custom actions used:", actions)
	writeListComment(yaml, "Container images used:", data.DockerImages)
	writeStopAndManualApprovalComments(yaml, data)

	yaml.WriteString("\n")
}

func writeResolvedWorkflowManifest(yaml *strings.Builder, data *WorkflowData) {
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
}

func writeEnvSourcesComment(yaml *strings.Builder, data *WorkflowData) {
	if len(data.EnvSources) > 0 {
		yaml.WriteString("#\n")
		yaml.WriteString("# Frontmatter env variables:\n")
		keys := sliceutil.SortedKeys(data.EnvSources)
		for _, k := range keys {
			fmt.Fprintf(yaml, "#   - %s: %s\n", k, data.EnvSources[k])
		}
	}
}

func writeInlinedImportsComment(yaml *strings.Builder, data *WorkflowData) {
	if data.InlinedImports {
		yaml.WriteString("#\n")
		yaml.WriteString("# inlined-imports: true\n")
	}
}

func writeListComment(yaml *strings.Builder, title string, items []string) {
	if len(items) > 0 {
		yaml.WriteString("#\n")
		fmt.Fprintf(yaml, "# %s\n", title)
		for _, item := range items {
			fmt.Fprintf(yaml, "#   - %s\n", item)
		}
	}
}

func writeStopAndManualApprovalComments(yaml *strings.Builder, data *WorkflowData) {
	if data.StopTime != "" {
		yaml.WriteString("#\n")
		cleanStopTime := stringutil.StripANSI(data.StopTime)
		fmt.Fprintf(yaml, "# Effective stop-time: %s\n", cleanStopTime)
	}
	if data.ManualApproval != "" {
		yaml.WriteString("#\n")
		cleanManualApproval := stringutil.StripANSI(data.ManualApproval)
		fmt.Fprintf(yaml, "# Manual approval required: environment '%s'\n", cleanManualApproval)
	}
}

func (c *Compiler) writeHeaderMetadata(yaml *strings.Builder, data *WorkflowData, frontmatterHash string, bodyHash string) {
	if frontmatterHash == "" {
		return
	}
	agentInfo := buildHeaderAgentMetadata(data)
	metadata := GenerateLockMetadata(LockHashInfo{FrontmatterHash: frontmatterHash, BodyHash: bodyHash}, data.StopTime, c.effectiveStrictMode(data.RawFrontmatter), agentInfo)
	if metadata.CompilerVersion == "" && c.GetActionTag() != "" {
		metadata.CompilerVersion = c.GetVersion()
	}
	if metadataJSON, err := metadata.ToJSON(); err == nil {
		fmt.Fprintf(yaml, "# gh-aw-metadata: %s\n", metadataJSON)
	} else {
		fmt.Fprintf(yaml, "# frontmatter-hash: %s\n", frontmatterHash)
	}
}

func buildHeaderAgentMetadata(data *WorkflowData) AgentMetadataInfo {
	agentInfo := AgentMetadataInfo{}
	if data.EngineConfig != nil && data.EngineConfig.ID != "" {
		agentInfo.AgentID = data.EngineConfig.ID
	} else if data.AI != "" {
		agentInfo.AgentID = data.AI
	}
	if data.EngineConfig != nil && data.EngineConfig.Model != "" {
		agentInfo.AgentModel = data.EngineConfig.Model
	}
	if data.SafeOutputs != nil && data.SafeOutputs.ThreatDetection != nil && data.SafeOutputs.ThreatDetection.EngineConfig != nil {
		agentInfo.DetectionAgentID = data.SafeOutputs.ThreatDetection.EngineConfig.ID
		agentInfo.DetectionAgentModel = data.SafeOutputs.ThreatDetection.EngineConfig.Model
	}
	agentInfo.EngineVersions = collectEngineVersionsForMetadata(data)
	agentInfo.AgentImageRunner = resolveAgentImageRunnerIdentifier(data.RawFrontmatter)
	return agentInfo
}

func writeHeaderManifest(yaml *strings.Builder, data *WorkflowData, secrets []string, actions []string) {
	manifest := NewGHAWManifest(secrets, actions, data.ActionResolutionFailures, data.DockerImagePins, data.Redirect, data.Skills, data.RawFrontmatter["on"])
	if manifestJSON, err := manifest.ToJSON(); err == nil {
		fmt.Fprintf(yaml, "# gh-aw-manifest: %s\n", manifestJSON)
	} else {
		compilerYamlHeaderLog.Printf("Failed to serialize gh-aw-manifest: %v. Safe update mode will not be available for future compilations of this workflow.", err)
	}
}

func writeGeneratedWorkflowBanner(yaml *strings.Builder, data *WorkflowData) {
	sourceFile := "the corresponding .md file"
	if data.Source != "" {
		sourceFile = data.Source
	}
	yaml.WriteString(GenerateWorkflowHeader(sourceFile, "gh-aw", ""))
}

func writeDescriptionAndSourceComments(yaml *strings.Builder, data *WorkflowData) {
	if data.Description != "" {
		cleanDescription := stringutil.StripANSI(data.Description)
		descriptionLines := strings.SplitSeq(strings.TrimSpace(cleanDescription), "\n")
		for line := range descriptionLines {
			fmt.Fprintf(yaml, "# %s\n", strings.TrimSpace(line))
		}
	}
	if data.Source != "" {
		yaml.WriteString("#\n")
		cleanSource := filepath.ToSlash(stringutil.StripANSI(data.Source))
		fmt.Fprintf(yaml, "# Source: %s\n", cleanSource)
	}
}
