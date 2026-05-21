package parser

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/logger"
)

var includeLog = logger.New("parser:include_processor")

// processIncludesWithVisited processes import directives with cycle detection
// emitIncludeDeprecationWarning emits a deprecation warning for legacy @include / {{#import}} syntax.
func emitIncludeDeprecationWarning(directive *ImportDirectiveMatch) {
	if !directive.IsLegacy {
		return
	}
	optionalMarker := ""
	if directive.IsOptional {
		optionalMarker = "?"
	}
	var suggestion string
	if strings.HasPrefix(strings.TrimSpace(directive.Original), "{{") {
		suggestion = fmt.Sprintf("Use {{#runtime-import%s %s}} for content injection or the 'imports:' frontmatter field for configuration merging.",
			optionalMarker, directive.Path)
	} else {
		suggestion = fmt.Sprintf("Use {{#runtime-import%s %s}} instead.", optionalMarker, directive.Path)
	}
	fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Deprecated syntax: %q. %s",
		directive.Original, suggestion)))
}

// resolveAndProcessInclude resolves an include directive path and processes the included file.
// Returns (content, skip, error) — skip=true means the directive should be silently skipped.
func resolveAndProcessInclude(directive *ImportDirectiveMatch, baseDir string, extractTools bool, visited map[string]bool) (string, bool, error) {
	includePath := directive.Path
	var filePath, sectionName string
	if strings.Contains(includePath, "#") {
		parts := strings.SplitN(includePath, "#", 2)
		filePath, sectionName = parts[0], parts[1]
	} else {
		filePath = includePath
	}
	fullPath, err := ResolveIncludePath(filePath, baseDir, nil)
	if err != nil {
		includeLog.Printf("Failed to resolve include path '%s': %v", filePath, err)
		if directive.IsOptional {
			if !extractTools {
				fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Optional include file not found: %s. You can create this file to configure the workflow.", filePath)))
			}
			return "", true, nil
		}
		return "", false, fmt.Errorf("failed to resolve required include '%s': %w", filePath, err)
	}
	if visited[fullPath] {
		includeLog.Printf("Skipping already included file: %s", fullPath)
		if !extractTools {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Already included: %s, skipping", filePath)))
		}
		return "", true, nil
	}
	includeLog.Printf("Processing include file: %s", fullPath)
	visited[fullPath] = true
	includedContent, err := processIncludedFileWithVisited(fullPath, sectionName, extractTools, visited)
	if err != nil {
		return "", false, fmt.Errorf("failed to process included file '%s': %w", fullPath, err)
	}
	return includedContent, false, nil
}

func processIncludesWithVisited(content, baseDir string, extractTools bool, visited map[string]bool) (string, error) {
	// Fast path: skip scanner allocation when no include/import directives are present.
	// ParseImportDirective only matches lines starting with '@' or '{{#import'.
	// For content mode, preserve the scanner's trailing-newline normalization behavior.
	if !hasIncludeDirectives(content) {
		if extractTools {
			return "", nil
		}
		if !strings.HasSuffix(content, "\n") {
			return content + "\n", nil
		}
		return content, nil
	}
	scanner := bufio.NewScanner(strings.NewReader(content))
	var result bytes.Buffer
	for scanner.Scan() {
		line := scanner.Text()
		directive := ParseImportDirective(line)
		if directive != nil {
			emitIncludeDeprecationWarning(directive)
			includedContent, skip, err := resolveAndProcessInclude(directive, baseDir, extractTools, visited)
			if err != nil {
				return "", err
			}
			if skip {
				continue
			}
			if extractTools {
				result.WriteString(includedContent + "\n")
			} else {
				result.WriteString(includedContent)
			}
		} else if !extractTools {
			result.WriteString(line + "\n")
		}
	}
	return result.String(), nil
}

// checkUnexpectedFrontmatterFields warns about frontmatter fields not in the allowed set.
func checkUnexpectedFrontmatterFields(frontmatter map[string]any, filePath string) {
	validFields := map[string]bool{
		"tools": true, "engine": true, "env": true, "network": true, "mcp-servers": true,
		"imports": true, "name": true, "description": true, "steps": true, "jobs": true,
		"safe-outputs": true, "mcp-scripts": true, "services": true, "runtimes": true,
		"permissions": true, "secret-masking": true, "applyTo": true, "inputs": true,
		"import-schema": true, "infer": true, "disable-model-invocation": true, "features": true,
	}
	var unexpectedFields []string
	for key := range frontmatter {
		if !validFields[key] {
			unexpectedFields = append(unexpectedFields, key)
		}
	}
	if len(unexpectedFields) > 0 {
		fmt.Fprintf(os.Stderr, "%s\n", console.FormatWarningMessage(
			fmt.Sprintf("Ignoring unexpected frontmatter fields in %s: %s",
				filePath, strings.Join(unexpectedFields, ", "))))
	}
}

// validateFilteredFrontmatter validates only the recognized configuration sections.
func validateFilteredFrontmatter(result *FrontmatterResult, filePath string, isAgentFile bool) {
	hasExpressions := frontmatterContainsExpressions(result.Frontmatter)
	filteredFrontmatter := map[string]any{}
	if !isAgentFile && !hasExpressions {
		if tools, hasTools := result.Frontmatter["tools"]; hasTools {
			filteredFrontmatter["tools"] = tools
		}
	}
	if engine, hasEngine := result.Frontmatter["engine"]; hasEngine {
		filteredFrontmatter["engine"] = engine
	}
	if network, hasNetwork := result.Frontmatter["network"]; hasNetwork {
		filteredFrontmatter["network"] = network
	}
	if !hasExpressions {
		if mcpServers, hasMCPServers := result.Frontmatter["mcp-servers"]; hasMCPServers {
			filteredFrontmatter["mcp-servers"] = mcpServers
		}
	}
	if len(filteredFrontmatter) > 0 {
		if err := ValidateIncludedFileFrontmatterWithSchemaAndLocation(filteredFrontmatter, filePath); err != nil {
			fmt.Fprintf(os.Stderr, "%s\n", console.FormatWarningMessage(
				fmt.Sprintf("Invalid configuration in %s: %v", filePath, err)))
		}
	}
}

// applyRelaxedValidation applies warning-only validation for non-workflow files.
func applyRelaxedValidation(result *FrontmatterResult, filePath string, isAgentFile bool) {
	if len(result.Frontmatter) == 0 {
		return
	}
	checkUnexpectedFrontmatterFields(result.Frontmatter, filePath)
	validateFilteredFrontmatter(result, filePath, isAgentFile)
}

// extractToolsFromIncludedFile extracts the tools configuration from a processed included file.
func extractToolsFromIncludedFile(result *FrontmatterResult, filePath string, isAgentFile, isWorkflowFile bool, validationErr error) (string, error) {
	if isAgentFile {
		return "{}", nil
	}
	if validationErr == nil || isWorkflowFile {
		return extractToolsFromFrontmatter(result.Frontmatter)
	}
	if tools, hasTools := result.Frontmatter["tools"]; hasTools {
		toolsJSON, err := json.Marshal(tools)
		if err != nil {
			return "{}", nil
		}
		return strings.TrimSpace(string(toolsJSON)), nil
	}
	return "{}", nil
}

// processIncludedFile processes a single included file, optionally extracting a section
// processIncludedFileWithVisited processes a single included file with cycle detection for nested includes
func processIncludedFileWithVisited(filePath, sectionName string, extractTools bool, visited map[string]bool) (string, error) {
	includeLog.Printf("Reading included file: %s (extractTools=%t, section=%s)", filePath, extractTools, sectionName)
	content, err := readFileFunc(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to read included file %s: %w", filePath, err)
	}
	includeLog.Printf("Read %d bytes from included file: %s", len(content), filePath)
	var result *FrontmatterResult
	if strings.HasPrefix(filePath, BuiltinPathPrefix) {
		result, err = ExtractFrontmatterFromBuiltinFile(filePath, content)
	} else {
		result, err = ExtractFrontmatterFromContent(string(content))
	}
	if err != nil {
		return "", fmt.Errorf("failed to extract frontmatter from included file %s: %w", filePath, err)
	}
	isWorkflowFile := isUnderWorkflowsDirectory(filePath)
	isAgentFile := isCustomAgentFile(filePath)
	var validationErr error
	if !isAgentFile && !strings.HasPrefix(filePath, BuiltinPathPrefix) {
		validationErr = ValidateIncludedFileFrontmatterWithSchemaAndLocation(result.Frontmatter, filePath)
	}
	if validationErr != nil {
		if isWorkflowFile {
			includeLog.Printf("Validation failed for workflow file %s: %v", filePath, validationErr)
			return "", fmt.Errorf("invalid frontmatter in included file %s: %w", filePath, validationErr)
		}
		includeLog.Printf("Validation failed for non-workflow file %s, applying relaxed validation", filePath)
		applyRelaxedValidation(result, filePath, isAgentFile)
	}
	if extractTools {
		return extractToolsFromIncludedFile(result, filePath, isAgentFile, isWorkflowFile, validationErr)
	}
	markdownContent, err := ExtractMarkdownContent(string(content))
	if err != nil {
		return "", fmt.Errorf("failed to extract markdown from %s: %w", filePath, err)
	}
	includedDir := filepath.Dir(filePath)
	markdownContent, err = processIncludesWithVisited(markdownContent, includedDir, extractTools, visited)
	if err != nil {
		return "", fmt.Errorf("failed to process nested includes in %s: %w", filePath, err)
	}
	if sectionName != "" {
		sectionContent, err := ExtractMarkdownSection(markdownContent, sectionName)
		if err != nil {
			return "", fmt.Errorf("failed to extract section '%s' from %s: %w", sectionName, filePath, err)
		}
		return strings.Trim(sectionContent, "\n") + "\n", nil
	}
	return strings.Trim(markdownContent, "\n") + "\n", nil
}

// frontmatterContainsExpressions reports whether any string value in the frontmatter map
// (recursively) contains an unsubstituted ${{ }} expression. Shared workflows that use
// import-schema parameterisation may have ${{ github.aw.import-inputs.* }} expressions in
// their frontmatter fields (e.g. tools.serena) that are only resolved at import time.
// Validation of such files is deferred to avoid false-positive schema warnings.
func frontmatterContainsExpressions(m map[string]any) bool {
	for _, v := range m {
		if containsExpression(v) {
			return true
		}
	}
	return false
}

func containsExpression(v any) bool {
	switch val := v.(type) {
	case string:
		return strings.Contains(val, "${{")
	case map[string]any:
		return frontmatterContainsExpressions(val)
	case []any:
		return slices.ContainsFunc(val, containsExpression)
	}
	return false
}
