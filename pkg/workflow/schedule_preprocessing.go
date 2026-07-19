package workflow

import (
	"errors"
	"fmt"
	"strings"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/parser"
)

var schedulePreprocessingLog = logger.New("workflow:schedule_preprocessing")

// normalizeScheduleString handles the common schedule string parsing, warning emission,
// fuzzy scattering, and validation logic. It returns the normalized cron expression
// and the original friendly format, or an error if validation fails.
func (c *Compiler) normalizeScheduleString(scheduleStr string, itemIndex int) (parsedCron string, friendlyFormat string, err error) {
	parsedCron, original, err := parser.ParseSchedule(scheduleStr)
	if err != nil {
		if itemIndex >= 0 {
			return "", "", fmt.Errorf("invalid schedule expression in item %d: %w", itemIndex, err)
		}
		return "", "", err
	}

	c.addScheduleWarningsForCron(parsedCron)
	parsedCron, original = c.scatterFuzzySchedule(parsedCron, original)
	if err := validateNormalizedSchedule(parsedCron, itemIndex); err != nil {
		return "", "", err
	}
	return parsedCron, original, nil
}

func (c *Compiler) addScheduleWarningsForCron(parsedCron string) {
	if parser.IsDailyCron(parsedCron) && !parser.IsFuzzyCron(parsedCron) {
		c.addDailyCronWarning(parsedCron)
	}
	if parser.IsHourlyCron(parsedCron) && !parser.IsFuzzyCron(parsedCron) {
		c.addHourlyCronWarning(parsedCron)
	}
	if parser.IsWeeklyCron(parsedCron) && !parser.IsFuzzyCron(parsedCron) {
		c.addWeeklyCronWarning(parsedCron)
	}
}

func (c *Compiler) scatterFuzzySchedule(parsedCron, original string) (string, string) {
	if !parser.IsFuzzyCron(parsedCron) || c.workflowIdentifier == "" {
		return parsedCron, original
	}
	seed := c.fuzzyScheduleSeed()
	scatteredCron, err := parser.ScatterSchedule(parsedCron, seed)
	if err != nil {
		schedulePreprocessingLog.Printf("Warning: failed to scatter fuzzy schedule: %v", err)
		return parsedCron, original
	}
	schedulePreprocessingLog.Printf("Scattered fuzzy schedule %s to %s for workflow %s", parsedCron, scatteredCron, c.workflowIdentifier)
	if original != "" {
		original = original + " (scattered)"
	}
	return scatteredCron, original
}

func (c *Compiler) fuzzyScheduleSeed() string {
	seed := c.workflowIdentifier
	if IsRelease() {
		if c.repositorySlug != "" {
			return c.repositorySlug + "/" + c.workflowIdentifier
		}
		schedulePreprocessingLog.Printf("Warning: repository slug not available for fuzzy schedule scattering")
		c.IncrementWarningCount()
		c.addScheduleWarning("Fuzzy schedule scattering without repository context. Workflows with the same name in different repositories may collide. Ensure you are in a git repository with a configured remote.")
		return seed
	}
	seed = "dev/" + c.workflowIdentifier
	schedulePreprocessingLog.Printf("Using dev mode seed for fuzzy schedule scattering: %s", seed)
	return seed
}

func validateNormalizedSchedule(parsedCron string, itemIndex int) error {
	if parser.IsFuzzyCron(parsedCron) {
		if itemIndex >= 0 {
			return fmt.Errorf("fuzzy cron expression '%s' in item %d must be scattered to proper cron format before compilation (missing workflow identifier: ensure the workflow identifier is set)", parsedCron, itemIndex)
		}
		return fmt.Errorf("fuzzy cron expression '%s' must be scattered to proper cron format before compilation (missing workflow identifier: ensure the workflow identifier is set)", parsedCron)
	}
	if !parser.IsCronExpression(parsedCron) {
		if itemIndex >= 0 {
			return fmt.Errorf("invalid cron expression '%s' in item %d: must have exactly 5 fields (minute hour day-of-month month day-of-week)", parsedCron, itemIndex)
		}
		return fmt.Errorf("invalid cron expression '%s': must have exactly 5 fields (minute hour day-of-month month day-of-week)", parsedCron)
	}
	return nil
}

// preprocessScheduleFields converts human-friendly schedule expressions to cron expressions
// in the frontmatter's "on" section. It modifies the frontmatter map in place.
func (c *Compiler) preprocessScheduleFields(frontmatter map[string]any, markdownPath string, content string) error {
	schedulePreprocessingLog.Print("Preprocessing schedule fields in frontmatter")

	onValue, exists := frontmatter["on"]
	if !exists {
		return nil
	}
	if onStr, ok := onValue.(string); ok {
		return c.preprocessOnString(frontmatter, markdownPath, content, onStr)
	}
	return c.preprocessOnMap(onValue)
}

func (c *Compiler) preprocessOnString(frontmatter map[string]any, markdownPath, content, onStr string) error {
	schedulePreprocessingLog.Printf("Processing on field as string: %s", onStr)
	if handled, err := c.preprocessCommandOrLabelTrigger(frontmatter, onStr); handled || err != nil {
		return err
	}
	triggerIR, err := ParseTriggerShorthand(onStr)
	if err != nil {
		return c.createTriggerParseError(markdownPath, content, onStr, err)
	}
	if triggerIR != nil {
		c.applyTriggerIR(frontmatter, onStr, triggerIR)
		return nil
	}
	return c.preprocessScheduleStringTrigger(frontmatter, onStr)
}

func (c *Compiler) preprocessCommandOrLabelTrigger(frontmatter map[string]any, onStr string) (bool, error) {
	commandName, isSlashCommand, err := parseSlashCommandShorthand(onStr)
	if err != nil {
		return false, err
	}
	if isSlashCommand {
		schedulePreprocessingLog.Printf("Converting shorthand 'on: %s' to slash_command + workflow_dispatch", onStr)
		frontmatter["on"] = expandSlashCommandShorthand(commandName)
		return true, nil
	}
	if labelName, ok := strings.CutPrefix(onStr, "label-command "); ok {
		labelName = strings.TrimSpace(labelName)
		if labelName == "" {
			return false, errors.New("label-command shorthand requires a label name after 'label-command'")
		}
		schedulePreprocessingLog.Printf("Converting shorthand 'on: %s' to label_command + workflow_dispatch", onStr)
		frontmatter["on"] = expandLabelCommandShorthand(labelName)
		return true, nil
	}
	entityType, labelNames, isLabelTrigger, err := parseLabelTriggerShorthand(onStr)
	if err != nil {
		return false, err
	}
	if isLabelTrigger {
		schedulePreprocessingLog.Printf("Converting shorthand 'on: %s' to %s labeled + workflow_dispatch", onStr, entityType)
		frontmatter["on"] = expandLabelTriggerShorthand(entityType, labelNames)
		return true, nil
	}
	return false, nil
}

func (c *Compiler) applyTriggerIR(frontmatter map[string]any, onStr string, triggerIR *TriggerIR) {
	schedulePreprocessingLog.Printf("Converting shorthand 'on: %s' to structured trigger", onStr)
	frontmatter["on"] = triggerIR.ToYAMLMap()
	if len(triggerIR.Conditions) == 0 {
		return
	}
	condition := strings.Join(triggerIR.Conditions, " && ")
	schedulePreprocessingLog.Printf("Setting if condition from trigger shorthand: %s", condition)
	if existing, ok := frontmatter["if"].(string); ok && existing != "" {
		existing = stripExpressionWrapper(existing)
		frontmatter["if"] = "(" + existing + ") && (" + condition + ")"
	} else {
		frontmatter["if"] = condition
	}
}

func (c *Compiler) preprocessScheduleStringTrigger(frontmatter map[string]any, onStr string) error {
	parsedCron, original, err := c.normalizeScheduleString(onStr, -1)
	if err != nil {
		if errors.Is(err, parser.ErrUnsupportedSyntax) {
			return err
		}
		schedulePreprocessingLog.Printf("Not a recognized shorthand or schedule: %s - leaving as-is", onStr)
		return nil
	}
	schedulePreprocessingLog.Printf("Converting shorthand 'on: %s' to schedule + workflow_dispatch", onStr)
	frontmatter["on"] = map[string]any{
		"schedule": []any{
			map[string]any{
				"cron": parsedCron,
			},
		},
		"workflow_dispatch": nil,
	}
	c.storeFriendlyScheduleFormat(0, original)
	return nil
}

func (c *Compiler) preprocessOnMap(onValue any) error {
	onMap, ok := onValue.(map[string]any)
	if !ok {
		return nil
	}
	scheduleValue, hasSchedule := onMap["schedule"]
	if !hasSchedule {
		return nil
	}
	if scheduleStr, ok := scheduleValue.(string); ok {
		return c.preprocessScheduleStringField(onMap, scheduleStr)
	}
	scheduleArray, ok := scheduleValue.([]any)
	if !ok {
		return errors.New("schedule field must be a string or an array")
	}
	return c.preprocessScheduleArray(onMap, scheduleArray)
}

func (c *Compiler) preprocessScheduleStringField(onMap map[string]any, scheduleStr string) error {
	schedulePreprocessingLog.Printf("Converting shorthand schedule string to array format: %s", scheduleStr)
	parsedCron, original, err := c.normalizeScheduleString(scheduleStr, -1)
	if err != nil {
		return fmt.Errorf("invalid schedule expression: %w", err)
	}
	onMap["schedule"] = []any{
		map[string]any{
			"cron": parsedCron,
		},
	}
	c.storeFriendlyScheduleFormat(0, original)
	ensureWorkflowDispatch(onMap)
	return nil
}

func (c *Compiler) preprocessScheduleArray(onMap map[string]any, scheduleArray []any) error {
	if c.scheduleFriendlyFormats == nil {
		c.scheduleFriendlyFormats = make(map[int]string)
	}
	schedulePreprocessingLog.Printf("Processing %d schedule items", len(scheduleArray))
	for i, item := range scheduleArray {
		if err := c.preprocessScheduleItem(item, i); err != nil {
			return err
		}
	}
	ensureWorkflowDispatch(onMap)
	return nil
}

func (c *Compiler) preprocessScheduleItem(item any, i int) error {
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
	if tzValue, hasTimezone := itemMap["timezone"]; hasTimezone {
		if _, ok := tzValue.(string); !ok {
			return fmt.Errorf("schedule item %d 'timezone' field must be a string (IANA timezone, e.g. \"America/New_York\")", i)
		}
	}
	parsedCron, original, err := c.normalizeScheduleString(cronStr, i)
	if err != nil {
		return err
	}
	itemMap["cron"] = parsedCron
	c.storeFriendlyScheduleFormat(i, original)
	return nil
}

func (c *Compiler) storeFriendlyScheduleFormat(index int, original string) {
	if original == "" {
		return
	}
	if c.scheduleFriendlyFormats == nil {
		c.scheduleFriendlyFormats = make(map[int]string)
	}
	c.scheduleFriendlyFormats[index] = original
}

func ensureWorkflowDispatch(onMap map[string]any) {
	if _, hasWorkflowDispatch := onMap["workflow_dispatch"]; !hasWorkflowDispatch {
		schedulePreprocessingLog.Printf("Adding workflow_dispatch to scheduled workflow")
		onMap["workflow_dispatch"] = nil
	}
}

// createTriggerParseError creates a detailed error for trigger parsing issues with source location
func (c *Compiler) createTriggerParseError(filePath, content, triggerStr string, err error) error {
	schedulePreprocessingLog.Printf("Creating trigger parse error for: %s", triggerStr)

	lines := strings.Split(content, "\n")
	onLine, onColumn := findFrontmatterOnField(lines)
	if onLine > 0 {
		return formatTriggerParseError(filePath, lines, onLine, onColumn, err)
	}
	schedulePreprocessingLog.Printf("Could not find 'on:' line in frontmatter, using fallback error")
	return fmt.Errorf("trigger syntax error: %w", err)
}

func findFrontmatterOnField(lines []string) (int, int) {
	inFrontmatter := false
	for i, line := range lines {
		if strings.TrimSpace(line) == "---" {
			if !inFrontmatter {
				inFrontmatter = true
			} else {
				break
			}
			continue
		}
		if inFrontmatter && strings.HasPrefix(strings.TrimSpace(line), "on:") {
			return i + 1, strings.Index(line, "on:") + 1
		}
	}
	return 0, 0
}

func formatTriggerParseError(filePath string, lines []string, onLine, onColumn int, err error) error {
	compilerErr := console.CompilerError{
		Position: console.ErrorPosition{
			File:   filePath,
			Line:   onLine,
			Column: onColumn,
		},
		Type:    "error",
		Message: "trigger syntax error: " + err.Error(),
		Context: triggerErrorContext(lines, onLine),
	}
	return errors.New(console.FormatError(compilerErr))
}

func triggerErrorContext(lines []string, onLine int) []string {
	var context []string
	startLine := max(1, onLine-2)
	endLine := min(len(lines), onLine+2)
	for i := startLine; i <= endLine; i++ {
		if i-1 < len(lines) {
			context = append(context, lines[i-1])
		}
	}
	return context
}

// addFriendlyScheduleComments adds comments showing the original friendly format for schedule cron expressions
// This function is called after the YAML has been generated from the frontmatter
func (c *Compiler) addFriendlyScheduleComments(yamlStr string, frontmatter map[string]any) string {
	// Retrieve the friendly formats for this compilation
	if len(c.scheduleFriendlyFormats) == 0 {
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

			// Add friendly format comment inline on the same line as the cron expression.
			// Placing it on a separate line triggers yamllint's comments-indentation rule
			// because the comment indentation would differ from the next non-comment line.
			if friendly, exists := c.scheduleFriendlyFormats[scheduleItemIndex]; exists {
				line = line + "  # Friendly format: " + friendly
			}
			result = append(result, line)
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

		c.emitScheduleWarning(warningMsg)
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

		c.emitScheduleWarning(warningMsg)
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

		c.emitScheduleWarning(warningMsg)
	}
}

func (c *Compiler) emitScheduleWarning(warning string) {
	// This warning is added to the warning count.
	// It will be collected and displayed by the compilation process.
	c.IncrementWarningCount()

	// Store the warning for later display.
	c.addScheduleWarning(warning)
}

// addScheduleWarning adds a warning to the compiler's schedule warnings list
func (c *Compiler) addScheduleWarning(warning string) {
	if c.scheduleWarnings == nil {
		c.scheduleWarnings = []string{}
	}
	c.scheduleWarnings = append(c.scheduleWarnings, warning)
}
