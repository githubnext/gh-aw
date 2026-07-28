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

func checkoutStringValue(m map[string]any, key string) (string, bool, error) {
	v, ok := m[key]
	if !ok {
		return "", false, nil
	}
	s, ok := v.(string)
	if !ok {
		return "", false, fmt.Errorf("checkout.%s must be a string", key)
	}
	return s, true, nil
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

func parseCheckoutCoreFields(m map[string]any, cfg *CheckoutConfig) error {
	for _, field := range []struct {
		key string
		set func(string)
	}{{"repository", func(value string) { cfg.Repository = value }}, {"ref", func(value string) { cfg.Ref = value }}} {
		if value, ok, err := checkoutStringValue(m, field.key); err != nil {
			return err
		} else if ok {
			field.set(value)
		}
	}
	if pathValue, ok, err := checkoutStringValue(m, "path"); err != nil {
		return err
	} else if ok {
		cfg.PathExplicit = true
		if pathValue == "." {
			pathValue = ""
		}
		cfg.Path = pathValue
	}
	if tokenValue, ok, err := checkoutStringValue(m, "github-token"); err != nil {
		return err
	} else if ok {
		cfg.GitHubToken = tokenValue
	} else if tokenValue, ok, err := checkoutStringValue(m, "token"); err != nil {
		return err
	} else if ok {
		cfg.GitHubToken = tokenValue
	}
	return nil
}

func parseCheckoutAuthFields(m map[string]any, cfg *CheckoutConfig) error {
	if value, ok := m["github-app"]; ok {
		appConfig, err := parseCheckoutAppConfig("github-app", value)
		if err != nil {
			return err
		}
		cfg.GitHubApp = appConfig
	}
	if value, ok := m["safe-outputs-github-app"]; ok {
		appConfig, err := parseCheckoutAppConfig("safe-outputs-github-app", value)
		if err != nil {
			return err
		}
		cfg.SafeOutputGitHubApp = appConfig
	}
	if _, ok := m["safe-output-github-app"]; ok {
		return errors.New("checkout.safe-output-github-app is not supported; use checkout.safe-outputs-github-app")
	}
	if cfg.GitHubToken != "" && cfg.GitHubApp != nil {
		checkoutManagerLog.Print("Rejecting checkout config: github-token and github-app are mutually exclusive")
		return errors.New("checkout: github-token and github-app are mutually exclusive; use one or the other")
	}
	return nil
}

func parseCheckoutFetchDepth(value any) (*int, error) {
	switch n := value.(type) {
	case int:
		depth := n
		return &depth, nil
	case int64:
		depth := int(n)
		return &depth, nil
	case uint64:
		depth := int(n)
		return &depth, nil
	case float64:
		if n != float64(int64(n)) {
			return nil, errors.New("checkout.fetch-depth must be an integer")
		}
		depth := int(n)
		return &depth, nil
	default:
		return nil, errors.New("checkout.fetch-depth must be an integer")
	}
}

func parseCheckoutFetch(value any) ([]string, error) {
	switch fv := value.(type) {
	case string:
		if strings.TrimSpace(fv) == "" {
			return nil, errors.New("checkout.fetch string value must not be empty")
		}
		return []string{fv}, nil
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
		return refs, nil
	default:
		return nil, errors.New("checkout.fetch must be a string or an array of strings")
	}
}

func parseCheckoutBehaviorFields(m map[string]any, cfg *CheckoutConfig) error {
	if value, ok := m["fetch-depth"]; ok {
		depth, err := parseCheckoutFetchDepth(value)
		if err != nil {
			return err
		}
		if depth != nil && *depth < 0 {
			return errors.New("checkout.fetch-depth must be >= 0")
		}
		cfg.FetchDepth = depth
	}
	if value, ok, err := checkoutStringValue(m, "sparse-checkout"); err != nil {
		return err
	} else if ok {
		cfg.SparseCheckout = value
	}
	if value, ok := m["submodules"]; ok {
		switch sv := value.(type) {
		case string:
			cfg.Submodules = sv
		case bool:
			if sv {
				cfg.Submodules = "true"
			} else {
				cfg.Submodules = "false"
			}
		default:
			return errors.New("checkout.submodules must be a string or boolean")
		}
	}
	for _, field := range []struct {
		key string
		set func(bool)
		err string
	}{{"lfs", func(v bool) { cfg.LFS = v }, "checkout.lfs must be a boolean"}, {"current", func(v bool) { cfg.Current = v }, "checkout.current must be a boolean"}, {"wiki", func(v bool) { cfg.Wiki = v }, "checkout.wiki must be a boolean"}, {"force-clean-git-credentials", func(v bool) { cfg.CleanGitCredentials = v }, "checkout.force-clean-git-credentials must be a boolean"}} {
		if value, ok := m[field.key]; ok {
			b, ok := value.(bool)
			if !ok {
				return errors.New(field.err)
			}
			field.set(b)
		}
	}
	if value, ok := m["fetch"]; ok {
		fetchRefs, err := parseCheckoutFetch(value)
		if err != nil {
			return err
		}
		cfg.Fetch = fetchRefs
	}
	return nil
}

// checkoutConfigFromMap converts a raw map to a CheckoutConfig.
func checkoutConfigFromMap(m map[string]any) (*CheckoutConfig, error) {
	cfg := &CheckoutConfig{}
	if err := parseCheckoutCoreFields(m, cfg); err != nil {
		return nil, err
	}
	if err := parseCheckoutAuthFields(m, cfg); err != nil {
		return nil, err
	}
	checkoutManagerLog.Printf("Parsed checkout config: repo=%q, ref=%q, path=%q, current=%v, hasToken=%v, hasApp=%v", cfg.Repository, cfg.Ref, cfg.Path, cfg.Current, cfg.GitHubToken != "", cfg.GitHubApp != nil)
	if err := parseCheckoutBehaviorFields(m, cfg); err != nil {
		return nil, err
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

		// Determine repo: use configured value or fall back to the triggering repository expression.
		// For wiki checkouts, append the ".wiki" suffix so the prompt accurately reflects what was checked out.
		repo := cfg.Repository
		if repo == "" {
			repo = "${{ github.repository }}"
		}
		if cfg.Wiki {
			if !strings.HasSuffix(repo, ".wiki") {
				repo += ".wiki"
			}
		}

		line := fmt.Sprintf("  - repo `%s` → `%s`", repo, absPath)
		if isRoot {
			line += " (cwd)"
		}
		if cfg.Wiki {
			line += " (wiki)"
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
		if strings.TrimSpace(cfg.SparseCheckout) != "" {
			line += " [sparse checkout enabled]"
		}

		sb.WriteString(line + "\n")
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
