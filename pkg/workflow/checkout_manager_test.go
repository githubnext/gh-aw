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

	t.Run("custom token on default checkout", func(t *testing.T) {
		cm := NewCheckoutManager([]*CheckoutConfig{
			{GitHubToken: "${{ secrets.MY_TOKEN }}"},
		})
		override := cm.GetDefaultCheckoutOverride()
		require.NotNil(t, override, "should have default override")
		assert.Equal(t, "${{ secrets.MY_TOKEN }}", override.token, "token should be set")
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

	t.Run("user token is included in default checkout", func(t *testing.T) {
		cm := NewCheckoutManager([]*CheckoutConfig{
			{GitHubToken: "${{ secrets.MY_TOKEN }}"},
		})
		lines := cm.GenerateDefaultCheckoutStep(false, "", getPin)
		combined := strings.Join(lines, "")
		assert.Contains(t, combined, "token: ${{ secrets.MY_TOKEN }}", "should include custom token")
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
}

// TestParseCheckoutConfigs verifies parsing of raw frontmatter values.
func TestParseCheckoutConfigs(t *testing.T) {
	t.Run("nil returns nil", func(t *testing.T) {
		configs, err := ParseCheckoutConfigs(nil)
		require.NoError(t, err, "nil should not error")
		assert.Nil(t, configs, "nil input should return nil configs")
	})

	t.Run("single object", func(t *testing.T) {
		raw := map[string]any{
			"fetch-depth":  float64(0),
			"github-token": "${{ secrets.MY_TOKEN }}",
		}
		configs, err := ParseCheckoutConfigs(raw)
		require.NoError(t, err, "single object should parse without error")
		require.Len(t, configs, 1, "should produce one config")
		assert.Equal(t, "${{ secrets.MY_TOKEN }}", configs[0].GitHubToken, "token should be set")
		require.NotNil(t, configs[0].FetchDepth, "fetch-depth should be set")
		assert.Equal(t, 0, *configs[0].FetchDepth, "fetch-depth should be 0")
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

// TestGetCurrentRepository verifies CheckoutManager.GetCurrentRepository behavior.
func TestGetCurrentRepository(t *testing.T) {
	t.Run("returns empty string when no current checkout", func(t *testing.T) {
		cm := NewCheckoutManager([]*CheckoutConfig{
			{Repository: "owner/repo", Path: "./libs"},
		})
		assert.Empty(t, cm.GetCurrentRepository(), "should return empty string without current flag")
	})

	t.Run("returns repository when current: true is set", func(t *testing.T) {
		cm := NewCheckoutManager([]*CheckoutConfig{
			{Repository: "owner/target-repo", Path: "./target", Current: true},
		})
		assert.Equal(t, "owner/target-repo", cm.GetCurrentRepository(), "should return current checkout repository")
	})

	t.Run("returns empty string when current: true but no repository", func(t *testing.T) {
		cm := NewCheckoutManager([]*CheckoutConfig{
			{Path: ".", Current: true},
		})
		assert.Empty(t, cm.GetCurrentRepository(), "should return empty string when repository is not set")
	})

	t.Run("returns repository from current in multiple checkouts", func(t *testing.T) {
		cm := NewCheckoutManager([]*CheckoutConfig{
			{Path: "."},
			{Repository: "owner/central", Path: "./central"},
			{Repository: "owner/target", Path: "./target", Current: true},
		})
		assert.Equal(t, "owner/target", cm.GetCurrentRepository(), "should return the current checkout repository")
	})
}

// TestGetCurrentCheckoutRepository verifies the standalone helper function.
func TestGetCurrentCheckoutRepository(t *testing.T) {
	t.Run("nil slice returns empty string", func(t *testing.T) {
		assert.Empty(t, getCurrentCheckoutRepository(nil), "nil slice should return empty string")
	})

	t.Run("no current flag returns empty string", func(t *testing.T) {
		configs := []*CheckoutConfig{
			{Repository: "owner/repo"},
		}
		assert.Empty(t, getCurrentCheckoutRepository(configs), "no current flag should return empty string")
	})

	t.Run("current: true returns repository", func(t *testing.T) {
		configs := []*CheckoutConfig{
			{Repository: "owner/other"},
			{Repository: "owner/target", Current: true},
		}
		assert.Equal(t, "owner/target", getCurrentCheckoutRepository(configs), "should return current checkout repository")
	})

	t.Run("current: true with no repository returns empty string", func(t *testing.T) {
		configs := []*CheckoutConfig{
			{Current: true},
		}
		assert.Empty(t, getCurrentCheckoutRepository(configs), "current without repository should return empty string")
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
}
