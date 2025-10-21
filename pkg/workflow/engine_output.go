package workflow

import (
	"fmt"
	"strings"
)

// generateCleanupStep generates the cleanup step YAML for workspace files, excluding /tmp/gh-aw/ files
// Returns the YAML string and whether a cleanup step was generated
func generateCleanupStep(outputFiles []string) (string, bool) {
	// Filter to get only workspace files (exclude /tmp/gh-aw/ files)
	var workspaceFiles []string
	for _, file := range outputFiles {
		if !strings.HasPrefix(file, "/tmp/gh-aw/") {
			workspaceFiles = append(workspaceFiles, file)
		}
	}

	// Only generate cleanup step if there are workspace files to delete
	if len(workspaceFiles) == 0 {
		return "", false
	}

	var yaml strings.Builder
	yaml.WriteString("      - name: Clean up engine output files\n")
	yaml.WriteString("        run: |\n")
	for _, file := range workspaceFiles {
		fmt.Fprintf(&yaml, "          rm -fr %s\n", file)
	}

	return yaml.String(), true
}

// generateEngineOutputCollection generates a step that collects engine-declared output files as artifacts
func (c *Compiler) generateEngineOutputCollection(yaml *strings.Builder, engine CodingAgentEngine, data *WorkflowData) {
	outputFiles := engine.GetDeclaredOutputFiles()
	if len(outputFiles) == 0 {
		return
	}

	// Add secret redaction step before uploading artifacts
	// Pass the current YAML content to scan for secret references
	c.generateSecretRedactionStep(yaml, yaml.String())

	// Add secret-masking-steps (if any) after secret redaction before artifacts
	c.generateSecretMaskingSteps(yaml, data)

	// Create a single upload step that handles all declared output files
	// The action will ignore missing files automatically with if-no-files-found: ignore
	yaml.WriteString("      - name: Upload engine output files\n")
	yaml.WriteString("        uses: actions/upload-artifact@v4\n")
	yaml.WriteString("        with:\n")
	yaml.WriteString("          name: agent_outputs\n")

	// Create the path list for all declared output files
	yaml.WriteString("          path: |\n")
	for _, file := range outputFiles {
		yaml.WriteString("            " + file + "\n")
	}

	yaml.WriteString("          if-no-files-found: ignore\n")

	// Add cleanup step to remove output files after upload
	// Only clean files under the workspace, ignore files in /tmp/gh-aw/
	cleanupYaml, hasCleanup := generateCleanupStep(outputFiles)
	if hasCleanup {
		yaml.WriteString(cleanupYaml)
	}
}
