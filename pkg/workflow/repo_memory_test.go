//go:build !integration

package workflow

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRepoMemoryConfigDefault tests basic repo-memory configuration with boolean true
func TestRepoMemoryConfigDefault(t *testing.T) {
	toolsMap := map[string]any{
		"repo-memory": true,
	}

	toolsConfig, err := ParseToolsConfig(toolsMap)
	if err != nil {
		t.Fatalf("Failed to parse tools config: %v", err)
	}

	compiler := NewCompiler()
	config, err := compiler.extractRepoMemoryConfig(toolsConfig, "my-workflow")
	if err != nil {
		t.Fatalf("Failed to extract repo-memory config: %v", err)
	}

	if config == nil {
		t.Fatal("Expected non-nil config")
	}

	if len(config.Memories) != 1 {
		t.Fatalf("Expected 1 memory, got %d", len(config.Memories))
	}

	memory := config.Memories[0]
	if memory.ID != "default" {
		t.Errorf("Expected ID 'default', got '%s'", memory.ID)
	}

	if memory.BranchName != "memory/my-workflow" {
		t.Errorf("Expected branch name 'memory/my-workflow', got '%s'", memory.BranchName)
	}

	if memory.MaxFileSize != 10240 {
		t.Errorf("Expected max file size 10240, got %d", memory.MaxFileSize)
	}

	if memory.MaxFileCount != 100 {
		t.Errorf("Expected max file count 100, got %d", memory.MaxFileCount)
	}

	if !memory.CreateOrphan {
		t.Error("Expected create-orphan to be true by default")
	}
}

// TestRepoMemoryConfigObject tests repo-memory configuration with object notation
func TestRepoMemoryConfigObject(t *testing.T) {
	toolsMap := map[string]any{
		"repo-memory": map[string]any{
			"target-repo":   "myorg/myrepo",
			"branch-name":   "memory/custom",
			"max-file-size": 524288,
			"description":   "Custom memory store",
		},
	}

	toolsConfig, err := ParseToolsConfig(toolsMap)
	if err != nil {
		t.Fatalf("Failed to parse tools config: %v", err)
	}

	compiler := NewCompiler()
	config, err := compiler.extractRepoMemoryConfig(toolsConfig, "")
	if err != nil {
		t.Fatalf("Failed to extract repo-memory config: %v", err)
	}

	if config == nil {
		t.Fatal("Expected non-nil config")
	}

	if len(config.Memories) != 1 {
		t.Fatalf("Expected 1 memory, got %d", len(config.Memories))
	}

	memory := config.Memories[0]
	if memory.TargetRepo != "myorg/myrepo" {
		t.Errorf("Expected target-repo 'myorg/myrepo', got '%s'", memory.TargetRepo)
	}

	if memory.BranchName != "memory/custom" {
		t.Errorf("Expected branch name 'memory/custom', got '%s'", memory.BranchName)
	}

	if memory.MaxFileSize != 524288 {
		t.Errorf("Expected max file size 524288, got %d", memory.MaxFileSize)
	}

	if memory.Description != "Custom memory store" {
		t.Errorf("Expected description 'Custom memory store', got '%s'", memory.Description)
	}
}

// TestRepoMemoryConfigArray tests repo-memory configuration with array notation
func TestRepoMemoryConfigArray(t *testing.T) {
	toolsMap := map[string]any{
		"repo-memory": []any{
			map[string]any{
				"id":          "session",
				"branch-name": "memory/session",
			},
			map[string]any{
				"id":            "logs",
				"branch-name":   "memory/logs",
				"max-file-size": 2097152,
			},
		},
	}

	toolsConfig, err := ParseToolsConfig(toolsMap)
	if err != nil {
		t.Fatalf("Failed to parse tools config: %v", err)
	}

	compiler := NewCompiler()
	config, err := compiler.extractRepoMemoryConfig(toolsConfig, "")
	if err != nil {
		t.Fatalf("Failed to extract repo-memory config: %v", err)
	}

	if config == nil {
		t.Fatal("Expected non-nil config")
	}

	if len(config.Memories) != 2 {
		t.Fatalf("Expected 2 memories, got %d", len(config.Memories))
	}

	// Check first memory
	memory1 := config.Memories[0]
	if memory1.ID != "session" {
		t.Errorf("Expected ID 'session', got '%s'", memory1.ID)
	}
	if memory1.BranchName != "memory/session" {
		t.Errorf("Expected branch name 'memory/session', got '%s'", memory1.BranchName)
	}

	// Check second memory
	memory2 := config.Memories[1]
	if memory2.ID != "logs" {
		t.Errorf("Expected ID 'logs', got '%s'", memory2.ID)
	}
	if memory2.BranchName != "memory/logs" {
		t.Errorf("Expected branch name 'memory/logs', got '%s'", memory2.BranchName)
	}
	if memory2.MaxFileSize != 2097152 {
		t.Errorf("Expected max file size 2097152, got %d", memory2.MaxFileSize)
	}
}

// TestRepoMemoryConfigDuplicateIDs tests that duplicate memory IDs are rejected
func TestRepoMemoryConfigDuplicateIDs(t *testing.T) {
	toolsMap := map[string]any{
		"repo-memory": []any{
			map[string]any{
				"id":          "session",
				"branch-name": "memory/session",
			},
			map[string]any{
				"id":          "session",
				"branch-name": "memory/session2",
			},
		},
	}

	toolsConfig, err := ParseToolsConfig(toolsMap)
	if err != nil {
		t.Fatalf("Failed to parse tools config: %v", err)
	}

	compiler := NewCompiler()
	_, err = compiler.extractRepoMemoryConfig(toolsConfig, "")
	if err == nil {
		t.Fatal("Expected error for duplicate memory IDs, got nil")
	}

	if !strings.Contains(err.Error(), "duplicate memory ID") {
		t.Errorf("Expected error about duplicate memory ID, got: %v", err)
	}
}

// TestRepoMemoryStepsGeneration tests that repo-memory steps are generated correctly
func TestRepoMemoryStepsGeneration(t *testing.T) {
	config := &RepoMemoryConfig{
		Memories: []RepoMemoryEntry{
			{
				ID:           "default",
				BranchName:   "memory/default",
				MaxFileSize:  10240,
				MaxFileCount: 100,
				CreateOrphan: true,
			},
		},
	}

	data := &WorkflowData{
		RepoMemoryConfig: config,
	}

	var builder strings.Builder
	generateRepoMemorySteps(&builder, data)

	output := builder.String()

	// Check for clone step
	if !strings.Contains(output, "Clone repo-memory branch (default)") {
		t.Error("Expected clone step for repo-memory")
	}

	// Check for script call
	if !strings.Contains(output, "bash \"${RUNNER_TEMP}/gh-aw/actions/clone_repo_memory_branch.sh\"") {
		t.Error("Expected clone_repo_memory_branch.sh script call")
	}

	// Check for environment variables
	if !strings.Contains(output, "BRANCH_NAME: memory/default") {
		t.Error("Expected BRANCH_NAME environment variable")
	}

	if !strings.Contains(output, "CREATE_ORPHAN: true") {
		t.Error("Expected CREATE_ORPHAN environment variable")
	}

	if !strings.Contains(output, "MEMORY_DIR: /tmp/gh-aw/repo-memory/default") {
		t.Error("Expected MEMORY_DIR environment variable")
	}

	// Check for memory directory creation
	if !strings.Contains(output, "/tmp/gh-aw/repo-memory/default") {
		t.Error("Expected memory directory path")
	}
}

// TestRepoMemoryPromptGeneration tests that prompt section is generated correctly
func TestRepoMemoryPromptGeneration(t *testing.T) {
	config := &RepoMemoryConfig{
		Memories: []RepoMemoryEntry{
			{
				ID:          "default",
				BranchName:  "memory/default",
				Description: "Persistent memory for agent state",
			},
		},
	}

	section := buildRepoMemoryPromptSection(config)

	require.NotNil(t, section, "Expected non-nil prompt section")
	assert.True(t, section.IsFile, "Should use template file")
	assert.Equal(t, repoMemoryPromptFile, section.Content, "Should reference repo memory prompt file")
	require.NotNil(t, section.EnvVars, "Should have environment variables")

	// Check for prompt header key
	assert.Equal(t, "/tmp/gh-aw/repo-memory/default/", section.EnvVars["GH_AW_MEMORY_DIR"], "Should have correct memory directory")

	// Check for description
	assert.Equal(t, " Persistent memory for agent state", section.EnvVars["GH_AW_MEMORY_DESCRIPTION"], "Expected custom description with leading space")

	// Check for key information
	assert.Equal(t, "memory/default", section.EnvVars["GH_AW_MEMORY_BRANCH_NAME"], "Should have correct branch name")
	assert.Equal(t, " of the current repository", section.EnvVars["GH_AW_MEMORY_TARGET_REPO"], "Should default to current repository")
}

// TestRepoMemoryMaxFileSizeValidation tests max-file-size boundary validation
func TestRepoMemoryMaxFileSizeValidation(t *testing.T) {
	tests := []struct {
		name        string
		maxFileSize int
		wantError   bool
		errorText   string
	}{
		{
			name:        "valid minimum size (1 byte)",
			maxFileSize: 1,
			wantError:   false,
		},
		{
			name:        "valid maximum size (104857600 bytes)",
			maxFileSize: 104857600,
			wantError:   false,
		},
		{
			name:        "valid mid-range size (10240 bytes)",
			maxFileSize: 10240,
			wantError:   false,
		},
		{
			name:        "invalid zero size",
			maxFileSize: 0,
			wantError:   true,
			errorText:   "max-file-size must be between 1 and 104857600, got 0",
		},
		{
			name:        "invalid negative size",
			maxFileSize: -1,
			wantError:   true,
			errorText:   "max-file-size must be between 1 and 104857600, got -1",
		},
		{
			name:        "invalid size exceeds maximum",
			maxFileSize: 104857601,
			wantError:   true,
			errorText:   "max-file-size must be between 1 and 104857600, got 104857601",
		},
		{
			name:        "invalid large size",
			maxFileSize: 200000000,
			wantError:   true,
			errorText:   "max-file-size must be between 1 and 104857600, got 200000000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			toolsMap := map[string]any{
				"repo-memory": map[string]any{
					"max-file-size": tt.maxFileSize,
				},
			}

			toolsConfig, err := ParseToolsConfig(toolsMap)
			if err != nil {
				t.Fatalf("Failed to parse tools config: %v", err)
			}

			compiler := NewCompiler()
			config, err := compiler.extractRepoMemoryConfig(toolsConfig, "")

			if tt.wantError {
				if err == nil {
					t.Errorf("Expected error, got nil")
				} else if !strings.Contains(err.Error(), tt.errorText) {
					t.Errorf("Expected error containing '%s', got '%s'", tt.errorText, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got: %v", err)
				}
				if config == nil {
					t.Fatal("Expected non-nil config")
				}
				if len(config.Memories) != 1 {
					t.Fatalf("Expected 1 memory, got %d", len(config.Memories))
				}
				if config.Memories[0].MaxFileSize != tt.maxFileSize {
					t.Errorf("Expected max file size %d, got %d", tt.maxFileSize, config.Memories[0].MaxFileSize)
				}
			}
		})
	}
}

// TestRepoMemoryMaxFileSizeValidationArray tests max-file-size validation in array notation
func TestRepoMemoryMaxFileSizeValidationArray(t *testing.T) {
	tests := []struct {
		name        string
		maxFileSize int
		wantError   bool
		errorText   string
	}{
		{
			name:        "valid size in array",
			maxFileSize: 524288,
			wantError:   false,
		},
		{
			name:        "invalid size in array (zero)",
			maxFileSize: 0,
			wantError:   true,
			errorText:   "max-file-size must be between 1 and 104857600, got 0",
		},
		{
			name:        "invalid size in array (exceeds max)",
			maxFileSize: 104857601,
			wantError:   true,
			errorText:   "max-file-size must be between 1 and 104857600, got 104857601",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			toolsMap := map[string]any{
				"repo-memory": []any{
					map[string]any{
						"id":            "test",
						"max-file-size": tt.maxFileSize,
					},
				},
			}

			toolsConfig, err := ParseToolsConfig(toolsMap)
			if err != nil {
				t.Fatalf("Failed to parse tools config: %v", err)
			}

			compiler := NewCompiler()
			config, err := compiler.extractRepoMemoryConfig(toolsConfig, "")

			if tt.wantError {
				if err == nil {
					t.Errorf("Expected error, got nil")
				} else if !strings.Contains(err.Error(), tt.errorText) {
					t.Errorf("Expected error containing '%s', got '%s'", tt.errorText, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got: %v", err)
				}
				if config == nil {
					t.Fatal("Expected non-nil config")
				}
				if len(config.Memories) != 1 {
					t.Fatalf("Expected 1 memory, got %d", len(config.Memories))
				}
				if config.Memories[0].MaxFileSize != tt.maxFileSize {
					t.Errorf("Expected max file size %d, got %d", tt.maxFileSize, config.Memories[0].MaxFileSize)
				}
			}
		})
	}
}

// TestRepoMemoryMaxFileCountValidation tests max-file-count boundary validation
func TestRepoMemoryMaxFileCountValidation(t *testing.T) {
	tests := []struct {
		name         string
		maxFileCount int
		wantError    bool
		errorText    string
	}{
		{
			name:         "valid minimum count (1 file)",
			maxFileCount: 1,
			wantError:    false,
		},
		{
			name:         "valid maximum count (1000 files)",
			maxFileCount: 1000,
			wantError:    false,
		},
		{
			name:         "valid mid-range count (100 files)",
			maxFileCount: 100,
			wantError:    false,
		},
		{
			name:         "invalid zero count",
			maxFileCount: 0,
			wantError:    true,
			errorText:    "max-file-count must be between 1 and 1000, got 0",
		},
		{
			name:         "invalid negative count",
			maxFileCount: -1,
			wantError:    true,
			errorText:    "max-file-count must be between 1 and 1000, got -1",
		},
		{
			name:         "invalid count exceeds maximum",
			maxFileCount: 1001,
			wantError:    true,
			errorText:    "max-file-count must be between 1 and 1000, got 1001",
		},
		{
			name:         "invalid large count",
			maxFileCount: 5000,
			wantError:    true,
			errorText:    "max-file-count must be between 1 and 1000, got 5000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			toolsMap := map[string]any{
				"repo-memory": map[string]any{
					"max-file-count": tt.maxFileCount,
				},
			}

			toolsConfig, err := ParseToolsConfig(toolsMap)
			if err != nil {
				t.Fatalf("Failed to parse tools config: %v", err)
			}

			compiler := NewCompiler()
			config, err := compiler.extractRepoMemoryConfig(toolsConfig, "")

			if tt.wantError {
				if err == nil {
					t.Errorf("Expected error, got nil")
				} else if !strings.Contains(err.Error(), tt.errorText) {
					t.Errorf("Expected error containing '%s', got '%s'", tt.errorText, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got: %v", err)
				}
				if config == nil {
					t.Fatal("Expected non-nil config")
				}
				if len(config.Memories) != 1 {
					t.Fatalf("Expected 1 memory, got %d", len(config.Memories))
				}
				if config.Memories[0].MaxFileCount != tt.maxFileCount {
					t.Errorf("Expected max file count %d, got %d", tt.maxFileCount, config.Memories[0].MaxFileCount)
				}
			}
		})
	}
}

// TestRepoMemoryMaxFileCountValidationArray tests max-file-count validation in array notation
func TestRepoMemoryMaxFileCountValidationArray(t *testing.T) {
	tests := []struct {
		name         string
		maxFileCount int
		wantError    bool
		errorText    string
	}{
		{
			name:         "valid count in array",
			maxFileCount: 50,
			wantError:    false,
		},
		{
			name:         "invalid count in array (zero)",
			maxFileCount: 0,
			wantError:    true,
			errorText:    "max-file-count must be between 1 and 1000, got 0",
		},
		{
			name:         "invalid count in array (exceeds max)",
			maxFileCount: 1001,
			wantError:    true,
			errorText:    "max-file-count must be between 1 and 1000, got 1001",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			toolsMap := map[string]any{
				"repo-memory": []any{
					map[string]any{
						"id":             "test",
						"max-file-count": tt.maxFileCount,
					},
				},
			}

			toolsConfig, err := ParseToolsConfig(toolsMap)
			if err != nil {
				t.Fatalf("Failed to parse tools config: %v", err)
			}

			compiler := NewCompiler()
			config, err := compiler.extractRepoMemoryConfig(toolsConfig, "")

			if tt.wantError {
				if err == nil {
					t.Errorf("Expected error, got nil")
				} else if !strings.Contains(err.Error(), tt.errorText) {
					t.Errorf("Expected error containing '%s', got '%s'", tt.errorText, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got: %v", err)
				}
				if config == nil {
					t.Fatal("Expected non-nil config")
				}
				if len(config.Memories) != 1 {
					t.Fatalf("Expected 1 memory, got %d", len(config.Memories))
				}
				if config.Memories[0].MaxFileCount != tt.maxFileCount {
					t.Errorf("Expected max file count %d, got %d", tt.maxFileCount, config.Memories[0].MaxFileCount)
				}
			}
		})
	}
}

// TestRepoMemoryMaxPatchSizeDefault tests that max-patch-size defaults to 10KB
func TestRepoMemoryMaxPatchSizeDefault(t *testing.T) {
	toolsMap := map[string]any{
		"repo-memory": true,
	}

	toolsConfig, err := ParseToolsConfig(toolsMap)
	require.NoError(t, err, "Should parse tools config")

	compiler := NewCompiler()
	config, err := compiler.extractRepoMemoryConfig(toolsConfig, "my-workflow")
	require.NoError(t, err, "Should extract repo-memory config")
	require.NotNil(t, config, "Config should not be nil")
	require.Len(t, config.Memories, 1, "Should have 1 memory")

	assert.Equal(t, 10240, config.Memories[0].MaxPatchSize, "Default max patch size should be 10240 bytes (10KB)")
}

// TestRepoMemoryMaxPatchSizeValidation tests max-patch-size boundary validation
func TestRepoMemoryMaxPatchSizeValidation(t *testing.T) {
	tests := []struct {
		name         string
		maxPatchSize int
		wantError    bool
		errorText    string
	}{
		{
			name:         "valid minimum size (1 byte)",
			maxPatchSize: 1,
			wantError:    false,
		},
		{
			name:         "valid maximum size (102400 bytes = 100KB)",
			maxPatchSize: 102400,
			wantError:    false,
		},
		{
			name:         "valid default size (10240 bytes)",
			maxPatchSize: 10240,
			wantError:    false,
		},
		{
			name:         "valid custom size (51200 bytes = 50KB)",
			maxPatchSize: 51200,
			wantError:    false,
		},
		{
			name:         "invalid zero size",
			maxPatchSize: 0,
			wantError:    true,
			errorText:    "max-patch-size must be between 1 and 102400, got 0",
		},
		{
			name:         "invalid negative size",
			maxPatchSize: -1,
			wantError:    true,
			errorText:    "max-patch-size must be between 1 and 102400, got -1",
		},
		{
			name:         "invalid size exceeds maximum",
			maxPatchSize: 102401,
			wantError:    true,
			errorText:    "max-patch-size must be between 1 and 102400, got 102401",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			toolsMap := map[string]any{
				"repo-memory": map[string]any{
					"max-patch-size": tt.maxPatchSize,
				},
			}

			toolsConfig, err := ParseToolsConfig(toolsMap)
			require.NoError(t, err, "Should parse tools config")

			compiler := NewCompiler()
			config, err := compiler.extractRepoMemoryConfig(toolsConfig, "")

			if tt.wantError {
				require.Error(t, err, "Should return an error")
				if err != nil {
					assert.Contains(t, err.Error(), tt.errorText, "Error message should match")
				}
			} else {
				require.NoError(t, err, "Should not return an error")
				if config != nil && len(config.Memories) > 0 {
					assert.Equal(t, tt.maxPatchSize, config.Memories[0].MaxPatchSize, "MaxPatchSize should match")
				}
			}
		})
	}
}

// TestRepoMemoryMaxPatchSizeValidationArray tests max-patch-size validation in array notation
func TestRepoMemoryMaxPatchSizeValidationArray(t *testing.T) {
	tests := []struct {
		name         string
		maxPatchSize int
		wantError    bool
		errorText    string
	}{
		{
			name:         "valid size in array",
			maxPatchSize: 10240,
			wantError:    false,
		},
		{
			name:         "invalid size in array (zero)",
			maxPatchSize: 0,
			wantError:    true,
			errorText:    "max-patch-size must be between 1 and 102400, got 0",
		},
		{
			name:         "invalid size in array (exceeds max)",
			maxPatchSize: 102401,
			wantError:    true,
			errorText:    "max-patch-size must be between 1 and 102400, got 102401",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			toolsMap := map[string]any{
				"repo-memory": []any{
					map[string]any{
						"id":             "test",
						"max-patch-size": tt.maxPatchSize,
					},
				},
			}

			toolsConfig, err := ParseToolsConfig(toolsMap)
			require.NoError(t, err, "Should parse tools config")

			compiler := NewCompiler()
			config, err := compiler.extractRepoMemoryConfig(toolsConfig, "")

			if tt.wantError {
				require.Error(t, err, "Should return an error")
				if err != nil {
					assert.Contains(t, err.Error(), tt.errorText, "Error message should match")
				}
			} else {
				require.NoError(t, err, "Should not return an error")
				if config != nil && len(config.Memories) > 0 {
					assert.Equal(t, tt.maxPatchSize, config.Memories[0].MaxPatchSize, "MaxPatchSize should match")
				}
			}
		})
	}
}

// TestBranchPrefixValidation tests the validateBranchPrefix function
func TestBranchPrefixValidation(t *testing.T) {
	tests := []struct {
		name    string
		prefix  string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "empty prefix (use default)",
			prefix:  "",
			wantErr: false,
		},
		{
			name:    "valid prefix - alphanumeric",
			prefix:  "memories",
			wantErr: false,
		},
		{
			name:    "valid prefix - daily (5 chars)",
			prefix:  "daily",
			wantErr: false,
		},
		{
			name:    "valid prefix - with hyphens",
			prefix:  "my-memory",
			wantErr: false,
		},
		{
			name:    "valid prefix - with underscores",
			prefix:  "my_memory",
			wantErr: false,
		},
		{
			name:    "valid prefix - mixed",
			prefix:  "test_mem-123",
			wantErr: false,
		},
		{
			name:    "valid prefix - exactly 4 chars",
			prefix:  "mem1",
			wantErr: false,
		},
		{
			name:    "valid prefix - 5 chars",
			prefix:  "mem12",
			wantErr: false,
		},
		{
			name:    "valid prefix - exactly 32 chars",
			prefix:  "12345678901234567890123456789012",
			wantErr: false,
		},
		{
			name:    "invalid - too short (3 chars)",
			prefix:  "mem",
			wantErr: true,
			errMsg:  "must be at least 4 characters long",
		},
		{
			name:    "invalid - too long (33 chars)",
			prefix:  "123456789012345678901234567890123",
			wantErr: true,
			errMsg:  "must be at most 32 characters long",
		},
		{
			name:    "invalid - contains slash",
			prefix:  "memory/branch",
			wantErr: true,
			errMsg:  "must contain only alphanumeric characters",
		},
		{
			name:    "invalid - contains space",
			prefix:  "my memory",
			wantErr: true,
			errMsg:  "must contain only alphanumeric characters",
		},
		{
			name:    "invalid - contains special char",
			prefix:  "memory@branch",
			wantErr: true,
			errMsg:  "must contain only alphanumeric characters",
		},
		{
			name:    "invalid - reserved word 'copilot'",
			prefix:  "copilot",
			wantErr: true,
			errMsg:  "cannot be 'copilot' (reserved)",
		},
		{
			name:    "invalid - reserved word 'Copilot' (case-insensitive)",
			prefix:  "Copilot",
			wantErr: true,
			errMsg:  "cannot be 'copilot' (reserved)",
		},
		{
			name:    "invalid - reserved word 'COPILOT' (case-insensitive)",
			prefix:  "COPILOT",
			wantErr: true,
			errMsg:  "cannot be 'copilot' (reserved)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateBranchPrefix(tt.prefix)
			if tt.wantErr {
				require.Error(t, err, "Expected error for prefix: %s", tt.prefix)
				assert.Contains(t, err.Error(), tt.errMsg, "Error message should contain: %s", tt.errMsg)
			} else {
				require.NoError(t, err, "Expected no error for prefix: %s", tt.prefix)
			}
		})
	}
}

// TestBranchPrefixInConfig tests branch-prefix in object configuration
func TestBranchPrefixInConfig(t *testing.T) {
	toolsMap := map[string]any{
		"repo-memory": map[string]any{
			"branch-prefix": "campaigns",
		},
	}

	toolsConfig, err := ParseToolsConfig(toolsMap)
	require.NoError(t, err, "Failed to parse tools config")

	compiler := NewCompiler()
	config, err := compiler.extractRepoMemoryConfig(toolsConfig, "my-workflow")
	require.NoError(t, err, "Failed to extract repo-memory config")
	require.NotNil(t, config, "Expected non-nil config")

	assert.Equal(t, "campaigns", config.BranchPrefix, "Expected branch-prefix 'campaigns'")
	assert.Len(t, config.Memories, 1, "Expected 1 memory")

	memory := config.Memories[0]
	assert.Equal(t, "campaigns/my-workflow", memory.BranchName, "Expected branch name 'campaigns/my-workflow'")
}

// TestBranchPrefixInArrayConfig tests branch-prefix in array configuration
func TestBranchPrefixInArrayConfig(t *testing.T) {
	toolsMap := map[string]any{
		"repo-memory": []any{
			map[string]any{
				"id":            "session",
				"branch-prefix": "my-prefix",
			},
			map[string]any{
				"id": "logs",
			},
		},
	}

	toolsConfig, err := ParseToolsConfig(toolsMap)
	require.NoError(t, err, "Failed to parse tools config")

	compiler := NewCompiler()
	config, err := compiler.extractRepoMemoryConfig(toolsConfig, "")
	require.NoError(t, err, "Failed to extract repo-memory config")
	require.NotNil(t, config, "Expected non-nil config")

	assert.Equal(t, "my-prefix", config.BranchPrefix, "Expected branch-prefix 'my-prefix'")
	assert.Len(t, config.Memories, 2, "Expected 2 memories")

	// Both memories should use the same prefix
	assert.Equal(t, "my-prefix/session", config.Memories[0].BranchName, "Expected branch name 'my-prefix/session'")
	assert.Equal(t, "my-prefix/logs", config.Memories[1].BranchName, "Expected branch name 'my-prefix/logs'")
}

// TestBranchPrefixWithExplicitBranchName tests that explicit branch-name overrides prefix
func TestBranchPrefixWithExplicitBranchName(t *testing.T) {
	toolsMap := map[string]any{
		"repo-memory": map[string]any{
			"branch-prefix": "campaigns",
			"branch-name":   "custom/branch",
		},
	}

	toolsConfig, err := ParseToolsConfig(toolsMap)
	require.NoError(t, err, "Failed to parse tools config")

	compiler := NewCompiler()
	config, err := compiler.extractRepoMemoryConfig(toolsConfig, "")
	require.NoError(t, err, "Failed to extract repo-memory config")
	require.NotNil(t, config, "Expected non-nil config")

	assert.Equal(t, "campaigns", config.BranchPrefix, "Expected branch-prefix 'campaigns'")

	memory := config.Memories[0]
	// Explicit branch-name should override the prefix
	assert.Equal(t, "custom/branch", memory.BranchName, "Expected explicit branch name 'custom/branch'")
}

// TestInvalidBranchPrefixRejectsConfig tests that invalid prefix causes error
func TestInvalidBranchPrefixRejectsConfig(t *testing.T) {
	tests := []struct {
		name   string
		prefix string
	}{
		{"too long", "this_is_a_very_long_prefix_that_exceeds_32_characters"},
		{"reserved word", "copilot"},
		{"special chars", "my@prefix"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			toolsMap := map[string]any{
				"repo-memory": map[string]any{
					"branch-prefix": tt.prefix,
				},
			}

			toolsConfig, err := ParseToolsConfig(toolsMap)
			require.NoError(t, err, "Failed to parse tools config")

			compiler := NewCompiler()
			config, err := compiler.extractRepoMemoryConfig(toolsConfig, "")
			require.Error(t, err, "Expected error for invalid branch-prefix: %s", tt.prefix)
			assert.Nil(t, config, "Expected nil config on error")
		})
	}
}

// TestRepoMemoryWikiObjectConfig tests that wiki:true in object config works correctly
func TestRepoMemoryWikiObjectConfig(t *testing.T) {
	toolsMap := map[string]any{
		"repo-memory": map[string]any{
			"wiki": true,
		},
	}

	toolsConfig, err := ParseToolsConfig(toolsMap)
	require.NoError(t, err, "Failed to parse tools config")

	compiler := NewCompiler()
	config, err := compiler.extractRepoMemoryConfig(toolsConfig, "my-workflow")
	require.NoError(t, err, "Failed to extract repo-memory config")
	require.NotNil(t, config, "Expected non-nil config")
	require.Len(t, config.Memories, 1, "Expected 1 memory")

	memory := config.Memories[0]
	assert.True(t, memory.Wiki, "Expected wiki to be true")
	assert.Equal(t, "master", memory.BranchName, "Wiki mode should default to master branch")
	assert.False(t, memory.CreateOrphan, "Wiki mode should disable create-orphan")
}

// TestRepoMemoryWikiArrayConfig tests that wiki:true in array config works correctly
func TestRepoMemoryWikiArrayConfig(t *testing.T) {
	toolsMap := map[string]any{
		"repo-memory": []any{
			map[string]any{
				"id":   "wiki-memory",
				"wiki": true,
			},
		},
	}

	toolsConfig, err := ParseToolsConfig(toolsMap)
	require.NoError(t, err, "Failed to parse tools config")

	compiler := NewCompiler()
	config, err := compiler.extractRepoMemoryConfig(toolsConfig, "my-workflow")
	require.NoError(t, err, "Failed to extract repo-memory config")
	require.NotNil(t, config, "Expected non-nil config")
	require.Len(t, config.Memories, 1, "Expected 1 memory")

	memory := config.Memories[0]
	assert.Equal(t, "wiki-memory", memory.ID, "Expected correct memory ID")
	assert.True(t, memory.Wiki, "Expected wiki to be true")
	assert.Equal(t, "master", memory.BranchName, "Wiki mode should default to master branch")
	assert.False(t, memory.CreateOrphan, "Wiki mode should disable create-orphan")
}

// TestRepoMemoryWikiCustomBranchName tests that wiki mode respects an explicit branch-name
func TestRepoMemoryWikiCustomBranchName(t *testing.T) {
	toolsMap := map[string]any{
		"repo-memory": map[string]any{
			"wiki":        true,
			"branch-name": "custom-branch",
		},
	}

	toolsConfig, err := ParseToolsConfig(toolsMap)
	require.NoError(t, err, "Failed to parse tools config")

	compiler := NewCompiler()
	config, err := compiler.extractRepoMemoryConfig(toolsConfig, "")
	require.NoError(t, err, "Failed to extract repo-memory config")
	require.NotNil(t, config, "Expected non-nil config")

	memory := config.Memories[0]
	assert.True(t, memory.Wiki, "Expected wiki to be true")
	assert.Equal(t, "custom-branch", memory.BranchName, "Wiki mode should respect explicit branch-name")
}

// TestRepoMemoryWikiStepsGeneration tests that wiki:true appends .wiki to TARGET_REPO
func TestRepoMemoryWikiStepsGeneration(t *testing.T) {
	config := &RepoMemoryConfig{
		Memories: []RepoMemoryEntry{
			{
				ID:         "default",
				BranchName: "master",
				Wiki:       true,
			},
		},
	}

	data := &WorkflowData{
		RepoMemoryConfig: config,
	}

	var builder strings.Builder
	generateRepoMemorySteps(&builder, data)

	output := builder.String()
	assert.Contains(t, output, "TARGET_REPO: ${{ github.repository }}.wiki",
		"Wiki mode should append .wiki to TARGET_REPO")
	assert.Contains(t, output, "- name: Clone wiki-memory branch (default)",
		"Wiki mode should use wiki-memory step name")
}

// TestRepoMemoryWikiStepsWithTargetRepo tests wiki mode with explicit target-repo
func TestRepoMemoryWikiStepsWithTargetRepo(t *testing.T) {
	config := &RepoMemoryConfig{
		Memories: []RepoMemoryEntry{
			{
				ID:         "default",
				BranchName: "master",
				TargetRepo: "myorg/myrepo",
				Wiki:       true,
			},
		},
	}

	data := &WorkflowData{
		RepoMemoryConfig: config,
	}

	var builder strings.Builder
	generateRepoMemorySteps(&builder, data)

	output := builder.String()
	assert.Contains(t, output, "TARGET_REPO: myorg/myrepo.wiki",
		"Wiki mode should append .wiki to explicit target-repo")
	assert.Contains(t, output, "- name: Clone wiki-memory branch (default)",
		"Wiki mode should use wiki-memory step name")
}

// TestRepoMemoryWikiPromptSection tests that wiki mode injects wiki note into prompt
func TestRepoMemoryWikiPromptSection(t *testing.T) {
	config := &RepoMemoryConfig{
		Memories: []RepoMemoryEntry{
			{
				ID:         "default",
				BranchName: "master",
				Wiki:       true,
			},
		},
	}

	section := buildRepoMemoryPromptSection(config)

	require.NotNil(t, section, "Expected non-nil prompt section")
	require.NotNil(t, section.EnvVars, "Expected env vars")

	wikiNote := section.EnvVars["GH_AW_WIKI_NOTE"]
	assert.NotEmpty(t, wikiNote, "Wiki mode should set GH_AW_WIKI_NOTE")
	assert.Contains(t, wikiNote, "GitHub Wiki", "Wiki note should mention GitHub Wiki")
	assert.Contains(t, wikiNote, "Markdown", "Wiki note should mention Markdown syntax")
}

// TestRepoMemoryNonWikiPromptSection tests that non-wiki mode uses empty string expression for wiki note
func TestRepoMemoryNonWikiPromptSection(t *testing.T) {
	config := &RepoMemoryConfig{
		Memories: []RepoMemoryEntry{
			{
				ID:         "default",
				BranchName: "memory/default",
			},
		},
	}

	section := buildRepoMemoryPromptSection(config)

	require.NotNil(t, section, "Expected non-nil prompt section")
	require.NotNil(t, section.EnvVars, "Expected env vars")

	wikiNote := section.EnvVars["GH_AW_WIKI_NOTE"]
	// Non-wiki mode should use a GitHub expression that evaluates to empty string.
	// This ensures __GH_AW_WIKI_NOTE__ is always substituted via expression interpolation.
	assert.Equal(t, ghaEmptyStringExpr, wikiNote, "Non-wiki mode should use empty string expression for GH_AW_WIKI_NOTE")
}

// TestRepoMemoryWikiPushAllowedRepos tests that wiki mode sets REPO_MEMORY_ALLOWED_REPOS
// in the push step so the push script accepts the .wiki repo as a valid target.
func TestRepoMemoryWikiPushAllowedRepos(t *testing.T) {
	config := &RepoMemoryConfig{
		Memories: []RepoMemoryEntry{
			{
				ID:         "default",
				BranchName: "master",
				Wiki:       true,
			},
		},
	}

	data := &WorkflowData{
		RepoMemoryConfig: config,
	}

	compiler := NewCompiler()
	pushJob, err := compiler.buildPushRepoMemoryJob(data, false)
	require.NoError(t, err, "Should build push job without error")
	require.NotNil(t, pushJob, "Should produce a push job")

	pushJobOutput := strings.Join(pushJob.Steps, "\n")
	assert.Contains(t, pushJobOutput, "REPO_MEMORY_ALLOWED_REPOS: ${{ github.repository }}.wiki",
		"Wiki push step should pre-populate allowed repos with the wiki repo")
}

// TestRepoMemoryNonWikiPushNoAllowedRepos tests that non-wiki mode does NOT set REPO_MEMORY_ALLOWED_REPOS
func TestRepoMemoryNonWikiPushNoAllowedRepos(t *testing.T) {
	config := &RepoMemoryConfig{
		Memories: []RepoMemoryEntry{
			{
				ID:         "default",
				BranchName: "memory/default",
			},
		},
	}

	data := &WorkflowData{
		RepoMemoryConfig: config,
	}

	compiler := NewCompiler()
	pushJob, err := compiler.buildPushRepoMemoryJob(data, false)
	require.NoError(t, err, "Should build push job without error")
	require.NotNil(t, pushJob, "Should produce a push job")

	pushJobOutput := strings.Join(pushJob.Steps, "\n")
	assert.NotContains(t, pushJobOutput, "REPO_MEMORY_ALLOWED_REPOS",
		"Non-wiki push step should not set REPO_MEMORY_ALLOWED_REPOS")
}

// TestBuildPushRepoMemoryConcurrencyGroup verifies that the concurrency group key is scoped
// to the actual branch targets, so workflows pushing to different memory branches do not
// contend with each other.
func TestBuildPushRepoMemoryConcurrencyGroup(t *testing.T) {
	tests := []struct {
		name     string
		memories []RepoMemoryEntry
		expected string
	}{
		{
			name: "single memory uses branch name in key",
			memories: []RepoMemoryEntry{
				{ID: "default", BranchName: "memory/daily-news"},
			},
			expected: "push-repo-memory-${{ github.repository }}|memory/daily-news",
		},
		{
			name: "multiple memories sorted for deterministic key",
			memories: []RepoMemoryEntry{
				{ID: "b", BranchName: "memory/workflow-b"},
				{ID: "a", BranchName: "memory/workflow-a"},
			},
			expected: "push-repo-memory-${{ github.repository }}|memory/workflow-a|memory/workflow-b",
		},
		{
			name: "non-default target repo is prefixed to branch",
			memories: []RepoMemoryEntry{
				{ID: "default", BranchName: "memory/shared", TargetRepo: "other-org/other-repo"},
			},
			expected: "push-repo-memory-${{ github.repository }}|other-org/other-repo:memory/shared",
		},
		{
			name: "mix of default and custom target repos",
			memories: []RepoMemoryEntry{
				{ID: "local", BranchName: "memory/local"},
				{ID: "remote", BranchName: "memory/remote", TargetRepo: "other-org/other-repo"},
			},
			expected: "push-repo-memory-${{ github.repository }}|memory/local|other-org/other-repo:memory/remote",
		},
		{
			name: "branches with hyphens use pipe separator in key",
			memories: []RepoMemoryEntry{
				{ID: "a", BranchName: "memory/workflow-a"},
				{ID: "b", BranchName: "memory/workflow-b"},
			},
			// This expectation documents the current use of "|" as the list separator in the key.
			// If the concurrency key encoding changes (e.g., different delimiter or encoding scheme),
			// update this test to match the new encoding strategy.
			expected: "push-repo-memory-${{ github.repository }}|memory/workflow-a|memory/workflow-b",
		},
		{
			name: "pipe in branch name is percent-encoded so the separator stays unambiguous",
			memories: []RepoMemoryEntry{
				{ID: "unusual", BranchName: "memory/foo|bar"},
			},
			expected: "push-repo-memory-${{ github.repository }}|memory/foo%7Cbar",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildPushRepoMemoryConcurrencyGroup(tt.memories)
			assert.Equal(t, tt.expected, got, "Concurrency group key should match expected value")
		})
	}
}

// TestPushRepoMemoryJobConcurrencyKey verifies that buildPushRepoMemoryJob sets a concurrency
// key scoped to the actual memory branch rather than a repo-wide key.
func TestPushRepoMemoryJobConcurrencyKey(t *testing.T) {
	data := &WorkflowData{
		RepoMemoryConfig: &RepoMemoryConfig{
			Memories: []RepoMemoryEntry{
				{ID: "default", BranchName: "memory/my-workflow"},
			},
		},
	}

	compiler := NewCompiler()
	pushJob, err := compiler.buildPushRepoMemoryJob(data, false)
	require.NoError(t, err, "Should build push job without error")
	require.NotNil(t, pushJob, "Should produce a push job")

	assert.Contains(t, pushJob.Concurrency, "memory/my-workflow",
		"Concurrency key should include the memory branch name")
	// Ensure the old repo-wide-only key is not used
	assert.NotContains(t, pushJob.Concurrency, "push-repo-memory-${{ github.repository }}\"",
		"Concurrency key should not be the old repo-wide-only key format")
}

// TestPushRepoMemoryJobConditionGatesOnAgentNotSkipped verifies that the push_repo_memory
// job condition uses needs.agent.result != 'skipped' and !cancelled() so the job does not
// run on no-op workflow invocations (e.g. bot comments where pre_activation is skipped)
// or after workflow cancellation.
func TestPushRepoMemoryJobConditionGatesOnAgentNotSkipped(t *testing.T) {
	data := &WorkflowData{
		RepoMemoryConfig: &RepoMemoryConfig{
			Memories: []RepoMemoryEntry{
				{ID: "default", BranchName: "memory/my-workflow"},
			},
		},
	}

	compiler := NewCompiler()

	t.Run("without threat detection", func(t *testing.T) {
		pushJob, err := compiler.buildPushRepoMemoryJob(data, false)
		require.NoError(t, err, "Should build push job without error")
		require.NotNil(t, pushJob, "Should produce a push job")

		assert.Equal(t,
			"always() && (!cancelled()) && needs.agent.result == 'success'",
			pushJob.If,
			"Condition should use always() && (!cancelled()) && agent == 'success'",
		)
	})

	t.Run("with threat detection", func(t *testing.T) {
		pushJob, err := compiler.buildPushRepoMemoryJob(data, true)
		require.NoError(t, err, "Should build push job without error")
		require.NotNil(t, pushJob, "Should produce a push job")

		assert.Contains(t, pushJob.If, "always()",
			"Condition should contain always()")
		assert.Contains(t, pushJob.If, "!cancelled()",
			"Condition should contain !cancelled() to prevent running after cancellation")
		assert.Contains(t, pushJob.If, "needs.agent.result == 'success'",
			"Condition should check agent result == 'success'")
		assert.Contains(t, pushJob.If, "needs.detection.result",
			"Condition should still check detection result when threat detection is enabled")
		assert.NotContains(t, pushJob.If, "needs.agent.result != 'skipped'",
			"Condition should NOT use != 'skipped' for agent check")
	})
}
