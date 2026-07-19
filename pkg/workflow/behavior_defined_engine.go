package workflow

import (
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/setutil"
	"github.com/goccy/go-yaml"
)

const (
	behaviorSecretStrategyUniversalLLMConsumer  = "universal-llm-consumer"
	behaviorProviderEnvModeUniversalLLMConsumer = "universal-llm-consumer"
	behaviorConfigMergeJSON                     = "json-merge"
)

var behaviorDefinedEngineLog = logger.New("workflow:behavior_defined_engine")

// BehaviorDefinedEngine is a declarative CodingAgentEngine built from an engine
// definition's behaviors block.
type BehaviorDefinedEngine struct {
	UniversalLLMConsumerEngine
	definition *EngineDefinition
}

var _ CodingAgentEngine = (*BehaviorDefinedEngine)(nil)

func NewBehaviorDefinedEngine(def *EngineDefinition) (*BehaviorDefinedEngine, error) {
	if def == nil {
		return nil, errors.New("engine definition is required")
	}
	if def.Behaviors == nil {
		return nil, fmt.Errorf("engine definition %q is missing behaviors", def.ID)
	}
	engine := &BehaviorDefinedEngine{
		UniversalLLMConsumerEngine: UniversalLLMConsumerEngine{
			BaseEngine: BaseEngine{
				id:               def.ID,
				displayName:      def.DisplayName,
				description:      def.Description,
				experimental:     def.Experimental,
				ghSkillAgentName: def.GHSkillAgentName,
				capabilities:     def.Behaviors.Capabilities.ToRuntimeCapabilities(),
			},
		},
		definition: def,
	}
	return engine, nil
}

func newBuiltinBehaviorDefinedEngine(id string) (*BehaviorDefinedEngine, error) {
	def, err := getBuiltinEngineDefinition(id)
	if err != nil {
		return nil, err
	}
	return NewBehaviorDefinedEngine(def)
}

func (e *BehaviorDefinedEngine) behavior() *EngineBehaviorDefinition {
	if e == nil || e.definition == nil {
		return nil
	}
	return e.definition.Behaviors
}

func (e *BehaviorDefinedEngine) usesUniversalLLMConsumer() bool {
	behavior := e.behavior()
	return behavior != nil && behavior.SecretStrategy == behaviorSecretStrategyUniversalLLMConsumer
}

func (e *BehaviorDefinedEngine) GetModelEnvVarName() string {
	behavior := e.behavior()
	if behavior == nil || behavior.Execution == nil {
		return ""
	}
	return behavior.Execution.ModelEnvVarName
}

func (e *BehaviorDefinedEngine) GetRequiredSecretNames(workflowData *WorkflowData) []string {
	if e.usesUniversalLLMConsumer() {
		return e.GetUniversalRequiredSecretNames(workflowData)
	}

	seen := make(map[string]struct{})
	var secrets []string
	addSecret := func(secret string) {
		if secret == "" || setutil.Contains(seen, secret) {
			return
		}
		seen[secret] = struct{}{}
		secrets = append(secrets, secret)
	}
	for _, binding := range e.definition.Auth {
		addSecret(binding.Secret)
	}
	for _, secret := range collectCommonMCPSecrets(workflowData) {
		addSecret(secret)
	}
	parsedTools, tools := extractToolsConfig(workflowData)
	if hasGitHubTool(parsedTools) {
		addSecret("GITHUB_MCP_SERVER_TOKEN")
	}
	for varName := range collectHTTPMCPHeaderSecrets(tools) {
		addSecret(varName)
	}
	return secrets
}

func (e *BehaviorDefinedEngine) GetSupportedEnvVarKeys() []string {
	behavior := e.behavior()
	if behavior == nil {
		return nil
	}
	if len(behavior.SupportedEnvVarKeys) > 0 {
		return behavior.SupportedEnvVarKeys
	}
	keys := make([]string, 0, len(e.definition.Auth))
	for _, binding := range e.definition.Auth {
		if binding.Secret != "" {
			keys = append(keys, binding.Secret)
		}
	}
	slices.Sort(keys)
	return slices.Compact(keys)
}

func (e *BehaviorDefinedEngine) GetSecretValidationStep(workflowData *WorkflowData) GitHubActionStep {
	behavior := e.behavior()
	if behavior == nil || behavior.Installation == nil {
		return GitHubActionStep{}
	}
	if e.usesUniversalLLMConsumer() {
		return e.GetUniversalSecretValidationStep(
			workflowData,
			e.definition.DisplayName,
			behavior.Installation.DocumentationURL,
		)
	}
	secrets := e.GetRequiredSecretNames(workflowData)
	if len(secrets) == 0 {
		return GitHubActionStep{}
	}
	return BuildDefaultSecretValidationStep(workflowData, secrets, e.definition.DisplayName, behavior.Installation.DocumentationURL)
}

func (e *BehaviorDefinedEngine) GetInstallationSteps(workflowData *WorkflowData) []GitHubActionStep {
	behavior := e.behavior()
	if behavior == nil || behavior.Installation == nil {
		return nil
	}
	if workflowData != nil && workflowData.EngineConfig != nil && workflowData.EngineConfig.Command != "" {
		return nil
	}

	install := behavior.Installation
	if install.PackageManager != "npm" {
		return nil
	}
	version := install.Version
	if workflowData != nil && workflowData.EngineConfig != nil && workflowData.EngineConfig.Version != "" {
		version = workflowData.EngineConfig.Version
	}

	npmSteps := GenerateNpmInstallSteps(
		install.PackageName,
		version,
		install.StepName,
		install.BinaryName,
		NPMInstallOptions{
			IncludeNodeSetup:  install.IncludeNodeSetup,
			RunInstallScripts: install.PostInstallScripts,
			CooldownEnabled:   install.Cooldown,
		},
	)
	if install.VerifyCommand != "" {
		npmSteps = append(npmSteps, GitHubActionStep{
			"      - name: " + install.VerifyStepName,
			"        run: " + install.VerifyCommand,
		})
	}
	return BuildNpmEngineInstallStepsWithAWF(npmSteps, workflowData)
}

func (e *BehaviorDefinedEngine) GetAgentManifestFiles() []string {
	behavior := e.behavior()
	if behavior == nil || behavior.Manifest == nil {
		return nil
	}
	return behavior.Manifest.Files
}

func (e *BehaviorDefinedEngine) GetAgentManifestPathPrefixes() []string {
	behavior := e.behavior()
	if behavior == nil || behavior.Manifest == nil {
		return nil
	}
	return behavior.Manifest.PathPrefixes
}

func (e *BehaviorDefinedEngine) RenderMCPConfig(sb *strings.Builder, tools map[string]any, mcpTools []string, workflowData *WorkflowData) error {
	behavior := e.behavior()
	if behavior == nil || behavior.MCP == nil || behavior.MCP.ConfigPath == "" {
		return nil
	}
	return renderDefaultJSONMCPConfig(sb, tools, mcpTools, workflowData, behavior.MCP.ConfigPath)
}

// harnessScriptHeredocDelimiter is the shell heredoc delimiter used when writing
// the harness script to disk. It is intentionally long and project-specific so that
// it is extremely unlikely to appear at the start of a line in any JavaScript harness.
const harnessScriptHeredocDelimiter = "GHAW_HARNESS_SCRIPT_3c7b9f1a_EOF"

// harnessScriptFilename returns the filename (not path) for the engine's harness script.
func (e *BehaviorDefinedEngine) harnessScriptFilename() string {
	return e.GetID() + "_harness.cjs"
}

// buildHarnessWriteStep generates a GitHub Actions step that writes the behavior-defined
// engine's harness-script content to ${RUNNER_TEMP}/gh-aw/actions/<engine-id>_harness.cjs
// so it can be executed as a Node.js harness during the engine execution step.
// Returns nil and logs a warning if the harness script contains the heredoc delimiter,
// which would break the generated shell command.
func (e *BehaviorDefinedEngine) buildHarnessWriteStep() GitHubActionStep {
	behavior := e.behavior()
	if behavior == nil || behavior.HarnessScript == "" {
		return nil
	}
	// Safety check: if the harness script contains the heredoc delimiter at the start
	// of any line, the heredoc would be terminated prematurely. Detect this at
	// compile time and log a clear error rather than generating a broken step.
	if strings.Contains(behavior.HarnessScript, "\n"+harnessScriptHeredocDelimiter) ||
		strings.HasPrefix(behavior.HarnessScript, harnessScriptHeredocDelimiter) {
		behaviorDefinedEngineLog.Printf(
			"WARNING: engine %q harness-script contains heredoc delimiter %q; harness write step skipped",
			e.GetID(), harnessScriptHeredocDelimiter,
		)
		return nil
	}
	filename := e.harnessScriptFilename()
	command := fmt.Sprintf(
		"mkdir -p %[1]s\ncat <<'%[4]s' > %[1]s/%[2]s\n%[3]s\n%[4]s\nchmod 755 %[1]s/%[2]s",
		SetupActionDestinationShell,
		filename,
		behavior.HarnessScript,
		harnessScriptHeredocDelimiter,
	)
	stepLines := []string{"      - name: Write " + e.GetDisplayName() + " harness script"}
	stepLines = FormatStepWithCommandAndEnv(stepLines, command, nil)
	return GitHubActionStep(stepLines)
}

func (e *BehaviorDefinedEngine) GetExecutionSteps(workflowData *WorkflowData, logFile string) []GitHubActionStep {
	behavior := e.behavior()
	if behavior == nil || behavior.Execution == nil {
		return nil
	}

	var steps []GitHubActionStep
	if configStep := e.buildConfigFileStep(); len(configStep) > 0 {
		steps = append(steps, configStep)
	}
	if harnessStep := e.buildHarnessWriteStep(); len(harnessStep) > 0 {
		steps = append(steps, harnessStep)
	}

	exec := behavior.Execution
	commandName := exec.CommandName
	if workflowData != nil && workflowData.EngineConfig != nil && workflowData.EngineConfig.Command != "" {
		commandName = workflowData.EngineConfig.Command
	}

	var engineCommand string
	if behavior.HarnessScript != "" {
		// Harness execution: the harness is responsible for reading GH_AW_PROMPT and
		// spawning the engine CLI. Pass the shell-escaped command name and configured args
		// so the harness can forward them or use them to build the full command.
		harnessArgs := []string{shellEscapeArg(commandName)}
		if len(exec.Args) > 0 {
			harnessArgs = append(harnessArgs, shellJoinArgs(exec.Args))
		}
		harnessPath := SetupActionDestinationShell + "/" + e.harnessScriptFilename()
		engineCommand = fmt.Sprintf("%s %s %s", nodeRuntimeResolutionCommand, harnessPath, strings.Join(harnessArgs, " "))
	} else {
		commandParts := []string{commandName}
		if len(exec.Args) > 0 {
			commandParts = append(commandParts, shellJoinArgs(exec.Args))
		}
		if modelFragment := e.modelFlagFragment(exec, workflowData); modelFragment != "" {
			commandParts = append(commandParts, modelFragment)
		}
		if mcpFragment := e.mcpFlagFragment(exec, workflowData); mcpFragment != "" {
			commandParts = append(commandParts, mcpFragment)
		}
		commandParts = append(commandParts, fmt.Sprintf(`"$(cat %s)"`, constants.AwPromptsFile))
		engineCommand = strings.Join(commandParts, " ")
		engineCommand = getWorkspaceCommandPrefixFor(workflowData.EngineConfig) + engineCommand
	}

	firewallEnabled := isFirewallEnabled(workflowData)
	// harness-script requires the AWF API proxy sidecar (/reflect) to dynamically
	// configure the agentic engine at runtime. Force AWF execution when a harness is
	// present unless the sandbox has been explicitly disabled via sandbox.agent: false,
	// which also prevents the /reflect endpoint from being available.
	if behavior.HarnessScript != "" && !isFirewallDisabledBySandboxAgent(workflowData) {
		firewallEnabled = true
	}
	var command string
	if firewallEnabled {
		command = e.buildFirewallCommand(exec, workflowData, logFile, engineCommand)
	} else if exec.WriteTimestamp {
		command = fmt.Sprintf("set -o pipefail\nexport no_proxy=\"${NO_PROXY:-}\"\nprintf '%%s' \"$(date +%%s%%3N)\" > %s\n%s 2>&1 | tee -a %s",
			AgentCLIStartMsPath, engineCommand, logFile)
	} else {
		command = fmt.Sprintf("set -o pipefail\nexport no_proxy=\"${NO_PROXY:-}\"\n%s 2>&1 | tee -a %s", engineCommand, logFile)
	}

	env := map[string]string{
		"GH_AW_PROMPT":     constants.AwPromptsFile,
		"GITHUB_WORKSPACE": "${{ github.workspace }}",
		"RUNNER_TEMP":      "${{ runner.temp }}",
		// Set NO_PROXY so that the AWF agent's HTTP client skips the squid proxy
		// for local endpoints. The lowercase no_proxy variant is exported inside
		// the run script rather than as a YAML env key because GitHub's workflow
		// parser rejects case-insensitive duplicate env keys (NO_PROXY/no_proxy),
		// which causes workflow_dispatch to fail with "failed to parse workflow".
		"NO_PROXY": constants.AWFNoProxyHosts,
	}
	injectWorkflowCallNetworkAllowedEnv(env, workflowData)

	// Apply static env vars declared in the engine definition first so that
	// the dynamic AWF vars below can still override them if needed.
	maps.Copy(env, exec.Env)

	if exec.ProviderEnvMode == behaviorProviderEnvModeUniversalLLMConsumer {
		e.ApplyUniversalProviderEnv(env, workflowData, firewallEnabled)
	}

	if exec.MCPConfigEnvVar != "" && HasMCPServers(workflowData) {
		if behavior.ConfigFile != nil {
			env[exec.MCPConfigEnvVar] = "${{ github.workspace }}/" + behavior.ConfigFile.Path
		} else {
			mcpPath := constants.McpServersJsonPathExpr
			if behavior.MCP != nil && behavior.MCP.ConfigPath != "" {
				mcpPath = behavior.MCP.ConfigPath
			}
			env[exec.MCPConfigEnvVar] = mcpPath
		}
	}

	for _, binding := range e.definition.Auth {
		if binding.Secret != "" {
			env[binding.Secret] = "${{ secrets." + binding.Secret + " }}"
		}
	}

	// When a harness script is present and the AWF firewall is running, signal to the
	// harness that the AWF API proxy sidecar is available so it can read /reflect data.
	if behavior.HarnessScript != "" && firewallEnabled {
		env["AWF_REFLECT_ENABLED"] = "1"
	}

	applySafeOutputEnvToMap(env, workflowData)
	applyTraceContextEnvToMap(env)
	applyOptionalEngineToolTimeouts(env, workflowData)
	applyEngineMaxTurnsEnv(env, workflowData)
	applyEngineCwdEnv(env, workflowData)
	applyEngineAndAgentEnv(env, workflowData, behaviorDefinedEngineLog)
	applyMCPScriptsSecretEnv(env, workflowData)

	if exec.ModelEnvVarName != "" {
		if workflowData != nil && workflowData.Model != "" {
			modelVal := workflowData.Model
			if exec.ModelEnvProviderPrefix != "" {
				if parts := strings.SplitN(modelVal, "/", 2); len(parts) == 2 {
					modelVal = exec.ModelEnvProviderPrefix + "/" + parts[1]
				}
			}
			env[exec.ModelEnvVarName] = modelVal
		}
	}

	stepLines := []string{
		"      - name: " + exec.StepName,
		"        id: agentic_execution",
	}
	if workflowData != nil && workflowData.TimeoutMinutes != "" {
		timeoutValue := strings.TrimPrefix(workflowData.TimeoutMinutes, "timeout-minutes: ")
		stepLines = append(stepLines, "        timeout-minutes: "+timeoutValue)
	} else {
		stepLines = append(stepLines, fmt.Sprintf("        timeout-minutes: %d", int(constants.DefaultAgenticWorkflowTimeout/time.Minute)))
	}
	filteredEnv := FilterEnvForSecrets(env, e.GetRequiredSecretNames(workflowData))
	addCliProxyGHTokenToEnv(filteredEnv, workflowData)
	stepLines = FormatStepWithCommandAndEnv(stepLines, command, filteredEnv)
	steps = append(steps, GitHubActionStep(stepLines))
	return steps
}

func (e *BehaviorDefinedEngine) modelFlagFragment(exec *EngineExecutionDefinition, workflowData *WorkflowData) string {
	if exec.ModelEnvVarName == "" || exec.ModelFlag == "" {
		return ""
	}
	if workflowData == nil || workflowData.EngineConfig == nil || workflowData.Model == "" {
		return ""
	}
	return fmt.Sprintf(`%s "$%s"`, exec.ModelFlag, exec.ModelEnvVarName)
}

func (e *BehaviorDefinedEngine) mcpFlagFragment(exec *EngineExecutionDefinition, workflowData *WorkflowData) string {
	if exec.MCPConfigFlag == "" || !HasMCPServers(workflowData) {
		return ""
	}
	path := constants.McpServersJsonPathExpr
	if behavior := e.behavior(); behavior != nil && behavior.MCP != nil {
		if behavior.MCP.ConfigPath != "" {
			path = behavior.MCP.ConfigPath
		}
	}
	return shellJoinArgs([]string{exec.MCPConfigFlag, path})
}

func (e *BehaviorDefinedEngine) buildFirewallCommand(exec *EngineExecutionDefinition, workflowData *WorkflowData, logFile, engineCommand string) string {
	allowedDomains := e.allowedDomains(workflowData)
	// Propagate no_proxy inside the AWF container.  --env-all forwards NO_PROXY
	// from the YAML env block, but Bun (and other runtimes) also check the
	// lowercase variant, so we export it explicitly from the uppercase value.
	engineCommandWithPath := fmt.Sprintf("export no_proxy=\"${NO_PROXY:-}\" && %s && %s", GetNpmBinPathSetup(), engineCommand)
	if mcpCLIPath := GetMCPCLIPathSetup(workflowData); mcpCLIPath != "" {
		engineCommandWithPath = fmt.Sprintf("%s && %s", mcpCLIPath, engineCommandWithPath)
	}

	return BuildAWFCommand(AWFCommandConfig{
		EngineName:         e.GetID(),
		EngineCommand:      engineCommandWithPath,
		LogFile:            logFile,
		WorkflowData:       workflowData,
		UsesTTY:            false,
		AllowedDomains:     allowedDomains,
		ExcludeEnvVarNames: ComputeAWFExcludeEnvVarNames(workflowData, e.GetRequiredSecretNames(workflowData)),
	})
}

func (e *BehaviorDefinedEngine) allowedDomains(workflowData *WorkflowData) string {
	engineName := constants.EngineName(e.GetID())
	if e.usesUniversalLLMConsumer() && workflowData != nil && workflowData.EngineConfig != nil {
		return mustGetAllowedDomainsForEngineWithModel(engineName, workflowData.Model, workflowData.NetworkPermissions, workflowData.Tools, workflowData.Runtimes)
	}
	return GetAllowedDomainsForEngine(engineName, workflowData.NetworkPermissions, workflowData.Tools, workflowData.Runtimes)
}

func (e *BehaviorDefinedEngine) buildConfigFileStep() GitHubActionStep {
	behavior := e.behavior()
	if behavior == nil || behavior.ConfigFile == nil || behavior.ConfigFile.Path == "" {
		return nil
	}
	config := behavior.ConfigFile
	command := fmt.Sprintf(`umask 077
mkdir -p "$(dirname "$GITHUB_WORKSPACE/%s")"
CONFIG="$GITHUB_WORKSPACE/%s"
BASE_CONFIG='%s'
if [ -f "$CONFIG" ]; then
  MERGED=$(jq -n --argjson base "$BASE_CONFIG" --argjson existing "$(cat "$CONFIG")" '$existing * $base')
  echo "$MERGED" > "$CONFIG"
else
  echo "$BASE_CONFIG" > "$CONFIG"
fi
chmod 600 "$CONFIG"`, config.Path, config.Path, config.Content)
	if config.MergeStrategy != behaviorConfigMergeJSON {
		command = fmt.Sprintf(`umask 077
mkdir -p "$(dirname "$GITHUB_WORKSPACE/%s")"
cat <<'EOF' > "$GITHUB_WORKSPACE/%s"
%s
EOF
chmod 600 "$GITHUB_WORKSPACE/%s"`, config.Path, config.Path, config.Content, config.Path)
	}

	stepLines := []string{"      - name: " + config.StepName}
	stepLines = FormatStepWithCommandAndEnv(stepLines, command, nil)
	return GitHubActionStep(stepLines)
}

func isEngineDefinitionForm(def *EngineDefinition) bool {
	if def == nil {
		return false
	}
	// Treat richer metadata-only objects as shared engine definitions. Plain engine
	// config objects ("id", "model", "env", etc.) should continue down the normal
	// EngineConfig path instead of being registered as catalog entries.
	if def.DisplayName != "" || def.RuntimeID != "" || def.Experimental || def.GHSkillAgentName != "" || def.Behaviors != nil || len(def.Auth) > 0 {
		return true
	}
	if def.Provider.Name != "" || def.Provider.Auth != nil || def.Provider.Request != nil {
		return true
	}
	return def.Models.Default != "" || len(def.Models.Supported) > 0 || len(def.Options) > 0
}

// engineDefinitionBuiltinKeys is the set of JSON strings corresponding to
// built-in engine definitions. It is populated once at startup by
// loadBuiltinEngineDefinitions (via registerBuiltinEngineDefinitionJSON) and
// never modified afterward. Only JSON keys present in this set are eligible for
// caching in engineDefinitionCache, which prevents unbounded growth in long-lived
// compile --watch sessions where each edit to a custom engine definition would
// otherwise create a new cache entry.
var engineDefinitionBuiltinKeys sync.Map // map[string]struct{}

// registerBuiltinEngineDefinitionJSON marks jsonKey as a known built-in engine
// JSON string so that parseEngineDefinitionFromJSON will cache it.
func registerBuiltinEngineDefinitionJSON(jsonKey string) {
	engineDefinitionBuiltinKeys.Store(jsonKey, struct{}{})
}

// engineDefinitionCache caches parsed EngineDefinition values for built-in
// engine JSON strings. Built-in engine files are injected as imports on every
// CompileWorkflow call and their JSON representation is always identical, so
// caching avoids the repeated JSON→any→YAML→struct round-trip that accounted
// for ~24% of BenchmarkCompileMCPWorkflow wall-clock time.
//
// Only keys present in engineDefinitionBuiltinKeys are stored to bound the cache
// to the fixed set of built-in engines. Deep copies are returned on every lookup
// so callers cannot corrupt the cached state through mutations to pointers, slices,
// or maps within the returned definition.
var engineDefinitionCache sync.Map // map[string]EngineDefinition

func parseEngineDefinitionFromJSON(engineJSON string) (*EngineDefinition, error) {
	if engineJSON == "" {
		return nil, nil
	}
	// Fast path: return a deep copy of the cached definition when available.
	if cached, ok := engineDefinitionCache.Load(engineJSON); ok {
		if def, ok := cached.(EngineDefinition); ok {
			defCopy := deepCopyEngineDefinition(def)
			return &defCopy, nil
		}
		// Type assertion failure indicates cache corruption or a concurrent Store with
		// an unexpected type. Log and fall through to re-parse so the caller still works.
		behaviorDefinedEngineLog.Printf("engineDefinitionCache: unexpected value type %T for key (len=%d); re-parsing", cached, len(engineJSON))
		engineDefinitionCache.Delete(engineJSON)
	}
	var engineData any
	if err := json.Unmarshal([]byte(engineJSON), &engineData); err != nil {
		return nil, fmt.Errorf("failed to parse engine JSON: %w", err)
	}
	if _, ok := engineData.(map[string]any); !ok {
		return nil, nil
	}
	yamlBytes, err := yaml.Marshal(engineData)
	if err != nil {
		return nil, fmt.Errorf("failed to convert engine JSON to yaml: %w", err)
	}
	var def EngineDefinition
	if err := yaml.Unmarshal(yamlBytes, &def); err != nil {
		return nil, fmt.Errorf("failed to parse engine definition: %w", err)
	}
	if def.RuntimeID == "" {
		def.RuntimeID = def.ID
	}
	// Cache only built-in engine definitions (keys pre-seeded by loadBuiltinEngineDefinitions)
	// to prevent unbounded memory growth. Store a deep copy so that any mutations the
	// caller makes to the returned definition cannot corrupt the cached entry.
	if _, isBuiltin := engineDefinitionBuiltinKeys.Load(engineJSON); isBuiltin {
		cacheCopy := deepCopyEngineDefinition(def)
		engineDefinitionCache.Store(engineJSON, cacheCopy)
	}
	return &def, nil
}

// deepCopyAny returns a fully independent copy of v for values produced by
// yaml.Unmarshal into interface{}. The possible concrete types are:
// nil, bool, int, float64, string, []any, and map[string]any.
func deepCopyAny(v any) any {
	switch val := v.(type) {
	case map[string]any:
		cp := make(map[string]any, len(val))
		for k, elem := range val {
			cp[k] = deepCopyAny(elem)
		}
		return cp
	case []any:
		cp := make([]any, len(val))
		for i, elem := range val {
			cp[i] = deepCopyAny(elem)
		}
		return cp
	default:
		// Scalars (nil, bool, int, float64, string) are immutable value types.
		return v
	}
}

// deepCopyEngineDefinition returns a fully independent copy of src. All reference
// types (pointers, slices, maps) are recursively copied so that neither the caller
// nor the cache can corrupt the other's state through shared references.
func deepCopyEngineDefinition(src EngineDefinition) EngineDefinition {
	dst := src // value copy covers all scalar fields

	// Models.Supported
	if src.Models.Supported != nil {
		dst.Models.Supported = make([]string, len(src.Models.Supported))
		copy(dst.Models.Supported, src.Models.Supported)
	}

	// Auth ([]AuthBinding; elements contain only string fields so element copy suffices)
	if src.Auth != nil {
		dst.Auth = make([]AuthBinding, len(src.Auth))
		copy(dst.Auth, src.Auth)
	}

	// Options (map[string]any; values may contain nested maps or slices from YAML unmarshal)
	if src.Options != nil {
		dst.Options = make(map[string]any, len(src.Options))
		for k, v := range src.Options {
			dst.Options[k] = deepCopyAny(v)
		}
	}

	// Provider.Auth
	if src.Provider.Auth != nil {
		authCopy := *src.Provider.Auth // AuthDefinition contains only string fields
		dst.Provider.Auth = &authCopy
	}

	// Provider.Request
	if src.Provider.Request != nil {
		reqCopy := *src.Provider.Request
		if src.Provider.Request.Query != nil {
			reqCopy.Query = make(map[string]string, len(src.Provider.Request.Query))
			maps.Copy(reqCopy.Query, src.Provider.Request.Query)
		}
		if src.Provider.Request.BodyInject != nil {
			reqCopy.BodyInject = make(map[string]string, len(src.Provider.Request.BodyInject))
			maps.Copy(reqCopy.BodyInject, src.Provider.Request.BodyInject)
		}
		dst.Provider.Request = &reqCopy
	}

	// Behaviors
	if src.Behaviors != nil {
		behaviorsCopy := deepCopyEngineBehaviorDefinition(*src.Behaviors)
		dst.Behaviors = &behaviorsCopy
	}

	return dst
}

// deepCopyEngineBehaviorDefinition returns a fully independent copy of src.
func deepCopyEngineBehaviorDefinition(src EngineBehaviorDefinition) EngineBehaviorDefinition {
	dst := src // value copy covers all scalar fields

	// SupportedEnvVarKeys
	if src.SupportedEnvVarKeys != nil {
		dst.SupportedEnvVarKeys = make([]string, len(src.SupportedEnvVarKeys))
		copy(dst.SupportedEnvVarKeys, src.SupportedEnvVarKeys)
	}

	// Manifest
	if src.Manifest != nil {
		manifestCopy := *src.Manifest
		if src.Manifest.Files != nil {
			manifestCopy.Files = make([]string, len(src.Manifest.Files))
			copy(manifestCopy.Files, src.Manifest.Files)
		}
		if src.Manifest.PathPrefixes != nil {
			manifestCopy.PathPrefixes = make([]string, len(src.Manifest.PathPrefixes))
			copy(manifestCopy.PathPrefixes, src.Manifest.PathPrefixes)
		}
		dst.Manifest = &manifestCopy
	}

	// Installation (only scalar fields; pointer dereference suffices)
	if src.Installation != nil {
		installCopy := *src.Installation
		dst.Installation = &installCopy
	}

	// ConfigFile (only scalar fields)
	if src.ConfigFile != nil {
		cfCopy := *src.ConfigFile
		dst.ConfigFile = &cfCopy
	}

	// Execution (has Args []string and Env map[string]string)
	if src.Execution != nil {
		execCopy := *src.Execution
		if src.Execution.Args != nil {
			execCopy.Args = make([]string, len(src.Execution.Args))
			copy(execCopy.Args, src.Execution.Args)
		}
		if src.Execution.Env != nil {
			execCopy.Env = make(map[string]string, len(src.Execution.Env))
			maps.Copy(execCopy.Env, src.Execution.Env)
		}
		dst.Execution = &execCopy
	}

	// MCP (only scalar fields)
	if src.MCP != nil {
		mcpCopy := *src.MCP
		dst.MCP = &mcpCopy
	}

	return dst
}

func (c *Compiler) registerNamedEngineDefinitionFromJSON(engineJSON string) error {
	def, err := parseEngineDefinitionFromJSON(engineJSON)
	if err != nil || !isEngineDefinitionForm(def) {
		return err
	}
	if def.Behaviors != nil {
		engine, buildErr := NewBehaviorDefinedEngine(def)
		if buildErr != nil {
			return buildErr
		}
		if regErr := c.engineRegistry.Register(engine); regErr != nil {
			return regErr
		}
		def.RuntimeID = engine.GetID()
	}
	c.engineCatalog.Register(def)
	return nil
}
