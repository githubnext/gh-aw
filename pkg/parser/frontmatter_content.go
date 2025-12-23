package parser

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
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
	FrontmatterLines []string // Original frontmatter lines for error context
	FrontmatterStart int      // Line number where frontmatter starts (1-based)
}

// ExtractFrontmatterFromContent parses YAML frontmatter from markdown content string
func ExtractFrontmatterFromContent(content string) (*FrontmatterResult, error) {
	log.Printf("Extracting frontmatter from content: size=%d bytes", len(content))
	lines := strings.Split(content, "\n")

	// Check if file starts with frontmatter delimiter
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		log.Print("No frontmatter delimiter found, returning content as markdown")
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
		// Use yaml.FormatError to provide colorized, source-positioned error output
		formattedErr := yaml.FormatError(err, true, true)
		return nil, fmt.Errorf("failed to parse frontmatter:\n%s", formattedErr)
	}

	// Ensure frontmatter is never nil (yaml.Unmarshal sets it to nil for empty YAML)
	if frontmatter == nil {
		frontmatter = make(map[string]any)
	}

	// Extract markdown content (everything after the closing ---)
	var markdownLines []string
	if endIndex+1 < len(lines) {
		markdownLines = lines[endIndex+1:]
	}
	markdown := strings.Join(markdownLines, "\n")

	log.Printf("Successfully extracted frontmatter: fields=%d, markdown_size=%d bytes", len(frontmatter), len(markdown))
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
	log.Printf("Extracting markdown section: section=%s, content_size=%d bytes", sectionName, len(content))
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
		log.Printf("Section not found: %s", sectionName)
		return "", fmt.Errorf("section '%s' not found", sectionName)
	}

	extractedContent := strings.TrimSpace(sectionContent.String())
	log.Printf("Successfully extracted section: size=%d bytes", len(extractedContent))
	return extractedContent, nil
}

// ExtractFrontmatterString extracts only the YAML frontmatter as a string
// This matches the bash extract_frontmatter function
func ExtractFrontmatterString(content string) (string, error) {
	log.Printf("Extracting frontmatter string from content: size=%d bytes", len(content))
	result, err := ExtractFrontmatterFromContent(content)
	if err != nil {
		return "", err
	}

	// Convert frontmatter map back to YAML string
	if len(result.Frontmatter) == 0 {
		log.Print("No frontmatter fields found, returning empty string")
		return "", nil
	}

	yamlBytes, err := yaml.Marshal(result.Frontmatter)
	if err != nil {
		return "", fmt.Errorf("failed to marshal frontmatter: %w", err)
	}

	// Post-process YAML to ensure cron expressions are quoted
	// The YAML library may drop quotes from cron expressions like "0 14 * * 1-5"
	// which causes validation errors since they start with numbers but contain spaces
	yamlString := string(yamlBytes)
	yamlString = QuoteCronExpressions(yamlString)

	return strings.TrimSpace(yamlString), nil
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
	log.Printf("Extracting YAML chunk: key=%s, content_size=%d bytes", key, len(yamlContent))

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
	log.Printf("Extracting workflow name from markdown: file=%s", filePath)

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
			workflowName := strings.TrimSpace(line[2:])
			log.Printf("Found workflow name from H1 header: %s", workflowName)
			return workflowName, nil
		}
	}

	// No H1 header found, generate default name from filename
	defaultName := generateDefaultWorkflowName(filePath)
	log.Printf("No H1 header found, using default name: %s", defaultName)
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

// ExtractMarkdown extracts markdown content from a file (excluding frontmatter)
// This matches the bash extract_markdown function
func ExtractMarkdown(filePath string) (string, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to read file %s: %w", filePath, err)
	}

	return ExtractMarkdownContent(string(content))
}
