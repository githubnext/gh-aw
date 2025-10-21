package parser

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/githubnext/gh-aw/pkg/console"
	"github.com/goccy/go-yaml"
)

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

// FrontmatterResult holds parsed frontmatter and markdown content
type FrontmatterResult struct {
	Frontmatter map[string]any
	Markdown    string
	// Additional fields for error context
	FrontmatterLines []string // Original frontmatter lines for error context
	FrontmatterStart int      // Line number where frontmatter starts (1-based)
}

// ImportsResult holds the result of processing imports from frontmatter
type ImportsResult struct {
	MergedTools       string   // Merged tools configuration from all imports
	MergedMCPServers  string   // Merged mcp-servers configuration from all imports
	MergedEngines     []string // Merged engine configurations from all imports
	MergedSafeOutputs []string // Merged safe-outputs configurations from all imports
	MergedMarkdown    string   // Merged markdown content from all imports
	MergedSteps       any      // Merged steps configuration (array or object with pre/post-redaction/post)
	MergedRuntimes    string   // Merged runtimes configuration from all imports
	MergedServices    string   // Merged services configuration from all imports
	ImportedFiles     []string // List of imported file paths (for manifest)
}

// ExtractFrontmatterFromContent parses YAML frontmatter from markdown content string
func ExtractFrontmatterFromContent(content string) (*FrontmatterResult, error) {
	lines := strings.Split(content, "\n")

	// Check if file starts with frontmatter delimiter
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		// No frontmatter, return entire content as markdown
		return &FrontmatterResult{
			Frontmatter:      make(map[string]any),
			Markdown:         content,
			FrontmatterLines: []string{},
			FrontmatterStart: 0,
		}, nil
	}

	// Find end of frontmatter
	endIndex := -1
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			endIndex = i
			break
		}
	}

	if endIndex == -1 {
		return nil, fmt.Errorf("frontmatter not properly closed")
	}

	// Extract frontmatter YAML
	frontmatterLines := lines[1:endIndex]
	frontmatterYAML := strings.Join(frontmatterLines, "\n")

	// Parse YAML
	var frontmatter map[string]any
	if err := yaml.Unmarshal([]byte(frontmatterYAML), &frontmatter); err != nil {
		return nil, fmt.Errorf("failed to parse frontmatter: %w", err)
	}

	// Extract markdown content (everything after the closing ---)
	var markdownLines []string
	if endIndex+1 < len(lines) {
		markdownLines = lines[endIndex+1:]
	}
	markdown := strings.Join(markdownLines, "\n")

	return &FrontmatterResult{
		Frontmatter:      frontmatter,
		Markdown:         strings.TrimSpace(markdown),
		FrontmatterLines: frontmatterLines,
		FrontmatterStart: 2, // Line 2 is where frontmatter content starts (after opening ---)
	}, nil
}

// ExtractMarkdownSection extracts a specific section from markdown content
// Supports H1-H3 headers and proper nesting (matches bash implementation)
func ExtractMarkdownSection(content, sectionName string) (string, error) {
	scanner := bufio.NewScanner(strings.NewReader(content))
	var sectionContent bytes.Buffer
	inSection := false
	var sectionLevel int

	// Create regex pattern to match headers at any level (H1-H3) with flexible spacing
	headerPattern := regexp.MustCompile(`^(#{1,3})[\s\t]+` + regexp.QuoteMeta(sectionName) + `[\s\t]*$`)
	levelPattern := regexp.MustCompile(`^(#{1,3})[\s\t]+`)

	for scanner.Scan() {
		line := scanner.Text()

		// Check if this line matches our target section
		if matches := headerPattern.FindStringSubmatch(line); matches != nil {
			inSection = true
			sectionLevel = len(matches[1]) // Number of # characters
			sectionContent.WriteString(line + "\n")
			continue
		}

		// If we're in the section, check if we've hit another header at same or higher level
		if inSection {
			if levelMatches := levelPattern.FindStringSubmatch(line); levelMatches != nil {
				currentLevel := len(levelMatches[1])
				// Stop if we encounter same or higher level header
				if currentLevel <= sectionLevel {
					break
				}
			}
			sectionContent.WriteString(line + "\n")
		}
	}

	if !inSection {
		return "", fmt.Errorf("section '%s' not found", sectionName)
	}

	return strings.TrimSpace(sectionContent.String()), nil
}

// ExtractFrontmatterString extracts only the YAML frontmatter as a string
// This matches the bash extract_frontmatter function
func ExtractFrontmatterString(content string) (string, error) {
	result, err := ExtractFrontmatterFromContent(content)
	if err != nil {
		return "", err
	}

	// Convert frontmatter map back to YAML string
	if len(result.Frontmatter) == 0 {
		return "", nil
	}

	yamlBytes, err := yaml.Marshal(result.Frontmatter)
	if err != nil {
		return "", fmt.Errorf("failed to marshal frontmatter: %w", err)
	}

	return strings.TrimSpace(string(yamlBytes)), nil
}

// ExtractMarkdownContent extracts only the markdown content (excluding frontmatter)
// This matches the bash extract_markdown function
func ExtractMarkdownContent(content string) (string, error) {
	result, err := ExtractFrontmatterFromContent(content)
	if err != nil {
		return "", err
	}

	return result.Markdown, nil
}

// ExtractYamlChunk extracts a specific YAML section with proper indentation handling
// This matches the bash extract_yaml_chunk function exactly
func ExtractYamlChunk(yamlContent, key string) (string, error) {
	if yamlContent == "" || key == "" {
		return "", nil
	}

	scanner := bufio.NewScanner(strings.NewReader(yamlContent))
	var result bytes.Buffer
	inSection := false
	var keyLevel int
	// Match both quoted and unquoted keys
	keyPattern := regexp.MustCompile(`^(\s*)(?:"` + regexp.QuoteMeta(key) + `"|` + regexp.QuoteMeta(key) + `):\s*(.*)$`)

	for scanner.Scan() {
		line := scanner.Text()

		// Skip empty lines when not in section
		if !inSection && strings.TrimSpace(line) == "" {
			continue
		}

		// Check if this line starts our target key
		if matches := keyPattern.FindStringSubmatch(line); matches != nil {
			inSection = true
			keyLevel = len(matches[1]) // Indentation level
			result.WriteString(line + "\n")

			// If it's a single-line value, we're done
			if strings.TrimSpace(matches[2]) != "" {
				break
			}
			continue
		}

		// If we're in the section, check indentation
		if inSection {
			// Skip empty lines
			if strings.TrimSpace(line) == "" {
				continue
			}

			// Count leading spaces
			spaces := 0
			for _, char := range line {
				if char == ' ' {
					spaces++
				} else {
					break
				}
			}

			// If indentation is less than or equal to key level, we've left the section
			if spaces <= keyLevel {
				break
			}

			result.WriteString(line + "\n")
		}
	}

	if !inSection {
		return "", nil
	}

	return strings.TrimRight(result.String(), "\n"), nil
}

// ExtractWorkflowNameFromMarkdown extracts workflow name from first H1 header
// This matches the bash extract_workflow_name_from_markdown function exactly
func ExtractWorkflowNameFromMarkdown(filePath string) (string, error) {
	// First extract markdown content (excluding frontmatter)
	markdownContent, err := ExtractMarkdown(filePath)
	if err != nil {
		return "", err
	}

	// Look for first H1 header (line starting with "# ")
	scanner := bufio.NewScanner(strings.NewReader(markdownContent))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "# ") {
			// Extract text after "# "
			return strings.TrimSpace(line[2:]), nil
		}
	}

	// No H1 header found, generate default name from filename
	return generateDefaultWorkflowName(filePath), nil
}

// generateDefaultWorkflowName creates a default workflow name from filename
// This matches the bash implementation's fallback behavior
func generateDefaultWorkflowName(filePath string) string {
	// Get base filename without extension
	baseName := filepath.Base(filePath)
	baseName = strings.TrimSuffix(baseName, filepath.Ext(baseName))

	// Convert hyphens to spaces
	baseName = strings.ReplaceAll(baseName, "-", " ")

	// Capitalize first letter of each word
	words := strings.Fields(baseName)
	for i, word := range words {
		if len(word) > 0 {
			words[i] = strings.ToUpper(word[:1]) + word[1:]
		}
	}

	return strings.Join(words, " ")
}

// ExtractMarkdown extracts markdown content from a file (excluding frontmatter)
// This matches the bash extract_markdown function
func ExtractMarkdown(filePath string) (string, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to read file %s: %w", filePath, err)
	}

	return ExtractMarkdownContent(string(content))
}

// ProcessImportsFromFrontmatter processes imports field from frontmatter
// Returns merged tools and engines from imported files
func ProcessImportsFromFrontmatter(frontmatter map[string]any, baseDir string) (mergedTools string, mergedEngines []string, err error) {
	result, err := ProcessImportsFromFrontmatterWithManifest(frontmatter, baseDir)
	if err != nil {
		return "", nil, err
	}
	return result.MergedTools, result.MergedEngines, nil
}

// ProcessImportsFromFrontmatterWithManifest processes imports field from frontmatter
// Returns result containing merged tools, engines, markdown content, and list of imported files
func ProcessImportsFromFrontmatterWithManifest(frontmatter map[string]any, baseDir string) (*ImportsResult, error) {
	// Check if imports field exists
	importsField, exists := frontmatter["imports"]
	if !exists {
		return &ImportsResult{}, nil
	}

	// Convert to array of strings
	var imports []string
	switch v := importsField.(type) {
	case []any:
		for _, item := range v {
			if str, ok := item.(string); ok {
				imports = append(imports, str)
			}
		}
	case []string:
		imports = v
	default:
		return nil, fmt.Errorf("imports field must be an array of strings")
	}

	if len(imports) == 0 {
		return &ImportsResult{}, nil
	}

	// Track visited to prevent cycles
	visited := make(map[string]bool)

	// Process each import
	var toolsBuilder strings.Builder
	var mcpServersBuilder strings.Builder
	var markdownBuilder strings.Builder
	var mergedSteps []any  // Array of steps from imports
	var runtimesBuilder strings.Builder
	var servicesBuilder strings.Builder
	var engines []string
	var safeOutputs []string
	var processedFiles []string

	for _, importPath := range imports {
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
		fullPath, err := resolveIncludePath(filePath, baseDir)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve import '%s': %w", filePath, err)
		}

		// Check for cycles
		if visited[fullPath] {
			continue
		}
		visited[fullPath] = true

		// Add to list of processed files (use original importPath for manifest)
		processedFiles = append(processedFiles, importPath)

		// Extract tools from imported file
		toolsContent, err := processIncludedFileWithVisited(fullPath, sectionName, true, visited)
		if err != nil {
			return nil, fmt.Errorf("failed to process imported file '%s': %w", fullPath, err)
		}
		toolsBuilder.WriteString(toolsContent + "\n")

		// Extract markdown content from imported file
		markdownContent, err := processIncludedFileWithVisited(fullPath, sectionName, false, visited)
		if err != nil {
			return nil, fmt.Errorf("failed to process markdown from imported file '%s': %w", fullPath, err)
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
		content, err := os.ReadFile(fullPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read imported file '%s': %w", fullPath, err)
		}

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

		// Extract steps from imported file (all three types merged into single array)
		stepsContent, err := extractStepsFromContent(string(content))
		if err == nil && stepsContent != "" {
			// Parse as array
			var wrapper map[string]any
			if err := yaml.Unmarshal([]byte(stepsContent), &wrapper); err == nil {
				if steps, hasSteps := wrapper["steps"]; hasSteps {
					if arr, ok := steps.([]any); ok {
						mergedSteps = append(mergedSteps, arr...)
					}
				}
			}
		}

		// Extract post-steps from imported file (legacy support during migration)
		postStepsContent, err := extractPostStepsFromContent(string(content))
		if err == nil && postStepsContent != "" {
			// This will be removed after all workflows migrated
		}

		// Extract secret-masking-steps from imported file (legacy support during migration)
		secretMaskingStepsContent, err := extractSecretMaskingStepsFromContent(string(content))
		if err == nil && secretMaskingStepsContent != "" {
			// This will be removed after all workflows migrated
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
	}

	// Convert merged steps to interface{} - nil if no steps
	var finalMergedSteps any
	if len(mergedSteps) > 0 {
		finalMergedSteps = mergedSteps
	}

	return &ImportsResult{
		MergedTools:       toolsBuilder.String(),
		MergedMCPServers:  mcpServersBuilder.String(),
		MergedEngines:     engines,
		MergedSafeOutputs: safeOutputs,
		MergedMarkdown:    markdownBuilder.String(),
		MergedSteps:       finalMergedSteps,
		MergedRuntimes:    runtimesBuilder.String(),
		MergedServices:    servicesBuilder.String(),
		ImportedFiles:     processedFiles,
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
			fullPath, err := resolveIncludePath(filePath, baseDir)
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

// isUnderWorkflowsDirectory checks if a file path is under .github/workflows/ directory
func isUnderWorkflowsDirectory(filePath string) bool {
	// Normalize the path to use forward slashes
	normalizedPath := filepath.ToSlash(filePath)

	// Check if the path contains .github/workflows/
	return strings.Contains(normalizedPath, ".github/workflows/")
}

// resolveIncludePath resolves include path based on workflowspec format or relative path
func resolveIncludePath(filePath, baseDir string) (string, error) {
	// Check if this is a workflowspec (contains owner/repo/path format)
	// Format: owner/repo/path@ref or owner/repo/path@ref#section
	if isWorkflowSpec(filePath) {
		// Download from GitHub using workflowspec
		return downloadIncludeFromWorkflowSpec(filePath)
	}

	// Regular path, resolve relative to base directory
	fullPath := filepath.Join(baseDir, filePath)
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		return "", fmt.Errorf("file not found: %s", fullPath)
	}
	return fullPath, nil
}

// isWorkflowSpec checks if a path looks like a workflowspec (owner/repo/path[@ref])
func isWorkflowSpec(path string) bool {
	// Remove section reference if present
	cleanPath := path
	if idx := strings.Index(path, "#"); idx != -1 {
		cleanPath = path[:idx]
	}

	// Remove ref if present
	if idx := strings.Index(cleanPath, "@"); idx != -1 {
		cleanPath = cleanPath[:idx]
	}

	// Check if it has at least 3 parts (owner/repo/path)
	parts := strings.Split(cleanPath, "/")
	if len(parts) < 3 {
		return false
	}

	// Reject paths that start with "." (local paths like .github/workflows/...)
	if strings.HasPrefix(cleanPath, ".") {
		return false
	}

	// Reject paths that start with "shared/" (local shared files)
	if strings.HasPrefix(cleanPath, "shared/") {
		return false
	}

	// Reject absolute paths
	if strings.HasPrefix(cleanPath, "/") {
		return false
	}

	return true
}

// downloadIncludeFromWorkflowSpec downloads an include file from GitHub using workflowspec
func downloadIncludeFromWorkflowSpec(spec string) (string, error) {
	// Parse the workflowspec
	// Format: owner/repo/path@ref or owner/repo/path@ref#section

	// Remove section reference if present
	cleanSpec := spec
	if idx := strings.Index(spec, "#"); idx != -1 {
		cleanSpec = spec[:idx]
	}

	// Split on @ to get path and ref
	parts := strings.SplitN(cleanSpec, "@", 2)
	pathPart := parts[0]
	var ref string
	if len(parts) == 2 {
		ref = parts[1]
	} else {
		ref = "main" // default to main branch
	}

	// Parse path: owner/repo/path/to/file.md
	slashParts := strings.Split(pathPart, "/")
	if len(slashParts) < 3 {
		return "", fmt.Errorf("invalid workflowspec: must be owner/repo/path[@ref]")
	}

	owner := slashParts[0]
	repo := slashParts[1]
	filePath := strings.Join(slashParts[2:], "/")

	// Download the file content from GitHub
	content, err := downloadFileFromGitHub(owner, repo, filePath, ref)
	if err != nil {
		return "", fmt.Errorf("failed to download include from %s: %w", spec, err)
	}

	// Create a temporary file to store the downloaded content
	tempFile, err := os.CreateTemp("", "gh-aw-include-*.md")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}

	if _, err := tempFile.Write(content); err != nil {
		tempFile.Close()
		os.Remove(tempFile.Name())
		return "", fmt.Errorf("failed to write temp file: %w", err)
	}

	if err := tempFile.Close(); err != nil {
		os.Remove(tempFile.Name())
		return "", fmt.Errorf("failed to close temp file: %w", err)
	}

	return tempFile.Name(), nil
}

// downloadFileFromGitHub downloads a file from GitHub using gh CLI
func downloadFileFromGitHub(owner, repo, path, ref string) ([]byte, error) {
	// Use gh CLI to download the file
	cmd := exec.Command("gh", "api", fmt.Sprintf("/repos/%s/%s/contents/%s?ref=%s", owner, repo, path, ref), "--jq", ".content")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch file content: %w", err)
	}

	// The content is base64 encoded, decode it
	contentBase64 := strings.TrimSpace(string(output))
	cmd = exec.Command("base64", "-d")
	cmd.Stdin = strings.NewReader(contentBase64)
	content, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to decode file content: %w", err)
	}

	return content, nil
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
				// Check for unexpected frontmatter fields (anything other than tools and engine)
				unexpectedFields := make([]string, 0)
				for key := range result.Frontmatter {
					if key != "tools" && key != "engine" {
						unexpectedFields = append(unexpectedFields, key)
					}
				}

				if len(unexpectedFields) > 0 {
					// Show warning for unexpected frontmatter fields
					fmt.Fprintf(os.Stderr, "%s\n", console.FormatWarningMessage(
						fmt.Sprintf("Ignoring unexpected frontmatter fields in %s: %s",
							filePath, strings.Join(unexpectedFields, ", "))))
				}

				// Validate the tools and engine sections if present
				filteredFrontmatter := map[string]any{}
				if tools, hasTools := result.Frontmatter["tools"]; hasTools {
					filteredFrontmatter["tools"] = tools
				}
				if engine, hasEngine := result.Frontmatter["engine"]; hasEngine {
					filteredFrontmatter["engine"] = engine
				}
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

// extractPostStepsFromContent extracts post-steps section from frontmatter as YAML string
func extractPostStepsFromContent(content string) (string, error) {
	result, err := ExtractFrontmatterFromContent(content)
	if err != nil {
		return "", nil // Return empty string on error
	}

	// Extract post-steps section
	postSteps, exists := result.Frontmatter["post-steps"]
	if !exists {
		return "", nil
	}

	// Convert to YAML string (similar to how steps are handled in compiler)
	postStepsYAML, err := yaml.Marshal(postSteps)
	if err != nil {
		return "", nil
	}

	return strings.TrimSpace(string(postStepsYAML)), nil
}

// extractSecretMaskingStepsFromContent extracts secret-masking-steps section from frontmatter as YAML string
func extractSecretMaskingStepsFromContent(content string) (string, error) {
	result, err := ExtractFrontmatterFromContent(content)
	if err != nil {
		return "", nil // Return empty string on error
	}

	// Extract secret-masking-steps section
	secretMaskingSteps, exists := result.Frontmatter["secret-masking-steps"]
	if !exists {
		return "", nil
	}

	// Convert to YAML string (similar to how steps are handled in compiler)
	secretMaskingStepsYAML, err := yaml.Marshal(secretMaskingSteps)
	if err != nil {
		return "", nil
	}

	return strings.TrimSpace(string(secretMaskingStepsYAML)), nil
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
			fullPath, err := resolveIncludePath(filePath, baseDir)
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

// StripANSI removes all ANSI escape sequences from a string
// This handles:
// - CSI (Control Sequence Introducer) sequences: \x1b[...
// - OSC (Operating System Command) sequences: \x1b]...\x07 or \x1b]...\x1b\\
// - Simple escape sequences: \x1b followed by a single character
func StripANSI(s string) string {
	if s == "" {
		return s
	}

	var result strings.Builder
	result.Grow(len(s)) // Pre-allocate capacity for efficiency

	i := 0
	for i < len(s) {
		if s[i] == '\x1b' {
			if i+1 >= len(s) {
				// ESC at end of string, skip it
				i++
				continue
			}
			// Found ESC character, determine sequence type
			switch s[i+1] {
			case '[':
				// CSI sequence: \x1b[...final_char
				// Parameters are in range 0x30-0x3F (0-?), intermediate chars 0x20-0x2F (space-/)
				// Final characters are in range 0x40-0x7E (@-~)
				i += 2 // Skip ESC and [
				for i < len(s) {
					if isFinalCSIChar(s[i]) {
						i++ // Skip the final character
						break
					} else if isCSIParameterChar(s[i]) {
						i++ // Skip parameter/intermediate character
					} else {
						// Invalid character in CSI sequence, stop processing this escape
						break
					}
				}
			case ']':
				// OSC sequence: \x1b]...terminator
				// Terminators: \x07 (BEL) or \x1b\\ (ST)
				i += 2 // Skip ESC and ]
				for i < len(s) {
					if s[i] == '\x07' {
						i++ // Skip BEL
						break
					} else if s[i] == '\x1b' && i+1 < len(s) && s[i+1] == '\\' {
						i += 2 // Skip ESC and \
						break
					}
					i++
				}
			case '(':
				// G0 character set selection: \x1b(char
				i += 2 // Skip ESC and (
				if i < len(s) {
					i++ // Skip the character
				}
			case ')':
				// G1 character set selection: \x1b)char
				i += 2 // Skip ESC and )
				if i < len(s) {
					i++ // Skip the character
				}
			case '=':
				// Application keypad mode: \x1b=
				i += 2
			case '>':
				// Normal keypad mode: \x1b>
				i += 2
			case 'c':
				// Reset: \x1bc
				i += 2
			default:
				// Other escape sequences (2-character)
				// Handle common ones like \x1b7, \x1b8, \x1bD, \x1bE, \x1bH, \x1bM
				if i+1 < len(s) && (s[i+1] >= '0' && s[i+1] <= '~') {
					i += 2
				} else {
					// Invalid or incomplete escape sequence, just skip ESC
					i++
				}
			}
		} else {
			// Regular character, keep it
			result.WriteByte(s[i])
			i++
		}
	}

	return result.String()
}

// isFinalCSIChar checks if a character is a valid CSI final character
// Final characters are in range 0x40-0x7E (@-~)
func isFinalCSIChar(b byte) bool {
	return b >= 0x40 && b <= 0x7E
}

// isCSIParameterChar checks if a character is a valid CSI parameter or intermediate character
// Parameter characters are in range 0x30-0x3F (0-?)
// Intermediate characters are in range 0x20-0x2F (space-/)
func isCSIParameterChar(b byte) bool {
	return (b >= 0x20 && b <= 0x2F) || (b >= 0x30 && b <= 0x3F)
}

// UpdateWorkflowFrontmatter updates the frontmatter of a workflow file using a callback function
func UpdateWorkflowFrontmatter(workflowPath string, updateFunc func(frontmatter map[string]any) error, verbose bool) error {
	// Read the workflow file
	content, err := os.ReadFile(workflowPath)
	if err != nil {
		return fmt.Errorf("failed to read workflow file: %w", err)
	}

	// Parse frontmatter using existing helper
	result, err := ExtractFrontmatterFromContent(string(content))
	if err != nil {
		return fmt.Errorf("failed to parse frontmatter: %w", err)
	}

	// Ensure frontmatter map exists
	if result.Frontmatter == nil {
		result.Frontmatter = make(map[string]any)
	}

	// Apply the update function
	if err := updateFunc(result.Frontmatter); err != nil {
		return err
	}

	// Convert back to YAML
	updatedFrontmatter, err := yaml.Marshal(result.Frontmatter)
	if err != nil {
		return fmt.Errorf("failed to marshal updated frontmatter: %w", err)
	}

	// Reconstruct the file content
	updatedContent, err := reconstructWorkflowFile(string(updatedFrontmatter), result.Markdown)
	if err != nil {
		return fmt.Errorf("failed to reconstruct workflow file: %w", err)
	}

	// Write the updated content back to the file
	if err := os.WriteFile(workflowPath, []byte(updatedContent), 0644); err != nil {
		return fmt.Errorf("failed to write updated workflow file: %w", err)
	}

	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Updated workflow file: %s", console.ToRelativePath(workflowPath))))
	}

	return nil
}

// EnsureToolsSection ensures the tools section exists in frontmatter and returns it
func EnsureToolsSection(frontmatter map[string]any) map[string]any {
	if frontmatter["tools"] == nil {
		frontmatter["tools"] = make(map[string]any)
	}

	tools, ok := frontmatter["tools"].(map[string]any)
	if !ok {
		// If tools exists but is not a map, replace it
		tools = make(map[string]any)
		frontmatter["tools"] = tools
	}

	return tools
}

// reconstructWorkflowFile reconstructs a complete workflow file from frontmatter YAML and markdown content
func reconstructWorkflowFile(frontmatterYAML, markdownContent string) (string, error) {
	var lines []string

	// Add opening frontmatter delimiter
	lines = append(lines, "---")

	// Add frontmatter content (trim trailing newline from YAML marshal)
	frontmatterStr := strings.TrimSuffix(frontmatterYAML, "\n")
	if frontmatterStr != "" {
		lines = append(lines, strings.Split(frontmatterStr, "\n")...)
	}

	// Add closing frontmatter delimiter
	lines = append(lines, "---")

	// Add markdown content if present
	if markdownContent != "" {
		lines = append(lines, markdownContent)
	}

	return strings.Join(lines, "\n"), nil
}
