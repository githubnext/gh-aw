package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/githubnext/gh-aw/pkg/console"
	"github.com/githubnext/gh-aw/pkg/logger"
	"github.com/githubnext/gh-aw/pkg/workflow"
)

var generateActionMetadataLog = logger.New("cli:generate_action_metadata")

// ActionMetadata represents metadata extracted from a JavaScript file
type ActionMetadata struct {
	Name         string
	Description  string
	Filename     string // e.g., "noop.cjs"
	ActionName   string // e.g., "noop"
	Inputs       []ActionInput
	Outputs      []ActionOutput
	Dependencies []string
}

// ActionInput represents an input parameter
type ActionInput struct {
	Name        string
	Description string
	Required    bool
	Default     string
}

// ActionOutput represents an output parameter
type ActionOutput struct {
	Name        string
	Description string
}

// GenerateActionMetadataCommand generates action.yml and README.md files for JavaScript modules
// Uses the agent-output schema to discover which safe output types should have custom actions
func GenerateActionMetadataCommand() error {
	jsDir := "pkg/workflow/js"
	actionsDir := "actions"
	schemaPath := "schemas/agent-output.json"

	generateActionMetadataLog.Print("Starting schema-driven action metadata generation")

	// Load the safe output schema
	schema, err := workflow.LoadSafeOutputSchema(schemaPath)
	if err != nil {
		return fmt.Errorf("failed to load schema: %w", err)
	}

	// Extract safe output types from the schema
	safeOutputTypes := workflow.GetSafeOutputTypes(schema)

	// Filter to only types that should have custom actions
	var targetTypes []workflow.SafeOutputTypeSchema
	for _, typeSchema := range safeOutputTypes {
		if workflow.ShouldGenerateCustomAction(typeSchema.TypeName) {
			targetTypes = append(targetTypes, typeSchema)
		}
	}

	fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("🔍 Found %d safe output types in schema", len(safeOutputTypes))))
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("🎯 Generating actions for %d types...", len(targetTypes))))

	generatedCount := 0
	for _, typeSchema := range targetTypes {
		filename := workflow.GetJavaScriptFilename(typeSchema.TypeName)
		jsPath := filepath.Join(jsDir, filename)

		// Check if JavaScript file exists
		if _, err := os.Stat(jsPath); os.IsNotExist(err) {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("⚠ Skipping %s: JavaScript file not found", typeSchema.TypeName)))
			continue
		}

		// Read file content directly from filesystem
		contentBytes, err := os.ReadFile(jsPath)
		if err != nil {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("⚠ Skipping %s: %s", filename, err.Error())))
			continue
		}
		content := string(contentBytes)

		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("\n📦 Processing: %s (%s)", typeSchema.TypeName, typeSchema.Title)))

		// Extract metadata from both schema and JavaScript file
		metadata, err := extractActionMetadataFromSchema(filename, content, typeSchema)
		if err != nil {
			fmt.Fprintln(os.Stderr, console.FormatErrorMessage(fmt.Sprintf("✗ Failed to extract metadata from %s: %s", filename, err.Error())))
			continue
		}

		// Create action directory (using hyphenated name for GitHub Actions convention)
		actionDirName := workflow.GetActionDirectoryName(typeSchema.TypeName)
		actionDir := filepath.Join(actionsDir, actionDirName)
		if err := os.MkdirAll(actionDir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", actionDir, err)
		}

		// Create src directory
		srcDir := filepath.Join(actionDir, "src")
		if err := os.MkdirAll(srcDir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", srcDir, err)
		}

		// Generate action.yml
		if err := generateActionYml(actionDir, metadata); err != nil {
			fmt.Fprintln(os.Stderr, console.FormatErrorMessage(fmt.Sprintf("✗ Failed to generate action.yml: %s", err.Error())))
			continue
		}
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("  ✓ Generated action.yml"))

		// Generate README.md
		if err := generateReadme(actionDir, metadata); err != nil {
			fmt.Fprintln(os.Stderr, console.FormatErrorMessage(fmt.Sprintf("✗ Failed to generate README.md: %s", err.Error())))
			continue
		}
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("  ✓ Generated README.md"))

		// Transform source file to use FILES pattern for bundling
		transformedContent := transformSourceForBundling(content, metadata.Dependencies)
		srcPath := filepath.Join(srcDir, "index.js")
		if err := os.WriteFile(srcPath, []byte(transformedContent), 0644); err != nil {
			return fmt.Errorf("failed to write source file: %w", err)
		}
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("  ✓ Generated source file for bundling"))

		generatedCount++
	}

	if generatedCount == 0 {
		return fmt.Errorf("no actions were generated")
	}

	fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("\n✨ Successfully generated %d action(s)", generatedCount)))
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage("\nNext steps:"))
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage("  1. Review the generated action.yml and README.md files"))
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage("  2. Update dependency mapping in pkg/cli/actions_build_command.go"))
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage("  3. Run 'make actions-build' to build the actions"))
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage("  4. Test the actions in a workflow"))

	return nil
}

// extractActionMetadataFromSchema combines schema information with JavaScript analysis
func extractActionMetadataFromSchema(filename, content string, typeSchema workflow.SafeOutputTypeSchema) (*ActionMetadata, error) {
	generateActionMetadataLog.Printf("Extracting metadata from schema for %s", typeSchema.TypeName)

	// Use schema description if available, otherwise fallback to JSDoc
	description := typeSchema.Description
	if description == "" {
		description = extractDescription(content)
	}
	if description == "" {
		description = fmt.Sprintf("Process %s safe output", typeSchema.TypeName)
	}

	// Use schema title for action name
	name := typeSchema.Title
	if name == "" {
		name = generateHumanReadableName(typeSchema.TypeName)
	}

	// Extract inputs from JavaScript (core.getInput calls)
	inputs := extractInputs(content)

	// Add standard token input if not already present
	hasToken := false
	for _, input := range inputs {
		if input.Name == "token" {
			hasToken = true
			break
		}
	}
	if !hasToken {
		inputs = append([]ActionInput{{
			Name:        "token",
			Description: "GitHub token for API authentication",
			Required:    true,
			Default:     "",
		}}, inputs...)
	}

	// Extract outputs from JavaScript (core.setOutput calls)
	outputs := extractOutputs(content)

	// Extract dependencies from require() calls
	dependencies := extractDependencies(content)

	metadata := &ActionMetadata{
		Name:         name,
		Description:  description,
		Filename:     filename,
		ActionName:   typeSchema.TypeName,
		Inputs:       inputs,
		Outputs:      outputs,
		Dependencies: dependencies,
	}

	generateActionMetadataLog.Printf("Extracted metadata: %d inputs, %d outputs, %d dependencies",
		len(inputs), len(outputs), len(dependencies))

	return metadata, nil
}

// extractDescription extracts description from JSDoc comment
func extractDescription(content string) string {
	// Look for JSDoc block comment at the start of main() or file
	jsdocRegex := regexp.MustCompile(`/\*\*\s*\n\s*\*\s*([^\n]+)`)
	matches := jsdocRegex.FindStringSubmatch(content)
	if len(matches) > 1 {
		return strings.TrimSpace(matches[1])
	}
	return ""
}

// generateHumanReadableName converts action name to human-readable format
func generateHumanReadableName(actionName string) string {
	// Replace underscores with spaces and capitalize words
	words := strings.Split(actionName, "_")
	for i, word := range words {
		if len(word) > 0 {
			words[i] = strings.ToUpper(word[:1]) + word[1:]
		}
	}
	return strings.Join(words, " ")
}

// extractInputs extracts input parameters from core.getInput() calls
func extractInputs(content string) []ActionInput {
	var inputs []ActionInput
	seen := make(map[string]bool)

	// Match core.getInput('name') or core.getInput("name")
	inputRegex := regexp.MustCompile(`core\.getInput\(['"]([^'"]+)['"]\)`)
	matches := inputRegex.FindAllStringSubmatch(content, -1)

	for _, match := range matches {
		if len(match) > 1 {
			inputName := match[1]
			if !seen[inputName] {
				inputs = append(inputs, ActionInput{
					Name:        inputName,
					Description: fmt.Sprintf("Input parameter: %s", inputName),
					Required:    false,
					Default:     "",
				})
				seen[inputName] = true
			}
		}
	}

	// Sort inputs by name for consistency
	sort.Slice(inputs, func(i, j int) bool {
		return inputs[i].Name < inputs[j].Name
	})

	return inputs
}

// extractOutputs extracts output parameters from core.setOutput() calls
func extractOutputs(content string) []ActionOutput {
	var outputs []ActionOutput
	seen := make(map[string]bool)

	// Match core.setOutput('name', ...) or core.setOutput("name", ...)
	outputRegex := regexp.MustCompile(`core\.setOutput\(['"]([^'"]+)['"]`)
	matches := outputRegex.FindAllStringSubmatch(content, -1)

	for _, match := range matches {
		if len(match) > 1 {
			outputName := match[1]
			if !seen[outputName] {
				outputs = append(outputs, ActionOutput{
					Name:        outputName,
					Description: fmt.Sprintf("Output parameter: %s", outputName),
				})
				seen[outputName] = true
			}
		}
	}

	// Sort outputs by name for consistency
	sort.Slice(outputs, func(i, j int) bool {
		return outputs[i].Name < outputs[j].Name
	})

	return outputs
}

// extractDependencies extracts require() dependencies
func extractDependencies(content string) []string {
	var deps []string
	seen := make(map[string]bool)

	// Match require('./filename.cjs') or require("./filename.cjs")
	requireRegex := regexp.MustCompile(`require\(['"]\.\/([^'"]+\.cjs)['"]\)`)
	matches := requireRegex.FindAllStringSubmatch(content, -1)

	for _, match := range matches {
		if len(match) > 1 {
			dep := match[1]
			if !seen[dep] {
				deps = append(deps, dep)
				seen[dep] = true
			}
		}
	}

	// Sort dependencies for consistency
	sort.Strings(deps)

	return deps
}

// generateActionYml generates an action.yml file
func generateActionYml(actionDir string, metadata *ActionMetadata) error {
	var content strings.Builder

	content.WriteString(fmt.Sprintf("name: '%s'\n", metadata.Name))
	content.WriteString(fmt.Sprintf("description: '%s'\n", metadata.Description))
	content.WriteString("author: 'GitHub Next'\n\n")

	// Add inputs
	if len(metadata.Inputs) > 0 {
		content.WriteString("inputs:\n")
		for _, input := range metadata.Inputs {
			content.WriteString(fmt.Sprintf("  %s:\n", input.Name))
			content.WriteString(fmt.Sprintf("    description: '%s'\n", input.Description))
			content.WriteString(fmt.Sprintf("    required: %t\n", input.Required))
			if input.Default != "" {
				content.WriteString(fmt.Sprintf("    default: '%s'\n", input.Default))
			}
		}
		content.WriteString("\n")
	}

	// Add outputs
	if len(metadata.Outputs) > 0 {
		content.WriteString("outputs:\n")
		for _, output := range metadata.Outputs {
			content.WriteString(fmt.Sprintf("  %s:\n", output.Name))
			content.WriteString(fmt.Sprintf("    description: '%s'\n", output.Description))
		}
		content.WriteString("\n")
	}

	// Add runs section
	content.WriteString("runs:\n")
	content.WriteString("  using: 'node20'\n")
	content.WriteString("  main: 'index.js'\n\n")

	// Add branding
	content.WriteString("branding:\n")
	content.WriteString("  icon: 'package'\n")
	content.WriteString("  color: 'blue'\n")

	// Write to file
	ymlPath := filepath.Join(actionDir, "action.yml")
	if err := os.WriteFile(ymlPath, []byte(content.String()), 0644); err != nil {
		return fmt.Errorf("failed to write action.yml: %w", err)
	}

	return nil
}

// generateReadme generates a README.md file
func generateReadme(actionDir string, metadata *ActionMetadata) error {
	var content strings.Builder

	content.WriteString(fmt.Sprintf("# %s\n\n", metadata.Name))
	content.WriteString(fmt.Sprintf("%s\n\n", metadata.Description))

	content.WriteString("## Overview\n\n")
	content.WriteString(fmt.Sprintf("This action is generated from `pkg/workflow/js/%s` and provides functionality ", metadata.Filename))
	content.WriteString("for GitHub Agentic Workflows.\n\n")

	// Usage section
	content.WriteString("## Usage\n\n")
	content.WriteString("```yaml\n")
	content.WriteString(fmt.Sprintf("- uses: ./actions/%s\n", metadata.ActionName))
	if len(metadata.Inputs) > 0 {
		content.WriteString("  with:\n")
		for _, input := range metadata.Inputs {
			content.WriteString(fmt.Sprintf("    %s: 'value'  # %s\n", input.Name, input.Description))
		}
	}
	content.WriteString("```\n\n")

	// Inputs section
	if len(metadata.Inputs) > 0 {
		content.WriteString("## Inputs\n\n")
		for _, input := range metadata.Inputs {
			content.WriteString(fmt.Sprintf("### `%s`\n\n", input.Name))
			content.WriteString(fmt.Sprintf("**Description**: %s\n\n", input.Description))
			content.WriteString(fmt.Sprintf("**Required**: %t\n\n", input.Required))
			if input.Default != "" {
				content.WriteString(fmt.Sprintf("**Default**: `%s`\n\n", input.Default))
			}
		}
	}

	// Outputs section
	if len(metadata.Outputs) > 0 {
		content.WriteString("## Outputs\n\n")
		for _, output := range metadata.Outputs {
			content.WriteString(fmt.Sprintf("### `%s`\n\n", output.Name))
			content.WriteString(fmt.Sprintf("**Description**: %s\n\n", output.Description))
		}
	}

	// Dependencies section
	if len(metadata.Dependencies) > 0 {
		content.WriteString("## Dependencies\n\n")
		content.WriteString("This action depends on the following JavaScript modules:\n\n")
		for _, dep := range metadata.Dependencies {
			content.WriteString(fmt.Sprintf("- `%s`\n", dep))
		}
		content.WriteString("\n")
	}

	// Development section
	content.WriteString("## Development\n\n")
	content.WriteString("### Building\n\n")
	content.WriteString("To build this action, you need to:\n\n")
	content.WriteString(fmt.Sprintf("1. Update the dependency mapping in `pkg/cli/actions_build_command.go` for `%s`\n", metadata.ActionName))
	content.WriteString("2. Run `make actions-build` to bundle the JavaScript dependencies\n")
	content.WriteString("3. The bundled `index.js` will be generated and committed\n\n")

	content.WriteString("### Testing\n\n")
	content.WriteString("Test this action by creating a workflow:\n\n")
	content.WriteString("```yaml\n")
	content.WriteString("jobs:\n")
	content.WriteString("  test:\n")
	content.WriteString("    runs-on: ubuntu-latest\n")
	content.WriteString("    steps:\n")
	content.WriteString(fmt.Sprintf("      - uses: ./actions/%s\n", metadata.ActionName))
	content.WriteString("```\n\n")

	// License
	content.WriteString("## License\n\n")
	content.WriteString("MIT\n")

	// Write to file
	readmePath := filepath.Join(actionDir, "README.md")
	if err := os.WriteFile(readmePath, []byte(content.String()), 0644); err != nil {
		return fmt.Errorf("failed to write README.md: %w", err)
	}

	return nil
}

// transformSourceForBundling transforms JavaScript source to use FILES pattern for bundling
// This converts require('./file.cjs') statements to use embedded FILES object
func transformSourceForBundling(content string, dependencies []string) string {
	var transformed strings.Builder

	// Add FILES placeholder at the top (will be populated by actions-build)
	transformed.WriteString("// Embedded files for bundling\n")
	transformed.WriteString("const FILES = {\n")
	transformed.WriteString("  // This will be populated by the build script\n")
	transformed.WriteString("};\n\n")

	// Add helper function to load files from FILES object
	transformed.WriteString("// Helper to load embedded files\n")
	transformed.WriteString("function requireFile(filename) {\n")
	transformed.WriteString("  const content = FILES[filename];\n")
	transformed.WriteString("  if (!content) {\n")
	transformed.WriteString("    throw new Error(`File not found: ${filename}`);\n")
	transformed.WriteString("  }\n")
	transformed.WriteString("  const exports = {};\n")
	transformed.WriteString("  const module = { exports };\n")
	transformed.WriteString("  const func = new Function('exports', 'module', 'require', content);\n")
	transformed.WriteString("  func(exports, module, requireFile);\n")
	transformed.WriteString("  return module.exports;\n")
	transformed.WriteString("}\n\n")

	// Transform require() statements to use requireFile()
	requireRegex := regexp.MustCompile(`require\(['"]\./([^'"]+\.cjs)['"]\)`)
	transformedContent := requireRegex.ReplaceAllString(content, `requireFile('$1')`)

	// Add the transformed original content
	transformed.WriteString(transformedContent)

	return transformed.String()
}
