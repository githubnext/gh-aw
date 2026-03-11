package workflow

import (
	"fmt"
	"strings"

	"github.com/github/gh-aw/pkg/logger"
)

var engineOutputLog = logger.New("workflow:engine_output")

// RedactedURLsLogPath is the path where redacted URL domains are logged during sanitization
const RedactedURLsLogPath = "/tmp/gh-aw/redacted-urls.log"

// AgentStepSummaryPath is the path used as GITHUB_STEP_SUMMARY inside the agent sandbox.
// The agent writes its step summary content to this file, which is then appended to the
// real $GITHUB_STEP_SUMMARY after secret redaction.
const AgentStepSummaryPath = "/tmp/gh-aw/agent-step-summary.md"

// generateCleanupStep generates the cleanup step YAML for workspace files, excluding /tmp/gh-aw/ files
// Returns the YAML string and whether a cleanup step was generated
func generateCleanupStep(outputFiles []string) (string, bool) {
	if engineOutputLog.Enabled() {
		engineOutputLog.Printf("Generating cleanup step for %d output files", len(outputFiles))
	}
	// Filter to get only workspace files (exclude /tmp/gh-aw/ files)
	var workspaceFiles []string
	for _, file := range outputFiles {
		if !strings.HasPrefix(file, "/tmp/gh-aw/") {
			workspaceFiles = append(workspaceFiles, file)
		}
	}

	// Only generate cleanup step if there are workspace files to delete
	if len(workspaceFiles) == 0 {
		engineOutputLog.Print("No workspace files to clean up")
		return "", false
	}

	engineOutputLog.Printf("Generated cleanup step for %d workspace files", len(workspaceFiles))

	var yaml strings.Builder
	yaml.WriteString("      - name: Clean up engine output files\n")
	yaml.WriteString("        run: |\n")
	for _, file := range workspaceFiles {
		fmt.Fprintf(&yaml, "          rm -fr %s\n", file)
	}

	return yaml.String(), true
}

// getEngineArtifactPaths returns the file paths declared by the engine that should be
// included in the unified agent artifact. The redacted URLs log is always appended.
// Returns nil if the engine declares no output files.
func getEngineArtifactPaths(engine CodingAgentEngine) []string {
	outputFiles := engine.GetDeclaredOutputFiles()
	if len(outputFiles) == 0 {
		return nil
	}
	// Always include the redacted URLs log alongside engine output files.
	// This file is created during content sanitization when URLs were redacted.
	outputFiles = append(outputFiles, RedactedURLsLogPath)
	return outputFiles
}

// generateEngineOutputCleanup generates the workspace cleanup step for engine-declared output
// files. It does NOT upload an artifact — paths are added to the unified agent artifact instead.
func (c *Compiler) generateEngineOutputCleanup(yaml *strings.Builder, engine CodingAgentEngine) {
	outputFiles := engine.GetDeclaredOutputFiles()
	if len(outputFiles) == 0 {
		engineOutputLog.Print("No engine output files to clean up")
		return
	}

	engineOutputLog.Printf("Generating engine output cleanup for %d files", len(outputFiles))

	// Add cleanup step to remove workspace output files after they have been collected.
	// Only files outside /tmp/gh-aw/ need explicit removal.
	cleanupYaml, hasCleanup := generateCleanupStep(outputFiles)
	if hasCleanup {
		yaml.WriteString(cleanupYaml)
	}
}
