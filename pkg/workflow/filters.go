package workflow

import (
	"slices"

	"github.com/github/gh-aw/pkg/logger"
)

var filtersLog = logger.New("workflow:filters")

type labelEventSection struct {
	eventName  string
	eventValue any
}

// applyPullRequestDraftFilter applies draft filter conditions for pull_request triggers
func (c *Compiler) applyPullRequestDraftFilter(data *WorkflowData, frontmatter map[string]any) {
	filtersLog.Print("Applying pull request draft filter")

	prMap, ok := getPullRequestMapFromFrontmatter(data, frontmatter)
	if !ok {
		return
	}

	draftValue, hasDraft := prMap["draft"]
	if !hasDraft {
		return
	}

	draftBool, isDraftBool := draftValue.(bool)
	if !isDraftBool {
		return
	}

	filtersLog.Printf("Found draft filter configuration: draft=%v", draftBool)
	applyConditionToWorkflow(data, buildDraftCondition(draftBool))
}

// applyPullRequestForkFilter applies fork filter conditions for pull_request triggers
// Supports "forks: []string" with glob patterns
// Default behavior: When forks field is not specified, only same-repo PRs are allowed (forks are disallowed by default)
func (c *Compiler) applyPullRequestForkFilter(data *WorkflowData, frontmatter map[string]any) {
	filtersLog.Print("Applying pull request fork filter")

	prMap, ok := getPullRequestMapFromFrontmatter(data, frontmatter)
	if !ok {
		return
	}

	allowedForks, ok := getAllowedForks(prMap)
	if !ok {
		return
	}

	if slices.Contains(allowedForks, "*") {
		filtersLog.Print("Wildcard fork pattern detected, allowing all forks")
		return
	}

	forkCondition := &OrNode{
		Left:  buildNotPullRequestEventCondition(),
		Right: BuildFromAllowedForks(allowedForks),
	}
	applyConditionToWorkflow(data, forkCondition)
}

// applyLabelFilter applies label name filter conditions for labeled/unlabeled triggers
// Supports "names: []string" to filter which label changes trigger the workflow
func (c *Compiler) applyLabelFilter(data *WorkflowData, frontmatter map[string]any) {
	filtersLog.Print("Applying label filter")

	onMap, ok := getOnMapFromFrontmatter(data, frontmatter)
	if !ok {
		return
	}

	var labelConditions []ConditionNode
	for _, section := range getLabelEventSections(onMap) {
		sectionCondition := buildLabelSectionCondition(section)
		if sectionCondition != nil {
			labelConditions = append(labelConditions, sectionCondition)
		}
	}

	if len(labelConditions) == 0 {
		return
	}

	filtersLog.Printf("Applying label name filters: %d conditions found", len(labelConditions))
	applyConditionToWorkflow(data, combineConditionsWithAnd(labelConditions))
}

func getOnMapFromFrontmatter(data *WorkflowData, frontmatter map[string]any) (map[string]any, bool) {
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
	if !isOnMap {
		return nil, false
	}
	return onMap, true
}

func getPullRequestMapFromFrontmatter(data *WorkflowData, frontmatter map[string]any) (map[string]any, bool) {
	onMap, ok := getOnMapFromFrontmatter(data, frontmatter)
	if !ok {
		return nil, false
	}

	prValue, hasPR := onMap["pull_request"]
	if !hasPR {
		return nil, false
	}

	prMap, isPRMap := prValue.(map[string]any)
	if !isPRMap {
		return nil, false
	}
	return prMap, true
}

func applyConditionToWorkflow(data *WorkflowData, condition ConditionNode) {
	existingCondition := data.If
	conditionTree := BuildConditionTree(existingCondition, condition.Render())
	data.If = RenderCondition(conditionTree)
}

func buildNotPullRequestEventCondition() ConditionNode {
	return BuildNotEquals(
		BuildPropertyAccess("github.event_name"),
		BuildStringLiteral("pull_request"),
	)
}

func buildDraftCondition(draftBool bool) ConditionNode {
	return &OrNode{
		Left: buildNotPullRequestEventCondition(),
		Right: BuildEquals(
			BuildPropertyAccess("github.event.pull_request.draft"),
			BuildBooleanLiteral(draftBool),
		),
	}
}

func getAllowedForks(prMap map[string]any) ([]string, bool) {
	forksValue, hasForks := prMap["forks"]
	if !hasForks {
		filtersLog.Print("No forks field specified - applying default fork filter (disallow all forks)")
		return []string{}, true
	}

	filtersLog.Print("Found forks filter configuration")
	return parseForksList(forksValue)
}

func parseForksList(forksValue any) ([]string, bool) {
	if forksStr, isForksStr := forksValue.(string); isForksStr {
		return []string{forksStr}, true
	}

	forksArray, isForksArray := forksValue.([]any)
	if !isForksArray {
		return nil, false
	}

	var allowedForks []string
	for _, fork := range forksArray {
		if forkStr, isForkStr := fork.(string); isForkStr {
			allowedForks = append(allowedForks, forkStr)
		}
	}
	return allowedForks, true
}

func getLabelEventSections(onMap map[string]any) []labelEventSection {
	return []labelEventSection{
		{eventName: "issues", eventValue: onMap["issues"]},
		{eventName: "pull_request", eventValue: onMap["pull_request"]},
		{eventName: "discussion", eventValue: onMap["discussion"]},
	}
}

func buildLabelSectionCondition(section labelEventSection) ConditionNode {
	if section.eventValue == nil {
		return nil
	}

	sectionMap, isSectionMap := section.eventValue.(map[string]any)
	if !isSectionMap || usesNativeLabelFilter(sectionMap, section.eventName) {
		return nil
	}

	hasLabeled, hasUnlabeled := parseLabelTypes(sectionMap)
	if !hasLabeled && !hasUnlabeled {
		return nil
	}

	labelNames, ok := parseLabelNames(sectionMap["names"])
	if !ok || len(labelNames) == 0 {
		return nil
	}

	return buildLabelActionCondition(section.eventName, hasLabeled, hasUnlabeled, buildLabelNameMatch(labelNames))
}

func usesNativeLabelFilter(sectionMap map[string]any, eventName string) bool {
	nativeFilterValue, hasNativeFilter := sectionMap["__gh_aw_native_label_filter__"]
	if !hasNativeFilter {
		return false
	}

	usesNativeFilter, ok := nativeFilterValue.(bool)
	if ok && usesNativeFilter {
		filtersLog.Printf("Skipping label filter for %s: using native GitHub Actions label filtering", eventName)
		return true
	}
	return false
}

func parseLabelTypes(sectionMap map[string]any) (bool, bool) {
	typesValue, hasTypes := sectionMap["types"]
	if !hasTypes {
		return false, false
	}

	typesArray, isTypesArray := typesValue.([]any)
	if !isTypesArray {
		return false, false
	}

	var hasLabeled, hasUnlabeled bool
	for _, t := range typesArray {
		tStr, isTStr := t.(string)
		if !isTStr {
			continue
		}
		if tStr == "labeled" {
			hasLabeled = true
		}
		if tStr == "unlabeled" {
			hasUnlabeled = true
		}
	}
	return hasLabeled, hasUnlabeled
}

func parseLabelNames(namesValue any) ([]string, bool) {
	if namesValue == nil {
		return nil, false
	}
	if namesStr, isNamesStr := namesValue.(string); isNamesStr {
		return []string{namesStr}, true
	}

	namesArray, isNamesArray := namesValue.([]any)
	if !isNamesArray {
		return nil, false
	}

	var labelNames []string
	for _, name := range namesArray {
		if nameStr, isNameStr := name.(string); isNameStr {
			labelNames = append(labelNames, nameStr)
		}
	}
	return labelNames, true
}

func buildLabelNameMatch(labelNames []string) ConditionNode {
	var labelNameConditions []ConditionNode
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
	notThisEvent := BuildNotEquals(
		BuildPropertyAccess("github.event_name"),
		BuildStringLiteral(eventName),
	)
	if hasLabeled && hasUnlabeled {
		return &OrNode{
			Left: notThisEvent,
			Right: &OrNode{
				Left: &AndNode{
					Left:  buildNotActionCondition("labeled"),
					Right: buildNotActionCondition("unlabeled"),
				},
				Right: labelNameMatch,
			},
		}
	}

	action := "unlabeled"
	if hasLabeled {
		action = "labeled"
	}
	return &OrNode{
		Left: notThisEvent,
		Right: &OrNode{
			Left:  buildNotActionCondition(action),
			Right: labelNameMatch,
		},
	}
}

func buildNotActionCondition(action string) ConditionNode {
	return BuildNotEquals(
		BuildPropertyAccess("github.event.action"),
		BuildStringLiteral(action),
	)
}

func combineConditionsWithAnd(conditions []ConditionNode) ConditionNode {
	finalCondition := conditions[0]
	for i := 1; i < len(conditions); i++ {
		finalCondition = &AndNode{
			Left:  finalCondition,
			Right: conditions[i],
		}
	}
	return finalCondition
}
