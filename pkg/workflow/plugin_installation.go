package workflow

import (
	"fmt"
	"path"
	"strings"

	"github.com/github/gh-aw/pkg/logger"
)

var pluginInstallationLog = logger.New("workflow:plugin_installation")

// GetPluginInstallationSteps installs pinned Agent Plugins through the Copilot CLI.
func (e *CopilotEngine) GetPluginInstallationSteps(workflowData *WorkflowData) []GitHubActionStep {
	commandName := "copilot"
	if workflowData != nil && workflowData.EngineConfig != nil && workflowData.EngineConfig.Command != "" {
		commandName = workflowData.EngineConfig.Command
	}
	return generatePluginInstallationSteps(workflowData, commandName)
}

func generatePluginInstallationSteps(workflowData *WorkflowData, commandName string) []GitHubActionStep {
	if workflowData == nil || len(workflowData.Plugins) == 0 {
		return nil
	}

	steps := make([]GitHubActionStep, 0, len(workflowData.Plugins)*2)
	checkoutAction := getActionPinForData("actions/checkout", workflowData)
	for i, plugin := range workflowData.Plugins {
		parsed := parseSkillRefSpec(plugin)
		if !parsed.isRemote || parsed.ref == "" {
			pluginInstallationLog.Printf("Skipping invalid plugin reference after validation: %q", plugin)
			continue
		}

		repoParts := strings.Split(parsed.repoPath, "/")
		repository := strings.Join(repoParts[:2], "/")
		pluginSubpath := strings.Join(repoParts[2:], "/")
		checkoutPath := fmt.Sprintf(".gh-aw-plugins/plugin-%d", i)
		installPath := path.Join(checkoutPath, pluginSubpath)

		steps = append(steps, GitHubActionStep{
			"      - name: Checkout agent plugin " + parsed.repoPath,
			"        uses: " + checkoutAction,
			"        with:",
			"          repository: " + repository,
			"          ref: " + parsed.ref,
			"          path: " + checkoutPath,
			"          persist-credentials: false",
		})

		installCommand := shellJoinArgs([]string{commandName, "plugin", "install", "./" + installPath})
		installStep := []string{"      - name: Install agent plugin " + parsed.repoPath}
		steps = append(steps, FormatStepWithCommandAndEnv(installStep, installCommand, nil))
	}

	return steps
}
