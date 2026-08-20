package workflow

import (
	"fmt"
	"path"
	"regexp"
	"strings"

	"github.com/github/gh-aw/pkg/logger"
)

var pluginInstallationLog = logger.New("workflow:plugin_installation")

// pluginInstallSpec describes how an engine consumes checked-out Agent Plugins.
//
// Every engine with Agent Plugins support checks each plugin out at its pinned SHA.
// Engines whose CLI exposes a plugin installation command additionally run
// "<Command> <InstallArgs...> <plugin path>" for each plugin. Engines that discover
// plugins from a well-known workspace folder instead set Directory, and each plugin
// is staged into "<Directory>/<plugin name>".
type pluginInstallSpec struct {
	// Command is the engine CLI executable used for CLI-based installation.
	Command string
	// InstallArgs are the CLI arguments placed before the local plugin path.
	InstallArgs []string
	// Directory is the folder the engine scans for plugins. It is either
	// workspace-relative (for example ".kiro/powers") or home-relative
	// (for example "~/.cursor/plugins/local").
	Directory string
}

// pluginDirectoryRegexp restricts plugin staging directories to a safe character set so
// the generated shell commands cannot be manipulated through an engine definition.
var pluginDirectoryRegexp = regexp.MustCompile(`^(?:~/)?[A-Za-z0-9._-][A-Za-z0-9_./-]*$`)

// resolvePluginDirectory validates the declared plugin directory and expands a leading
// "~/" to "$HOME/" so the generated shell commands resolve it at runtime.
func resolvePluginDirectory(directory string) (string, bool) {
	if !pluginDirectoryRegexp.MatchString(directory) || strings.Contains(directory, "..") {
		return "", false
	}
	if rest, found := strings.CutPrefix(directory, "~/"); found {
		return "$HOME/" + rest, true
	}
	return directory, true
}

// pluginLocalPaths returns the workspace-relative directory of each checked-out plugin,
// in the same order as workflowData.Plugins. Invalid entries are skipped because they are
// rejected earlier by validatePlugins.
func pluginLocalPaths(workflowData *WorkflowData) []string {
	if workflowData == nil {
		return nil
	}
	paths := make([]string, 0, len(workflowData.Plugins))
	for i, plugin := range workflowData.Plugins {
		parsed := parseSkillRefSpec(plugin)
		if !parsed.isRemote || parsed.ref == "" {
			continue
		}
		paths = append(paths, pluginCheckoutSubpath(parsed, i))
	}
	return paths
}

// pluginCheckoutPath returns the workspace-relative checkout folder of the index-th plugin.
func pluginCheckoutPath(index int) string {
	return fmt.Sprintf(".gh-aw-plugins/plugin-%d", index)
}

// pluginCheckoutSubpath returns the workspace-relative path of the plugin directory
// inside its checkout folder.
func pluginCheckoutSubpath(parsed parsedSkillRefSpec, index int) string {
	repoParts := strings.Split(parsed.repoPath, "/")
	return path.Join(pluginCheckoutPath(index), strings.Join(repoParts[2:], "/"))
}

// GetPluginInstallationSteps checks out pinned Agent Plugins for the Claude engine.
// Claude Code loads plugin directories through the --plugin-dir flag added by
// appendClaudePluginArgs, so no CLI installation command is required.
func (e *ClaudeEngine) GetPluginInstallationSteps(workflowData *WorkflowData) []GitHubActionStep {
	return generatePluginInstallationSteps(workflowData, pluginInstallSpec{})
}

// GetPluginInstallationSteps installs pinned Agent Plugins through the Copilot CLI.
func (e *CopilotEngine) GetPluginInstallationSteps(workflowData *WorkflowData) []GitHubActionStep {
	commandName := "copilot"
	if workflowData != nil && workflowData.EngineConfig != nil && workflowData.EngineConfig.Command != "" {
		commandName = workflowData.EngineConfig.Command
	}
	return generatePluginInstallationSteps(workflowData, pluginInstallSpec{
		Command:     commandName,
		InstallArgs: []string{"plugin", "install"},
	})
}

func generatePluginInstallationSteps(workflowData *WorkflowData, spec pluginInstallSpec) []GitHubActionStep {
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
		checkoutPath := pluginCheckoutPath(i)
		installPath := pluginCheckoutSubpath(parsed, i)

		steps = append(steps, GitHubActionStep{
			"      - name: Checkout agent plugin " + parsed.repoPath,
			"        uses: " + checkoutAction,
			"        with:",
			"          repository: " + repository,
			"          ref: " + parsed.ref,
			"          path: " + checkoutPath,
			"          persist-credentials: false",
		})

		if spec.Directory != "" {
			stageDirectory, ok := resolvePluginDirectory(spec.Directory)
			if !ok {
				pluginInstallationLog.Printf("Skipping unsupported plugin directory: %q", spec.Directory)
			} else {
				targetPath := path.Join(stageDirectory, path.Base(parsed.repoPath))
				stageCommand := strings.Join([]string{
					fmt.Sprintf("mkdir -p %q", stageDirectory),
					fmt.Sprintf("rm -rf %q", targetPath),
					fmt.Sprintf("cp -R %q %q", "./"+installPath, targetPath),
				}, "\n")
				stageStep := []string{"      - name: Stage agent plugin " + parsed.repoPath}
				steps = append(steps, FormatStepWithCommandAndEnv(stageStep, stageCommand, nil))
			}
		}

		if spec.Command != "" && len(spec.InstallArgs) > 0 {
			installArgs := make([]string, 0, len(spec.InstallArgs)+2)
			installArgs = append(installArgs, spec.Command)
			installArgs = append(installArgs, spec.InstallArgs...)
			installArgs = append(installArgs, "./"+installPath)
			installCommand := shellJoinArgs(installArgs)
			installStep := []string{"      - name: Install agent plugin " + parsed.repoPath}
			steps = append(steps, FormatStepWithCommandAndEnv(installStep, installCommand, nil))
		}
	}

	return steps
}
