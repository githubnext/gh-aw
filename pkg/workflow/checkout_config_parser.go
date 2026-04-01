package workflow

import (
	"errors"
	"fmt"
	"strings"
)

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
