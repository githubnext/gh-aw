package workflow

import (
	"errors"
	"fmt"
	"maps"
	"path/filepath"
	"strings"

	"github.com/goccy/go-yaml"

	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/parser"
	"github.com/github/gh-aw/pkg/setutil"
	"github.com/github/gh-aw/pkg/sliceutil"
)

var triggerParserLog = logger.New("workflow:trigger_parser")

// TriggerIR represents the intermediate representation of a parsed trigger
type TriggerIR struct {
	// Event is the main GitHub Actions event type (e.g., "push", "pull_request", "issues")
	Event string

	// Types contains the activity types for the event (e.g., ["opened", "edited"])
	Types []string

	// Filters contains additional event filters (branches, paths, tags, labels, etc.)
	Filters map[string]any

	// Conditions contains job-level conditions for complex filtering
	Conditions []string

	// AdditionalEvents contains other events to include (e.g., workflow_dispatch)
	AdditionalEvents map[string]any
}

// ParseTriggerShorthand parses a human-readable trigger shorthand string
// and returns a structured intermediate representation that can be converted to YAML.
// Returns nil if the input is not a recognized trigger shorthand.
func ParseTriggerShorthand(input string) (*TriggerIR, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return nil, errors.New("trigger shorthand cannot be empty")
	}

	triggerParserLog.Printf("Parsing trigger shorthand: %s", input)

	for _, parser := range triggerShorthandParsers() {
		if ir, err := parser(input); ir != nil || err != nil {
			return ir, err
		}
	}

	// Not a recognized trigger shorthand
	return nil, nil
}

type triggerShorthandParser func(string) (*TriggerIR, error)

func triggerShorthandParsers() []triggerShorthandParser {
	return []triggerShorthandParser{
		parseSlashCommandTrigger,
		parseLabelTrigger,
		parseSourceControlTrigger,
		parseIssueDiscussionTrigger,
		parseManualTrigger,
		parseCommentTrigger,
		parseReleaseRepositoryTrigger,
		parseSecurityTrigger,
		parseExternalTrigger,
		parseDeploymentTrigger,
	}
}

// ToYAMLMap converts a TriggerIR to a map structure suitable for YAML generation
func (ir *TriggerIR) ToYAMLMap() map[string]any {
	result := make(map[string]any)

	// Add the main event
	if ir.Event != "" {
		eventConfig := make(map[string]any)

		// Add types if specified
		if len(ir.Types) > 0 {
			eventConfig["types"] = ir.Types
		}

		// Add filters
		maps.Copy(eventConfig, ir.Filters)

		// If event config has content, add it; otherwise omit the event entirely for simple triggers
		if len(eventConfig) > 0 {
			result[ir.Event] = eventConfig
		} else {
			// For events with no configuration, use an empty map instead of nil
			// This ensures proper YAML generation without "null" values
			result[ir.Event] = map[string]any{}
		}
	}

	// Add additional events
	maps.Copy(result, ir.AdditionalEvents)

	return result
}

// parseSlashCommandTrigger parses slash command triggers like "/test"
func parseSlashCommandTrigger(input string) (*TriggerIR, error) {
	commandName, isSlashCommand, err := parseSlashCommandShorthand(input)
	if err != nil {
		return nil, err
	}
	if !isSlashCommand {
		return nil, nil
	}

	triggerParserLog.Printf("Parsed slash command trigger: %s", commandName)

	// Note: slash_command is handled specially in the compiler, not as a standard GitHub event
	// We return nil here to let the existing slash command processing handle it
	return nil, nil
}

// parseLabelTrigger parses label triggers like "issue labeled bug" or "pull_request labeled needs-review"
func parseLabelTrigger(input string) (*TriggerIR, error) {
	entityType, labelNames, isLabelTrigger, err := parseLabelTriggerShorthand(input)
	if err != nil {
		return nil, err
	}
	if !isLabelTrigger {
		return nil, nil
	}

	triggerParserLog.Printf("Parsed label trigger: %s labeled %v", entityType, labelNames)

	// Note: Label triggers are handled specially via expandLabelTriggerShorthand
	// We return nil here to let the existing label trigger processing handle it
	return nil, nil
}

// parseSourceControlTrigger parses source control triggers
func parseSourceControlTrigger(input string) (*TriggerIR, error) {
	tokens := strings.Fields(input)
	if len(tokens) == 0 {
		return nil, nil
	}

	switch tokens[0] {
	case "push":
		return parsePushTrigger(tokens)
	case "pull", "pull_request":
		// Normalize "pull" to "pull_request"
		normalizedTokens := append([]string{"pull_request"}, tokens[1:]...)
		return parsePullRequestTrigger(normalizedTokens)
	default:
		return nil, nil
	}
}

// parsePushTrigger parses push-related triggers
func parsePushTrigger(tokens []string) (*TriggerIR, error) {
	if len(tokens) == 1 {
		// Simple "push" trigger - leave as simple string, don't convert
		// GitHub Actions supports simple event names as strings: on: push
		return nil, nil
	}

	if len(tokens) >= 3 && tokens[1] == "to" {
		// "push to <branch>"
		branch := strings.Join(tokens[2:], " ")
		triggerParserLog.Printf("Parsed push-to-branch trigger: branch=%s", branch)
		return &TriggerIR{
			Event: "push",
			Filters: map[string]any{
				"branches": []string{branch},
			},
			AdditionalEvents: map[string]any{
				"workflow_dispatch": nil,
			},
		}, nil
	}

	if len(tokens) >= 3 && tokens[1] == "tags" {
		// "push tags <pattern>"
		pattern := strings.Join(tokens[2:], " ")
		triggerParserLog.Printf("Parsed push-tags trigger: pattern=%s", pattern)
		return &TriggerIR{
			Event: "push",
			Filters: map[string]any{
				"tags": []string{pattern},
			},
			AdditionalEvents: map[string]any{
				"workflow_dispatch": nil,
			},
		}, nil
	}

	return nil, fmt.Errorf("invalid push trigger format: '%s'. Expected format: 'push to <branch>' or 'push tags <pattern>'. Example: 'push to main' or 'push tags v*'", strings.Join(tokens, " "))
}

// parsePullRequestTrigger parses pull request triggers
func parsePullRequestTrigger(tokens []string) (*TriggerIR, error) {
	if len(tokens) == 1 {
		// Simple "pull_request" trigger - leave as simple string
		// GitHub Actions supports: on: pull_request
		return nil, nil
	}

	// Check for activity type: "pull_request opened", "pull_request merged", etc.
	activityType := tokens[1]

	// Special case: "merged" is not a real type, it's a condition on "closed"
	if activityType == "merged" {
		triggerParserLog.Print("Parsed pull_request merged trigger (maps to closed with merge condition)")
		return mergedPullRequestTrigger(), nil
	}

	if setutil.Contains(validPullRequestActivityTypes(), activityType) {
		return pullRequestActivityTrigger(activityType, tokens), nil
	}

	// Check for "affecting" without activity type: "pull_request affecting <path>"
	if activityType == "affecting" && len(tokens) >= 3 {
		path := strings.Join(tokens[2:], " ")
		return &TriggerIR{
			Event: "pull_request",
			Types: []string{"opened", "synchronize", "reopened"},
			Filters: map[string]any{
				"paths": []string{path},
			},
			AdditionalEvents: map[string]any{
				"workflow_dispatch": nil,
			},
		}, nil
	}

	return nil, fmt.Errorf("invalid pull_request trigger format: '%s'. Expected format: 'pull_request <type>' or 'pull_request affecting <path>'. Valid types: opened, edited, closed, reopened, synchronize, merged, labeled, unlabeled. Example: 'pull_request opened' or 'pull_request affecting src/**'", strings.Join(tokens, " "))
}

func validPullRequestActivityTypes() map[string]struct{} {
	return map[string]struct{}{
		"opened":           {},
		"edited":           {},
		"closed":           {},
		"reopened":         {},
		"synchronize":      {},
		"assigned":         {},
		"unassigned":       {},
		"labeled":          {},
		"unlabeled":        {},
		"review_requested": {},
	}
}

func mergedPullRequestTrigger() *TriggerIR {
	return &TriggerIR{
		Event:      "pull_request",
		Types:      []string{"closed"},
		Conditions: []string{"github.event.pull_request.merged == true"},
		AdditionalEvents: map[string]any{
			"workflow_dispatch": nil,
		},
	}
}

func pullRequestActivityTrigger(activityType string, tokens []string) *TriggerIR {
	ir := &TriggerIR{
		Event: "pull_request",
		Types: []string{activityType},
		AdditionalEvents: map[string]any{
			"workflow_dispatch": nil,
		},
	}
	if len(tokens) >= 4 && tokens[2] == "affecting" {
		path := strings.Join(tokens[3:], " ")
		ir.Filters = map[string]any{
			"paths": []string{path},
		}
	}
	return ir
}

// parseIssueDiscussionTrigger parses issue and discussion triggers
func parseIssueDiscussionTrigger(input string) (*TriggerIR, error) {
	tokens := strings.Fields(input)
	if len(tokens) < 2 {
		return nil, nil
	}

	switch tokens[0] {
	case "issue":
		return parseIssueTrigger(tokens)
	case "discussion":
		return parseDiscussionTrigger(tokens)
	default:
		return nil, nil
	}
}

// parseIssueTrigger parses issue triggers
func parseIssueTrigger(tokens []string) (*TriggerIR, error) {
	if len(tokens) < 2 {
		return nil, errors.New("issue trigger requires an activity type. Expected format: 'issue <type>'. Valid types: opened, edited, closed, reopened, assigned, unassigned, labeled, unlabeled, deleted, transferred. Example: 'issue opened'")
	}

	activityType := tokens[1]

	// Map common activity types
	validTypes := map[string]struct {
	}{
		"opened":      {},
		"edited":      {},
		"closed":      {},
		"reopened":    {},
		"assigned":    {},
		"unassigned":  {},
		"labeled":     {},
		"unlabeled":   {},
		"deleted":     {},
		"transferred": {},
	}

	if !setutil.Contains(validTypes, activityType) {
		return nil, fmt.Errorf("invalid issue activity type: '%s'. Valid types: opened, edited, closed, reopened, assigned, unassigned, labeled, unlabeled, deleted, transferred. Example: 'issue opened'", activityType)
	}

	ir := &TriggerIR{
		Event: "issues",
		Types: []string{activityType},
		AdditionalEvents: map[string]any{
			"workflow_dispatch": nil,
		},
	}

	// Check for label filter: "issue opened labeled <label>"
	if len(tokens) >= 4 && tokens[2] == "labeled" {
		label := strings.Join(tokens[3:], " ")
		triggerParserLog.Printf("Parsed issue trigger with label filter: type=%s, label=%s", activityType, label)
		ir.Conditions = []string{
			fmt.Sprintf("contains(github.event.issue.labels.*.name, '%s')", label),
		}
	} else {
		triggerParserLog.Printf("Parsed issue trigger: type=%s", activityType)
	}

	return ir, nil
}

// parseDiscussionTrigger parses discussion triggers
func parseDiscussionTrigger(tokens []string) (*TriggerIR, error) {
	if len(tokens) < 2 {
		return nil, errors.New("discussion trigger requires an activity type. Expected format: 'discussion <type>'. Valid types: created, edited, deleted, transferred, pinned, unpinned, labeled, unlabeled, locked, unlocked, category_changed, answered, unanswered. Example: 'discussion created'")
	}

	activityType := tokens[1]

	// Map common activity types
	validTypes := map[string]struct {
	}{
		"created":          {},
		"edited":           {},
		"deleted":          {},
		"transferred":      {},
		"pinned":           {},
		"unpinned":         {},
		"labeled":          {},
		"unlabeled":        {},
		"locked":           {},
		"unlocked":         {},
		"category_changed": {},
		"answered":         {},
		"unanswered":       {},
	}

	if !setutil.Contains(validTypes, activityType) {
		return nil, fmt.Errorf("invalid discussion activity type: '%s'. Valid types: created, edited, deleted, transferred, pinned, unpinned, labeled, unlabeled, locked, unlocked, category_changed, answered, unanswered. Example: 'discussion created'", activityType)
	}

	return &TriggerIR{
		Event: "discussion",
		Types: []string{activityType},
		AdditionalEvents: map[string]any{
			"workflow_dispatch": nil,
		},
	}, nil
}

// parseManualTrigger parses manual invocation triggers
func parseManualTrigger(input string) (*TriggerIR, error) {
	tokens := strings.Fields(input)
	if len(tokens) == 0 {
		return nil, nil
	}

	if tokens[0] == "manual" {
		ir := &TriggerIR{
			AdditionalEvents: map[string]any{
				"workflow_dispatch": nil,
			},
		}

		// Check for input specification: "manual with input <name>"
		if len(tokens) >= 4 && tokens[1] == "with" && tokens[2] == "input" {
			inputName := tokens[3]
			triggerParserLog.Printf("Parsed manual trigger with input: %s", inputName)
			ir.AdditionalEvents["workflow_dispatch"] = map[string]any{
				"inputs": map[string]any{
					inputName: map[string]any{
						"description": "Input for " + inputName,
						"required":    false,
						"type":        "string",
					},
				},
			}
		} else {
			triggerParserLog.Print("Parsed manual trigger (workflow_dispatch)")
		}

		return ir, nil
	}

	if len(tokens) >= 3 && tokens[0] == "workflow" && tokens[1] == "completed" {
		// "workflow completed <workflow-name>"
		workflowName := strings.Join(tokens[2:], " ")
		return &TriggerIR{
			Event: "workflow_run",
			Types: []string{"completed"},
			Filters: map[string]any{
				"workflows": []string{workflowName},
			},
		}, nil
	}

	return nil, nil
}

// parseCommentTrigger parses comment triggers
func parseCommentTrigger(input string) (*TriggerIR, error) {
	tokens := strings.Fields(input)
	if len(tokens) < 2 {
		return nil, nil
	}

	if tokens[0] == "comment" && tokens[1] == "created" {
		// "comment created" - supports both issue and PR comments
		return &TriggerIR{
			Event: "issue_comment",
			Types: []string{"created"},
			AdditionalEvents: map[string]any{
				"workflow_dispatch": nil,
			},
		}, nil
	}

	return nil, nil
}

// parseReleaseRepositoryTrigger parses release and repository lifecycle triggers
func parseReleaseRepositoryTrigger(input string) (*TriggerIR, error) {
	tokens := strings.Fields(input)
	if len(tokens) < 2 {
		return nil, nil
	}

	switch tokens[0] {
	case "release":
		return parseReleaseTrigger(tokens)
	case "repository":
		return parseRepositoryTrigger(tokens)
	default:
		return nil, nil
	}
}

// parseReleaseTrigger parses release triggers
func parseReleaseTrigger(tokens []string) (*TriggerIR, error) {
	if len(tokens) < 2 {
		return nil, errors.New("release trigger requires an activity type. Expected format: 'release <type>'. Valid types: published, unpublished, created, edited, deleted, prereleased, released. Example: 'release published'")
	}

	activityType := tokens[1]

	validTypes := map[string]struct {
	}{
		"published":   {},
		"unpublished": {},
		"created":     {},
		"edited":      {},
		"deleted":     {},
		"prereleased": {},
		"released":    {},
	}

	if !setutil.Contains(validTypes, activityType) {
		return nil, fmt.Errorf("invalid release activity type: '%s'. Valid types: published, unpublished, created, edited, deleted, prereleased, released. Example: 'release published'", activityType)
	}

	return &TriggerIR{
		Event: "release",
		Types: []string{activityType},
		AdditionalEvents: map[string]any{
			"workflow_dispatch": nil,
		},
	}, nil
}

// parseRepositoryTrigger parses repository lifecycle triggers
func parseRepositoryTrigger(tokens []string) (*TriggerIR, error) {
	if len(tokens) < 2 {
		return nil, errors.New("repository trigger requires an activity type. Expected format: 'repository <type>'. Valid types: starred, forked. Example: 'repository starred'")
	}

	activityType := tokens[1]

	// Map activity types to events
	switch activityType {
	case "starred":
		// GitHub Actions uses "watch" event for starring
		return &TriggerIR{
			Event: "watch",
			Types: []string{"started"},
			AdditionalEvents: map[string]any{
				"workflow_dispatch": nil,
			},
		}, nil
	case "forked":
		return &TriggerIR{
			Event:   "fork",
			Filters: map[string]any{}, // Empty map to avoid null in YAML
			AdditionalEvents: map[string]any{
				"workflow_dispatch": nil,
			},
		}, nil
	default:
		return nil, fmt.Errorf("invalid repository activity type: '%s'. Valid types: starred, forked. Example: 'repository starred'", activityType)
	}
}

// parseSecurityTrigger parses security-related triggers
func parseSecurityTrigger(input string) (*TriggerIR, error) {
	tokens := strings.Fields(input)
	if len(tokens) < 2 {
		return nil, nil
	}

	if tokens[0] == "dependabot" && len(tokens) >= 3 && tokens[1] == "pull" && tokens[2] == "request" {
		// "dependabot pull request" - filter pull requests by Dependabot author.
		// Guard against the Dependabot Confused Deputy attack (@dependabot recreate) by
		// requiring the PR author to also be dependabot[bot], not just the current actor.
		// Reference: https://labs.boostsecurity.io/articles/weaponizing-dependabot-pwn-request-at-its-finest/
		return &TriggerIR{
			Event:      "pull_request",
			Types:      []string{"opened", "synchronize", "reopened"},
			Conditions: []string{"github.actor == 'dependabot[bot]' && github.event.pull_request.user.login == 'dependabot[bot]'"},
			AdditionalEvents: map[string]any{
				"workflow_dispatch": nil,
			},
		}, nil
	}

	if tokens[0] == "security" && tokens[1] == "alert" {
		// "security alert" - code scanning alert
		return &TriggerIR{
			Event: "code_scanning_alert",
			Types: []string{"created", "reopened", "fixed"},
			AdditionalEvents: map[string]any{
				"workflow_dispatch": nil,
			},
		}, nil
	}

	if len(tokens) >= 3 && tokens[0] == "code" && tokens[1] == "scanning" && tokens[2] == "alert" {
		// "code scanning alert" - explicit code scanning alert
		return &TriggerIR{
			Event: "code_scanning_alert",
			Types: []string{"created", "reopened", "fixed"},
			AdditionalEvents: map[string]any{
				"workflow_dispatch": nil,
			},
		}, nil
	}

	return nil, nil
}

// parseExternalTrigger parses external integration triggers
func parseExternalTrigger(input string) (*TriggerIR, error) {
	tokens := strings.Fields(input)
	if len(tokens) < 3 {
		return nil, nil
	}

	if tokens[0] == "api" && tokens[1] == "dispatch" {
		// "api dispatch <event-type>"
		eventType := strings.Join(tokens[2:], " ")
		return &TriggerIR{
			Event: "repository_dispatch",
			Filters: map[string]any{
				"types": []string{eventType},
			},
		}, nil
	}

	return nil, nil
}

// parseDeploymentTrigger parses deployment status triggers with optional state filtering.
// Supported patterns:
//   - "deployment failed"          → deployment_status filtered to failure
//   - "deployment error"           → deployment_status filtered to error
//   - "deployment failed or error" → deployment_status filtered to failure or error
//   - "deployment_status"          → deployment_status (all states, no filter)
func parseDeploymentTrigger(input string) (*TriggerIR, error) {
	tokens := strings.Fields(input)
	if len(tokens) == 0 {
		return nil, nil
	}

	// Only handle "deployment" or "deployment_status" prefix
	if tokens[0] != "deployment" && tokens[0] != "deployment_status" {
		return nil, nil
	}

	// Bare "deployment_status" with no further args - let it fall through as a simple string
	if len(tokens) == 1 {
		return nil, nil
	}

	states, ok := parseDeploymentStates(tokens[1:])
	if !ok {
		return nil, nil
	}
	if len(states) == 0 {
		return nil, nil
	}

	condition := deploymentStateCondition(states)
	triggerParserLog.Printf("Parsed deployment trigger with states %v, condition: %s", states, condition)

	return &TriggerIR{
		Event:      "deployment_status",
		Conditions: []string{condition},
	}, nil
}

func parseDeploymentStates(tokens []string) ([]string, bool) {
	stateAliases := map[string]string{
		"failed":    "failure",
		"failure":   "failure",
		"error":     "error",
		"errored":   "error",
		"success":   "success",
		"succeeded": "success",
		"pending":   "pending",
		"inactive":  "inactive",
	}
	var states []string
	seenStates := make(map[string]struct{})
	conjunctions := map[string]struct{}{"or": {}, "and": {}}
	for _, tok := range tokens {
		tok = strings.ToLower(strings.TrimRight(tok, ","))
		if setutil.Contains(conjunctions, tok) {
			continue
		}
		state, ok := stateAliases[tok]
		if !ok {
			return nil, false
		}
		if !setutil.Contains(seenStates, state) {
			states = append(states, state)
			seenStates[state] = struct{}{}
		}
	}
	return states, true
}

func deploymentStateCondition(states []string) string {
	parts := make([]string, 0, len(states))
	for _, s := range states {
		parts = append(parts, "github.event.deployment_status.state == '"+s+"'")
	}
	stateExpr := strings.Join(parts, " || ")
	return "github.event_name != 'deployment_status' || (" + stateExpr + ")"
}

func mergeCommandOtherEvents(existing map[string]any, incoming map[string]any) map[string]any {
	if len(existing) == 0 {
		return incoming
	}
	if len(incoming) == 0 {
		return existing
	}
	merged := maps.Clone(existing)
	for eventName, incomingValue := range incoming {
		if existingValue, hasExisting := merged[eventName]; hasExisting {
			merged[eventName] = mergeEventConfig(existingValue, incomingValue)
			continue
		}
		merged[eventName] = incomingValue
	}
	return merged
}

func mergeEventConfig(existing any, incoming any) any {
	existingMap, existingOK := existing.(map[string]any)
	incomingMap, incomingOK := incoming.(map[string]any)
	if !existingOK || !incomingOK {
		return incoming
	}
	merged := maps.Clone(existingMap)
	maps.Copy(merged, incomingMap)

	existingTypes, existingTypesOK := parseEventTypes(existingMap["types"])
	incomingTypes, incomingTypesOK := parseEventTypes(incomingMap["types"])
	if existingTypesOK && incomingTypesOK {
		combined := sliceutil.MergeUnique(existingTypes, incomingTypes...)
		merged["types"] = combined
	}

	return merged
}

func parseEventTypes(value any) ([]string, bool) {
	switch typed := value.(type) {
	case []string:
		return typed, true
	case []any:
		out := make([]string, 0, len(typed))
		for _, entry := range typed {
			entryStr, ok := entry.(string)
			if !ok {
				return nil, false
			}
			out = append(out, entryStr)
		}
		return out, true
	default:
		return nil, false
	}
}

// parseOnSection handles parsing of the "on" section from frontmatter, extracting command triggers,
// reactions, and stop-after configurations while detecting conflicts with other event types.
func (c *Compiler) parseOnSection(frontmatter map[string]any, workflowData *WorkflowData, markdownPath string) error {
	triggerParserLog.Printf("Parsing on section: workflow=%s, markdownPath=%s", workflowData.Name, markdownPath)
	state := &onSectionParseState{}
	if onValue, exists := getOnSectionValue(frontmatter, workflowData); exists {
		if onMap, ok := onValue.(map[string]any); ok {
			if err := c.parseOnMap(onMap, workflowData, markdownPath, state); err != nil {
				return err
			}
		}
	}

	applyOnSectionDefaults(workflowData, state)
	return c.storeOnSectionOtherEvents(frontmatter, workflowData, state)
}

type onSectionParseState struct {
	hasCommand       bool
	hasLabelCommand  bool
	hasReaction      bool
	hasStopAfter     bool
	hasStatusComment bool
	otherEvents      map[string]any
}

func getOnSectionValue(frontmatter map[string]any, workflowData *WorkflowData) (any, bool) {
	if workflowData.ParsedFrontmatter != nil && workflowData.ParsedFrontmatter.On != nil {
		return workflowData.ParsedFrontmatter.On, true
	}
	onValue, exists := frontmatter["on"]
	return onValue, exists
}

func (c *Compiler) parseOnMap(onMap map[string]any, workflowData *WorkflowData, markdownPath string, state *onSectionParseState) error {
	if _, hasStopAfterKey := onMap["stop-after"]; hasStopAfterKey {
		state.hasStopAfter = true
	}
	if err := c.extractOnSectionReaction(onMap, workflowData, state); err != nil {
		return err
	}
	if err := c.extractOnSectionStatusComment(onMap, workflowData, state); err != nil {
		return err
	}
	extractOnSectionLockForAgent(onMap, workflowData)
	if err := extractOnSectionCommand(onMap, workflowData, markdownPath, state); err != nil {
		return err
	}
	if err := extractOnSectionLabelCommand(onMap, workflowData, markdownPath, state); err != nil {
		return err
	}
	state.otherEvents = excludeMapKeys(onMap, "slash_command", "command", "label_command", "reaction", "status-comment", "stop-after", "github-token", "github-app", "needs")
	return nil
}

func (c *Compiler) extractOnSectionReaction(onMap map[string]any, workflowData *WorkflowData, state *onSectionParseState) error {
	reactionValue, hasReactionField := onMap["reaction"]
	if !hasReactionField {
		return nil
	}
	state.hasReaction = true
	reactionStr, reactionIssues, reactionPullRequests, reactionDiscussions, err := parseReactionConfig(reactionValue)
	if err != nil {
		return err
	}
	if !isValidReaction(reactionStr) {
		return fmt.Errorf("invalid reaction value '%s': must be one of %v", reactionStr, getValidReactions())
	}
	workflowData.AIReaction = reactionStr
	workflowData.ReactionIssues = reactionIssues
	workflowData.ReactionPullRequests = reactionPullRequests
	workflowData.ReactionDiscussions = reactionDiscussions
	return nil
}

func (c *Compiler) extractOnSectionStatusComment(onMap map[string]any, workflowData *WorkflowData, state *onSectionParseState) error {
	statusCommentValue, hasStatusCommentField := onMap["status-comment"]
	if !hasStatusCommentField {
		return nil
	}
	state.hasStatusComment = true
	if statusCommentBool, ok := statusCommentValue.(bool); ok {
		workflowData.StatusComment = &statusCommentBool
		triggerParserLog.Printf("status-comment set to: %v", statusCommentBool)
		return nil
	}
	statusCommentMap, ok := statusCommentValue.(map[string]any)
	if !ok {
		return fmt.Errorf("status-comment must be a boolean or object value, got %T", statusCommentValue)
	}
	return applyStatusCommentMap(statusCommentMap, workflowData)
}

func applyStatusCommentMap(statusCommentMap map[string]any, workflowData *WorkflowData) error {
	statusCommentIssues, err := statusCommentTargetValue(statusCommentMap, "issues")
	if err != nil {
		return err
	}
	statusCommentPullRequests, err := statusCommentTargetValue(statusCommentMap, "pull-requests")
	if err != nil {
		return err
	}
	statusCommentDiscussions, err := statusCommentTargetValue(statusCommentMap, "discussions")
	if err != nil {
		return err
	}
	if !statusCommentIssues && !statusCommentPullRequests && !statusCommentDiscussions {
		return errors.New("status-comment object requires at least one target to be enabled (issues, pull-requests, or discussions)")
	}
	statusCommentEnabled := true
	workflowData.StatusComment = &statusCommentEnabled
	workflowData.StatusCommentIssues = &statusCommentIssues
	workflowData.StatusCommentPullRequests = &statusCommentPullRequests
	workflowData.StatusCommentDiscussions = &statusCommentDiscussions
	triggerParserLog.Printf("status-comment object set: issues=%v pullRequests=%v discussions=%v", statusCommentIssues, statusCommentPullRequests, statusCommentDiscussions)
	return nil
}

func statusCommentTargetValue(statusCommentMap map[string]any, key string) (bool, error) {
	value, exists := statusCommentMap[key]
	if !exists {
		return true, nil
	}
	boolValue, ok := value.(bool)
	if !ok {
		return false, fmt.Errorf("status-comment.%s must be a boolean value, got %T", key, value)
	}
	return boolValue, nil
}

func extractOnSectionLockForAgent(onMap map[string]any, workflowData *WorkflowData) {
	extractLockForAgentFromEvent(onMap, workflowData, "issues")
	extractLockForAgentFromEvent(onMap, workflowData, "issue_comment")
}

func extractLockForAgentFromEvent(onMap map[string]any, workflowData *WorkflowData, eventName string) {
	eventValue, hasEvent := onMap[eventName]
	if !hasEvent {
		return
	}
	eventMap, ok := eventValue.(map[string]any)
	if !ok {
		return
	}
	if lockForAgent, hasLockForAgent := eventMap["lock-for-agent"]; hasLockForAgent {
		if lockBool, ok := lockForAgent.(bool); ok {
			workflowData.LockForAgent = lockBool
			triggerParserLog.Printf("lock-for-agent enabled for %s: %v", eventName, lockBool)
		}
	}
}

func extractOnSectionCommand(onMap map[string]any, workflowData *WorkflowData, markdownPath string, state *onSectionParseState) error {
	if _, hasSlashCommandKey := onMap["slash_command"]; hasSlashCommandKey {
		state.hasCommand = true
		setDefaultCommand(workflowData, markdownPath)
		if !workflowData.CommandCentralized {
			if err := validateCommandTriggerConflicts(onMap, "slash_command"); err != nil {
				return err
			}
		}
		workflowData.On = ""
	} else if _, hasCommandKey := onMap["command"]; hasCommandKey {
		state.hasCommand = true
		setDefaultCommand(workflowData, markdownPath)
		if err := validateCommandTriggerConflicts(onMap, "command"); err != nil {
			return err
		}
		workflowData.On = ""
	}
	return nil
}

func setDefaultCommand(workflowData *WorkflowData, markdownPath string) {
	if len(workflowData.Command) == 0 {
		baseName := strings.TrimSuffix(filepath.Base(markdownPath), ".md")
		workflowData.Command = []string{baseName}
	}
}

func validateCommandTriggerConflicts(onMap map[string]any, triggerName string) error {
	conflictingEvents := []string{"issues", "issue_comment", "pull_request", "pull_request_review_comment"}
	for _, eventName := range conflictingEvents {
		if eventValue, hasConflict := onMap[eventName]; hasConflict {
			if (eventName == "issues" || eventName == "pull_request") && parser.IsNonConflictingCommandEvent(eventValue) {
				continue
			}
			return fmt.Errorf("cannot use '%s' with '%s' in the same workflow", triggerName, eventName)
		}
	}
	return nil
}

func extractOnSectionLabelCommand(onMap map[string]any, workflowData *WorkflowData, markdownPath string, state *onSectionParseState) error {
	if _, hasLabelCommandKey := onMap["label_command"]; !hasLabelCommandKey {
		return nil
	}
	state.hasLabelCommand = true
	if len(workflowData.LabelCommand) == 0 {
		baseName := strings.TrimSuffix(filepath.Base(markdownPath), ".md")
		workflowData.LabelCommand = []string{baseName}
	}
	if !workflowData.LabelCommandDecentralized {
		if err := validateLabelCommandConflicts(onMap); err != nil {
			return err
		}
	}
	workflowData.On = ""
	return nil
}

func validateLabelCommandConflicts(onMap map[string]any) error {
	labelConflictingEvents := []string{"issues", "pull_request", "discussion"}
	for _, eventName := range labelConflictingEvents {
		if eventValue, hasConflict := onMap[eventName]; hasConflict {
			if !parser.IsLabelOnlyEvent(eventValue) {
				return fmt.Errorf("cannot use 'label_command' with '%s' trigger (non-label types); use only labeled/unlabeled types or remove this trigger", eventName)
			}
		}
	}
	return nil
}

func applyOnSectionDefaults(workflowData *WorkflowData, state *onSectionParseState) {
	if !state.hasCommand {
		workflowData.Command = nil
	}
	if !state.hasLabelCommand {
		workflowData.LabelCommand = nil
		workflowData.LabelCommandEvents = nil
		workflowData.LabelCommandDecentralized = false
	}
	if (state.hasCommand || state.hasLabelCommand) && !state.hasReaction && workflowData.AIReaction == "" {
		workflowData.AIReaction = "eyes"
	}
	if (state.hasCommand || state.hasLabelCommand) && !state.hasStatusComment && workflowData.StatusComment == nil {
		trueVal := true
		workflowData.StatusComment = &trueVal
	}
}

func (c *Compiler) storeOnSectionOtherEvents(frontmatter map[string]any, workflowData *WorkflowData, state *onSectionParseState) error {
	if state.hasCommand && len(state.otherEvents) > 0 {
		workflowData.On = ""
		workflowData.CommandOtherEvents = mergeCommandOtherEvents(workflowData.CommandOtherEvents, state.otherEvents)
	} else if state.hasLabelCommand && len(state.otherEvents) > 0 {
		workflowData.On = ""
		workflowData.LabelCommandOtherEvents = state.otherEvents
	} else if (state.hasReaction || state.hasStopAfter || state.hasStatusComment) && len(state.otherEvents) > 0 {
		c.remarshalOnSectionOtherEvents(frontmatter, workflowData, state.otherEvents)
	}
	return nil
}

func (c *Compiler) remarshalOnSectionOtherEvents(frontmatter map[string]any, workflowData *WorkflowData, otherEvents map[string]any) {
	onEventsYAML, err := yaml.MarshalWithOptions(map[string]any{"on": otherEvents}, yaml.IndentSequence(true))
	if err != nil {
		workflowData.On = c.extractTopLevelYAMLSection(frontmatter, "on")
		return
	}
	yamlStr := strings.TrimSuffix(string(onEventsYAML), "\n")
	yamlStr = parser.QuoteCronExpressions(yamlStr)
	yamlStr = c.commentOutProcessedFieldsInOnSection(yamlStr, frontmatter)
	yamlStr = c.addZizmorIgnoreForWorkflowRun(yamlStr)
	workflowData.On = yamlStr
}
