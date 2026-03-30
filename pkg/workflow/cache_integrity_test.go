//go:build !integration

package workflow

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestComputePolicyHash_NoPolicy verifies that workflows without a guard policy
// use the "nopolicy" sentinel hash.
func TestComputePolicyHash_NoPolicy(t *testing.T) {
	tests := []struct {
		name         string
		githubConfig *GitHubToolConfig
	}{
		{name: "nil config", githubConfig: nil},
		{name: "empty config", githubConfig: &GitHubToolConfig{}},
		{name: "config without min-integrity", githubConfig: &GitHubToolConfig{AllowedRepos: "all"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash := computePolicyHash(tt.githubConfig)
			assert.Equal(t, noPolicySentinel, hash, "Should return nopolicy sentinel when no guard policy is configured")
		})
	}
}

// TestComputePolicyHash_Deterministic verifies that the same policy always produces the same hash.
func TestComputePolicyHash_Deterministic(t *testing.T) {
	cfg := &GitHubToolConfig{
		MinIntegrity: GitHubIntegrityUnapproved,
		AllowedRepos: []any{"github/gh-aw"},
		BlockedUsers: []string{"attacker1"},
	}

	hash1 := computePolicyHash(cfg)
	hash2 := computePolicyHash(cfg)
	assert.Equal(t, hash1, hash2, "Same policy must always produce the same hash")
	assert.Len(t, hash1, 8, "Hash must be 8 characters long")
	assert.NotEqual(t, noPolicySentinel, hash1, "Hash with policy must not equal nopolicy sentinel")
}

// TestComputePolicyHash_FieldChanges verifies that changing any single policy field produces a different hash.
func TestComputePolicyHash_FieldChanges(t *testing.T) {
	base := &GitHubToolConfig{
		MinIntegrity: GitHubIntegrityUnapproved,
		AllowedRepos: []any{"github/gh-aw"},
		BlockedUsers: []string{},
	}
	baseHash := computePolicyHash(base)

	tests := []struct {
		name string
		cfg  *GitHubToolConfig
	}{
		{
			name: "change min-integrity",
			cfg: &GitHubToolConfig{
				MinIntegrity: GitHubIntegrityApproved,
				AllowedRepos: []any{"github/gh-aw"},
				BlockedUsers: []string{},
			},
		},
		{
			name: "change repos",
			cfg: &GitHubToolConfig{
				MinIntegrity: GitHubIntegrityUnapproved,
				AllowedRepos: []any{"github/other-repo"},
				BlockedUsers: []string{},
			},
		},
		{
			name: "add blocked user",
			cfg: &GitHubToolConfig{
				MinIntegrity: GitHubIntegrityUnapproved,
				AllowedRepos: []any{"github/gh-aw"},
				BlockedUsers: []string{"attacker1"},
			},
		},
		{
			name: "change repos to 'all'",
			cfg: &GitHubToolConfig{
				MinIntegrity: GitHubIntegrityUnapproved,
				AllowedRepos: "all",
				BlockedUsers: []string{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash := computePolicyHash(tt.cfg)
			assert.NotEqual(t, baseHash, hash, "Changing '%s' must produce a different hash", tt.name)
		})
	}
}

// TestComputePolicyHash_ListOrderIndependent verifies that list order does not affect the hash.
func TestComputePolicyHash_ListOrderIndependent(t *testing.T) {
	cfg1 := &GitHubToolConfig{
		MinIntegrity: GitHubIntegrityUnapproved,
		AllowedRepos: []any{"github/gh-aw-mcpg", "github/gh-aw"},
		BlockedUsers: []string{"bob", "alice"},
	}
	cfg2 := &GitHubToolConfig{
		MinIntegrity: GitHubIntegrityUnapproved,
		AllowedRepos: []any{"github/gh-aw", "github/gh-aw-mcpg"},
		BlockedUsers: []string{"alice", "bob"},
	}

	assert.Equal(t, computePolicyHash(cfg1), computePolicyHash(cfg2),
		"Different list ordering must produce the same hash")
}

// TestComputePolicyHash_DuplicatesDeduped verifies that duplicate entries in lists do not affect the hash.
func TestComputePolicyHash_DuplicatesDeduped(t *testing.T) {
	cfg1 := &GitHubToolConfig{
		MinIntegrity: GitHubIntegrityNone,
		AllowedRepos: []any{"github/gh-aw"},
		BlockedUsers: []string{"alice"},
	}
	cfg2 := &GitHubToolConfig{
		MinIntegrity: GitHubIntegrityNone,
		AllowedRepos: []any{"github/gh-aw", "github/gh-aw"},
		BlockedUsers: []string{"alice", "alice"},
	}

	assert.Equal(t, computePolicyHash(cfg1), computePolicyHash(cfg2),
		"Duplicate list entries must be deduplicated before hashing")
}

// TestComputePolicyHash_BlockedUsersExpr verifies that BlockedUsersExpr is included
// in the policy hash so that expression-based policies are correctly isolated.
func TestComputePolicyHash_BlockedUsersExpr(t *testing.T) {
	base := &GitHubToolConfig{
		MinIntegrity: GitHubIntegrityUnapproved,
		AllowedRepos: []any{"github/gh-aw"},
		BlockedUsers: []string{},
	}
	baseHash := computePolicyHash(base)

	// Switching to an expression-based blocked-users should produce a different hash
	cfgWithExpr := &GitHubToolConfig{
		MinIntegrity:     GitHubIntegrityUnapproved,
		AllowedRepos:     []any{"github/gh-aw"},
		BlockedUsersExpr: "${{ vars.BLOCKED_USERS }}",
	}
	assert.NotEqual(t, baseHash, computePolicyHash(cfgWithExpr),
		"Expression-based blocked-users must produce a different hash than an empty list")

	// Different expressions must produce different hashes
	cfgWithExpr2 := &GitHubToolConfig{
		MinIntegrity:     GitHubIntegrityUnapproved,
		AllowedRepos:     []any{"github/gh-aw"},
		BlockedUsersExpr: "${{ vars.OTHER_BLOCKED_USERS }}",
	}
	assert.NotEqual(t, computePolicyHash(cfgWithExpr), computePolicyHash(cfgWithExpr2),
		"Different expressions must produce different hashes")

	// Same expression must produce the same hash (deterministic)
	cfgWithExprCopy := &GitHubToolConfig{
		MinIntegrity:     GitHubIntegrityUnapproved,
		AllowedRepos:     []any{"github/gh-aw"},
		BlockedUsersExpr: "${{ vars.BLOCKED_USERS }}",
	}
	assert.Equal(t, computePolicyHash(cfgWithExpr), computePolicyHash(cfgWithExprCopy),
		"Same expression must produce the same hash")
}

// TestComputePolicyHash_CaseInsensitive verifies that user/repo names are lowercased before hashing.
func TestComputePolicyHash_CaseInsensitive(t *testing.T) {
	cfg1 := &GitHubToolConfig{
		MinIntegrity: GitHubIntegrityNone,
		AllowedRepos: []any{"GitHub/GH-AW"},
		BlockedUsers: []string{"Alice"},
	}
	cfg2 := &GitHubToolConfig{
		MinIntegrity: GitHubIntegrityNone,
		AllowedRepos: []any{"github/gh-aw"},
		BlockedUsers: []string{"alice"},
	}

	assert.Equal(t, computePolicyHash(cfg1), computePolicyHash(cfg2),
		"Policy hash must be case-insensitive for user and repo names")
}

// TestCanonicalReposScope verifies canonical forms for all repo scope types.
func TestCanonicalReposScope(t *testing.T) {
	tests := []struct {
		name     string
		repos    GitHubReposScope
		expected string
	}{
		{name: "nil", repos: nil, expected: ""},
		{name: "all string", repos: "all", expected: "all"},
		{name: "public string", repos: "public", expected: "public"},
		{name: "uppercase string", repos: "ALL", expected: "all"},
		{name: "single repo array", repos: []any{"github/gh-aw"}, expected: "github/gh-aw"},
		{name: "multi repo array sorted", repos: []any{"github/z-repo", "github/a-repo"}, expected: "github/a-repo\ngithub/z-repo"},
		{name: "multi repo array uppercase", repos: []any{"GitHub/GH-AW"}, expected: "github/gh-aw"},
		{name: "dedup array", repos: []any{"github/gh-aw", "github/gh-aw"}, expected: "github/gh-aw"},
		{name: "sorted string slice", repos: []string{"b", "a"}, expected: "a\nb"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := canonicalReposScope(tt.repos)
			assert.Equal(t, tt.expected, result, "Canonical scope form mismatch for %s", tt.name)
		})
	}
}

// TestCanonicalUserList verifies the canonical form for user lists.
func TestCanonicalUserList(t *testing.T) {
	tests := []struct {
		name     string
		users    []string
		expected string
	}{
		{name: "nil", users: nil, expected: ""},
		{name: "empty", users: []string{}, expected: ""},
		{name: "single user", users: []string{"alice"}, expected: "alice"},
		{name: "sorted", users: []string{"charlie", "alice", "bob"}, expected: "alice\nbob\ncharlie"},
		{name: "uppercase lowercased", users: []string{"ALICE", "Bob"}, expected: "alice\nbob"},
		{name: "deduplicated", users: []string{"alice", "alice", "bob"}, expected: "alice\nbob"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := canonicalUserList(tt.users)
			assert.Equal(t, tt.expected, result, "Canonical user list mismatch for %s", tt.name)
		})
	}
}

// TestGenerateIntegrityAwareCacheKey verifies the new cache key format.
func TestGenerateIntegrityAwareCacheKey(t *testing.T) {
	tests := []struct {
		name           string
		cacheID        string
		integrityLevel string
		policyHash     string
		expected       string
	}{
		{
			name:           "default cache with policy",
			cacheID:        "default",
			integrityLevel: "unapproved",
			policyHash:     "7e4d9f12",
			expected:       "memory-unapproved-7e4d9f12-${{ env.GH_AW_WORKFLOW_ID_SANITIZED }}-${{ github.run_id }}",
		},
		{
			name:           "default cache without policy (sentinel)",
			cacheID:        "default",
			integrityLevel: "none",
			policyHash:     "nopolicy",
			expected:       "memory-none-nopolicy-${{ env.GH_AW_WORKFLOW_ID_SANITIZED }}-${{ github.run_id }}",
		},
		{
			name:           "named cache with policy",
			cacheID:        "session",
			integrityLevel: "merged",
			policyHash:     "abcd1234",
			expected:       "memory-merged-abcd1234-session-${{ env.GH_AW_WORKFLOW_ID_SANITIZED }}-${{ github.run_id }}",
		},
		{
			name:           "empty cache ID treated as default",
			cacheID:        "",
			integrityLevel: "none",
			policyHash:     "nopolicy",
			expected:       "memory-none-nopolicy-${{ env.GH_AW_WORKFLOW_ID_SANITIZED }}-${{ github.run_id }}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generateIntegrityAwareCacheKey(tt.cacheID, tt.integrityLevel, tt.policyHash)
			assert.Equal(t, tt.expected, result, "Cache key format mismatch")
		})
	}
}

// TestCacheIntegrityLevel verifies integrity level extraction from config.
func TestCacheIntegrityLevel(t *testing.T) {
	tests := []struct {
		name     string
		cfg      *GitHubToolConfig
		expected string
	}{
		{name: "nil config", cfg: nil, expected: defaultCacheIntegrityLevel},
		{name: "empty config", cfg: &GitHubToolConfig{}, expected: defaultCacheIntegrityLevel},
		{name: "merged integrity", cfg: &GitHubToolConfig{MinIntegrity: GitHubIntegrityMerged}, expected: "merged"},
		{name: "approved integrity", cfg: &GitHubToolConfig{MinIntegrity: GitHubIntegrityApproved}, expected: "approved"},
		{name: "unapproved integrity", cfg: &GitHubToolConfig{MinIntegrity: GitHubIntegrityUnapproved}, expected: "unapproved"},
		{name: "none integrity", cfg: &GitHubToolConfig{MinIntegrity: GitHubIntegrityNone}, expected: "none"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cacheIntegrityLevel(tt.cfg)
			assert.Equal(t, tt.expected, result, "Integrity level mismatch for %s", tt.name)
		})
	}
}

// TestHigherIntegrityLevels verifies the merge-down logic helper.
func TestHigherIntegrityLevels(t *testing.T) {
	tests := []struct {
		name     string
		level    string
		expected []string
	}{
		{name: "merged (highest) has no higher", level: "merged", expected: nil},
		{name: "approved has merged above", level: "approved", expected: []string{"merged"}},
		{name: "unapproved has merged+approved above", level: "unapproved", expected: []string{"merged", "approved"}},
		{name: "none (lowest) has all above", level: "none", expected: []string{"merged", "approved", "unapproved"}},
		{name: "unknown level has all levels above (no match = all)", level: "unknown", expected: []string{"merged", "approved", "unapproved", "none"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := higherIntegrityLevels(tt.level)
			assert.Equal(t, tt.expected, result, "Higher integrity levels mismatch for '%s'", tt.level)
		})
	}
}

// TestComputeIntegrityCacheKey_WithGitHubConfig verifies that computeIntegrityCacheKey
// produces the correct key when a GitHub guard policy is configured.
func TestComputeIntegrityCacheKey_WithGitHubConfig(t *testing.T) {
	cfg := &GitHubToolConfig{
		MinIntegrity: GitHubIntegrityUnapproved,
		AllowedRepos: []any{"github/gh-aw"},
	}
	policyHash := computePolicyHash(cfg)
	require.Len(t, policyHash, 8, "Policy hash must be 8 characters")

	entry := CacheMemoryEntry{ID: "default"}
	key := computeIntegrityCacheKey(entry, cfg)

	expectedPrefix := "memory-unapproved-" + policyHash + "-"
	assert.True(t, strings.HasPrefix(key, expectedPrefix),
		"Cache key should start with 'memory-unapproved-{hash}-', got: %s", key)
	assert.True(t, strings.HasSuffix(key, "-${{ github.run_id }}"),
		"Cache key should end with run_id suffix, got: %s", key)
}

// TestComputeIntegrityCacheKey_NoPolicy verifies that computeIntegrityCacheKey uses
// the nopolicy sentinel for workflows without a guard policy.
func TestComputeIntegrityCacheKey_NoPolicy(t *testing.T) {
	entry := CacheMemoryEntry{ID: "default"}
	key := computeIntegrityCacheKey(entry, nil)

	assert.True(t, strings.HasPrefix(key, "memory-none-nopolicy-"),
		"Cache key without policy should start with 'memory-none-nopolicy-', got: %s", key)
}

// TestComputeIntegrityCacheKey_CustomKey verifies that custom keys get the integrity prefix
// to prevent cross-integrity cache sharing.
func TestComputeIntegrityCacheKey_CustomKey(t *testing.T) {
	cfg := &GitHubToolConfig{
		MinIntegrity: GitHubIntegrityMerged,
		AllowedRepos: "all",
	}
	policyHash := computePolicyHash(cfg)

	entry := CacheMemoryEntry{
		ID:  "default",
		Key: "my-custom-key",
	}
	key := computeIntegrityCacheKey(entry, cfg)

	// Custom keys must be prefixed with integrity/policy to prevent cross-level sharing
	expectedPrefix := "memory-merged-" + policyHash + "-"
	assert.True(t, strings.HasPrefix(key, expectedPrefix),
		"Custom keys must be prefixed with integrity/policy, got: %s", key)
	assert.True(t, strings.HasSuffix(key, "-${{ github.run_id }}"),
		"Custom keys should end with run_id suffix, got: %s", key)
}

// TestComputeIntegrityCacheKey_CustomKeyWithRunID verifies that custom keys already containing
// the run_id suffix are not duplicated, but still get the integrity prefix.
func TestComputeIntegrityCacheKey_CustomKeyWithRunID(t *testing.T) {
	entry := CacheMemoryEntry{
		ID:  "default",
		Key: "my-custom-key-${{ github.run_id }}",
	}
	key := computeIntegrityCacheKey(entry, nil)

	// Should have none-nopolicy prefix + custom key (with single run_id)
	assert.True(t, strings.HasPrefix(key, "memory-none-nopolicy-"),
		"Custom keys must be prefixed even without a guard policy, got: %s", key)
	assert.Equal(t, 1, strings.Count(key, "${{ github.run_id }}"),
		"run_id suffix should appear exactly once, got: %s", key)
}

// TestCacheMemoryStepsIncludeGitSetup verifies that generated workflow YAML includes
// the git setup step after the cache restore step.
func TestCacheMemoryStepsIncludeGitSetup(t *testing.T) {
	toolsMap := map[string]any{
		"cache-memory": true,
		"github": map[string]any{
			"allowed":       []any{"get_repository"},
			"min-integrity": "unapproved",
			"allowed-repos": []any{"github/gh-aw"},
		},
	}

	toolsConfig, err := ParseToolsConfig(toolsMap)
	require.NoError(t, err, "Should parse tools config")

	compiler := NewCompiler()
	cacheMemoryConfig, err := compiler.extractCacheMemoryConfig(toolsConfig)
	require.NoError(t, err, "Should extract cache-memory config")

	parsedTools := NewTools(toolsMap)

	data := &WorkflowData{
		CacheMemoryConfig: cacheMemoryConfig,
		ParsedTools:       parsedTools,
	}

	var builder strings.Builder
	generateCacheMemorySteps(&builder, data)
	output := builder.String()

	assert.Contains(t, output, "Set up cache-memory git repository",
		"Should include git setup step")
	assert.Contains(t, output, "setup_cache_memory_git.sh",
		"Should reference the git setup script")
	assert.Contains(t, output, "GH_AW_MIN_INTEGRITY: unapproved",
		"Should set the integrity level env var")
	assert.Contains(t, output, "GH_AW_CACHE_DIR: /tmp/gh-aw/cache-memory",
		"Should set the cache dir env var")
}

// TestCacheMemoryStepsIntegrityAwareKey verifies that the generated cache key
// includes the integrity level and policy hash.
func TestCacheMemoryStepsIntegrityAwareKey(t *testing.T) {
	toolsMap := map[string]any{
		"cache-memory": true,
		"github": map[string]any{
			"allowed":       []any{"get_repository"},
			"min-integrity": "unapproved",
			"allowed-repos": []any{"github/gh-aw"},
		},
	}

	toolsConfig, err := ParseToolsConfig(toolsMap)
	require.NoError(t, err, "Should parse tools config")

	compiler := NewCompiler()
	cacheMemoryConfig, err := compiler.extractCacheMemoryConfig(toolsConfig)
	require.NoError(t, err, "Should extract cache-memory config")

	parsedTools := NewTools(toolsMap)

	data := &WorkflowData{
		CacheMemoryConfig: cacheMemoryConfig,
		ParsedTools:       parsedTools,
	}

	var builder strings.Builder
	generateCacheMemorySteps(&builder, data)
	output := builder.String()

	// Key should start with "memory-unapproved-" followed by an 8-char hash
	assert.Contains(t, output, "key: memory-unapproved-",
		"Cache key should include 'unapproved' integrity level")
	// Should NOT contain the old format (without integrity prefix)
	assert.NotContains(t, output, "key: memory-${{ env.GH_AW_WORKFLOW_ID_SANITIZED }}",
		"Cache key should not use the old format without integrity prefix")
}

// TestCacheMemoryStepsNoPolicy verifies that the generated cache key uses the
// nopolicy sentinel when no guard policy is configured.
func TestCacheMemoryStepsNoPolicy(t *testing.T) {
	toolsMap := map[string]any{
		"cache-memory": true,
		"github": map[string]any{
			"allowed": []any{"get_repository"},
		},
	}

	toolsConfig, err := ParseToolsConfig(toolsMap)
	require.NoError(t, err, "Should parse tools config")

	compiler := NewCompiler()
	cacheMemoryConfig, err := compiler.extractCacheMemoryConfig(toolsConfig)
	require.NoError(t, err, "Should extract cache-memory config")

	parsedTools := NewTools(toolsMap)

	data := &WorkflowData{
		CacheMemoryConfig: cacheMemoryConfig,
		ParsedTools:       parsedTools,
	}

	var builder strings.Builder
	generateCacheMemorySteps(&builder, data)
	output := builder.String()

	assert.Contains(t, output, "key: memory-none-nopolicy-",
		"Cache key without policy should use none-nopolicy prefix")
}

// TestCacheMemoryGitCommitSteps verifies that the post-agent git commit step is generated.
func TestCacheMemoryGitCommitSteps(t *testing.T) {
	toolsMap := map[string]any{
		"cache-memory": true,
	}

	toolsConfig, err := ParseToolsConfig(toolsMap)
	require.NoError(t, err, "Should parse tools config")

	compiler := NewCompiler()
	cacheMemoryConfig, err := compiler.extractCacheMemoryConfig(toolsConfig)
	require.NoError(t, err, "Should extract cache-memory config")

	data := &WorkflowData{
		CacheMemoryConfig: cacheMemoryConfig,
	}

	var builder strings.Builder
	generateCacheMemoryGitCommitSteps(&builder, data)
	output := builder.String()

	assert.Contains(t, output, "Commit cache-memory changes",
		"Should include git commit step")
	assert.Contains(t, output, "commit_cache_memory_git.sh",
		"Should reference the git commit script")
	assert.Contains(t, output, "if: always()",
		"Git commit step should always run")
	assert.Contains(t, output, "GH_AW_CACHE_DIR: /tmp/gh-aw/cache-memory",
		"Should set the cache dir env var")
}

// TestCacheMemoryGitCommitSteps_RestoreOnlySkipped verifies that restore-only caches
// do not get a git commit step.
func TestCacheMemoryGitCommitSteps_RestoreOnlySkipped(t *testing.T) {
	data := &WorkflowData{
		CacheMemoryConfig: &CacheMemoryConfig{
			Caches: []CacheMemoryEntry{
				{ID: "default", RestoreOnly: true},
			},
		},
	}

	var builder strings.Builder
	generateCacheMemoryGitCommitSteps(&builder, data)
	output := builder.String()

	assert.Empty(t, output, "Restore-only caches should not generate a git commit step")
}
