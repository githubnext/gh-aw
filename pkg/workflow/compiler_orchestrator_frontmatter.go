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

// parseFrontmatterSection reads the workflow file and parses its frontmatter.
// It returns a frontmatterParseResult containing the parsed data and validation information.
// If the workflow is detected as a shared workflow (no 'on' field), isSharedWorkflow is set to true.
// If the workflow is detected as a redirect-only file (has redirect but no 'on' field),
// isRedirectOnly is set to true with the redirect target in redirectTarget.
func (c *Compiler) parseFrontmatterSection(markdownPath string) (*frontmatterParseResult, error) {
	orchestratorFrontmatterLog.Printf("Starting frontmatter parsing: %s", markdownPath)
	workflowLog.Printf("Reading file: %s", markdownPath)

	// Clean the path to prevent path traversal issues (gosec G304)
	// filepath.Clean removes ".." and other problematic path elements
	cleanPath := filepath.Clean(markdownPath)

	// Read the file
	content, err := os.ReadFile(cleanPath)
	if err != nil {
		orchestratorFrontmatterLog.Printf("Failed to read file: %s, error: %v", cleanPath, err)
		// Intentionally not wrapping to avoid exposing internal path details
		return nil, fmt.Errorf("failed to read file: %v", err) //nolint:errorfwrapv,errorlint // intentionally not wrapping to avoid exposing os.PathError
	}
	contentString := string(content)

	workflowLog.Printf("File size: %d bytes", len(content))

	// Parse frontmatter and markdown
	orchestratorFrontmatterLog.Printf("Parsing frontmatter from file: %s", cleanPath)
	result, err := parser.ExtractFrontmatterFromContent(contentString)
	if err != nil {
		orchestratorFrontmatterLog.Printf("Frontmatter extraction failed: %v", err)
		// Use FrontmatterStart from result if available, otherwise default to line 2 (after opening ---)
		frontmatterStart := 2
		if result != nil && result.FrontmatterStart > 0 {
			frontmatterStart = result.FrontmatterStart
		}
		return nil, c.createFrontmatterError(cleanPath, contentString, err, frontmatterStart)
	}

	if len(result.Frontmatter) == 0 {
		orchestratorFrontmatterLog.Print("No frontmatter found in file")
		return nil, errors.New("no frontmatter found")
	}

	// Preprocess schedule fields to convert human-friendly format to cron expressions
	if err := c.preprocessScheduleFields(result.Frontmatter, cleanPath, contentString); err != nil {
		orchestratorFrontmatterLog.Printf("Schedule preprocessing failed: %v", err)
		return nil, err
	}

	// Create a copy of frontmatter without internal markers for schema validation
	// Keep the original frontmatter with markers for YAML generation
	frontmatterForValidation := c.copyFrontmatterWithoutInternalMarkers(result.Frontmatter)

	// Check if user accidentally used "triggers:" instead of the correct "on:" keyword
	if _, hasTriggers := frontmatterForValidation["triggers"]; hasTriggers {
		return nil, fmt.Errorf("%s: invalid frontmatter key 'triggers:' — use 'on:' to define workflow triggers", cleanPath)
	}

	// Check if "on" field is missing - if so, treat as a shared/imported workflow
	_, hasOnField := frontmatterForValidation["on"]
	if !hasOnField {
		// Check if this is a redirect-only placeholder (has a redirect field but no 'on' trigger).
		// Redirect-only files are distinct from regular shared workflows: they are placeholders
		// that point to a workflow's new canonical location and are not intended to be imported.
		// They occur when `gh aw add` downloads a workflow that has been moved but the redirect
		// was not resolved to the full content during download.
		if redirectVal, hasRedirect := frontmatterForValidation["redirect"]; hasRedirect {
			if redirectStr, ok := redirectVal.(string); ok {
				if redirectTarget := strings.TrimSpace(redirectStr); redirectTarget != "" {
					detectionLog.Printf("Redirect-only workflow detected: redirect=%s", redirectTarget)
					return &frontmatterParseResult{
						cleanPath:                cleanPath,
						content:                  content,
						frontmatterResult:        result,
						frontmatterForValidation: frontmatterForValidation,
						markdownDir:              filepath.Dir(cleanPath),
						isRedirectOnly:           true,
						redirectTarget:           redirectTarget,
					}, nil
				}
			}
		}

		detectionLog.Printf("No 'on' field detected - treating as shared agentic workflow")

		// Validate as an included/shared workflow (uses main_workflow_schema with forbidden field checks)
		if err := parser.ValidateIncludedFileFrontmatterWithSchemaAndLocation(frontmatterForValidation, cleanPath); err != nil {
			orchestratorFrontmatterLog.Printf("Shared workflow validation failed: %v", err)
			return nil, err
		}

		return &frontmatterParseResult{
			cleanPath:                cleanPath,
			content:                  content,
			frontmatterResult:        result,
			frontmatterForValidation: frontmatterForValidation,
			markdownDir:              filepath.Dir(cleanPath),
			isSharedWorkflow:         true,
		}, nil
	}

	// For main workflows (with 'on' field), markdown content is required
	if result.Markdown == "" {
		orchestratorFrontmatterLog.Print("No markdown content found for main workflow")
		return nil, errors.New("no markdown content found")
	}

	// Validate main workflow frontmatter contains only expected entries
	orchestratorFrontmatterLog.Printf("Validating main workflow frontmatter schema")
	if err := parser.ValidateMainWorkflowFrontmatterWithSchemaAndLocation(frontmatterForValidation, cleanPath); err != nil {
		orchestratorFrontmatterLog.Printf("Main workflow frontmatter validation failed: %v", err)
		return nil, err
	}

	// Validate event filter mutual exclusivity (branches/branches-ignore, paths/paths-ignore)
	if err := ValidateEventFilters(frontmatterForValidation); err != nil {
		orchestratorFrontmatterLog.Printf("Event filter validation failed: %v", err)
		return nil, err
	}

	// Validate event type names in the 'on:' section for potential typos
	if err := ValidateEventTypes(frontmatterForValidation); err != nil {
		orchestratorFrontmatterLog.Printf("Event type validation failed: %v", err)
		return nil, err
	}

	// Validate glob pattern syntax in event filters (branches, tags, paths, etc.)
	if err := ValidateGlobPatterns(frontmatterForValidation); err != nil {
		orchestratorFrontmatterLog.Printf("Glob pattern validation failed: %v", err)
		return nil, err
	}

	// Validate that the runs-on field does not specify unsupported runner types (e.g. macOS)
	if err := validateRunsOn(frontmatterForValidation, cleanPath); err != nil {
		orchestratorFrontmatterLog.Printf("runs-on validation failed: %v", err)
		return nil, err
	}

	// Validate that @include/@import directives are not used inside template regions
	if err := validateNoIncludesInTemplateRegions(result.Markdown); err != nil {
		orchestratorFrontmatterLog.Printf("Template region validation failed: %v", err)
		return nil, fmt.Errorf("template region validation failed: %w", err)
	}

	// Validate that pre-expanded __GH_AW_EXPERIMENTS_*__ placeholders are not used in template conditions
	if err := validateNoPreExpandedExperimentPlaceholders(result.Markdown); err != nil {
		orchestratorFrontmatterLog.Printf("Pre-expanded experiment placeholder validation failed: %v", err)
		return nil, fmt.Errorf("template condition validation failed: %w", err)
	}

	// Warn when experiment comparison expressions use double-quoted string literals.
	// GitHub Actions expression syntax only supports single-quoted string literals, so
	// the compiler converts double quotes to single quotes automatically — but authors
	// should fix the source to use single quotes to keep it consistent with the output.
	for _, w := range detectDoubleQuotedExperimentComparisons(result.Markdown) {
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(w))
		c.IncrementWarningCount()
	}

	workflowLog.Printf("Frontmatter: %d chars, Markdown: %d chars", len(result.Frontmatter), len(result.Markdown))

	return &frontmatterParseResult{
		cleanPath:                cleanPath,
		content:                  content,
		frontmatterResult:        result,
		frontmatterForValidation: frontmatterForValidation,
		markdownDir:              filepath.Dir(cleanPath),
		isSharedWorkflow:         false,
	}, nil
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
