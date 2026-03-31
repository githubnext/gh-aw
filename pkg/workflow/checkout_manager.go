package workflow

import (
	"errors"
	"fmt"
	"strings"

	"github.com/github/gh-aw/pkg/logger"
)

var checkoutManagerLog = logger.New("workflow:checkout_manager")

// CheckoutConfig represents a single checkout configuration from workflow frontmatter.
// It controls how actions/checkout is invoked in the agent job.
//
// Supports all relevant options from actions/checkout:
//
//	checkout:
//	  fetch-depth: 0
//	  github-token: ${{ secrets.MY_TOKEN }}
//
// Or multiple checkouts:
//
//	checkout:
//	  - fetch-depth: 0
//	  - repository: owner/other-repo
//	    path: ./libs/other
//	    ref: main
//	    github-token: ${{ secrets.CROSS_REPO_PAT }}
//
// GitHub App authentication is also supported:
//
//	checkout:
//	  - repository: owner/other-repo
//	    path: ./libs/other
//	    app:
//	      app-id: ${{ vars.APP_ID }}
//	      private-key: ${{ secrets.APP_PRIVATE_KEY }}
type CheckoutConfig struct {
	// Repository to checkout in owner/repo format. Defaults to the current repository.
	Repository string `json:"repository,omitempty"`

	// Ref (branch, tag, or SHA) to checkout. Defaults to the ref that triggered the workflow.
	Ref string `json:"ref,omitempty"`

	// Path within GITHUB_WORKSPACE to place the checkout. Defaults to the workspace root.
	Path string `json:"path,omitempty"`

	// GitHubToken overrides the default GITHUB_TOKEN for authentication.
	// Use ${{ secrets.MY_TOKEN }} to reference a repository secret.
	// Maps to the "token" input of actions/checkout.
	// Mutually exclusive with GitHubApp.
	GitHubToken string `json:"github-token,omitempty"`

	// GitHubApp configures GitHub App-based authentication for this checkout.
	// When set, a token minting step is generated before checkout using
	// actions/create-github-app-token, and the minted token is passed
	// to actions/checkout as the "token" input.
	// Mutually exclusive with GitHubToken.
	GitHubApp *GitHubAppConfig `json:"github-app,omitempty"`

	// FetchDepth controls the number of commits to fetch.
	// 0 fetches all history (full clone). 1 is a shallow clone (default).
	FetchDepth *int `json:"fetch-depth,omitempty"`

	// SparseCheckout enables sparse-checkout mode. Provide newline-separated patterns
	// (e.g., ".github/\nsrc/"). When multiple configs target the same path, patterns
	// are merged into a single checkout.
	SparseCheckout string `json:"sparse-checkout,omitempty"`

	// Submodules controls submodule checkout behavior: "recursive", "true", or "false".
	Submodules string `json:"submodules,omitempty"`

	// LFS enables checkout of Git LFS objects.
	LFS bool `json:"lfs,omitempty"`

	// Current marks this checkout as the logical "current" repository for the workflow.
	// When set, the AI agent will treat this repository as its primary working target.
	// Only one checkout may have Current set to true.
	// This is useful for workflows that run from a central repo targeting a different repo.
	Current bool `json:"current,omitempty"`

	// Fetch specifies additional Git refs to fetch after checkout.
	// A git fetch step is emitted after the actions/checkout step.
	//
	// Supported values:
	//   - "*"            – fetch all remote branches
	//   - "refs/pulls/open/*" – GH-AW shorthand for all open pull-request refs
	//   - branch name    – e.g. "main" or "feature/my-branch"
	//   - glob pattern   – e.g. "feature/*"
	//
	// Example:
	//   fetch: ["*"]
	//   fetch: ["refs/pulls/open/*"]
	//   fetch: ["main", "feature/my-branch"]
	Fetch []string `json:"fetch,omitempty"`
}

// checkoutKey uniquely identifies a checkout target used for grouping/deduplication.
// Only repository and path are used as key fields — ref and token are settings
// that can be merged across configs targeting the same (repository, path).
type checkoutKey struct {
	repository string
	path       string
}

// resolvedCheckout is an internal merged checkout entry used by CheckoutManager.
type resolvedCheckout struct {
	key            checkoutKey
	ref            string           // last non-empty ref wins
	token          string           // last non-empty github-token wins
	githubApp      *GitHubAppConfig // GitHub App config (first non-nil wins)
	fetchDepth     *int             // nil means use default (1)
	sparsePatterns []string         // merged sparse-checkout patterns
	submodules     string
	lfs            bool
	current        bool     // true if this checkout is the logical current repository
	fetchRefs      []string // merged fetch ref patterns (see CheckoutConfig.Fetch)
}

// CheckoutManager collects checkout requests and merges them to minimize
// the number of actions/checkout steps emitted.
//
// Merging rules:
//   - Checkouts with the same (repository, ref, path, token) are merged into one.
//   - The deepest fetch-depth wins: 0 (full history) overrides any shallower value.
//   - Sparse-checkout patterns are unioned across merged configs.
//   - LFS and submodules are OR-ed (if any request enables them, the result enables them).
type CheckoutManager struct {
	// ordered preserves insertion order for deterministic output
	ordered []*resolvedCheckout
	// index maps checkoutKey to the position in ordered
	index map[checkoutKey]int
	// crossRepoTargetRepo holds the platform (host) repository to use when performing
	// .github/.agents sparse checkout steps for cross-repo workflow_call invocations.
	//
	// In the activation job this is set to "${{ steps.resolve-host-repo.outputs.target_repo }}".
	// In the agent and safe_outputs jobs it is set to "${{ needs.activation.outputs.target_repo }}".
	// An empty string means the checkout targets the current repository (github.repository).
	crossRepoTargetRepo string
	// crossRepoTargetRef holds the platform (host) ref (branch/tag/SHA) to use when
	// performing .github/.agents sparse checkout steps for cross-repo workflow_call
	// invocations pinned to a non-default branch.
	//
	// In the activation job this is set to "${{ steps.resolve-host-repo.outputs.target_ref }}".
	// In the agent and safe_outputs jobs it is set to "${{ needs.activation.outputs.target_ref }}".
	// An empty string means the checkout uses the repository's default branch.
	crossRepoTargetRef string
}

// NewCheckoutManager creates a new CheckoutManager pre-loaded with user-supplied
// CheckoutConfig entries from the frontmatter.
func NewCheckoutManager(userCheckouts []*CheckoutConfig) *CheckoutManager {
	checkoutManagerLog.Printf("Creating checkout manager with %d user checkout config(s)", len(userCheckouts))
	cm := &CheckoutManager{
		index: make(map[checkoutKey]int),
	}
	for _, cfg := range userCheckouts {
		cm.add(cfg)
	}
	return cm
}

// SetCrossRepoTargetRepo stores the platform (host) repository expression used for
// .github/.agents sparse checkout steps. Call this when the workflow has a workflow_call
// trigger and the checkout should target the platform repo rather than github.repository.
//
// In the activation job pass "${{ steps.resolve-host-repo.outputs.target_repo }}".
// In downstream jobs (agent, safe_outputs) pass "${{ needs.activation.outputs.target_repo }}".
func (cm *CheckoutManager) SetCrossRepoTargetRepo(repo string) {
	checkoutManagerLog.Printf("Setting cross-repo target repo: %q", repo)
	cm.crossRepoTargetRepo = repo
}

// GetCrossRepoTargetRepo returns the platform repo expression previously set by
// SetCrossRepoTargetRepo, or an empty string if no cross-repo target was set
// (same-repo invocation or inlined imports).
func (cm *CheckoutManager) GetCrossRepoTargetRepo() string {
	return cm.crossRepoTargetRepo
}

// SetCrossRepoTargetRef stores the platform (host) ref expression used for
// .github/.agents sparse checkout steps. Call this when the workflow has a workflow_call
// trigger and the checkout should target a specific branch rather than the default branch.
//
// In the activation job pass "${{ steps.resolve-host-repo.outputs.target_ref }}".
// In downstream jobs (agent, safe_outputs) pass "${{ needs.activation.outputs.target_ref }}".
func (cm *CheckoutManager) SetCrossRepoTargetRef(ref string) {
	checkoutManagerLog.Printf("Setting cross-repo target ref: %q", ref)
	cm.crossRepoTargetRef = ref
}

// GetCrossRepoTargetRef returns the platform ref expression previously set by
// SetCrossRepoTargetRef, or an empty string if no cross-repo ref was set.
func (cm *CheckoutManager) GetCrossRepoTargetRef() string {
	return cm.crossRepoTargetRef
}

// add processes a single CheckoutConfig and either creates a new entry or merges
// it into an existing entry with the same key.
func (cm *CheckoutManager) add(cfg *CheckoutConfig) {
	if cfg == nil {
		return
	}

	// Normalize path: "." and "" both refer to the workspace root.
	normalizedPath := cfg.Path
	if normalizedPath == "." {
		normalizedPath = ""
	}
	key := checkoutKey{
		repository: cfg.Repository,
		path:       normalizedPath,
	}

	if idx, exists := cm.index[key]; exists {
		// Merge into existing entry; first-seen wins for ref and token
		entry := cm.ordered[idx]
		entry.fetchDepth = deeperFetchDepth(entry.fetchDepth, cfg.FetchDepth)
		if cfg.Ref != "" && entry.ref == "" {
			entry.ref = cfg.Ref // first-seen ref wins
		}
		if cfg.GitHubToken != "" && entry.token == "" {
			entry.token = cfg.GitHubToken // first-seen github-token wins
		}
		if cfg.GitHubApp != nil && entry.githubApp == nil {
			entry.githubApp = cfg.GitHubApp // first-seen github-app wins
		}
		if cfg.SparseCheckout != "" {
			entry.sparsePatterns = mergeSparsePatterns(entry.sparsePatterns, cfg.SparseCheckout)
		}
		if cfg.LFS {
			entry.lfs = true
		}
		if cfg.Current {
			entry.current = true
		}
		if cfg.Submodules != "" && entry.submodules == "" {
			entry.submodules = cfg.Submodules
		}
		if len(cfg.Fetch) > 0 {
			entry.fetchRefs = mergeFetchRefs(entry.fetchRefs, cfg.Fetch)
		}
		checkoutManagerLog.Printf("Merged checkout for path=%q repository=%q", key.path, key.repository)
	} else {
		entry := &resolvedCheckout{
			key:        key,
			ref:        cfg.Ref,
			token:      cfg.GitHubToken,
			githubApp:  cfg.GitHubApp,
			fetchDepth: cfg.FetchDepth,
			submodules: cfg.Submodules,
			lfs:        cfg.LFS,
			current:    cfg.Current,
		}
		if cfg.SparseCheckout != "" {
			entry.sparsePatterns = mergeSparsePatterns(nil, cfg.SparseCheckout)
		}
		if len(cfg.Fetch) > 0 {
			entry.fetchRefs = mergeFetchRefs(nil, cfg.Fetch)
		}
		cm.index[key] = len(cm.ordered)
		cm.ordered = append(cm.ordered, entry)
		checkoutManagerLog.Printf("Added checkout for path=%q repository=%q", key.path, key.repository)
	}
}

// GetDefaultCheckoutOverride returns the resolved checkout for the default workspace
// (empty path, empty repository). Returns nil if the user did not configure one.
func (cm *CheckoutManager) GetDefaultCheckoutOverride() *resolvedCheckout {
	key := checkoutKey{}
	if idx, ok := cm.index[key]; ok {
		return cm.ordered[idx]
	}
	return nil
}

// HasAppAuth returns true if any checkout entry uses GitHub App authentication.
func (cm *CheckoutManager) HasAppAuth() bool {
	for _, entry := range cm.ordered {
		if entry.githubApp != nil {
			return true
		}
	}
	return false
}

// HasExternalRootCheckout returns true if any checkout entry targets an external
// repository (non-empty repository field) and writes to the workspace root (empty path).
// When such a checkout exists, the workspace root is replaced with the external
// repository content, which removes any locally-checked-out actions/setup directory.
// In dev mode, a "Restore actions folder" step must be added after such checkouts so
// the runner's post-step for the Setup Scripts action can find action.yml.
//
// Note: the "." path is normalized to "" in add(), so both "" and "." are covered
// by the entry.key.path == "" check.
func (cm *CheckoutManager) HasExternalRootCheckout() bool {
	for _, entry := range cm.ordered {
		if entry.key.repository != "" && entry.key.path == "" {
			return true
		}
	}
	return false
}

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

// generateDefaultCheckoutStep emits the default workspace checkout, applying any
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

// deeperFetchDepth returns the deeper of two fetch-depth values.
// 0 means full history and is always "deepest"; otherwise lower positive values
// are shallower. nil means "use default".
func deeperFetchDepth(a, b *int) *int {
	if a == nil && b == nil {
		return nil
	}
	if a == nil {
		return b
	}
	if b == nil {
		return a
	}
	// 0 = full history = deepest
	if *a == 0 || *b == 0 {
		zero := 0
		return &zero
	}
	// For positive depths, larger value = more history = deeper
	if *a > *b {
		return a
	}
	return b
}

// mergeSparsePatterns parses and unions sparse-checkout patterns.
// Patterns can be newline-separated.
func mergeSparsePatterns(existing []string, newPatterns string) []string {
	seen := make(map[string]bool, len(existing))
	result := make([]string, 0, len(existing))

	for _, p := range existing {
		p = strings.TrimSpace(p)
		if p != "" && !seen[p] {
			seen[p] = true
			result = append(result, p)
		}
	}

	for p := range strings.SplitSeq(newPatterns, "\n") {
		p = strings.TrimSpace(p)
		if p != "" && !seen[p] {
			seen[p] = true
			result = append(result, p)
		}
	}

	return result
}

// mergeFetchRefs unions two sets of fetch ref patterns preserving insertion order.
func mergeFetchRefs(existing []string, newRefs []string) []string {
	seen := make(map[string]bool, len(existing))
	result := make([]string, 0)
	for _, r := range existing {
		r = strings.TrimSpace(r)
		if r != "" && !seen[r] {
			seen[r] = true
			result = append(result, r)
		}
	}
	for _, r := range newRefs {
		r = strings.TrimSpace(r)
		if r != "" && !seen[r] {
			seen[r] = true
			result = append(result, r)
		}
	}
	return result
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

// ParseCheckoutConfigs converts a raw frontmatter value (single map or array of maps)
// into a slice of CheckoutConfig entries.
// Returns (nil, nil) if the value is nil; for non-nil values, invalid types or shapes
// result in a non-nil error.
func ParseCheckoutConfigs(raw any) ([]*CheckoutConfig, error) {
	if raw == nil {
		return nil, nil
	}
	checkoutManagerLog.Printf("Parsing checkout configuration: type=%T", raw)

	var configs []*CheckoutConfig

	// Try single object first
	if singleMap, ok := raw.(map[string]any); ok {
		cfg, err := checkoutConfigFromMap(singleMap)
		if err != nil {
			return nil, fmt.Errorf("invalid checkout configuration: %w", err)
		}
		configs = []*CheckoutConfig{cfg}
	} else if arr, ok := raw.([]any); ok {
		// Try array of objects
		configs = make([]*CheckoutConfig, 0, len(arr))
		for i, item := range arr {
			itemMap, ok := item.(map[string]any)
			if !ok {
				return nil, fmt.Errorf("checkout[%d]: expected object, got %T", i, item)
			}
			cfg, err := checkoutConfigFromMap(itemMap)
			if err != nil {
				return nil, fmt.Errorf("checkout[%d]: %w", i, err)
			}
			configs = append(configs, cfg)
		}
	} else {
		return nil, fmt.Errorf("checkout must be an object or an array of objects, got %T", raw)
	}

	// Validate that at most one logical checkout target has current: true.
	// Multiple current checkouts are not allowed since only one repo/path pair can be
	// the primary target for the agent at a time. Multiple configs that merge into the
	// same (repository, path) pair are treated as a single logical checkout.
	currentTargets := make(map[string]struct{})
	for _, cfg := range configs {
		if !cfg.Current {
			continue
		}

		repo := strings.TrimSpace(cfg.Repository)
		path := strings.TrimSpace(cfg.Path)
		key := repo + "\x00" + path

		currentTargets[key] = struct{}{}
	}
	if len(currentTargets) > 1 {
		return nil, fmt.Errorf("only one checkout target may have current: true, found %d", len(currentTargets))
	}

	return configs, nil
}

// checkoutConfigFromMap converts a raw map to a CheckoutConfig.
func checkoutConfigFromMap(m map[string]any) (*CheckoutConfig, error) {
	cfg := &CheckoutConfig{}

	if v, ok := m["repository"]; ok {
		s, ok := v.(string)
		if !ok {
			return nil, errors.New("checkout.repository must be a string")
		}
		cfg.Repository = s
	}

	if v, ok := m["ref"]; ok {
		s, ok := v.(string)
		if !ok {
			return nil, errors.New("checkout.ref must be a string")
		}
		cfg.Ref = s
	}

	if v, ok := m["path"]; ok {
		s, ok := v.(string)
		if !ok {
			return nil, errors.New("checkout.path must be a string")
		}
		// Normalize "." to empty string: both mean the workspace root and
		// are treated identically by the checkout step generator.
		if s == "." {
			s = ""
		}
		cfg.Path = s
	}

	// Support both "github-token" (preferred) and "token" (backward compat)
	if v, ok := m["github-token"]; ok {
		s, ok := v.(string)
		if !ok {
			return nil, errors.New("checkout.github-token must be a string")
		}
		cfg.GitHubToken = s
	} else if v, ok := m["token"]; ok {
		// Backward compatibility: "token" is accepted but "github-token" is preferred
		s, ok := v.(string)
		if !ok {
			return nil, errors.New("checkout.token must be a string")
		}
		cfg.GitHubToken = s
	}

	// Parse app configuration for GitHub App-based authentication
	if v, ok := m["github-app"]; ok {
		appMap, ok := v.(map[string]any)
		if !ok {
			return nil, errors.New("checkout.github-app must be an object")
		}
		cfg.GitHubApp = parseAppConfig(appMap)
		if cfg.GitHubApp.AppID == "" || cfg.GitHubApp.PrivateKey == "" {
			return nil, errors.New("checkout.github-app requires both app-id and private-key")
		}
	}

	// Validate mutual exclusivity of github-token and github-app
	if cfg.GitHubToken != "" && cfg.GitHubApp != nil {
		return nil, errors.New("checkout: github-token and github-app are mutually exclusive; use one or the other")
	}

	if v, ok := m["fetch-depth"]; ok {
		switch n := v.(type) {
		case int:
			depth := n
			cfg.FetchDepth = &depth
		case int64:
			depth := int(n)
			cfg.FetchDepth = &depth
		case uint64:
			depth := int(n)
			cfg.FetchDepth = &depth
		case float64:
			if n != float64(int64(n)) {
				return nil, errors.New("checkout.fetch-depth must be an integer")
			}
			depth := int(n)
			cfg.FetchDepth = &depth
		default:
			return nil, errors.New("checkout.fetch-depth must be an integer")
		}
	}

	if v, ok := m["sparse-checkout"]; ok {
		s, ok := v.(string)
		if !ok {
			return nil, errors.New("checkout.sparse-checkout must be a string")
		}
		cfg.SparseCheckout = s
	}

	if v, ok := m["submodules"]; ok {
		switch sv := v.(type) {
		case string:
			cfg.Submodules = sv
		case bool:
			if sv {
				cfg.Submodules = "true"
			} else {
				cfg.Submodules = "false"
			}
		default:
			return nil, errors.New("checkout.submodules must be a string or boolean")
		}
	}

	if v, ok := m["lfs"]; ok {
		b, ok := v.(bool)
		if !ok {
			return nil, errors.New("checkout.lfs must be a boolean")
		}
		cfg.LFS = b
	}

	if v, ok := m["current"]; ok {
		b, ok := v.(bool)
		if !ok {
			return nil, errors.New("checkout.current must be a boolean")
		}
		cfg.Current = b
	}

	if v, ok := m["fetch"]; ok {
		switch fv := v.(type) {
		case string:
			// Single string shorthand: treat as a one-element list
			if strings.TrimSpace(fv) == "" {
				return nil, errors.New("checkout.fetch string value must not be empty")
			}
			cfg.Fetch = []string{fv}
		case []any:
			refs := make([]string, 0, len(fv))
			for i, item := range fv {
				s, ok := item.(string)
				if !ok {
					return nil, fmt.Errorf("checkout.fetch[%d] must be a string, got %T", i, item)
				}
				if strings.TrimSpace(s) == "" {
					return nil, fmt.Errorf("checkout.fetch[%d] must not be empty", i)
				}
				refs = append(refs, s)
			}
			cfg.Fetch = refs
		default:
			return nil, errors.New("checkout.fetch must be a string or an array of strings")
		}
	}

	return cfg, nil
}

// buildCheckoutsPromptContent returns a markdown bullet list describing all user-configured
// checkouts for inclusion in the GitHub context prompt.
// Returns an empty string when no checkouts are configured.
//
// Each checkout is shown with its full absolute path relative to $GITHUB_WORKSPACE.
// The root checkout (path == "") is annotated as "(cwd)" since that is the working
// directory of the agent process. The generated content may include
// "${{ github.repository }}" for any checkout that does not have an explicit repository
// configured; callers must ensure these expressions are processed by an ExpressionExtractor
// so the placeholder substitution step can resolve them at runtime.
func buildCheckoutsPromptContent(checkouts []*CheckoutConfig) string {
	if len(checkouts) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("- **checkouts**: The following repositories have been checked out and are available in the workspace:\n")

	for _, cfg := range checkouts {
		if cfg == nil {
			continue
		}

		// Build the full absolute path using $GITHUB_WORKSPACE as root.
		// Normalize the path: strip "./" prefix; bare "." and "" both mean root.
		relPath := strings.TrimPrefix(cfg.Path, "./")
		if relPath == "." {
			relPath = ""
		}
		isRoot := relPath == ""
		absPath := "$GITHUB_WORKSPACE"
		if !isRoot {
			absPath += "/" + relPath
		}

		// Determine repo: use configured value or fall back to the triggering repository expression
		repo := cfg.Repository
		if repo == "" {
			repo = "${{ github.repository }}"
		}

		line := fmt.Sprintf("  - `%s` → `%s`", absPath, repo)
		if isRoot {
			line += " (cwd)"
		}
		if cfg.Current {
			line += " (**current** - this is the repository you are working on; use this as the target for all GitHub operations unless otherwise specified)"
		}

		// Annotate fetch-depth so the agent knows how much history is available
		if cfg.FetchDepth != nil && *cfg.FetchDepth == 0 {
			line += " [full history, all branches available as remote-tracking refs]"
		} else if cfg.FetchDepth != nil {
			line += fmt.Sprintf(" [shallow clone, fetch-depth=%d]", *cfg.FetchDepth)
		} else {
			line += " [shallow clone, fetch-depth=1 (default)]"
		}

		// Annotate additionally fetched refs
		if len(cfg.Fetch) > 0 {
			line += fmt.Sprintf(" [additional refs fetched: %s]", strings.Join(cfg.Fetch, ", "))
		}

		sb.WriteString(line + "\n")
	}

	// General guidance about unavailable branches
	sb.WriteString("  - **Note**: If a branch you need is not in the list above and is not listed as an additional fetched ref, " +
		"it has NOT been checked out. For private repositories you cannot fetch it without proper authentication. " +
		"If the branch is required and not available, exit with an error and ask the user to add it to the " +
		"`fetch:` option of the `checkout:` configuration (e.g., `fetch: [\"refs/pulls/open/*\"]` for all open PR refs, " +
		"or `fetch: [\"main\", \"feature/my-branch\"]` for specific branches).\n")

	return sb.String()
}
