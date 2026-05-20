package parser

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/goccy/go-yaml"
)

// FrontmatterResult holds parsed frontmatter and markdown content
type FrontmatterResult struct {
	Frontmatter map[string]any
	Markdown    string
	// Additional fields for error context
	FrontmatterLines []string       // Original frontmatter lines for error context
	FrontmatterStart int            // Line number where frontmatter starts (1-based)
	FieldLines       map[string]int // Absolute line numbers (1-based) of top-level frontmatter keys in the file
}

// extractTopLevelFieldLines scans YAML text and returns a map of top-level key names to
// their absolute line numbers in the source file. frontmatterStart is the 1-based line
// number of the first frontmatter content line (i.e. the line immediately after the
// opening "---" delimiter). The returned line numbers are absolute: they can be used
// directly as file:line positions for IDE-navigable error messages.
func extractTopLevelFieldLines(yamlContent string, frontmatterStart int) map[string]int {
	fieldLines := make(map[string]int)
	relLine := 0
	for line := range strings.SplitSeq(yamlContent, "\n") {
		relLine++
		// Skip empty lines and YAML comments
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		// Top-level keys have no leading indentation
		if len(line) > 0 && line[0] != ' ' && line[0] != '\t' {
			colonIdx := strings.Index(trimmed, ":")
			if colonIdx > 0 {
				key := strings.TrimSpace(trimmed[:colonIdx])
				// Accept simple unquoted keys only. Bracket characters in the key position
				// ({, }, [, ]) indicate inline YAML maps/sequences rather than plain string keys
				// (e.g. `[anchor]: value` or `{implicit_key}: value`). These forms are not used
				// in workflow frontmatter, so we skip them to avoid false positives.
				// Quoted YAML keys such as `"key[0]"` are also not used in workflow frontmatter
				// and are excluded by this check (the extracted substring will contain the quote).
				if key != "" && !strings.ContainsAny(key, " \t{}[]\"'") {
					if _, alreadySeen := fieldLines[key]; !alreadySeen {
						// absoluteLine = relLine + frontmatterStart - 1
						fieldLines[key] = relLine + frontmatterStart - 1
					}
				}
			}
		}
	}
	return fieldLines
}

// ExtractFrontmatterFromContent parses YAML frontmatter from markdown content string
func ExtractFrontmatterFromContent(content string) (*FrontmatterResult, error) {
	parserLog.Printf("Extracting frontmatter from content: size=%d bytes", len(content))
	// Fast-path: inspect only the first line to determine whether frontmatter exists.
	firstNewline := strings.IndexByte(content, '\n')
	firstLine := content
	if firstNewline >= 0 {
		firstLine = content[:firstNewline]
	}

	// Check if file starts with frontmatter delimiter.
	if !isFrontmatterDelimiterLine(firstLine) {
		parserLog.Print("No frontmatter delimiter found, returning content as markdown")
		// No frontmatter, return entire content as markdown
		return &FrontmatterResult{
			Frontmatter:      make(map[string]any),
			Markdown:         content,
			FrontmatterLines: []string{},
			FrontmatterStart: 0,
		}, nil
	}

	// Find end of frontmatter by scanning line-by-line without splitting the entire document.
	searchStart := len(content)
	if firstNewline >= 0 {
		searchStart = firstNewline + 1
	}
	frontmatterEndStart := -1
	markdownStart := len(content)
	for cursor := searchStart; cursor <= len(content); {
		lineStart := cursor
		lineEnd := len(content)
		nextCursor := len(content) + 1

		if cursor < len(content) {
			if relNewline := strings.IndexByte(content[cursor:], '\n'); relNewline >= 0 {
				lineEnd = cursor + relNewline
				nextCursor = lineEnd + 1
			}
		}

		if isFrontmatterDelimiterLine(content[lineStart:lineEnd]) {
			frontmatterEndStart = lineStart
			markdownStart = nextCursor
			break
		}

		if nextCursor > len(content) {
			break
		}
		cursor = nextCursor
	}

	if frontmatterEndStart == -1 {
		return nil, errors.New("frontmatter not properly closed")
	}

	// Extract frontmatter YAML
	frontmatterYAML := content[searchStart:frontmatterEndStart]
	frontmatterLines := []string{}
	if frontmatterYAML != "" {
		frontmatterLines = strings.Split(frontmatterYAML, "\n")
		// Preserve previous behavior from lines[1:endIndex]: a trailing newline before
		// the closing delimiter does not create an additional empty frontmatter line.
		if strings.HasSuffix(frontmatterYAML, "\n") {
			frontmatterLines = frontmatterLines[:len(frontmatterLines)-1]
		}
	}

	// Sanitize no-break whitespace characters (U+00A0) which break the YAML parser
	frontmatterYAML = strings.ReplaceAll(frontmatterYAML, "\u00A0", " ")

	// Parse YAML
	var frontmatter map[string]any
	if err := yaml.Unmarshal([]byte(frontmatterYAML), &frontmatter); err != nil {
		// Use FormatYAMLError to provide source-positioned error output with adjusted line numbers
		// FrontmatterStart is 2 (line 2 is where frontmatter content starts after opening ---)
		formattedErr := FormatYAMLError(err, 2, frontmatterYAML)
		return nil, &FormattedParserError{formatted: "failed to parse frontmatter:\n" + formattedErr, cause: err}
	}

	// Ensure frontmatter is never nil (yaml.Unmarshal sets it to nil for empty YAML)
	if frontmatter == nil {
		frontmatter = make(map[string]any)
	}

	// Extract markdown content (everything after the closing ---)
	markdown := ""
	if markdownStart <= len(content) {
		markdown = content[markdownStart:]
	}

	parserLog.Printf("Successfully extracted frontmatter: fields=%d, markdown_size=%d bytes", len(frontmatter), len(markdown))
	const frontmatterStartLine = 2 // Line 2 is where frontmatter content starts (after opening ---)
	return &FrontmatterResult{
		Frontmatter:      frontmatter,
		Markdown:         strings.TrimSpace(markdown),
		FrontmatterLines: frontmatterLines,
		FrontmatterStart: frontmatterStartLine,
		FieldLines:       extractTopLevelFieldLines(frontmatterYAML, frontmatterStartLine),
	}, nil
}

// isFrontmatterDelimiterLine returns true when a line consists of "---" with optional surrounding whitespace.
func isFrontmatterDelimiterLine(line string) bool {
	// Fast path for common delimiters.
	if line == "---" || line == "---\r" {
		return true
	}

	// Fast path for ASCII-trimmable whitespace.
	start, end := 0, len(line)
	for start < end {
		switch line[start] {
		case ' ', '\t', '\n', '\r', '\v', '\f':
			start++
		default:
			goto leftTrimmed
		}
	}
leftTrimmed:
	if start >= end {
		return false
	}
	for end > start {
		switch line[end-1] {
		case ' ', '\t', '\n', '\r', '\v', '\f':
			end--
		default:
			goto rightTrimmed
		}
	}
rightTrimmed:
	if end-start == 3 && line[start] == '-' && line[start+1] == '-' && line[start+2] == '-' {
		return true
	}

	// Fallback keeps previous behavior for uncommon Unicode whitespace.
	return strings.TrimSpace(line) == "---"
}

// ExtractFrontmatterFromBuiltinFile is a caching wrapper around ExtractFrontmatterFromContent
// for builtin virtual files. Because builtin files are registered once at startup and never
// change, the parsed FrontmatterResult is identical across calls. This function caches the
// first parse result in builtinFrontmatterCache and returns the cached (shared) value on
// subsequent calls, avoiding repeated YAML parsing for frequently imported engine definition
// files.
//
// IMPORTANT: The returned *FrontmatterResult is a shared, read-only reference.
// Callers MUST NOT mutate the result or any of its fields (Frontmatter map, slices, etc.).
// Use ExtractFrontmatterFromContent directly when you need a mutable copy.
//
// path must start with BuiltinPathPrefix ("@builtin:"); an error is returned otherwise.
func ExtractFrontmatterFromBuiltinFile(path string, content []byte) (*FrontmatterResult, error) {
	if !strings.HasPrefix(path, BuiltinPathPrefix) {
		return nil, fmt.Errorf("ExtractFrontmatterFromBuiltinFile: path %q does not start with %q", path, BuiltinPathPrefix)
	}
	if cached, ok := GetBuiltinFrontmatterCache(path); ok {
		return cached, nil
	}
	result, err := ExtractFrontmatterFromContent(string(content))
	if err != nil {
		return nil, err
	}
	// SetBuiltinFrontmatterCache uses LoadOrStore so concurrent races are safe.
	return SetBuiltinFrontmatterCache(path, result), nil
}

// ExtractMarkdownSection extracts a specific section from markdown content
// Supports H1-H3 headers and proper nesting (matches bash implementation)
func ExtractMarkdownSection(content, sectionName string) (string, error) {
	parserLog.Printf("Extracting markdown section: section=%s, content_size=%d bytes", sectionName, len(content))
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
		parserLog.Printf("Section not found: %s", sectionName)
		return "", fmt.Errorf("section '%s' not found", sectionName)
	}

	extractedContent := strings.TrimSpace(sectionContent.String())
	parserLog.Printf("Successfully extracted section: size=%d bytes", len(extractedContent))
	return extractedContent, nil
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

// findH1WorkflowName scans at most the first 64 lines of markdownBody for an H1 header
// and returns the trimmed title. Returns "" if no H1 is found within those lines.
func findH1WorkflowName(markdownBody string) string {
	const maxLines = 64
	lineCount := 0
	for line := range strings.Lines(markdownBody) {
		lineCount++
		if lineCount > maxLines {
			break
		}
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "# ") {
			return strings.TrimSpace(trimmed[2:])
		}
	}
	return ""
}

// ExtractWorkflowNameFromMarkdownBody extracts the workflow name from an already-extracted
// markdown body (i.e. the content after the frontmatter has been stripped). This is more
// efficient than ExtractWorkflowNameFromMarkdown or ExtractWorkflowNameFromContent because it
// avoids the redundant file-read and YAML-parse that those functions perform when the caller
// already holds the parsed FrontmatterResult.
func ExtractWorkflowNameFromMarkdownBody(markdownBody string, virtualPath string) (string, error) {
	parserLog.Printf("Extracting workflow name from markdown body: virtualPath=%s, size=%d bytes", virtualPath, len(markdownBody))

	if name := findH1WorkflowName(markdownBody); name != "" {
		parserLog.Printf("Found workflow name from H1 header: %s", name)
		return name, nil
	}

	defaultName := generateDefaultWorkflowName(virtualPath)
	parserLog.Printf("No H1 header found, using default name: %s", defaultName)
	return defaultName, nil
}

// ExtractWorkflowNameFromContent extracts the workflow name from markdown content string.
// This is the in-memory equivalent of ExtractWorkflowNameFromMarkdown, used by Wasm builds
// where filesystem access is unavailable.
func ExtractWorkflowNameFromContent(content string, virtualPath string) (string, error) {
	parserLog.Printf("Extracting workflow name from content: virtualPath=%s, size=%d bytes", virtualPath, len(content))

	markdownContent, err := ExtractMarkdownContent(content)
	if err != nil {
		return "", err
	}

	if name := findH1WorkflowName(markdownContent); name != "" {
		parserLog.Printf("Found workflow name from H1 header: %s", name)
		return name, nil
	}

	defaultName := generateDefaultWorkflowName(virtualPath)
	parserLog.Printf("No H1 header found, using default name: %s", defaultName)
	return defaultName, nil
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
