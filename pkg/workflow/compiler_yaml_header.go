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
	if c.skipHeader {
		return
	}
	writeHeaderMetadataLine(yaml, data, frontmatterHash, bodyHash, secrets, actions, c)
	yaml.WriteString(GenerateWorkflowHeader(resolveWorkflowHeaderSource(data), "gh-aw", ""))
	writeHeaderCommentBlock(yaml, data, secrets, actions)
	yaml.WriteString("\n")
}

func writeHeaderMetadataLine(yaml *strings.Builder, data *WorkflowData, frontmatterHash string, bodyHash string, secrets []string, actions []string, c *Compiler) {
	if frontmatterHash != "" {
		writeLockMetadataLine(yaml, data, frontmatterHash, bodyHash, c)
	}
	manifest := NewGHAWManifest(secrets, actions, data.ActionResolutionFailures, data.DockerImagePins, data.Redirect, data.Skills, data.RawFrontmatter["on"])
	if manifestJSON, err := manifest.ToJSON(); err == nil {
		fmt.Fprintf(yaml, "# gh-aw-manifest: %s\n", manifestJSON)
	} else {
		compilerYamlHeaderLog.Printf("Failed to serialize gh-aw-manifest: %v. Safe update mode will not be available for future compilations of this workflow.", err)
	}
}

func writeLockMetadataLine(yaml *strings.Builder, data *WorkflowData, frontmatterHash string, bodyHash string, c *Compiler) {
	agentInfo := AgentMetadataInfo{}
	if data.EngineConfig != nil && data.EngineConfig.ID != "" {
		agentInfo.AgentID = data.EngineConfig.ID
	} else if data.AI != "" {
		agentInfo.AgentID = data.AI
	}
	if data.EngineConfig != nil && data.EngineConfig.Model != "" {
		agentInfo.AgentModel = data.EngineConfig.Model
	}
	if hasDedicatedDetectionAgent(data) {
		agentInfo.DetectionAgentID = data.SafeOutputs.ThreatDetection.EngineConfig.ID
		agentInfo.DetectionAgentModel = data.SafeOutputs.ThreatDetection.EngineConfig.Model
	}
	agentInfo.EngineVersions = collectEngineVersionsForMetadata(data)
	agentInfo.AgentImageRunner = resolveAgentImageRunnerIdentifier(data.RawFrontmatter)
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

func hasDedicatedDetectionAgent(data *WorkflowData) bool {
	return data.SafeOutputs != nil && data.SafeOutputs.ThreatDetection != nil && data.SafeOutputs.ThreatDetection.EngineConfig != nil
}

func resolveWorkflowHeaderSource(data *WorkflowData) string {
	if data.Source != "" {
		return data.Source
	}
	return "the corresponding .md file"
}

func writeHeaderCommentBlock(yaml *strings.Builder, data *WorkflowData, secrets []string, actions []string) {
	writeDescriptionComments(yaml, data.Description)
	writeSourceComment(yaml, data.Source)
	writeResolvedManifestComment(yaml, visibleImportedFiles(data.ImportedFiles), data.IncludedFiles)
	writeInlinedImportsComment(yaml, data.InlinedImports)
	writeFrontmatterEnvComments(yaml, data.EnvSources)
	writeSimpleCommentList(yaml, "Secrets used", secrets)
	writeSimpleCommentList(yaml, "Custom actions used", actions)
	writeSimpleCommentList(yaml, "Container images used", data.DockerImages)
	writeSingleValueComment(yaml, "Effective stop-time", data.StopTime)
	writeManualApprovalComment(yaml, data.ManualApproval)
}

func writeDescriptionComments(yaml *strings.Builder, description string) {
	if description == "" {
		return
	}
	for line := range strings.SplitSeq(strings.TrimSpace(stringutil.StripANSI(description)), "\n") {
		fmt.Fprintf(yaml, "# %s\n", strings.TrimSpace(line))
	}
}

func writeSourceComment(yaml *strings.Builder, source string) {
	if source == "" {
		return
	}
	yaml.WriteString("#\n")
	fmt.Fprintf(yaml, "# Source: %s\n", filepath.ToSlash(stringutil.StripANSI(source)))
}

func visibleImportedFiles(importedFiles []string) []string {
	var visible []string
	for _, file := range importedFiles {
		if !strings.HasPrefix(file, parser.BuiltinPathPrefix) {
			visible = append(visible, file)
		}
	}
	return visible
}

func writeResolvedManifestComment(yaml *strings.Builder, imports []string, includes []string) {
	if len(imports) == 0 && len(includes) == 0 {
		return
	}
	yaml.WriteString("#\n# Resolved workflow manifest:\n")
	writeCommentSubList(yaml, "Imports", imports)
	writeCommentSubList(yaml, "Includes", includes)
}

func writeCommentSubList(yaml *strings.Builder, title string, values []string) {
	if len(values) == 0 {
		return
	}
	fmt.Fprintf(yaml, "#   %s:\n", title)
	for _, value := range values {
		fmt.Fprintf(yaml, "#     - %s\n", filepath.ToSlash(stringutil.StripANSI(value)))
	}
}

func writeInlinedImportsComment(yaml *strings.Builder, inlinedImports bool) {
	if inlinedImports {
		yaml.WriteString("#\n# inlined-imports: true\n")
	}
}

func writeFrontmatterEnvComments(yaml *strings.Builder, envSources map[string]string) {
	if len(envSources) == 0 {
		return
	}
	yaml.WriteString("#\n# Frontmatter env variables:\n")
	for _, key := range sliceutil.SortedKeys(envSources) {
		fmt.Fprintf(yaml, "#   - %s: %s\n", key, envSources[key])
	}
}

func writeSimpleCommentList(yaml *strings.Builder, title string, values []string) {
	if len(values) == 0 {
		return
	}
	yaml.WriteString("#\n")
	fmt.Fprintf(yaml, "# %s:\n", title)
	for _, value := range values {
		fmt.Fprintf(yaml, "#   - %s\n", value)
	}
}

func writeSingleValueComment(yaml *strings.Builder, label string, value string) {
	if value == "" {
		return
	}
	yaml.WriteString("#\n")
	fmt.Fprintf(yaml, "# %s: %s\n", label, stringutil.StripANSI(value))
}

func writeManualApprovalComment(yaml *strings.Builder, manualApproval string) {
	if manualApproval == "" {
		return
	}
	yaml.WriteString("#\n")
	fmt.Fprintf(yaml, "# Manual approval required: environment '%s'\n", stringutil.StripANSI(manualApproval))
}
