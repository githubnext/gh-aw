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
	tests := []struct {
		name             string
		safeOutputs      *SafeOutputsConfig
		trialMode        bool
		trialRepo        string
		checkContains    []string
		checkNotContains []string
	}{
		{
			name: "create pull request only",
			safeOutputs: &SafeOutputsConfig{
				CreatePullRequests: &CreatePullRequestsConfig{},
			},
			checkContains: []string{
				"name: Checkout repository",
				"uses: actions/checkout@",
				"token: ${{ secrets.GH_AW_GITHUB_TOKEN || secrets.GITHUB_TOKEN }}",
				"persist-credentials: false",
				"fetch-depth: 1",
				"name: Configure Git credentials",
				"git config --global user.email",
				"github-actions[bot]@users.noreply.github.com",
			},
		},
		{
			name: "push to PR branch only",
			safeOutputs: &SafeOutputsConfig{
				PushToPullRequestBranch: &PushToPullRequestBranchConfig{},
			},
			checkContains: []string{
				"name: Checkout repository",
				"name: Configure Git credentials",
			},
		},
		{
			name: "both create PR and push to PR branch",
			safeOutputs: &SafeOutputsConfig{
				CreatePullRequests:      &CreatePullRequestsConfig{},
				PushToPullRequestBranch: &PushToPullRequestBranchConfig{},
			},
			checkContains: []string{
				"name: Checkout repository",
				"name: Configure Git credentials",
			},
		},
		{
			name: "with GitHub App token",
			safeOutputs: &SafeOutputsConfig{
				GitHubApp: &GitHubAppConfig{
					AppID:      "12345",
					PrivateKey: "test-key",
				},
				CreatePullRequests: &CreatePullRequestsConfig{},
			},
			checkContains: []string{
				"token: ${{ steps.safe-outputs-app-token.outputs.token }}",
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
			name: "with per-config github-token",
			safeOutputs: &SafeOutputsConfig{
				CreatePullRequests: &CreatePullRequestsConfig{
					BaseSafeOutputConfig: BaseSafeOutputConfig{
						GitHubToken: "${{ secrets.GH_AW_CROSS_REPO_PAT }}",
					},
				},
			},
			checkContains: []string{
				"token: ${{ secrets.GH_AW_CROSS_REPO_PAT }}",
				"GIT_TOKEN: ${{ secrets.GH_AW_CROSS_REPO_PAT }}",
			},
		},
		{
			name: "with safe-outputs github-token",
			safeOutputs: &SafeOutputsConfig{
				GitHubToken:        "${{ secrets.SAFE_OUTPUTS_TOKEN }}",
				CreatePullRequests: &CreatePullRequestsConfig{},
			},
			checkContains: []string{
				"token: ${{ secrets.SAFE_OUTPUTS_TOKEN }}",
				"GIT_TOKEN: ${{ secrets.SAFE_OUTPUTS_TOKEN }}",
			},
		},
		{
			name: "cross-repo with custom token",
			safeOutputs: &SafeOutputsConfig{
				CreatePullRequests: &CreatePullRequestsConfig{
					BaseSafeOutputConfig: BaseSafeOutputConfig{
						GitHubToken: "${{ secrets.GH_AW_CROSS_REPO_PAT }}",
					},
					TargetRepoSlug: "org/target-repo",
				},
			},
			checkContains: []string{
				"repository: org/target-repo",
				"token: ${{ secrets.GH_AW_CROSS_REPO_PAT }}",
				"GIT_TOKEN: ${{ secrets.GH_AW_CROSS_REPO_PAT }}",
				`REPO_NAME: "org/target-repo"`,
				// Cross-repo checkout must not use github.ref_name
				"ref: ${{ github.base_ref || github.event.pull_request.base.ref || github.event.repository.default_branch }}",
			},
		},
		{
			name: "cross-repo without base-branch uses safe ref omitting github.ref_name",
			safeOutputs: &SafeOutputsConfig{
				CreatePullRequests: &CreatePullRequestsConfig{
					TargetRepoSlug: "org/other-repo",
				},
			},
			checkContains: []string{
				"ref: ${{ github.base_ref || github.event.pull_request.base.ref || github.event.repository.default_branch }}",
			},
			checkNotContains: []string{
				"github.ref_name",
			},
		},
		{
			name:      "trial mode cross-repo omits github.ref_name from checkout ref",
			trialMode: true,
			trialRepo: "org/trial-repo",
			safeOutputs: &SafeOutputsConfig{
				CreatePullRequests: &CreatePullRequestsConfig{},
			},
			checkContains: []string{
				"repository: org/trial-repo",
				"ref: ${{ github.base_ref || github.event.pull_request.base.ref || github.event.repository.default_branch }}",
			},
		},
		{
			name: "cross-repo with explicit base-branch uses base-branch not cross-repo fallback",
			safeOutputs: &SafeOutputsConfig{
				CreatePullRequests: &CreatePullRequestsConfig{
					TargetRepoSlug: "org/other-repo",
					BaseBranch:     "develop",
				},
			},
			checkContains: []string{
				"ref: develop",
			},
		},
		{
			name: "push-to-pull-request-branch with per-config token",
			safeOutputs: &SafeOutputsConfig{
				PushToPullRequestBranch: &PushToPullRequestBranchConfig{
					BaseSafeOutputConfig: BaseSafeOutputConfig{
						GitHubToken: "${{ secrets.PUSH_BRANCH_PAT }}",
					},
				},
			},
			checkContains: []string{
				"token: ${{ secrets.PUSH_BRANCH_PAT }}",
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
				"token: ${{ secrets.CREATE_PR_PAT }}",
				"GIT_TOKEN: ${{ secrets.CREATE_PR_PAT }}",
			},
		},
		{
			name: "default checkout ref uses github.base_ref || github.event.pull_request.base.ref || github.ref_name || github.event.repository.default_branch",
			safeOutputs: &SafeOutputsConfig{
				CreatePullRequests: &CreatePullRequestsConfig{},
			},
			checkContains: []string{
				"ref: ${{ github.base_ref || github.event.pull_request.base.ref || github.ref_name || github.event.repository.default_branch }}",
			},
		},
		{
			name: "checkout ref uses custom base-branch",
			safeOutputs: &SafeOutputsConfig{
				CreatePullRequests: &CreatePullRequestsConfig{
					BaseBranch: "develop",
				},
			},
			checkContains: []string{
				"ref: develop",
			},
		},
		{
			name: "checkout ref with release branch base-branch",
			safeOutputs: &SafeOutputsConfig{
				CreatePullRequests: &CreatePullRequestsConfig{
					BaseBranch: "release/v2.0",
				},
			},
			checkContains: []string{
				"ref: release/v2.0",
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
				Name:        "Test Workflow",
				SafeOutputs: tt.safeOutputs,
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

			steps := compiler.buildHandlerManagerStep(workflowData)

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
