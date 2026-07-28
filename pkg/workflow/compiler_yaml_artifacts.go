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

// generateMCPLogsArtifactUpload emits two steps that reliably capture MCP telemetry:
//  1. A best-effort chmod to make log files readable regardless of how the gateway
//     container wrote them (e.g. different UID in AWF isolation mode).
//  2. A dedicated actions/upload-artifact step for the mcp-logs directory, independent
//     of the unified agent artifact so telemetry is preserved even when the main upload
//     has issues.
//
// The artifact is named "mcp-logs-{sanitized-workflow-name}" (or
// "{prefix}mcp-logs-{name}" in workflow_call context).
// On ARC/DinD topologies the path is rewritten to ${{ runner.temp }}/gh-aw/mcp-logs/.
func (c *Compiler) generateMCPLogsArtifactUpload(yaml *strings.Builder, workflowName string, data *WorkflowData, prefix string) {
	sanitizedName := strings.ToLower(SanitizeWorkflowName(workflowName))
	artifactName := prefix + constants.MCPLogsArtifactBaseName + "-" + sanitizedName

	// Resolve the MCP logs path: on ARC/DinD, /tmp is not daemon-visible so logs
	// land under ${{ runner.temp }}/gh-aw (the same rewrite used for firewall logs).
	mcpLogsPath := constants.TmpMcpLogsDir
	if isArcDindTopology(data) {
		mcpLogsPath = constants.TmpMcpLogsDirExpr
	}

	compilerYamlArtifactsLog.Printf("Generating MCP logs chmod + dedicated artifact upload: artifact=%s path=%s", artifactName, mcpLogsPath)

	// Step 1: chmod – best-effort, non-fatal.
	// The MCP gateway runs with --user $(id -u):$(id -g) so files are normally
	// runner-owned, but AWF isolation or tool sub-containers may produce files
	// owned by a different user.  chmod -R a+rX restores read access without
	// requiring sudo, matching the non-rootless branch of print_firewall_logs.sh.
	yaml.WriteString("      - name: Fix MCP logs permissions\n")
	yaml.WriteString("        if: always()\n")
	yaml.WriteString("        continue-on-error: true\n")
	yaml.WriteString("        run: chmod -R a+rX /tmp/gh-aw/mcp-logs/ 2>/dev/null || true\n")

	// Step 2: dedicated artifact upload – always runs, ignores missing files so it
	// is a no-op on workflows where the gateway did not start.
	// Record the upload for step-order validation before writing YAML.
	c.stepOrderTracker.RecordArtifactUpload("Upload MCP logs", []string{mcpLogsPath})

	yaml.WriteString("      - name: Upload MCP logs\n")
	yaml.WriteString("        if: always()\n")
	yaml.WriteString("        continue-on-error: true\n")
	fmt.Fprintf(yaml, "        uses: %s\n", c.getActionPin("actions/upload-artifact"))
	yaml.WriteString("        with:\n")
	fmt.Fprintf(yaml, "          name: %s\n", artifactName)
	fmt.Fprintf(yaml, "          path: %s\n", mcpLogsPath)
	yaml.WriteString("          if-no-files-found: ignore\n")

	compilerYamlArtifactsLog.Printf("Generated MCP logs dedicated artifact upload step: %s", artifactName)
}
