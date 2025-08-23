package workflow

import (
	"strings"
)

// generateEngineOutputCollection generates a step that collects engine-declared output files as artifacts
func (c *Compiler) generateEngineOutputCollection(yaml *strings.Builder, engine AgenticEngine) {
	outputFiles := engine.GetDeclaredOutputFiles()
	if len(outputFiles) == 0 {
		return
	}

	// Create a single upload step that handles all declared output files
	// The action will ignore missing files automatically with if-no-files-found: ignore
	yaml.WriteString("      - name: Upload engine output files\n")
	yaml.WriteString("        if: always()\n")
	yaml.WriteString("        uses: actions/upload-artifact@v4\n")
	yaml.WriteString("        with:\n")
	yaml.WriteString("          name: agent_outputs\n")

	// Create the path list for all declared output files
	yaml.WriteString("          path: |\n")
	for _, file := range outputFiles {
		yaml.WriteString("            " + file + "\n")
	}

	yaml.WriteString("          if-no-files-found: ignore\n")
}
