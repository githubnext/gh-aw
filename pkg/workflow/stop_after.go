package workflow

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/github/gh-aw/pkg/stringutil"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/logger"
)

var stopAfterLog = logger.New("workflow:stop_after")

// extractStopAfterFromOn extracts the stop-after value from the on: section
func (c *Compiler) extractStopAfterFromOn(frontmatter map[string]any, workflowData ...*WorkflowData) (string, error) {
	// Use cached On field from ParsedFrontmatter if available (when workflowData is provided)
	var onSection any
	var exists bool
	if len(workflowData) > 0 && workflowData[0] != nil && workflowData[0].ParsedFrontmatter != nil && workflowData[0].ParsedFrontmatter.On != nil {
		onSection = workflowData[0].ParsedFrontmatter.On
		exists = true
	} else {
		onSection, exists = frontmatter["on"]
	}

	if !exists {
		return "", nil
	}

	// Handle different formats of the on: section
	switch on := onSection.(type) {
	case string:
		// Simple string format like "on: push" - no stop-after possible
		return "", nil
	case map[string]any:
		// Complex object format - look for stop-after
		if stopAfter, exists := on["stop-after"]; exists {
			if str, ok := stopAfter.(string); ok {
				return str, nil
			}
			return "", fmt.Errorf("stop-after value must be a string, got %T. Example: stop-after: \"+1d\"", stopAfter)
		}
		return "", nil
	default:
		return "", errors.New("invalid on: section format")
	}
}

// processStopAfterConfiguration extracts and processes stop-after configuration from frontmatter
func (c *Compiler) processStopAfterConfiguration(frontmatter map[string]any, workflowData *WorkflowData, markdownPath string) error {
	stopAfterLog.Printf("Processing stop-after configuration for workflow: %s", markdownPath)
	// Extract stop-after from the on: section
	stopAfter, err := c.extractStopAfterFromOn(frontmatter, workflowData)
	if err != nil {
		return err
	}
	workflowData.StopTime = stopAfter

	// Resolve relative stop-after to absolute time if needed
	if workflowData.StopTime != "" {
		stopAfterLog.Printf("Stop-after value specified: %s", workflowData.StopTime)
		// Check if there's already a lock file with a stop time (recompilation case)
		lockFile := stringutil.MarkdownToLockFile(markdownPath)
		existingStopTime := ExtractStopTimeFromLockFile(lockFile)

		// If refresh flag is set, always regenerate the stop time
		if c.refreshStopTime {
			stopAfterLog.Print("Refresh flag set, regenerating stop time")
			resolvedStopTime, err := resolveStopTime(workflowData.StopTime, time.Now().UTC())
			if err != nil {
				return fmt.Errorf("invalid stop-after format: %w", err)
			}
			originalStopTime := stopAfter
			workflowData.StopTime = resolvedStopTime
			stopAfterLog.Printf("Resolved stop time from %s to %s", originalStopTime, resolvedStopTime)

			if c.verbose && isRelativeStopTime(originalStopTime) {
				fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Refreshed relative stop-after to: "+resolvedStopTime))
			} else if c.verbose && originalStopTime != resolvedStopTime {
				fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Refreshed absolute stop-after from '%s' to: %s", originalStopTime, resolvedStopTime)))
			}
		} else if existingStopTime != "" {
			// Preserve existing stop time during recompilation (default behavior)
			stopAfterLog.Printf("Preserving existing stop time from lock file: %s", existingStopTime)
			workflowData.StopTime = existingStopTime
			if c.verbose {
				fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Preserving existing stop time from lock file: "+existingStopTime))
			}
		} else {
			// First compilation or no existing stop time, generate new one
			stopAfterLog.Print("First compilation, generating new stop time")
			resolvedStopTime, err := resolveStopTime(workflowData.StopTime, time.Now().UTC())
			if err != nil {
				return fmt.Errorf("invalid stop-after format: %w", err)
			}
			originalStopTime := stopAfter
			workflowData.StopTime = resolvedStopTime

			if c.verbose && isRelativeStopTime(originalStopTime) {
				fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Resolved relative stop-after to: "+resolvedStopTime))
			} else if c.verbose && originalStopTime != resolvedStopTime {
				fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Parsed absolute stop-after from '%s' to: %s", originalStopTime, resolvedStopTime)))
			}
		}
	}

	return nil
}

// resolveStopTime resolves a stop-time value to an absolute timestamp
// If the stop-time is relative (starts with '+'), it calculates the absolute time
// from the compilation time. Otherwise, it parses the absolute time using various formats.
func resolveStopTime(stopTime string, compilationTime time.Time) (string, error) {
	if stopTime == "" {
		return "", nil
	}

	if isRelativeStopTime(stopTime) {
		// Parse the relative time delta (minutes not allowed for stop-after)
		delta, err := parseTimeDeltaForStopAfter(stopTime)
		if err != nil {
			return "", err
		}

		// Calculate absolute time in UTC using precise calculation
		// Always use AddDate for months, weeks, and days for maximum precision
		absoluteTime := compilationTime.UTC()
		absoluteTime = absoluteTime.AddDate(0, delta.Months, delta.Weeks*7+delta.Days)
		absoluteTime = absoluteTime.Add(time.Duration(delta.Hours)*time.Hour + time.Duration(delta.Minutes)*time.Minute)

		// Format in the expected format: "YYYY-MM-DD HH:MM:SS"
		return absoluteTime.Format("2006-01-02 15:04:05"), nil
	}

	// Parse absolute date-time with flexible format support
	return parseAbsoluteDateTime(stopTime)
}

// ExtractStopTimeFromLockFile extracts the STOP_TIME value from a compiled workflow lock file
func ExtractStopTimeFromLockFile(lockFilePath string) string {
	content, err := os.ReadFile(lockFilePath)
	if err != nil {
		return ""
	}

	contentStr := string(content)

	// Try to extract from metadata first
	metadata, isLegacy, err := ExtractMetadataFromLockFile(contentStr)

	// If metadata extraction failed with an error (malformed JSON), log warning but don't fall back
	// This is different from no metadata (which is intentional for legacy files)
	if err != nil {
		stopAfterLog.Printf("Warning: Failed to parse metadata from %s: %v. Falling back to legacy extraction.", lockFilePath, err)
		// Malformed metadata - fall through to legacy extraction as a safety measure
		// but this indicates a potential issue with the lock file
	} else if metadata != nil && metadata.StopTime != "" {
		// Successfully extracted stop time from metadata
		stopAfterLog.Printf("Extracted stop time from metadata: %s", metadata.StopTime)
		return metadata.StopTime
	}

	// Validate lock file schema compatibility before parsing
	// Non-critical operation - continue even if validation fails
	if err := ValidateLockSchemaCompatibility(contentStr, lockFilePath); err != nil {
		stopAfterLog.Printf("Warning: Lock file schema validation failed for %s: %v", lockFilePath, err)
		// Continue anyway for legacy compatibility
	}

	// Legacy fallback: Look for GH_AW_STOP_TIME in the workflow body
	// Use legacy method if: no metadata, legacy format, metadata exists but stop_time is empty, or metadata was malformed
	if err != nil || metadata == nil || isLegacy || metadata.StopTime == "" {
		lines := strings.SplitSeq(contentStr, "\n")
		for line := range lines {
			// Look for GH_AW_STOP_TIME: YYYY-MM-DD HH:MM:SS
			// This is in the env section of the stop time check job
			if strings.Contains(line, "GH_AW_STOP_TIME:") {
				prefix := "GH_AW_STOP_TIME:"
				if _, after, ok := strings.Cut(line, prefix); ok {
					extracted := strings.TrimSpace(after)
					// Strip surrounding quotes added by newer compiler versions
					extracted = strings.Trim(extracted, "\"")
					return extracted
				}
			}
		}
	}
	return ""
}

// extractSkipIfMatchFromOn extracts the skip-if-match value from the on: section
func (c *Compiler) extractSkipIfMatchFromOn(frontmatter map[string]any, workflowData ...*WorkflowData) (*SkipIfMatchConfig, error) {
	onSection, exists := extractOnSection(frontmatter, workflowData...)
	if !exists {
		return nil, nil
	}

	// Handle different formats of the on: section
	switch on := onSection.(type) {
	case string:
		// Simple string format like "on: push" - no skip-if-match possible
		return nil, nil
	case map[string]any:
		return extractSkipIfMatchFromOnMap(on)
	default:
		return nil, errors.New("invalid on: section format")
	}
}

// extractSkipIfNoMatchFromOn extracts the skip-if-no-match value from the on: section
func (c *Compiler) extractSkipIfNoMatchFromOn(frontmatter map[string]any, workflowData ...*WorkflowData) (*SkipIfNoMatchConfig, error) {
	onSection, exists := extractOnSection(frontmatter, workflowData...)
	if !exists {
		return nil, nil
	}

	// Handle different formats of the on: section
	switch on := onSection.(type) {
	case string:
		// Simple string format like "on: push" - no skip-if-no-match possible
		return nil, nil
	case map[string]any:
		return extractSkipIfNoMatchFromOnMap(on)
	default:
		return nil, errors.New("invalid on: section format")
	}
}

func extractOnSection(frontmatter map[string]any, workflowData ...*WorkflowData) (any, bool) {
	if len(workflowData) > 0 &&
		workflowData[0] != nil &&
		workflowData[0].ParsedFrontmatter != nil &&
		workflowData[0].ParsedFrontmatter.On != nil {
		return workflowData[0].ParsedFrontmatter.On, true
	}
	onSection, exists := frontmatter["on"]
	return onSection, exists
}

func extractSkipIfMatchFromOnMap(on map[string]any) (*SkipIfMatchConfig, error) {
	skipIfMatch, exists := on["skip-if-match"]
	if !exists {
		return nil, nil
	}
	switch skip := skipIfMatch.(type) {
	case string:
		return &SkipIfMatchConfig{Query: skip, Max: 1}, nil
	case map[string]any:
		return parseSkipIfMatchObject(skip)
	default:
		return nil, fmt.Errorf("skip-if-match value must be a string or object, got %T. Examples:\n  skip-if-match: \"is:issue is:open\"\n  skip-if-match:\n    query: \"is:pr is:open\"\n    max: 3", skipIfMatch)
	}
}

func parseSkipIfMatchObject(skip map[string]any) (*SkipIfMatchConfig, error) {
	queryStr, err := requiredSkipQuery(skip, "skip-if-match", "is:issue is:open", "max: 3")
	if err != nil {
		return nil, err
	}
	maxVal, err := optionalPositiveInt(skip, "max", "skip-if-match", 1)
	if err != nil {
		return nil, err
	}
	scope, err := extractSkipIfScope(skip, "skip-if-match")
	if err != nil {
		return nil, err
	}
	return &SkipIfMatchConfig{Query: queryStr, Max: maxVal, Scope: scope}, nil
}

func extractSkipIfNoMatchFromOnMap(on map[string]any) (*SkipIfNoMatchConfig, error) {
	skipIfNoMatch, exists := on["skip-if-no-match"]
	if !exists {
		return nil, nil
	}
	switch skip := skipIfNoMatch.(type) {
	case string:
		return &SkipIfNoMatchConfig{Query: skip, Min: 1}, nil
	case map[string]any:
		return parseSkipIfNoMatchObject(skip)
	default:
		return nil, fmt.Errorf("skip-if-no-match value must be a string or object, got %T. Examples:\n  skip-if-no-match: \"is:pr is:open\"\n  skip-if-no-match:\n    query: \"is:pr is:open\"\n    min: 3", skipIfNoMatch)
	}
}

func parseSkipIfNoMatchObject(skip map[string]any) (*SkipIfNoMatchConfig, error) {
	queryStr, err := requiredSkipQuery(skip, "skip-if-no-match", "is:pr is:open", "min: 3")
	if err != nil {
		return nil, err
	}
	minVal, err := optionalPositiveInt(skip, "min", "skip-if-no-match", 1)
	if err != nil {
		return nil, err
	}
	scope, err := extractSkipIfScope(skip, "skip-if-no-match")
	if err != nil {
		return nil, err
	}
	return &SkipIfNoMatchConfig{Query: queryStr, Min: minVal, Scope: scope}, nil
}

func requiredSkipQuery(skip map[string]any, field, exampleQuery, exampleLimit string) (string, error) {
	queryVal, hasQuery := skip["query"]
	if !hasQuery {
		return "", fmt.Errorf("%s object must have a 'query' field. Example:\n  %s:\n    query: %q\n    %s", field, field, exampleQuery, exampleLimit)
	}
	queryStr, ok := queryVal.(string)
	if !ok {
		return "", fmt.Errorf("%s 'query' field must be a string, got %T", field, queryVal)
	}
	return queryStr, nil
}

func optionalPositiveInt(skip map[string]any, key, field string, defaultValue int) (int, error) {
	value := defaultValue
	raw, exists := skip[key]
	if !exists {
		return value, nil
	}
	switch v := raw.(type) {
	case int:
		value = v
	case int64:
		value = int(v)
	case uint64:
		value = int(v)
	case float64:
		value = int(v)
	default:
		return 0, fmt.Errorf("%s '%s' field must be an integer, got %T. Example: %s: 3", field, key, raw, key)
	}
	if value < 1 {
		return 0, fmt.Errorf("%s '%s' field must be at least 1, got %d", field, key, value)
	}
	return value, nil
}

// processSkipIfMatchConfiguration extracts and processes skip-if-match configuration from frontmatter
func (c *Compiler) processSkipIfMatchConfiguration(frontmatter map[string]any, workflowData *WorkflowData) error {
	// Extract skip-if-match from the on: section
	skipIfMatchConfig, err := c.extractSkipIfMatchFromOn(frontmatter, workflowData)
	if err != nil {
		return err
	}
	workflowData.SkipIfMatch = skipIfMatchConfig

	if workflowData.SkipIfMatch != nil {
		if workflowData.SkipIfMatch.Max == 1 {
			stopAfterLog.Printf("Skip-if-match query configured: %s (max: 1 match)", workflowData.SkipIfMatch.Query)
		} else {
			stopAfterLog.Printf("Skip-if-match query configured: %s (max: %d matches)", workflowData.SkipIfMatch.Query, workflowData.SkipIfMatch.Max)
		}
	}

	return nil
}

// processSkipIfNoMatchConfiguration extracts and processes skip-if-no-match configuration from frontmatter
func (c *Compiler) processSkipIfNoMatchConfiguration(frontmatter map[string]any, workflowData *WorkflowData) error {
	// Extract skip-if-no-match from the on: section
	skipIfNoMatchConfig, err := c.extractSkipIfNoMatchFromOn(frontmatter, workflowData)
	if err != nil {
		return err
	}
	workflowData.SkipIfNoMatch = skipIfNoMatchConfig

	if workflowData.SkipIfNoMatch != nil {
		if workflowData.SkipIfNoMatch.Min == 1 {
			stopAfterLog.Printf("Skip-if-no-match query configured: %s (min: 1 match)", workflowData.SkipIfNoMatch.Query)
		} else {
			stopAfterLog.Printf("Skip-if-no-match query configured: %s (min: %d matches)", workflowData.SkipIfNoMatch.Query, workflowData.SkipIfNoMatch.Min)
		}
	}

	return nil
}

// extractSkipIfCheckFailingFromOn extracts the skip-if-check-failing value from the on: section
func (c *Compiler) extractSkipIfCheckFailingFromOn(frontmatter map[string]any, workflowData ...*WorkflowData) (*SkipIfCheckFailingConfig, error) {
	onSection, exists := extractOnSection(frontmatter, workflowData...)
	if !exists {
		return nil, nil
	}

	// Handle different formats of the on: section
	switch on := onSection.(type) {
	case string:
		// Simple string format like "on: push" - no skip-if-check-failing possible
		return nil, nil
	case map[string]any:
		return extractSkipIfCheckFailingFromOnMap(on)
	default:
		return nil, errors.New("invalid on: section format")
	}
}

func extractSkipIfCheckFailingFromOnMap(on map[string]any) (*SkipIfCheckFailingConfig, error) {
	skipIfCheckFailing, exists := on["skip-if-check-failing"]
	if !exists {
		return nil, nil
	}
	switch skip := skipIfCheckFailing.(type) {
	case nil:
		return &SkipIfCheckFailingConfig{}, nil
	case bool:
		if !skip {
			return nil, errors.New("skip-if-check-failing: false is not valid; remove the field to disable the check")
		}
		return &SkipIfCheckFailingConfig{}, nil
	case map[string]any:
		return parseSkipIfCheckFailingObject(skip)
	default:
		return nil, fmt.Errorf("skip-if-check-failing value must be true or an object, got %T. Examples:\n  skip-if-check-failing:\n  skip-if-check-failing: true\n  skip-if-check-failing:\n    include:\n      - build\n    branch: main\n    allow-pending: true", skipIfCheckFailing)
	}
}

func parseSkipIfCheckFailingObject(skip map[string]any) (*SkipIfCheckFailingConfig, error) {
	config := &SkipIfCheckFailingConfig{}
	var err error
	config.Include, err = skipIfCheckFailingStringList(skip, "include", "      - build\n      - test")
	if err != nil {
		return nil, err
	}
	config.Exclude, err = skipIfCheckFailingStringList(skip, "exclude", "      - lint")
	if err != nil {
		return nil, err
	}
	if branchRaw, hasBranch := skip["branch"]; hasBranch {
		branchStr, ok := branchRaw.(string)
		if !ok {
			return nil, fmt.Errorf("skip-if-check-failing 'branch' field must be a string, got %T. Example: branch: main", branchRaw)
		}
		config.Branch = branchStr
	}
	if allowPendingRaw, hasAllowPending := skip["allow-pending"]; hasAllowPending {
		allowPending, ok := allowPendingRaw.(bool)
		if !ok {
			return nil, fmt.Errorf("skip-if-check-failing 'allow-pending' field must be a boolean, got %T. Example: allow-pending: true", allowPendingRaw)
		}
		config.AllowPending = allowPending
	}
	return config, nil
}

func skipIfCheckFailingStringList(skip map[string]any, field, exampleItems string) ([]string, error) {
	raw, exists := skip[field]
	if !exists {
		return nil, nil
	}
	items, ok := raw.([]any)
	if !ok {
		return nil, fmt.Errorf("skip-if-check-failing '%s' field must be a list of strings. Example:\n  skip-if-check-failing:\n    %s:\n%s", field, field, exampleItems)
	}
	values := make([]string, 0, len(items))
	for _, item := range items {
		s, ok := item.(string)
		if !ok {
			return nil, fmt.Errorf("skip-if-check-failing '%s' list items must be strings, got %T", field, item)
		}
		values = append(values, s)
	}
	return values, nil
}

// processSkipIfCheckFailingConfiguration extracts and processes skip-if-check-failing configuration from frontmatter
func (c *Compiler) processSkipIfCheckFailingConfiguration(frontmatter map[string]any, workflowData *WorkflowData) error {
	skipIfCheckFailingConfig, err := c.extractSkipIfCheckFailingFromOn(frontmatter, workflowData)
	if err != nil {
		return err
	}
	workflowData.SkipIfCheckFailing = skipIfCheckFailingConfig

	if workflowData.SkipIfCheckFailing != nil {
		stopAfterLog.Printf("Skip-if-check-failing configured: include=%v, exclude=%v, branch=%q, allow-pending=%v",
			workflowData.SkipIfCheckFailing.Include,
			workflowData.SkipIfCheckFailing.Exclude,
			workflowData.SkipIfCheckFailing.Branch,
			workflowData.SkipIfCheckFailing.AllowPending,
		)
	}

	return nil
}

// object configuration. Auth fields (github-token, github-app) are configured at the top-level
// on: section and are no longer accepted inside skip-if blocks.
// conditionName is used only for error messages (e.g. "skip-if-match").
func extractSkipIfScope(skip map[string]any, conditionName string) (string, error) {
	if scopeRaw, hasScope := skip["scope"]; hasScope {
		scopeStr, ok := scopeRaw.(string)
		if !ok {
			return "", fmt.Errorf("%s 'scope' field must be a string, got %T. Example: scope: none", conditionName, scopeRaw)
		}
		if scopeStr != "none" {
			return "", fmt.Errorf("%s 'scope' field must be \"none\" or omitted, got %q", conditionName, scopeStr)
		}
		return scopeStr, nil
	}
	return "", nil
}
