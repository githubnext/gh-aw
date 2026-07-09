//go:build !integration

package workflow

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBuildSharedPRCheckoutSteps tests shared PR checkout step generation
func TestBuildSharedPRCheckoutSteps(t *testing.T) {
	fetchDepthZero := 0

	tests := []struct {
		name             string
		safeOutputs      *SafeOutputsConfig
		checkoutConfigs  []*CheckoutConfig
		trialMode        bool
		trialRepo        string
		checkContains    []string
		checkNotContains []string
	}{
		{
			name: "create pull request only mirrors agent default checkout",
			safeOutputs: &SafeOutputsConfig{
				CreatePullRequests: &CreatePullRequestsConfig{},
			},
			checkContains: []string{
				"name: Checkout repository",
				"uses: actions/checkout@",
				// safe_outputs job retains credentials so the handlers can git fetch/push.
				"persist-credentials: true",
				"name: Configure Git credentials",
				"configure_git_credentials.sh",
				"GITHUB_REPOSITORY: ${{ github.repository }}",
			},
			checkNotContains: []string{
				// No checkout-time base ref or trusted-default-branch guard anymore;
				// the JS handler resolves the base branch at apply time.
				"trusted default branch for comment events",
				"ref: ${{ github.event.repository.default_branch }}",
				"steps.extract-base-branch.outputs.base-branch",
				// Credentials must NOT be stripped in the safe_outputs job.
				"persist-credentials: false",
			},
		},
		{
			name: "uses custom default checkout fetch-depth",
			safeOutputs: &SafeOutputsConfig{
				CreatePullRequests: &CreatePullRequestsConfig{},
			},
			checkoutConfigs: []*CheckoutConfig{
				{FetchDepth: &fetchDepthZero},
			},
			checkContains: []string{
				"fetch-depth: 0",
			},
		},
		{
			name: "default checkout with GitHub App token mints checkout app token",
			safeOutputs: &SafeOutputsConfig{
				CreatePullRequests: &CreatePullRequestsConfig{},
			},
			checkoutConfigs: []*CheckoutConfig{
				{
					GitHubApp: &GitHubAppConfig{
						AppID:      "12345",
						PrivateKey: "test-key",
					},
				},
			},
			checkContains: []string{
				"token: ${{ steps.checkout-app-token-0.outputs.token }}",
			},
		},
		{
			name: "safe-output checkout app token is minted and used for git credentials",
			safeOutputs: &SafeOutputsConfig{
				CreatePullRequests: &CreatePullRequestsConfig{
					TargetRepoSlug: "org/target-repo",
				},
			},
			checkoutConfigs: []*CheckoutConfig{
				{
					Repository: "org/target-repo",
					Path:       "./target-repo",
					SafeOutputGitHubApp: &GitHubAppConfig{
						AppID:      "12345",
						PrivateKey: "test-key",
					},
				},
			},
			checkContains: []string{
				"id: checkout-safe-output-app-token-0",
				"token: ${{ secrets.GH_AW_GITHUB_TOKEN || secrets.GITHUB_TOKEN }}",
				"GIT_TOKEN: ${{ steps.checkout-safe-output-app-token-0.outputs.token }}",
			},
			checkNotContains: []string{
				"id: checkout-safe-output-app-token-0\n        if:",
			},
		},
		{
			name:      "trial mode with target repo",
			trialMode: true,
			trialRepo: "org/trial-repo",
			safeOutputs: &SafeOutputsConfig{
				CreatePullRequests: &CreatePullRequestsConfig{},
			},
			checkContains: []string{
				"repository: org/trial-repo",
			},
		},
		{
			name: "create-pr per-config github-token flows into git credentials",
			safeOutputs: &SafeOutputsConfig{
				CreatePullRequests: &CreatePullRequestsConfig{
					BaseSafeOutputConfig: BaseSafeOutputConfig{
						GitHubToken: "${{ secrets.GH_AW_CROSS_REPO_PAT }}",
					},
				},
			},
			checkContains: []string{
				"GIT_TOKEN: ${{ secrets.GH_AW_CROSS_REPO_PAT }}",
			},
		},
		{
			name: "safe-outputs github-token flows into git credentials",
			safeOutputs: &SafeOutputsConfig{
				GitHubToken:        "${{ secrets.SAFE_OUTPUTS_TOKEN }}",
				CreatePullRequests: &CreatePullRequestsConfig{},
			},
			checkContains: []string{
				"GIT_TOKEN: ${{ secrets.SAFE_OUTPUTS_TOKEN }}",
			},
		},
		{
			name: "push-to-pull-request-branch per-config token flows into git credentials",
			safeOutputs: &SafeOutputsConfig{
				PushToPullRequestBranch: &PushToPullRequestBranchConfig{
					BaseSafeOutputConfig: BaseSafeOutputConfig{
						GitHubToken: "${{ secrets.PUSH_BRANCH_PAT }}",
					},
				},
			},
			checkContains: []string{
				"GIT_TOKEN: ${{ secrets.PUSH_BRANCH_PAT }}",
			},
		},
		{
			name: "both operations with create-pr token takes precedence",
			safeOutputs: &SafeOutputsConfig{
				CreatePullRequests: &CreatePullRequestsConfig{
					BaseSafeOutputConfig: BaseSafeOutputConfig{
						GitHubToken: "${{ secrets.CREATE_PR_PAT }}",
					},
				},
				PushToPullRequestBranch: &PushToPullRequestBranchConfig{
					BaseSafeOutputConfig: BaseSafeOutputConfig{
						GitHubToken: "${{ secrets.PUSH_BRANCH_PAT }}",
					},
				},
			},
			checkContains: []string{
				"GIT_TOKEN: ${{ secrets.CREATE_PR_PAT }}",
			},
		},
		{
			name: "default checkout config ref is honored",
			safeOutputs: &SafeOutputsConfig{
				CreatePullRequests: &CreatePullRequestsConfig{},
			},
			checkoutConfigs: []*CheckoutConfig{
				{Ref: "develop"},
			},
			checkContains: []string{
				"ref: develop",
			},
		},
		{
			// Issue #40121: a cross-repo target checked out into a subdirectory mirrors the
			// agent layout (workflow repo at root + target at its configured path) instead of
			// collapsing to a single checkout of the target at the workspace root.
			name: "cross-repo checkout into subdirectory mirrors agent layout",
			safeOutputs: &SafeOutputsConfig{
				CreatePullRequests: &CreatePullRequestsConfig{},
			},
			checkoutConfigs: []*CheckoutConfig{
				{Repository: "org/a", Path: "a"},
			},
			checkContains: []string{
				"name: Checkout org/a into a",
				"repository: org/a",
				"path: a",
				// Root workflow checkout is still present (the agent default checkout).
				"name: Checkout repository",
				// Subdirectory checkout is re-authenticated so the handler can push to it.
				`git -C "a" remote set-url origin`,
			},
		},
		{
			name: "two cross-repo checkouts check out both at their paths",
			safeOutputs: &SafeOutputsConfig{
				CreatePullRequests: &CreatePullRequestsConfig{},
			},
			checkoutConfigs: []*CheckoutConfig{
				{Repository: "org/a", Path: "a"},
				{Repository: "org/b", Path: "b"},
			},
			checkContains: []string{
				"name: Checkout org/a into a",
				"path: a",
				"name: Checkout org/b into b",
				"path: b",
			},
		},
		{
			// Issue #40121: a subdirectory cross-repo checkout that uses sparse-checkout
			// must emit the non-empty blob filter and the partial-clone-marker reset, just
			// like the agent job, so the later fetch does not fail on a credential-dependent
			// partial clone.
			name: "subdirectory cross-repo sparse checkout mirrors agent filter and partial-clone reset",
			safeOutputs: &SafeOutputsConfig{
				CreatePullRequests: &CreatePullRequestsConfig{},
			},
			checkoutConfigs: []*CheckoutConfig{
				{Repository: "org/a", Path: "a", SparseCheckout: ".github\nsrc"},
			},
			checkContains: []string{
				"name: Checkout org/a into a",
				"filter: 'blob:limit=1073741824'",
				"name: Clear partial clone markers after sparse checkout",
			},
			checkNotContains: []string{
				"--filter=blob:none",
			},
		},
		{
			name: "cross-repo checkout with fetch refs emits fetch step",
			safeOutputs: &SafeOutputsConfig{
				CreatePullRequests: &CreatePullRequestsConfig{},
			},
			checkoutConfigs: []*CheckoutConfig{
				{
					Repository: "org/target-repo",
					Path:       "target-repo",
					FetchDepth: func() *int { d := 1; return &d }(),
					Fetch:      []string{"master", "my/branch/*"},
				},
			},
			checkContains: []string{
				"+refs/heads/master:refs/remotes/origin/master",
				"+refs/heads/my/branch/*:refs/remotes/origin/my/branch/*",
			},
			checkNotContains: []string{
				"--filter=blob:none",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compiler := NewCompiler()
			if tt.trialMode {
				compiler.SetTrialMode(true)
			}
			if tt.trialRepo != "" {
				compiler.SetTrialLogicalRepoSlug(tt.trialRepo)
			}

			workflowData := &WorkflowData{
				Name:            "Test Workflow",
				SafeOutputs:     tt.safeOutputs,
				CheckoutConfigs: tt.checkoutConfigs,
			}

			steps := compiler.buildSharedPRCheckoutSteps(workflowData)

			require.NotEmpty(t, steps)

			stepsContent := strings.Join(steps, "")

			for _, expected := range tt.checkContains {
				assert.Contains(t, stepsContent, expected, "Expected to find: "+expected)
			}

			for _, notExpected := range tt.checkNotContains {
				assert.NotContains(t, stepsContent, notExpected, "Expected NOT to find: "+notExpected)
			}
		})
	}
}

// TestBuildSharedPRCheckoutStepsConditions tests conditional execution
func TestBuildSharedPRCheckoutStepsConditions(t *testing.T) {
	tests := []struct {
		name                   string
		createPR               bool
		pushToPRBranch         bool
		expectedConditionParts []string
	}{
		{
			name:                   "only create PR",
			createPR:               true,
			pushToPRBranch:         false,
			expectedConditionParts: []string{"create_pull_request"},
		},
		{
			name:                   "only push to PR branch",
			createPR:               false,
			pushToPRBranch:         true,
			expectedConditionParts: []string{"push_to_pull_request_branch"},
		},
		{
			name:                   "both operations",
			createPR:               true,
			pushToPRBranch:         true,
			expectedConditionParts: []string{"create_pull_request", "push_to_pull_request_branch", "||"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compiler := NewCompiler()

			safeOutputs := &SafeOutputsConfig{}
			if tt.createPR {
				safeOutputs.CreatePullRequests = &CreatePullRequestsConfig{}
			}
			if tt.pushToPRBranch {
				safeOutputs.PushToPullRequestBranch = &PushToPullRequestBranchConfig{}
			}

			workflowData := &WorkflowData{
				Name:        "Test Workflow",
				SafeOutputs: safeOutputs,
			}

			steps := compiler.buildSharedPRCheckoutSteps(workflowData)

			require.NotEmpty(t, steps)

			stepsContent := strings.Join(steps, "")

			for _, part := range tt.expectedConditionParts {
				assert.Contains(t, stepsContent, part, "Expected condition part: "+part)
			}
		})
	}
}

// TestBuildHandlerManagerStep tests handler manager step generation
func TestBuildHandlerManagerStep(t *testing.T) {
	tests := []struct {
		name              string
		safeOutputs       *SafeOutputsConfig
		parsedFrontmatter *FrontmatterConfig
		checkContains     []string
		checkNotContains  []string
	}{
		{
			name: "basic handler manager",
			safeOutputs: &SafeOutputsConfig{
				CreateIssues: &CreateIssuesConfig{},
			},
			checkContains: []string{
				"name: Process Safe Outputs",
				"id: process_safe_outputs",
				"uses: actions/github-script@",
				"GH_AW_AGENT_OUTPUT",
				"GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG",
				"setupGlobals",
				"safe_output_handler_manager.cjs",
			},
		},
		{
			name: "handler manager with multiple types",
			safeOutputs: &SafeOutputsConfig{
				CreateIssues: &CreateIssuesConfig{
					TitlePrefix: "[Issue] ",
				},
				AddComments: &AddCommentsConfig{
					BaseSafeOutputConfig: BaseSafeOutputConfig{
						Max: strPtr("5"),
					},
				},
				CreateDiscussions: &CreateDiscussionsConfig{
					Category: "general",
				},
			},
			checkContains: []string{
				"name: Process Safe Outputs",
				"GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG",
			},
		},
		{
			name: "handler manager with project URL from update-project config",
			safeOutputs: &SafeOutputsConfig{
				UpdateProjects: &UpdateProjectConfig{
					BaseSafeOutputConfig: BaseSafeOutputConfig{
						Max: strPtr("5"),
					},
					Project: "https://github.com/orgs/github-agentic-workflows/projects/1",
				},
			},
			parsedFrontmatter: &FrontmatterConfig{
				Engine: "copilot",
			},
			checkContains: []string{
				"name: Process Safe Outputs",
				"GH_AW_PROJECT_URL: \"https://github.com/orgs/github-agentic-workflows/projects/1\"",
			},
		},
		{
			name: "handler manager with project URL from update-project config",
			safeOutputs: &SafeOutputsConfig{
				UpdateProjects: &UpdateProjectConfig{
					BaseSafeOutputConfig: BaseSafeOutputConfig{
						Max: strPtr("5"),
					},
					Project: "https://github.com/orgs/github-agentic-workflows/projects/1",
				},
			},
			checkContains: []string{
				"GH_AW_PROJECT_URL: \"https://github.com/orgs/github-agentic-workflows/projects/1\"",
			},
		},
		{
			name: "handler manager with project URL from create-project-status-update config",
			safeOutputs: &SafeOutputsConfig{
				CreateProjectStatusUpdates: &CreateProjectStatusUpdateConfig{
					BaseSafeOutputConfig: BaseSafeOutputConfig{
						Max: strPtr("1"),
					},
					Project: "https://github.com/orgs/github-agentic-workflows/projects/1",
				},
			},
			checkContains: []string{
				"GH_AW_PROJECT_URL: \"https://github.com/orgs/github-agentic-workflows/projects/1\"",
			},
		},
		{
			name: "handler manager without project does not include GH_AW_PROJECT_URL",
			safeOutputs: &SafeOutputsConfig{
				CreateIssues: &CreateIssuesConfig{},
			},
			checkNotContains: []string{
				"GH_AW_PROJECT_URL",
			},
		},
		{
			name: "handler manager with allowed-domains propagates to process step",
			safeOutputs: &SafeOutputsConfig{
				AllowedDomains: []string{"docs.example.com", "api.example.com"},
				AddComments:    &AddCommentsConfig{},
			},
			checkContains: []string{
				"GH_AW_ALLOWED_DOMAINS:",
				"docs.example.com",
				"api.example.com",
				"GITHUB_SERVER_URL: ${{ github.server_url }}",
				"GITHUB_API_URL: ${{ github.api_url }}",
			},
		},
		{
			name: "handler manager with urls policy propagates to process step",
			safeOutputs: &SafeOutputsConfig{
				URLs:        SafeOutputsURLsPolicyAllowedOrCodeRegion,
				AddComments: &AddCommentsConfig{},
			},
			checkContains: []string{
				"GH_AW_SAFE_OUTPUTS_URLS: \"allowed-or-code-region\"",
			},
		},
		{
			name: "handler manager without allowed-domains still includes github urls",
			safeOutputs: &SafeOutputsConfig{
				CreateIssues: &CreateIssuesConfig{},
			},
			checkContains: []string{
				"GITHUB_SERVER_URL: ${{ github.server_url }}",
				"GITHUB_API_URL: ${{ github.api_url }}",
			},
		},
		// Note: create_project is now handled by the unified handler manager,
		// not the separate project handler manager
		{
			name: "handler manager with custom safe jobs includes GH_AW_SAFE_OUTPUT_JOBS",
			safeOutputs: &SafeOutputsConfig{
				CreateIssues: &CreateIssuesConfig{},
				Jobs: map[string]*SafeJobConfig{
					"send-slack-message": {
						Description: "Send a Slack message",
					},
				},
			},
			checkContains: []string{
				"GH_AW_SAFE_OUTPUT_JOBS: \"{\\\"send_slack_message\\\":\\\"\\\"}\"",
			},
		},
		{
			name: "handler manager without custom safe jobs does not include GH_AW_SAFE_OUTPUT_JOBS",
			safeOutputs: &SafeOutputsConfig{
				CreateIssues: &CreateIssuesConfig{},
			},
			checkNotContains: []string{
				"GH_AW_SAFE_OUTPUT_JOBS",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compiler := NewCompiler()

			workflowData := &WorkflowData{
				Name:              "Test Workflow",
				SafeOutputs:       tt.safeOutputs,
				ParsedFrontmatter: tt.parsedFrontmatter,
			}

			steps, err := compiler.buildHandlerManagerStep(workflowData)
			require.NoError(t, err)

			require.NotEmpty(t, steps)

			stepsContent := strings.Join(steps, "")

			for _, expected := range tt.checkContains {
				assert.Contains(t, stepsContent, expected, "Expected to find: "+expected)
			}

			for _, notExpected := range tt.checkNotContains {
				assert.NotContains(t, stepsContent, notExpected, "Expected NOT to find: "+notExpected)
			}
		})
	}
}

// TestStepOrderInConsolidatedJob tests that steps appear in correct order
func TestStepOrderInConsolidatedJob(t *testing.T) {
	compiler := NewCompiler()
	compiler.jobManager = NewJobManager()

	workflowData := &WorkflowData{
		Name: "Test Workflow",
		SafeOutputs: &SafeOutputsConfig{
			CreatePullRequests: &CreatePullRequestsConfig{
				TitlePrefix: "[Test] ",
			},
		},
	}

	job, _, err := compiler.buildConsolidatedSafeOutputsJob(workflowData, "agent", "test.md")

	require.NoError(t, err)
	require.NotNil(t, job)

	stepsContent := strings.Join(job.Steps, "")

	// Find positions of key steps
	setupPos := strings.Index(stepsContent, "name: Setup Scripts")
	downloadPos := strings.Index(stepsContent, "name: Download agent output")
	patchPos := strings.Index(stepsContent, "name: Download patch artifact")
	checkoutPos := strings.Index(stepsContent, "name: Checkout repository")
	gitConfigPos := strings.Index(stepsContent, "name: Configure Git credentials")
	handlerPos := strings.Index(stepsContent, "name: Process Safe Outputs")

	// Verify order
	if setupPos != -1 && downloadPos != -1 {
		assert.Less(t, setupPos, downloadPos, "Setup should come before download")
	}
	if downloadPos != -1 && patchPos != -1 {
		assert.Less(t, downloadPos, patchPos, "Agent output download should come before patch download")
	}
	if patchPos != -1 && checkoutPos != -1 {
		assert.Less(t, patchPos, checkoutPos, "Patch download should come before checkout")
	}
	if checkoutPos != -1 && gitConfigPos != -1 {
		assert.Less(t, checkoutPos, gitConfigPos, "Checkout should come before git config")
	}
	if gitConfigPos != -1 && handlerPos != -1 {
		assert.Less(t, gitConfigPos, handlerPos, "Git config should come before handler")
	}
}
