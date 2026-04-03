package workflow

import (
	"fmt"
	"strings"
)

// GenerateCheckoutAppTokenSteps generates GitHub App token minting steps for all
// checkout entries that use app authentication. Each app-authenticated checkout
// gets its own minting step with a unique step ID, so the minted token can be
// referenced in the corresponding checkout step.
//
// The step ID for each checkout is "checkout-app-token-{index}" where index is
// the position in the ordered checkout list.
func (cm *CheckoutManager) GenerateCheckoutAppTokenSteps(c *Compiler, permissions *Permissions) []string {
	var steps []string
	for i, entry := range cm.ordered {
		if entry.githubApp == nil {
			continue
		}
		checkoutManagerLog.Printf("Generating app token minting step for checkout index=%d repo=%q", i, entry.key.repository)
		// Pass empty fallback so the app token defaults to github.event.repository.name.
		// Checkout-specific cross-repo scoping is handled via the explicit repository field.
		appSteps := c.buildGitHubAppTokenMintStep(entry.githubApp, permissions, "")
		stepID := fmt.Sprintf("checkout-app-token-%d", i)
		for _, step := range appSteps {
			modified := strings.ReplaceAll(step, "id: safe-outputs-app-token", "id: "+stepID)
			steps = append(steps, modified)
		}
	}
	return steps
}

// GenerateCheckoutAppTokenInvalidationSteps generates token invalidation steps
// for all checkout entries that use app authentication.
func (cm *CheckoutManager) GenerateCheckoutAppTokenInvalidationSteps(c *Compiler) []string {
	var steps []string
	for i, entry := range cm.ordered {
		if entry.githubApp == nil {
			continue
		}
		checkoutManagerLog.Printf("Generating app token invalidation step for checkout index=%d", i)
		rawSteps := c.buildGitHubAppTokenInvalidationStep()
		stepID := fmt.Sprintf("checkout-app-token-%d", i)
		for _, step := range rawSteps {
			modified := strings.ReplaceAll(step, "steps.safe-outputs-app-token.outputs.token", "steps."+stepID+".outputs.token")
			// Update step name to indicate it's for checkout
			modified = strings.ReplaceAll(modified, "Invalidate GitHub App token", fmt.Sprintf("Invalidate checkout app token (%d)", i))
			steps = append(steps, modified)
		}
	}
	return steps
}

// GenerateAdditionalCheckoutSteps generates YAML step lines for all non-default
// (additional) checkouts — those that target a specific path other than the root.
// The caller is responsible for emitting the default workspace checkout separately.
func (cm *CheckoutManager) GenerateAdditionalCheckoutSteps(getActionPin func(string) string) []string {
	checkoutManagerLog.Printf("Generating additional checkout steps from %d configured entries", len(cm.ordered))
	var lines []string
	for i, entry := range cm.ordered {
		// Skip the default checkout (handled separately)
		if entry.key.path == "" && entry.key.repository == "" {
			continue
		}
		lines = append(lines, generateCheckoutStepLines(entry, i, getActionPin)...)
	}
	checkoutManagerLog.Printf("Generated %d additional checkout step(s)", len(lines))
	return lines
}

// GenerateGitHubFolderCheckoutStep generates YAML step lines for a sparse checkout of
// the .github and .agents folders. This is used in the activation job to access workflow
// configuration and runtime imports.
//
// Parameters:
//   - repository: the repository to checkout. May be a literal "owner/repo" value or a
//     GitHub Actions expression such as "${{ steps.resolve-host-repo.outputs.target_repo }}".
//     Pass an empty string to omit the repository field and check out the current repository.
//   - ref: the branch, tag, or SHA to checkout. May be a literal value or a GitHub Actions
//     expression such as "${{ steps.resolve-host-repo.outputs.target_ref }}".
//     Pass an empty string to omit the ref field and use the repository's default branch.
//   - getActionPin: resolves an action reference to a pinned SHA form.
//   - extraPaths: additional paths to include in the sparse-checkout beyond .github and .agents.
//
// Returns a slice of YAML lines (each ending with \n).
func (cm *CheckoutManager) GenerateGitHubFolderCheckoutStep(repository, ref string, getActionPin func(string) string, extraPaths ...string) []string {
	checkoutManagerLog.Printf("Generating .github/.agents folder checkout: repository=%q ref=%q", repository, ref)
	var sb strings.Builder

	sb.WriteString("      - name: Checkout .github and .agents folders\n")
	fmt.Fprintf(&sb, "        uses: %s\n", getActionPin("actions/checkout"))
	sb.WriteString("        with:\n")
	sb.WriteString("          persist-credentials: false\n")
	if repository != "" {
		fmt.Fprintf(&sb, "          repository: %s\n", repository)
	}
	if ref != "" {
		fmt.Fprintf(&sb, "          ref: %s\n", ref)
	}
	sb.WriteString("          sparse-checkout: |\n")
	sb.WriteString("            .github\n")
	sb.WriteString("            .agents\n")
	for _, p := range extraPaths {
		fmt.Fprintf(&sb, "            %s\n", p)
	}
	sb.WriteString("          sparse-checkout-cone-mode: true\n")
	sb.WriteString("          fetch-depth: 1\n")

	return []string{sb.String()}
}

// GenerateDefaultCheckoutStep emits the default workspace checkout, applying any
// user-supplied overrides (token, fetch-depth, ref, etc.) on top of the required
// security defaults (persist-credentials: false).
//
// Parameters:
//   - trialMode: if true, optionally sets repository and token for trial execution
//   - trialLogicalRepoSlug: the repository to checkout in trial mode
//   - getActionPin: resolves an action reference to a pinned SHA form
//
// Returns a slice of YAML lines (each ending with \n).
func (cm *CheckoutManager) GenerateDefaultCheckoutStep(
	trialMode bool,
	trialLogicalRepoSlug string,
	getActionPin func(string) string,
) []string {
	override := cm.GetDefaultCheckoutOverride()
	checkoutManagerLog.Printf("Generating default checkout step: trialMode=%t, hasOverride=%t", trialMode, override != nil)

	var sb strings.Builder
	sb.WriteString("      - name: Checkout repository\n")
	fmt.Fprintf(&sb, "        uses: %s\n", getActionPin("actions/checkout"))
	sb.WriteString("        with:\n")

	// Security: always disable credential persistence so the agent cannot
	// exfiltrate credentials from disk.
	sb.WriteString("          persist-credentials: false\n")

	// Apply trial mode overrides
	if trialMode {
		if trialLogicalRepoSlug != "" {
			fmt.Fprintf(&sb, "          repository: %s\n", trialLogicalRepoSlug)
		}
		effectiveToken := getEffectiveGitHubToken("")
		fmt.Fprintf(&sb, "          token: %s\n", effectiveToken)
	}

	// Apply user overrides (only when NOT in trial mode to avoid conflicts)
	if !trialMode && override != nil {
		if override.key.repository != "" {
			fmt.Fprintf(&sb, "          repository: %s\n", override.key.repository)
		}
		if override.ref != "" {
			fmt.Fprintf(&sb, "          ref: %s\n", override.ref)
		}
		// Determine effective token: github-app-minted token takes precedence
		effectiveOverrideToken := override.token
		if override.githubApp != nil {
			// The default checkout is always at index 0 in the ordered list
			//nolint:gosec // G101: False positive - this is a GitHub Actions expression template placeholder, not a hardcoded credential
			effectiveOverrideToken = "${{ steps.checkout-app-token-0.outputs.token }}"
		}
		if effectiveOverrideToken != "" {
			fmt.Fprintf(&sb, "          token: %s\n", effectiveOverrideToken)
		}
		if override.fetchDepth != nil {
			fmt.Fprintf(&sb, "          fetch-depth: %d\n", *override.fetchDepth)
		}
		if len(override.sparsePatterns) > 0 {
			sb.WriteString("          sparse-checkout: |\n")
			for _, pattern := range override.sparsePatterns {
				fmt.Fprintf(&sb, "            %s\n", strings.TrimSpace(pattern))
			}
		}
		if override.submodules != "" {
			fmt.Fprintf(&sb, "          submodules: %s\n", override.submodules)
		}
		if override.lfs {
			sb.WriteString("          lfs: true\n")
		}
	}

	steps := []string{sb.String()}

	// Emit a git fetch step if the user requested additional refs.
	// In trial mode the fetch step is still emitted so the behaviour
	// mirrors production as closely as possible.
	if override != nil && len(override.fetchRefs) > 0 {
		// Default checkout is at index 0 in the ordered list
		defaultIdx := 0
		if idx, ok := cm.index[checkoutKey{}]; ok {
			defaultIdx = idx
		}
		if fetchStep := generateFetchStepLines(override, defaultIdx); fetchStep != "" {
			steps = append(steps, fetchStep)
		}
	}

	return steps
}

// generateCheckoutStepLines generates YAML step lines for a single non-default checkout.
// The index parameter identifies the checkout's position in the ordered list, used to
// reference the correct app token minting step when app authentication is configured.
func generateCheckoutStepLines(entry *resolvedCheckout, index int, getActionPin func(string) string) []string {
	name := "Checkout " + checkoutStepName(entry.key)
	var sb strings.Builder
	fmt.Fprintf(&sb, "      - name: %s\n", name)
	fmt.Fprintf(&sb, "        uses: %s\n", getActionPin("actions/checkout"))
	sb.WriteString("        with:\n")

	// Security: always disable credential persistence
	sb.WriteString("          persist-credentials: false\n")

	if entry.key.repository != "" {
		fmt.Fprintf(&sb, "          repository: %s\n", entry.key.repository)
	}
	if entry.ref != "" {
		fmt.Fprintf(&sb, "          ref: %s\n", entry.ref)
	}
	if entry.key.path != "" {
		fmt.Fprintf(&sb, "          path: %s\n", entry.key.path)
	}
	// Determine effective token: github-app-minted token takes precedence
	effectiveToken := entry.token
	if entry.githubApp != nil {
		//nolint:gosec // G101: False positive - this is a GitHub Actions expression template placeholder, not a hardcoded credential
		effectiveToken = fmt.Sprintf("${{ steps.checkout-app-token-%d.outputs.token }}", index)
	}
	if effectiveToken != "" {
		fmt.Fprintf(&sb, "          token: %s\n", effectiveToken)
	}
	if entry.fetchDepth != nil {
		fmt.Fprintf(&sb, "          fetch-depth: %d\n", *entry.fetchDepth)
	}
	if len(entry.sparsePatterns) > 0 {
		sb.WriteString("          sparse-checkout: |\n")
		for _, pattern := range entry.sparsePatterns {
			fmt.Fprintf(&sb, "            %s\n", strings.TrimSpace(pattern))
		}
	}
	if entry.submodules != "" {
		fmt.Fprintf(&sb, "          submodules: %s\n", entry.submodules)
	}
	if entry.lfs {
		sb.WriteString("          lfs: true\n")
	}

	steps := []string{sb.String()}
	if fetchStep := generateFetchStepLines(entry, index); fetchStep != "" {
		steps = append(steps, fetchStep)
	}
	return steps
}

// checkoutStepName returns a human-readable description for a checkout step.
func checkoutStepName(key checkoutKey) string {
	if key.repository != "" && key.path != "" {
		return fmt.Sprintf("%s into %s", key.repository, key.path)
	}
	if key.repository != "" {
		return key.repository
	}
	if key.path != "" {
		return key.path
	}
	return "repository"
}

// fetchRefToRefspec converts a user-facing fetch pattern to a git refspec.
//
// Special values:
//   - "*"            → "+refs/heads/*:refs/remotes/origin/*"
//   - "refs/pulls/open/*" → "+refs/pull/*/head:refs/remotes/origin/pull/*/head"
//
// All other values are treated as branch names or glob patterns and mapped to
// the canonical remote-tracking refspec form.
func fetchRefToRefspec(pattern string) string {
	switch pattern {
	case "*":
		return "+refs/heads/*:refs/remotes/origin/*"
	case "refs/pulls/open/*":
		return "+refs/pull/*/head:refs/remotes/origin/pull/*/head"
	default:
		// Treat as branch name or glob: map to remote tracking ref
		return fmt.Sprintf("+refs/heads/%s:refs/remotes/origin/%s", pattern, pattern)
	}
}

// generateFetchStepLines generates a "Fetch additional refs" YAML step for the given checkout
// entry when it has fetch refs configured. Returns an empty string when there are no fetch refs.
// The index parameter identifies the checkout's position in the ordered list, used to
// reference the correct app token minting step when app authentication is configured.
//
// Authentication: the token is passed as the GH_AW_FETCH_TOKEN environment variable and
// injected via git's http.extraheader config option at the command level (-c flag), which
// avoids writing credentials to disk and is consistent with the persist-credentials: false
// policy. Note that http.extraheader values are visible in the git process's environment
// (like all GitHub Actions environment variables containing secrets); GitHub Actions
// automatically masks secret values in logs.
func generateFetchStepLines(entry *resolvedCheckout, index int) string {
	if len(entry.fetchRefs) == 0 {
		return ""
	}

	// Build step name
	name := "Fetch additional refs"
	if entry.key.repository != "" {
		name = "Fetch additional refs for " + entry.key.repository
	}

	// Determine authentication token
	token := entry.token
	if entry.githubApp != nil {
		//nolint:gosec // G101: False positive - this is a GitHub Actions expression template placeholder, not a hardcoded credential
		token = fmt.Sprintf("${{ steps.checkout-app-token-%d.outputs.token }}", index)
	}
	if token == "" {
		token = getEffectiveGitHubToken("")
	}

	// Build refspecs
	refspecs := make([]string, 0, len(entry.fetchRefs))
	for _, ref := range entry.fetchRefs {
		refspecs = append(refspecs, fmt.Sprintf("'%s'", fetchRefToRefspec(ref)))
	}

	// Build the git command, navigating to the checkout directory when needed
	gitPrefix := "git"
	if entry.key.path != "" {
		gitPrefix = fmt.Sprintf(`git -C "${{ github.workspace }}/%s"`, entry.key.path)
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "      - name: %s\n", name)
	sb.WriteString("        env:\n")
	fmt.Fprintf(&sb, "          GH_AW_FETCH_TOKEN: %s\n", token)
	sb.WriteString("        run: |\n")
	sb.WriteString("          header=$(printf \"x-access-token:%s\" \"${GH_AW_FETCH_TOKEN}\" | base64 -w 0)\n")
	fmt.Fprintf(&sb, `          %s -c "http.extraheader=Authorization: Basic ${header}" fetch origin %s`+"\n",
		gitPrefix, strings.Join(refspecs, " "))
	return sb.String()
}
