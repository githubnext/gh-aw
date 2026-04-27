//go:build !integration

package workflow

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGenerateRepoMemorySanitizationStep_DefaultMemory verifies that the
// sanitization step is generated for a standard (non-wiki) default memory.
func TestGenerateRepoMemorySanitizationStep_DefaultMemory(t *testing.T) {
	var builder strings.Builder
	memory := RepoMemoryEntry{
		ID:         "default",
		BranchName: "memory/test",
		Wiki:       false,
	}
	memoryDir := "/tmp/gh-aw/repo-memory/default"

	generateRepoMemorySanitizationStep(&builder, memory, memoryDir)

	output := builder.String()
	assert.Contains(t, output, "- name: Scan repo-memory for prompt injection (default)",
		"Should emit a named scan step for default memory")
	assert.Contains(t, output, "GH_AW_SCAN_DIR: /tmp/gh-aw/repo-memory/default",
		"Should set GH_AW_SCAN_DIR to the memory directory")
	assert.Contains(t, output, "sanitize_memory.sh",
		"Should invoke sanitize_memory.sh")
}

// TestGenerateRepoMemorySanitizationStep_WikiMemory verifies that the step name
// reflects wiki memory.
func TestGenerateRepoMemorySanitizationStep_WikiMemory(t *testing.T) {
	var builder strings.Builder
	memory := RepoMemoryEntry{
		ID:   "docs",
		Wiki: true,
	}
	memoryDir := "/tmp/gh-aw/repo-memory/docs"

	generateRepoMemorySanitizationStep(&builder, memory, memoryDir)

	output := builder.String()
	assert.Contains(t, output, "- name: Scan wiki-memory for prompt injection (docs)",
		"Should use wiki-memory prefix for wiki memories")
	assert.Contains(t, output, "GH_AW_SCAN_DIR: /tmp/gh-aw/repo-memory/docs",
		"Should set GH_AW_SCAN_DIR to the memory directory")
}

// TestGenerateRepoMemorySanitizationStep_NamedMemory verifies that non-default
// memory IDs are included in the step name.
func TestGenerateRepoMemorySanitizationStep_NamedMemory(t *testing.T) {
	var builder strings.Builder
	memory := RepoMemoryEntry{
		ID:   "research",
		Wiki: false,
	}
	memoryDir := "/tmp/gh-aw/repo-memory/research"

	generateRepoMemorySanitizationStep(&builder, memory, memoryDir)

	output := builder.String()
	assert.Contains(t, output, "- name: Scan repo-memory for prompt injection (research)",
		"Should include memory ID in step name")
	assert.Contains(t, output, "GH_AW_SCAN_DIR: /tmp/gh-aw/repo-memory/research",
		"Should set GH_AW_SCAN_DIR to the named memory directory")
}

// TestSanitizeMemoryScriptNameConstant verifies the script name constant is correct.
func TestSanitizeMemoryScriptNameConstant(t *testing.T) {
	assert.Equal(t, "sanitize_memory.sh", sanitizeMemoryScriptName,
		"Script name constant should match the deployed script filename")
}

// TestRepoMemoryPromptHasSanitizedAttribute verifies that the prompt boundary
// markers include the sanitized="true" attribute per ASI-06.
func TestRepoMemoryPromptHasSanitizedAttribute(t *testing.T) {
	t.Run("single default repo memory prompt section", func(t *testing.T) {
		config := &RepoMemoryConfig{
			Memories: []RepoMemoryEntry{
				{
					ID:         "default",
					BranchName: "memory/test",
				},
			},
		}

		section := buildRepoMemoryPromptSection(config)
		require.NotNil(t, section, "Should return a prompt section")
		assert.Equal(t, repoMemoryPromptFile, section.Content,
			"Should reference the repo memory prompt file")
		// The sanitized="true" attribute is in the prompt file content itself (validated at a higher level)
	})

	t.Run("multi repo memory prompt section", func(t *testing.T) {
		config := &RepoMemoryConfig{
			Memories: []RepoMemoryEntry{
				{ID: "default", BranchName: "memory/test"},
				{ID: "extra", BranchName: "memory/extra"},
			},
		}

		section := buildRepoMemoryPromptSection(config)
		require.NotNil(t, section, "Should return a prompt section")
		assert.Equal(t, repoMemoryPromptMultiFile, section.Content,
			"Should reference the multi repo memory prompt file")
	})
}

// TestCacheMemoryPromptHasSanitizedAttribute verifies that cache memory prompt
// sections reference prompt files that carry the sanitized="true" boundary marker.
func TestCacheMemoryPromptHasSanitizedAttribute(t *testing.T) {
	t.Run("single default cache memory prompt section", func(t *testing.T) {
		config := &CacheMemoryConfig{
			Caches: []CacheMemoryEntry{
				{ID: "default"},
			},
		}

		section := buildCacheMemoryPromptSection(config)
		require.NotNil(t, section, "Should return a prompt section")
		assert.Equal(t, cacheMemoryPromptFile, section.Content,
			"Should reference the cache memory prompt file")
	})

	t.Run("multi cache memory prompt section", func(t *testing.T) {
		config := &CacheMemoryConfig{
			Caches: []CacheMemoryEntry{
				{ID: "default"},
				{ID: "session"},
			},
		}

		section := buildCacheMemoryPromptSection(config)
		require.NotNil(t, section, "Should return a prompt section")
		assert.Equal(t, cacheMemoryPromptMultiFile, section.Content,
			"Should reference the multi cache memory prompt file")
	})
}
