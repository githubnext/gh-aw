package lint

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/github/gh-aw/pkg/workflow/actions"
	"github.com/github/gh-aw/pkg/workflow/aw"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type ruleContract struct {
	targets      ASTTargets
	unsafe       string
	safe         string
	path         string
	severity     Severity
	strictPolicy bool
}

const contractJob = "jobs:\n  check:\n    runs-on: ubuntu-latest\n"
const contractSafeSteps = "    steps:\n      - shell: bash\n        run: echo public\n"

func builtInRuleContracts() map[string]ruleContract {
	return map[string]ruleContract{
		"secret-leak": {
			targets: ActionsAST | AWAST,
			unsafe:  contractJob + "    env:\n      TOKEN: '${{ secrets.CONTRACT_SECRET }}'\n    steps:\n      - shell: bash\n        run: echo \"$TOKEN\"\n",
			safe:    contractJob + "    env:\n      TOKEN: '${{ secrets.CONTRACT_SECRET }}'\n    steps:\n      - shell: bash\n        run: test -n \"$TOKEN\"\n",
			path:    "jobs.check.steps[0].run",
		},
		"action-pin": {
			targets: ActionsAST | AWAST, strictPolicy: true,
			unsafe: contractJob + "    steps:\n      - uses: actions/checkout@v4\n",
			safe:   contractJob + "    steps:\n      - uses: actions/checkout@0123456789abcdef0123456789abcdef01234567\n",
			path:   "jobs.check.steps[0].uses",
		},
		"job-timeout": {
			targets: ActionsAST | AWAST,
			unsafe:  contractJob + "    timeout-minutes: '${{ inputs.timeout }}'\n" + contractSafeSteps,
			safe:    contractJob + "    timeout-minutes: 15\n" + contractSafeSteps,
			path:    "jobs.check.timeout-minutes",
		},
		"template-injection": {
			targets: ActionsAST | AWAST,
			unsafe:  contractJob + "    steps:\n      - shell: bash\n        run: echo '${{ github.event.issue.title }}'\n",
			safe:    contractJob + "    steps:\n      - shell: bash\n        env:\n          TITLE: '${{ github.event.issue.title }}'\n        run: printf '%s\\n' \"$TITLE\"\n",
			path:    "jobs.check.steps[0].run",
		},
		"workflow-run-scope": {
			targets: ActionsAST | AWAST, strictPolicy: true,
			unsafe: "on:\n  workflow_run:\n    workflows: [CI]\n    branches: ['*']\n" + contractJob + contractSafeSteps,
			safe:   "on:\n  workflow_run:\n    workflows: [CI]\n    branches: [main]\n" + contractJob + contractSafeSteps,
			path:   "on.workflow_run.branches",
		},
		"permission-scope": {
			targets: ActionsAST | AWAST, strictPolicy: true,
			unsafe: "permissions: write-all\n" + contractJob + contractSafeSteps,
			safe:   "permissions:\n  contents: read\n  id-token: write\n" + contractJob + contractSafeSteps,
			path:   "permissions",
		},
		"version-check": {
			targets: AWAST, strictPolicy: true,
			unsafe: "check-for-updates: false\n",
			safe:   "check-for-updates: true\n",
			path:   "check-for-updates",
		},
		"conditional-import": {
			targets: AWAST,
			unsafe:  "imports:\n  - path: shared.md\n    if: false\n",
			safe:    "imports:\n  - path: shared.md\n    with:\n      if: false\n",
			path:    "imports[0].if",
		},
		"private-to-public": {
			targets: AWAST, strictPolicy: true,
			unsafe: "tools:\n  github:\n    private-to-public-flows: allow\n",
			safe:   "tools:\n  github:\n    mode: local\n",
			path:   "tools.github.private-to-public-flows",
		},
		"install-scripts": {
			targets: AWAST, strictPolicy: true,
			unsafe: "runtimes:\n  node:\n    run-install-scripts: true\n",
			safe:   "runtimes:\n  node:\n    run-install-scripts: false\n",
			path:   "runtimes.node.run-install-scripts",
		},
		"allowed-labels": {
			targets: AWAST,
			unsafe:  "safe-outputs:\n  create-issue:\n    allowed-labels: ['*']\n",
			safe:    "safe-outputs:\n  create-issue:\n    allowed-labels: ['team-*']\n",
			path:    "safe-outputs.create-issue.allowed-labels",
		},
		"network-scope": {
			targets: AWAST, strictPolicy: true,
			unsafe: "network:\n  allowed: [defaults, '*']\n",
			safe:   "network:\n  allowed: [defaults, '*.example.com']\n",
			path:   "network.allowed",
		},
		"push-target": {
			targets: AWAST, severity: Warning,
			unsafe: "safe-outputs:\n  push-to-pull-request-branch:\n    target: '*'\n",
			safe:   "safe-outputs:\n  push-to-pull-request-branch:\n    target: '*'\n    required-labels: [automated]\n",
			path:   "safe-outputs.push-to-pull-request-branch.target",
		},
		"tool-scope": {
			targets: AWAST, strictPolicy: true,
			unsafe: "tools:\n  bash: false\n  cli-proxy: true\n",
			safe:   "tools:\n  bash: false\n  cli-proxy: false\n",
			path:   "tools.cli-proxy",
		},
		"sandbox-config": {
			targets: AWAST, strictPolicy: true,
			unsafe: "sandbox:\n  agent: false\n",
			safe:   "sandbox:\n  agent: awf\n",
			path:   "sandbox.agent",
		},
		"image-argument": {
			targets: ActionsAST | AWAST,
			unsafe:  contractJob + "    container:\n      image: '--contract-image-option'\n" + contractSafeSteps,
			safe:    contractJob + "    container:\n      image: 'registry.example/app:1'\n" + contractSafeSteps,
			path:    "jobs.check.container.image",
		},
		"checkout-mode": {
			targets: AWAST,
			unsafe:  "checkout:\n  multi: true\n  path: repositories\n  allowed-repos: ['*/*']\n",
			safe:    "checkout:\n  multi: true\n  path: repositories\n  allowed-repos: ['owner/*']\n",
			path:    "checkout.allowed-repos",
		},
		"fork-association": {
			targets: AWAST,
			unsafe:  "on:\n  pull_request:\n    forks:\n      author-association: [owner]\n",
			safe:    "on:\n  pull_request:\n    forks:\n      author-association: [OWNER]\n",
			path:    "on.pull_request.forks.author-association",
		},
	}
}

func TestEveryBuiltInRulePositiveAndNegative(t *testing.T) {
	t.Parallel()
	contracts := builtInRuleContracts()
	rules := Default().Rules()
	require.Len(t, contracts, len(rules), "every registered rule needs a positive and negative contract")
	for _, rule := range rules {
		t.Run(rule.ID, func(t *testing.T) {
			t.Parallel()
			contract, exists := contracts[rule.ID]
			require.True(t, exists, "add a contract for the newly registered rule")
			require.Equal(t, contract.targets, rule.Targets, "each supported AST needs positive and negative coverage")
			for _, target := range []ASTTargets{AWAST, ActionsAST} {
				if contract.targets&target == 0 {
					continue
				}
				t.Run(target.String(), func(t *testing.T) {
					runRuleContractPair(t, rule.ID, contract, target, false)
				})
			}
			if contract.targets&AWAST != 0 {
				t.Run("aw-nonstrict", func(t *testing.T) {
					runRuleContractPair(t, rule.ID, contract, AWAST, true)
				})
			}
		})
	}
}

func runRuleContractPair(t *testing.T, id string, contract ruleContract, target ASTTargets, nonstrict bool) {
	t.Helper()
	for _, test := range []struct {
		name, source string
		finding      bool
	}{
		{"positive", contract.unsafe, true},
		{"negative", contract.safe, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			input := ruleContractInput(t, test.source, target, nonstrict)
			before, err := json.Marshal(input)
			require.NoError(t, err)
			runner, err := Default().Select(id)
			require.NoError(t, err)
			diagnostics, err := runner.Run(context.Background(), input)
			require.NoError(t, err, "fixtures must exercise rules, not fail parsing or traversal")
			after, err := json.Marshal(input)
			require.NoError(t, err)
			assert.Equal(t, before, after, "rules must not mutate either AST")
			assertRuleContractDiagnostics(t, id, contract, target, nonstrict, test.finding, diagnostics)
			again, err := runner.Run(context.Background(), input)
			require.NoError(t, err)
			assert.Equal(t, diagnostics, again, "results must be repeatable")
		})
	}
}

func ruleContractInput(t *testing.T, source string, target ASTTargets, nonstrict bool) Input {
	t.Helper()
	if target == ActionsAST {
		workflow, err := actions.Parse([]byte(source))
		require.NoError(t, err)
		return Input{Workflow: workflow}
	}
	if nonstrict {
		source = "strict: false\n" + source
	}
	document, err := aw.Parse("---\n" + source + "---\nPublic contract-test prompt.\n")
	require.NoError(t, err)
	return Input{Document: document}
}

func assertRuleContractDiagnostics(t *testing.T, id string, contract ruleContract, target ASTTargets, nonstrict, positive bool, diagnostics []Diagnostic) {
	t.Helper()
	want := make(map[string]Severity)
	if id == "secret-leak" && target == AWAST {
		// AW-only analysis must retain its generated-execution coverage gap,
		// including in the negative case; it is not a secret disclosure.
		want["workflow"] = Warning
	}
	if positive {
		severity := contract.severity
		if severity == "" {
			severity = Error
		}
		if contract.strictPolicy && (nonstrict || target == ActionsAST) {
			severity = Warning
		}
		want[contract.path] = severity
	}
	require.Len(t, diagnostics, len(want), "positive cases must detect the violation; negative cases must reject false positives")
	for _, diagnostic := range diagnostics {
		severity, expected := want[diagnostic.Path]
		require.True(t, expected, "unexpected diagnostic at %s", diagnostic.Path)
		assert.Equal(t, id, diagnostic.RuleID)
		assert.Equal(t, severity, diagnostic.Severity)
		assert.NotEmpty(t, diagnostic.Message)
		if diagnostic.Path == "workflow" {
			assert.Equal(t, "Incomplete secret-leak analysis: only authored AW execution nodes are available; imports, generated steps, cross-job flows and compiler defaults require the compiled Actions AST", diagnostic.Message)
		}
		delete(want, diagnostic.Path)
	}
	assert.Empty(t, want, "every expected diagnostic must be emitted")
}
