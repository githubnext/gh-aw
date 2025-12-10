package workflow

import (
	"fmt"
	"strings"

	"github.com/githubnext/gh-aw/pkg/parser"
)

// scheduleFriendlyFormats stores the friendly formats for schedule cron expressions
// Key is: "on.schedule[index]"
var scheduleFriendlyFormats = make(map[string]map[int]string)

// preprocessScheduleFields converts human-friendly schedule expressions to cron expressions
// in the frontmatter's "on" section. It modifies the frontmatter map in place.
func (c *Compiler) preprocessScheduleFields(frontmatter map[string]any) error {
	// Check if "on" field exists
	onValue, exists := frontmatter["on"]
	if !exists {
		return nil
	}

	// Only process if "on" is a map (object format)
	onMap, ok := onValue.(map[string]any)
	if !ok {
		// If "on" is a string, it's a simple event trigger, not a schedule
		return nil
	}

	// Check if schedule field exists in the "on" map
	scheduleValue, hasSchedule := onMap["schedule"]
	if !hasSchedule {
		return nil
	}

	// Handle shorthand string format: schedule: "daily at 02:00"
	if scheduleStr, ok := scheduleValue.(string); ok {
		// Convert string to array format with single item
		parsedCron, original, err := parser.ParseSchedule(scheduleStr)
		if err != nil {
			return fmt.Errorf("invalid schedule expression: %w", err)
		}

		// Create array format
		scheduleArray := []any{
			map[string]any{
				"cron": parsedCron,
			},
		}
		onMap["schedule"] = scheduleArray

		// Store friendly format if it was converted
		if original != "" {
			friendlyFormatsKey := fmt.Sprintf("%p", frontmatter)
			friendlyFormats := make(map[int]string)
			friendlyFormats[0] = original
			scheduleFriendlyFormats[friendlyFormatsKey] = friendlyFormats
		}

		return nil
	}

	// Schedule should be an array of schedule items
	scheduleArray, ok := scheduleValue.([]any)
	if !ok {
		return fmt.Errorf("schedule field must be a string or an array")
	}

	// Store friendly formats in a compiler-specific map
	// Use the frontmatter map's pointer as a unique key
	friendlyFormatsKey := fmt.Sprintf("%p", frontmatter)
	friendlyFormats := make(map[int]string)

	// Process each schedule item
	for i, item := range scheduleArray {
		itemMap, ok := item.(map[string]any)
		if !ok {
			return fmt.Errorf("schedule item %d must be an object with a 'cron' field", i)
		}

		cronValue, hasCron := itemMap["cron"]
		if !hasCron {
			return fmt.Errorf("schedule item %d missing 'cron' field", i)
		}

		cronStr, ok := cronValue.(string)
		if !ok {
			return fmt.Errorf("schedule item %d 'cron' field must be a string", i)
		}

		// Try to parse as human-friendly schedule
		parsedCron, original, err := parser.ParseSchedule(cronStr)
		if err != nil {
			// If parsing fails, it might be an invalid expression
			return fmt.Errorf("invalid schedule expression in item %d: %w", i, err)
		}

		// Update the cron field with the parsed cron expression
		itemMap["cron"] = parsedCron

		// If there was an original friendly format, store it for later use
		if original != "" {
			friendlyFormats[i] = original
		}
	}

	// Store the friendly formats if any were found
	if len(friendlyFormats) > 0 {
		scheduleFriendlyFormats[friendlyFormatsKey] = friendlyFormats
	}

	return nil
}

// addFriendlyScheduleComments adds comments showing the original friendly format for schedule cron expressions
// This function is called after the YAML has been generated from the frontmatter
func (c *Compiler) addFriendlyScheduleComments(yamlStr string, frontmatter map[string]any) string {
	// Retrieve the friendly formats for this frontmatter
	friendlyFormatsKey := fmt.Sprintf("%p", frontmatter)
	friendlyFormats, exists := scheduleFriendlyFormats[friendlyFormatsKey]
	if !exists || len(friendlyFormats) == 0 {
		return yamlStr
	}

	// Process the YAML string to add comments
	lines := strings.Split(yamlStr, "\n")
	var result []string
	scheduleItemIndex := -1
	inScheduleArray := false

	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)

		// Check if we're entering the schedule array
		if strings.HasPrefix(trimmedLine, "schedule:") {
			inScheduleArray = true
			scheduleItemIndex = -1
			result = append(result, line)
			continue
		}

		// Check if we're leaving the schedule section (new top-level key)
		if inScheduleArray && strings.TrimSpace(line) != "" && !strings.HasPrefix(line, "  ") && !strings.HasPrefix(line, "\t") {
			inScheduleArray = false
		}

		// If we're in the schedule array and find a cron line, add the friendly comment
		if inScheduleArray && strings.Contains(trimmedLine, "cron:") {
			scheduleItemIndex++
			result = append(result, line)

			// Add friendly format comment if available
			if friendly, exists := friendlyFormats[scheduleItemIndex]; exists {
				// Get the indentation of the cron line
				indentation := ""
				if len(line) > len(trimmedLine) {
					indentation = line[:len(line)-len(trimmedLine)]
				}
				// Add comment with friendly format on the next line
				comment := indentation + "  # Friendly format: " + friendly
				result = append(result, comment)
			}
			continue
		}

		result = append(result, line)
	}

	return strings.Join(result, "\n")
}
