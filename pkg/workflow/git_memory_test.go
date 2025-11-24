package workflow

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGenerateDefaultBranchName tests default branch name generation
func TestGenerateDefaultBranchName(t *testing.T) {
	tests := []struct {
		name     string
		branchID string
		expected string
	}{
		{
			name:     "default branch ID",
			branchID: "default",
			expected: "memory/default",
		},
		{
			name:     "custom branch ID",
			branchID: "session",
			expected: "memory/session",
		},
		{
			name:     "hyphenated branch ID",
			branchID: "my-state",
			expected: "memory/my-state",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generateDefaultBranchName(tt.branchID)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestValidateBranchName tests branch name validation
func TestValidateBranchName(t *testing.T) {
	tests := []struct {
		name       string
		branchName string
		expectErr  bool
	}{
		{
			name:       "valid branch name with memory prefix",
			branchName: "memory/default",
			expectErr:  false,
		},
		{
			name:       "valid branch name with nested path",
			branchName: "memory/session/state",
			expectErr:  false,
		},
		{
			name:       "invalid branch name without memory prefix",
			branchName: "main",
			expectErr:  true,
		},
		{
			name:       "invalid branch name with wrong prefix",
			branchName: "feature/memory",
			expectErr:  true,
		},
		{
			name:       "empty branch name",
			branchName: "",
			expectErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateBranchName(tt.branchName)
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestExtractGitMemoryConfigBoolean tests boolean configuration extraction
func TestExtractGitMemoryConfigBoolean(t *testing.T) {
	compiler := NewCompiler(false, "", "test")

	tests := []struct {
		name          string
		toolsConfig   *ToolsConfig
		expectedEmpty bool
		expectedLen   int
		expectedID    string
		expectedBranch string
	}{
		{
			name: "nil tools config",
			toolsConfig: &ToolsConfig{
				GitMemory: nil,
			},
			expectedEmpty: true,
		},
		{
			name: "boolean true",
			toolsConfig: &ToolsConfig{
				GitMemory: &GitMemoryToolConfig{Raw: true},
			},
			expectedLen:    1,
			expectedID:     "default",
			expectedBranch: "memory/default",
		},
		{
			name: "boolean false",
			toolsConfig: &ToolsConfig{
				GitMemory: &GitMemoryToolConfig{Raw: false},
			},
			expectedLen: 0,
		},
		{
			name: "nil raw value",
			toolsConfig: &ToolsConfig{
				GitMemory: &GitMemoryToolConfig{Raw: nil},
			},
			expectedLen:    1,
			expectedID:     "default",
			expectedBranch: "memory/default",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, err := compiler.extractGitMemoryConfig(tt.toolsConfig)
			require.NoError(t, err)

			if tt.expectedEmpty {
				assert.Nil(t, config)
				return
			}

			require.NotNil(t, config)
			assert.Len(t, config.Branches, tt.expectedLen)

			if tt.expectedLen > 0 {
				assert.Equal(t, tt.expectedID, config.Branches[0].ID)
				assert.Equal(t, tt.expectedBranch, config.Branches[0].Branch)
			}
		})
	}
}

// TestExtractGitMemoryConfigObject tests object configuration extraction
func TestExtractGitMemoryConfigObject(t *testing.T) {
	compiler := NewCompiler(false, "", "test")

	tests := []struct {
		name           string
		configMap      map[string]any
		expectedLen    int
		expectedID     string
		expectedBranch string
		expectedDesc   string
		expectErr      bool
	}{
		{
			name: "object with custom branch",
			configMap: map[string]any{
				"branch": "memory/session",
			},
			expectedLen:    1,
			expectedID:     "default",
			expectedBranch: "memory/session",
		},
		{
			name: "object with branch and description",
			configMap: map[string]any{
				"branch":      "memory/audit",
				"description": "Audit workflow state",
			},
			expectedLen:    1,
			expectedID:     "default",
			expectedBranch: "memory/audit",
			expectedDesc:   "Audit workflow state",
		},
		{
			name: "object with default branch",
			configMap: map[string]any{
				"description": "Default memory",
			},
			expectedLen:    1,
			expectedID:     "default",
			expectedBranch: "memory/default",
			expectedDesc:   "Default memory",
		},
		{
			name: "object with invalid branch name",
			configMap: map[string]any{
				"branch": "main",
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			toolsConfig := &ToolsConfig{
				GitMemory: &GitMemoryToolConfig{Raw: tt.configMap},
			}

			config, err := compiler.extractGitMemoryConfig(toolsConfig)

			if tt.expectErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, config)
			assert.Len(t, config.Branches, tt.expectedLen)

			if tt.expectedLen > 0 {
				branch := config.Branches[0]
				assert.Equal(t, tt.expectedID, branch.ID)
				assert.Equal(t, tt.expectedBranch, branch.Branch)
				assert.Equal(t, tt.expectedDesc, branch.Description)
			}
		})
	}
}

// TestExtractGitMemoryConfigArray tests array configuration extraction
func TestExtractGitMemoryConfigArray(t *testing.T) {
	compiler := NewCompiler(false, "", "test")

	tests := []struct {
		name        string
		configArray []any
		expectedLen int
		validate    func(t *testing.T, config *GitMemoryConfig)
		expectErr   bool
	}{
		{
			name: "array with multiple branches",
			configArray: []any{
				map[string]any{
					"id":     "default",
					"branch": "memory/default",
				},
				map[string]any{
					"id":     "session",
					"branch": "memory/session",
				},
			},
			expectedLen: 2,
			validate: func(t *testing.T, config *GitMemoryConfig) {
				assert.Equal(t, "default", config.Branches[0].ID)
				assert.Equal(t, "memory/default", config.Branches[0].Branch)
				assert.Equal(t, "session", config.Branches[1].ID)
				assert.Equal(t, "memory/session", config.Branches[1].Branch)
			},
		},
		{
			name: "array with descriptions",
			configArray: []any{
				map[string]any{
					"id":          "audit",
					"branch":      "memory/audit",
					"description": "Audit state",
				},
				map[string]any{
					"id":          "logs",
					"branch":      "memory/logs",
					"description": "Log data",
				},
			},
			expectedLen: 2,
			validate: func(t *testing.T, config *GitMemoryConfig) {
				assert.Equal(t, "audit", config.Branches[0].ID)
				assert.Equal(t, "Audit state", config.Branches[0].Description)
				assert.Equal(t, "logs", config.Branches[1].ID)
				assert.Equal(t, "Log data", config.Branches[1].Description)
			},
		},
		{
			name: "array with missing IDs defaults to 'default'",
			configArray: []any{
				map[string]any{
					"branch": "memory/custom",
				},
			},
			expectedLen: 1,
			validate: func(t *testing.T, config *GitMemoryConfig) {
				assert.Equal(t, "default", config.Branches[0].ID)
				assert.Equal(t, "memory/custom", config.Branches[0].Branch)
			},
		},
		{
			name: "array with duplicate IDs",
			configArray: []any{
				map[string]any{
					"id":     "default",
					"branch": "memory/default",
				},
				map[string]any{
					"id":     "default",
					"branch": "memory/other",
				},
			},
			expectErr: true,
		},
		{
			name: "array with invalid branch name",
			configArray: []any{
				map[string]any{
					"id":     "test",
					"branch": "main",
				},
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			toolsConfig := &ToolsConfig{
				GitMemory: &GitMemoryToolConfig{Raw: tt.configArray},
			}

			config, err := compiler.extractGitMemoryConfig(toolsConfig)

			if tt.expectErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, config)
			assert.Len(t, config.Branches, tt.expectedLen)

			if tt.validate != nil {
				tt.validate(t, config)
			}
		})
	}
}

// TestValidateNoDuplicateGitMemoryIDs tests duplicate ID validation
func TestValidateNoDuplicateGitMemoryIDs(t *testing.T) {
	tests := []struct {
		name      string
		branches  []GitMemoryEntry
		expectErr bool
	}{
		{
			name: "no duplicates",
			branches: []GitMemoryEntry{
				{ID: "default", Branch: "memory/default"},
				{ID: "session", Branch: "memory/session"},
			},
			expectErr: false,
		},
		{
			name: "duplicate IDs",
			branches: []GitMemoryEntry{
				{ID: "default", Branch: "memory/default"},
				{ID: "default", Branch: "memory/other"},
			},
			expectErr: true,
		},
		{
			name:      "empty array",
			branches:  []GitMemoryEntry{},
			expectErr: false,
		},
		{
			name: "single branch",
			branches: []GitMemoryEntry{
				{ID: "default", Branch: "memory/default"},
			},
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateNoDuplicateGitMemoryIDs(tt.branches)
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
