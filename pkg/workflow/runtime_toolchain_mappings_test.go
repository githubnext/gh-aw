package workflow

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCollectToolchainMappings(t *testing.T) {
	tests := []struct {
		name                string
		requirements        []RuntimeRequirement
		expectedRuntimes    []string
		expectedEnvVarKeys  []string
		expectedMountCount  int
		hasGoCacheMount     bool
		hasNodeCacheMount   bool
		hasPythonCacheMount bool
	}{
		{
			name: "single Go runtime",
			requirements: []RuntimeRequirement{
				{
					Runtime: &Runtime{
						ID:   "go",
						Name: "Go",
					},
				},
			},
			expectedRuntimes:   []string{"go"},
			expectedEnvVarKeys: []string{"GOCACHE", "GOMODCACHE", "GOPATH"},
			expectedMountCount: 2, // GOPATH and GOCACHE
			hasGoCacheMount:    true,
		},
		{
			name: "multiple runtimes - Go and Node",
			requirements: []RuntimeRequirement{
				{
					Runtime: &Runtime{
						ID:   "go",
						Name: "Go",
					},
				},
				{
					Runtime: &Runtime{
						ID:   "node",
						Name: "Node.js",
					},
				},
			},
			expectedRuntimes:   []string{"go", "node"},
			expectedEnvVarKeys: []string{"GOCACHE", "GOMODCACHE", "GOPATH", "NPM_CONFIG_CACHE"},
			expectedMountCount: 3, // GOPATH, GOCACHE, NPM_CONFIG_CACHE
			hasGoCacheMount:    true,
			hasNodeCacheMount:  true,
		},
		{
			name: "Python runtime",
			requirements: []RuntimeRequirement{
				{
					Runtime: &Runtime{
						ID:   "python",
						Name: "Python",
					},
				},
			},
			expectedRuntimes:    []string{"python"},
			expectedEnvVarKeys:  []string{"PIP_CACHE_DIR"},
			expectedMountCount:  1,
			hasPythonCacheMount: true,
		},
		{
			name:               "no requirements",
			requirements:       []RuntimeRequirement{},
			expectedRuntimes:   []string{},
			expectedEnvVarKeys: nil, // nil for empty case
			expectedMountCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mappings := CollectToolchainMappings(tt.requirements)

			// Check runtime count
			assert.Len(t, mappings.Mappings, len(tt.expectedRuntimes), "Expected %d runtime mappings", len(tt.expectedRuntimes))

			// Check that expected runtimes are present
			for _, runtimeID := range tt.expectedRuntimes {
				_, exists := mappings.Mappings[runtimeID]
				assert.True(t, exists, "Expected runtime %s to be present", runtimeID)
			}

			// Check environment variables
			allEnvVars := mappings.GetAllEnvVars()
			if tt.expectedEnvVarKeys == nil {
				// For empty case, just check count
				assert.Empty(t, allEnvVars, "Expected no environment variables")
			} else {
				var envVarKeys []string
				for key := range allEnvVars {
					envVarKeys = append(envVarKeys, key)
				}
				sort.Strings(envVarKeys)
				sort.Strings(tt.expectedEnvVarKeys)
				assert.Equal(t, tt.expectedEnvVarKeys, envVarKeys, "Environment variable keys don't match")
			}

			// Check mount count
			allMounts := mappings.GetAllMounts()
			assert.Len(t, allMounts, tt.expectedMountCount, "Expected %d mounts", tt.expectedMountCount)

			// Check specific mounts if needed
			if tt.hasGoCacheMount {
				found := false
				for _, mount := range allMounts {
					if mount == "/home/runner/.cache/go-build:/home/runner/.cache/go-build:rw" {
						found = true
						break
					}
				}
				assert.True(t, found, "Expected Go cache mount to be present")
			}

			if tt.hasNodeCacheMount {
				found := false
				for _, mount := range allMounts {
					if mount == "/home/runner/.npm:/home/runner/.npm:rw" {
						found = true
						break
					}
				}
				assert.True(t, found, "Expected Node cache mount to be present")
			}

			if tt.hasPythonCacheMount {
				found := false
				for _, mount := range allMounts {
					if mount == "/home/runner/.cache/pip:/home/runner/.cache/pip:rw" {
						found = true
						break
					}
				}
				assert.True(t, found, "Expected Python cache mount to be present")
			}

			// Verify mounts are sorted (only check if we have mounts)
			if len(allMounts) > 0 {
				sortedMounts := make([]string, len(allMounts))
				copy(sortedMounts, allMounts)
				sort.Strings(sortedMounts)
				assert.Equal(t, sortedMounts, allMounts, "Mounts should be sorted")
			}
		})
	}
}

func TestMergeMountsWithDedup(t *testing.T) {
	tests := []struct {
		name          string
		existing      []string
		new           []string
		expected      []string
		expectedCount int
	}{
		{
			name:          "no duplicates",
			existing:      []string{"/path/a:/path/a:rw"},
			new:           []string{"/path/b:/path/b:rw"},
			expected:      []string{"/path/a:/path/a:rw", "/path/b:/path/b:rw"},
			expectedCount: 2,
		},
		{
			name:          "with duplicates",
			existing:      []string{"/path/a:/path/a:rw", "/path/b:/path/b:rw"},
			new:           []string{"/path/b:/path/b:rw", "/path/c:/path/c:rw"},
			expected:      []string{"/path/a:/path/a:rw", "/path/b:/path/b:rw", "/path/c:/path/c:rw"},
			expectedCount: 3,
		},
		{
			name:          "all same",
			existing:      []string{"/path/a:/path/a:rw"},
			new:           []string{"/path/a:/path/a:rw"},
			expected:      []string{"/path/a:/path/a:rw"},
			expectedCount: 1,
		},
		{
			name:          "empty existing",
			existing:      []string{},
			new:           []string{"/path/a:/path/a:rw", "/path/b:/path/b:rw"},
			expected:      []string{"/path/a:/path/a:rw", "/path/b:/path/b:rw"},
			expectedCount: 2,
		},
		{
			name:          "empty new",
			existing:      []string{"/path/a:/path/a:rw", "/path/b:/path/b:rw"},
			new:           []string{},
			expected:      []string{"/path/a:/path/a:rw", "/path/b:/path/b:rw"},
			expectedCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MergeMountsWithDedup(tt.existing, tt.new)

			// Check count
			assert.Len(t, result, tt.expectedCount, "Expected %d mounts after merge", tt.expectedCount)

			// Verify no duplicates
			seen := make(map[string]bool)
			for _, mount := range result {
				assert.False(t, seen[mount], "Found duplicate mount: %s", mount)
				seen[mount] = true
			}

			// Verify sorted
			sortedExpected := make([]string, len(tt.expected))
			copy(sortedExpected, tt.expected)
			sort.Strings(sortedExpected)
			assert.Equal(t, sortedExpected, result, "Result should be sorted and match expected")
		})
	}
}

func TestToolchainMappings_AddMapping(t *testing.T) {
	mappings := NewToolchainMappings()

	// Add first mapping
	envVars1 := map[string]string{"VAR1": "value1"}
	mounts1 := []string{"/mount1:/mount1:rw"}
	mappings.AddMapping("go", envVars1, mounts1)

	require.NotNil(t, mappings.Mappings["go"])
	assert.Equal(t, "go", mappings.Mappings["go"].RuntimeID)
	assert.Equal(t, "value1", mappings.Mappings["go"].EnvVars["VAR1"])
	assert.Contains(t, mappings.Mappings["go"].Mounts, "/mount1:/mount1:rw")

	// Add second mapping to same runtime (should merge)
	envVars2 := map[string]string{"VAR2": "value2"}
	mounts2 := []string{"/mount2:/mount2:rw"}
	mappings.AddMapping("go", envVars2, mounts2)

	assert.Equal(t, "value1", mappings.Mappings["go"].EnvVars["VAR1"])
	assert.Equal(t, "value2", mappings.Mappings["go"].EnvVars["VAR2"])
	assert.Contains(t, mappings.Mappings["go"].Mounts, "/mount1:/mount1:rw")
	assert.Contains(t, mappings.Mappings["go"].Mounts, "/mount2:/mount2:rw")
}

func TestToolchainMappings_GetAllEnvVars(t *testing.T) {
	mappings := NewToolchainMappings()

	mappings.AddMapping("go", map[string]string{"GOPATH": "/go"}, nil)
	mappings.AddMapping("node", map[string]string{"NPM_CONFIG_CACHE": "/npm"}, nil)

	allEnvVars := mappings.GetAllEnvVars()

	assert.Len(t, allEnvVars, 2)
	assert.Equal(t, "/go", allEnvVars["GOPATH"])
	assert.Equal(t, "/npm", allEnvVars["NPM_CONFIG_CACHE"])
}

func TestToolchainMappings_GetAllMounts(t *testing.T) {
	mappings := NewToolchainMappings()

	mappings.AddMapping("go", nil, []string{"/mount1:/mount1:rw", "/mount2:/mount2:rw"})
	mappings.AddMapping("node", nil, []string{"/mount2:/mount2:rw", "/mount3:/mount3:rw"})

	allMounts := mappings.GetAllMounts()

	// Should have 3 unique mounts
	assert.Len(t, allMounts, 3)

	// Should be sorted
	sortedMounts := make([]string, len(allMounts))
	copy(sortedMounts, allMounts)
	sort.Strings(sortedMounts)
	assert.Equal(t, sortedMounts, allMounts)

	// Should contain all three unique mounts
	mountSet := make(map[string]bool)
	for _, mount := range allMounts {
		mountSet[mount] = true
	}
	assert.True(t, mountSet["/mount1:/mount1:rw"])
	assert.True(t, mountSet["/mount2:/mount2:rw"])
	assert.True(t, mountSet["/mount3:/mount3:rw"])
}
