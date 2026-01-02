package workflow

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestValidateBranchPrefix tests the branch-prefix validation function
func TestValidateBranchPrefix(t *testing.T) {
	tests := []struct {
		name          string
		branchPrefix  string
		expectedError bool
		errorContains string
	}{
		{
			name:          "valid lowercase letters",
			branchPrefix:  "testmemory",
			expectedError: false,
		},
		{
			name:          "valid lowercase with numbers",
			branchPrefix:  "test123memory",
			expectedError: false,
		},
		{
			name:          "valid all numbers",
			branchPrefix:  "12345678",
			expectedError: false,
		},
		{
			name:          "valid minimum length (4 chars)",
			branchPrefix:  "test",
			expectedError: false,
		},
		{
			name:          "valid maximum length (64 chars)",
			branchPrefix:  "a123456789012345678901234567890123456789012345678901234567890123",
			expectedError: false,
		},
		{
			name:          "empty prefix is allowed (defaults to memory)",
			branchPrefix:  "",
			expectedError: false,
		},
		{
			name:          "too short (3 chars)",
			branchPrefix:  "abc",
			expectedError: true,
			errorContains: "must be between 4 and 64 characters",
		},
		{
			name:          "too long (65 chars)",
			branchPrefix:  "a1234567890123456789012345678901234567890123456789012345678901234",
			expectedError: true,
			errorContains: "must be between 4 and 64 characters",
		},
		{
			name:          "contains uppercase",
			branchPrefix:  "TestMemory",
			expectedError: true,
			errorContains: "must contain only lowercase letters and numbers",
		},
		{
			name:          "contains hyphen",
			branchPrefix:  "test-memory",
			expectedError: true,
			errorContains: "must contain only lowercase letters and numbers",
		},
		{
			name:          "contains underscore",
			branchPrefix:  "test_memory",
			expectedError: true,
			errorContains: "must contain only lowercase letters and numbers",
		},
		{
			name:          "contains space",
			branchPrefix:  "test memory",
			expectedError: true,
			errorContains: "must contain only lowercase letters and numbers",
		},
		{
			name:          "contains special characters",
			branchPrefix:  "test@memory",
			expectedError: true,
			errorContains: "must contain only lowercase letters and numbers",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateBranchPrefix(tt.branchPrefix)

			if tt.expectedError {
				require.Error(t, err, "Expected validation to fail")
				assert.Contains(t, err.Error(), tt.errorContains,
					"Error message should contain expected text")
			} else {
				assert.NoError(t, err, "Expected validation to pass")
			}
		})
	}
}

// TestGenerateDefaultBranchNameWithPrefix tests branch name generation with custom prefix
func TestGenerateDefaultBranchNameWithPrefix(t *testing.T) {
	tests := []struct {
		name               string
		memoryID           string
		branchPrefix       string
		expectedBranchName string
	}{
		{
			name:               "default prefix (empty string)",
			memoryID:           "default",
			branchPrefix:       "",
			expectedBranchName: "memory/default",
		},
		{
			name:               "custom prefix",
			memoryID:           "default",
			branchPrefix:       "foobar",
			expectedBranchName: "foobar/default",
		},
		{
			name:               "custom prefix with custom ID",
			memoryID:           "session",
			branchPrefix:       "myprefix",
			expectedBranchName: "myprefix/session",
		},
		{
			name:               "numeric prefix",
			memoryID:           "data",
			branchPrefix:       "1234",
			expectedBranchName: "1234/data",
		},
		{
			name:               "alphanumeric prefix",
			memoryID:           "cache",
			branchPrefix:       "test123",
			expectedBranchName: "test123/cache",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			branchName := generateDefaultBranchName(tt.memoryID, tt.branchPrefix)
			assert.Equal(t, tt.expectedBranchName, branchName,
				"Generated branch name should match expected pattern")
		})
	}
}

// TestRepoMemoryConfigWithBranchPrefix tests configuration parsing with branch-prefix
func TestRepoMemoryConfigWithBranchPrefix(t *testing.T) {
	tests := []struct {
		name               string
		toolsMap           map[string]any
		expectedBranchName string
		expectedError      bool
		errorContains      string
	}{
		{
			name: "object config with custom branch-prefix",
			toolsMap: map[string]any{
				"repo-memory": map[string]any{
					"branch-prefix": "foobar",
				},
			},
			expectedBranchName: "foobar/default",
			expectedError:      false,
		},
		{
			name: "object config with branch-prefix and explicit branch-name",
			toolsMap: map[string]any{
				"repo-memory": map[string]any{
					"branch-prefix": "testprefix",
					"branch-name":   "custom/branch",
				},
			},
			expectedBranchName: "custom/branch",
			expectedError:      false,
		},
		{
			name: "array config with custom branch-prefix",
			toolsMap: map[string]any{
				"repo-memory": []any{
					map[string]any{
						"id":            "session",
						"branch-prefix": "myprefix",
					},
				},
			},
			expectedBranchName: "myprefix/session",
			expectedError:      false,
		},
		{
			name: "array config with invalid branch-prefix (too short)",
			toolsMap: map[string]any{
				"repo-memory": []any{
					map[string]any{
						"id":            "session",
						"branch-prefix": "abc",
					},
				},
			},
			expectedError: true,
			errorContains: "must be between 4 and 64 characters",
		},
		{
			name: "array config with invalid branch-prefix (uppercase)",
			toolsMap: map[string]any{
				"repo-memory": []any{
					map[string]any{
						"id":            "session",
						"branch-prefix": "TestPrefix",
					},
				},
			},
			expectedError: true,
			errorContains: "must contain only lowercase letters and numbers",
		},
		{
			name: "object config with invalid branch-prefix (hyphen)",
			toolsMap: map[string]any{
				"repo-memory": map[string]any{
					"branch-prefix": "test-prefix",
				},
			},
			expectedError: true,
			errorContains: "must contain only lowercase letters and numbers",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			toolsConfig, err := ParseToolsConfig(tt.toolsMap)
			require.NoError(t, err, "Failed to parse tools config")

			compiler := NewCompiler(false, "", "test")
			config, err := compiler.extractRepoMemoryConfig(toolsConfig)

			if tt.expectedError {
				require.Error(t, err, "Expected extraction to fail")
				assert.Contains(t, err.Error(), tt.errorContains,
					"Error message should contain expected text")
			} else {
				require.NoError(t, err, "Failed to extract repo-memory config")
				require.NotNil(t, config, "Expected non-nil config")
				require.NotEmpty(t, config.Memories, "Expected at least one memory")
				assert.Equal(t, tt.expectedBranchName, config.Memories[0].BranchName,
					"Branch name should match expected value")
			}
		})
	}
}

// TestRepoMemoryPathConsistencyWithCustomPrefix tests path consistency with custom branch prefix
func TestRepoMemoryPathConsistencyWithCustomPrefix(t *testing.T) {
	tests := []struct {
		name                 string
		memoryID             string
		branchPrefix         string
		expectedBranchName   string
		expectedMemoryDir    string
		expectedPromptPath   string
		expectedArtifactName string
	}{
		{
			name:                 "custom prefix foobar",
			memoryID:             "default",
			branchPrefix:         "foobar",
			expectedBranchName:   "foobar/default",
			expectedMemoryDir:    "/tmp/gh-aw/repo-memory/default",
			expectedPromptPath:   "/tmp/gh-aw/repo-memory/default/",
			expectedArtifactName: "repo-memory-default",
		},
		{
			name:                 "custom prefix with custom ID",
			memoryID:             "session",
			branchPrefix:         "myprefix",
			expectedBranchName:   "myprefix/session",
			expectedMemoryDir:    "/tmp/gh-aw/repo-memory/session",
			expectedPromptPath:   "/tmp/gh-aw/repo-memory/session/",
			expectedArtifactName: "repo-memory-session",
		},
		{
			name:                 "numeric prefix",
			memoryID:             "data",
			branchPrefix:         "1234",
			expectedBranchName:   "1234/data",
			expectedMemoryDir:    "/tmp/gh-aw/repo-memory/data",
			expectedPromptPath:   "/tmp/gh-aw/repo-memory/data/",
			expectedArtifactName: "repo-memory-data",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &RepoMemoryConfig{
				Memories: []RepoMemoryEntry{
					{
						ID:           tt.memoryID,
						BranchPrefix: tt.branchPrefix,
						BranchName:   tt.expectedBranchName,
						MaxFileSize:  10240,
						MaxFileCount: 100,
						CreateOrphan: true,
					},
				},
			}

			data := &WorkflowData{
				RepoMemoryConfig: config,
			}

			// Test prompt path (with trailing slash)
			var promptBuilder strings.Builder
			generateRepoMemoryPromptSection(&promptBuilder, config)
			promptOutput := promptBuilder.String()

			assert.Contains(t, promptOutput, tt.expectedPromptPath,
				"Prompt should contain memory path with trailing slash")

			// Test artifact upload path (no trailing slash)
			var artifactUploadBuilder strings.Builder
			generateRepoMemoryArtifactUpload(&artifactUploadBuilder, data)
			artifactUploadOutput := artifactUploadBuilder.String()

			assert.Contains(t, artifactUploadOutput,
				"name: "+tt.expectedArtifactName,
				"Artifact upload should use correct artifact name")

			assert.Contains(t, artifactUploadOutput,
				"path: "+tt.expectedMemoryDir,
				"Artifact upload should use correct memory directory path (no trailing slash)")

			// Test clone steps contain correct branch name
			var cloneBuilder strings.Builder
			generateRepoMemorySteps(&cloneBuilder, data)
			cloneOutput := cloneBuilder.String()

			assert.Contains(t, cloneOutput,
				"BRANCH_NAME: "+tt.expectedBranchName,
				"Clone steps should reference correct branch name with custom prefix")
		})
	}
}
