package workflow

import (
	"fmt"
	"strings"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/logger"
)

var compilerYamlArtifactsLog = logger.New("workflow:compiler_yaml_artifacts")

// generateExtractAccessLogs is a legacy method that no longer does anything
// Network filtering is now handled at the workflow level
func (c *Compiler) generateExtractAccessLogs(yaml *strings.Builder, tools map[string]any) {
	// No proxy tools anymore - network filtering is handled at workflow level
}

// generateUploadAccessLogs is a legacy method that no longer does anything
// Network filtering is now handled at the workflow level
func (c *Compiler) generateUploadAccessLogs(yaml *strings.Builder, tools map[string]any) {
	// No proxy tools anymore - network filtering is handled at workflow level
}

// generateUnifiedArtifactUpload generates a single step that uploads all agent job artifacts
// This consolidates multiple individual upload steps into one, improving workflow readability
// and reliability. The step always runs (even on cancellation) and ignores missing files.
// prefix is prepended to the artifact name to avoid clashes in workflow_call context.
func (c *Compiler) generateUnifiedArtifactUpload(yaml *strings.Builder, paths []string, prefix string) {
	if len(paths) == 0 {
		compilerYamlArtifactsLog.Print("No paths to upload, skipping unified artifact upload")
		return
	}

	compilerYamlArtifactsLog.Printf("Generating unified artifact upload with %d paths", len(paths))

	artifactName := prefix + "agent"

	// Record the unified upload so the step-order validator can verify it comes after
	// secret redaction, covering all collected paths in a single check.
	c.stepOrderTracker.RecordArtifactUpload("Upload agent artifacts", paths)

	yaml.WriteString("      - name: Upload agent artifacts\n")
	yaml.WriteString("        if: always()\n")
	yaml.WriteString("        continue-on-error: true\n")
	fmt.Fprintf(yaml, "        uses: %s\n", c.getActionPin("actions/upload-artifact"))
	yaml.WriteString("        with:\n")
	fmt.Fprintf(yaml, "          name: %s\n", artifactName)

	// Write paths as multi-line YAML string
	yaml.WriteString("          path: |\n")
	for _, path := range paths {
		fmt.Fprintf(yaml, "            %s\n", path)
	}

	yaml.WriteString("          if-no-files-found: ignore\n")

	compilerYamlArtifactsLog.Printf("Generated unified artifact upload step with %d paths", len(paths))
}

// generateObservabilityArtifactUploads emits small, dedicated observability artifacts
// for the firewall and MCP gateway logs in addition to the unified agent artifact.
// Keeping these logs in named artifacts makes post-run security/debug analysis
// resilient to consumers that request only a specific observability artifact set.
func (c *Compiler) generateObservabilityArtifactUploads(yaml *strings.Builder, data *WorkflowData, prefix string) {
	mcpLogsDir := constants.TmpMcpLogsDir
	if isArcDindTopology(data) {
		mcpLogsDir = constants.GhAwRootDir + "/mcp-logs/"
	}
	c.generateDedicatedArtifactUpload(yaml, "Upload MCP observability logs", prefix+constants.MCPLogsArtifactName, []string{mcpLogsDir})

	if !isFirewallEnabled(data) {
		return
	}

	firewallPaths := []string{
		constants.AWFProxyLogsDir + "/",
		constants.AWFAuditDir + "/",
		constants.AWFReflectFilePath,
		constants.AWFConfigFilePath,
	}
	if isArcDindTopology(data) {
		firewallPaths = []string{
			constants.AWFProxyLogsDirExpr + "/",
			constants.AWFAuditDirExpr + "/",
			constants.AWFReflectFilePathExpr,
			constants.AWFConfigFilePathExpr,
		}
	}
	c.generateDedicatedArtifactUpload(yaml, "Upload firewall observability logs", prefix+constants.FirewallAuditArtifactName, firewallPaths)
}

func (c *Compiler) generateDedicatedArtifactUpload(yaml *strings.Builder, stepName string, artifactName string, paths []string) {
	if len(paths) == 0 {
		return
	}

	c.stepOrderTracker.RecordArtifactUpload(stepName, paths)

	fmt.Fprintf(yaml, "      - name: %s\n", stepName)
	yaml.WriteString("        if: always()\n")
	yaml.WriteString("        continue-on-error: true\n")
	fmt.Fprintf(yaml, "        uses: %s\n", c.getActionPin("actions/upload-artifact"))
	yaml.WriteString("        with:\n")
	fmt.Fprintf(yaml, "          name: %s\n", artifactName)
	yaml.WriteString("          path: |\n")
	for _, path := range paths {
		fmt.Fprintf(yaml, "            %s\n", path)
	}
	yaml.WriteString("          if-no-files-found: ignore\n")
}
