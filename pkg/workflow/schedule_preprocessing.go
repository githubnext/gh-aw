package workflow

import (
	"fmt"
	"strings"

	"github.com/githubnext/gh-aw/pkg/logger"
	"github.com/githubnext/gh-aw/pkg/parser"
)

var schedulePreprocessingLog = logger.New("workflow:schedule_preprocessing")

// scheduleFriendlyFormats stores the friendly formats for schedule cron expressions
// Key is: "on.schedule[index]"
var scheduleFriendlyFormats = make(map[string]map[int]string)

// preprocessScheduleFields converts human-friendly schedule expressions to cron expressions
// in the frontmatter's "on" section. It modifies the frontmatter map in place.
func (c *Compiler) preprocessScheduleFields(frontmatter map[string]any) error {
	schedulePreprocessingLog.Print("Preprocessing schedule fields in frontmatter")

	// Check if "on" field exists
	onValue, exists := frontmatter["on"]
	if !exists {
		return nil
	}

	// Check if "on" is a string - might be a schedule expression, slash command shorthand, or label trigger shorthand
	if onStr, ok := onValue.(string); ok {
		schedulePreprocessingLog.Printf("Processing on field as string: %s", onStr)

		// Check if it's a slash command shorthand (starts with /)
		commandName, isSlashCommand, err := parseSlashCommandShorthand(onStr)
		if err != nil {
			return err
		}
		if isSlashCommand {
			schedulePreprocessingLog.Printf("Converting shorthand 'on: %s' to slash_command + workflow_dispatch", onStr)

			// Create the expanded format
			onMap := expandSlashCommandShorthand(commandName)
			frontmatter["on"] = onMap

			return nil
		}

		// Check if it's a label trigger shorthand (labeled label1 label2...)
		entityType, labelNames, isLabelTrigger, err := parseLabelTriggerShorthand(onStr)
		if err != nil {
			return err
		}
		if isLabelTrigger {
			schedulePreprocessingLog.Printf("Converting shorthand 'on: %s' to %s labeled + workflow_dispatch", onStr, entityType)

			// Create the expanded format
			onMap := expandLabelTriggerShorthand(entityType, labelNames)
			frontmatter["on"] = onMap

			return nil
		}

		// Try to parse as a schedule expression
		parsedCron, original, err := parser.ParseSchedule(onStr)
		if err != nil {
			// Not a schedule expression, treat as a simple event trigger
			schedulePreprocessingLog.Printf("Not a schedule expression: %s", onStr)
			return nil
		}

		schedulePreprocessingLog.Printf("Converting shorthand 'on: %s' to schedule + workflow_dispatch", onStr)

		// Warn if using explicit daily cron pattern
		if parser.IsDailyCron(parsedCron) && !parser.IsFuzzyCron(parsedCron) {
			c.addDailyCronWarning(parsedCron)
		}

		// Warn if using hourly interval with fixed minute
		if parser.IsHourlyCron(parsedCron) && !parser.IsFuzzyCron(parsedCron) {
			c.addHourlyCronWarning(parsedCron)
		}

		// Warn if using explicit weekly cron pattern with fixed time
		if parser.IsWeeklyCron(parsedCron) && !parser.IsFuzzyCron(parsedCron) {
			c.addWeeklyCronWarning(parsedCron)
		}

		// Scatter fuzzy schedules if workflow identifier is set
		if parser.IsFuzzyCron(parsedCron) && c.workflowIdentifier != "" {
			// Combine repo slug and workflow identifier for scattering seed
			seed := c.workflowIdentifier
			if c.repositorySlug != "" {
				seed = c.repositorySlug + "/" + c.workflowIdentifier
			}
			scatteredCron, err := parser.ScatterSchedule(parsedCron, seed)
			if err != nil {
				schedulePreprocessingLog.Printf("Warning: failed to scatter fuzzy schedule: %v", err)
				// Keep the original fuzzy schedule as fallback
			} else {
				schedulePreprocessingLog.Printf("Scattered fuzzy schedule %s to %s for workflow %s", parsedCron, scatteredCron, c.workflowIdentifier)
				parsedCron = scatteredCron
				// Update the friendly format to show the scattering
				if original != "" {
					original = fmt.Sprintf("%s (scattered)", original)
				}
			}
		}

		// Validate final cron expression has correct syntax (5 fields)
		// FUZZY cron expressions are not supported by GitHub Actions
		if parser.IsFuzzyCron(parsedCron) {
			return fmt.Errorf("fuzzy cron expression '%s' must be scattered to proper cron format before compilation (ensure workflow identifier is set)", parsedCron)
		}
		if !parser.IsCronExpression(parsedCron) {
			return fmt.Errorf("invalid cron expression '%s': must have exactly 5 fields (minute hour day-of-month month day-of-week)", parsedCron)
		}

		// Create schedule array format with workflow_dispatch
		scheduleArray := []any{
			map[string]any{
				"cron": parsedCron,
			},
		}

		// Replace the simple "on: schedule" with expanded format
		onMap := map[string]any{
			"schedule":          scheduleArray,
			"workflow_dispatch": nil,
		}
		frontmatter["on"] = onMap

		// Store friendly format if it was converted
		if original != "" {
			friendlyFormatsKey := fmt.Sprintf("%p", frontmatter)
			friendlyFormats := make(map[int]string)
			friendlyFormats[0] = original
			scheduleFriendlyFormats[friendlyFormatsKey] = friendlyFormats
		}

		return nil
	}

	// Only process if "on" is a map (object format)
	onMap, ok := onValue.(map[string]any)
	if !ok {
		// If "on" is neither string nor map, something is wrong
		return nil
	}

	// Check if schedule field exists in the "on" map
	scheduleValue, hasSchedule := onMap["schedule"]
	if !hasSchedule {
		return nil
	}

	// Handle shorthand string format: schedule: "daily at 02:00"
	if scheduleStr, ok := scheduleValue.(string); ok {
		schedulePreprocessingLog.Printf("Converting shorthand schedule string to array format: %s", scheduleStr)
		// Convert string to array format with single item
		parsedCron, original, err := parser.ParseSchedule(scheduleStr)
		if err != nil {
			return fmt.Errorf("invalid schedule expression: %w", err)
		}

		// Warn if using explicit daily cron pattern
		if parser.IsDailyCron(parsedCron) && !parser.IsFuzzyCron(parsedCron) {
			c.addDailyCronWarning(parsedCron)
		}

		// Warn if using hourly interval with fixed minute
		if parser.IsHourlyCron(parsedCron) && !parser.IsFuzzyCron(parsedCron) {
			c.addHourlyCronWarning(parsedCron)
		}

		// Warn if using explicit weekly cron pattern with fixed time
		if parser.IsWeeklyCron(parsedCron) && !parser.IsFuzzyCron(parsedCron) {
			c.addWeeklyCronWarning(parsedCron)
		}

		// Scatter fuzzy schedules if workflow identifier is set
		if parser.IsFuzzyCron(parsedCron) && c.workflowIdentifier != "" {
			// Combine repo slug and workflow identifier for scattering seed
			seed := c.workflowIdentifier
			if c.repositorySlug != "" {
				seed = c.repositorySlug + "/" + c.workflowIdentifier
			}
			scatteredCron, err := parser.ScatterSchedule(parsedCron, seed)
			if err != nil {
				schedulePreprocessingLog.Printf("Warning: failed to scatter fuzzy schedule: %v", err)
				// Keep the original fuzzy schedule as fallback
			} else {
				schedulePreprocessingLog.Printf("Scattered fuzzy schedule %s to %s for workflow %s", parsedCron, scatteredCron, c.workflowIdentifier)
				parsedCron = scatteredCron
				// Update the friendly format to show the scattering
				if original != "" {
					original = fmt.Sprintf("%s (scattered)", original)
				}
			}
		}

		// Validate final cron expression has correct syntax (5 fields)
		// FUZZY cron expressions are not supported by GitHub Actions
		if parser.IsFuzzyCron(parsedCron) {
			return fmt.Errorf("fuzzy cron expression '%s' must be scattered to proper cron format before compilation (ensure workflow identifier is set)", parsedCron)
		}
		if !parser.IsCronExpression(parsedCron) {
			return fmt.Errorf("invalid cron expression '%s': must have exactly 5 fields (minute hour day-of-month month day-of-week)", parsedCron)
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
	schedulePreprocessingLog.Printf("Processing %d schedule items", len(scheduleArray))
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

		// Warn if using explicit daily cron pattern
		if parser.IsDailyCron(parsedCron) && !parser.IsFuzzyCron(parsedCron) {
			c.addDailyCronWarning(parsedCron)
		}

		// Warn if using hourly interval with fixed minute
		if parser.IsHourlyCron(parsedCron) && !parser.IsFuzzyCron(parsedCron) {
			c.addHourlyCronWarning(parsedCron)
		}

		// Warn if using explicit weekly cron pattern with fixed time
		if parser.IsWeeklyCron(parsedCron) && !parser.IsFuzzyCron(parsedCron) {
			c.addWeeklyCronWarning(parsedCron)
		}

		// Scatter fuzzy schedules if workflow identifier is set
		if parser.IsFuzzyCron(parsedCron) && c.workflowIdentifier != "" {
			// Combine repo slug and workflow identifier for scattering seed
			seed := c.workflowIdentifier
			if c.repositorySlug != "" {
				seed = c.repositorySlug + "/" + c.workflowIdentifier
			}
			scatteredCron, err := parser.ScatterSchedule(parsedCron, seed)
			if err != nil {
				schedulePreprocessingLog.Printf("Warning: failed to scatter fuzzy schedule: %v", err)
				// Keep the original fuzzy schedule as fallback
			} else {
				schedulePreprocessingLog.Printf("Scattered fuzzy schedule %s to %s for workflow %s", parsedCron, scatteredCron, c.workflowIdentifier)
				parsedCron = scatteredCron
				// Update the friendly format to show the scattering
				if original != "" {
					original = fmt.Sprintf("%s (scattered)", original)
				}
			}
		}

		// Validate final cron expression has correct syntax (5 fields)
		// FUZZY cron expressions are not supported by GitHub Actions
		if parser.IsFuzzyCron(parsedCron) {
			return fmt.Errorf("fuzzy cron expression '%s' in item %d must be scattered to proper cron format before compilation (ensure workflow identifier is set)", parsedCron, i)
		}
		if !parser.IsCronExpression(parsedCron) {
			return fmt.Errorf("invalid cron expression '%s' in item %d: must have exactly 5 fields (minute hour day-of-month month day-of-week)", parsedCron, i)
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

// addDailyCronWarning emits a warning when a daily cron pattern with fixed time is detected
func (c *Compiler) addDailyCronWarning(cronExpr string) {
	// Extract hour and minute from the cron expression
	fields := strings.Fields(cronExpr)
	if len(fields) >= 2 {
		minute := fields[0]
		hour := fields[1]
		schedulePreprocessingLog.Printf("Warning: detected daily cron with fixed time: %s", cronExpr)

		// Construct the warning message
		warningMsg := fmt.Sprintf(
			"Schedule uses fixed daily time (%s:%s UTC). Consider using fuzzy schedule 'daily' instead to distribute workflow execution times and reduce load spikes.",
			hour, minute,
		)

		// This warning is added to the warning count
		// It will be collected and displayed by the compilation process
		c.IncrementWarningCount()

		// Store the warning for later display
		c.addScheduleWarning(warningMsg)
	}
}

// addHourlyCronWarning emits a warning when an hourly interval with fixed minute is detected
func (c *Compiler) addHourlyCronWarning(cronExpr string) {
	// Extract minute and interval from the cron expression
	fields := strings.Fields(cronExpr)
	if len(fields) >= 2 {
		minute := fields[0]
		hourField := fields[1]
		schedulePreprocessingLog.Printf("Warning: detected hourly cron with fixed minute: %s", cronExpr)

		// Extract the interval from */N pattern
		interval := strings.TrimPrefix(hourField, "*/")

		// Construct the warning message
		warningMsg := fmt.Sprintf(
			"Schedule uses hourly interval with fixed minute offset (%s). Consider using fuzzy schedule 'every %sh' instead to distribute workflow execution times and reduce load spikes.",
			minute, interval,
		)

		// This warning is added to the warning count
		c.IncrementWarningCount()

		// Store the warning for later display
		c.addScheduleWarning(warningMsg)
	}
}

// addWeeklyCronWarning emits a warning when a weekly cron pattern with fixed time is detected
func (c *Compiler) addWeeklyCronWarning(cronExpr string) {
	// Extract minute, hour, and weekday from the cron expression
	fields := strings.Fields(cronExpr)
	if len(fields) >= 5 {
		minute := fields[0]
		hour := fields[1]
		weekday := fields[4]
		schedulePreprocessingLog.Printf("Warning: detected weekly cron with fixed time: %s", cronExpr)

		// Map weekday number to name for better readability
		weekdayNames := map[string]string{
			"0": "Sunday",
			"1": "Monday",
			"2": "Tuesday",
			"3": "Wednesday",
			"4": "Thursday",
			"5": "Friday",
			"6": "Saturday",
		}
		weekdayName := weekdayNames[weekday]
		if weekdayName == "" {
			weekdayName = "day " + weekday
		}

		// Construct the warning message
		warningMsg := fmt.Sprintf(
			"Schedule uses fixed weekly time (%s %s:%s UTC). Consider using fuzzy schedule 'weekly on %s' instead to distribute workflow execution times and reduce load spikes.",
			weekdayName, hour, minute, strings.ToLower(weekdayName),
		)

		// This warning is added to the warning count
		c.IncrementWarningCount()

		// Store the warning for later display
		c.addScheduleWarning(warningMsg)
	}
}

// addScheduleWarning adds a warning to the compiler's schedule warnings list
func (c *Compiler) addScheduleWarning(warning string) {
	if c.scheduleWarnings == nil {
		c.scheduleWarnings = []string{}
	}
	c.scheduleWarnings = append(c.scheduleWarnings, warning)
}
