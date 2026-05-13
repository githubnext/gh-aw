package workflow

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/logger"
)

var centralSlashCommandWorkflowLog = logger.New("workflow:central_slash_command_workflow")

const (
	centralSlashCommandWorkflowFilename       = "agentic_commands.yml"
	legacyCentralSlashCommandWorkflowFilename = "agentic_slash_commands.yml"
)

type slashCommandRoute struct {
	Workflow   string   `json:"workflow"`
	Events     []string `json:"events"`
	AIReaction string   `json:"ai_reaction,omitempty"`
}

type commandsHeaderMetadata struct {
	PayloadVersion string   `json:"payload_version"`
	SchemaVersion  string   `json:"schema_version"`
	Compiler       string   `json:"compiler_version"`
	Commands       []string `json:"commands"`
	Workflows      []string `json:"workflows"`
}

// GenerateCentralSlashCommandWorkflow generates a single centralized slash-command trigger
// workflow for workflows that opt into on.slash_command.strategy: centralized.
// When no centralized slash-command workflows are found, any existing generated file is deleted.
func GenerateCentralSlashCommandWorkflow(workflowDataList []*WorkflowData, workflowDir string) error {
	centralSlashCommandWorkflowLog.Printf("Generating centralized slash-command workflow from %d workflow(s)", len(workflowDataList))
	routesByCommand, mergedEvents := collectCentralSlashCommandRoutes(workflowDataList)

	triggerFile := filepath.Join(workflowDir, centralSlashCommandWorkflowFilename)
	legacyTriggerFile := filepath.Join(workflowDir, legacyCentralSlashCommandWorkflowFilename)
	if len(routesByCommand) == 0 || len(mergedEvents) == 0 {
		centralSlashCommandWorkflowLog.Print("No centralized slash-command participants found")
		if err := removeIfExists(triggerFile); err != nil {
			return fmt.Errorf("failed to delete centralized slash-command workflow: %w", err)
		}
		if err := cleanupLegacyCentralSlashCommandWorkflow(legacyTriggerFile); err != nil {
			return err
		}
		return nil
	}

	actionMode := DetectActionMode(GetVersion())
	setupActionRef := ResolveSetupActionReference(actionMode, GetVersion(), "", nil)

	content, err := buildCentralSlashCommandWorkflowYAML(routesByCommand, mergedEvents, resolveCentralSlashRunsOn(workflowDataList), setupActionRef)
	if err != nil {
		return err
	}

	if err := os.WriteFile(triggerFile, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write centralized slash-command workflow: %w", err)
	}
	if err := cleanupLegacyCentralSlashCommandWorkflow(legacyTriggerFile); err != nil {
		return err
	}
	centralSlashCommandWorkflowLog.Printf("Wrote centralized slash-command workflow: %s", triggerFile)
	return nil
}

func cleanupLegacyCentralSlashCommandWorkflow(path string) error {
	if err := removeIfExists(path); err != nil {
		return fmt.Errorf("failed to delete legacy centralized slash-command workflow: %w", err)
	}
	return nil
}

func removeIfExists(path string) error {
	if _, err := os.Stat(path); err == nil {
		return os.Remove(path)
	} else if !os.IsNotExist(err) {
		return err
	}
	return nil
}

func collectCentralSlashCommandRoutes(workflowDataList []*WorkflowData) (map[string][]slashCommandRoute, map[string]map[string]bool) {
	routesByCommand := make(map[string][]slashCommandRoute)
	mergedEvents := make(map[string]map[string]bool)

	for _, wd := range workflowDataList {
		if wd == nil || !wd.CommandCentralized || len(wd.Command) == 0 {
			continue
		}

		filteredEvents := FilterCommentEvents(wd.CommandEvents)
		if len(filteredEvents) == 0 {
			continue
		}

		routeEvents := GetCommentEventNames(filteredEvents)
		routeEvents = uniqueSorted(routeEvents)
		if len(routeEvents) == 0 {
			continue
		}

		// Merge workflow-level subscriptions using YAML-ready GitHub event names.
		for _, event := range MergeEventsForYAML(filteredEvents) {
			if mergedEvents[event.EventName] == nil {
				mergedEvents[event.EventName] = make(map[string]bool)
			}
			for _, t := range event.Types {
				mergedEvents[event.EventName][t] = true
			}
		}

		for _, commandName := range wd.Command {
			for _, route := range buildCentralizedRoutes(wd, routeEvents) {
				routesByCommand[commandName] = append(routesByCommand[commandName], route)
			}
		}
	}

	// Stable ordering for deterministic output.
	for commandName := range routesByCommand {
		sort.Slice(routesByCommand[commandName], func(i, j int) bool {
			left := routesByCommand[commandName][i]
			right := routesByCommand[commandName][j]
			if left.Workflow != right.Workflow {
				return left.Workflow < right.Workflow
			}
			leftEvents := strings.Join(left.Events, ",")
			rightEvents := strings.Join(right.Events, ",")
			if leftEvents != rightEvents {
				return leftEvents < rightEvents
			}
			return left.AIReaction < right.AIReaction
		})
	}

	return routesByCommand, mergedEvents
}

func buildCentralizedRoutes(wd *WorkflowData, routeEvents []string) []slashCommandRoute {
	if wd == nil {
		return nil
	}
	eventGroups := map[string][]string{}
	groupOrder := make([]string, 0, len(routeEvents))
	for _, eventName := range routeEvents {
		reaction := resolveCentralizedEventReaction(wd, eventName)
		if _, exists := eventGroups[reaction]; !exists {
			groupOrder = append(groupOrder, reaction)
		}
		eventGroups[reaction] = append(eventGroups[reaction], eventName)
	}
	routes := make([]slashCommandRoute, 0, len(groupOrder))
	for _, reaction := range groupOrder {
		routes = append(routes, slashCommandRoute{
			Workflow:   wd.WorkflowID,
			Events:     slices.Clone(eventGroups[reaction]),
			AIReaction: reaction,
		})
	}
	return routes
}

func resolveCentralizedEventReaction(wd *WorkflowData, eventName string) string {
	if wd == nil || wd.AIReaction == "" || wd.AIReaction == "none" {
		return ""
	}

	switch eventName {
	case "issues", "issue_comment":
		if shouldIncludeIssueReactions(wd) {
			return wd.AIReaction
		}
	case "pull_request", "pull_request_comment", "pull_request_review_comment":
		if shouldIncludePullRequestReactions(wd) {
			return wd.AIReaction
		}
	case "discussion", "discussion_comment":
		if shouldIncludeDiscussionReactions(wd) {
			return wd.AIReaction
		}
	}

	return ""
}

func buildCentralSlashCommandWorkflowYAML(routesByCommand map[string][]slashCommandRoute, mergedEvents map[string]map[string]bool, runsOn string, setupActionRef string) (string, error) {
	routesJSON, err := json.Marshal(routesByCommand)
	if err != nil {
		return "", fmt.Errorf("failed to marshal centralized slash-command routes: %w", err)
	}

	commandsMetadata, err := json.Marshal(buildCommandsHeaderMetadata(routesByCommand))
	if err != nil {
		return "", fmt.Errorf("failed to marshal centralized slash-command metadata: %w", err)
	}

	header := GenerateWorkflowHeader("", "gh-aw", "")

	var b strings.Builder
	b.WriteString("# gh-aw-commands: ")
	b.Write(commandsMetadata)
	b.WriteString("\n")
	b.WriteString(header)
	b.WriteString(`name: "Agentic Commands"

on:
`)
	writeCentralSlashEventsYAML(&b, mergedEvents)
	b.WriteString(`
permissions: {}

jobs:
  route:
    runs-on: ` + runsOn + `
`)
	writeCentralSlashRoutePermissions(&b, mergedEvents)
	b.WriteString(`
    steps:
      - name: Checkout repository
        uses: ` + getActionPin("actions/checkout") + `

      - name: Setup Scripts
        uses: ` + setupActionRef + `
        with:
          destination: ` + SetupActionDestination + `

      - name: Route slash command
        uses: ` + getActionPin("actions/github-script") + `
        env:
          GH_AW_SLASH_ROUTING: '` + escapeSingleQuotedYAMLString(string(routesJSON)) + `'
        with:
          script: |
            const { setupGlobals } = require('` + SetupActionDestination + `/setup_globals.cjs');
            setupGlobals(core, github, context, exec, io, getOctokit);
            const { main } = require('` + SetupActionDestination + `/route_slash_command.cjs');
            await main();
`)
	return b.String(), nil
}

func writeCentralSlashRoutePermissions(b *strings.Builder, mergedEvents map[string]map[string]bool) {
	b.WriteString(`    permissions:
      actions: write
      contents: read
`)
	if mergedEvents["issues"] != nil || mergedEvents["issue_comment"] != nil || mergedEvents["pull_request"] != nil {
		b.WriteString("      issues: write\n")
	}
	if mergedEvents["pull_request"] != nil || mergedEvents["pull_request_comment"] != nil || mergedEvents["pull_request_review_comment"] != nil {
		b.WriteString("      pull-requests: write\n")
	}
	if mergedEvents["discussion"] != nil || mergedEvents["discussion_comment"] != nil {
		b.WriteString("      discussions: write\n")
	}
}

func buildCommandsHeaderMetadata(routesByCommand map[string][]slashCommandRoute) commandsHeaderMetadata {
	commands := make([]string, 0, len(routesByCommand))
	workflowSet := make(map[string]bool)
	for command, routes := range routesByCommand {
		commands = append(commands, command)
		for _, route := range routes {
			if route.Workflow != "" {
				workflowSet[route.Workflow] = true
			}
		}
	}
	sort.Strings(commands)
	workflows := make([]string, 0, len(workflowSet))
	for workflowID := range workflowSet {
		workflows = append(workflows, workflowID)
	}
	sort.Strings(workflows)
	metadataCompilerVersion := "dev"
	if IsRelease() && strings.TrimSpace(GetVersion()) != "" {
		metadataCompilerVersion = GetVersion()
	}
	return commandsHeaderMetadata{
		PayloadVersion: "v1",
		SchemaVersion:  "v1",
		Compiler:       metadataCompilerVersion,
		Commands:       commands,
		Workflows:      workflows,
	}
}

func resolveCentralSlashRunsOn(workflowDataList []*WorkflowData) string {
	counts := map[string]int{}
	for _, wd := range workflowDataList {
		if wd == nil || !wd.CommandCentralized || len(wd.Command) == 0 {
			continue
		}

		resolved := constants.DefaultActivationJobRunnerImage
		if wd.SafeOutputs != nil && strings.TrimSpace(wd.SafeOutputs.RunsOn) != "" {
			resolved = strings.TrimSpace(wd.SafeOutputs.RunsOn)
		} else if strings.TrimSpace(wd.RunsOnSlim) != "" {
			resolved = strings.TrimSpace(wd.RunsOnSlim)
		}
		counts[resolved]++
	}

	best := constants.DefaultActivationJobRunnerImage
	bestCount := counts[best]
	for candidate, count := range counts {
		if count > bestCount || (count == bestCount && candidate < best) {
			best = candidate
			bestCount = count
		}
	}
	return best
}

func writeCentralSlashEventsYAML(b *strings.Builder, mergedEvents map[string]map[string]bool) {
	eventOrder := []string{
		"issues",
		"issue_comment",
		"pull_request",
		"pull_request_review_comment",
		"discussion",
		"discussion_comment",
	}

	for _, eventName := range eventOrder {
		typeSet := mergedEvents[eventName]
		if len(typeSet) == 0 {
			continue
		}
		types := make([]string, 0, len(typeSet))
		for t := range typeSet {
			types = append(types, t)
		}
		sort.Strings(types)
		b.WriteString("  " + eventName + ":\n")
		b.WriteString("    types: [" + strings.Join(types, ", ") + "]\n")
	}
}

func uniqueSorted(values []string) []string {
	seen := make(map[string]bool, len(values))
	for _, v := range values {
		seen[v] = true
	}
	result := make([]string, 0, len(seen))
	for v := range seen {
		result = append(result, v)
	}
	sort.Strings(result)
	return result
}

func escapeSingleQuotedYAMLString(input string) string {
	return strings.ReplaceAll(input, "'", "''")
}
