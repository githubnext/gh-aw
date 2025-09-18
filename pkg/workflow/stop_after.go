package workflow

import (
	"fmt"
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
func (c *Compiler) processStopAfterConfiguration(frontmatter map[string]any, workflowData *WorkflowData) error {
	// Extract stop-after from the on: section
	stopAfter, err := c.extractStopAfterFromOn(frontmatter)
	if err != nil {
		return err
	}
	workflowData.StopTime = stopAfter

	// Resolve relative stop-after to absolute time if needed
	if workflowData.StopTime != "" {
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
		// Parse the relative time delta
		delta, err := parseTimeDelta(stopTime)
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

// generateStopTimeChecks generates safety checks for stop-time before executing agentic tools
func (c *Compiler) generateStopTimeChecks(yaml *strings.Builder, data *WorkflowData) {
	// If no safety settings, skip generating safety checks
	if data.StopTime == "" {
		return
	}

	yaml.WriteString("      - name: Safety checks\n")
	yaml.WriteString("        run: |\n")
	yaml.WriteString("          set -e\n")
	yaml.WriteString("          echo \"Performing safety checks before executing agentic tools...\"\n")

	// Extract workflow name for gh workflow commands
	workflowName := data.Name
	fmt.Fprintf(yaml, "          WORKFLOW_NAME=\"%s\"\n", workflowName)

	// Add stop-time check
	if data.StopTime != "" {
		yaml.WriteString("          \n")
		yaml.WriteString("          # Check stop-time limit\n")
		fmt.Fprintf(yaml, "          STOP_TIME=\"%s\"\n", data.StopTime)
		yaml.WriteString("          echo \"Checking stop-time limit: $STOP_TIME\"\n")
		yaml.WriteString("          \n")
		yaml.WriteString("          # Convert stop time to epoch seconds\n")
		yaml.WriteString("          STOP_EPOCH=$(date -d \"$STOP_TIME\" +%s 2>/dev/null || echo \"invalid\")\n")
		yaml.WriteString("          if [ \"$STOP_EPOCH\" = \"invalid\" ]; then\n")
		yaml.WriteString("            echo \"Warning: Invalid stop-time format: $STOP_TIME. Expected format: YYYY-MM-DD HH:MM:SS\"\n")
		yaml.WriteString("          else\n")
		yaml.WriteString("            CURRENT_EPOCH=$(date +%s)\n")
		yaml.WriteString("            echo \"Current time: $(date)\"\n")
		yaml.WriteString("            echo \"Stop time: $STOP_TIME\"\n")
		yaml.WriteString("            \n")
		yaml.WriteString("            if [ \"$CURRENT_EPOCH\" -ge \"$STOP_EPOCH\" ]; then\n")
		yaml.WriteString("              echo \"Stop time reached. Attempting to disable workflow to prevent cost overrun, then exiting.\"\n")
		yaml.WriteString("              gh workflow disable \"$WORKFLOW_NAME\"\n")
		yaml.WriteString("              echo \"Workflow disabled. No future runs will be triggered.\"\n")
		yaml.WriteString("              exit 1\n")
		yaml.WriteString("            fi\n")
		yaml.WriteString("          fi\n")
	}

	yaml.WriteString("          echo \"All safety checks passed. Proceeding with agentic tool execution.\"\n")
	yaml.WriteString("        env:\n")
	yaml.WriteString("          GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}\n")
}
