package workflow

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/github/gh-aw/pkg/logger"
)

func computeEngineToolsCore(tools map[string]any, toolsLog *logger.Logger, engineName string) []string {
	toolsCore := []string{
		"glob",
		"grep_search",
		"list_directory",
		"read_file",
		"read_many_files",
	}

	if tools == nil {
		return toolsCore
	}

	if bashConfig, hasBash := tools["bash"]; hasBash {
		toolsCore = appendBashToolsCore(toolsCore, bashConfig, toolsLog)
	}

	if _, hasEdit := tools["edit"]; hasEdit {
		toolsLog.Printf("edit → replace, write_file (%s)", engineName)
		toolsCore = append(toolsCore, "replace", "write_file")
	}

	if _, hasWebFetch := tools["web-fetch"]; hasWebFetch {
		toolsLog.Printf("web-fetch → web_fetch (%s)", engineName)
		toolsCore = append(toolsCore, "web_fetch")
	}

	sort.Strings(toolsCore)
	return toolsCore
}

func appendBashToolsCore(toolsCore []string, bashConfig any, toolsLog *logger.Logger) []string {
	bashCommands, ok := bashConfig.([]any)
	if !ok || len(bashCommands) == 0 {
		toolsLog.Print("bash (no specific commands) → run_shell_command")
		return append(toolsCore, "run_shell_command")
	}

	var specific []string
	for _, cmd := range bashCommands {
		cmdStr, ok := cmd.(string)
		if !ok {
			continue
		}
		if cmdStr == "*" || cmdStr == ":*" {
			toolsLog.Print("bash wildcard → run_shell_command")
			return append(toolsCore, "run_shell_command")
		}
		normalized, _ := normalizeBashCommand(cmdStr)
		entry := fmt.Sprintf("run_shell_command(%s)", normalized)
		toolsLog.Printf("bash %q → %s", cmdStr, entry)
		specific = append(specific, entry)
	}
	return append(toolsCore, specific...)
}

func generateEngineSettingsStep(
	workflowData *WorkflowData,
	toolsLog *logger.Logger,
	engineName, settingsDir, baseConfigEnvVar, stepName string,
) GitHubActionStep {
	toolsLog.Printf("Generating %s settings step for: %s", engineName, workflowData.Name)

	tools := workflowData.Tools
	if tools == nil {
		tools = make(map[string]any)
	}
	workflowDataWithEffectiveTools := *workflowData
	workflowDataWithEffectiveTools.Tools = tools
	tools = withMountedCLIShellCommandsInRestrictedBash(&workflowDataWithEffectiveTools)

	toolsCore := computeEngineToolsCore(tools, toolsLog, engineName)
	toolsLog.Printf("tools.core entries: %d", len(toolsCore))

	config := map[string]any{
		"context": map[string]any{
			"includeDirectories": []string{"/tmp/"},
		},
		"tools": map[string]any{
			"core": toolsCore,
		},
	}

	configJSON, err := json.Marshal(config)
	if err != nil {
		toolsLog.Printf("ERROR: Failed to marshal %s settings: %v", engineName, err)
		configJSON = []byte(`{"context":{"includeDirectories":["/tmp/"]},"tools":{"core":[]}}`)
	}

	command := fmt.Sprintf(`mkdir -p "$GITHUB_WORKSPACE/%s"
SETTINGS="$GITHUB_WORKSPACE/%s/settings.json"
BASE_CONFIG="$%s"
if [ -f "$SETTINGS" ]; then
  MERGED=$(jq -n --argjson base "$BASE_CONFIG" --argjson existing "$(cat "$SETTINGS")" '$existing * $base')
  echo "$MERGED" > "$SETTINGS"
else
  echo "$BASE_CONFIG" > "$SETTINGS"
fi`, settingsDir, settingsDir, baseConfigEnvVar)

	stepLines := []string{
		"      - name: " + stepName,
	}
	env := map[string]string{
		baseConfigEnvVar: string(configJSON),
	}
	stepLines = FormatStepWithCommandAndEnv(stepLines, command, env)
	return GitHubActionStep(stepLines)
}
