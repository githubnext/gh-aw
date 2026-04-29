package parser

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/typeutil"
)

var frontmatterHashLog = logger.New("parser:frontmatter_hash")

// templateExpressionRegex matches ${{ ... }} expressions (pre-compiled for performance)
var templateExpressionRegex = regexp.MustCompile(`\$\{\{(.*?)\}\}`)

// parseBoolFromFrontmatter extracts a boolean value from a frontmatter map.
// Returns false if the key is absent, the map is nil, or the value is not a bool.
func parseBoolFromFrontmatter(m map[string]any, key string) bool {
	return typeutil.ParseBool(m, key)
}

// FileReader is a function type that reads file content
// This abstraction allows for different file reading strategies (disk, GitHub API, in-memory, etc.)
type FileReader func(filePath string) ([]byte, error)

// DefaultFileReader reads files from disk using os.ReadFile
var DefaultFileReader FileReader = os.ReadFile

// marshalJSONWithoutHTMLEscape marshals a value to JSON without HTML escaping
// This matches JavaScript's JSON.stringify behavior
func marshalJSONWithoutHTMLEscape(v any) (string, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return "", err
	}
	// Remove the trailing newline that Encoder adds
	result := buf.String()
	return strings.TrimSuffix(result, "\n"), nil
}

// marshalSorted recursively marshals data with sorted keys
func marshalSorted(data any) string {
	switch v := data.(type) {
	case map[string]any:
		if len(v) == 0 {
			return "{}"
		}

		// Sort keys
		keys := make([]string, 0, len(v))
		for key := range v {
			keys = append(keys, key)
		}
		sort.Strings(keys)

		// Build JSON string with sorted keys
		var result strings.Builder
		result.WriteString("{")
		for i, key := range keys {
			if i > 0 {
				result.WriteString(",")
			}
			// Marshal the key without HTML escaping
			keyJSON, err := marshalJSONWithoutHTMLEscape(key)
			if err != nil {
				frontmatterHashLog.Printf("Warning: failed to marshal key %s: %v", key, err)
				continue
			}
			result.WriteString(keyJSON)
			result.WriteString(":")
			// Marshal the value recursively
			result.WriteString(marshalSorted(v[key]))
		}
		result.WriteString("}")
		return result.String()

	case []any:
		if len(v) == 0 {
			return "[]"
		}

		var result strings.Builder
		result.WriteString("[")
		for i, elem := range v {
			if i > 0 {
				result.WriteString(",")
			}
			result.WriteString(marshalSorted(elem))
		}
		result.WriteString("]")
		return result.String()

	case string, int, int64, float64, bool, nil:
		// Use JSON marshaling without HTML escaping to match JavaScript behavior
		jsonStr, err := marshalJSONWithoutHTMLEscape(v)
		if err != nil {
			// This should rarely happen for primitives, but log it for debugging
			frontmatterHashLog.Printf("Warning: failed to marshal primitive value: %v", err)
			return "null"
		}
		return jsonStr

	default:
		// Fallback to JSON marshaling without HTML escaping
		jsonStr, err := marshalJSONWithoutHTMLEscape(v)
		if err != nil {
			frontmatterHashLog.Printf("Warning: failed to marshal value of type %T: %v", v, err)
			return "null"
		}
		return jsonStr
	}
}

// ComputeFrontmatterHashFromParsedContent computes the frontmatter hash from already-parsed
// workflow data, avoiding a redundant file read when content has already been loaded.
// frontmatterText is the raw text between the --- delimiters (e.g. WorkflowData.FrontmatterYAML).
// markdownBody is the raw markdown body before include expansion (e.g. WorkflowData.RawMarkdown).
// parsedFrontmatter is used to detect the inlined-imports flag.
// baseDir is the directory containing the workflow file, used for resolving imports.
func ComputeFrontmatterHashFromParsedContent(frontmatterText, markdownBody string, parsedFrontmatter map[string]any, baseDir string, cache *ImportCache, fileReader FileReader) (string, error) {
	frontmatterHashLog.Printf("Computing hash from parsed content (baseDir=%s)", baseDir)

	inlinedImports := parseBoolFromFrontmatter(parsedFrontmatter, "inlined-imports")

	var relevantExpressions []string
	var fullBody string
	if inlinedImports {
		fullBody = normalizeFrontmatterText(markdownBody)
	} else {
		relevantExpressions = extractRelevantTemplateExpressions(markdownBody)
	}

	return computeFrontmatterHashTextBasedWithReader(frontmatterText, fullBody, baseDir, cache, relevantExpressions, fileReader)
}

// ComputeFrontmatterHashFromFile computes the frontmatter hash for a workflow file
// using text-based approach (no YAML parsing) to match JavaScript implementation
func ComputeFrontmatterHashFromFile(filePath string, cache *ImportCache) (string, error) {
	return ComputeFrontmatterHashFromFileWithReader(filePath, cache, DefaultFileReader)
}

// ComputeFrontmatterHashFromFileWithParsedFrontmatter computes the frontmatter hash using
// a pre-parsed frontmatter map. The parsedFrontmatter must not be nil; callers are responsible
// for parsing the frontmatter before calling this function.
func ComputeFrontmatterHashFromFileWithParsedFrontmatter(filePath string, parsedFrontmatter map[string]any, cache *ImportCache, fileReader FileReader) (string, error) {
	frontmatterHashLog.Printf("Computing hash for file: %s", filePath)

	// Read file content using the provided file reader
	content, err := fileReader(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	return computeFrontmatterHashFromContent(string(content), parsedFrontmatter, filePath, cache, fileReader)
}

// ComputeFrontmatterHashFromFileWithReader computes the frontmatter hash for a workflow file
// using a custom file reader function (e.g., for GitHub API, in-memory file system, etc.)
// It parses the frontmatter once from the file content, then delegates to the core logic.
func ComputeFrontmatterHashFromFileWithReader(filePath string, cache *ImportCache, fileReader FileReader) (string, error) {
	frontmatterHashLog.Printf("Computing hash for file: %s", filePath)

	// Read file content using the provided file reader
	content, err := fileReader(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	// Parse frontmatter once from content; treat inlined-imports as false if parsing fails
	var parsedFrontmatter map[string]any
	if parsed, parseErr := ExtractFrontmatterFromContent(string(content)); parseErr == nil {
		parsedFrontmatter = parsed.Frontmatter
	}

	return computeFrontmatterHashFromContent(string(content), parsedFrontmatter, filePath, cache, fileReader)
}

// computeFrontmatterHashFromContent is the shared core that computes the hash given the
// already-read file content and pre-parsed frontmatter map (may be nil).
func computeFrontmatterHashFromContent(content string, parsedFrontmatter map[string]any, filePath string, cache *ImportCache, fileReader FileReader) (string, error) {
	frontmatterHashLog.Printf("Computing hash from content: filePath=%s, content_size=%d bytes", filePath, len(content))

	// Extract frontmatter and markdown as text (no YAML parsing)
	frontmatterText, markdown, err := extractFrontmatterAndBodyText(content)
	if err != nil {
		return "", fmt.Errorf("failed to extract frontmatter: %w", err)
	}

	// Get base directory for resolving imports
	baseDir := filepath.Dir(filePath)

	// Detect inlined-imports from the pre-parsed frontmatter map.
	// If nil (parsing failed or not provided), inlined-imports is treated as false.
	inlinedImports := parseBoolFromFrontmatter(parsedFrontmatter, "inlined-imports")
	frontmatterHashLog.Printf("Hash strategy: inlined_imports=%v, markdown_size=%d bytes", inlinedImports, len(markdown))

	// When inlined-imports is enabled, the entire markdown body is compiled into the lock
	// file, so any change to the body must invalidate the hash. Include the full body text.
	// Otherwise, only extract the relevant template expressions (env./vars. references).
	var relevantExpressions []string
	var fullBody string
	if inlinedImports {
		fullBody = normalizeFrontmatterText(markdown)
	} else {
		relevantExpressions = extractRelevantTemplateExpressions(markdown)
	}

	// Compute hash using text-based approach with custom file reader
	return computeFrontmatterHashTextBasedWithReader(frontmatterText, fullBody, baseDir, cache, relevantExpressions, fileReader)
}

// extractRelevantTemplateExpressions extracts template expressions from markdown
// that reference env. or vars. contexts
func extractRelevantTemplateExpressions(markdown string) []string {
	frontmatterHashLog.Printf("Extracting relevant template expressions from markdown: size=%d bytes", len(markdown))
	var expressions []string
	seen := make(map[string]bool)

	// Regex to match ${{ ... }} expressions
	matches := templateExpressionRegex.FindAllStringSubmatch(markdown, -1)

	for _, match := range matches {
		if len(match) < 2 {
			continue
		}

		content := strings.TrimSpace(match[1])

		// Check if expression references env. or vars.
		if strings.Contains(content, "env.") || strings.Contains(content, "vars.") {
			// Store the full expression including ${{ }}
			expr := match[0]
			// Deduplicate expressions
			if !seen[expr] {
				expressions = append(expressions, expr)
				seen[expr] = true
			}
		}
	}

	// Sort for deterministic output
	sort.Strings(expressions)
	frontmatterHashLog.Printf("Found %d relevant template expression(s) referencing env./vars.", len(expressions))
	return expressions
}

// extractFrontmatterAndBodyText extracts frontmatter as raw text without parsing YAML
// Returns: frontmatterText, markdownBody, error
func extractFrontmatterAndBodyText(content string) (string, string, error) {
	// Normalize CRLF to LF so that files with Windows line-endings produce the
	// same frontmatter text (and therefore the same hash) as equivalent LF files.
	content = strings.ReplaceAll(content, "\r\n", "\n")

	lines := strings.Split(content, "\n")

	// Check if content starts with frontmatter delimiter
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		// No frontmatter
		return "", content, nil
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
		return "", "", errors.New("frontmatter not properly closed")
	}

	// Extract frontmatter text (lines between --- delimiters)
	frontmatterText := strings.Join(lines[1:endIndex], "\n")

	// Extract markdown body (everything after closing ---)
	var markdown string
	if endIndex+1 < len(lines) {
		markdown = strings.Join(lines[endIndex+1:], "\n")
	}

	return frontmatterText, markdown, nil
}

// normalizeFrontmatterText normalizes frontmatter text for consistent hashing
// Removes leading/trailing whitespace and normalizes line endings
func normalizeFrontmatterText(text string) string {
	// Normalize Windows line endings to Unix
	normalized := strings.ReplaceAll(text, "\r\n", "\n")
	// Trim leading and trailing whitespace
	return strings.TrimSpace(normalized)
}

// extractImportsFromText extracts import paths from frontmatter text using simple text parsing.
// For the array form, extracts all top-level array items under "imports:".
// For the object form, extracts array items under "imports.aw:" only
// (the "apm-packages" subfield contains package names, not import paths).
func extractImportsFromText(frontmatterText string) []string {
	var imports []string
	lines := strings.Split(frontmatterText, "\n")

	inImports := false
	baseIndent := 0
	inAwSubfield := false // true when inside the "aw:" subfield of the object form
	awIndent := 0
	isObjectForm := false // true when imports is in object form (map)

	for i := range lines {
		line := lines[i]
		trimmed := strings.TrimSpace(line)

		// Skip empty lines and comments
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		// Check if this is the imports: key
		if strings.HasPrefix(trimmed, "imports:") {
			inImports = true
			inAwSubfield = false
			isObjectForm = false
			// Find the base indentation (position of first non-whitespace character)
			baseIndent = len(line) - len(strings.TrimLeft(line, " \t"))
			continue
		}

		if inImports {
			// Calculate current line's indentation
			lineIndent := len(line) - len(strings.TrimLeft(line, " \t"))

			// If indentation decreased or same level, we're out of the imports block
			if lineIndent <= baseIndent && trimmed != "" && !strings.HasPrefix(trimmed, "#") {
				break
			}

			// Detect the 'aw:' subfield (object form)
			if lineIndent == baseIndent+2 && strings.HasPrefix(trimmed, "aw:") {
				isObjectForm = true
				inAwSubfield = true
				awIndent = lineIndent
				continue
			}

			// Detect other object-form subfields (e.g. 'apm-packages:') — skip their contents
			if isObjectForm && lineIndent == baseIndent+2 && strings.Contains(trimmed, ":") && !strings.HasPrefix(trimmed, "-") {
				inAwSubfield = false
				continue
			}

			// In array form: collect top-level array items directly under imports:
			// In object form: collect array items only under the 'aw:' subfield
			if strings.HasPrefix(trimmed, "-") {
				if !isObjectForm {
					// Array form — all items belong to imports
					item := strings.TrimSpace(trimmed[1:])
					item = strings.Trim(item, `"'`)
					if item != "" {
						imports = append(imports, item)
					}
				} else if inAwSubfield && lineIndent > awIndent {
					// Object form — only items under 'aw:' are import paths
					item := strings.TrimSpace(trimmed[1:])
					item = strings.Trim(item, `"'`)
					if item != "" {
						imports = append(imports, item)
					}
				}
			}
		}
	}

	return imports
}

// processImportsTextBased processes imports from frontmatter using text-based parsing
// Returns: importedFiles (list of import paths), importedFrontmatterTexts (list of frontmatter texts)
func processImportsTextBased(frontmatterText, baseDir string, visited map[string]bool, fileReader FileReader) ([]string, []string, error) {
	var importedFiles []string
	var importedFrontmatterTexts []string

	// Extract imports from frontmatter text
	imports := extractImportsFromText(frontmatterText)

	if len(imports) == 0 {
		return importedFiles, importedFrontmatterTexts, nil
	}

	frontmatterHashLog.Printf("Processing %d import(s) text-based from baseDir=%s", len(imports), baseDir)

	// Sort imports for deterministic processing
	sort.Strings(imports)

	for _, importPath := range imports {
		// Resolve import path relative to base directory
		fullPath := filepath.Join(baseDir, importPath)

		// Skip if already visited (cycle detection)
		if visited[fullPath] {
			frontmatterHashLog.Printf("Skipping already-visited import (cycle detection): %s", fullPath)
			continue
		}
		visited[fullPath] = true

		// Read imported file using the provided file reader
		content, err := fileReader(fullPath)
		if err != nil {
			// Skip missing imports silently (matches JavaScript behavior)
			continue
		}

		// Extract frontmatter text from imported file
		importFrontmatterText, _, err := extractFrontmatterAndBodyText(string(content))
		if err != nil {
			// Skip files with invalid frontmatter
			continue
		}

		// Add to imported files and texts
		importedFiles = append(importedFiles, importPath)
		importedFrontmatterTexts = append(importedFrontmatterTexts, importFrontmatterText)

		// Recursively process imports in the imported file
		importBaseDir := filepath.Dir(fullPath)
		nestedFiles, nestedTexts, err := processImportsTextBased(importFrontmatterText, importBaseDir, visited, fileReader)
		if err != nil {
			// Continue processing other imports even if one fails
			continue
		}

		// Add nested imports
		importedFiles = append(importedFiles, nestedFiles...)
		importedFrontmatterTexts = append(importedFrontmatterTexts, nestedTexts...)
	}

	frontmatterHashLog.Printf("Processed imports: found %d imported file(s) from baseDir=%s", len(importedFiles), baseDir)
	return importedFiles, importedFrontmatterTexts, nil
}

// computeFrontmatterHashTextBasedWithReader computes the hash using text-based approach with custom file reader.
// When markdown is non-empty, it is included as the full body text in the canonical data (used for
// inlined-imports mode where the entire body is compiled into the lock file).
func computeFrontmatterHashTextBasedWithReader(frontmatterText, markdown, baseDir string, cache *ImportCache, expressions []string, fileReader FileReader) (string, error) {
	frontmatterHashLog.Print("Computing frontmatter hash using text-based approach")

	// Process imports using text-based parsing with custom file reader
	visited := make(map[string]bool)
	importedFiles, importedFrontmatterTexts, err := processImportsTextBased(frontmatterText, baseDir, visited, fileReader)
	if err != nil {
		return "", fmt.Errorf("failed to process imports: %w", err)
	}

	// Build canonical representation from text
	canonical := make(map[string]any)

	// Add the main frontmatter text as-is (trimmed and normalized)
	canonical["frontmatter-text"] = normalizeFrontmatterText(frontmatterText)

	// Add sorted imported files list
	if len(importedFiles) > 0 {
		sort.Strings(importedFiles)
		canonical["imports"] = importedFiles
	}

	// Add sorted imported frontmatter texts (concatenated with delimiter)
	if len(importedFrontmatterTexts) > 0 {
		// Normalize and sort all imported texts
		normalizedTexts := make([]string, len(importedFrontmatterTexts))
		for i, text := range importedFrontmatterTexts {
			normalizedTexts[i] = normalizeFrontmatterText(text)
		}
		sort.Strings(normalizedTexts)
		canonical["imported-frontmatters"] = strings.Join(normalizedTexts, "\n---\n")
	}

	// When inlined-imports is enabled, include the full markdown body so any content
	// change invalidates the hash. Otherwise, include only relevant template expressions.
	if markdown != "" {
		canonical["body-text"] = markdown
	} else if len(expressions) > 0 {
		canonical["template-expressions"] = expressions
	}

	// Serialize to canonical JSON
	canonicalJSON := marshalSorted(canonical)

	frontmatterHashLog.Printf("Canonical JSON length: %d bytes", len(canonicalJSON))

	// Compute SHA-256 hash
	hash := sha256.Sum256([]byte(canonicalJSON))
	hashHex := hex.EncodeToString(hash[:])

	frontmatterHashLog.Printf("Computed hash: %s", hashHex)
	return hashHex, nil
}
