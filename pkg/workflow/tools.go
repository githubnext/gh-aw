package workflow

import (
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"strings"
	"time"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/parser"
	"github.com/github/gh-aw/pkg/workflow/compilerenv"
	"github.com/goccy/go-yaml"
)

var toolsLog = logger.New("workflow:tools")

// applyDefaults applies default values for missing workflow sections
func (c *Compiler) applyDefaults(data *WorkflowData, markdownPath string) error {
	toolsLog.Printf("Applying defaults to workflow: name=%s, path=%s", data.Name, markdownPath)
	defer cacheWorkflowDefaultValues(data)

	isCommandTrigger, isLabelCommandTrigger := detectCommandTriggers(data, markdownPath)
	if data.On == "" {
		if err := c.applyDefaultOnSection(data, isCommandTrigger, isLabelCommandTrigger); err != nil {
			return err
		}
	}

	if c.trialMode && c.hasIssueTrigger(data.On) {
		data.On = c.injectWorkflowDispatchForIssue(data.On)
	}

	c.applyWorkflowExecutionDefaults(data, isCommandTrigger || isLabelCommandTrigger)
	return c.applyPermissionDefaults(data)
}

func cacheWorkflowDefaultValues(data *WorkflowData) {
	data.CachedPermissions = NewPermissionsParser(data.Permissions).ToPermissions()
	data.CachedPermissionScopeNamesErr = ValidatePermissionScopeNames(data.Permissions)
	data.CachedPermissionScopeNamesSet = true
	data.ConcurrencyGroupExpr = extractConcurrencyGroupFromYAML(data.Concurrency)
	if data.ConcurrencyGroupExpr != "" {
		data.CachedConcurrencyGroupExprErr = validateConcurrencyGroupExpression(data.ConcurrencyGroupExpr)
	}
	data.CachedConcurrencyGroupExprSet = true
	if data.ParsedTools != nil && data.ParsedTools.GitHub != nil {
		data.CachedParsedToolsets = ParseGitHubToolsets(data.ParsedTools.GitHub.GetToolsets())
	}
}

func detectCommandTriggers(data *WorkflowData, markdownPath string) (bool, bool) {
	if data.On != "" {
		return false, false
	}
	if len(data.Command) > 0 {
		return true, false
	}
	if len(data.LabelCommand) > 0 {
		return false, true
	}
	return detectCommandTriggersFromFrontmatter(markdownPath)
}

func detectCommandTriggersFromFrontmatter(markdownPath string) (bool, bool) {
	content, err := os.ReadFile(markdownPath)
	if err != nil {
		return false, false
	}
	result, err := parser.ExtractFrontmatterFromContent(string(content))
	if err != nil {
		return false, false
	}
	onValue, exists := result.Frontmatter["on"]
	if !exists {
		return false, false
	}
	onMap, ok := onValue.(map[string]any)
	if !ok {
		return false, false
	}
	if _, hasSlashCommand := onMap["slash_command"]; hasSlashCommand {
		return true, false
	}
	if _, hasCommand := onMap["command"]; hasCommand {
		return true, false
	}
	if _, hasLabelCommand := onMap["label_command"]; hasLabelCommand {
		return false, true
	}
	return false, false
}

func (c *Compiler) applyDefaultOnSection(data *WorkflowData, isCommandTrigger, isLabelCommandTrigger bool) error {
	switch {
	case isCommandTrigger:
		return c.configureCommandTriggerDefaults(data)
	case isLabelCommandTrigger:
		return c.configureLabelCommandTriggerDefaults(data)
	default:
		data.On = defaultWorkflowOnSection()
		return nil
	}
}

func (c *Compiler) configureCommandTriggerDefaults(data *WorkflowData) error {
	toolsLog.Print("Workflow is command trigger, configuring command events")
	commandEventsMap, filteredEvents := buildCommandTriggerEventsMap(data)
	mergeLabelCommandEventsIntoCommandMap(commandEventsMap, data)
	if err := c.setOnFromEventsMap(data, commandEventsMap, "command events"); err != nil {
		return err
	}
	return c.applyCommandTriggerCondition(data, filteredEvents)
}

func buildCommandTriggerEventsMap(data *WorkflowData) (map[string]any, []CommentEventMapping) {
	commandEventsMap := make(map[string]any)
	var filteredEvents []CommentEventMapping
	if data.CommandCentralized {
		if len(data.CommandOtherEvents) > 0 {
			maps.Copy(commandEventsMap, data.CommandOtherEvents)
		}
		if _, hasWorkflowDispatch := commandEventsMap["workflow_dispatch"]; !hasWorkflowDispatch {
			commandEventsMap["workflow_dispatch"] = nil
		}
		return commandEventsMap, filteredEvents
	}
	filteredEvents = FilterCommentEvents(data.CommandEvents)
	for _, event := range MergeEventsForYAML(filteredEvents) {
		commandEventsMap[event.EventName] = map[string]any{"types": event.Types}
	}
	if len(data.CommandOtherEvents) > 0 {
		maps.Copy(commandEventsMap, data.CommandOtherEvents)
	}
	return commandEventsMap, filteredEvents
}

func mergeLabelCommandEventsIntoCommandMap(commandEventsMap map[string]any, data *WorkflowData) {
	if len(data.LabelCommand) == 0 || data.CommandCentralized {
		return
	}
	for _, eventName := range FilterLabelCommandEvents(data.LabelCommandEvents) {
		if existingAny, ok := commandEventsMap[eventName]; ok {
			mergeLabeledTypeIntoEvent(existingAny)
		} else {
			commandEventsMap[eventName] = map[string]any{"types": []any{"labeled"}}
		}
	}
}

func mergeLabeledTypeIntoEvent(existingAny any) {
	existingMap, ok := existingAny.(map[string]any)
	if !ok {
		return
	}
	switch t := existingMap["types"].(type) {
	case []string:
		newTypes := make([]any, len(t)+1)
		for i, s := range t {
			newTypes[i] = s
		}
		newTypes[len(t)] = "labeled"
		existingMap["types"] = newTypes
	case []any:
		existingMap["types"] = append(t, "labeled")
	}
}

func (c *Compiler) setOnFromEventsMap(data *WorkflowData, eventsMap map[string]any, label string) error {
	mergedEventsYAML, err := yaml.MarshalWithOptions(map[string]any{"on": eventsMap}, yaml.IndentSequence(true))
	if err != nil {
		return fmt.Errorf("failed to marshal %s: %w", label, err)
	}
	yamlStr := strings.TrimSuffix(string(mergedEventsYAML), "\n")
	yamlStr = parser.QuoteCronExpressions(yamlStr)
	yamlStr = c.commentOutProcessedFieldsInOnSection(yamlStr, map[string]any{})
	data.On = yamlStr
	return nil
}

func (c *Compiler) applyCommandTriggerCondition(data *WorkflowData, _ []CommentEventMapping) error {
	if !data.CommandCentralized {
		return c.applyDecentralizedCommandCondition(data)
	}
	if data.If == "" && len(data.LabelCommand) > 0 {
		labelConditionTree, err := buildDispatchLabelCommandCondition(data.LabelCommand, data.LabelCommandEvents)
		if err != nil {
			return fmt.Errorf("failed to build label-command condition: %w", err)
		}
		data.If = RenderCondition(labelConditionTree)
	}
	return nil
}

func (c *Compiler) applyDecentralizedCommandCondition(data *WorkflowData) error {
	hasOtherEvents := len(data.CommandOtherEvents) > 0
	commandConditionTree, err := buildEventAwareCommandCondition(data.Command, data.CommandEvents, hasOtherEvents)
	if err != nil {
		return fmt.Errorf("failed to build command condition: %w", err)
	}
	if data.If != "" {
		return nil
	}
	if len(data.LabelCommand) == 0 {
		data.If = RenderCondition(commandConditionTree)
		return nil
	}
	labelConditionTree, err := buildLabelCommandCondition(data.LabelCommand, data.LabelCommandEvents, false)
	if err != nil {
		return fmt.Errorf("failed to build combined label-command condition: %w", err)
	}
	data.If = RenderCondition(&OrNode{Left: commandConditionTree, Right: labelConditionTree})
	return nil
}

func (c *Compiler) configureLabelCommandTriggerDefaults(data *WorkflowData) error {
	toolsLog.Print("Workflow is label-command trigger, configuring label events")
	labelEventsMap := buildLabelCommandEventsMap(data)
	if err := c.setOnFromEventsMap(data, labelEventsMap, "label-command events"); err != nil {
		return err
	}
	return c.applyLabelCommandCondition(data)
}

func buildLabelCommandEventsMap(data *WorkflowData) map[string]any {
	labelEventsMap := make(map[string]any)
	if data.LabelCommandDecentralized {
		if len(data.LabelCommandOtherEvents) > 0 {
			maps.Copy(labelEventsMap, data.LabelCommandOtherEvents)
		}
		if ensureWorkflowDispatchItemNumberInput(labelEventsMap) {
			data.HasDispatchItemNumber = true
		}
		return labelEventsMap
	}
	for _, eventName := range FilterLabelCommandEvents(data.LabelCommandEvents) {
		labelEventsMap[eventName] = map[string]any{"types": []any{"labeled"}}
	}
	if ensureWorkflowDispatchItemNumberInput(labelEventsMap) {
		data.HasDispatchItemNumber = true
	}
	mergeLabelCommandOtherEvents(labelEventsMap, data.LabelCommandOtherEvents)
	return labelEventsMap
}

func mergeLabelCommandOtherEvents(labelEventsMap, otherEvents map[string]any) {
	for eventKey, eventVal := range otherEvents {
		if existing, exists := labelEventsMap[eventKey]; exists {
			mergeLabelCommandEventConfig(existing, eventVal)
		} else {
			labelEventsMap[eventKey] = eventVal
		}
	}
}

func mergeLabelCommandEventConfig(existing, eventVal any) {
	existingMap, _ := existing.(map[string]any)
	userMap, _ := eventVal.(map[string]any)
	if existingMap == nil || userMap == nil {
		return
	}
	existingTypes, _ := existingMap["types"].([]any)
	userTypes, _ := userMap["types"].([]any)
	merged := make([]any, 0, safeAllocationCapacity(len(existingTypes), len(userTypes)))
	merged = append(merged, existingTypes...)
	merged = append(merged, userTypes...)
	existingMap["types"] = merged
	for k, v := range userMap {
		if k != "types" {
			existingMap[k] = v
		}
	}
}

func (c *Compiler) applyLabelCommandCondition(data *WorkflowData) error {
	hasOtherEvents := len(data.LabelCommandOtherEvents) > 0
	labelConditionTree, err := buildLabelCommandCondition(data.LabelCommand, data.LabelCommandEvents, hasOtherEvents)
	if err != nil {
		return fmt.Errorf("failed to build label-command condition: %w", err)
	}
	if data.If == "" {
		if data.LabelCommandDecentralized {
			labelConditionTree, err = buildDispatchLabelCommandCondition(data.LabelCommand, data.LabelCommandEvents)
			if err != nil {
				return fmt.Errorf("failed to build decentralized label-command condition: %w", err)
			}
		}
		data.If = RenderCondition(labelConditionTree)
	}
	return nil
}

func defaultWorkflowOnSection() string {
	return `on:
  # Start either every 10 minutes, or when some kind of human event occurs.
  # Because of the implicit "concurrency" section, only one instance of this
  # workflow will run at a time.
  schedule:
    - cron: "0/10 * * * *"
  issues:
    types: [opened, edited, closed]
  issue_comment:
    types: [created, edited]
  pull_request:
    types: [opened, edited, closed]
  push:
    branches:
      - main
  workflow_dispatch:`
}

func (c *Compiler) applyWorkflowExecutionDefaults(data *WorkflowData, isCommandOrLabelTrigger bool) {
	data.Concurrency = GenerateConcurrencyConfig(data, isCommandOrLabelTrigger)
	if data.RunName == "" {
		data.RunName = fmt.Sprintf(`run-name: "%s"`, data.Name)
	}
	if data.TimeoutMinutes == "" {
		defaultTimeoutMinutes := compilerenv.ResolveDefaultTimeoutMinutes(int(constants.DefaultAgenticWorkflowTimeout / time.Minute))
		data.TimeoutMinutes = fmt.Sprintf("timeout-minutes: %d", defaultTimeoutMinutes)
	}
	if data.RunsOn == "" {
		data.RunsOn = "runs-on: ubuntu-latest"
	}
	data.Tools = c.applyDefaultTools(data.Tools, data.SafeOutputs, data.SandboxConfig, data.NetworkPermissions)
	data.ParsedTools = NewTools(data.Tools)
}

func (c *Compiler) applyPermissionDefaults(data *WorkflowData) error {
	if data.Permissions == "permissions: {}" {
		return nil
	}
	if data.Permissions == "" {
		perms := NewPermissionsContentsRead()
		yaml := perms.RenderToYAML()
		lines := strings.Split(yaml, "\n")
		for i := 1; i < len(lines); i++ {
			if strings.HasPrefix(lines[i], "      ") {
				lines[i] = "  " + lines[i][6:]
			}
		}
		data.Permissions = strings.Join(lines, "\n")
	}
	return nil
}

func ensureWorkflowDispatchItemNumberInput(eventsMap map[string]any) bool {
	dispatchAny, hasDispatch := eventsMap["workflow_dispatch"]
	if !hasDispatch || dispatchAny == nil {
		eventsMap["workflow_dispatch"] = map[string]any{
			"inputs": map[string]any{
				"item_number": map[string]any{
					"description": "The number of the issue, pull request, or discussion",
					"required":    false,
					"default":     "",
					"type":        "string",
				},
			},
		}
		return true
	}

	dispatchMap, ok := dispatchAny.(map[string]any)
	if !ok {
		toolsLog.Print("Skipping workflow_dispatch item_number injection: workflow_dispatch is not a map")
		return false
	}

	inputsAny, hasInputs := dispatchMap["inputs"]
	if !hasInputs || inputsAny == nil {
		dispatchMap["inputs"] = map[string]any{}
		inputsAny = dispatchMap["inputs"]
	}
	inputsMap, ok := inputsAny.(map[string]any)
	if !ok {
		toolsLog.Print("Skipping workflow_dispatch item_number injection: workflow_dispatch.inputs is not a map")
		return false
	}

	if _, hasItemNumber := inputsMap["item_number"]; !hasItemNumber {
		inputsMap["item_number"] = map[string]any{
			"description": "The number of the issue, pull request, or discussion",
			"required":    false,
			"default":     "",
			"type":        "string",
		}
	}
	return true
}

// mergeToolsAndMCPServers merges tools, mcp-servers, and included tools
func (c *Compiler) mergeToolsAndMCPServers(topTools, mcpServers map[string]any, includedTools string) (map[string]any, error) {
	toolsLog.Printf("Merging tools and MCP servers: topTools=%d, mcpServers=%d", len(topTools), len(mcpServers))

	// Start with top-level tools
	result := topTools
	if result == nil {
		result = make(map[string]any)
	}

	// Add MCP servers to the tools collection
	maps.Copy(result, mcpServers)

	// Merge included tools
	return c.MergeTools(result, includedTools)
}

// mergeRuntimes merges runtime configurations from frontmatter and imports
func mergeRuntimes(topRuntimes map[string]any, importedRuntimesJSON string) (map[string]any, error) {
	toolsLog.Printf("Merging runtimes: topRuntimes=%d", len(topRuntimes))
	result := make(map[string]any)

	// Start with top-level runtimes
	maps.Copy(result, topRuntimes)

	// Merge imported runtimes (newline-separated JSON objects)
	if importedRuntimesJSON != "" {
		lines := strings.SplitSeq(strings.TrimSpace(importedRuntimesJSON), "\n")
		for line := range lines {
			line = strings.TrimSpace(line)
			if line == "" || line == "{}" {
				continue
			}

			var importedRuntimes map[string]any
			if err := json.Unmarshal([]byte(line), &importedRuntimes); err != nil {
				return nil, fmt.Errorf("failed to parse imported runtimes JSON: %w", err)
			}

			// Merge imported runtimes - later imports override earlier ones
			maps.Copy(result, importedRuntimes)
		}
	}

	toolsLog.Printf("Merged %d total runtimes", len(result))
	return result, nil
}

// hasIssueTrigger checks if the workflow has an issue trigger in its 'on' section
func (c *Compiler) hasIssueTrigger(onSection string) bool {
	hasIssue := strings.Contains(onSection, "issues:") ||
		strings.Contains(onSection, "issue:") ||
		strings.Contains(onSection, "issue_comment:")
	toolsLog.Printf("Checking for issue trigger: has_issue=%t", hasIssue)
	return hasIssue
}

// injectWorkflowDispatchForIssue adds workflow_dispatch trigger with issue_number input
func (c *Compiler) injectWorkflowDispatchForIssue(onSection string) string {
	toolsLog.Print("Injecting workflow_dispatch trigger for issue workflows")
	// Parse the existing on section to understand its structure
	var onData map[string]any
	if err := yaml.Unmarshal([]byte(onSection), &onData); err != nil {
		// If parsing fails, append workflow_dispatch manually
		return onSection + "\n  workflow_dispatch:\n    inputs:\n      issue_number:\n        description: 'Issue number for trial mode'\n        required: true\n        type: string"
	}

	// Get the 'on' section
	if onMap, exists := onData["on"]; exists {
		if triggers, ok := onMap.(map[string]any); ok {
			// Add workflow_dispatch with issue_number input
			triggers["workflow_dispatch"] = map[string]any{
				"inputs": map[string]any{
					"issue_number": map[string]any{
						"description": "Issue number for trial mode",
						"required":    true,
						"type":        "string",
					},
				},
			}

			// Convert back to YAML
			updatedOnData := map[string]any{"on": triggers}
			if yamlBytes, err := yaml.Marshal(updatedOnData); err == nil {
				yamlStr := string(yamlBytes)
				// Keep "on" quoted as it's a YAML boolean keyword
				return strings.TrimSuffix(yamlStr, "\n")
			}
		}
	}

	// Fallback: append workflow_dispatch manually
	return onSection + "\n  workflow_dispatch:\n    inputs:\n      issue_number:\n        description: 'Issue number for trial mode'\n        required: true\n        type: string"
}

// replaceIssueNumberReferences replaces github.event.issue.number with inputs.issue_number in YAML content
func (c *Compiler) replaceIssueNumberReferences(yamlContent string) string {
	// Replace all occurrences of github.event.issue.number with inputs.issue_number
	return strings.ReplaceAll(yamlContent, "github.event.issue.number", "inputs.issue_number")
}

// applyDefaultTools adds default read-only GitHub MCP tools, creating github tool if not present
func (c *Compiler) applyDefaultTools(tools map[string]any, safeOutputs *SafeOutputsConfig, sandboxConfig *SandboxConfig, networkPermissions *NetworkPermissions) map[string]any {
	toolsLog.Printf("Applying default tools: existingToolCount=%d", len(tools))
	if tools == nil {
		tools = make(map[string]any)
	}
	applyDefaultGitHubTool(tools)
	applySandboxDefaultTools(tools, sandboxConfig, networkPermissions)
	applyGitCommandDefaultTools(tools, safeOutputs)
	applyBashDefaultCommands(tools, safeOutputs)
	return tools
}

func applyDefaultGitHubTool(tools map[string]any) {
	githubTool := tools["github"]
	if githubTool == false {
		delete(tools, "github")
		return
	}
	githubConfig := map[string]any{}
	if toolConfig, ok := githubTool.(map[string]any); ok {
		maps.Copy(githubConfig, toolConfig)
	}
	parsedConfig := parseGitHubTool(githubTool)
	if parsedConfig != nil && len(parsedConfig.Allowed) > 0 {
		existingAllowed := make([]any, 0, len(parsedConfig.Allowed))
		for _, tool := range parsedConfig.Allowed {
			existingAllowed = append(existingAllowed, string(tool))
		}
		githubConfig["allowed"] = existingAllowed
	}
	tools["github"] = githubConfig
}

func applySandboxDefaultTools(tools map[string]any, sandboxConfig *SandboxConfig, networkPermissions *NetworkPermissions) {
	if !isSandboxEnabled(sandboxConfig, networkPermissions) {
		return
	}
	toolsLog.Print("Sandbox enabled, applying default edit and bash tools")
	if _, exists := tools["edit"]; !exists {
		tools["edit"] = true
		toolsLog.Print("Added edit tool (sandbox enabled)")
	}
	if _, exists := tools["bash"]; !exists {
		tools["bash"] = []any{"*"}
		toolsLog.Print("Added bash tool with wildcard (sandbox enabled)")
	}
}

func applyGitCommandDefaultTools(tools map[string]any, safeOutputs *SafeOutputsConfig) {
	if safeOutputs == nil || !needsGitCommands(safeOutputs) {
		return
	}
	if _, exists := tools["edit"]; !exists {
		tools["edit"] = nil
	}
	gitCommands := defaultGitCommands()
	if _, exists := tools["bash"]; !exists {
		tools["bash"] = gitCommands
		return
	}
	mergeGitCommandsIntoBashTool(tools, gitCommands)
}

func defaultGitCommands() []any {
	return []any{
		"git checkout:*",
		"git branch:*",
		"git switch:*",
		"git add:*",
		"git rm:*",
		"git commit:*",
		"git merge:*",
		"git status",
	}
}

func mergeGitCommandsIntoBashTool(tools map[string]any, gitCommands []any) {
	existingBash := tools["bash"]
	if existingCommands, ok := existingBash.([]any); ok {
		merged, complete := mergeGitCommands(existingCommands, gitCommands)
		if !complete {
			tools["bash"] = merged
		}
	} else if existingBash == false {
		toolsLog.Print("Overriding bash: false with git commands (required for PR operations)")
		tools["bash"] = gitCommands
	} else if existingBash == nil {
		_ = existingBash
	}
}

func mergeGitCommands(existingCommands []any, gitCommands []any) ([]any, bool) {
	existingSet := make(map[string]struct{})
	for _, cmd := range existingCommands {
		if cmdStr, ok := cmd.(string); ok {
			existingSet[cmdStr] = struct{}{}
			if cmdStr == ":*" || cmdStr == "*" {
				return existingCommands, true
			}
		}
	}
	newCommands := append([]any(nil), existingCommands...)
	for _, gitCmd := range gitCommands {
		if gitCmdStr, ok := gitCmd.(string); ok {
			if _, ok := existingSet[gitCmdStr]; !ok {
				newCommands = append(newCommands, gitCmd)
			}
		}
	}
	return newCommands, false
}

func applyBashDefaultCommands(tools map[string]any, safeOutputs *SafeOutputsConfig) {
	bashTool, exists := tools["bash"]
	if !exists {
		return
	}
	switch bashTool := bashTool.(type) {
	case nil:
		if safeOutputs == nil || !needsGitCommands(safeOutputs) {
			tools["bash"] = defaultBashCommands()
		}
	case bool:
		if bashTool {
			tools["bash"] = []any{"*"}
		} else {
			delete(tools, "bash")
		}
	case []any:
		if len(bashTool) > 0 {
			tools["bash"] = mergeDefaultBashCommands(bashTool)
		}
	}
}

func defaultBashCommands() []any {
	defaultCommands := make([]any, len(constants.DefaultBashTools))
	for i, cmd := range constants.DefaultBashTools {
		defaultCommands[i] = cmd
	}
	return defaultCommands
}

func mergeDefaultBashCommands(bashArray []any) []any {
	existingCommands := make(map[string]struct{})
	for _, cmd := range bashArray {
		if cmdStr, ok := cmd.(string); ok {
			existingCommands[cmdStr] = struct{}{}
		}
	}
	var mergedCommands []any
	for _, cmd := range constants.DefaultBashTools {
		if _, ok := existingCommands[cmd]; !ok {
			mergedCommands = append(mergedCommands, cmd)
		}
	}
	mergedCommands = append(mergedCommands, bashArray...)
	return mergedCommands
}
