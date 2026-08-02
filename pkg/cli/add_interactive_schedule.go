package cli

import (
	"fmt"
	"os"
	"strings"

	"charm.land/huh/v2"
	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/parser"
	"github.com/github/gh-aw/pkg/sliceutil"
)

var scheduleWizardLog = logger.New("cli:add_interactive_schedule")

// scheduleFrequencyOption represents a frequency option in the schedule wizard
type scheduleFrequencyOption struct {
	Label      string // Human-readable display label
	Value      string // Internal identifier
	Expression string // Schedule expression to write to frontmatter
}

// standardScheduleFrequencies defines the standard frequency options ordered from most to least frequent
var standardScheduleFrequencies = []scheduleFrequencyOption{
	{Label: "Hourly - runs every hour", Value: "hourly", Expression: "every 1h"},
	{Label: "Every 3 hours", Value: "3-hourly", Expression: "every 3h"},
	{Label: "Daily - runs once per day", Value: "daily", Expression: "daily"},
	{Label: "Weekly - runs once per week", Value: "weekly", Expression: "weekly"},
	{Label: "Monthly - runs on the 1st of each month", Value: "monthly", Expression: "0 0 1 * *"},
}

// scheduleDetection holds the result of detecting schedule info from workflow content.
type scheduleDetection struct {
	RawExpr        string // The original schedule expression (e.g., "daily", "0 9 * * *")
	Frequency      string // Classified frequency ("hourly", "daily", "weekly", etc.)
	IsUpdatable    bool   // Whether the schedule can be updated by the wizard
	IsMultiTrigger bool   // True when on: is a map with triggers besides schedule/workflow_dispatch
	IsOnMap        bool   // True when on: is a map (not a simple scalar string)
}

// detectWorkflowScheduleInfo extracts the schedule expression and classifies its frequency
// from workflow content. Returns a scheduleDetection struct.
//
// Workflows whose "on:" field is a simple string schedule, or a map containing a "schedule"
// key, are considered updatable. For multi-trigger workflows (with triggers beyond
// schedule/workflow_dispatch), IsMultiTrigger is set so the caller knows to update the
// "schedule" sub-field rather than the entire "on:" field.
func detectWorkflowScheduleInfo(content string) scheduleDetection {
	result, err := parser.ExtractFrontmatterFromContent(content)
	if err != nil || result.Frontmatter == nil {
		return scheduleDetection{}
	}
	onValue, exists := result.Frontmatter["on"]
	if !exists {
		return scheduleDetection{}
	}
	if onStr, ok := onValue.(string); ok {
		return detectStringSchedule(onStr)
	}
	onMap, ok := onValue.(map[string]any)
	if !ok {
		return scheduleDetection{}
	}
	return detectMappedSchedule(onMap)
}

func detectStringSchedule(onStr string) scheduleDetection {
	if _, _, err := parser.ParseSchedule(onStr); err != nil {
		return scheduleDetection{}
	}
	return scheduleDetection{RawExpr: onStr, Frequency: classifyScheduleFrequency(onStr), IsUpdatable: true}
}

func detectMappedSchedule(onMap map[string]any) scheduleDetection {
	schedValue, hasSchedule := onMap["schedule"]
	if !hasSchedule {
		return scheduleDetection{}
	}
	isMultiTrigger := scheduleHasMultipleTriggers(onMap)
	if schedStr, ok := schedValue.(string); ok {
		return scheduleDetection{RawExpr: schedStr, Frequency: classifyScheduleFrequency(schedStr), IsUpdatable: true, IsMultiTrigger: isMultiTrigger, IsOnMap: true}
	}
	return detectArraySchedule(schedValue, isMultiTrigger)
}

func scheduleHasMultipleTriggers(onMap map[string]any) bool {
	for key := range onMap {
		if key != "schedule" && key != "workflow_dispatch" {
			scheduleWizardLog.Printf("Multi-trigger on: map detected (trigger '%s')", key)
			return true
		}
	}
	return false
}

func detectArraySchedule(schedValue any, isMultiTrigger bool) scheduleDetection {
	schedArray, ok := schedValue.([]any)
	if !ok || len(schedArray) == 0 {
		return scheduleDetection{}
	}
	if len(schedArray) > 1 {
		scheduleWizardLog.Printf("Multiple cron entries (%d) detected — not updatable", len(schedArray))
		return scheduleDetection{}
	}
	item, ok := schedArray[0].(map[string]any)
	if !ok {
		return scheduleDetection{}
	}
	cronVal, ok := item["cron"].(string)
	if !ok {
		return scheduleDetection{}
	}
	return scheduleDetection{RawExpr: cronVal, Frequency: classifyScheduleFrequency(cronVal), IsUpdatable: true, IsMultiTrigger: isMultiTrigger, IsOnMap: true}
}

// classifyScheduleFrequency determines which standard frequency a schedule expression represents.
// classifyScheduleFrequency determines which standard frequency a schedule expression represents.
// Returns one of: "hourly", "3-hourly", "daily", "weekly", "monthly", or "custom".
func classifyScheduleFrequency(scheduleStr string) string {
	normalized := strings.ToLower(strings.TrimSpace(scheduleStr))
	if frequency, ok := classifyFriendlyScheduleFrequency(normalized); ok {
		return frequency
	}
	if frequency, ok := classifyFuzzyScheduleFrequency(normalized); ok {
		return frequency
	}
	if frequency, ok := classifyCronScheduleFrequency(scheduleStr); ok {
		return frequency
	}
	return "custom"
}

func classifyFriendlyScheduleFrequency(normalized string) (string, bool) {
	switch normalized {
	case "hourly", "every 1h", "every 1 hour", "every 1 hours":
		return "hourly", true
	case "every 3h", "every 3 hours":
		return "3-hourly", true
	case "daily":
		return "daily", true
	case "weekly":
		return "weekly", true
	default:
		return "", false
	}
}

func classifyFuzzyScheduleFrequency(normalized string) (string, bool) {
	switch {
	case strings.HasPrefix(normalized, "fuzzy:hourly/1 ") || normalized == "fuzzy:hourly/1":
		return "hourly", true
	case strings.HasPrefix(normalized, "fuzzy:hourly/3 ") || normalized == "fuzzy:hourly/3":
		return "3-hourly", true
	case strings.HasPrefix(normalized, "fuzzy:daily"):
		return "daily", true
	case strings.HasPrefix(normalized, "fuzzy:weekly"):
		return "weekly", true
	default:
		return "", false
	}
}

func classifyCronScheduleFrequency(scheduleStr string) (string, bool) {
	if parser.IsHourlyCron(scheduleStr) {
		return classifyHourlyCronFrequency(scheduleStr), true
	}
	if parser.IsDailyCron(scheduleStr) {
		return "daily", true
	}
	if parser.IsWeeklyCron(scheduleStr) {
		return "weekly", true
	}
	if isMonthlyCron(scheduleStr) {
		return "monthly", true
	}
	return "", false
}

func classifyHourlyCronFrequency(scheduleStr string) string {
	fields := strings.Fields(scheduleStr)
	if len(fields) == 5 {
		interval := strings.TrimPrefix(fields[1], "*/")
		if interval == "1" {
			return "hourly"
		}
		if interval == "3" {
			return "3-hourly"
		}
	}
	return "custom"
}

func isMonthlyCron(scheduleStr string) bool {
	fields := strings.Fields(scheduleStr)
	if len(fields) != 5 || fields[3] != "*" || fields[4] != "*" {
		return false
	}
	day := fields[2]
	return day != "*" && !strings.ContainsAny(day, "*/-,")
}

// selectScheduleFrequency presents a schedule-frequency selection form to the user when the
// selectScheduleFrequency presents a schedule-frequency selection form to the user when the
// workflow being added has a schedule trigger. If the user picks a different frequency the
// resolved workflow content is updated in memory so the change is reflected in the PR.
func (c *AddInteractiveConfig) selectScheduleFrequency() error {
	if c.resolvedWorkflows == nil || len(c.resolvedWorkflows.Workflows) == 0 {
		return nil
	}
	for _, wf := range c.resolvedWorkflows.Workflows {
		if err := c.selectWorkflowScheduleFrequency(wf); err != nil {
			return err
		}
	}
	return nil
}

func (c *AddInteractiveConfig) selectWorkflowScheduleFrequency(wf *ResolvedWorkflow) error {
	content := string(wf.Content)
	detection := detectWorkflowScheduleInfo(content)
	if !detection.IsUpdatable {
		return nil
	}
	scheduleWizardLog.Printf("Detected schedule: expr=%q, freq=%s, multiTrigger=%v", detection.RawExpr, detection.Frequency, detection.IsMultiTrigger)
	selected, err := c.promptForScheduleFrequency(detection)
	if err != nil || selected == "custom" || selected == detection.Frequency {
		if err == nil && (selected == "custom" || selected == detection.Frequency) {
			scheduleWizardLog.Printf("Schedule unchanged: keeping %q", detection.RawExpr)
		}
		return err
	}
	return updateResolvedWorkflowSchedule(wf, content, detection, selected)
}

func (c *AddInteractiveConfig) promptForScheduleFrequency(detection scheduleDetection) (string, error) {
	options := buildScheduleOptions(detection.RawExpr, detection.Frequency)
	var selected string
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage("This workflow runs on a schedule."))
	form := console.NewSelectForm(huh.NewSelect[string]().Title("How often should this workflow run?").Description("Current schedule: " + detection.RawExpr).Options(options...).Value(&selected))
	if err := form.RunWithContext(c.Ctx); err != nil {
		return "", fmt.Errorf("failed to select schedule frequency: %w", err)
	}
	scheduleWizardLog.Printf("User selected frequency: %s", selected)
	return selected, nil
}

func updateResolvedWorkflowSchedule(wf *ResolvedWorkflow, content string, detection scheduleDetection, selected string) error {
	newExpr := scheduleExpressionForFrequency(selected)
	if newExpr == "" {
		return nil
	}
	updatedContent, err := rewriteWorkflowSchedule(content, detection, newExpr)
	if err != nil {
		scheduleWizardLog.Printf("Failed to update schedule (isOnMap=%v): %v", detection.IsOnMap, err)
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Could not update schedule: %v", err)))
		return nil
	}
	wf.Content = []byte(updatedContent)
	if wf.SourceInfo != nil {
		wf.SourceInfo.Content = []byte(updatedContent)
	}
	fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Schedule updated to: "+selected))
	return nil
}

func scheduleExpressionForFrequency(selected string) string {
	for _, opt := range standardScheduleFrequencies {
		if opt.Value == selected {
			return opt.Expression
		}
	}
	return ""
}

func rewriteWorkflowSchedule(content string, detection scheduleDetection, newExpr string) (string, error) {
	if detection.IsOnMap {
		return UpdateScheduleInOnBlock(content, newExpr)
	}
	return UpdateFieldInFrontmatter(content, "on", newExpr)
}

// buildScheduleOptions constructs the huh option list for the schedule frequency form.
// buildScheduleOptions constructs the huh option list for the schedule frequency form.
// The default option (matching the current frequency) is placed first.
func buildScheduleOptions(rawExpr, currentFreq string) []huh.Option[string] {
	options := make([]huh.Option[string], 0, len(standardScheduleFrequencies)+1)

	// If the current schedule doesn't match any standard frequency, add a "Custom" entry
	if currentFreq == "custom" {
		label := fmt.Sprintf("Custom: %s (keep existing)", rawExpr)
		options = append(options, huh.NewOption(label, "custom"))
	}

	// Standard frequency options — mark the one matching the current schedule
	for _, f := range standardScheduleFrequencies {
		label := f.Label
		if f.Value == currentFreq {
			label += " (current)"
		}
		options = append(options, huh.NewOption(label, f.Value))
	}

	// Move the default option to the front so huh selects it initially.
	// classifyScheduleFrequency always returns a non-empty string ("custom" as its last resort),
	// so currentFreq is never empty when this function is called from selectScheduleFrequency.
	reordered := sliceutil.Filter(options, func(opt huh.Option[string]) bool { return opt.Value == currentFreq })
	rest := sliceutil.Filter(options, func(opt huh.Option[string]) bool { return opt.Value != currentFreq })
	return append(reordered, rest...)
}
