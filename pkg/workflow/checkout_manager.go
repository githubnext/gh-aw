package workflow

import (
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

// CheckoutAppTokenOutputs returns a map of activation job output names to their
// step token expressions, for all checkouts using app authentication.
// Output names follow the pattern "checkout_app_token_{index}".
// Step ID expressions follow the pattern "steps.checkout-app-token-{index}.outputs.token".
// These outputs are set by the activation job so the agent job can reference them as
// needs.activation.outputs.checkout_app_token_{index}.
func (cm *CheckoutManager) CheckoutAppTokenOutputs() map[string]string {
	outputs := make(map[string]string)
	for i, entry := range cm.ordered {
		if entry.githubApp == nil {
			continue
		}
		outputName := fmt.Sprintf("checkout_app_token_%d", i)
		stepID := fmt.Sprintf("checkout-app-token-%d", i)
		outputs[outputName] = fmt.Sprintf("${{ steps.%s.outputs.token }}", stepID)
	}
	return outputs
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
