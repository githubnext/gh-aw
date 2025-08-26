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
	"github.com/goccy/go-yaml"
)

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

// ProcessIncludes processes @include directives in markdown content
// This matches the bash process_includes function behavior
func ProcessIncludes(content, baseDir string, extractTools bool) (string, error) {
	scanner := bufio.NewScanner(strings.NewReader(content))
	var result bytes.Buffer
	includePattern := regexp.MustCompile(`^@include(\?)?\s+(.+)$`)

	for scanner.Scan() {
		line := scanner.Text()

		// Check if this line is an @include directive
		if matches := includePattern.FindStringSubmatch(line); matches != nil {
			isOptional := matches[1] == "?"
			includePath := strings.TrimSpace(matches[2])

			// Handle section references (file.md#Section)
			var filePath, sectionName string
			if strings.Contains(includePath, "#") {
				parts := strings.SplitN(includePath, "#", 2)
				filePath = parts[0]
				sectionName = parts[1]
			} else {
				filePath = includePath
			}

			// Resolve file path
			fullPath, err := resolveIncludePath(filePath, baseDir)
			if err != nil {
				if isOptional {
					// For optional includes, show a friendly informational message to stdout
					if !extractTools {
						fmt.Println(console.FormatInfoMessage(fmt.Sprintf("Optional include file not found: %s. You can create this file to configure the workflow.", filePath)))
					}
					continue
				}
				// For required includes, fail compilation with an error
				return "", fmt.Errorf("failed to resolve required include '%s': %w", filePath, err)
			}

			// Process the included file
			includedContent, err := processIncludedFile(fullPath, sectionName, extractTools)
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

// resolveIncludePath resolves include path based on @ prefix or relative path
func resolveIncludePath(filePath, baseDir string) (string, error) {
	// Regular path, resolve relative to base directory
	fullPath := filepath.Join(baseDir, filePath)
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		return "", fmt.Errorf("file not found: %s", fullPath)
	}
	return fullPath, nil
}

// processIncludedFile processes a single included file, optionally extracting a section
func processIncludedFile(filePath, sectionName string, extractTools bool) (string, error) {
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
				// Check for unexpected frontmatter fields (anything other than tools)
				unexpectedFields := make([]string, 0)
				for key := range result.Frontmatter {
					if key != "tools" {
						unexpectedFields = append(unexpectedFields, key)
					}
				}

				if len(unexpectedFields) > 0 {
					// Show warning for unexpected frontmatter fields
					fmt.Fprintf(os.Stderr, "%s\n", console.FormatWarningMessage(
						fmt.Sprintf("Ignoring unexpected frontmatter fields in %s: %s",
							filePath, strings.Join(unexpectedFields, ", "))))
				}

				// Validate the tools section if present
				if tools, hasTools := result.Frontmatter["tools"]; hasTools {
					filteredFrontmatter := map[string]any{"tools": tools}
					if err := ValidateIncludedFileFrontmatterWithSchemaAndLocation(filteredFrontmatter, filePath); err != nil {
						fmt.Fprintf(os.Stderr, "%s\n", console.FormatWarningMessage(
							fmt.Sprintf("Invalid tools configuration in %s: %v", filePath, err)))
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

// extractToolsFromContent extracts tools section from frontmatter as JSON string
func extractToolsFromContent(content string) (string, error) {
	result, err := ExtractFrontmatterFromContent(content)
	if err != nil {
		return "{}", nil // Return empty object on error to match bash behavior
	}

	// Extract tools section
	tools, exists := result.Frontmatter["tools"]
	if !exists {
		return "{}", nil
	}

	// Convert to JSON string
	toolsJSON, err := json.Marshal(tools)
	if err != nil {
		return "{}", nil
	}

	return strings.TrimSpace(string(toolsJSON)), nil
}

// ExpandIncludes recursively expands @include directives until no more remain
// This matches the bash expand_includes function behavior
func ExpandIncludes(content, baseDir string, extractTools bool) (string, error) {
	const maxDepth = 10
	currentContent := content

	for depth := 0; depth < maxDepth; depth++ {
		// Process includes in current content
		processedContent, err := ProcessIncludes(currentContent, baseDir, extractTools)
		if err != nil {
			return "", err
		}

		// For tools mode, check if we still have @include directives
		if extractTools {
			if !strings.Contains(processedContent, "@include") {
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

	if extractTools {
		// For tools mode, merge all extracted JSON objects
		return mergeToolsFromJSON(currentContent)
	}

	return currentContent, nil
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

// MergeTools merges two tool objects and returns errors for conflicts
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
			existingMap, existingIsMap := existingValue.(map[string]any)
			newMap, newIsMap := newValue.(map[string]any)

			if existingIsMap && newIsMap {
				// Special handling for Claude section in new format
				if key == "claude" {
					result[key] = mergeClaudeSection(existingMap, newMap)
					continue
				}

				// Check if this is an MCP tool (has MCP-compatible type in new format)
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

				// Both are maps, check for 'allowed' arrays to merge at this level
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

				// No 'allowed' arrays to merge at this level, recursively merge the maps
				recursiveMerged, err := MergeTools(existingMap, newMap)
				if err != nil {
					return nil, err
				}
				result[key] = recursiveMerged
			} else {
				// Not both maps, overwrite
				result[key] = newValue
			}
		} else {
			// New key, just add it
			result[key] = newValue
		}
	}

	return result, nil
}

// mergeClaudeSection merges two Claude sections in the new format where tools are under 'allowed'
func mergeClaudeSection(base, additional map[string]any) map[string]any {
	result := make(map[string]any)

	// Copy base
	for k, v := range base {
		result[k] = v
	}

	// Copy additional, merging the allowed section if both have it
	for k, v := range additional {
		if k == "allowed" && result["allowed"] != nil {
			// Both have allowed sections, merge them
			baseAllowed, baseOk := result["allowed"].(map[string]any)
			additionalAllowed, additionalOk := v.(map[string]any)

			if baseOk && additionalOk {
				mergedAllowed := make(map[string]any)

				// Copy base allowed
				for toolName, toolValue := range baseAllowed {
					mergedAllowed[toolName] = toolValue
				}

				// Merge additional allowed
				for toolName, toolValue := range additionalAllowed {
					if existing, exists := mergedAllowed[toolName]; exists {
						// Both have the same tool, merge them
						if toolName == "Bash" {
							// Special handling for Bash - merge command arrays
							mergedAllowed[toolName] = mergeBashCommands(existing, toolValue)
						} else {
							// For other tools, additional overrides base
							mergedAllowed[toolName] = toolValue
						}
					} else {
						// New tool, just add it
						mergedAllowed[toolName] = toolValue
					}
				}

				result["allowed"] = mergedAllowed
			} else {
				// Can't merge, use additional
				result[k] = v
			}
		} else {
			result[k] = v
		}
	}

	return result
}

// mergeBashCommands merges two Bash command configurations
func mergeBashCommands(existing, additional any) any {
	// If either is nil, return non-nil one (nil means allow all)
	if existing == nil {
		return existing // nil means allow all, so keep that
	}
	if additional == nil {
		return additional // nil means allow all, so use that
	}

	// Both are non-nil, try to merge as arrays
	existingArray, existingOk := existing.([]any)
	additionalArray, additionalOk := additional.([]any)

	if existingOk && additionalOk {
		// Merge the arrays
		seen := make(map[string]bool)
		var result []string

		// Add existing commands
		for _, cmd := range existingArray {
			if cmdStr, ok := cmd.(string); ok {
				if !seen[cmdStr] {
					result = append(result, cmdStr)
					seen[cmdStr] = true
				}
			}
		}

		// Add additional commands
		for _, cmd := range additionalArray {
			if cmdStr, ok := cmd.(string); ok {
				if !seen[cmdStr] {
					result = append(result, cmdStr)
					seen[cmdStr] = true
				}
			}
		}

		// Convert back to []any
		var resultAny []any
		for _, cmd := range result {
			resultAny = append(resultAny, cmd)
		}
		return resultAny
	}

	// Can't merge, use additional
	return additional
}

// mergeAllowedArrays merges two allowed arrays and removes duplicates
func mergeAllowedArrays(existing, new any) []string {
	var result []string
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
						merged := mergeAllowedArrays(existingArray, newArray)
						// Convert back to []any
						var mergedAny []any
						for _, item := range merged {
							mergedAny = append(mergedAny, item)
						}
						result[key] = mergedAny
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
