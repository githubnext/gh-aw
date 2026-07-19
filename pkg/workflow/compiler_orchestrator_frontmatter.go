package workflow

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/parser"
)

var orchestratorFrontmatterLog = logger.New("workflow:compiler_orchestrator_frontmatter")

// frontmatterParseResult holds the results of parsing and validating frontmatter
type frontmatterParseResult struct {
	cleanPath                string
	content                  []byte
	frontmatterResult        *parser.FrontmatterResult
	frontmatterForValidation map[string]any
	markdownDir              string
	isSharedWorkflow         bool
	// isRedirectOnly is true when the file has a redirect field but no 'on' trigger.
	// Such files are redirect-only placeholders that point to a workflow's new location.
	isRedirectOnly bool
	// redirectTarget holds the redirect destination (workflow spec or URL) for informational messages.
	redirectTarget string
}

type frontmatterReadError struct {
	message string
}

func (e frontmatterReadError) Error() string {
	return e.message
}

func (c *Compiler) validateEngineBeforeSchema(
	cleanPath string,
	content []byte,
	result *parser.FrontmatterResult,
	frontmatterForValidation map[string]any,
) error {
	engineValue, ok := frontmatterForValidation["engine"].(string)
	// Keep the empty-string default-engine behavior, but let whitespace-only values
	// fall through to getAgenticEngine so they surface as invalid engine typos.
	if !ok || engineValue == "" {
		return nil
	}

	if _, err := c.getAgenticEngine(engineValue); err != nil {
		line := result.FieldLines["engine"]
		if line == 0 {
			line = findFrontmatterFieldLine(result.FrontmatterLines, result.FrontmatterStart, "engine")
		}
		if line == 0 {
			line = 1
		}

		return formatCompilerErrorWithContext(
			cleanPath,
			line,
			// Point to the field key for invalid string engine names so the location
			// stays stable even when the specific invalid value changes.
			1,
			"error",
			err.Error(),
			err,
			readSourceContextLines(content, line),
		)
	}

	return nil
}

// parseFrontmatterSection reads the workflow file and parses its frontmatter.
// It returns a frontmatterParseResult containing the parsed data and validation information.
// If the workflow is detected as a shared workflow (no 'on' field), isSharedWorkflow is set to true.
// If the workflow is detected as a redirect-only file (has redirect but no 'on' field),
// isRedirectOnly is set to true with the redirect target in redirectTarget.
func (c *Compiler) parseFrontmatterSection(markdownPath string) (*frontmatterParseResult, error) {
	orchestratorFrontmatterLog.Printf("Starting frontmatter parsing: %s", markdownPath)
	workflowLog.Printf("Reading file: %s", markdownPath)

	cleanPath := filepath.Clean(markdownPath)
	content, err := readFrontmatterFile(cleanPath)
	if err != nil {
		return nil, err
	}
	contentString := string(content)
	workflowLog.Printf("File size: %d bytes", len(content))

	orchestratorFrontmatterLog.Printf("Parsing frontmatter from file: %s", cleanPath)
	result, err := parser.ExtractFrontmatterFromContent(contentString)
	if err != nil {
		return nil, c.handleFrontmatterExtractionError(cleanPath, contentString, result, err)
	}
	if len(result.Frontmatter) == 0 {
		orchestratorFrontmatterLog.Print("No frontmatter found in file")
		return nil, errors.New("no frontmatter found")
	}
	if err := c.preprocessScheduleFields(result.Frontmatter, cleanPath, contentString); err != nil {
		orchestratorFrontmatterLog.Printf("Schedule preprocessing failed: %v", err)
		return nil, err
	}

	frontmatterForValidation := c.copyFrontmatterWithoutInternalMarkers(result.Frontmatter)
	if _, hasTriggers := frontmatterForValidation["triggers"]; hasTriggers {
		return nil, fmt.Errorf("%s: invalid frontmatter key 'triggers:' — use 'on:' to define workflow triggers", cleanPath)
	}

	_, hasOnField := frontmatterForValidation["on"]
	if !hasOnField {
		return handleSharedOrRedirectFrontmatter(cleanPath, content, result, frontmatterForValidation)
	}

	if result.Markdown == "" {
		orchestratorFrontmatterLog.Print("No markdown content found for main workflow")
		return nil, errors.New("no markdown content found")
	}

	if err := c.validateEngineBeforeSchema(cleanPath, content, result, frontmatterForValidation); err != nil {
		orchestratorFrontmatterLog.Printf("String engine pre-validation failed: %v", err)
		return nil, err
	}

	if err := c.validateMainFrontmatterSection(cleanPath, result, frontmatterForValidation); err != nil {
		return nil, err
	}

	workflowLog.Printf("Frontmatter: %d chars, Markdown: %d chars", len(result.Frontmatter), len(result.Markdown))

	return newFrontmatterParseResult(cleanPath, content, result, frontmatterForValidation, false, false, ""), nil
}

func readFrontmatterFile(cleanPath string) ([]byte, error) {
	content, err := os.ReadFile(cleanPath)
	if err != nil {
		orchestratorFrontmatterLog.Printf("Failed to read file: %s, error: %v", cleanPath, err)
		return nil, fmt.Errorf("failed to read file: %w", frontmatterReadError{message: err.Error()})
	}
	return content, nil
}

func (c *Compiler) handleFrontmatterExtractionError(cleanPath, contentString string, result *parser.FrontmatterResult, err error) error {
	orchestratorFrontmatterLog.Printf("Frontmatter extraction failed: %v", err)
	frontmatterStart := 2
	if result != nil && result.FrontmatterStart > 0 {
		frontmatterStart = result.FrontmatterStart
	}
	return c.createFrontmatterError(cleanPath, contentString, err, frontmatterStart)
}

func handleSharedOrRedirectFrontmatter(cleanPath string, content []byte, result *parser.FrontmatterResult, frontmatterForValidation map[string]any) (*frontmatterParseResult, error) {
	if redirectTarget := redirectOnlyTarget(frontmatterForValidation); redirectTarget != "" {
		detectionLog.Printf("Redirect-only workflow detected: redirect=%s", redirectTarget)
		return newFrontmatterParseResult(cleanPath, content, result, frontmatterForValidation, true, false, redirectTarget), nil
	}
	detectionLog.Printf("No 'on' field detected - treating as shared agentic workflow")
	if err := parser.ValidateIncludedFileFrontmatterWithSchemaAndLocation(frontmatterForValidation, cleanPath); err != nil {
		orchestratorFrontmatterLog.Printf("Shared workflow validation failed: %v", err)
		return nil, err
	}
	return newFrontmatterParseResult(cleanPath, content, result, frontmatterForValidation, false, true, ""), nil
}

func redirectOnlyTarget(frontmatterForValidation map[string]any) string {
	if redirectVal, hasRedirect := frontmatterForValidation["redirect"]; hasRedirect {
		if redirectStr, ok := redirectVal.(string); ok {
			return strings.TrimSpace(redirectStr)
		}
	}
	return ""
}

func newFrontmatterParseResult(cleanPath string, content []byte, result *parser.FrontmatterResult, frontmatterForValidation map[string]any, isRedirectOnly, isSharedWorkflow bool, redirectTarget string) *frontmatterParseResult {
	return &frontmatterParseResult{
		cleanPath:                cleanPath,
		content:                  content,
		frontmatterResult:        result,
		frontmatterForValidation: frontmatterForValidation,
		markdownDir:              filepath.Dir(cleanPath),
		isRedirectOnly:           isRedirectOnly,
		isSharedWorkflow:         isSharedWorkflow,
		redirectTarget:           redirectTarget,
	}
}

func (c *Compiler) validateMainFrontmatterSection(cleanPath string, result *parser.FrontmatterResult, frontmatterForValidation map[string]any) error {
	if err := validateMainFrontmatterSchema(cleanPath, frontmatterForValidation); err != nil {
		return err
	}
	if err := c.validateMainFrontmatterEvents(frontmatterForValidation); err != nil {
		return err
	}
	if err := validateRunsOn(frontmatterForValidation, cleanPath); err != nil {
		orchestratorFrontmatterLog.Printf("runs-on validation failed: %v", err)
		return err
	}
	return c.validateMainFrontmatterMarkdown(result.Markdown)
}

func validateMainFrontmatterSchema(cleanPath string, frontmatterForValidation map[string]any) error {
	orchestratorFrontmatterLog.Printf("Validating main workflow frontmatter schema")
	if err := parser.ValidateMainWorkflowFrontmatterWithSchemaAndLocation(frontmatterForValidation, cleanPath); err != nil {
		orchestratorFrontmatterLog.Printf("Main workflow frontmatter validation failed: %v", err)
		return err
	}
	if err := validateFrontmatterSkills(frontmatterForValidation); err != nil {
		orchestratorFrontmatterLog.Printf("Skills frontmatter validation failed: %v", err)
		return err
	}
	return nil
}

func (c *Compiler) validateMainFrontmatterEvents(frontmatterForValidation map[string]any) error {
	if err := ValidateEventFilters(frontmatterForValidation); err != nil {
		orchestratorFrontmatterLog.Printf("Event filter validation failed: %v", err)
		return err
	}
	if err := c.validatePushBranchScope(frontmatterForValidation); err != nil {
		return err
	}
	if err := ValidateEventTypes(frontmatterForValidation); err != nil {
		orchestratorFrontmatterLog.Printf("Event type validation failed: %v", err)
		return err
	}
	if err := ValidateGlobPatterns(frontmatterForValidation); err != nil {
		orchestratorFrontmatterLog.Printf("Glob pattern validation failed: %v", err)
		return err
	}
	return nil
}

func (c *Compiler) validatePushBranchScope(frontmatterForValidation map[string]any) error {
	if err := ValidatePushBranchScope(frontmatterForValidation); err != nil {
		if c.effectiveStrictMode(frontmatterForValidation) {
			orchestratorFrontmatterLog.Printf("Push branch/tag scope validation failed: %v", err)
			return err
		}
		orchestratorFrontmatterLog.Printf("Push branch/tag scope warning (non-strict mode): %v", err)
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(err.Error()))
		c.IncrementWarningCount()
	}
	return nil
}

func (c *Compiler) validateMainFrontmatterMarkdown(markdown string) error {
	if err := validateNoIncludesInTemplateRegions(markdown); err != nil {
		orchestratorFrontmatterLog.Printf("Template region validation failed: %v", err)
		return fmt.Errorf("template region validation failed: %w", err)
	}
	if err := validateNoPreExpandedExperimentPlaceholders(markdown); err != nil {
		orchestratorFrontmatterLog.Printf("Pre-expanded experiment placeholder validation failed: %v", err)
		return fmt.Errorf("template condition validation failed: %w", err)
	}
	for _, w := range detectDoubleQuotedExperimentComparisons(markdown) {
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(w))
		c.IncrementWarningCount()
	}
	return nil
}

// copyFrontmatterWithoutInternalMarkers creates a copy of frontmatter without internal marker fields.
// This is used for schema validation while preserving markers in the original for YAML generation.
// As an optimization, it checks whether any internal markers are present before allocating a copy.
// If no markers exist (the common case for most workflows), the original map is returned as-is.
func (c *Compiler) copyFrontmatterWithoutInternalMarkers(frontmatter map[string]any) map[string]any {
	// Fast path: check if any internal markers are present before allocating a copy.
	// Markers may appear in on.issues, on.pull_request, on.discussion, and on.issue_comment sub-maps.
	hasMarkers := false
	if onValue, hasOn := frontmatter["on"]; hasOn {
		if onMap, ok := onValue.(map[string]any); ok {
			for _, eventKey := range []string{"issues", "pull_request", "discussion", "issue_comment"} {
				if sectionValue, exists := onMap[eventKey]; exists {
					if sectionMap, ok := sectionValue.(map[string]any); ok {
						if _, hasMarker := sectionMap["__gh_aw_native_label_filter__"]; hasMarker {
							hasMarkers = true
							break
						}
					}
				}
			}
		}
	}

	// If no markers found, return the original map directly (no copy needed).
	if !hasMarkers {
		return frontmatter
	}

	// Markers exist: build a copy without them.
	copy := make(map[string]any, len(frontmatter))
	for k, v := range frontmatter {
		if k == "on" {
			// Special handling for "on" field - need to deep copy and remove markers
			if onMap, ok := v.(map[string]any); ok {
				onCopy := make(map[string]any, len(onMap))
				for onKey, onValue := range onMap {
					if onKey == "issues" || onKey == "pull_request" || onKey == "discussion" || onKey == "issue_comment" {
						// Deep copy the section and remove marker
						if sectionMap, ok := onValue.(map[string]any); ok {
							sectionCopy := make(map[string]any, len(sectionMap))
							for sectionKey, sectionValue := range sectionMap {
								if sectionKey != "__gh_aw_native_label_filter__" {
									sectionCopy[sectionKey] = sectionValue
								}
							}
							onCopy[onKey] = sectionCopy
						} else {
							onCopy[onKey] = onValue
						}
					} else {
						onCopy[onKey] = onValue
					}
				}
				copy[k] = onCopy
			} else {
				copy[k] = v
			}
		} else {
			copy[k] = v
		}
	}
	return copy
}
