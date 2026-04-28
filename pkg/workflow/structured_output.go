// Package workflow provides support for structured output mode in agentic workflows.
//
// Structured output mode allows workflows to declare a JSON Schema that constrains
// the agent's primary response. This enables deterministic machine-readable output
// from agent jobs, making multi-agent pipelines and data extraction workflows reliable.
//
// The feature works as follows:
//  1. Compile-time: gh aw compile validates the schema is well-formed JSON Schema.
//  2. Pre-agent: The runtime writes the schema to /tmp/gh-aw/structured-output-schema.json
//     and sets GH_AW_STRUCTURED_OUTPUT_SCHEMA so the agent knows to produce structured output.
//  3. Post-agent: A validation step checks the agent wrote a valid JSON file at
//     /tmp/gh-aw/structured-output.json and exposes it as a typed job output.

package workflow

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/github/gh-aw/pkg/logger"
	"github.com/santhosh-tekuri/jsonschema/v6"
)

var structuredOutputLog = logger.New("workflow:structured_output")

const (
	// StructuredOutputSchemaPath is the runtime path for the compiled schema file.
	StructuredOutputSchemaPath = "/tmp/gh-aw/structured-output-schema.json"

	// StructuredOutputFilePath is the runtime path where the agent must write its structured output.
	StructuredOutputFilePath = "/tmp/gh-aw/structured-output.json"
)

// StructuredOutputConfig holds configuration for structured output mode.
// Either Schema (inline) or SchemaFile (file reference) must be specified, not both.
type StructuredOutputConfig struct {
	// Schema is an inline JSON Schema object (draft-07 compatible).
	Schema map[string]any `json:"schema,omitempty" yaml:"schema,omitempty"`

	// SchemaFile is a repo-root-relative or workflow-relative path to a JSON Schema file.
	SchemaFile string `json:"schema-file,omitempty" yaml:"schema-file,omitempty"`

	// resolvedSchemaJSON holds the serialized JSON string of the resolved schema,
	// populated during compilation after reading schema-file if specified.
	resolvedSchemaJSON string
}

// GetResolvedSchemaJSON returns the resolved JSON schema as a compact string.
func (c *StructuredOutputConfig) GetResolvedSchemaJSON() string {
	return c.resolvedSchemaJSON
}

// HasStructuredOutput returns true when the workflow has structured output configured.
func HasStructuredOutput(data *WorkflowData) bool {
	return data != nil && data.StructuredOutputConfig != nil
}

// extractStructuredOutputConfig extracts and validates the structured-output frontmatter section.
// workflowDir is the directory containing the workflow file, used to resolve schema-file paths.
func extractStructuredOutputConfig(frontmatter map[string]any, workflowDir string) (*StructuredOutputConfig, error) {
	raw, exists := frontmatter["structured-output"]
	if !exists || raw == nil {
		return nil, nil
	}

	configMap, ok := raw.(map[string]any)
	if !ok {
		return nil, errors.New("structured-output must be an object")
	}

	config := &StructuredOutputConfig{}

	// Parse inline schema
	if schemaRaw, ok := configMap["schema"]; ok && schemaRaw != nil {
		if schemaMap, ok := schemaRaw.(map[string]any); ok {
			config.Schema = schemaMap
		} else {
			return nil, errors.New("structured-output.schema must be a JSON Schema object")
		}
	}

	// Parse schema-file reference
	if schemaFileRaw, ok := configMap["schema-file"]; ok && schemaFileRaw != nil {
		if schemaFileStr, ok := schemaFileRaw.(string); ok && schemaFileStr != "" {
			config.SchemaFile = schemaFileStr
		} else {
			return nil, errors.New("structured-output.schema-file must be a non-empty string path")
		}
	}

	// Validate: exactly one of schema or schema-file must be specified
	hasInline := config.Schema != nil
	hasFile := config.SchemaFile != ""
	if !hasInline && !hasFile {
		return nil, errors.New("structured-output requires either 'schema' (inline object) or 'schema-file' (file path)")
	}
	if hasInline && hasFile {
		return nil, errors.New("structured-output cannot specify both 'schema' and 'schema-file'")
	}

	// Load schema-file contents at compile time
	if hasFile {
		schemaPath := config.SchemaFile
		if !filepath.IsAbs(schemaPath) {
			schemaPath = filepath.Join(workflowDir, schemaPath)
		}
		schemaBytes, err := os.ReadFile(schemaPath)
		if err != nil {
			return nil, fmt.Errorf("structured-output.schema-file: cannot read %q: %w", config.SchemaFile, err)
		}
		var schemaMap map[string]any
		if err := json.Unmarshal(schemaBytes, &schemaMap); err != nil {
			return nil, fmt.Errorf("structured-output.schema-file: %q is not valid JSON: %w", config.SchemaFile, err)
		}
		config.Schema = schemaMap
	}

	// Validate that the schema is well-formed JSON Schema (draft-07)
	if err := validateJSONSchema(config.Schema); err != nil {
		return nil, fmt.Errorf("structured-output schema is not valid JSON Schema: %w", err)
	}

	// Serialize the resolved schema for embedding in generated workflow steps
	schemaJSON, err := json.Marshal(config.Schema)
	if err != nil {
		return nil, fmt.Errorf("structured-output: failed to serialize schema: %w", err)
	}
	config.resolvedSchemaJSON = string(schemaJSON)

	structuredOutputLog.Printf("Extracted structured-output config: schema=%d chars", len(config.resolvedSchemaJSON))
	return config, nil
}

// validateJSONSchema checks that the provided map is a valid JSON Schema using the
// santhosh-tekuri/jsonschema compiler (draft-07 compatible).
func validateJSONSchema(schema map[string]any) error {
	compiler := jsonschema.NewCompiler()

	// The compiler's AddResource accepts a pre-parsed document (any), not raw bytes.
	const schemaURL = "urn:gh-aw:structured-output-schema"
	if err := compiler.AddResource(schemaURL, schema); err != nil {
		return fmt.Errorf("invalid JSON Schema: %w", err)
	}
	if _, err := compiler.Compile(schemaURL); err != nil {
		return fmt.Errorf("invalid JSON Schema: %w", err)
	}
	return nil
}

// applyStructuredOutputEnvToMap adds structured-output related environment variables
// to the provided env map. It is safe to call when structured output is not configured
// (the function is a no-op in that case).
//
// The following env vars are set for non-detection runs:
//   - GH_AW_STRUCTURED_OUTPUT_SCHEMA: path to the JSON Schema file on disk
//   - GH_AW_STRUCTURED_OUTPUT_FILE:   path where the agent must write its JSON output
func applyStructuredOutputEnvToMap(env map[string]string, data *WorkflowData) {
	if !HasStructuredOutput(data) || data.IsDetectionRun {
		return
	}
	env["GH_AW_STRUCTURED_OUTPUT_SCHEMA"] = StructuredOutputSchemaPath
	env["GH_AW_STRUCTURED_OUTPUT_FILE"] = StructuredOutputFilePath
}

// generateStructuredOutputSchemaStep returns the YAML for a pre-agent step that writes
// the resolved JSON Schema to disk so the agent can discover it at runtime.
// Returns an empty string when structured output is not configured.
func generateStructuredOutputSchemaStep(data *WorkflowData) string {
	if !HasStructuredOutput(data) {
		return ""
	}

	schemaJSON := data.StructuredOutputConfig.GetResolvedSchemaJSON()
	if schemaJSON == "" {
		return ""
	}

	// Escape single quotes so the JSON can be safely embedded in a shell single-quoted argument.
	// json.Marshal produces compact JSON (no newlines), so the entire schema is one line.
	escaped := shellEscapeSingleQuote(schemaJSON)

	var sb strings.Builder
	sb.WriteString("      - name: Set up structured output schema\n")
	sb.WriteString("        run: |\n")
	sb.WriteString("          mkdir -p /tmp/gh-aw\n")
	fmt.Fprintf(&sb, "          printf '%%s' '%s' > %s\n", escaped, StructuredOutputSchemaPath)
	sb.WriteString("\n")

	structuredOutputLog.Printf("Generated structured-output schema write step (%d chars schema)", len(schemaJSON))
	return sb.String()
}

// generateStructuredOutputValidationStep returns the YAML for a post-agent step that
// reads the agent's structured output file, validates it is well-formed JSON, and
// exposes it as the `structured_output` step output for downstream consumption.
//
// The step always runs (if: always()) so it reports validation failures even when the
// agent step itself has failed.  Returns an empty string when structured output is not configured.
func generateStructuredOutputValidationStep(data *WorkflowData, actionPinFunc func(string, *WorkflowData) string) string {
	if !HasStructuredOutput(data) {
		return ""
	}

	actionPin := "actions/github-script@v7"
	if actionPinFunc != nil {
		pin := actionPinFunc("actions/github-script", data)
		if pin != "" {
			actionPin = pin
		}
	}

	// Build JavaScript validation script
	var script strings.Builder
	fmt.Fprintf(&script, "            const fs = require('fs');\n")
	fmt.Fprintf(&script, "            const outputPath = '%s';\n", StructuredOutputFilePath)
	fmt.Fprintf(&script, "            if (!fs.existsSync(outputPath)) {\n")
	fmt.Fprintf(&script, "              core.setFailed(`Structured output file not found: ${outputPath}. The agent must write its JSON response to this path.`);\n")
	fmt.Fprintf(&script, "              return;\n")
	fmt.Fprintf(&script, "            }\n")
	fmt.Fprintf(&script, "            const raw = fs.readFileSync(outputPath, 'utf8');\n")
	fmt.Fprintf(&script, "            let parsed;\n")
	fmt.Fprintf(&script, "            try {\n")
	fmt.Fprintf(&script, "              parsed = JSON.parse(raw);\n")
	fmt.Fprintf(&script, "            } catch (e) {\n")
	fmt.Fprintf(&script, "              core.setFailed(`Structured output is not valid JSON: ${e.message}`);\n")
	fmt.Fprintf(&script, "              return;\n")
	fmt.Fprintf(&script, "            }\n")
	fmt.Fprintf(&script, "            const compact = JSON.stringify(parsed);\n")
	fmt.Fprintf(&script, "            core.setOutput('structured_output', compact);\n")
	fmt.Fprintf(&script, "            core.info(`Structured output validated: ${compact.length} chars`);\n")

	var sb strings.Builder
	sb.WriteString("      - name: Validate structured output\n")
	sb.WriteString("        if: always()\n")
	sb.WriteString("        id: validate_structured_output\n")
	fmt.Fprintf(&sb, "        uses: %s\n", actionPin)
	sb.WriteString("        with:\n")
	sb.WriteString("          script: |\n")
	sb.WriteString(script.String())
	sb.WriteString("\n")

	return sb.String()
}

// shellEscapeSingleQuote escapes a string for use inside single quotes in a shell command.
// In single-quoted strings the only character that requires special handling is the
// single quote itself, which is represented as: '\”
func shellEscapeSingleQuote(s string) string {
	return strings.ReplaceAll(s, "'", `'\''`)
}
