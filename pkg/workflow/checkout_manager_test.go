//go:build !integration

package workflow

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewCheckoutManager verifies that a CheckoutManager can be created with user configs.
func TestNewCheckoutManager(t *testing.T) {
	t.Run("empty configs produces empty manager", func(t *testing.T) {
		cm := NewCheckoutManager(nil)
		// HasUserCheckouts removed (dead code)
		assert.Nil(t, cm.GetDefaultCheckoutOverride(), "empty manager should have no default override")
	})

	t.Run("single default override", func(t *testing.T) {
		depth := 0
		cm := NewCheckoutManager([]*CheckoutConfig{
			{FetchDepth: &depth},
		})
		// HasUserCheckouts removed (dead code)
		override := cm.GetDefaultCheckoutOverride()
		require.NotNil(t, override, "should have default override")
		require.NotNil(t, override.fetchDepth, "fetch depth should be set")
		assert.Equal(t, 0, *override.fetchDepth, "fetch depth should be 0")
	})

	t.Run("custom github-token on default checkout", func(t *testing.T) {
		cm := NewCheckoutManager([]*CheckoutConfig{
			{GitHubToken: "${{ secrets.MY_TOKEN }}"},
		})
		override := cm.GetDefaultCheckoutOverride()
		require.NotNil(t, override, "should have default override")
		assert.Equal(t, "${{ secrets.MY_TOKEN }}", override.token, "github-token should be set")
	})
}

// TestCheckoutManagerMerging verifies that duplicate checkout configs are merged.
func TestCheckoutManagerMerging(t *testing.T) {
	t.Run("duplicate default checkout takes deepest fetch-depth", func(t *testing.T) {
		depth1 := 1
		depth10 := 10
		cm := NewCheckoutManager([]*CheckoutConfig{
			{FetchDepth: &depth1},
			{FetchDepth: &depth10},
		})
		assert.Len(t, cm.ordered, 1, "should have merged into a single entry")
		override := cm.GetDefaultCheckoutOverride()
		require.NotNil(t, override.fetchDepth, "fetch depth should be set after merge")
		assert.Equal(t, 10, *override.fetchDepth, "should use deeper fetch-depth (10 > 1)")
	})

	t.Run("zero fetch-depth wins over any positive value", func(t *testing.T) {
		depth0 := 0
		depth5 := 5
		cm := NewCheckoutManager([]*CheckoutConfig{
			{FetchDepth: &depth5},
			{FetchDepth: &depth0},
		})
		override := cm.GetDefaultCheckoutOverride()
		require.NotNil(t, override.fetchDepth, "fetch depth should be set")
		assert.Equal(t, 0, *override.fetchDepth, "0 (full history) should win")
	})

	t.Run("sparse-checkout patterns are merged", func(t *testing.T) {
		cm := NewCheckoutManager([]*CheckoutConfig{
			{Path: "./workspace", SparseCheckout: ".github/"},
			{Path: "./workspace", SparseCheckout: "src/"},
		})
		assert.Len(t, cm.ordered, 1, "should have merged into a single entry")
		additional := cm.GenerateAdditionalCheckoutSteps(func(s string) string { return s })
		combined := strings.Join(additional, "")
		assert.Contains(t, combined, ".github/", "should contain first sparse pattern")
		assert.Contains(t, combined, "src/", "should contain second sparse pattern")
	})

	t.Run("different paths produce separate checkouts", func(t *testing.T) {
		cm := NewCheckoutManager([]*CheckoutConfig{
			{Path: "./workspace1"},
			{Path: "./workspace2"},
		})
		assert.Len(t, cm.ordered, 2, "different paths should not be merged")
	})

	t.Run("different repos produce separate checkouts", func(t *testing.T) {
		cm := NewCheckoutManager([]*CheckoutConfig{
			{Repository: "owner/repo1", Path: "./r1"},
			{Repository: "owner/repo2", Path: "./r2"},
		})
		assert.Len(t, cm.ordered, 2, "different repos should not be merged")
	})

	t.Run("same path with different refs merges to first ref", func(t *testing.T) {
		cm := NewCheckoutManager([]*CheckoutConfig{
			{Path: "./workspace", Ref: "main"},
			{Path: "./workspace", Ref: "develop"},
		})
		assert.Len(t, cm.ordered, 1, "same path should be merged")
		assert.Equal(t, "main", cm.ordered[0].ref, "first-seen ref should win")
	})

	t.Run("path dot and empty path are normalized to the same root checkout", func(t *testing.T) {
		depth0 := 0
		cm := NewCheckoutManager([]*CheckoutConfig{
			{Path: ".", FetchDepth: nil},
			{Path: "", FetchDepth: &depth0},
		})
		assert.Len(t, cm.ordered, 1, "path '.' and '' should merge as the same root checkout")
		assert.Empty(t, cm.ordered[0].key.path, "normalized path should be empty string")
		require.NotNil(t, cm.ordered[0].fetchDepth, "fetch depth should be set from second config")
		assert.Equal(t, 0, *cm.ordered[0].fetchDepth, "fetch depth 0 should win")
	})
}

// TestGenerateDefaultCheckoutStep verifies the default checkout step output.
func TestGenerateDefaultCheckoutStep(t *testing.T) {
	getPin := func(action string) string { return action + "@v4" }

	t.Run("default checkout has persist-credentials false", func(t *testing.T) {
		cm := NewCheckoutManager(nil)
		lines := cm.GenerateDefaultCheckoutStep(false, "", getPin)
		combined := strings.Join(lines, "")
		assert.Contains(t, combined, "persist-credentials: false", "must always have persist-credentials: false")
		assert.Contains(t, combined, "Checkout repository", "should have default step name")
		assert.Contains(t, combined, "actions/checkout@v4", "should use pinned checkout action")
	})

	t.Run("user github-token is included in default checkout", func(t *testing.T) {
		cm := NewCheckoutManager([]*CheckoutConfig{
			{GitHubToken: "${{ secrets.MY_TOKEN }}"},
		})
		lines := cm.GenerateDefaultCheckoutStep(false, "", getPin)
		combined := strings.Join(lines, "")
		assert.Contains(t, combined, "token: ${{ secrets.MY_TOKEN }}", "should include custom token in actions/checkout 'token' input")
		assert.Contains(t, combined, "persist-credentials: false", "must always have persist-credentials: false even with custom token")
	})

	t.Run("fetch-depth override is included", func(t *testing.T) {
		depth := 0
		cm := NewCheckoutManager([]*CheckoutConfig{
			{FetchDepth: &depth},
		})
		lines := cm.GenerateDefaultCheckoutStep(false, "", getPin)
		combined := strings.Join(lines, "")
		assert.Contains(t, combined, "fetch-depth: 0", "should include fetch-depth override")
	})

	t.Run("ref override is included", func(t *testing.T) {
		cm := NewCheckoutManager([]*CheckoutConfig{
			{Ref: "develop"},
		})
		lines := cm.GenerateDefaultCheckoutStep(false, "", getPin)
		combined := strings.Join(lines, "")
		assert.Contains(t, combined, "ref: develop", "should include ref override")
	})

	t.Run("trial mode overrides user config", func(t *testing.T) {
		cm := NewCheckoutManager([]*CheckoutConfig{
			{GitHubToken: "${{ secrets.MY_TOKEN }}"},
		})
		lines := cm.GenerateDefaultCheckoutStep(true, "owner/trial-repo", getPin)
		combined := strings.Join(lines, "")
		assert.Contains(t, combined, "repository: owner/trial-repo", "trial repo should be in output")
		// In trial mode, user token should NOT be emitted (trial uses its own token)
		assert.NotContains(t, combined, "secrets.MY_TOKEN", "user token should not appear in trial mode")
	})

	t.Run("sparse-checkout override is included", func(t *testing.T) {
		cm := NewCheckoutManager([]*CheckoutConfig{
			{SparseCheckout: ".github/\nsrc/"},
		})
		lines := cm.GenerateDefaultCheckoutStep(false, "", getPin)
		combined := strings.Join(lines, "")
		assert.Contains(t, combined, "sparse-checkout: |", "should include sparse-checkout header")
		assert.Contains(t, combined, ".github/", "should include first pattern")
		assert.Contains(t, combined, "src/", "should include second pattern")
	})
}

// TestGenerateAdditionalCheckoutSteps verifies that non-default checkouts are emitted correctly.
func TestGenerateAdditionalCheckoutSteps(t *testing.T) {
	getPin := func(action string) string { return action + "@v4" }

	t.Run("no additional checkouts when only default configured", func(t *testing.T) {
		depth := 0
		cm := NewCheckoutManager([]*CheckoutConfig{
			{FetchDepth: &depth},
		})
		lines := cm.GenerateAdditionalCheckoutSteps(getPin)
		assert.Empty(t, lines, "should produce no additional checkout steps")
	})

	t.Run("additional checkout for different path", func(t *testing.T) {
		cm := NewCheckoutManager([]*CheckoutConfig{
			{Repository: "owner/libs", Path: "./libs/owner-libs", Ref: "main"},
		})
		lines := cm.GenerateAdditionalCheckoutSteps(getPin)
		combined := strings.Join(lines, "")
		assert.Contains(t, combined, "repository: owner/libs", "should include repo")
		assert.Contains(t, combined, "path: ./libs/owner-libs", "should include path")
		assert.Contains(t, combined, "ref: main", "should include ref")
		assert.Contains(t, combined, "persist-credentials: false", "must always have persist-credentials: false")
	})

	t.Run("additional checkout with LFS enabled", func(t *testing.T) {
		cm := NewCheckoutManager([]*CheckoutConfig{
			{Path: "./lfs-repo", LFS: true},
		})
		lines := cm.GenerateAdditionalCheckoutSteps(getPin)
		combined := strings.Join(lines, "")
		assert.Contains(t, combined, "lfs: true", "should include LFS option")
	})

	t.Run("additional checkout with recursive submodules", func(t *testing.T) {
		cm := NewCheckoutManager([]*CheckoutConfig{
			{Path: "./with-submodules", Submodules: "recursive"},
		})
		lines := cm.GenerateAdditionalCheckoutSteps(getPin)
		combined := strings.Join(lines, "")
		assert.Contains(t, combined, "submodules: recursive", "should include submodules option")
	})

	t.Run("additional checkout emits actions/checkout token input from github-token config", func(t *testing.T) {
		cm := NewCheckoutManager([]*CheckoutConfig{
			{Path: "./libs", Repository: "owner/libs", GitHubToken: "${{ secrets.MY_TOKEN }}"},
		})
		lines := cm.GenerateAdditionalCheckoutSteps(getPin)
		combined := strings.Join(lines, "")
		assert.Contains(t, combined, "token: ${{ secrets.MY_TOKEN }}", "actions/checkout input must be 'token' even when frontmatter uses 'github-token'")
		assert.NotContains(t, combined, "github-token:", "must not emit 'github-token' as actions/checkout input")
	})
}

// TestParseCheckoutConfigs verifies parsing of raw frontmatter values.
func TestParseCheckoutConfigs(t *testing.T) {
	t.Run("nil returns nil", func(t *testing.T) {
		configs, err := ParseCheckoutConfigs(nil)
		require.NoError(t, err, "nil should not error")
		assert.Nil(t, configs, "nil input should return nil configs")
	})

	t.Run("single object with github-token", func(t *testing.T) {
		raw := map[string]any{
			"fetch-depth":  float64(0),
			"github-token": "${{ secrets.MY_TOKEN }}",
		}
		configs, err := ParseCheckoutConfigs(raw)
		require.NoError(t, err, "single object should parse without error")
		require.Len(t, configs, 1, "should produce one config")
		assert.Equal(t, "${{ secrets.MY_TOKEN }}", configs[0].GitHubToken, "github-token should be set")
		require.NotNil(t, configs[0].FetchDepth, "fetch-depth should be set")
		assert.Equal(t, 0, *configs[0].FetchDepth, "fetch-depth should be 0")
	})

	t.Run("backward compat: token key still works", func(t *testing.T) {
		raw := map[string]any{
			"token": "${{ secrets.MY_TOKEN }}",
		}
		configs, err := ParseCheckoutConfigs(raw)
		require.NoError(t, err, "legacy token key should parse without error")
		require.Len(t, configs, 1, "should produce one config")
		assert.Equal(t, "${{ secrets.MY_TOKEN }}", configs[0].GitHubToken, "legacy token should populate GitHubToken")
	})

	t.Run("github-app config is parsed", func(t *testing.T) {
		raw := map[string]any{
			"repository": "owner/target-repo",
			"github-app": map[string]any{
				"app-id":      "${{ vars.APP_ID }}",
				"private-key": "${{ secrets.APP_PRIVATE_KEY }}",
			},
		}
		configs, err := ParseCheckoutConfigs(raw)
		require.NoError(t, err, "github-app config should parse without error")
		require.Len(t, configs, 1)
		require.NotNil(t, configs[0].GitHubApp, "github-app config should be set")
		assert.Equal(t, "${{ vars.APP_ID }}", configs[0].GitHubApp.AppID, "app-id should be set")
		assert.Equal(t, "${{ secrets.APP_PRIVATE_KEY }}", configs[0].GitHubApp.PrivateKey, "private-key should be set")
	})

	t.Run("github-app config with owner and repositories", func(t *testing.T) {
		raw := map[string]any{
			"repository": "owner/target-repo",
			"github-app": map[string]any{
				"app-id":       "${{ vars.APP_ID }}",
				"private-key":  "${{ secrets.APP_PRIVATE_KEY }}",
				"owner":        "my-org",
				"repositories": []any{"repo-a", "repo-b"},
			},
		}
		configs, err := ParseCheckoutConfigs(raw)
		require.NoError(t, err, "github-app config with owner should parse without error")
		require.Len(t, configs, 1)
		require.NotNil(t, configs[0].GitHubApp)
		assert.Equal(t, "my-org", configs[0].GitHubApp.Owner)
		assert.Equal(t, []string{"repo-a", "repo-b"}, configs[0].GitHubApp.Repositories)
	})

	t.Run("github-app config accepts client-id", func(t *testing.T) {
		raw := map[string]any{
			"repository": "owner/target-repo",
			"github-app": map[string]any{
				"client-id":   "${{ vars.CLIENT_ID }}",
				"private-key": "${{ secrets.APP_PRIVATE_KEY }}",
			},
		}
		configs, err := ParseCheckoutConfigs(raw)
		require.NoError(t, err, "github-app config with client-id should parse without error")
		require.Len(t, configs, 1)
		require.NotNil(t, configs[0].GitHubApp, "github-app config should be set")
		assert.Equal(t, "${{ vars.CLIENT_ID }}", configs[0].GitHubApp.AppID, "client-id should populate AppID")
	})

	t.Run("github-token and github-app are mutually exclusive", func(t *testing.T) {
		raw := map[string]any{
			"github-token": "${{ secrets.MY_TOKEN }}",
			"github-app": map[string]any{
				"app-id":      "${{ vars.APP_ID }}",
				"private-key": "${{ secrets.APP_PRIVATE_KEY }}",
			},
		}
		_, err := ParseCheckoutConfigs(raw)
		require.Error(t, err, "github-token and github-app together should return error")
		assert.Contains(t, err.Error(), "mutually exclusive", "error should mention mutual exclusivity")
	})

	t.Run("github-app config missing app-id returns error", func(t *testing.T) {
		raw := map[string]any{
			"github-app": map[string]any{
				"private-key": "${{ secrets.APP_PRIVATE_KEY }}",
			},
		}
		_, err := ParseCheckoutConfigs(raw)
		require.Error(t, err, "github-app without app-id should return error")
		assert.Contains(t, err.Error(), "client-id (or app-id) and private-key")
	})

	t.Run("github-app config missing private-key returns error", func(t *testing.T) {
		raw := map[string]any{
			"github-app": map[string]any{
				"app-id": "${{ vars.APP_ID }}",
			},
		}
		_, err := ParseCheckoutConfigs(raw)
		require.Error(t, err, "github-app without private-key should return error")
		assert.Contains(t, err.Error(), "client-id (or app-id) and private-key")
	})

	t.Run("github-app must be an object", func(t *testing.T) {
		raw := map[string]any{
			"github-app": "not-an-object",
		}
		_, err := ParseCheckoutConfigs(raw)
		require.Error(t, err, "non-object github-app should return error")
		assert.Contains(t, err.Error(), "checkout.github-app must be an object")
	})

	t.Run("array of objects", func(t *testing.T) {
		raw := []any{
			map[string]any{"path": "."},
			map[string]any{"repository": "owner/repo", "path": "./libs"},
		}
		configs, err := ParseCheckoutConfigs(raw)
		require.NoError(t, err, "array should parse without error")
		require.Len(t, configs, 2, "should produce two configs")
		assert.Empty(t, configs[0].Path, "first path should be normalized from '.' to empty")
		assert.Equal(t, "owner/repo", configs[1].Repository, "second repo should be set")
	})

	t.Run("invalid type returns error", func(t *testing.T) {
		_, err := ParseCheckoutConfigs("invalid")
		assert.Error(t, err, "string value should return an error")
	})

	t.Run("array with non-object entry returns error", func(t *testing.T) {
		raw := []any{"not-an-object"}
		_, err := ParseCheckoutConfigs(raw)
		assert.Error(t, err, "array with non-object entry should return error")
	})

	t.Run("submodules as bool true", func(t *testing.T) {
		raw := map[string]any{"submodules": true}
		configs, err := ParseCheckoutConfigs(raw)
		require.NoError(t, err)
		require.Len(t, configs, 1)
		assert.Equal(t, "true", configs[0].Submodules, "bool true should convert to string 'true'")
	})

	t.Run("submodules as bool false", func(t *testing.T) {
		raw := map[string]any{"submodules": false}
		configs, err := ParseCheckoutConfigs(raw)
		require.NoError(t, err)
		require.Len(t, configs, 1)
		assert.Equal(t, "false", configs[0].Submodules, "bool false should convert to string 'false'")
	})

	t.Run("submodules as string recursive", func(t *testing.T) {
		raw := map[string]any{"submodules": "recursive"}
		configs, err := ParseCheckoutConfigs(raw)
		require.NoError(t, err)
		require.Len(t, configs, 1)
		assert.Equal(t, "recursive", configs[0].Submodules, "string should be preserved")
	})
}

// TestDeeperFetchDepth tests the fetch-depth comparison logic.
func TestDeeperFetchDepth(t *testing.T) {
	ptr := func(n int) *int { return &n }

	tests := []struct {
		name     string
		a, b     *int
		expected *int
	}{
		{"both nil returns nil", nil, nil, nil},
		{"a nil returns b", nil, ptr(5), ptr(5)},
		{"b nil returns a", ptr(5), nil, ptr(5)},
		{"0 beats positive", ptr(0), ptr(5), ptr(0)},
		{"positive beats 0 (reversed)", ptr(5), ptr(0), ptr(0)},
		{"larger positive wins", ptr(3), ptr(10), ptr(10)},
		{"smaller positive loses", ptr(10), ptr(3), ptr(10)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := deeperFetchDepth(tt.a, tt.b)
			if tt.expected == nil {
				assert.Nil(t, result, "should be nil")
			} else {
				require.NotNil(t, result, "should not be nil")
				assert.Equal(t, *tt.expected, *result, "should return correct value")
			}
		})
	}
}

// TestMergeSparsePatterns tests pattern deduplication and merging.
func TestMergeSparsePatterns(t *testing.T) {
	t.Run("merges unique patterns", func(t *testing.T) {
		result := mergeSparsePatterns([]string{".github/"}, "src/\ndocs/")
		assert.Equal(t, []string{".github/", "src/", "docs/"}, result, "should contain all unique patterns")
	})

	t.Run("deduplicates patterns", func(t *testing.T) {
		result := mergeSparsePatterns([]string{".github/"}, ".github/\nsrc/")
		assert.Equal(t, []string{".github/", "src/"}, result, "should deduplicate .github/")
	})

	t.Run("nil existing with new patterns", func(t *testing.T) {
		result := mergeSparsePatterns(nil, "src/\ndocs/")
		assert.Equal(t, []string{"src/", "docs/"}, result, "should return new patterns")
	})

	t.Run("empty new patterns preserves existing", func(t *testing.T) {
		result := mergeSparsePatterns([]string{"src/"}, "")
		assert.Equal(t, []string{"src/"}, result, "should preserve existing patterns")
	})
}

// TestCheckoutCurrentFlag verifies the current: true checkout flag behavior.
func TestCheckoutCurrentFlag(t *testing.T) {
	t.Run("parse current: true from single object", func(t *testing.T) {
		raw := map[string]any{
			"repository": "owner/target-repo",
			"current":    true,
		}
		configs, err := ParseCheckoutConfigs(raw)
		require.NoError(t, err, "should parse without error")
		require.Len(t, configs, 1, "should produce one config")
		assert.True(t, configs[0].Current, "current flag should be true")
		assert.Equal(t, "owner/target-repo", configs[0].Repository, "repository should be set")
	})

	t.Run("parse current: false from map", func(t *testing.T) {
		raw := map[string]any{"current": false}
		configs, err := ParseCheckoutConfigs(raw)
		require.NoError(t, err, "should parse without error")
		require.Len(t, configs, 1)
		assert.False(t, configs[0].Current, "current flag should be false")
	})

	t.Run("invalid current type returns error", func(t *testing.T) {
		raw := map[string]any{"current": "yes"}
		_, err := ParseCheckoutConfigs(raw)
		assert.Error(t, err, "non-boolean current should return error")
	})

	t.Run("multiple current: true in array returns error", func(t *testing.T) {
		raw := []any{
			map[string]any{"repository": "owner/repo1", "path": "./r1", "current": true},
			map[string]any{"repository": "owner/repo2", "path": "./r2", "current": true},
		}
		_, err := ParseCheckoutConfigs(raw)
		require.Error(t, err, "multiple current: true should return error")
		assert.Contains(t, err.Error(), "only one checkout target may have current: true", "error should mention the constraint")
	})

	t.Run("single current: true in array is valid", func(t *testing.T) {
		raw := []any{
			map[string]any{"path": "."},
			map[string]any{"repository": "owner/target", "path": "./target", "current": true},
		}
		configs, err := ParseCheckoutConfigs(raw)
		require.NoError(t, err, "single current: true in array should be valid")
		require.Len(t, configs, 2)
		assert.False(t, configs[0].Current, "first checkout should not be current")
		assert.True(t, configs[1].Current, "second checkout should be current")
	})
}

// TestBuildCheckoutsPromptContent verifies the prompt content generation for the checkout list.
func TestBuildCheckoutsPromptContent(t *testing.T) {
	t.Run("nil slice returns empty string", func(t *testing.T) {
		assert.Empty(t, buildCheckoutsPromptContent(nil), "nil should return empty string")
	})

	t.Run("empty slice returns empty string", func(t *testing.T) {
		assert.Empty(t, buildCheckoutsPromptContent([]*CheckoutConfig{}), "empty slice should return empty string")
	})

	t.Run("default checkout with no repo uses github.repository expression and cwd", func(t *testing.T) {
		content := buildCheckoutsPromptContent([]*CheckoutConfig{
			{},
		})
		assert.Contains(t, content, "$GITHUB_WORKSPACE", "should show full workspace path for root checkout")
		assert.Contains(t, content, "(cwd)", "root checkout should be marked as cwd")
		assert.Contains(t, content, "${{ github.repository }}", "should reference github.repository expression for default checkout")
	})

	t.Run("checkout with explicit repo shows full path", func(t *testing.T) {
		content := buildCheckoutsPromptContent([]*CheckoutConfig{
			{Repository: "owner/target", Path: "./target"},
		})
		assert.Contains(t, content, "$GITHUB_WORKSPACE/target", "should show full workspace path")
		assert.Contains(t, content, "owner/target", "should show the configured repo")
		assert.NotContains(t, content, "github.repository", "should not include github.repository expression for explicit repo")
		assert.NotContains(t, content, "(cwd)", "non-root checkout should not be marked as cwd")
	})

	t.Run("current checkout is marked", func(t *testing.T) {
		content := buildCheckoutsPromptContent([]*CheckoutConfig{
			{Repository: "owner/target", Path: "./target", Current: true},
		})
		assert.Contains(t, content, "**current**", "current checkout should be marked")
		assert.Contains(t, content, "this is the repository you are working on", "current checkout should have instructions")
	})

	t.Run("non-current checkout is not marked", func(t *testing.T) {
		content := buildCheckoutsPromptContent([]*CheckoutConfig{
			{Repository: "owner/libs", Path: "./libs"},
		})
		assert.NotContains(t, content, "**current**", "non-current checkout should not be marked")
	})

	t.Run("multiple checkouts all listed", func(t *testing.T) {
		content := buildCheckoutsPromptContent([]*CheckoutConfig{
			{Path: ""},
			{Repository: "owner/target", Path: "./target", Current: true},
			{Repository: "owner/libs", Path: "./libs"},
		})
		assert.Contains(t, content, "$GITHUB_WORKSPACE", "should include workspace root for root checkout")
		assert.Contains(t, content, "(cwd)", "root checkout should be marked as cwd")
		assert.Contains(t, content, "$GITHUB_WORKSPACE/target", "should include full path for target checkout")
		assert.Contains(t, content, "owner/target", "should include target repo")
		assert.Contains(t, content, "$GITHUB_WORKSPACE/libs", "should include full path for libs checkout")
		assert.Contains(t, content, "owner/libs", "should include libs repo")
		assert.Contains(t, content, "**current**", "current checkout should be marked")
	})

	t.Run("default fetch-depth annotation shows shallow clone", func(t *testing.T) {
		content := buildCheckoutsPromptContent([]*CheckoutConfig{
			{Repository: "owner/repo"},
		})
		assert.Contains(t, content, "shallow clone, fetch-depth=1 (default)", "should show default shallow clone annotation")
	})

	t.Run("fetch-depth 0 annotation shows full history", func(t *testing.T) {
		depth := 0
		content := buildCheckoutsPromptContent([]*CheckoutConfig{
			{Repository: "owner/repo", FetchDepth: &depth},
		})
		assert.Contains(t, content, "full history", "should show full history annotation")
	})

	t.Run("non-zero fetch-depth annotation shows value", func(t *testing.T) {
		depth := 50
		content := buildCheckoutsPromptContent([]*CheckoutConfig{
			{Repository: "owner/repo", FetchDepth: &depth},
		})
		assert.Contains(t, content, "fetch-depth=50", "should show configured fetch-depth")
	})

	t.Run("fetch refs are listed in prompt", func(t *testing.T) {
		content := buildCheckoutsPromptContent([]*CheckoutConfig{
			{Repository: "owner/repo", Fetch: []string{"refs/pulls/open/*", "main"}},
		})
		assert.Contains(t, content, "additional refs fetched", "should mention additional refs")
		assert.Contains(t, content, "refs/pulls/open/*", "should list the refs/pulls/open/* pattern")
		assert.Contains(t, content, "main", "should list the main branch")
	})

	t.Run("unavailable branch note is always present", func(t *testing.T) {
		content := buildCheckoutsPromptContent([]*CheckoutConfig{
			{Repository: "owner/repo"},
		})
		assert.Contains(t, content, "has NOT been checked out", "should mention branches that are not checked out")
		assert.Contains(t, content, "fetch:", "should mention the fetch option for resolution")
	})
}

// TestParseFetchField verifies parsing of the fetch field in checkout configuration.
func TestParseFetchField(t *testing.T) {
	t.Run("fetch as array of strings", func(t *testing.T) {
		raw := map[string]any{
			"fetch": []any{"*", "refs/pulls/open/*"},
		}
		configs, err := ParseCheckoutConfigs(raw)
		require.NoError(t, err, "should parse without error")
		require.Len(t, configs, 1)
		assert.Equal(t, []string{"*", "refs/pulls/open/*"}, configs[0].Fetch, "fetch should be set")
	})

	t.Run("fetch as single string", func(t *testing.T) {
		raw := map[string]any{
			"fetch": "*",
		}
		configs, err := ParseCheckoutConfigs(raw)
		require.NoError(t, err, "single string fetch should parse without error")
		require.Len(t, configs, 1)
		assert.Equal(t, []string{"*"}, configs[0].Fetch, "single string should become a one-element slice")
	})

	t.Run("fetch with specific branch names", func(t *testing.T) {
		raw := map[string]any{
			"fetch": []any{"main", "feature/my-branch"},
		}
		configs, err := ParseCheckoutConfigs(raw)
		require.NoError(t, err)
		require.Len(t, configs, 1)
		assert.Equal(t, []string{"main", "feature/my-branch"}, configs[0].Fetch)
	})

	t.Run("invalid fetch type returns error", func(t *testing.T) {
		raw := map[string]any{
			"fetch": 42,
		}
		_, err := ParseCheckoutConfigs(raw)
		assert.Error(t, err, "integer fetch should return error")
	})

	t.Run("fetch array with non-string element returns error", func(t *testing.T) {
		raw := map[string]any{
			"fetch": []any{"main", 123},
		}
		_, err := ParseCheckoutConfigs(raw)
		assert.Error(t, err, "array with non-string entry should return error")
	})

	t.Run("fetch absent means no fetch refs", func(t *testing.T) {
		raw := map[string]any{
			"repository": "owner/repo",
		}
		configs, err := ParseCheckoutConfigs(raw)
		require.NoError(t, err)
		require.Len(t, configs, 1)
		assert.Empty(t, configs[0].Fetch, "absent fetch should produce empty slice")
	})
}

// TestFetchRefToRefspec verifies the refspec expansion logic.
func TestFetchRefToRefspec(t *testing.T) {
	tests := []struct {
		pattern  string
		expected string
	}{
		{"*", "+refs/heads/*:refs/remotes/origin/*"},
		{"refs/pulls/open/*", "+refs/pull/*/head:refs/remotes/origin/pull/*/head"},
		{"main", "+refs/heads/main:refs/remotes/origin/main"},
		{"feature/my-branch", "+refs/heads/feature/my-branch:refs/remotes/origin/feature/my-branch"},
		{"feature/*", "+refs/heads/feature/*:refs/remotes/origin/feature/*"},
	}
	for _, tt := range tests {
		t.Run(tt.pattern, func(t *testing.T) {
			got := fetchRefToRefspec(tt.pattern)
			assert.Equal(t, tt.expected, got, "refspec should match expected value")
		})
	}
}

// TestMergeFetchRefs verifies that fetch ref lists are properly unioned.
func TestMergeFetchRefs(t *testing.T) {
	t.Run("union of two disjoint sets", func(t *testing.T) {
		result := mergeFetchRefs([]string{"*"}, []string{"refs/pulls/open/*"})
		assert.Equal(t, []string{"*", "refs/pulls/open/*"}, result)
	})

	t.Run("removes duplicate refs", func(t *testing.T) {
		result := mergeFetchRefs([]string{"main"}, []string{"main", "develop"})
		assert.Equal(t, []string{"main", "develop"}, result)
	})

	t.Run("nil existing returns new refs", func(t *testing.T) {
		result := mergeFetchRefs(nil, []string{"*"})
		assert.Equal(t, []string{"*"}, result)
	})
}

// TestGenerateFetchStep verifies that the git fetch YAML step is generated correctly.
func TestGenerateFetchStep(t *testing.T) {
	t.Run("no fetch refs returns empty string", func(t *testing.T) {
		entry := &resolvedCheckout{}
		got := generateFetchStepLines(entry, 0)
		assert.Empty(t, got, "empty fetchRefs should produce no step")
	})

	t.Run("fetch all branches uses star refspec", func(t *testing.T) {
		entry := &resolvedCheckout{
			fetchRefs: []string{"*"},
		}
		got := generateFetchStepLines(entry, 0)
		assert.Contains(t, got, "Fetch additional refs", "should include step name")
		assert.Contains(t, got, "+refs/heads/*:refs/remotes/origin/*", "should include correct refspec")
		assert.Contains(t, got, "GH_AW_FETCH_TOKEN", "should set fetch token env var")
		assert.Contains(t, got, "http.extraheader=Authorization:", "should configure credentials via http.extraheader")
		// When no custom token set, falls back to the effective GitHub token chain
		assert.Contains(t, got, "GH_AW_GITHUB_TOKEN", "should fall back to GH_AW token chain when no checkout token set")
		// base64 must use -w 0 to prevent line wrapping with long tokens (e.g. fine-grained PATs)
		assert.Contains(t, got, "base64 -w 0", "should use base64 -w 0 to disable line wrapping")
	})

	t.Run("fetch refs/pulls/open/* uses PR refspec", func(t *testing.T) {
		entry := &resolvedCheckout{
			fetchRefs: []string{"refs/pulls/open/*"},
		}
		got := generateFetchStepLines(entry, 0)
		assert.Contains(t, got, "+refs/pull/*/head:refs/remotes/origin/pull/*/head", "should include PR refspec")
	})

	t.Run("custom token is used in fetch step", func(t *testing.T) {
		entry := &resolvedCheckout{
			fetchRefs: []string{"main"},
			token:     "${{ secrets.MY_PAT }}",
		}
		got := generateFetchStepLines(entry, 0)
		assert.Contains(t, got, "${{ secrets.MY_PAT }}", "should use custom token from checkout config")
		assert.NotContains(t, got, "github.token", "should not fall back to github.token when custom token set")
	})

	t.Run("repository name used in step name", func(t *testing.T) {
		entry := &resolvedCheckout{
			key:       checkoutKey{repository: "owner/side-repo"},
			fetchRefs: []string{"main"},
		}
		got := generateFetchStepLines(entry, 0)
		assert.Contains(t, got, "Fetch additional refs for owner/side-repo", "should include repo in step name")
	})

	t.Run("non-root path uses -C flag", func(t *testing.T) {
		entry := &resolvedCheckout{
			key:       checkoutKey{path: "libs/other"},
			fetchRefs: []string{"main"},
		}
		got := generateFetchStepLines(entry, 0)
		assert.Contains(t, got, `-C "${{ github.workspace }}/libs/other"`, "should use -C flag for non-root path")
	})

	t.Run("root path does not add -C flag", func(t *testing.T) {
		entry := &resolvedCheckout{
			fetchRefs: []string{"main"},
		}
		got := generateFetchStepLines(entry, 0)
		assert.NotContains(t, got, "-C ", "root checkout should not use -C flag")
	})

	t.Run("multiple refspecs in single command", func(t *testing.T) {
		entry := &resolvedCheckout{
			fetchRefs: []string{"*", "refs/pulls/open/*"},
		}
		got := generateFetchStepLines(entry, 0)
		assert.Contains(t, got, "+refs/heads/*:refs/remotes/origin/*", "should include branches refspec")
		assert.Contains(t, got, "+refs/pull/*/head:refs/remotes/origin/pull/*/head", "should include PR refspec")
	})
}

// TestCheckoutManagerFetchMerging verifies that fetch refs are merged correctly.
func TestCheckoutManagerFetchMerging(t *testing.T) {
	t.Run("fetch refs are merged for same checkout", func(t *testing.T) {
		cm := NewCheckoutManager([]*CheckoutConfig{
			{Repository: "owner/repo", Path: "./r", Fetch: []string{"main"}},
			{Repository: "owner/repo", Path: "./r", Fetch: []string{"develop"}},
		})
		require.Len(t, cm.ordered, 1, "same (repo, path) should merge")
		assert.Equal(t, []string{"main", "develop"}, cm.ordered[0].fetchRefs, "fetch refs should be unioned")
	})

	t.Run("fetch refs preserved when no merge", func(t *testing.T) {
		cm := NewCheckoutManager([]*CheckoutConfig{
			{Repository: "owner/repo", Path: "./r", Fetch: []string{"*", "refs/pulls/open/*"}},
		})
		assert.Equal(t, []string{"*", "refs/pulls/open/*"}, cm.ordered[0].fetchRefs)
	})
}

// TestGenerateAdditionalCheckoutStepsWithFetch verifies that fetch steps are
// appended after additional checkout steps when fetch refs are configured.
func TestGenerateAdditionalCheckoutStepsWithFetch(t *testing.T) {
	getPin := func(action string) string { return action + "@v4" }

	t.Run("fetch step appended after checkout step", func(t *testing.T) {
		cm := NewCheckoutManager([]*CheckoutConfig{
			{Repository: "owner/side-repo", Path: "./side", Fetch: []string{"refs/pulls/open/*"}},
		})
		lines := cm.GenerateAdditionalCheckoutSteps(getPin)
		combined := strings.Join(lines, "")
		assert.Contains(t, combined, "repository: owner/side-repo", "should include checkout step")
		assert.Contains(t, combined, "Fetch additional refs for owner/side-repo", "should include fetch step")
		assert.Contains(t, combined, "+refs/pull/*/head:refs/remotes/origin/pull/*/head", "should include PR refspec")
	})

	t.Run("no fetch step when fetch not configured", func(t *testing.T) {
		cm := NewCheckoutManager([]*CheckoutConfig{
			{Repository: "owner/side-repo", Path: "./side"},
		})
		lines := cm.GenerateAdditionalCheckoutSteps(getPin)
		combined := strings.Join(lines, "")
		assert.NotContains(t, combined, "Fetch additional refs", "should not include fetch step without fetch config")
	})
}

// TestGenerateDefaultCheckoutStepWithFetch verifies that fetch steps are appended
// after the default checkout step when fetch refs are configured.
func TestGenerateDefaultCheckoutStepWithFetch(t *testing.T) {
	getPin := func(action string) string { return action + "@v4" }

	t.Run("fetch step appended after default checkout", func(t *testing.T) {
		cm := NewCheckoutManager([]*CheckoutConfig{
			{Fetch: []string{"*"}},
		})
		lines := cm.GenerateDefaultCheckoutStep(false, "", getPin)
		combined := strings.Join(lines, "")
		assert.Contains(t, combined, "Checkout repository", "should include default checkout step")
		assert.Contains(t, combined, "Fetch additional refs", "should include fetch step")
		assert.Contains(t, combined, "+refs/heads/*:refs/remotes/origin/*", "should include all-branches refspec")
	})

	t.Run("no fetch step when fetch not configured on default checkout", func(t *testing.T) {
		cm := NewCheckoutManager(nil)
		lines := cm.GenerateDefaultCheckoutStep(false, "", getPin)
		combined := strings.Join(lines, "")
		assert.NotContains(t, combined, "Fetch additional refs", "should not include fetch step without config")
	})

	t.Run("fetch with custom github-token uses that token", func(t *testing.T) {
		cm := NewCheckoutManager([]*CheckoutConfig{
			{GitHubToken: "${{ secrets.MY_PAT }}", Fetch: []string{"refs/pulls/open/*"}},
		})
		lines := cm.GenerateDefaultCheckoutStep(false, "", getPin)
		combined := strings.Join(lines, "")
		// Token should appear both in the checkout step and the fetch env var
		assert.Contains(t, combined, "${{ secrets.MY_PAT }}", "custom token should be in output")
		assert.Contains(t, combined, "+refs/pull/*/head:refs/remotes/origin/pull/*/head", "PR refspec should be present")
	})
}

func TestHasAppAuth(t *testing.T) {
	t.Run("returns false when no app configured", func(t *testing.T) {
		cm := NewCheckoutManager([]*CheckoutConfig{
			{GitHubToken: "${{ secrets.MY_PAT }}"},
		})
		assert.False(t, cm.HasAppAuth(), "should be false when no app is configured")
	})

	t.Run("returns false for nil configs", func(t *testing.T) {
		cm := NewCheckoutManager(nil)
		assert.False(t, cm.HasAppAuth(), "should be false for nil configs")
	})

	t.Run("returns true when default checkout has app", func(t *testing.T) {
		cm := NewCheckoutManager([]*CheckoutConfig{
			{GitHubApp: &GitHubAppConfig{AppID: "${{ vars.APP_ID }}", PrivateKey: "${{ secrets.KEY }}"}},
		})
		assert.True(t, cm.HasAppAuth(), "should be true when default checkout has app")
	})

	t.Run("returns true when additional checkout has app", func(t *testing.T) {
		cm := NewCheckoutManager([]*CheckoutConfig{
			{GitHubToken: "${{ secrets.MY_PAT }}"},
			{Repository: "other/repo", Path: "deps", GitHubApp: &GitHubAppConfig{AppID: "${{ vars.APP_ID }}", PrivateKey: "${{ secrets.KEY }}"}},
		})
		assert.True(t, cm.HasAppAuth(), "should be true when any checkout has app")
	})
}

func TestDefaultCheckoutWithAppAuth(t *testing.T) {
	getPin := func(ref string) string { return ref }

	t.Run("checkout step uses app token reference", func(t *testing.T) {
		cm := NewCheckoutManager([]*CheckoutConfig{
			{GitHubApp: &GitHubAppConfig{AppID: "${{ vars.APP_ID }}", PrivateKey: "${{ secrets.KEY }}"}},
		})
		lines := cm.GenerateDefaultCheckoutStep(false, "", getPin)
		combined := strings.Join(lines, "")
		// Token is now minted in the agent job itself (same-job step reference)
		assert.Contains(t, combined, "steps.checkout-app-token-0.outputs.token", "checkout should reference step output in same job")
		assert.NotContains(t, combined, "needs.activation.outputs.checkout_app_token_0", "checkout must not reference activation job outputs (masked values are dropped by runner)")
	})
}

func TestAdditionalCheckoutWithAppAuth(t *testing.T) {
	getPin := func(ref string) string { return ref }

	t.Run("additional checkout uses app token reference", func(t *testing.T) {
		cm := NewCheckoutManager([]*CheckoutConfig{
			{GitHubToken: "${{ secrets.MY_PAT }}"}, // default checkout
			{
				Repository: "other/repo",
				Path:       "deps",
				GitHubApp:  &GitHubAppConfig{AppID: "${{ vars.APP_ID }}", PrivateKey: "${{ secrets.KEY }}"},
			},
		})
		lines := cm.GenerateAdditionalCheckoutSteps(getPin)
		combined := strings.Join(lines, "")
		// Token is now minted in the agent job itself (same-job step reference)
		assert.Contains(t, combined, "steps.checkout-app-token-1.outputs.token", "additional checkout should reference step output at index 1")
		assert.NotContains(t, combined, "needs.activation.outputs.checkout_app_token_1", "checkout must not reference activation job outputs (masked values are dropped by runner)")
		assert.Contains(t, combined, "other/repo", "should reference the additional repo")
	})
}

// TestCrossRepoTargetRepo verifies the SetCrossRepoTargetRepo/GetCrossRepoTargetRepo lifecycle.
func TestCrossRepoTargetRepo(t *testing.T) {
	t.Run("default is empty string (same-repo)", func(t *testing.T) {
		cm := NewCheckoutManager(nil)
		assert.Empty(t, cm.GetCrossRepoTargetRepo(), "new checkout manager should have no cross-repo target")
	})

	t.Run("activation job expression is stored and retrievable", func(t *testing.T) {
		cm := NewCheckoutManager(nil)
		cm.SetCrossRepoTargetRepo("${{ steps.resolve-host-repo.outputs.target_repo }}")
		assert.Equal(t, "${{ steps.resolve-host-repo.outputs.target_repo }}", cm.GetCrossRepoTargetRepo())
	})

	t.Run("downstream job expression (needs.activation.outputs) is stored and retrievable", func(t *testing.T) {
		cm := NewCheckoutManager(nil)
		cm.SetCrossRepoTargetRepo("${{ needs.activation.outputs.target_repo }}")
		assert.Equal(t, "${{ needs.activation.outputs.target_repo }}", cm.GetCrossRepoTargetRepo())
	})

	t.Run("GenerateGitHubFolderCheckoutStep uses stored value", func(t *testing.T) {
		cm := NewCheckoutManager(nil)
		cm.SetCrossRepoTargetRepo("${{ needs.activation.outputs.target_repo }}")

		lines := cm.GenerateGitHubFolderCheckoutStep(cm.GetCrossRepoTargetRepo(), "", "", getActionPin)
		combined := strings.Join(lines, "")

		assert.Contains(t, combined, "repository: ${{ needs.activation.outputs.target_repo }}",
			"checkout step should use the cross-repo target")
	})
}

// TestCrossRepoTargetRef verifies the SetCrossRepoTargetRef/GetCrossRepoTargetRef lifecycle
// and that GenerateGitHubFolderCheckoutStep emits a ref: field when a ref is provided.
func TestCrossRepoTargetRef(t *testing.T) {
	t.Run("default is empty string", func(t *testing.T) {
		cm := NewCheckoutManager(nil)
		assert.Empty(t, cm.GetCrossRepoTargetRef(), "new checkout manager should have no cross-repo ref")
	})

	t.Run("activation job ref expression is stored and retrievable", func(t *testing.T) {
		cm := NewCheckoutManager(nil)
		cm.SetCrossRepoTargetRef("${{ steps.resolve-host-repo.outputs.target_ref }}")
		assert.Equal(t, "${{ steps.resolve-host-repo.outputs.target_ref }}", cm.GetCrossRepoTargetRef())
	})

	t.Run("downstream job ref expression (needs.activation.outputs) is stored and retrievable", func(t *testing.T) {
		cm := NewCheckoutManager(nil)
		cm.SetCrossRepoTargetRef("${{ needs.activation.outputs.target_ref }}")
		assert.Equal(t, "${{ needs.activation.outputs.target_ref }}", cm.GetCrossRepoTargetRef())
	})

	t.Run("GenerateGitHubFolderCheckoutStep emits ref: when ref is provided", func(t *testing.T) {
		cm := NewCheckoutManager(nil)
		cm.SetCrossRepoTargetRepo("${{ steps.resolve-host-repo.outputs.target_repo }}")
		cm.SetCrossRepoTargetRef("${{ steps.resolve-host-repo.outputs.target_ref }}")

		lines := cm.GenerateGitHubFolderCheckoutStep(cm.GetCrossRepoTargetRepo(), cm.GetCrossRepoTargetRef(), "", getActionPin)
		combined := strings.Join(lines, "")

		assert.Contains(t, combined, "repository: ${{ steps.resolve-host-repo.outputs.target_repo }}",
			"checkout step should include repository field")
		assert.Contains(t, combined, "ref: ${{ steps.resolve-host-repo.outputs.target_ref }}",
			"checkout step should include ref field")
	})

	t.Run("GenerateGitHubFolderCheckoutStep omits ref: when ref is empty", func(t *testing.T) {
		cm := NewCheckoutManager(nil)

		lines := cm.GenerateGitHubFolderCheckoutStep("org/repo", "", "", getActionPin)
		combined := strings.Join(lines, "")

		assert.NotContains(t, combined, "ref:", "checkout step should not include ref field when empty")
	})
}

// TestHasExternalRootCheckout verifies detection of external checkouts targeting the workspace root.
func TestHasExternalRootCheckout(t *testing.T) {
	t.Run("returns false for nil configs", func(t *testing.T) {
		cm := NewCheckoutManager(nil)
		assert.False(t, cm.HasExternalRootCheckout(), "should be false for nil configs")
	})

	t.Run("returns false for empty configs", func(t *testing.T) {
		cm := NewCheckoutManager([]*CheckoutConfig{})
		assert.False(t, cm.HasExternalRootCheckout(), "should be false for empty configs")
	})

	t.Run("returns false for default checkout only (no repository)", func(t *testing.T) {
		cm := NewCheckoutManager([]*CheckoutConfig{
			{GitHubToken: "${{ secrets.MY_PAT }}"},
		})
		assert.False(t, cm.HasExternalRootCheckout(), "should be false when only default checkout is configured")
	})

	t.Run("returns false for external checkout with non-root path", func(t *testing.T) {
		cm := NewCheckoutManager([]*CheckoutConfig{
			{Repository: "other/repo", Path: "libs/other"},
		})
		assert.False(t, cm.HasExternalRootCheckout(), "should be false when external repo uses a subdirectory path")
	})

	t.Run("returns true for external checkout without path (workspace root)", func(t *testing.T) {
		cm := NewCheckoutManager([]*CheckoutConfig{
			{Repository: "githubnext/gh-aw-side-repo", GitHubToken: "${{ secrets.SIDE_REPO_PAT }}"},
		})
		assert.True(t, cm.HasExternalRootCheckout(), "should be true when external repo checks out to workspace root")
	})

	t.Run("returns true for external checkout with explicit dot path", func(t *testing.T) {
		cm := NewCheckoutManager([]*CheckoutConfig{
			{Repository: "other/repo", Path: "."},
		})
		assert.True(t, cm.HasExternalRootCheckout(), "should be true when external repo uses '.' as path (workspace root)")
	})

	t.Run("returns true when one of multiple checkouts targets external root", func(t *testing.T) {
		cm := NewCheckoutManager([]*CheckoutConfig{
			{Repository: "other/repo", Path: "libs/other"},
			{Repository: "githubnext/gh-aw-side-repo"},
		})
		assert.True(t, cm.HasExternalRootCheckout(), "should be true when any checkout targets external root")
	})
}
