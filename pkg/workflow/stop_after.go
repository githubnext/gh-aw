package workflow

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/githubnext/gh-aw/pkg/console"
)

// extractStopAfterFromOn extracts the stop-after value from the on: section
func (c *Compiler) extractStopAfterFromOn(frontmatter map[string]any) (string, error) {
	onSection, exists := frontmatter["on"]
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
			return "", fmt.Errorf("stop-after value must be a string")
		}
		return "", nil
	default:
		return "", fmt.Errorf("invalid on: section format")
	}
}

// processStopAfterConfiguration extracts and processes stop-after configuration from frontmatter
func (c *Compiler) processStopAfterConfiguration(frontmatter map[string]any, workflowData *WorkflowData, markdownPath string) error {
	// Extract stop-after from the on: section
	stopAfter, err := c.extractStopAfterFromOn(frontmatter)
	if err != nil {
		return err
	}
	workflowData.StopTime = stopAfter

	// Resolve relative stop-after to absolute time if needed
	if workflowData.StopTime != "" {
		// Check if there's already a lock file with a stop time (recompilation case)
		lockFile := strings.TrimSuffix(markdownPath, ".md") + ".lock.yml"
		existingStopTime := ExtractStopTimeFromLockFile(lockFile)

		if existingStopTime != "" {
			// Preserve existing stop time during recompilation
			workflowData.StopTime = existingStopTime
			if c.verbose {
				fmt.Println(console.FormatInfoMessage(fmt.Sprintf("Preserving existing stop time from lock file: %s", existingStopTime)))
			}
		} else {
			// First compilation or no existing stop time, generate new one
			resolvedStopTime, err := resolveStopTime(workflowData.StopTime, time.Now().UTC())
			if err != nil {
				return fmt.Errorf("invalid stop-after format: %w", err)
			}
			originalStopTime := stopAfter
			workflowData.StopTime = resolvedStopTime

			if c.verbose && isRelativeStopTime(originalStopTime) {
				fmt.Println(console.FormatInfoMessage(fmt.Sprintf("Resolved relative stop-after to: %s", resolvedStopTime)))
			} else if c.verbose && originalStopTime != resolvedStopTime {
				fmt.Println(console.FormatInfoMessage(fmt.Sprintf("Parsed absolute stop-after from '%s' to: %s", originalStopTime, resolvedStopTime)))
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

	// Look for the STOP_TIME line in the safety checks section
	// Pattern: STOP_TIME="YYYY-MM-DD HH:MM:SS"
	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		if strings.Contains(line, "STOP_TIME=") {
			// Extract the value between quotes
			start := strings.Index(line, `"`) + 1
			end := strings.LastIndex(line, `"`)
			if start > 0 && end > start {
				return line[start:end]
			}
		}
	}
	return ""
}
