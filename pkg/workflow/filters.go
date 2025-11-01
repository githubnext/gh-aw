package workflow

import "github.com/githubnext/gh-aw/pkg/logger"

var filtersLog = logger.New("workflow:filters")

// applyPullRequestDraftFilter applies draft filter conditions for pull_request triggers
func (c *Compiler) applyPullRequestDraftFilter(data *WorkflowData, frontmatter map[string]any) {
	filtersLog.Print("Applying pull request draft filter")

	// Check if there's an "on" section in the frontmatter
	onValue, hasOn := frontmatter["on"]
	if !hasOn {
		return
	}

	// Check if "on" is an object (not a string)
	onMap, isOnMap := onValue.(map[string]any)
	if !isOnMap {
		return
	}

	// Check if there's a pull_request section
	prValue, hasPR := onMap["pull_request"]
	if !hasPR {
		return
	}

	// Check if pull_request is an object with draft settings
	prMap, isPRMap := prValue.(map[string]any)
	if !isPRMap {
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

	// Generate conditional logic based on draft value using expression nodes
	var draftCondition ConditionNode
	if draftBool {
		// draft: true - include only draft PRs
		// The condition should be true for non-pull_request events or for draft pull_requests
		notPullRequestEvent := BuildNotEquals(
			BuildPropertyAccess("github.event_name"),
			BuildStringLiteral("pull_request"),
		)
		isDraftPR := BuildEquals(
			BuildPropertyAccess("github.event.pull_request.draft"),
			BuildBooleanLiteral(true),
		)
		draftCondition = &OrNode{
			Left:  notPullRequestEvent,
			Right: isDraftPR,
		}
	} else {
		// draft: false - exclude draft PRs
		// The condition should be true for non-pull_request events or for non-draft pull_requests
		notPullRequestEvent := BuildNotEquals(
			BuildPropertyAccess("github.event_name"),
			BuildStringLiteral("pull_request"),
		)
		isNotDraftPR := BuildEquals(
			BuildPropertyAccess("github.event.pull_request.draft"),
			BuildBooleanLiteral(false),
		)
		draftCondition = &OrNode{
			Left:  notPullRequestEvent,
			Right: isNotDraftPR,
		}
	}

	// Build condition tree and render
	existingCondition := data.If
	conditionTree := buildConditionTree(existingCondition, draftCondition.Render())
	data.If = conditionTree.Render()
}

// Note: applyPullRequestForkFilter has been removed as the forks field is now a boolean
// that controls fork prevention via applyPullRequestForkPrevention, not a filter condition with patterns.

// applyPullRequestForkPrevention adds fork prevention to the workflow-level if condition
// This ensures workflows don't execute from forked PRs unless explicitly allowed with forks: true
func (c *Compiler) applyPullRequestForkPrevention(data *WorkflowData, frontmatter map[string]any) {
	filtersLog.Print("Applying pull request fork prevention")

	// Check if this workflow has PR-related triggers
	if !c.hasPullRequestTrigger(data.On) {
		filtersLog.Print("No pull request trigger found, skipping fork prevention")
		return
	}

	// Check if forks are allowed from frontmatter
	allowForks := c.getAllowForksFromFrontmatter(frontmatter)
	if allowForks {
		filtersLog.Print("Forks explicitly allowed in frontmatter, skipping fork prevention")
		return
	}

	filtersLog.Print("Adding fork prevention condition")

	// Build fork prevention condition that only applies to PR events
	// For non-PR events, the condition is always true
	// For PR events, check that the fork is from the same repository

	// List of PR-related event names
	prEvents := []string{"pull_request", "pull_request_target", "pull_request_review", "pull_request_review_comment"}

	// Build condition: (event is not PR-related) OR (PR is from same repo)
	var eventConditions []ConditionNode
	for _, eventName := range prEvents {
		eventConditions = append(eventConditions, BuildNotEquals(
			BuildPropertyAccess("github.event_name"),
			BuildStringLiteral(eventName),
		))
	}

	// Combine all "not PR event" conditions with AND
	// This creates: (event != pull_request) && (event != pull_request_target) && ...
	notPREvent := eventConditions[0]
	for i := 1; i < len(eventConditions); i++ {
		notPREvent = &AndNode{
			Left:  notPREvent,
			Right: eventConditions[i],
		}
	}

	// Fork check: PR is from the same repository
	forkCheck := BuildNotFromFork()

	// Final condition: (not a PR event) OR (PR is from same repo)
	forkPreventionCondition := &OrNode{
		Left:  notPREvent,
		Right: forkCheck,
	}

	// Build condition tree and render
	existingCondition := data.If
	conditionTree := buildConditionTree(existingCondition, forkPreventionCondition.Render())
	data.If = conditionTree.Render()
}

// applyLabelFilter applies label name filter conditions for labeled/unlabeled triggers
// Supports "names: []string" to filter which label changes trigger the workflow
func (c *Compiler) applyLabelFilter(data *WorkflowData, frontmatter map[string]any) {
	filtersLog.Print("Applying label filter")

	// Check if there's an "on" section in the frontmatter
	onValue, hasOn := frontmatter["on"]
	if !hasOn {
		return
	}

	// Check if "on" is an object (not a string)
	onMap, isOnMap := onValue.(map[string]any)
	if !isOnMap {
		return
	}

	// Check both issues and pull_request sections for labeled/unlabeled with names
	eventSections := []struct {
		eventName    string
		eventValue   any
		eventNameStr string // For condition checks
	}{
		{"issues", onMap["issues"], "issues"},
		{"pull_request", onMap["pull_request"], "pull_request"},
	}

	var labelConditions []ConditionNode

	for _, section := range eventSections {
		if section.eventValue == nil {
			continue
		}

		// Check if the section is an object with types and names
		sectionMap, isSectionMap := section.eventValue.(map[string]any)
		if !isSectionMap {
			continue
		}

		// Check for "types" field
		typesValue, hasTypes := sectionMap["types"]
		if !hasTypes {
			continue
		}

		// Convert types to []string
		var types []string
		if typesArray, isTypesArray := typesValue.([]any); isTypesArray {
			for _, t := range typesArray {
				if tStr, isTStr := t.(string); isTStr {
					types = append(types, tStr)
				}
			}
		}

		// Check if types includes "labeled" or "unlabeled"
		hasLabeled := false
		hasUnlabeled := false
		for _, t := range types {
			if t == "labeled" {
				hasLabeled = true
			}
			if t == "unlabeled" {
				hasUnlabeled = true
			}
		}

		if !hasLabeled && !hasUnlabeled {
			continue
		}

		// Check for "names" field
		namesValue, hasNames := sectionMap["names"]
		if !hasNames {
			continue
		}

		// Convert names to []string, handling both string and array formats
		var labelNames []string
		if namesStr, isNamesStr := namesValue.(string); isNamesStr {
			labelNames = []string{namesStr}
		} else if namesArray, isNamesArray := namesValue.([]any); isNamesArray {
			for _, name := range namesArray {
				if nameStr, isNameStr := name.(string); isNameStr {
					labelNames = append(labelNames, nameStr)
				}
			}
		} else {
			// Invalid names format, skip
			continue
		}

		if len(labelNames) == 0 {
			continue
		}

		// Build condition for this event section
		// The condition should be:
		// (event_name != 'issues' OR action != 'labeled' OR label.name in names) AND
		// (event_name != 'issues' OR action != 'unlabeled' OR label.name in names)

		// For each label name, create a condition
		var labelNameConditions []ConditionNode
		for _, labelName := range labelNames {
			labelNameConditions = append(labelNameConditions, BuildEquals(
				BuildPropertyAccess("github.event.label.name"),
				BuildStringLiteral(labelName),
			))
		}

		// Combine label name conditions with OR
		var labelNameMatch ConditionNode
		if len(labelNameConditions) == 1 {
			labelNameMatch = labelNameConditions[0]
		} else {
			labelNameMatch = &DisjunctionNode{Terms: labelNameConditions}
		}

		// Build conditions for labeled and unlabeled
		var sectionCondition ConditionNode

		if hasLabeled && hasUnlabeled {
			// Both labeled and unlabeled: check for either action
			notThisEvent := BuildNotEquals(
				BuildPropertyAccess("github.event_name"),
				BuildStringLiteral(section.eventNameStr),
			)

			notLabeledAction := BuildNotEquals(
				BuildPropertyAccess("github.event.action"),
				BuildStringLiteral("labeled"),
			)

			notUnlabeledAction := BuildNotEquals(
				BuildPropertyAccess("github.event.action"),
				BuildStringLiteral("unlabeled"),
			)

			// (event_name != 'issues') OR (action != 'labeled' AND action != 'unlabeled') OR (label.name matches)
			notLabelAction := &AndNode{Left: notLabeledAction, Right: notUnlabeledAction}
			sectionCondition = &OrNode{
				Left: notThisEvent,
				Right: &OrNode{
					Left:  notLabelAction,
					Right: labelNameMatch,
				},
			}
		} else if hasLabeled {
			// Only labeled
			notThisEvent := BuildNotEquals(
				BuildPropertyAccess("github.event_name"),
				BuildStringLiteral(section.eventNameStr),
			)

			notLabeledAction := BuildNotEquals(
				BuildPropertyAccess("github.event.action"),
				BuildStringLiteral("labeled"),
			)

			// (event_name != 'issues') OR (action != 'labeled') OR (label.name matches)
			sectionCondition = &OrNode{
				Left: notThisEvent,
				Right: &OrNode{
					Left:  notLabeledAction,
					Right: labelNameMatch,
				},
			}
		} else if hasUnlabeled {
			// Only unlabeled
			notThisEvent := BuildNotEquals(
				BuildPropertyAccess("github.event_name"),
				BuildStringLiteral(section.eventNameStr),
			)

			notUnlabeledAction := BuildNotEquals(
				BuildPropertyAccess("github.event.action"),
				BuildStringLiteral("unlabeled"),
			)

			// (event_name != 'issues') OR (action != 'unlabeled') OR (label.name matches)
			sectionCondition = &OrNode{
				Left: notThisEvent,
				Right: &OrNode{
					Left:  notUnlabeledAction,
					Right: labelNameMatch,
				},
			}
		}

		if sectionCondition != nil {
			labelConditions = append(labelConditions, sectionCondition)
		}
	}

	// If we have label conditions, combine them and apply to the workflow
	if len(labelConditions) > 0 {
		var finalCondition ConditionNode
		if len(labelConditions) == 1 {
			finalCondition = labelConditions[0]
		} else {
			// Combine all conditions with AND
			finalCondition = labelConditions[0]
			for i := 1; i < len(labelConditions); i++ {
				finalCondition = &AndNode{
					Left:  finalCondition,
					Right: labelConditions[i],
				}
			}
		}

		// Build condition tree and render
		existingCondition := data.If
		conditionTree := buildConditionTree(existingCondition, finalCondition.Render())
		data.If = conditionTree.Render()
	}
}
