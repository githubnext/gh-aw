package parser

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/githubnext/gh-aw/pkg/console"
	"github.com/githubnext/gh-aw/pkg/logger"
	"github.com/goccy/go-yaml"
)

var log = logger.New("parser:frontmatter")

// IncludeDirectivePattern matches @include, @import (deprecated), or {{#import (new) directives
// The colon after #import is optional and ignored if present
var IncludeDirectivePattern = regexp.MustCompile(`^(?:@(?:include|import)(\?)?\s+(.+)|{{#import(\?)?\s*:?\s*(.+?)\s*}})$`)

// LegacyIncludeDirectivePattern matches only the deprecated @include and @import directives
var LegacyIncludeDirectivePattern = regexp.MustCompile(`^@(?:include|import)(\?)?\s+(.+)$`)

// ImportDirectiveMatch holds the parsed components of an import directive
type ImportDirectiveMatch struct {
	IsOptional bool
	Path       string
	IsLegacy   bool
	Original   string
}

// ParseImportDirective parses an import directive and returns its components
func ParseImportDirective(line string) *ImportDirectiveMatch {
	trimmedLine := strings.TrimSpace(line)

	// Check if it matches the import pattern at all
	matches := IncludeDirectivePattern.FindStringSubmatch(trimmedLine)
	if matches == nil {
		return nil
	}

	// Check if it's legacy syntax
	isLegacy := LegacyIncludeDirectivePattern.MatchString(trimmedLine)

	var isOptional bool
	var path string

	if isLegacy {
		// Legacy syntax: @include? path or @import? path
		// Group 1: optional marker, Group 2: path
		isOptional = matches[1] == "?"
		path = strings.TrimSpace(matches[2])
	} else {
		// New syntax: {{#import?: path}} or {{#import: path}} (colon is optional)
		// Group 3: optional marker, Group 4: path
		isOptional = matches[3] == "?"
		path = strings.TrimSpace(matches[4])
	}

	return &ImportDirectiveMatch{
		IsOptional: isOptional,
		Path:       path,
		IsLegacy:   isLegacy,
		Original:   trimmedLine,
	}
}

// isMCPType checks if a type string represents an MCP-compatible type
func isMCPType(typeStr string) bool {
	switch typeStr {
	case "stdio", "http":
		return true
	default:
		return false
	}
}

// ImportsResult holds the result of processing imports from frontmatter
type ImportsResult struct {
	MergedTools         string         // Merged tools configuration from all imports
	MergedMCPServers    string         // Merged mcp-servers configuration from all imports
	MergedEngines       []string       // Merged engine configurations from all imports
	MergedSafeOutputs   []string       // Merged safe-outputs configurations from all imports
	MergedMarkdown      string         // Merged markdown content from all imports
	MergedSteps         string         // Merged steps configuration from all imports
	MergedRuntimes      string         // Merged runtimes configuration from all imports
	MergedServices      string         // Merged services configuration from all imports
	MergedNetwork       string         // Merged network configuration from all imports
	MergedPermissions   string         // Merged permissions configuration from all imports
	MergedSecretMasking string         // Merged secret-masking steps from all imports
	ImportedFiles       []string       // List of imported file paths (for manifest)
	AgentFile           string         // Path to custom agent file (if imported)
	ImportInputs        map[string]any // Aggregated input values from all imports (key = input name, value = input value)
}

// ImportInputDefinition defines an input parameter for a shared workflow import.
// Uses the same schema as workflow_dispatch inputs.
type ImportInputDefinition struct {
	Description string   `yaml:"description,omitempty" json:"description,omitempty"`
	Required    bool     `yaml:"required,omitempty" json:"required,omitempty"`
	Default     any      `yaml:"default,omitempty" json:"default,omitempty"` // Can be string, number, or boolean
	Type        string   `yaml:"type,omitempty" json:"type,omitempty"`       // "string", "choice", "boolean", "number"
	Options     []string `yaml:"options,omitempty" json:"options,omitempty"` // Options for choice type
}

// ImportSpec represents a single import specification (either a string path or an object with path and inputs)
type ImportSpec struct {
	Path   string         // Import path (required)
	Inputs map[string]any // Optional input values to pass to the imported workflow (values are string, number, or boolean)
}

// ProcessImportsFromFrontmatter processes imports field from frontmatter
// Returns merged tools and engines from imported files
func ProcessImportsFromFrontmatter(frontmatter map[string]any, baseDir string) (mergedTools string, mergedEngines []string, err error) {
	result, err := ProcessImportsFromFrontmatterWithManifest(frontmatter, baseDir, nil)
	if err != nil {
		return "", nil, err
	}
	return result.MergedTools, result.MergedEngines, nil
}

// importQueueItem represents a file to be imported with its context
type importQueueItem struct {
	importPath  string         // Original import path (e.g., "file.md" or "file.md#Section")
	fullPath    string         // Resolved absolute file path
	sectionName string         // Optional section name (from file.md#Section syntax)
	baseDir     string         // Base directory for resolving nested imports
	inputs      map[string]any // Optional input values from parent import
}

// ProcessImportsFromFrontmatterWithManifest processes imports field from frontmatter
// Returns result containing merged tools, engines, markdown content, and list of imported files
// Uses BFS traversal with queues for deterministic ordering and cycle detection
func ProcessImportsFromFrontmatterWithManifest(frontmatter map[string]any, baseDir string, cache *ImportCache) (*ImportsResult, error) {
	// Check if imports field exists
	importsField, exists := frontmatter["imports"]
	if !exists {
		return &ImportsResult{}, nil
	}

	log.Print("Processing imports from frontmatter with recursive BFS")

	// Parse imports field - can be array of strings or objects with path and inputs
	var importSpecs []ImportSpec
	switch v := importsField.(type) {
	case []any:
		for _, item := range v {
			switch importItem := item.(type) {
			case string:
				// Simple string import
				importSpecs = append(importSpecs, ImportSpec{Path: importItem})
			case map[string]any:
				// Object import with path and optional inputs
				pathValue, hasPath := importItem["path"]
				if !hasPath {
					return nil, fmt.Errorf("import object must have a 'path' field")
				}
				pathStr, ok := pathValue.(string)
				if !ok {
					return nil, fmt.Errorf("import 'path' must be a string")
				}
				var inputs map[string]any
				if inputsValue, hasInputs := importItem["inputs"]; hasInputs {
					if inputsMap, ok := inputsValue.(map[string]any); ok {
						inputs = inputsMap
					} else {
						return nil, fmt.Errorf("import 'inputs' must be an object")
					}
				}
				importSpecs = append(importSpecs, ImportSpec{Path: pathStr, Inputs: inputs})
			default:
				return nil, fmt.Errorf("import item must be a string or an object with 'path' field")
			}
		}
	case []string:
		for _, s := range v {
			importSpecs = append(importSpecs, ImportSpec{Path: s})
		}
	default:
		return nil, fmt.Errorf("imports field must be an array of strings or objects")
	}

	if len(importSpecs) == 0 {
		return &ImportsResult{}, nil
	}

	log.Printf("Found %d direct imports to process", len(importSpecs))

	// Initialize BFS queue and visited set for cycle detection
	var queue []importQueueItem
	visited := make(map[string]bool)
	processedOrder := []string{} // Track processing order for manifest

	// Initialize result accumulators
	var toolsBuilder strings.Builder
	var mcpServersBuilder strings.Builder
	var markdownBuilder strings.Builder
	var stepsBuilder strings.Builder
	var runtimesBuilder strings.Builder
	var servicesBuilder strings.Builder
	var networkBuilder strings.Builder
	var permissionsBuilder strings.Builder
	var secretMaskingBuilder strings.Builder
	var engines []string
	var safeOutputs []string
	var agentFile string                 // Track custom agent file
	importInputs := make(map[string]any) // Aggregated input values from all imports

	// Seed the queue with initial imports
	for _, importSpec := range importSpecs {
		importPath := importSpec.Path
		// Handle section references (file.md#Section)
		var filePath, sectionName string
		if strings.Contains(importPath, "#") {
			parts := strings.SplitN(importPath, "#", 2)
			filePath = parts[0]
			sectionName = parts[1]
		} else {
			filePath = importPath
		}

		// Resolve import path (supports workflowspec format)
		fullPath, err := resolveIncludePath(filePath, baseDir, cache)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve import '%s': %w", filePath, err)
		}

		// Check for duplicates before adding to queue
		if !visited[fullPath] {
			visited[fullPath] = true
			queue = append(queue, importQueueItem{
				importPath:  importPath,
				fullPath:    fullPath,
				sectionName: sectionName,
				baseDir:     baseDir,
				inputs:      importSpec.Inputs,
			})
			log.Printf("Queued import: %s (resolved to %s)", importPath, fullPath)
		}
	}

	// BFS traversal: process queue until empty
	for len(queue) > 0 {
		// Dequeue first item (FIFO for BFS)
		item := queue[0]
		queue = queue[1:]

		log.Printf("Processing import from queue: %s", item.fullPath)

		// Merge inputs from this import into the aggregated inputs map
		for k, v := range item.inputs {
			importInputs[k] = v
		}

		// Add to processing order
		processedOrder = append(processedOrder, item.importPath)

		// Check if this is a custom agent file (any markdown file under .github/agents)
		isAgentFile := strings.Contains(item.fullPath, "/.github/agents/") && strings.HasSuffix(strings.ToLower(item.fullPath), ".md")
		if isAgentFile {
			if agentFile != "" {
				// Multiple agent files found - error
				return nil, fmt.Errorf("multiple agent files found in imports: '%s' and '%s'. Only one agent file is allowed per workflow", agentFile, item.importPath)
			}
			// Extract relative path from repository root (from .github/ onwards)
			// This ensures the path works at runtime with $GITHUB_WORKSPACE
			if idx := strings.Index(item.fullPath, "/.github/"); idx >= 0 {
				agentFile = item.fullPath[idx+1:] // +1 to skip the leading slash
			} else {
				agentFile = item.fullPath
			}
			log.Printf("Found agent file: %s (resolved to: %s)", item.fullPath, agentFile)

			// For agent files, only extract markdown content
			markdownContent, err := processIncludedFileWithVisited(item.fullPath, item.sectionName, false, visited)
			if err != nil {
				return nil, fmt.Errorf("failed to process markdown from agent file '%s': %w", item.fullPath, err)
			}
			if markdownContent != "" {
				markdownBuilder.WriteString(markdownContent)
				// Add blank line separator between imported files
				if !strings.HasSuffix(markdownContent, "\n\n") {
					if strings.HasSuffix(markdownContent, "\n") {
						markdownBuilder.WriteString("\n")
					} else {
						markdownBuilder.WriteString("\n\n")
					}
				}
			}

			// Agent files don't have nested imports, skip to next item
			continue
		}

		// Read the imported file to extract nested imports
		content, err := os.ReadFile(item.fullPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read imported file '%s': %w", item.fullPath, err)
		}

		// Extract frontmatter from imported file to discover nested imports
		result, err := ExtractFrontmatterFromContent(string(content))
		if err != nil {
			// If frontmatter extraction fails, continue with other processing
			log.Printf("Failed to extract frontmatter from %s: %v", item.fullPath, err)
		} else if result.Frontmatter != nil {
			// Check for nested imports field
			if nestedImportsField, hasImports := result.Frontmatter["imports"]; hasImports {
				var nestedImports []string
				switch v := nestedImportsField.(type) {
				case []any:
					for _, nestedItem := range v {
						if str, ok := nestedItem.(string); ok {
							nestedImports = append(nestedImports, str)
						}
					}
				case []string:
					nestedImports = v
				}

				// Add nested imports to queue (BFS: append to end)
				// Use the original baseDir for resolving nested imports, not the nested file's directory
				// This ensures that all imports are resolved relative to the workflows directory
				for _, nestedImportPath := range nestedImports {
					// Handle section references
					var nestedFilePath, nestedSectionName string
					if strings.Contains(nestedImportPath, "#") {
						parts := strings.SplitN(nestedImportPath, "#", 2)
						nestedFilePath = parts[0]
						nestedSectionName = parts[1]
					} else {
						nestedFilePath = nestedImportPath
					}

					// Resolve nested import path relative to the workflows directory, not the nested file's directory
					nestedFullPath, err := resolveIncludePath(nestedFilePath, baseDir, cache)
					if err != nil {
						return nil, fmt.Errorf("failed to resolve nested import '%s' from '%s': %w", nestedFilePath, item.fullPath, err)
					}

					// Check for cycles - skip if already visited
					if !visited[nestedFullPath] {
						visited[nestedFullPath] = true
						queue = append(queue, importQueueItem{
							importPath:  nestedImportPath,
							fullPath:    nestedFullPath,
							sectionName: nestedSectionName,
							baseDir:     baseDir, // Use original baseDir, not nestedBaseDir
						})
						log.Printf("Discovered nested import: %s -> %s (queued)", item.fullPath, nestedFullPath)
					} else {
						log.Printf("Skipping already visited nested import: %s (cycle detected)", nestedFullPath)
					}
				}
			}
		}

		// Extract tools from imported file
		toolsContent, err := processIncludedFileWithVisited(item.fullPath, item.sectionName, true, visited)
		if err != nil {
			return nil, fmt.Errorf("failed to process imported file '%s': %w", item.fullPath, err)
		}
		toolsBuilder.WriteString(toolsContent + "\n")

		// Extract markdown content from imported file
		markdownContent, err := processIncludedFileWithVisited(item.fullPath, item.sectionName, false, visited)
		if err != nil {
			return nil, fmt.Errorf("failed to process markdown from imported file '%s': %w", item.fullPath, err)
		}
		if markdownContent != "" {
			markdownBuilder.WriteString(markdownContent)
			// Add blank line separator between imported files
			if !strings.HasSuffix(markdownContent, "\n\n") {
				if strings.HasSuffix(markdownContent, "\n") {
					markdownBuilder.WriteString("\n")
				} else {
					markdownBuilder.WriteString("\n\n")
				}
			}
		}

		// Extract engines from imported file
		engineContent, err := extractEngineFromContent(string(content))
		if err == nil && engineContent != "" {
			engines = append(engines, engineContent)
		}

		// Extract mcp-servers from imported file
		mcpServersContent, err := extractMCPServersFromContent(string(content))
		if err == nil && mcpServersContent != "" && mcpServersContent != "{}" {
			mcpServersBuilder.WriteString(mcpServersContent + "\n")
		}

		// Extract safe-outputs from imported file
		safeOutputsContent, err := extractSafeOutputsFromContent(string(content))
		if err == nil && safeOutputsContent != "" && safeOutputsContent != "{}" {
			safeOutputs = append(safeOutputs, safeOutputsContent)
		}

		// Extract steps from imported file
		stepsContent, err := extractStepsFromContent(string(content))
		if err == nil && stepsContent != "" {
			stepsBuilder.WriteString(stepsContent + "\n")
		}

		// Extract runtimes from imported file
		runtimesContent, err := extractRuntimesFromContent(string(content))
		if err == nil && runtimesContent != "" && runtimesContent != "{}" {
			runtimesBuilder.WriteString(runtimesContent + "\n")
		}

		// Extract services from imported file
		servicesContent, err := extractServicesFromContent(string(content))
		if err == nil && servicesContent != "" {
			servicesBuilder.WriteString(servicesContent + "\n")
		}

		// Extract network from imported file
		networkContent, err := extractNetworkFromContent(string(content))
		if err == nil && networkContent != "" && networkContent != "{}" {
			networkBuilder.WriteString(networkContent + "\n")
		}

		// Extract permissions from imported file
		permissionsContent, err := ExtractPermissionsFromContent(string(content))
		if err == nil && permissionsContent != "" && permissionsContent != "{}" {
			permissionsBuilder.WriteString(permissionsContent + "\n")
		}

		// Extract secret-masking from imported file
		secretMaskingContent, err := extractSecretMaskingFromContent(string(content))
		if err == nil && secretMaskingContent != "" && secretMaskingContent != "{}" {
			secretMaskingBuilder.WriteString(secretMaskingContent + "\n")
		}
	}

	log.Printf("Completed BFS traversal. Processed %d imports in total", len(processedOrder))

	return &ImportsResult{
		MergedTools:         toolsBuilder.String(),
		MergedMCPServers:    mcpServersBuilder.String(),
		MergedEngines:       engines,
		MergedSafeOutputs:   safeOutputs,
		MergedMarkdown:      markdownBuilder.String(),
		MergedSteps:         stepsBuilder.String(),
		MergedRuntimes:      runtimesBuilder.String(),
		MergedServices:      servicesBuilder.String(),
		MergedNetwork:       networkBuilder.String(),
		MergedPermissions:   permissionsBuilder.String(),
		MergedSecretMasking: secretMaskingBuilder.String(),
		ImportedFiles:       processedOrder,
		AgentFile:           agentFile,
		ImportInputs:        importInputs,
	}, nil
}

// ProcessIncludes processes @include, @import (deprecated), and {{#import: directives in markdown content
// This matches the bash process_includes function behavior
func ProcessIncludes(content, baseDir string, extractTools bool) (string, error) {
	visited := make(map[string]bool)
	return processIncludesWithVisited(content, baseDir, extractTools, visited)
}

// processIncludesWithVisited processes import directives with cycle detection
func processIncludesWithVisited(content, baseDir string, extractTools bool, visited map[string]bool) (string, error) {
	scanner := bufio.NewScanner(strings.NewReader(content))
	var result bytes.Buffer

	for scanner.Scan() {
		line := scanner.Text()

		// Parse import directive
		directive := ParseImportDirective(line)
		if directive != nil {
			// Emit deprecation warning for legacy syntax
			if directive.IsLegacy {
				// Security: Escape strings to prevent quote injection in warning messages
				// Use %q format specifier to safely quote strings containing special characters
				optionalMarker := ""
				if directive.IsOptional {
					optionalMarker = "?"
				}
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Deprecated syntax: %q. Use {{#import%s %s}} instead.",
					directive.Original,
					optionalMarker,
					directive.Path)))
			}

			isOptional := directive.IsOptional
			includePath := directive.Path

			// Handle section references (file.md#Section)
			var filePath, sectionName string
			if strings.Contains(includePath, "#") {
				parts := strings.SplitN(includePath, "#", 2)
				filePath = parts[0]
				sectionName = parts[1]
			} else {
				filePath = includePath
			}

			// Resolve file path first to get the canonical path
			fullPath, err := resolveIncludePath(filePath, baseDir, nil)
			if err != nil {
				if isOptional {
					// For optional includes, show a friendly informational message to stdout
					if !extractTools {
						fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Optional include file not found: %s. You can create this file to configure the workflow.", filePath)))
					}
					continue
				}
				// For required includes, fail compilation with an error
				return "", fmt.Errorf("failed to resolve required include '%s': %w", filePath, err)
			}

			// Check for repeated imports using the resolved full path
			if visited[fullPath] {
				if !extractTools {
					fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Already included: %s, skipping", filePath)))
				}
				continue
			}

			// Mark as visited using the resolved full path
			visited[fullPath] = true

			// Process the included file
			includedContent, err := processIncludedFileWithVisited(fullPath, sectionName, extractTools, visited)
			if err != nil {
				// For any processing errors, fail compilation
				return "", fmt.Errorf("failed to process included file '%s': %w", fullPath, err)
			}

			if extractTools {
				// For tools mode, add each JSON on a separate line
				result.WriteString(includedContent + "\n")
			} else {
				result.WriteString(includedContent)
			}
		} else {
			// Regular line, just pass through (unless extracting tools)
			if !extractTools {
				result.WriteString(line + "\n")
			}
		}
	}

	return result.String(), nil
}

// processIncludedFile processes a single included file, optionally extracting a section
// processIncludedFileWithVisited processes a single included file with cycle detection for nested includes
func processIncludedFileWithVisited(filePath, sectionName string, extractTools bool, visited map[string]bool) (string, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to read included file %s: %w", filePath, err)
	}

	// Validate included file frontmatter based on file location
	result, err := ExtractFrontmatterFromContent(string(content))
	if err != nil {
		return "", fmt.Errorf("failed to extract frontmatter from included file %s: %w", filePath, err)
	}

	// Check if file is under .github/workflows/ for strict validation
	isWorkflowFile := isUnderWorkflowsDirectory(filePath)

	// Always try strict validation first
	validationErr := ValidateIncludedFileFrontmatterWithSchemaAndLocation(result.Frontmatter, filePath)

	if validationErr != nil {
		if isWorkflowFile {
			// For workflow files, strict validation must pass
			return "", fmt.Errorf("invalid frontmatter in included file %s: %w", filePath, validationErr)
		} else {
			// For non-workflow files, fall back to relaxed validation with warnings
			if len(result.Frontmatter) > 0 {
				// Check for unexpected frontmatter fields (anything other than tools, engine, network, mcp-servers, and imports)
				unexpectedFields := make([]string, 0)
				for key := range result.Frontmatter {
					if key != "tools" && key != "engine" && key != "network" && key != "mcp-servers" && key != "imports" {
						unexpectedFields = append(unexpectedFields, key)
					}
				}

				if len(unexpectedFields) > 0 {
					// Show warning for unexpected frontmatter fields
					fmt.Fprintf(os.Stderr, "%s\n", console.FormatWarningMessage(
						fmt.Sprintf("Ignoring unexpected frontmatter fields in %s: %s",
							filePath, strings.Join(unexpectedFields, ", "))))
				}

				// Validate the tools, engine, network, and mcp-servers sections if present
				filteredFrontmatter := map[string]any{}
				if tools, hasTools := result.Frontmatter["tools"]; hasTools {
					filteredFrontmatter["tools"] = tools
				}
				if engine, hasEngine := result.Frontmatter["engine"]; hasEngine {
					filteredFrontmatter["engine"] = engine
				}
				if network, hasNetwork := result.Frontmatter["network"]; hasNetwork {
					filteredFrontmatter["network"] = network
				}
				if mcpServers, hasMCPServers := result.Frontmatter["mcp-servers"]; hasMCPServers {
					filteredFrontmatter["mcp-servers"] = mcpServers
				}
				// Note: we don't validate imports field as it's handled separately
				if len(filteredFrontmatter) > 0 {
					if err := ValidateIncludedFileFrontmatterWithSchemaAndLocation(filteredFrontmatter, filePath); err != nil {
						fmt.Fprintf(os.Stderr, "%s\n", console.FormatWarningMessage(
							fmt.Sprintf("Invalid configuration in %s: %v", filePath, err)))
					}
				}
			}
		}
	}

	if extractTools {
		// Extract tools from frontmatter, using filtered frontmatter for non-workflow files with validation errors
		if validationErr == nil || isWorkflowFile {
			// If validation passed or it's a workflow file (which must have valid frontmatter), use original extraction
			return extractToolsFromContent(string(content))
		} else {
			// For non-workflow files with validation errors, only extract tools section
			if tools, hasTools := result.Frontmatter["tools"]; hasTools {
				toolsJSON, err := json.Marshal(tools)
				if err != nil {
					return "{}", nil
				}
				return strings.TrimSpace(string(toolsJSON)), nil
			}
			return "{}", nil
		}
	}

	// Extract markdown content
	markdownContent, err := ExtractMarkdownContent(string(content))
	if err != nil {
		return "", fmt.Errorf("failed to extract markdown from %s: %w", filePath, err)
	}

	// Process nested includes recursively
	includedDir := filepath.Dir(filePath)
	markdownContent, err = processIncludesWithVisited(markdownContent, includedDir, extractTools, visited)
	if err != nil {
		return "", fmt.Errorf("failed to process nested includes in %s: %w", filePath, err)
	}

	// If section specified, extract only that section
	if sectionName != "" {
		sectionContent, err := ExtractMarkdownSection(markdownContent, sectionName)
		if err != nil {
			return "", fmt.Errorf("failed to extract section '%s' from %s: %w", sectionName, filePath, err)
		}
		return strings.Trim(sectionContent, "\n") + "\n", nil
	}

	return strings.Trim(markdownContent, "\n") + "\n", nil
}

// extractToolsFromContent extracts tools and mcp-servers sections from frontmatter as merged JSON string
func extractToolsFromContent(content string) (string, error) {
	result, err := ExtractFrontmatterFromContent(content)
	if err != nil {
		return "{}", nil // Return empty object on error to match bash behavior
	}

	// Create a map to hold the merged result
	extracted := make(map[string]any)

	// Helper function to merge a field into extracted map
	mergeField := func(fieldName string) {
		if fieldValue, exists := result.Frontmatter[fieldName]; exists {
			if fieldMap, ok := fieldValue.(map[string]any); ok {
				for key, value := range fieldMap {
					extracted[key] = value
				}
			}
		}
	}

	// Extract and merge tools section (tools are stored as tool_name: tool_config)
	mergeField("tools")

	// Extract and merge mcp-servers section (mcp-servers are stored as server_name: server_config)
	mergeField("mcp-servers")

	// If nothing was extracted, return empty object
	if len(extracted) == 0 {
		return "{}", nil
	}

	// Convert to JSON string
	extractedJSON, err := json.Marshal(extracted)
	if err != nil {
		return "{}", nil
	}

	return strings.TrimSpace(string(extractedJSON)), nil
}

// extractSafeOutputsFromContent extracts safe-outputs section from frontmatter as JSON string
func extractSafeOutputsFromContent(content string) (string, error) {
	return extractFrontmatterField(content, "safe-outputs", "{}")
}

// extractMCPServersFromContent extracts mcp-servers section from frontmatter as JSON string
func extractMCPServersFromContent(content string) (string, error) {
	return extractFrontmatterField(content, "mcp-servers", "{}")
}

// extractStepsFromContent extracts steps section from frontmatter as YAML string
func extractStepsFromContent(content string) (string, error) {
	result, err := ExtractFrontmatterFromContent(content)
	if err != nil {
		return "", nil // Return empty string on error
	}

	// Extract steps section
	steps, exists := result.Frontmatter["steps"]
	if !exists {
		return "", nil
	}

	// Convert to YAML string (similar to how CustomSteps are handled in compiler)
	stepsYAML, err := yaml.Marshal(steps)
	if err != nil {
		return "", nil
	}

	return strings.TrimSpace(string(stepsYAML)), nil
}

// extractEngineFromContent extracts engine section from frontmatter as JSON string
func extractEngineFromContent(content string) (string, error) {
	return extractFrontmatterField(content, "engine", "")
}

// extractRuntimesFromContent extracts runtimes section from frontmatter as JSON string
func extractRuntimesFromContent(content string) (string, error) {
	return extractFrontmatterField(content, "runtimes", "{}")
}

// extractServicesFromContent extracts services section from frontmatter as YAML string
func extractServicesFromContent(content string) (string, error) {
	result, err := ExtractFrontmatterFromContent(content)
	if err != nil {
		return "", nil // Return empty string on error
	}

	// Extract services section
	services, exists := result.Frontmatter["services"]
	if !exists {
		return "", nil
	}

	// Convert to YAML string (similar to how steps are handled)
	servicesYAML, err := yaml.Marshal(services)
	if err != nil {
		return "", nil
	}

	return strings.TrimSpace(string(servicesYAML)), nil
}

// extractNetworkFromContent extracts network section from frontmatter as JSON string
func extractNetworkFromContent(content string) (string, error) {
	return extractFrontmatterField(content, "network", "{}")
}

// ExtractPermissionsFromContent extracts permissions section from frontmatter as JSON string
func ExtractPermissionsFromContent(content string) (string, error) {
	return extractFrontmatterField(content, "permissions", "{}")
}

// extractSecretMaskingFromContent extracts secret-masking section from frontmatter as JSON string
func extractSecretMaskingFromContent(content string) (string, error) {
	return extractFrontmatterField(content, "secret-masking", "{}")
}

// extractFrontmatterField extracts a specific field from frontmatter as JSON string
func extractFrontmatterField(content, fieldName, emptyValue string) (string, error) {
	result, err := ExtractFrontmatterFromContent(content)
	if err != nil {
		return emptyValue, nil // Return empty value on error
	}

	// Extract the requested field
	fieldValue, exists := result.Frontmatter[fieldName]
	if !exists {
		return emptyValue, nil
	}

	// Convert to JSON string
	fieldJSON, err := json.Marshal(fieldValue)
	if err != nil {
		return emptyValue, nil
	}

	return strings.TrimSpace(string(fieldJSON)), nil
}

// ExpandIncludes recursively expands @include and @import directives until no more remain
// This matches the bash expand_includes function behavior
func ExpandIncludes(content, baseDir string, extractTools bool) (string, error) {
	expandedContent, _, err := ExpandIncludesWithManifest(content, baseDir, extractTools)
	return expandedContent, err
}

// ExpandIncludesWithManifest recursively expands @include and @import directives and returns list of included files
func ExpandIncludesWithManifest(content, baseDir string, extractTools bool) (string, []string, error) {
	const maxDepth = 10
	currentContent := content
	visited := make(map[string]bool)

	for depth := 0; depth < maxDepth; depth++ {
		// Process includes in current content
		processedContent, err := processIncludesWithVisited(currentContent, baseDir, extractTools, visited)
		if err != nil {
			return "", nil, err
		}

		// For tools mode, check if we still have @include or @import directives
		if extractTools {
			if !strings.Contains(processedContent, "@include") && !strings.Contains(processedContent, "@import") {
				// No more includes to process for tools mode
				currentContent = processedContent
				break
			}
		} else {
			// For content mode, check if content changed
			if processedContent == currentContent {
				// No more includes to process
				break
			}
		}

		currentContent = processedContent
	}

	// Convert visited map to slice of file paths (make them relative to baseDir if possible)
	var includedFiles []string
	for filePath := range visited {
		// Try to make path relative to baseDir for cleaner output
		relPath, err := filepath.Rel(baseDir, filePath)
		if err == nil && !strings.HasPrefix(relPath, "..") {
			includedFiles = append(includedFiles, relPath)
		} else {
			includedFiles = append(includedFiles, filePath)
		}
	}

	if extractTools {
		// For tools mode, merge all extracted JSON objects
		mergedTools, err := mergeToolsFromJSON(currentContent)
		return mergedTools, includedFiles, err
	}

	return currentContent, includedFiles, nil
}

// ExpandIncludesForEngines recursively expands @include and @import directives to extract engine configurations
func ExpandIncludesForEngines(content, baseDir string) ([]string, error) {
	return expandIncludesForField(content, baseDir, extractEngineFromContent, "")
}

// ExpandIncludesForSafeOutputs recursively expands @include and @import directives to extract safe-outputs configurations
func ExpandIncludesForSafeOutputs(content, baseDir string) ([]string, error) {
	return expandIncludesForField(content, baseDir, extractSafeOutputsFromContent, "{}")
}

// expandIncludesForField recursively expands includes to extract a specific frontmatter field
func expandIncludesForField(content, baseDir string, extractFunc func(string) (string, error), emptyValue string) ([]string, error) {
	const maxDepth = 10
	var results []string
	currentContent := content

	for depth := 0; depth < maxDepth; depth++ {
		// Process includes in current content to extract the field
		processedResults, processedContent, err := processIncludesForField(currentContent, baseDir, extractFunc, emptyValue)
		if err != nil {
			return nil, err
		}

		// Add found results to the list
		results = append(results, processedResults...)

		// Check if content changed
		if processedContent == currentContent {
			// No more includes to process
			break
		}

		currentContent = processedContent
	}

	return results, nil
}

// ProcessIncludesForEngines processes import directives to extract engine configurations
func ProcessIncludesForEngines(content, baseDir string) ([]string, string, error) {
	return processIncludesForField(content, baseDir, extractEngineFromContent, "")
}

// ProcessIncludesForSafeOutputs processes import directives to extract safe-outputs configurations
func ProcessIncludesForSafeOutputs(content, baseDir string) ([]string, string, error) {
	return processIncludesForField(content, baseDir, extractSafeOutputsFromContent, "{}")
}

// processIncludesForField processes import directives to extract a specific frontmatter field
func processIncludesForField(content, baseDir string, extractFunc func(string) (string, error), emptyValue string) ([]string, string, error) {
	scanner := bufio.NewScanner(strings.NewReader(content))
	var result bytes.Buffer
	var results []string

	for scanner.Scan() {
		line := scanner.Text()

		// Parse import directive
		directive := ParseImportDirective(line)
		if directive != nil {
			isOptional := directive.IsOptional
			includePath := directive.Path

			// Handle section references (file.md#Section) - for frontmatter fields, we ignore sections
			var filePath string
			if strings.Contains(includePath, "#") {
				parts := strings.SplitN(includePath, "#", 2)
				filePath = parts[0]
				// Note: section references are ignored for frontmatter field extraction
			} else {
				filePath = includePath
			}

			// Resolve file path
			fullPath, err := resolveIncludePath(filePath, baseDir, nil)
			if err != nil {
				if isOptional {
					// For optional includes, skip extraction
					continue
				}
				// For required includes, fail compilation with an error
				return nil, "", fmt.Errorf("failed to resolve required include '%s': %w", filePath, err)
			}

			// Read the included file
			fileContent, err := os.ReadFile(fullPath)
			if err != nil {
				// For any processing errors, fail compilation
				return nil, "", fmt.Errorf("failed to read included file '%s': %w", fullPath, err)
			}

			// Extract the field using the provided extraction function
			fieldJSON, err := extractFunc(string(fileContent))
			if err != nil {
				return nil, "", fmt.Errorf("failed to extract field from '%s': %w", fullPath, err)
			}

			if fieldJSON != "" && fieldJSON != emptyValue {
				results = append(results, fieldJSON)
			}
		} else {
			// Regular line, just pass through
			result.WriteString(line + "\n")
		}
	}

	return results, result.String(), nil
}

// mergeToolsFromJSON merges multiple JSON tool objects from content
func mergeToolsFromJSON(content string) (string, error) {
	// Clean up the content first
	content = strings.TrimSpace(content)

	// Try to parse as a single JSON object first
	var singleObj map[string]any
	if err := json.Unmarshal([]byte(content), &singleObj); err == nil {
		if len(singleObj) > 0 {
			result, err := json.Marshal(singleObj)
			if err != nil {
				return "{}", err
			}
			return string(result), nil
		}
	}

	// Find all JSON objects in the content (line by line)
	var jsonObjects []map[string]any

	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || line == "{}" {
			continue
		}

		var toolsObj map[string]any
		if err := json.Unmarshal([]byte(line), &toolsObj); err == nil {
			if len(toolsObj) > 0 { // Only add non-empty objects
				jsonObjects = append(jsonObjects, toolsObj)
			}
		}
	}

	// If no valid objects found, return empty
	if len(jsonObjects) == 0 {
		return "{}", nil
	}

	// Merge all objects
	merged := make(map[string]any)
	for _, obj := range jsonObjects {
		var err error
		merged, err = MergeTools(merged, obj)
		if err != nil {
			return "{}", err
		}
	}

	// Convert back to JSON
	result, err := json.Marshal(merged)
	if err != nil {
		return "{}", err
	}

	return string(result), nil
}

// MergeTools merges two neutral tool configurations.
// Only supports merging arrays and maps for neutral tools (bash, web-fetch, web-search, edit, mcp-*).
// Removes all legacy Claude tool merging logic.
func MergeTools(base, additional map[string]any) (map[string]any, error) {
	result := make(map[string]any)

	// Copy base
	for k, v := range base {
		result[k] = v
	}

	// Merge additional
	for key, newValue := range additional {
		if existingValue, exists := result[key]; exists {
			// Both have the same key, merge them

			// If both are arrays, merge and deduplicate
			_, existingIsArray := existingValue.([]any)
			_, newIsArray := newValue.([]any)
			if existingIsArray && newIsArray {
				merged := mergeAllowedArrays(existingValue, newValue)
				result[key] = merged
				continue
			}

			// If both are maps, check for special merging cases
			existingMap, existingIsMap := existingValue.(map[string]any)
			newMap, newIsMap := newValue.(map[string]any)
			if existingIsMap && newIsMap {
				// Check if this is an MCP tool (has MCP-compatible type)
				var existingType, newType string
				if existingMcp, hasMcp := existingMap["mcp"]; hasMcp {
					if mcpMap, ok := existingMcp.(map[string]any); ok {
						existingType, _ = mcpMap["type"].(string)
					}
				}
				if newMcp, hasMcp := newMap["mcp"]; hasMcp {
					if mcpMap, ok := newMcp.(map[string]any); ok {
						newType, _ = mcpMap["type"].(string)
					}
				}

				if isExistingMCP := isMCPType(existingType); isExistingMCP {
					if isNewMCP := isMCPType(newType); isNewMCP {
						// Both are MCP tools, check for conflicts
						mergedMap, err := mergeMCPTools(existingMap, newMap)
						if err != nil {
							return nil, fmt.Errorf("MCP tool conflict for '%s': %v", key, err)
						}
						result[key] = mergedMap
						continue
					}
				}

				// Both are maps, check for 'allowed' arrays to merge
				if existingAllowed, hasExistingAllowed := existingMap["allowed"]; hasExistingAllowed {
					if newAllowed, hasNewAllowed := newMap["allowed"]; hasNewAllowed {
						// Merge allowed arrays
						merged := mergeAllowedArrays(existingAllowed, newAllowed)
						mergedMap := make(map[string]any)
						for k, v := range existingMap {
							mergedMap[k] = v
						}
						for k, v := range newMap {
							mergedMap[k] = v
						}
						mergedMap["allowed"] = merged
						result[key] = mergedMap
						continue
					}
				}

				// No 'allowed' arrays to merge, recursively merge the maps
				recursiveMerged, err := MergeTools(existingMap, newMap)
				if err != nil {
					return nil, err
				}
				result[key] = recursiveMerged
			} else {
				// Not both same type, overwrite with new value
				result[key] = newValue
			}
		} else {
			// New key, just add it
			result[key] = newValue
		}
	}

	return result, nil
}

// mergeAllowedArrays merges two allowed arrays and removes duplicates
func mergeAllowedArrays(existing, new any) []any {
	var result []any
	seen := make(map[string]bool)

	// Add existing items
	if existingSlice, ok := existing.([]any); ok {
		for _, item := range existingSlice {
			if str, ok := item.(string); ok {
				if !seen[str] {
					result = append(result, str)
					seen[str] = true
				}
			}
		}
	}

	// Add new items
	if newSlice, ok := new.([]any); ok {
		for _, item := range newSlice {
			if str, ok := item.(string); ok {
				if !seen[str] {
					result = append(result, str)
					seen[str] = true
				}
			}
		}
	}

	return result
}

// mergeMCPTools merges two MCP tool configurations, detecting conflicts except for 'allowed' arrays
func mergeMCPTools(existing, new map[string]any) (map[string]any, error) {
	result := make(map[string]any)

	// Copy existing properties
	for k, v := range existing {
		result[k] = v
	}

	// Merge new properties, checking for conflicts
	for key, newValue := range new {
		if existingValue, exists := result[key]; exists {
			if key == "allowed" {
				// Special handling for allowed arrays - merge them
				if existingArray, ok := existingValue.([]any); ok {
					if newArray, ok := newValue.([]any); ok {
						result[key] = mergeAllowedArrays(existingArray, newArray)
						continue
					}
				}
				// If not arrays, fall through to conflict check
			} else if key == "mcp" {
				// Special handling for mcp sub-objects - merge them recursively
				if existingMcp, ok := existingValue.(map[string]any); ok {
					if newMcp, ok := newValue.(map[string]any); ok {
						mergedMcp, err := mergeMCPTools(existingMcp, newMcp)
						if err != nil {
							return nil, fmt.Errorf("MCP config conflict: %v", err)
						}
						result[key] = mergedMcp
						continue
					}
				}
				// If not both maps, fall through to conflict check
			}

			// Check for conflicts (values must be equal)
			if !areEqual(existingValue, newValue) {
				return nil, fmt.Errorf("conflicting values for '%s': existing=%v, new=%v", key, existingValue, newValue)
			}
			// Values are equal, keep existing
		} else {
			// New property, add it
			result[key] = newValue
		}
	}

	return result, nil
}

// areEqual compares two values for equality, handling different types appropriately
func areEqual(a, b any) bool {
	// Convert to JSON for comparison to handle different types consistently
	aJSON, aErr := json.Marshal(a)
	bJSON, bErr := json.Marshal(b)

	if aErr != nil || bErr != nil {
		return false
	}

	return string(aJSON) == string(bJSON)
}
