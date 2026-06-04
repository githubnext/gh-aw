package workflow

import (
	"fmt"
	"strings"

	"github.com/github/gh-aw/pkg/logger"
)

var consolidatedSafeOutputsStepsLog = logger.New("workflow:compiler_safe_outputs_steps")

// buildExtractBaseBranchStep builds a step that extracts the base branch from the
// downloaded agent output JSON. The agent stores the resolved base branch in the
// safe output entry at generation time (in safe_outputs_handlers.cjs), so the
// apply-time checkout can use it directly instead of inferring from event context.
//
// This is the key decoupling that resolves the known limitation for issue_comment
// events on PRs targeting non-default branches: the checkout step can now use the
// correct base branch regardless of event type.
//
// Cross-repo items (where the entry's `repo` field is set and differs from the
// workflow repository) are skipped. Their base_branch belongs to a different
// repository and must not be used to checkout the workflow repo, as that branch
// will not exist there and the checkout step would fail.
//
// The step writes the extracted branch to GITHUB_OUTPUT as "base-branch" so the
// checkout step can reference it via ${{ steps.extract-base-branch.outputs.base-branch }}.
func buildExtractBaseBranchStep() []string {
	return []string{
		"      - name: Extract base branch from agent output\n",
		"        id: extract-base-branch\n",
		"        if: steps.download-agent-output.outcome == 'success'\n",
		fmt.Sprintf("        uses: %s\n", getActionPin("actions/github-script")),
		"        with:\n",
		"          script: |\n",
		fmt.Sprintf("            const { setupGlobals } = require('%s/setup_globals.cjs');\n", SetupActionDestination),
		"            setupGlobals(core, github, context, exec, io, getOctokit);\n",
		fmt.Sprintf("            const { main } = require('%s/extract_base_branch_from_agent_output.cjs');\n", SetupActionDestination),
		"            await main();\n",
	}
}

// appendSparseCheckoutLines appends the "sparse-checkout" block lines to steps when
// sparsePatterns is non-empty. Each pattern is trimmed of leading/trailing whitespace.
func appendSparseCheckoutLines(steps []string, sparsePatterns []string) []string {
	if len(sparsePatterns) == 0 {
		return steps
	}
	steps = append(steps, "          sparse-checkout: |\n")
	for _, pattern := range sparsePatterns {
		steps = append(steps, fmt.Sprintf("            %s\n", strings.TrimSpace(pattern)))
	}
	return steps
}

// buildSharedPRCheckoutSteps builds checkout and git configuration steps that are shared
// between create-pull-request and push-to-pull-request-branch operations.
// These steps are added once with a combined condition to avoid duplication.
func (c *Compiler) buildSharedPRCheckoutSteps(data *WorkflowData) []string {
	consolidatedSafeOutputsStepsLog.Print("Building shared PR checkout steps")
	var steps []string
	fetchDepth := 1
	var sparsePatterns []string

	// Build a single CheckoutManager so we can query both the default and cross-repo entries.
	checkoutMgr := NewCheckoutManager(data.CheckoutConfigs)

	// Same-repo fallback: use the default (workspace-root) checkout's fetch-depth and
	// sparse-checkout patterns. Both are overridden below for cross-repo targets once
	// targetRepoSlug is known.
	if defaultCheckout := checkoutMgr.GetDefaultCheckoutOverride(); defaultCheckout != nil {
		if defaultCheckout.fetchDepth != nil {
			fetchDepth = *defaultCheckout.fetchDepth
			consolidatedSafeOutputsStepsLog.Printf("Using custom checkout fetch-depth for safe_outputs: %d", fetchDepth)
		}
		if len(defaultCheckout.sparsePatterns) > 0 {
			sparsePatterns = defaultCheckout.sparsePatterns
			consolidatedSafeOutputsStepsLog.Printf("Using %d sparse-checkout pattern(s) from default checkout for safe_outputs", len(sparsePatterns))
		}
	}

	// Determine which token to use for checkout
	// Uses resolvePRCheckoutToken for consistent token resolution (GitHub App or PAT chain)
	checkoutToken, _ := resolvePRCheckoutToken(data.SafeOutputs)
	gitRemoteToken := checkoutToken

	// Build combined condition: execute if either create_pull_request or push_to_pull_request_branch will run
	var condition ConditionNode
	if data.SafeOutputs.CreatePullRequests != nil && data.SafeOutputs.PushToPullRequestBranch != nil {
		// Both enabled: combine conditions with OR
		condition = BuildOr(
			BuildSafeOutputType("create_pull_request"),
			BuildSafeOutputType("push_to_pull_request_branch"),
		)
	} else if data.SafeOutputs.CreatePullRequests != nil {
		// Only create_pull_request
		condition = BuildSafeOutputType("create_pull_request")
	} else {
		// Only push_to_pull_request_branch
		condition = BuildSafeOutputType("push_to_pull_request_branch")
	}

	// Determine target repository for checkout and git config.
	// Only git-writing operations (create-pull-request, push-to-pull-request-branch) influence
	// the shared git checkout; update-pull-request is API-only and must NOT affect the git remote.
	// Priority: create-pull-request target-repo > push-to-pull-request-branch target-repo > trialLogicalRepoSlug > default (source repo)
	var targetRepoSlug string
	if data.SafeOutputs.CreatePullRequests != nil && data.SafeOutputs.CreatePullRequests.TargetRepoSlug != "" {
		targetRepoSlug = data.SafeOutputs.CreatePullRequests.TargetRepoSlug
		consolidatedSafeOutputsStepsLog.Printf("Using target-repo from create-pull-request: %s", targetRepoSlug)
	} else if data.SafeOutputs.PushToPullRequestBranch != nil && data.SafeOutputs.PushToPullRequestBranch.TargetRepoSlug != "" {
		targetRepoSlug = data.SafeOutputs.PushToPullRequestBranch.TargetRepoSlug
		consolidatedSafeOutputsStepsLog.Printf("Using target-repo from push-to-pull-request-branch: %s", targetRepoSlug)
	} else if c.trialMode && c.trialLogicalRepoSlug != "" {
		targetRepoSlug = c.trialLogicalRepoSlug
		consolidatedSafeOutputsStepsLog.Printf("Using trialLogicalRepoSlug: %s", targetRepoSlug)
	}

	// Wildcard target-repo: the agent chooses the target repo at runtime.
	// Instead of a single cross-repo checkout, emit checkout steps for ALL repos
	// declared in checkout: configs (mirroring the agent job), so that any of them
	// can be targeted by the agent's safe output messages at runtime.
	// The JS handler uses findRepoCheckout() to locate the correct directory.
	if targetRepoSlug == "*" {
		consolidatedSafeOutputsStepsLog.Print("Wildcard target-repo: generating multi-repo checkout steps for safe_outputs")
		return c.buildMultiRepoCheckoutSteps(data, checkoutMgr, checkoutToken, gitRemoteToken, condition)
	}

	// For cross-repo targets, override fetch-depth and sparse-checkout patterns
	// from the checkout: config entry that targets the same repository.  The agent
	// job already uses these values; the safe_outputs job must mirror them so that
	// (a) large repos are not checked out in full unnecessarily and (b) the working
	// tree is consistent with what the agent operated on.
	if targetRepoSlug != "" {
		if targetEntry := checkoutMgr.GetCheckoutForRepository(targetRepoSlug); targetEntry != nil {
			if targetEntry.fetchDepth != nil {
				fetchDepth = *targetEntry.fetchDepth
				consolidatedSafeOutputsStepsLog.Printf("Using checkout fetch-depth for cross-repo target %s: %d", targetRepoSlug, fetchDepth)
			}
			if len(targetEntry.sparsePatterns) > 0 {
				sparsePatterns = targetEntry.sparsePatterns
				consolidatedSafeOutputsStepsLog.Printf("Using %d sparse-checkout pattern(s) for cross-repo target %s", len(sparsePatterns), targetRepoSlug)
			}
		}
	}

	// Determine the ref (branch) to checkout
	// Priority: create-pull-request base-branch > extracted base-branch from agent output > fallback expression
	// This is critical: we must checkout the base branch, not github.sha (the triggering commit),
	// because github.sha might be an older commit with different workflow files. A shallow clone
	// of an old commit followed by git fetch/checkout may not properly update all files,
	// leading to spurious "workflow file changed" errors on push.
	//
	// The extract-base-branch step reads the base_branch field from the agent output, which was
	// stored by safe_outputs_handlers.cjs at agent-execution time using getBaseBranch(). This makes
	// the checkout independent of event context: the correct base branch is embedded in the safe
	// output payload itself rather than inferred from event-specific GitHub Actions expressions.
	//
	// The event-context fallbacks remain as a safety net for cases where the agent output is
	// unavailable or does not contain a base_branch (e.g., older agent output format):
	// - github.base_ref: set for pull_request/pull_request_target events
	// - github.event.pull_request.base.ref: set for pull_request_review, pull_request_review_comment events
	// - github.event.repository.default_branch: fallback for edge cases
	//
	const baseBranchFallbackExpr = "${{ steps.extract-base-branch.outputs.base-branch || github.base_ref || github.event.pull_request.base.ref || github.ref_name || github.event.repository.default_branch }}"
	// Cross-repo fallback omits github.ref_name because it refers to the branch in the triggering repository,
	// which may not exist in the target repository (e.g., when triggered via workflow_dispatch from a feature branch).
	const crossRepoFallbackExpr = "${{ steps.extract-base-branch.outputs.base-branch || github.base_ref || github.event.pull_request.base.ref || github.event.repository.default_branch }}"
	var checkoutRef string
	if data.SafeOutputs.CreatePullRequests != nil && data.SafeOutputs.CreatePullRequests.BaseBranch != "" {
		checkoutRef = data.SafeOutputs.CreatePullRequests.BaseBranch
		consolidatedSafeOutputsStepsLog.Printf("Using custom base-branch from create-pull-request for checkout ref: %s", checkoutRef)
	} else if targetRepoSlug != "" {
		// Cross-repo checkout: avoid github.ref_name which refers to the triggering branch,
		// not a branch in the target repository.
		checkoutRef = crossRepoFallbackExpr
		consolidatedSafeOutputsStepsLog.Printf("Using cross-repo fallback base branch expression for checkout ref (no github.ref_name)")
	} else {
		checkoutRef = baseBranchFallbackExpr
		consolidatedSafeOutputsStepsLog.Printf("Using fallback base branch expression for checkout ref")
	}

	// Step 1a: For comment-triggered privileged events, force checkout to trusted default branch.
	// This avoids checking out potentially untrusted refs inferred from event context.
	commentEventCondition := BuildDisjunction(
		false,
		BuildEventTypeEquals("issue_comment"),
		BuildEventTypeEquals("pull_request_review_comment"),
	)
	nonCommentEventCondition := BuildAnd(
		BuildNotEquals(BuildPropertyAccess("github.event_name"), BuildStringLiteral("issue_comment")),
		BuildNotEquals(BuildPropertyAccess("github.event_name"), BuildStringLiteral("pull_request_review_comment")),
	)

	// Only emit the trusted-default-branch path for same-repo checkouts.
	// Cross-repo checkouts rely on explicit target-repo branch selection.
	if targetRepoSlug == "" {
		steps = append(steps, "      - name: Checkout repository (trusted default branch for comment events)\n")
		steps = append(steps, fmt.Sprintf("        if: %s\n", RenderCondition(BuildAnd(condition, commentEventCondition))))
		steps = append(steps, fmt.Sprintf("        uses: %s\n", getActionPin("actions/checkout")))
		steps = append(steps, "        with:\n")
		steps = append(steps, "          ref: ${{ github.event.repository.default_branch }}\n")
		steps = append(steps, fmt.Sprintf("          token: %s\n", checkoutToken))
		steps = append(steps, "          persist-credentials: false\n")
		steps = append(steps, fmt.Sprintf("          fetch-depth: %d\n", fetchDepth))
		steps = appendSparseCheckoutLines(steps, sparsePatterns)
	}

	// Step 1b: Checkout repository with conditional execution
	steps = append(steps, "      - name: Checkout repository\n")
	if targetRepoSlug == "" {
		steps = append(steps, fmt.Sprintf("        if: %s\n", RenderCondition(BuildAnd(condition, nonCommentEventCondition))))
	} else {
		steps = append(steps, fmt.Sprintf("        if: %s\n", RenderCondition(condition)))
	}
	steps = append(steps, fmt.Sprintf("        uses: %s\n", getActionPin("actions/checkout")))
	steps = append(steps, "        with:\n")

	// Set repository parameter if checking out a different repository
	if targetRepoSlug != "" {
		steps = append(steps, fmt.Sprintf("          repository: %s\n", targetRepoSlug))
		consolidatedSafeOutputsStepsLog.Printf("Added repository parameter: %s", targetRepoSlug)
	}

	// Set ref to checkout the base branch, not github.sha
	steps = append(steps, fmt.Sprintf("          ref: %s\n", checkoutRef))
	steps = append(steps, fmt.Sprintf("          token: %s\n", checkoutToken))
	steps = append(steps, "          persist-credentials: false\n")
	steps = append(steps, fmt.Sprintf("          fetch-depth: %d\n", fetchDepth))
	steps = appendSparseCheckoutLines(steps, sparsePatterns)

	// Step 2: Configure Git credentials with conditional execution
	// Security: Pass GitHub token through environment variable to prevent template injection

	// Determine REPO_NAME value based on target repository
	repoNameValue := "${{ github.repository }}"
	if targetRepoSlug != "" {
		repoNameValue = fmt.Sprintf("%q", targetRepoSlug)
		consolidatedSafeOutputsStepsLog.Printf("Using target repo for REPO_NAME: %s", targetRepoSlug)
	}

	gitConfigSteps := []string{
		"      - name: Configure Git credentials\n",
		fmt.Sprintf("        if: %s\n", RenderCondition(condition)),
		"        env:\n",
		fmt.Sprintf("          REPO_NAME: %s\n", repoNameValue),
		"          SERVER_URL: ${{ github.server_url }}\n",
		fmt.Sprintf("          GIT_TOKEN: %s\n", gitRemoteToken),
		"        run: |\n",
		"          git config --global user.email \"github-actions[bot]@users.noreply.github.com\"\n",
		"          git config --global user.name \"github-actions[bot]\"\n",
		"          git config --global am.keepcr true\n",
		"          # Re-authenticate git with GitHub token\n",
		"          SERVER_URL_STRIPPED=\"${SERVER_URL#https://}\"\n",
		"          git remote set-url origin \"https://x-access-token:${GIT_TOKEN}@${SERVER_URL_STRIPPED}/${REPO_NAME}.git\"\n",
		"          echo \"Git configured with standard GitHub Actions identity\"\n",
	}
	steps = append(steps, gitConfigSteps...)

	// Step 3: Fetch additional refs for cross-repo checkouts when declared in checkout:.
	// This mirrors what the agent job emits via generateFetchStepLines and ensures the
	// safe_outputs job has the same remote-tracking refs available when applying bundles.
	// Without this, applyBundleToBranch must fall back to per-SHA git fetch (prerequisite
	// recovery), which requires uploadpack.allowReachableSHA1InWant on the server.
	if targetRepoSlug != "" {
		if matchedEntry := checkoutMgr.GetCheckoutForRepository(targetRepoSlug); matchedEntry != nil && len(matchedEntry.fetchRefs) > 0 {
			consolidatedSafeOutputsStepsLog.Printf("Adding fetch refs step for cross-repo target %s (%d refs)", targetRepoSlug, len(matchedEntry.fetchRefs))
			if fetchStep := buildSafeOutputsFetchRefsStep(targetRepoSlug, checkoutToken, matchedEntry.fetchRefs, matchedEntry.fetchDepth, RenderCondition(condition)); fetchStep != "" {
				steps = append(steps, fetchStep)
			}
		}
	}

	consolidatedSafeOutputsStepsLog.Printf("Added shared checkout with condition: %s", condition.Render())
	return steps
}

// buildMultiRepoCheckoutSteps generates checkout steps for ALL repositories declared
// in the checkout: config, mirroring what the agent job does. This is used when
// target-repo is "*" (wildcard), meaning the agent decides at runtime which repository
// to target. Each repository is checked out to its configured path (or workspace root
// for the default checkout), so the JS handler can locate it via findRepoCheckout().
//
// The git credential configuration step sets up authentication for ALL checked-out
// repositories, enabling push operations to any of them at runtime.
func (c *Compiler) buildMultiRepoCheckoutSteps(data *WorkflowData, checkoutMgr *CheckoutManager, checkoutToken, gitRemoteToken string, condition ConditionNode) []string {
	var steps []string
	conditionStr := RenderCondition(condition)

	// Step 1: Checkout the default (workspace root) repository.
	// This mirrors the single-repo path but without a cross-repo repository: parameter.
	defaultCheckout := checkoutMgr.GetDefaultCheckoutOverride()
	defaultFetchDepth := 1
	var defaultSparsePatterns []string
	if defaultCheckout != nil {
		if defaultCheckout.fetchDepth != nil {
			defaultFetchDepth = *defaultCheckout.fetchDepth
		}
		if len(defaultCheckout.sparsePatterns) > 0 {
			defaultSparsePatterns = defaultCheckout.sparsePatterns
		}
	}

	// Checkout ref: use extracted base branch from agent output with event-context fallbacks.
	// For comment-triggered privileged events, force checkout to trusted default branch.
	const baseBranchFallbackExpr = "${{ (github.event_name == 'issue_comment' || github.event_name == 'pull_request_review_comment') && github.event.repository.default_branch || steps.extract-base-branch.outputs.base-branch || github.base_ref || github.event.pull_request.base.ref || github.ref_name || github.event.repository.default_branch }}"
	steps = append(steps, "      - name: Checkout repository\n")
	steps = append(steps, fmt.Sprintf("        if: %s\n", conditionStr))
	steps = append(steps, fmt.Sprintf("        uses: %s\n", getActionPin("actions/checkout")))
	steps = append(steps, "        with:\n")
	steps = append(steps, fmt.Sprintf("          ref: %s\n", baseBranchFallbackExpr))
	steps = append(steps, fmt.Sprintf("          token: %s\n", checkoutToken))
	steps = append(steps, "          persist-credentials: false\n")
	steps = append(steps, fmt.Sprintf("          fetch-depth: %d\n", defaultFetchDepth))
	steps = appendSparseCheckoutLines(steps, defaultSparsePatterns)

	// Step 2: Checkout additional repositories from checkout: configs into their paths.
	// Only include entries that have a non-empty repository and path (cross-repo checkouts).
	for _, cfg := range data.CheckoutConfigs {
		if cfg == nil || cfg.Repository == "" || cfg.Path == "" {
			continue
		}
		if cfg.Wiki {
			// Wiki checkouts are not relevant for PR/push operations.
			continue
		}

		entryFetchDepth := 1
		if cfg.FetchDepth != nil {
			entryFetchDepth = *cfg.FetchDepth
		}
		var entrySparsePatterns []string
		if cfg.SparseCheckout != "" {
			entrySparsePatterns = strings.Split(cfg.SparseCheckout, "\n")
		}

		// Use the safe-outputs token for authentication (consistent with single-repo path)
		entryToken := checkoutToken
		if cfg.GitHubToken != "" {
			entryToken = cfg.GitHubToken
		}

		steps = append(steps, fmt.Sprintf("      - name: Checkout %s into %s\n", cfg.Repository, cfg.Path))
		steps = append(steps, fmt.Sprintf("        if: %s\n", conditionStr))
		steps = append(steps, fmt.Sprintf("        uses: %s\n", getActionPin("actions/checkout")))
		steps = append(steps, "        with:\n")
		steps = append(steps, fmt.Sprintf("          repository: %s\n", cfg.Repository))
		steps = append(steps, fmt.Sprintf("          path: %s\n", cfg.Path))
		steps = append(steps, fmt.Sprintf("          token: %s\n", entryToken))
		steps = append(steps, "          persist-credentials: false\n")
		steps = append(steps, fmt.Sprintf("          fetch-depth: %d\n", entryFetchDepth))
		steps = appendSparseCheckoutLines(steps, entrySparsePatterns)

		consolidatedSafeOutputsStepsLog.Printf("Added multi-repo checkout: %s -> %s", cfg.Repository, cfg.Path)
	}

	// Step 3: Configure Git credentials for ALL repositories.
	// Set up authentication for the workspace root and each subdirectory checkout.
	gitConfigSteps := []string{
		"      - name: Configure Git credentials\n",
		fmt.Sprintf("        if: %s\n", conditionStr),
		"        env:\n",
		"          REPO_NAME: ${{ github.repository }}\n",
		"          SERVER_URL: ${{ github.server_url }}\n",
		fmt.Sprintf("          GIT_TOKEN: %s\n", gitRemoteToken),
		"        run: |\n",
		"          git config --global user.email \"github-actions[bot]@users.noreply.github.com\"\n",
		"          git config --global user.name \"github-actions[bot]\"\n",
		"          git config --global am.keepcr true\n",
		"          # Re-authenticate git with GitHub token for workspace root\n",
		"          SERVER_URL_STRIPPED=\"${SERVER_URL#https://}\"\n",
		"          git remote set-url origin \"https://x-access-token:${GIT_TOKEN}@${SERVER_URL_STRIPPED}/${REPO_NAME}.git\"\n",
	}

	// Also configure credentials for each subdirectory checkout
	for _, cfg := range data.CheckoutConfigs {
		if cfg == nil || cfg.Repository == "" || cfg.Path == "" || cfg.Wiki {
			continue
		}
		gitConfigSteps = append(gitConfigSteps,
			fmt.Sprintf("          # Re-authenticate git for %s\n", cfg.Repository),
			fmt.Sprintf("          git -C \"%s\" remote set-url origin \"https://x-access-token:${GIT_TOKEN}@${SERVER_URL_STRIPPED}/%s.git\"\n", cfg.Path, cfg.Repository),
		)
	}

	gitConfigSteps = append(gitConfigSteps,
		"          echo \"Git configured with standard GitHub Actions identity\"\n",
	)
	steps = append(steps, gitConfigSteps...)

	// Step 4: Fetch additional refs for each repository that declares them.
	for _, cfg := range data.CheckoutConfigs {
		if cfg == nil || cfg.Repository == "" || cfg.Path == "" || cfg.Wiki {
			continue
		}
		if entry := checkoutMgr.GetCheckoutForRepository(cfg.Repository); entry != nil && len(entry.fetchRefs) > 0 {
			consolidatedSafeOutputsStepsLog.Printf("Adding fetch refs step for multi-repo target %s (%d refs)", cfg.Repository, len(entry.fetchRefs))
			if fetchStep := buildSafeOutputsMultiRepoFetchRefsStep(cfg.Repository, cfg.Path, checkoutToken, entry.fetchRefs, entry.fetchDepth, conditionStr); fetchStep != "" {
				steps = append(steps, fetchStep)
			}
		}
	}

	consolidatedSafeOutputsStepsLog.Printf("Added multi-repo checkout steps with condition: %s", condition.Render())
	return steps
}

// buildSafeOutputsMultiRepoFetchRefsStep generates a conditional "Fetch additional refs"
// step for a repository checked out into a subdirectory (multi-repo wildcard scenario).
// Unlike buildSafeOutputsFetchRefsStep, this step targets a specific subdirectory via -C.
func buildSafeOutputsMultiRepoFetchRefsStep(repoSlug, path, token string, fetchRefs []string, fetchDepth *int, condition string) string {
	if len(fetchRefs) == 0 {
		return ""
	}
	refspecs := make([]string, 0, len(fetchRefs))
	for _, ref := range fetchRefs {
		refspecs = append(refspecs, fmt.Sprintf("'%s'", fetchRefToRefspec(ref)))
	}

	depthFlag := ""
	effectiveDepth := 1
	if fetchDepth != nil {
		effectiveDepth = *fetchDepth
	}
	if effectiveDepth > 0 {
		depthFlag = fmt.Sprintf(" --depth=%d", effectiveDepth)
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "      - name: Fetch additional refs for %s\n", repoSlug)
	if condition != "" {
		fmt.Fprintf(&sb, "        if: %s\n", condition)
	}
	sb.WriteString("        env:\n")
	fmt.Fprintf(&sb, "          GH_AW_FETCH_TOKEN: %s\n", token)
	sb.WriteString("        run: |\n")
	sb.WriteString("          header=$(printf \"x-access-token:%s\" \"${GH_AW_FETCH_TOKEN}\" | base64 -w 0)\n")
	fmt.Fprintf(&sb, "          git -C \"%s\" -c \"http.extraheader=Authorization: Basic ${header}\" fetch origin%s %s\n", path, depthFlag, strings.Join(refspecs, " "))
	return sb.String()
}

// buildSafeOutputsFetchRefsStep generates a conditional "Fetch additional refs" step
// for the safe_outputs job's cross-repo checkout.
//
// Unlike the agent-job fetch step (which targets a subdirectory via -C and runs
// unconditionally), this step:
//   - Runs under the same condition as the shared PR checkout step
//   - Targets the workspace root — safe_outputs checks out the single cross-repo
//     target to the workspace root (no path: parameter), never to a subdirectory.
//     NOTE: safe_outputs supports only one cross-repo checkout at a time. If multiple
//     distinct target repositories were needed, this step would need a -C <path>
//     argument and the checkout step would need a path: parameter, which is not
//     currently supported.
//   - Uses the resolved safe_outputs checkout token (from resolvePRCheckoutToken)
//     rather than the CheckoutConfig's token
func buildSafeOutputsFetchRefsStep(repoSlug, token string, fetchRefs []string, fetchDepth *int, condition string) string {
	if len(fetchRefs) == 0 {
		return ""
	}
	refspecs := make([]string, 0, len(fetchRefs))
	for _, ref := range fetchRefs {
		refspecs = append(refspecs, fmt.Sprintf("'%s'", fetchRefToRefspec(ref)))
	}

	// Mirror the fetch-depth from the checkout step to avoid expanding a shallow clone.
	depthFlag := ""
	effectiveDepth := 1
	if fetchDepth != nil {
		effectiveDepth = *fetchDepth
	}
	if effectiveDepth > 0 {
		depthFlag = fmt.Sprintf(" --depth=%d", effectiveDepth)
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "      - name: Fetch additional refs for %s\n", repoSlug)
	if condition != "" {
		fmt.Fprintf(&sb, "        if: %s\n", condition)
	}
	sb.WriteString("        env:\n")
	fmt.Fprintf(&sb, "          GH_AW_FETCH_TOKEN: %s\n", token)
	sb.WriteString("        run: |\n")
	sb.WriteString("          header=$(printf \"x-access-token:%s\" \"${GH_AW_FETCH_TOKEN}\" | base64 -w 0)\n")
	fmt.Fprintf(&sb, "          git -c \"http.extraheader=Authorization: Basic ${header}\" fetch origin%s %s\n", depthFlag, strings.Join(refspecs, " "))
	return sb.String()
}

// buildHandlerManagerStep builds a single step that uses the safe output handler manager
// to dispatch messages to appropriate handlers. This replaces multiple individual steps
// with a single dispatcher step that processes all safe output types.
func (c *Compiler) buildHandlerManagerStep(data *WorkflowData) ([]string, error) {
	consolidatedSafeOutputsStepsLog.Print("Building handler manager step")

	var steps []string

	// Add per-handler GitHub App token minting steps before the handler manager step.
	// These run before the main handler step so the minted token expressions (e.g.
	// ${{ steps.create-check-run-app-token.outputs.token }}) are resolved at runtime.
	if data.SafeOutputs != nil && data.SafeOutputs.CreateCheckRun != nil && data.SafeOutputs.CreateCheckRun.GitHubApp != nil {
		consolidatedSafeOutputsStepsLog.Print("Adding per-handler GitHub App token minting step for create-check-run")
		permissions := NewPermissionsContentsReadChecksWrite()
		for _, step := range c.buildGitHubAppTokenMintStep(data.SafeOutputs.CreateCheckRun.GitHubApp, permissions, "") {
			steps = append(steps, replaceStepID(step, "safe-outputs-app-token", "create-check-run-app-token"))
		}
	}

	// Step name and metadata
	steps = append(steps, "      - name: Process Safe Outputs\n")
	steps = append(steps, "        id: process_safe_outputs\n")
	steps = append(steps, fmt.Sprintf("        uses: %s\n", getCachedActionPin("actions/github-script", data)))

	// Environment variables
	steps = append(steps, "        env:\n")
	steps = append(steps, "          GH_AW_AGENT_OUTPUT: ${{ steps.setup-agent-output-env.outputs.GH_AW_AGENT_OUTPUT }}\n")
	steps = append(steps, "          GH_AW_COMMENT_ID: ${{ needs.activation.outputs.comment_id }}\n")

	// Add allowed domains configuration for URL sanitization in safe output handlers.
	// Without this, sanitizeContent() in safe_output_handler_manager.cjs only allows
	// default GitHub domains, causing user-configured allowed domains to be redacted.
	var domainsStr string
	if data.SafeOutputs != nil && len(data.SafeOutputs.AllowedDomains) > 0 {
		// allowed-domains: additional domains unioned with engine/network base set; supports ecosystem identifiers
		expanded, err := c.computeExpandedAllowedDomainsForSanitization(data)
		if err != nil {
			return nil, err
		}
		domainsStr = expanded
	} else {
		computed, err := c.computeAllowedDomainsForSanitization(data)
		if err != nil {
			return nil, err
		}
		domainsStr = computed
	}
	if domainsStr != "" {
		steps = append(steps, fmt.Sprintf("          GH_AW_ALLOWED_DOMAINS: %q\n", domainsStr))
	}
	// Pass GitHub server/API URLs so buildAllowedDomains() can add GHES domains dynamically
	steps = append(steps, "          GITHUB_SERVER_URL: ${{ github.server_url }}\n")
	steps = append(steps, "          GITHUB_API_URL: ${{ github.api_url }}\n")

	// Note: The project handler manager has been removed.
	// All project-related operations are now handled by the unified handler.

	// Add GH_AW_SAFE_OUTPUT_JOBS so the handler manager knows which message types are
	// handled by custom safe-output job steps and should be silently skipped rather than
	// reported as "No handler loaded for message type '...'".
	if customJobsJSON := buildCustomSafeOutputJobsJSON(data); customJobsJSON != "" {
		steps = append(steps, fmt.Sprintf("          GH_AW_SAFE_OUTPUT_JOBS: %q\n", customJobsJSON))
		consolidatedSafeOutputsStepsLog.Print("Added GH_AW_SAFE_OUTPUT_JOBS env var for custom safe job types")
	}

	// Add GH_AW_SAFE_OUTPUT_SCRIPTS so the handler manager can load inline script handlers.
	// The env var maps normalized script names to their .cjs filenames in the actions folder.
	if customScriptsJSON := buildCustomSafeOutputScriptsJSON(data); customScriptsJSON != "" {
		steps = append(steps, fmt.Sprintf("          GH_AW_SAFE_OUTPUT_SCRIPTS: %q\n", customScriptsJSON))
		consolidatedSafeOutputsStepsLog.Print("Added GH_AW_SAFE_OUTPUT_SCRIPTS env var for custom script handlers")
	}

	// Add GH_AW_SAFE_OUTPUT_ACTIONS so the handler manager can load custom action handlers.
	// The env var maps normalized action names to themselves (reserved for future extensibility).
	if customActionsJSON := buildCustomSafeOutputActionsJSON(data); customActionsJSON != "" {
		steps = append(steps, fmt.Sprintf("          GH_AW_SAFE_OUTPUT_ACTIONS: %q\n", customActionsJSON))
		consolidatedSafeOutputsStepsLog.Print("Added GH_AW_SAFE_OUTPUT_ACTIONS env var for custom action handlers")
	}

	// Add custom safe output env vars
	c.addCustomSafeOutputEnvVars(&steps, data)

	// Add handler manager config as JSON
	c.addHandlerManagerConfigEnvVar(&steps, data)

	// Add all safe output configuration env vars (still needed by individual handlers)
	c.addAllSafeOutputConfigEnvVars(&steps, data)

	// Add extra empty commit token if create-pull-request or push-to-pull-request-branch is configured.
	// This token is used to push an empty commit after code changes to trigger CI events,
	// working around the GITHUB_TOKEN limitation where events don't trigger other workflows.
	// Only emit this env var when one of these safe outputs is actually configured.
	if usesPatchesAndCheckouts(data.SafeOutputs) {
		var ciTriggerToken string
		if data.SafeOutputs.CreatePullRequests != nil && data.SafeOutputs.CreatePullRequests.GithubTokenForExtraEmptyCommit != "" {
			ciTriggerToken = data.SafeOutputs.CreatePullRequests.GithubTokenForExtraEmptyCommit
		} else if data.SafeOutputs.PushToPullRequestBranch != nil && data.SafeOutputs.PushToPullRequestBranch.GithubTokenForExtraEmptyCommit != "" {
			ciTriggerToken = data.SafeOutputs.PushToPullRequestBranch.GithubTokenForExtraEmptyCommit
		}

		switch ciTriggerToken {
		case "app":
			steps = append(steps, "          GH_AW_CI_TRIGGER_TOKEN: ${{ steps.safe-outputs-app-token.outputs.token || '' }}\n")
			consolidatedSafeOutputsStepsLog.Print("Extra empty commit using GitHub App token")
		default:
			// Use the magic GH_AW_CI_TRIGGER_TOKEN secret (default behavior when not explicitly configured)
			steps = append(steps, fmt.Sprintf("          GH_AW_CI_TRIGGER_TOKEN: %s\n", getEffectiveCITriggerGitHubToken(ciTriggerToken)))
			consolidatedSafeOutputsStepsLog.Print("Extra empty commit using GH_AW_CI_TRIGGER_TOKEN")
		}
	}

	// Add GH_AW_PROJECT_URL and GH_AW_PROJECT_GITHUB_TOKEN environment variables for project operations
	// These are set from the project URL and token configured in any project-related safe-output:
	// - update-project
	// - create-project-status-update
	// - create-project
	//
	// The project field is REQUIRED in update-project and create-project-status-update (enforced by schema validation)
	// Agents can optionally override this per-message by including a project field in their output
	//
	// Note: If multiple project configs are present, we prefer update-project > create-project-status-update > create-project
	// This is only relevant for the environment variables - each configuration must explicitly specify its own settings
	projectURL, projectToken := resolveProjectURLAndToken(data.SafeOutputs)

	if projectURL != "" {
		steps = append(steps, fmt.Sprintf("          GH_AW_PROJECT_URL: %q\n", projectURL))
	}

	if projectToken != "" {
		steps = append(steps, fmt.Sprintf("          GH_AW_PROJECT_GITHUB_TOKEN: %s\n", projectToken))
	}

	// Add GH_AW_ASSIGN_TO_AGENT_TOKEN when assign-to-agent is configured OR when create-issue
	// or create-pull-request is configured with copilot in assignees. All handlers create a
	// dedicated Octokit using this token (agent token preference chain), which is required
	// because the Copilot assignment API only accepts PATs (not GitHub App tokens). This env
	// var is evaluated as a GitHub Actions expression, so it resolves to the actual token value
	// before the step runs.
	if data.SafeOutputs != nil && data.SafeOutputs.AssignToAgent != nil {
		agentTokenStr := getEffectiveCopilotCodingAgentGitHubToken(data.SafeOutputs.AssignToAgent.GitHubToken)
		//nolint:gosec // G101: False positive - this is a GitHub Actions expression template, not a hardcoded credential
		steps = append(steps, fmt.Sprintf("          GH_AW_ASSIGN_TO_AGENT_TOKEN: %s\n", agentTokenStr))
		consolidatedSafeOutputsStepsLog.Print("Added GH_AW_ASSIGN_TO_AGENT_TOKEN env var for assign-to-agent handler")
	} else if data.SafeOutputs != nil && data.SafeOutputs.CreateIssues != nil && hasCopilotAssignee(data.SafeOutputs.CreateIssues.Assignees) {
		agentTokenStr := getEffectiveCopilotCodingAgentGitHubToken(data.SafeOutputs.CreateIssues.GitHubToken)
		//nolint:gosec // G101: False positive - this is a GitHub Actions expression template, not a hardcoded credential
		steps = append(steps, fmt.Sprintf("          GH_AW_ASSIGN_TO_AGENT_TOKEN: %s\n", agentTokenStr))
		consolidatedSafeOutputsStepsLog.Print("Added GH_AW_ASSIGN_TO_AGENT_TOKEN env var for create-issue copilot assignment handler")
	} else if data.SafeOutputs != nil && data.SafeOutputs.CreatePullRequests != nil && hasCopilotAssignee(data.SafeOutputs.CreatePullRequests.Assignees) {
		agentTokenStr := getEffectiveCopilotCodingAgentGitHubToken(data.SafeOutputs.CreatePullRequests.GitHubToken)
		//nolint:gosec // G101: False positive - this is a GitHub Actions expression template, not a hardcoded credential
		steps = append(steps, fmt.Sprintf("          GH_AW_ASSIGN_TO_AGENT_TOKEN: %s\n", agentTokenStr))
		consolidatedSafeOutputsStepsLog.Print("Added GH_AW_ASSIGN_TO_AGENT_TOKEN env var for create-pull-request copilot assignment handler")
	}

	// Add GH_AW_AGENT_SESSION_TOKEN when create-agent-session is configured.
	// The create_agent_session handler passes this token as GH_TOKEN to the gh CLI
	// (agent token preference chain), which is required because the default GITHUB_TOKEN
	// does not have permission to create agent sessions via gh agent-task create.
	if data.SafeOutputs != nil && data.SafeOutputs.CreateAgentSessions != nil {
		agentSessionTokenStr := getEffectiveCopilotCodingAgentGitHubToken(data.SafeOutputs.CreateAgentSessions.GitHubToken)
		//nolint:gosec // G101: False positive - this is a GitHub Actions expression template, not a hardcoded credential
		steps = append(steps, fmt.Sprintf("          GH_AW_AGENT_SESSION_TOKEN: %s\n", agentSessionTokenStr))
		consolidatedSafeOutputsStepsLog.Print("Added GH_AW_AGENT_SESSION_TOKEN env var for create-agent-session handler")
	}

	// When create-pull-request or push-to-pull-request-branch is configured with a custom token
	// (including GitHub App), expose that token as GITHUB_TOKEN so that git CLI operations in
	// the JavaScript handlers can authenticate. The create_pull_request.cjs handler reads
	// process.env.GITHUB_TOKEN to enable dynamic repo checkout for multi-repo/cross-repo
	// scenarios (allowed-repos). Without this, the handler falls back to the default
	// repo-scoped token which lacks access to other repos.
	if usesPatchesAndCheckouts(data.SafeOutputs) {
		gitToken, isCustom := resolvePRCheckoutToken(data.SafeOutputs)
		// Only override GITHUB_TOKEN when a custom token (app or PAT) is explicitly configured.
		// When no custom token is set, the default repo-scoped GITHUB_TOKEN from GitHub Actions
		// is already in the environment and overriding it with the same default is unnecessary.
		if isCustom {
			//nolint:gosec // G101: False positive - this is a GitHub Actions expression template, not a hardcoded credential
			steps = append(steps, fmt.Sprintf("          GITHUB_TOKEN: %s\n", gitToken))
			consolidatedSafeOutputsStepsLog.Printf("Adding GITHUB_TOKEN env var for cross-repo git CLI operations")
		}
	}

	// With section for github-token
	// Use the standard safe outputs token for all operations.
	// If project operations are configured, prefer the project token for the github-script client.
	// Rationale: update_project/create_project_status_update call the Projects v2 GraphQL API, which
	// cannot be accessed with the default GITHUB_TOKEN. GH_AW_PROJECT_GITHUB_TOKEN is the required
	// token for Projects v2 operations.
	steps = append(steps, "        with:\n")
	// Token precedence for the handler manager step:
	//   1. Project token (if project operations are configured) - already set above
	//   2. Safe-outputs level token (so.GitHubToken)
	//   3. Magic secret fallback via getEffectiveSafeOutputGitHubToken()
	//
	// Note: We do NOT fall back to per-output tokens (add-comment, create-issue, etc.)
	// because those are specific to their operations. The handler manager needs a
	// general-purpose token for the github-script client.
	configToken := ""
	if projectToken != "" {
		configToken = projectToken
	} else if data.SafeOutputs != nil && data.SafeOutputs.GitHubToken != "" {
		configToken = data.SafeOutputs.GitHubToken
	}
	c.addSafeOutputGitHubTokenForConfig(&steps, data, configToken)

	steps = append(steps, "          script: |\n")
	steps = append(steps, "            const { setupGlobals } = require('"+SetupActionDestination+"/setup_globals.cjs');\n")
	steps = append(steps, "            setupGlobals(core, github, context, exec, io, getOctokit);\n")
	steps = append(steps, "            const { main } = require('"+SetupActionDestination+"/safe_output_handler_manager.cjs');\n")
	steps = append(steps, "            await main();\n")

	return steps, nil
}

// replaceStepID replaces all occurrences of oldID with newID in a YAML step string.
// Used to generate per-handler token steps from the generic safe-outputs-app-token template.
func replaceStepID(step, oldID, newID string) string {
	return strings.ReplaceAll(step, oldID, newID)
}
