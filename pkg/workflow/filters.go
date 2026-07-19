package workflow

import (
	"slices"

	"github.com/github/gh-aw/pkg/logger"
)

var filtersLog = logger.New("workflow:filters")

// applyPullRequestDraftFilter applies draft filter conditions for pull_request triggers
func (c *Compiler) applyPullRequestDraftFilter(data *WorkflowData, frontmatter map[string]any) {
	filtersLog.Print("Applying pull request draft filter")

	prMap, ok := getPullRequestFilterMap(data, frontmatter)
	if !ok {
		return
	}

	// Check if draft is specified
	draftValue, hasDraft := prMap["draft"]
	if !hasDraft {
		return
	}

	// Check if draft is a boolean
	draftBool, isDraftBool := draftValue.(bool)
	if !isDraftBool {
		// If draft is not a boolean, don't add filter
		return
	}

	filtersLog.Printf("Found draft filter configuration: draft=%v", draftBool)
	applyConditionNode(data, buildDraftCondition(draftBool))
}

// applyPullRequestForkFilter applies fork filter conditions for pull_request triggers
// Supports "forks: []string" with glob patterns
// Default behavior: When forks field is not specified, only same-repo PRs are allowed (forks are disallowed by default)
func (c *Compiler) applyPullRequestForkFilter(data *WorkflowData, frontmatter map[string]any) {
	filtersLog.Print("Applying pull request fork filter")

	prMap, ok := getPullRequestFilterMap(data, frontmatter)
	if !ok {
		return
	}

	allowedForks, ok := parseAllowedForks(prMap)
	if !ok {
		return
	}

	// If "*" wildcard is present, skip fork filtering (allow all forks)
	if slices.Contains(allowedForks, "*") {
		filtersLog.Print("Wildcard fork pattern detected, allowing all forks")
		return // No fork filtering needed
	}

	applyConditionNode(data, buildForkCondition(allowedForks))
}

// applyLabelFilter applies label name filter conditions for labeled/unlabeled triggers
// Supports "names: []string" to filter which label changes trigger the workflow
func (c *Compiler) applyLabelFilter(data *WorkflowData, frontmatter map[string]any) {
	filtersLog.Print("Applying label filter")

	onMap, ok := getOnFilterMap(data, frontmatter)
	if !ok {
		return
	}

	labelConditions := collectLabelConditions(onMap)

	// If we have label conditions, combine them and apply to the workflow
	if len(labelConditions) > 0 {
		filtersLog.Printf("Applying label name filters: %d conditions found", len(labelConditions))
		applyConditionNode(data, combineConditionsWithAnd(labelConditions))
	}
}

func getOnFilterMap(data *WorkflowData, frontmatter map[string]any) (map[string]any, bool) {
	var onValue any
	var hasOn bool
	if data.ParsedFrontmatter != nil && data.ParsedFrontmatter.On != nil {
		onValue = data.ParsedFrontmatter.On
		hasOn = true
	} else {
		onValue, hasOn = frontmatter["on"]
	}
	if !hasOn {
		return nil, false
	}
	onMap, isOnMap := onValue.(map[string]any)
	return onMap, isOnMap
}

func getPullRequestFilterMap(data *WorkflowData, frontmatter map[string]any) (map[string]any, bool) {
	onMap, ok := getOnFilterMap(data, frontmatter)
	if !ok {
		return nil, false
	}
	prValue, hasPR := onMap["pull_request"]
	if !hasPR {
		return nil, false
	}
	prMap, isPRMap := prValue.(map[string]any)
	return prMap, isPRMap
}

func buildDraftCondition(draftBool bool) ConditionNode {
	notPullRequestEvent := BuildNotEquals(
		BuildPropertyAccess("github.event_name"),
		BuildStringLiteral("pull_request"),
	)
	draftCheck := BuildEquals(
		BuildPropertyAccess("github.event.pull_request.draft"),
		BuildBooleanLiteral(draftBool),
	)
	return &OrNode{
		Left:  notPullRequestEvent,
		Right: draftCheck,
	}
}

func parseAllowedForks(prMap map[string]any) ([]string, bool) {
	forksValue, hasForks := prMap["forks"]
	if !hasForks {
		filtersLog.Print("No forks field specified - applying default fork filter (disallow all forks)")
		return []string{}, true
	}
	filtersLog.Print("Found forks filter configuration")
	if forksStr, isForksStr := forksValue.(string); isForksStr {
		return []string{forksStr}, true
	}
	if forksArray, isForksArray := forksValue.([]any); isForksArray {
		allowedForks := make([]string, 0, len(forksArray))
		for _, fork := range forksArray {
			if forkStr, isForkStr := fork.(string); isForkStr {
				allowedForks = append(allowedForks, forkStr)
			}
		}
		return allowedForks, true
	}
	return nil, false
}

func buildForkCondition(allowedForks []string) ConditionNode {
	return &OrNode{
		Left: BuildNotEquals(
			BuildPropertyAccess("github.event_name"),
			BuildStringLiteral("pull_request"),
		),
		Right: BuildFromAllowedForks(allowedForks),
	}
}

func applyConditionNode(data *WorkflowData, condition ConditionNode) {
	conditionTree := BuildConditionTree(data.If, condition.Render())
	data.If = RenderCondition(conditionTree)
}

type labelFilterEventSection struct {
	eventName  string
	eventValue any
}

func collectLabelConditions(onMap map[string]any) []ConditionNode {
	eventSections := []labelFilterEventSection{
		{"issues", onMap["issues"]},
		{"pull_request", onMap["pull_request"]},
		{"discussion", onMap["discussion"]},
	}
	var labelConditions []ConditionNode
	for _, section := range eventSections {
		condition := buildLabelConditionForSection(section)
		if condition != nil {
			labelConditions = append(labelConditions, condition)
		}
	}
	return labelConditions
}

func buildLabelConditionForSection(section labelFilterEventSection) ConditionNode {
	if section.eventValue == nil {
		return nil
	}
	sectionMap, isSectionMap := section.eventValue.(map[string]any)
	if !isSectionMap {
		return nil
	}
	hasLabeled, hasUnlabeled := labelEventTypes(sectionMap["types"])
	if !hasLabeled && !hasUnlabeled {
		return nil
	}
	if usesNativeLabelFilter(sectionMap) {
		filtersLog.Printf("Skipping label filter for %s: using native GitHub Actions label filtering", section.eventName)
		return nil
	}
	labelNames, ok := parseStringOrStringArray(sectionMap["names"])
	if !ok || len(labelNames) == 0 {
		return nil
	}
	return buildLabelActionCondition(section.eventName, hasLabeled, hasUnlabeled, buildLabelNameMatch(labelNames))
}

func labelEventTypes(typesValue any) (bool, bool) {
	var hasLabeled bool
	var hasUnlabeled bool
	if typesArray, isTypesArray := typesValue.([]any); isTypesArray {
		for _, t := range typesArray {
			tStr, isTStr := t.(string)
			hasLabeled = hasLabeled || (isTStr && tStr == "labeled")
			hasUnlabeled = hasUnlabeled || (isTStr && tStr == "unlabeled")
		}
	}
	return hasLabeled, hasUnlabeled
}

func usesNativeLabelFilter(sectionMap map[string]any) bool {
	nativeFilterValue, hasNativeFilter := sectionMap["__gh_aw_native_label_filter__"]
	usesNativeFilter, ok := nativeFilterValue.(bool)
	return hasNativeFilter && ok && usesNativeFilter
}

func parseStringOrStringArray(value any) ([]string, bool) {
	if str, ok := value.(string); ok {
		return []string{str}, true
	}
	array, ok := value.([]any)
	if !ok {
		return nil, false
	}
	values := make([]string, 0, len(array))
	for _, item := range array {
		if itemStr, isString := item.(string); isString {
			values = append(values, itemStr)
		}
	}
	return values, true
}

func buildLabelNameMatch(labelNames []string) ConditionNode {
	labelNameConditions := make([]ConditionNode, 0, len(labelNames))
	for _, labelName := range labelNames {
		labelNameConditions = append(labelNameConditions, BuildEquals(
			BuildPropertyAccess("github.event.label.name"),
			BuildStringLiteral(labelName),
		))
	}
	if len(labelNameConditions) == 1 {
		return labelNameConditions[0]
	}
	return &DisjunctionNode{Terms: labelNameConditions}
}

func buildLabelActionCondition(eventName string, hasLabeled, hasUnlabeled bool, labelNameMatch ConditionNode) ConditionNode {
	notThisEvent := BuildNotEquals(BuildPropertyAccess("github.event_name"), BuildStringLiteral(eventName))
	if hasLabeled && hasUnlabeled {
		notLabelAction := &AndNode{
			Left:  BuildNotEquals(BuildPropertyAccess("github.event.action"), BuildStringLiteral("labeled")),
			Right: BuildNotEquals(BuildPropertyAccess("github.event.action"), BuildStringLiteral("unlabeled")),
		}
		return &OrNode{Left: notThisEvent, Right: &OrNode{Left: notLabelAction, Right: labelNameMatch}}
	}
	if hasLabeled {
		notLabeledAction := BuildNotEquals(BuildPropertyAccess("github.event.action"), BuildStringLiteral("labeled"))
		return &OrNode{Left: notThisEvent, Right: &OrNode{Left: notLabeledAction, Right: labelNameMatch}}
	}
	notUnlabeledAction := BuildNotEquals(BuildPropertyAccess("github.event.action"), BuildStringLiteral("unlabeled"))
	return &OrNode{Left: notThisEvent, Right: &OrNode{Left: notUnlabeledAction, Right: labelNameMatch}}
}

func combineConditionsWithAnd(conditions []ConditionNode) ConditionNode {
	if len(conditions) == 1 {
		return conditions[0]
	}
	finalCondition := conditions[0]
	for i := 1; i < len(conditions); i++ {
		finalCondition = &AndNode{
			Left:  finalCondition,
			Right: conditions[i],
		}
	}
	return finalCondition
}
