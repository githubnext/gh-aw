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
//	  - path: .
//	    fetch-depth: 0
//	  - repository: owner/other-repo
//	    path: ./libs/other
//	    ref: main
type CheckoutConfig struct {
	// Repository to checkout in owner/repo format. Defaults to the current repository.
	Repository string `json:"repository,omitempty"`

	// Ref (branch, tag, or SHA) to checkout. Defaults to the ref that triggered the workflow.
	Ref string `json:"ref,omitempty"`

	// Path within GITHUB_WORKSPACE to place the checkout. Defaults to the workspace root.
	Path string `json:"path,omitempty"`

	// GitHubToken overrides the default GITHUB_TOKEN for authentication.
	// Use ${{ secrets.MY_TOKEN }} to reference a repository secret.
	GitHubToken string `json:"github-token,omitempty"`

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
	ref            string   // last non-empty ref wins
	token          string   // last non-empty token wins
	fetchDepth     *int     // nil means use default (1)
	sparsePatterns []string // merged sparse-checkout patterns
	submodules     string
	lfs            bool
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
}

// NewCheckoutManager creates a new CheckoutManager pre-loaded with user-supplied
// CheckoutConfig entries from the frontmatter.
func NewCheckoutManager(userCheckouts []*CheckoutConfig) *CheckoutManager {
	cm := &CheckoutManager{
		index: make(map[checkoutKey]int),
	}
	for _, cfg := range userCheckouts {
		cm.add(cfg)
	}
	return cm
}

// add processes a single CheckoutConfig and either creates a new entry or merges
// it into an existing entry with the same key.
func (cm *CheckoutManager) add(cfg *CheckoutConfig) {
	if cfg == nil {
		return
	}

	key := checkoutKey{
		repository: cfg.Repository,
		path:       cfg.Path,
	}

	if idx, exists := cm.index[key]; exists {
		// Merge into existing entry; first-seen wins for ref and token
		entry := cm.ordered[idx]
		entry.fetchDepth = deeperFetchDepth(entry.fetchDepth, cfg.FetchDepth)
		if cfg.Ref != "" && entry.ref == "" {
			entry.ref = cfg.Ref // first-seen ref wins
		}
		if cfg.GitHubToken != "" && entry.token == "" {
			entry.token = cfg.GitHubToken // first-seen token wins
		}
		if cfg.SparseCheckout != "" {
			entry.sparsePatterns = mergeSparsePatterns(entry.sparsePatterns, cfg.SparseCheckout)
		}
		if cfg.LFS {
			entry.lfs = true
		}
		if cfg.Submodules != "" && entry.submodules == "" {
			entry.submodules = cfg.Submodules
		}
		checkoutManagerLog.Printf("Merged checkout for path=%q repository=%q", key.path, key.repository)
	} else {
		entry := &resolvedCheckout{
			key:        key,
			ref:        cfg.Ref,
			token:      cfg.GitHubToken,
			fetchDepth: cfg.FetchDepth,
			submodules: cfg.Submodules,
			lfs:        cfg.LFS,
		}
		if cfg.SparseCheckout != "" {
			entry.sparsePatterns = mergeSparsePatterns(nil, cfg.SparseCheckout)
		}
		cm.index[key] = len(cm.ordered)
		cm.ordered = append(cm.ordered, entry)
		checkoutManagerLog.Printf("Added checkout for path=%q repository=%q", key.path, key.repository)
	}
}

// HasUserCheckouts returns true if any user-supplied checkouts were registered.
func (cm *CheckoutManager) HasUserCheckouts() bool {
	return len(cm.ordered) > 0
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

// GenerateAdditionalCheckoutSteps generates YAML step lines for all non-default
// (additional) checkouts — those that target a specific path other than the root.
// The caller is responsible for emitting the default workspace checkout separately.
func (cm *CheckoutManager) GenerateAdditionalCheckoutSteps(getActionPin func(string) string) []string {
	var lines []string
	for _, entry := range cm.ordered {
		// Skip the default checkout (handled separately)
		if entry.key.path == "" && entry.key.repository == "" {
			continue
		}
		lines = append(lines, generateCheckoutStepLines(entry, getActionPin)...)
	}
	return lines
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
		if override.token != "" {
			fmt.Fprintf(&sb, "          github-token: %s\n", override.token)
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

	return []string{sb.String()}
}

// generateCheckoutStepLines generates YAML step lines for a single non-default checkout.
func generateCheckoutStepLines(entry *resolvedCheckout, getActionPin func(string) string) []string {
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
	if entry.token != "" {
		fmt.Fprintf(&sb, "          github-token: %s\n", entry.token)
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

	return []string{sb.String()}
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

// ParseCheckoutConfigs converts a raw frontmatter value (single map or array of maps)
// into a slice of CheckoutConfig entries.
// Returns (nil, nil) if the value is nil; for non-nil values, invalid types or shapes
// result in a non-nil error.
func ParseCheckoutConfigs(raw any) ([]*CheckoutConfig, error) {
	if raw == nil {
		return nil, nil
	}

	// Try single object first
	if singleMap, ok := raw.(map[string]any); ok {
		cfg, err := checkoutConfigFromMap(singleMap)
		if err != nil {
			return nil, fmt.Errorf("invalid checkout configuration: %w", err)
		}
		return []*CheckoutConfig{cfg}, nil
	}

	// Try array of objects
	if arr, ok := raw.([]any); ok {
		configs := make([]*CheckoutConfig, 0, len(arr))
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
		return configs, nil
	}

	return nil, fmt.Errorf("checkout must be an object or an array of objects, got %T", raw)
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
		cfg.Path = s
	}

	if v, ok := m["github-token"]; ok {
		s, ok := v.(string)
		if !ok {
			return nil, errors.New("checkout.github-token must be a string")
		}
		cfg.GitHubToken = s
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

	return cfg, nil
}
