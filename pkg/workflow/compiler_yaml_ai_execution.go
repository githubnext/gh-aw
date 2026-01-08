package workflow

import (
	"encoding/json"
	"fmt"
	"strings"
)

// generateEngineExecutionSteps generates the GitHub Actions steps for executing the AI engine
func (c *Compiler) generateEngineExecutionSteps(yaml *strings.Builder, data *WorkflowData, engine CodingAgentEngine, logFile string) {

	steps := engine.GetExecutionSteps(data, logFile)

	for _, step := range steps {
		for _, line := range step {
			yaml.WriteString(line + "\n")
		}
	}
}

// generateLogParsing generates a step that parses the agent's logs and adds them to the step summary
func (c *Compiler) generateLogParsing(yaml *strings.Builder, engine CodingAgentEngine) {
	parserScriptName := engine.GetLogParserScriptId()
	if parserScriptName == "" {
		// Skip log parsing if engine doesn't provide a parser
		compilerYamlLog.Printf("Skipping log parsing: engine %s has no parser script", engine.GetID())
		return
	}

	compilerYamlLog.Printf("Generating log parsing step for engine: %s (parser=%s)", engine.GetID(), parserScriptName)

	logParserScript := GetLogParserScript(parserScriptName)
	if logParserScript == "" {
		// Skip if parser script not found
		compilerYamlLog.Printf("Warning: parser script %s not found, skipping log parsing", parserScriptName)
		return
	}

	// Get the log file path for parsing (may be different from stdout/stderr log)
	logFileForParsing := engine.GetLogFileForParsing()

	yaml.WriteString("      - name: Parse agent logs for step summary\n")
	yaml.WriteString("        if: always()\n")
	fmt.Fprintf(yaml, "        uses: %s\n", GetActionPin("actions/github-script"))
	yaml.WriteString("        env:\n")
	fmt.Fprintf(yaml, "          GH_AW_AGENT_OUTPUT: %s\n", logFileForParsing)
	yaml.WriteString("        with:\n")
	yaml.WriteString("          script: |\n")

	// Use the setup_globals helper to store GitHub Actions objects in global scope
	yaml.WriteString("            const { setupGlobals } = require('" + SetupActionDestination + "/setup_globals.cjs');\n")
	yaml.WriteString("            setupGlobals(core, github, context, exec, io);\n")
	// Load log parser script from external file using require()
	yaml.WriteString("            const { main } = require('/tmp/gh-aw/actions/" + parserScriptName + ".cjs');\n")
	yaml.WriteString("            await main();\n")
}

// generateSafeInputsLogParsing generates a step that parses safe-inputs logs and adds them to the step summary
func (c *Compiler) generateSafeInputsLogParsing(yaml *strings.Builder) {
	compilerYamlLog.Print("Generating safe-inputs log parsing step")

	yaml.WriteString("      - name: Parse safe-inputs logs for step summary\n")
	yaml.WriteString("        if: always()\n")
	fmt.Fprintf(yaml, "        uses: %s\n", GetActionPin("actions/github-script"))
	yaml.WriteString("        with:\n")
	yaml.WriteString("          script: |\n")

	// Use the setup_globals helper to store GitHub Actions objects in global scope
	yaml.WriteString("            const { setupGlobals } = require('" + SetupActionDestination + "/setup_globals.cjs');\n")
	yaml.WriteString("            setupGlobals(core, github, context, exec, io);\n")
	// Load safe-inputs log parser script from external file using require()
	yaml.WriteString("            const { main } = require('/tmp/gh-aw/actions/parse_safe_inputs_logs.cjs');\n")
	yaml.WriteString("            await main();\n")
}

// convertGoPatternToJavaScript converts a Go regex pattern to JavaScript-compatible format
// This removes Go's (?i) inline case-insensitive flag since JavaScript doesn't support it
func (c *Compiler) convertGoPatternToJavaScript(goPattern string) string {
	// Convert (?i) inline case-insensitive flag by removing it
	// JavaScript RegExp will be created with "gi" flags to handle case insensitivity
	if strings.HasPrefix(goPattern, "(?i)") {
		return goPattern[4:] // Remove (?i) prefix
	}
	return goPattern
}

// convertErrorPatternsToJavaScript converts a slice of Go error patterns to JavaScript-compatible patterns
func (c *Compiler) convertErrorPatternsToJavaScript(goPatterns []ErrorPattern) []ErrorPattern {
	jsPatterns := make([]ErrorPattern, len(goPatterns))
	for i, pattern := range goPatterns {
		jsPatterns[i] = ErrorPattern{
			Pattern:      c.convertGoPatternToJavaScript(pattern.Pattern),
			LevelGroup:   pattern.LevelGroup,
			MessageGroup: pattern.MessageGroup,
			Description:  pattern.Description,
		}
	}
	return jsPatterns
}

// generateErrorValidation generates a step that validates the agent's logs for errors
func (c *Compiler) generateErrorValidation(yaml *strings.Builder, engine CodingAgentEngine, data *WorkflowData) {
	// Get the log file path for validation (may be different from stdout/stderr log)
	logFileForValidation := engine.GetLogFileForParsing()

	yaml.WriteString("      - name: Validate agent logs for errors\n")
	yaml.WriteString("        if: always()\n")
	fmt.Fprintf(yaml, "        uses: %s\n", GetActionPin("actions/github-script"))
	yaml.WriteString("        env:\n")
	fmt.Fprintf(yaml, "          GH_AW_AGENT_OUTPUT: %s\n", logFileForValidation)

	// Pass the engine ID so JavaScript can load appropriate patterns
	fmt.Fprintf(yaml, "          GH_AW_ENGINE_ID: %s\n", engine.GetID())

	// Add custom error patterns if configured by user
	if data.EngineConfig != nil && len(data.EngineConfig.ErrorPatterns) > 0 {
		// Convert user-configured patterns to JavaScript-compatible format
		jsCompatiblePatterns := c.convertErrorPatternsToJavaScript(data.EngineConfig.ErrorPatterns)
		patternsJSON, err := json.Marshal(jsCompatiblePatterns)
		if err == nil {
			fmt.Fprintf(yaml, "          GH_AW_CUSTOM_ERROR_PATTERNS: %q\n", string(patternsJSON))
		}
	}

	yaml.WriteString("        with:\n")
	yaml.WriteString("          script: |\n")

	// Use the setup_globals helper to store GitHub Actions objects in global scope
	yaml.WriteString("            const { setupGlobals } = require('" + SetupActionDestination + "/setup_globals.cjs');\n")
	yaml.WriteString("            setupGlobals(core, github, context, exec, io);\n")
	// Load error validation script from external file using require()
	yaml.WriteString("            const { main } = require('/tmp/gh-aw/actions/validate_errors.cjs');\n")
	yaml.WriteString("            await main();\n")
}
