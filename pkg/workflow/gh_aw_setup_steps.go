package workflow

import (
	"fmt"
	"sort"
)

type ghAwSetupStepConfig struct {
	actionMode           ActionMode
	ifCondition          string
	cliVersion           string
	actionRepo           string
	fallbackActionRefTag string
	workflowData         *WorkflowData
	withFields           map[string]string
}

func generateGhAwSetupStep(config ghAwSetupStepConfig) (GitHubActionStep, error) {
	if config.actionMode == ActionModeDev {
		step := GitHubActionStep{"      - name: Build and install gh-aw CLI from source"}
		if config.ifCondition != "" {
			step = append(step, "        if: "+config.ifCondition)
		}
		step = append(step,
			"        run: |",
			"          gh extension remove aw || true",
			"          make build",
			"          gh extension install .",
			"          gh aw version",
			"        env:",
			"          GH_TOKEN: ${{ github.token }}",
		)
		return step, nil
	}

	// Pinning errors are non-fatal: we still emit a valid step with the fallback
	// action reference so compilation and workflow execution can continue.
	actionRef, pinErr := resolveGhAwSetupActionRef(config)
	step := GitHubActionStep{
		"      - name: Install gh-aw extension",
	}
	if config.ifCondition != "" {
		step = append(step, "        if: "+config.ifCondition)
	}
	step = append(step, "        uses: "+actionRef)
	step = append(step, "        with:")
	step = append(step, fmt.Sprintf("          version: '%s'", config.cliVersion))

	var keys []string
	for key := range config.withFields {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		step = append(step, fmt.Sprintf("          %s: %s", key, config.withFields[key]))
	}

	return step, pinErr
}

// resolveGhAwSetupActionRef resolves the setup-cli action reference in priority order:
//  1. Use workflow-aware pin resolution (getActionPinWithData) when WorkflowData exists.
//  2. Otherwise use the static pin table (getActionPin) when available.
//  3. Otherwise fall back to repo@tag, then repo with no ref as a final fallback.
func resolveGhAwSetupActionRef(config ghAwSetupStepConfig) (string, error) {
	if config.workflowData != nil {
		actionRef := fmt.Sprintf("%s@%s", config.actionRepo, config.cliVersion)
		pinnedRef, err := getActionPinWithData(config.actionRepo, config.cliVersion, config.workflowData)
		if err != nil {
			return actionRef, err
		}
		if pinnedRef != "" {
			return pinnedRef, nil
		}
		return actionRef, nil
	}

	actionRef := getActionPin(config.actionRepo)
	if actionRef != "" {
		return actionRef, nil
	}

	if config.fallbackActionRefTag != "" {
		return fmt.Sprintf("%s@%s", config.actionRepo, config.fallbackActionRefTag), nil
	}
	return config.actionRepo, nil
}
