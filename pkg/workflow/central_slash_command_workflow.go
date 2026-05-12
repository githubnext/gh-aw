package workflow

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"github.com/github/gh-aw/pkg/logger"
)

var centralSlashCommandWorkflowLog = logger.New("workflow:central_slash_command_workflow")

const centralSlashCommandWorkflowFilename = "agentics-slash-command-trigger.yml"

type slashCommandRoute struct {
	Workflow string   `json:"workflow"`
	Events   []string `json:"events"`
}

// GenerateCentralSlashCommandWorkflow generates a single centralized slash-command trigger
// workflow for workflows that opt into on.slash_command.strategy: centralized.
// When no centralized slash-command workflows are found, any existing generated file is deleted.
func GenerateCentralSlashCommandWorkflow(workflowDataList []*WorkflowData, workflowDir string) error {
	centralSlashCommandWorkflowLog.Printf("Generating centralized slash-command workflow from %d workflow(s)", len(workflowDataList))
	routesByCommand, mergedEvents := collectCentralSlashCommandRoutes(workflowDataList)

	triggerFile := filepath.Join(workflowDir, centralSlashCommandWorkflowFilename)
	if len(routesByCommand) == 0 || len(mergedEvents) == 0 {
		centralSlashCommandWorkflowLog.Print("No centralized slash-command participants found")
		if _, err := os.Stat(triggerFile); err == nil {
			if err := os.Remove(triggerFile); err != nil {
				return fmt.Errorf("failed to delete centralized slash-command workflow: %w", err)
			}
		}
		return nil
	}

	content, err := buildCentralSlashCommandWorkflowYAML(routesByCommand, mergedEvents)
	if err != nil {
		return err
	}

	if err := os.WriteFile(triggerFile, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write centralized slash-command workflow: %w", err)
	}
	centralSlashCommandWorkflowLog.Printf("Wrote centralized slash-command workflow: %s", triggerFile)
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
			route := slashCommandRoute{
				Workflow: wd.WorkflowID,
				Events:   slices.Clone(routeEvents),
			}
			routesByCommand[commandName] = append(routesByCommand[commandName], route)
		}
	}

	// Stable ordering for deterministic output.
	for commandName := range routesByCommand {
		sort.Slice(routesByCommand[commandName], func(i, j int) bool {
			return routesByCommand[commandName][i].Workflow < routesByCommand[commandName][j].Workflow
		})
	}

	return routesByCommand, mergedEvents
}

func buildCentralSlashCommandWorkflowYAML(routesByCommand map[string][]slashCommandRoute, mergedEvents map[string]map[string]bool) (string, error) {
	routesJSON, err := json.Marshal(routesByCommand)
	if err != nil {
		return "", fmt.Errorf("failed to marshal centralized slash-command routes: %w", err)
	}

	header := GenerateWorkflowHeader("", "pkg/workflow/central_slash_command_workflow.go", "")

	var b strings.Builder
	b.WriteString(header)
	b.WriteString(`name: "Agentic Slash Command Trigger"

on:
`)
	writeCentralSlashEventsYAML(&b, mergedEvents)
	b.WriteString(`
permissions:
  actions: write
  contents: read

jobs:
  route:
    runs-on: ubuntu-slim
    steps:
      - name: Checkout repository
        uses: ` + getActionPin("actions/checkout") + `

      - name: Route slash command
        uses: ` + getActionPin("actions/github-script") + `
        env:
          GH_AW_SLASH_ROUTING: '` + escapeSingleQuotedYAMLString(string(routesJSON)) + `'
        with:
          script: |
            const routeMap = JSON.parse(process.env.GH_AW_SLASH_ROUTING || "{}");
            const bodyByEvent = {
              issues: context.payload?.issue?.body ?? "",
              pull_request: context.payload?.pull_request?.body ?? "",
              issue_comment: context.payload?.comment?.body ?? "",
              pull_request_review_comment: context.payload?.comment?.body ?? "",
              discussion: context.payload?.discussion?.body ?? "",
              discussion_comment: context.payload?.comment?.body ?? "",
            };

            function eventIdentifier() {
              if (context.eventName !== "issue_comment") {
                return context.eventName;
              }
              return context.payload?.issue?.pull_request ? "pull_request_comment" : "issue_comment";
            }

            const text = bodyByEvent[context.eventName] ?? "";
            const firstWord = String(text).trim().split(/\s+/)[0] ?? "";
            if (!firstWord.startsWith("/")) {
              core.info("No slash command found at start of payload text; skipping dispatch.");
              return;
            }

            const commandName = firstWord.slice(1);
            const identifier = eventIdentifier();
            const routes = (routeMap[commandName] ?? []).filter(route => Array.isArray(route.events) && route.events.includes(identifier));
            if (routes.length === 0) {
              core.info("No centralized routes matched command '/" + commandName + "' for event '" + identifier + "'.");
              return;
            }

            const { setupGlobals } = require(process.env.GITHUB_WORKSPACE + "/actions/setup/js/setup_globals.cjs");
            setupGlobals(core, github, context, exec, io, getOctokit);
            const { buildAwContext } = require(process.env.GITHUB_WORKSPACE + "/actions/setup/js/aw_context.cjs");

            const ref = process.env.GITHUB_HEAD_REF ? "refs/heads/" + process.env.GITHUB_HEAD_REF : (process.env.GITHUB_REF || context.ref || "refs/heads/" + (context.payload?.repository?.default_branch || "main"));
            for (const route of routes) {
              const awContext = buildAwContext();
              awContext.command_name = commandName;
              await github.rest.actions.createWorkflowDispatch({
                owner: context.repo.owner,
                repo: context.repo.repo,
                workflow_id: route.workflow + ".lock.yml",
                ref,
                inputs: {
                  aw_context: JSON.stringify(awContext),
                },
              });
              core.info("Dispatched '" + route.workflow + "' for '/" + commandName + "'");
            }
`)
	return b.String(), nil
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
