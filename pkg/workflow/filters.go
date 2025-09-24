package workflow

// applyPullRequestDraftFilter applies draft filter conditions for pull_request triggers
func (c *Compiler) applyPullRequestDraftFilter(data *WorkflowData, frontmatter map[string]any) {
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

// applyPullRequestForkFilter applies fork filter conditions for pull_request triggers
// Supports "forks: []string" with glob patterns
func (c *Compiler) applyPullRequestForkFilter(data *WorkflowData, frontmatter map[string]any) {
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

	// Check if pull_request is an object with fork settings
	prMap, isPRMap := prValue.(map[string]any)
	if !isPRMap {
		return
	}

	// Check for "forks" field (string or array)
	forksValue, hasForks := prMap["forks"]

	if !hasForks {
		return
	}

	// Convert forks value to []string, handling both string and array formats
	var allowedForks []string

	// Handle string format (e.g., forks: "*" or forks: "org/*")
	if forksStr, isForksStr := forksValue.(string); isForksStr {
		allowedForks = []string{forksStr}
	} else if forksArray, isForksArray := forksValue.([]any); isForksArray {
		// Handle array format (e.g., forks: ["*", "org/repo"])
		for _, fork := range forksArray {
			if forkStr, isForkStr := fork.(string); isForkStr {
				allowedForks = append(allowedForks, forkStr)
			}
		}
	} else {
		// Invalid forks format, skip
		return
	}

	// If "*" wildcard is present, skip fork filtering (allow all forks)
	for _, pattern := range allowedForks {
		if pattern == "*" {
			return // No fork filtering needed
		}
	}

	// Build condition for allowed forks with glob support
	notPullRequestEvent := BuildNotEquals(
		BuildPropertyAccess("github.event_name"),
		BuildStringLiteral("pull_request"),
	)
	allowedForksCondition := BuildFromAllowedForks(allowedForks)

	forkCondition := &OrNode{
		Left:  notPullRequestEvent,
		Right: allowedForksCondition,
	}

	// Build condition tree and render
	existingCondition := data.If
	conditionTree := buildConditionTree(existingCondition, forkCondition.Render())
	data.If = conditionTree.Render()
}
