package workflow

import (
	_ "embed"
	"errors"
	"fmt"
	"strings"
)

//go:embed prompts/checkouts_no_credentials_warning.md
var checkoutsNoCredentialsWarning string

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
	// same (repository, path, wiki) tuple are treated as a single logical checkout.
	currentTargets := make(map[string]struct{})
	for _, cfg := range configs {
		if !cfg.Current {
			continue
		}

		repo := strings.TrimSpace(cfg.Repository)
		path := strings.TrimSpace(cfg.Path)
		wiki := "false"
		if cfg.Wiki {
			wiki = "true"
		}
		key := repo + "\x00" + path + "\x00" + wiki

		currentTargets[key] = struct{}{}
	}
	if len(currentTargets) > 1 {
		checkoutManagerLog.Printf("Rejecting checkout config: %d distinct current targets, only one allowed", len(currentTargets))
		return nil, fmt.Errorf("only one checkout target may have current: true, found %d", len(currentTargets))
	}

	checkoutManagerLog.Printf("Parsed %d checkout configuration(s), current-targets=%d", len(configs), len(currentTargets))
	return configs, nil
}

// checkoutConfigFromMap converts a raw map to a CheckoutConfig.
func checkoutConfigFromMap(m map[string]any) (*CheckoutConfig, error) {
	cfg := &CheckoutConfig{}

	if err := parseCheckoutIdentityFields(cfg, m); err != nil {
		return nil, err
	}
	if err := parseCheckoutAuthFields(cfg, m); err != nil {
		return nil, err
	}
	if cfg.GitHubToken != "" && cfg.GitHubApp != nil {
		checkoutManagerLog.Print("Rejecting checkout config: github-token and github-app are mutually exclusive")
		return nil, errors.New("checkout: github-token and github-app are mutually exclusive; use one or the other")
	}
	checkoutManagerLog.Printf("Parsed checkout config: repo=%q, ref=%q, path=%q, current=%v, hasToken=%v, hasApp=%v",
		cfg.Repository, cfg.Ref, cfg.Path, cfg.Current, cfg.GitHubToken != "", cfg.GitHubApp != nil)
	if err := parseCheckoutGitFields(cfg, m); err != nil {
		return nil, err
	}
	if err := parseCheckoutBooleanFields(cfg, m); err != nil {
		return nil, err
	}

	return cfg, nil
}

func parseCheckoutIdentityFields(cfg *CheckoutConfig, m map[string]any) error {
	if v, ok := m["repository"]; ok {
		s, ok := v.(string)
		if !ok {
			return errors.New("checkout.repository must be a string")
		}
		cfg.Repository = s
	}
	if v, ok := m["ref"]; ok {
		s, ok := v.(string)
		if !ok {
			return errors.New("checkout.ref must be a string")
		}
		cfg.Ref = s
	}
	if v, ok := m["path"]; ok {
		s, ok := v.(string)
		if !ok {
			return errors.New("checkout.path must be a string")
		}
		cfg.PathExplicit = true
		if s == "." {
			s = ""
		}
		cfg.Path = s
	}
	return nil
}

func parseCheckoutAuthFields(cfg *CheckoutConfig, m map[string]any) error {
	if v, ok := m["github-token"]; ok {
		s, ok := v.(string)
		if !ok {
			return errors.New("checkout.github-token must be a string")
		}
		cfg.GitHubToken = s
	} else if v, ok := m["token"]; ok {
		s, ok := v.(string)
		if !ok {
			return errors.New("checkout.token must be a string")
		}
		cfg.GitHubToken = s
	}
	if v, ok := m["github-app"]; ok {
		appConfig, err := parseCheckoutAppConfig("github-app", v)
		if err != nil {
			return err
		}
		cfg.GitHubApp = appConfig
	}
	if v, ok := m["safe-outputs-github-app"]; ok {
		appConfig, err := parseCheckoutAppConfig("safe-outputs-github-app", v)
		if err != nil {
			return err
		}
		cfg.SafeOutputGitHubApp = appConfig
	}
	if _, ok := m["safe-output-github-app"]; ok {
		return errors.New("checkout.safe-output-github-app is not supported; use checkout.safe-outputs-github-app")
	}
	return nil
}

func parseCheckoutAppConfig(fieldName string, value any) (*GitHubAppConfig, error) {
	appMap, ok := value.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("checkout.%s must be an object", fieldName)
	}
	appConfig := parseAppConfig(appMap)
	if appConfig.AppID == "" || appConfig.PrivateKey == "" {
		return nil, fmt.Errorf("checkout.%s requires both client-id (or app-id) and private-key", fieldName)
	}
	return appConfig, nil
}

func parseCheckoutGitFields(cfg *CheckoutConfig, m map[string]any) error {
	if v, ok := m["fetch-depth"]; ok {
		depth, err := parseCheckoutFetchDepth(v)
		if err != nil {
			return err
		}
		cfg.FetchDepth = depth
	}
	if v, ok := m["sparse-checkout"]; ok {
		s, ok := v.(string)
		if !ok {
			return errors.New("checkout.sparse-checkout must be a string")
		}
		cfg.SparseCheckout = s
	}
	if v, ok := m["submodules"]; ok {
		submodules, err := parseCheckoutSubmodules(v)
		if err != nil {
			return err
		}
		cfg.Submodules = submodules
	}
	if v, ok := m["fetch"]; ok {
		refs, err := parseCheckoutFetchRefs(v)
		if err != nil {
			return err
		}
		cfg.Fetch = refs
	}
	return nil
}

func parseCheckoutFetchDepth(v any) (*int, error) {
	var depth int
	switch n := v.(type) {
	case int:
		depth = n
	case int64:
		depth = int(n)
	case uint64:
		depth = int(n)
	case float64:
		if n != float64(int64(n)) {
			return nil, errors.New("checkout.fetch-depth must be an integer")
		}
		depth = int(n)
	default:
		return nil, errors.New("checkout.fetch-depth must be an integer")
	}
	if depth < 0 {
		return nil, errors.New("checkout.fetch-depth must be >= 0")
	}
	return &depth, nil
}

func parseCheckoutSubmodules(v any) (string, error) {
	switch sv := v.(type) {
	case string:
		return sv, nil
	case bool:
		if sv {
			return "true", nil
		}
		return "false", nil
	default:
		return "", errors.New("checkout.submodules must be a string or boolean")
	}
}

func parseCheckoutFetchRefs(v any) ([]string, error) {
	switch fv := v.(type) {
	case string:
		if strings.TrimSpace(fv) == "" {
			return nil, errors.New("checkout.fetch string value must not be empty")
		}
		return []string{fv}, nil
	case []any:
		return parseCheckoutFetchRefArray(fv)
	default:
		return nil, errors.New("checkout.fetch must be a string or an array of strings")
	}
}

func parseCheckoutFetchRefArray(fv []any) ([]string, error) {
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
	return refs, nil
}

func parseCheckoutBooleanFields(cfg *CheckoutConfig, m map[string]any) error {
	boolFields := []struct {
		key string
		set func(bool)
	}{
		{"lfs", func(v bool) { cfg.LFS = v }},
		{"current", func(v bool) { cfg.Current = v }},
		{"wiki", func(v bool) { cfg.Wiki = v }},
		{"force-clean-git-credentials", func(v bool) { cfg.CleanGitCredentials = v }},
	}
	for _, field := range boolFields {
		if v, ok := m[field.key]; ok {
			b, ok := v.(bool)
			if !ok {
				return fmt.Errorf("checkout.%s must be a boolean", field.key)
			}
			field.set(b)
		}
	}
	return nil
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
		checkoutManagerLog.Print("buildCheckoutsPromptContent: no checkouts configured, returning empty content")
		return ""
	}
	checkoutManagerLog.Printf("Building checkouts prompt content for %d checkout(s)", len(checkouts))

	var sb strings.Builder
	sb.WriteString("- **checkouts**: The following repositories have been checked out and are available in the workspace:\n")

	for _, cfg := range checkouts {
		if cfg == nil {
			continue
		}

		sb.WriteString(buildCheckoutPromptLine(cfg) + "\n")
	}

	// General guidance about unavailable branches
	sb.WriteString("  - **Note**: If a branch you need is not in the list above and is not listed as an additional fetched ref, " +
		"it has NOT been checked out. For private repositories you cannot fetch it. " +
		"If the branch is required and not available, exit with an error and ask the user to add it to the " +
		"`fetch:` option of the `checkout:` configuration (e.g., `fetch: [\"refs/pulls/open/*\"]` for all open PR refs, " +
		"or `fetch: [\"main\", \"feature/my-branch\"]` for specific branches).\n")

	// Credential warning — always present for any checkout-enabled workflow.
	sb.WriteString(checkoutsNoCredentialsWarning)

	return sb.String()
}

func buildCheckoutPromptLine(cfg *CheckoutConfig) string {
	absPath, isRoot := checkoutPromptPath(cfg)
	line := fmt.Sprintf("  - repo `%s` → `%s`", checkoutPromptRepo(cfg), absPath)
	if isRoot {
		line += " (cwd)"
	}
	if cfg.Wiki {
		line += " (wiki)"
	}
	if cfg.Current {
		line += " (**current** - this is the repository you are working on; use this as the target for all GitHub operations unless otherwise specified)"
	}
	line += checkoutPromptFetchDepth(cfg)
	if len(cfg.Fetch) > 0 {
		line += fmt.Sprintf(" [additional refs fetched: %s]", strings.Join(cfg.Fetch, ", "))
	}
	if strings.TrimSpace(cfg.SparseCheckout) != "" {
		line += " [sparse checkout enabled]"
	}
	return line
}

func checkoutPromptPath(cfg *CheckoutConfig) (string, bool) {
	relPath := strings.TrimPrefix(cfg.Path, "./")
	if relPath == "." {
		relPath = ""
	}
	isRoot := relPath == ""
	if isRoot {
		return "$GITHUB_WORKSPACE", true
	}
	return "$GITHUB_WORKSPACE/" + relPath, false
}

func checkoutPromptRepo(cfg *CheckoutConfig) string {
	repo := cfg.Repository
	if repo == "" {
		repo = "${{ github.repository }}"
	}
	if cfg.Wiki && !strings.HasSuffix(repo, ".wiki") {
		repo += ".wiki"
	}
	return repo
}

func checkoutPromptFetchDepth(cfg *CheckoutConfig) string {
	if cfg.FetchDepth != nil && *cfg.FetchDepth == 0 {
		return " [full history, all branches available as remote-tracking refs]"
	}
	if cfg.FetchDepth != nil {
		return fmt.Sprintf(" [shallow clone, fetch-depth=%d]", *cfg.FetchDepth)
	}
	return " [shallow clone, fetch-depth=1 (default)]"
}
