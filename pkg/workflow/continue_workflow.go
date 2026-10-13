package workflow

import (
	"errors"
	"fmt"
	"maps"
	"sort"
	"strconv"
	"strings"

	"github.com/github/gh-aw/pkg/logger"
	"github.com/goccy/go-yaml"
)

var continueWorkflowLog = logger.New("workflow:continue_workflow")

const (
	// DefaultContinueWorkflowMaxHops is the default number of continuation runs after the first run.
	DefaultContinueWorkflowMaxHops = 3
	// MaxContinueWorkflowMaxHops is the largest accepted continue-workflow max-hops value.
	MaxContinueWorkflowMaxHops = 10
)

// ContinueWorkflowConfig enables the opt-in continue_workflow safe output, which hands the
// agent's committed progress to a fresh run of the same workflow on one stable work branch.
// It is unrelated to the Copilot engine's max-continuations setting.
type ContinueWorkflowConfig struct {
	BaseSafeOutputConfig `yaml:",inline"`
	MaxHops              *int `yaml:"max-hops,omitempty"`
}

// EffectiveMaxHops returns the configured hop budget or the default of 3.
func (c *ContinueWorkflowConfig) EffectiveMaxHops() int {
	if c == nil || c.MaxHops == nil {
		return DefaultContinueWorkflowMaxHops
	}
	return *c.MaxHops
}

// parseContinueWorkflowConfig handles continue-workflow configuration. Both `true` and an
// empty value enable the handler with defaults; `false` leaves it disabled.
func (c *Compiler) parseContinueWorkflowConfig(outputMap map[string]any) *ContinueWorkflowConfig {
	raw, exists := outputMap["continue-workflow"]
	if !exists {
		return nil
	}
	if enabled, ok := raw.(bool); ok {
		if !enabled {
			continueWorkflowLog.Print("continue-workflow explicitly disabled")
			return nil
		}
		raw = nil
	}
	normalized := maps.Clone(outputMap)
	normalized["continue-workflow"] = raw
	return parseConfigScaffoldWithPostProcess(normalized, "continue-workflow", continueWorkflowLog,
		func(err error) *ContinueWorkflowConfig {
			continueWorkflowLog.Printf("Failed to unmarshal continue-workflow config, disabling handler: %v", err)
			return nil
		},
		func(config *ContinueWorkflowConfig) {
			// One continuation request per run; the hop budget is max-hops.
			one := "1"
			config.Max = &one
			continueWorkflowLog.Printf("Parsed configuration: max_hops=%d", config.EffectiveMaxHops())
		})
}

// hasContinueWorkflow reports whether the workflow enables continue-workflow.
func hasContinueWorkflow(data *WorkflowData) bool {
	return data != nil && data.SafeOutputs != nil && data.SafeOutputs.ContinueWorkflow != nil
}

// validateContinueWorkflow checks the constraints that make continuation hops safe to resume.
func validateContinueWorkflow(data *WorkflowData) error {
	if !hasContinueWorkflow(data) {
		return nil
	}
	cfg := data.SafeOutputs.ContinueWorkflow
	if hops := cfg.EffectiveMaxHops(); hops < 1 || hops > MaxContinueWorkflowMaxHops {
		return fmt.Errorf("safe-outputs.continue-workflow.max-hops must be between 1 and %d, got %d", MaxContinueWorkflowMaxHops, hops)
	}
	cpr := data.SafeOutputs.CreatePullRequests
	if cpr == nil {
		return errors.New("safe-outputs.continue-workflow requires safe-outputs.create-pull-request: the final run of a continued workflow creates the pull request from the work branch")
	}
	if strings.TrimSpace(cpr.TargetRepoSlug) != "" || strings.TrimSpace(cpr.HeadRepoSlug) != "" {
		return errors.New("safe-outputs.continue-workflow cannot be combined with create-pull-request target-repo or head-repo: continuation work branches live in the workflow repository")
	}
	if cpr.PreserveBranchName {
		return errors.New("safe-outputs.continue-workflow cannot be combined with create-pull-request preserve-branch-name: the work branch name is assigned by the continuation")
	}
	if strings.TrimSpace(cpr.PatchFormat) == "am" {
		return errors.New("safe-outputs.continue-workflow requires create-pull-request patch-format: bundle (the default)")
	}
	if !canResumeContinuation(data) {
		return errors.New("safe-outputs.continue-workflow requires the repository checkout and contents: read permission so each continuation run can resume on its work branch")
	}
	if currentPath := normalizeCurrentCheckoutPatchPath(NewCheckoutManager(data.CheckoutConfigs).GetCurrentCheckoutPath()); currentPath != "" {
		return fmt.Errorf("safe-outputs.continue-workflow requires the workflow repository to be checked out at the workspace root, but the current checkout uses path %q", currentPath)
	}
	dispatch, found := workflowDispatchTrigger(data)
	if !found {
		return errors.New("safe-outputs.continue-workflow requires an on.workflow_dispatch trigger: Agentic Maintenance dispatches each continuation run")
	}
	if required := requiredWorkflowDispatchInputs(dispatch); len(required) > 0 {
		return fmt.Errorf("safe-outputs.continue-workflow cannot dispatch continuation runs because on.workflow_dispatch.inputs %s are required without a default; set required: false or add a default", strings.Join(required, ", "))
	}
	return nil
}

// canResumeContinuation reports whether the agent job checks out the repository with read
// access, which the continuation checkout step needs to resume on the work branch.
func canResumeContinuation(data *WorkflowData) bool {
	if data.CheckoutDisabled {
		return false
	}
	if data.CachedPermissions != nil {
		return data.CachedPermissions.HasContentsReadAccess()
	}
	return NewPermissionsParser(data.Permissions).HasContentsReadAccess()
}

// workflowDispatchTrigger returns the workflow_dispatch trigger configuration, if present.
func workflowDispatchTrigger(data *WorkflowData) (map[string]any, bool) {
	if strings.TrimSpace(data.On) != "" {
		var parsed map[string]any
		if err := yaml.Unmarshal([]byte(data.On), &parsed); err == nil {
			if dispatch, found := lookupWorkflowDispatch(parsed["on"]); found {
				return dispatch, true
			}
		} else {
			continueWorkflowLog.Printf("Failed to parse rendered on section: %v", err)
		}
	}
	if data.RawFrontmatter != nil {
		return lookupWorkflowDispatch(data.RawFrontmatter["on"])
	}
	return nil, false
}

func lookupWorkflowDispatch(on any) (map[string]any, bool) {
	switch value := on.(type) {
	case string:
		return nil, value == "workflow_dispatch"
	case []any:
		for _, item := range value {
			if name, ok := item.(string); ok && name == "workflow_dispatch" {
				return nil, true
			}
		}
	case map[string]any:
		dispatch, found := value["workflow_dispatch"]
		if !found {
			return nil, false
		}
		if dispatch == nil {
			return nil, true
		}
		dispatchMap, ok := dispatch.(map[string]any)
		if !ok {
			return nil, false
		}
		return dispatchMap, true
	}
	return nil, false
}

func requiredWorkflowDispatchInputs(dispatch map[string]any) []string {
	inputs, ok := dispatch["inputs"].(map[string]any)
	if !ok {
		return nil
	}
	var required []string
	for name, raw := range inputs {
		input, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		_, hasDefault := input["default"]
		if isRequired, ok := input["required"].(bool); ok && isRequired && !hasDefault {
			required = append(required, name)
		}
	}
	sort.Strings(required)
	return required
}

// buildContinueWorkflowHandlerConfig shares the create-pull-request policy (protected files,
// allowed files, patch limits, signed commits, branch prefix and base branch) so every
// continuation hop is held to exactly the policy of the final pull request.
func buildContinueWorkflowHandlerConfig(cfg *SafeOutputsConfig) map[string]any {
	if cfg.ContinueWorkflow == nil || cfg.CreatePullRequests == nil {
		return nil
	}
	c := cfg.ContinueWorkflow
	config := newCreatePullRequestHandlerConfigBuilder(cfg, cfg.CreatePullRequests).Build()
	delete(config, "staged")
	config["max"] = 1
	config["max_hops"] = c.EffectiveMaxHops()
	if token := resolveHandlerGitHubToken(c.GitHubApp, "continue-workflow", c.GitHubToken); token != "" {
		config["github-token"] = token
	}
	if c.Staged != nil {
		staged := newHandlerConfigBuilder().AddTemplatableBool("staged", templatableBoolPtrToStringPtr(c.Staged)).Build()
		if value, ok := staged["staged"]; ok {
			config["staged"] = value
		}
	}
	return config
}

// generateContinueWorkflowCheckout adds the agent-job step that resumes a continuation hop:
// it checks out the recorded head of the work branch and writes the continuation context
// (including the plan.md location) for the agent. Runs that are not continuations skip it.
func (c *Compiler) generateContinueWorkflowCheckout(yaml *strings.Builder, data *WorkflowData) {
	if !hasContinueWorkflow(data) || !canResumeContinuation(data) {
		return
	}
	continueWorkflowLog.Print("Generating continue-workflow checkout step")
	checkoutManager := NewCheckoutManager(data.CheckoutConfigs)
	checkoutToken := resolveGitHubToken("")
	if override := checkoutManager.GetDefaultCheckoutOverride(); override != nil {
		checkoutToken = resolveCheckoutTokenExpression(override, checkoutManager.defaultCheckoutIndex(override), true)
	}
	yaml.WriteString(c.continueWorkflowCheckoutStep(data, checkoutToken, "checkout-continuation"))
}

func (c *Compiler) continueWorkflowCheckoutStep(data *WorkflowData, checkoutToken, stepID string) string {
	var yaml strings.Builder
	yaml.WriteString("      - name: Checkout continuation branch\n")
	yaml.WriteString("        id: " + stepID + "\n")
	RenderConditionAsIf(&yaml, BuildEventTypeEquals("workflow_dispatch"), "          ")
	yaml.WriteString("        uses: " + getCachedActionPin("actions/github-script", data) + "\n")
	yaml.WriteString("        with:\n")
	yaml.WriteString("          github-token: " + checkoutToken + "\n")
	yaml.WriteString("          script: |\n")
	yaml.WriteString("            const { setupGlobals } = require('" + SetupActionDestination + "/setup_globals.cjs');\n")
	yaml.WriteString("            setupGlobals(core, github, context, exec, io, getOctokit);\n")
	yaml.WriteString("            const { main } = require('" + SetupActionDestination + "/checkout_continue_workflow_branch.cjs');\n")
	yaml.WriteString("            await main();\n")
	return yaml.String()
}

func (c *Compiler) continueWorkflowCustomCheckoutSteps(data *WorkflowData) (map[int]GitHubActionStep, error) {
	if !hasContinueWorkflow(data) || !canResumeContinuation(data) {
		return nil, nil
	}
	var document struct {
		Steps []struct {
			Uses string `yaml:"uses"`
			With struct {
				Path  string `yaml:"path"`
				Token string `yaml:"token"`
			} `yaml:"with"`
		} `yaml:"steps"`
	}
	if err := yaml.Unmarshal([]byte(data.CustomSteps), &document); err != nil {
		return nil, fmt.Errorf("cannot insert continuation checkout in custom steps: %w", err)
	}
	insertions := make(map[int]GitHubActionStep)
	for index, step := range document.Steps {
		if !isCheckoutActionReference(step.Uses) || normalizeCurrentCheckoutPatchPath(step.With.Path) != "" {
			continue
		}
		rendered := c.continueWorkflowCheckoutStep(data, resolveGitHubToken(step.With.Token), "checkout-continuation-"+strconv.Itoa(index))
		insertions[index] = strings.Split(strings.TrimSuffix(rendered, "\n"), "\n")
	}
	if len(insertions) == 0 {
		return nil, errors.New("safe-outputs.continue-workflow requires a workspace-root checkout in custom steps")
	}
	return insertions, nil
}

func firstContinuationCheckoutIndex(insertions map[int]GitHubActionStep, fallback int) int {
	first := -1
	for index := range insertions {
		if first == -1 || index < first {
			first = index
		}
	}
	if first == -1 {
		return fallback
	}
	return first
}
