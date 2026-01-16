package workflow

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateUnifiedPromptStep_AllSections(t *testing.T) {
	// Test that all prompt sections are included when all features are enabled
	compiler := &Compiler{
		trialMode:            false,
		trialLogicalRepoSlug: "",
	}

	data := &WorkflowData{
		ParsedTools: NewTools(map[string]any{
			"playwright": true,
			"github":     true,
		}),
		CacheMemoryConfig: &CacheMemoryConfig{
			Caches: []CacheMemoryEntry{
				{ID: "default"},
			},
		},
		RepoMemoryConfig: &RepoMemoryConfig{
			Memories: []RepoMemoryEntry{
				{ID: "default", BranchName: "memory"},
			},
		},
		SafeOutputs: &SafeOutputsConfig{
			CreateIssues: &CreateIssuesConfig{},
		},
		Permissions: "contents: read",
		On:          "issue_comment",
	}

	var yaml strings.Builder
	compiler.generateUnifiedPromptStep(&yaml, data)

	output := yaml.String()

	// Verify single step is created with correct name
	assert.Contains(t, output, "- name: Append context instructions to prompt")

	// Verify all sections are included
	assert.Contains(t, output, "temp_folder_prompt.md", "Should include temp folder instructions")
	assert.Contains(t, output, "playwright_prompt.md", "Should include playwright instructions")
	assert.Contains(t, output, "Cache Folder Available", "Should include cache memory instructions")
	assert.Contains(t, output, "Repo Memory Available", "Should include repo memory instructions")
	assert.Contains(t, output, "<safe-outputs>", "Should include safe outputs instructions")
	assert.Contains(t, output, "<github-context>", "Should include GitHub context")

	// Verify environment variables are declared at the top
	lines := strings.Split(output, "\n")
	envSectionStarted := false
	runSectionStarted := false
	for _, line := range lines {
		if strings.Contains(line, "env:") {
			envSectionStarted = true
		}
		if strings.Contains(line, "run: |") {
			runSectionStarted = true
		}
		// Check that environment variable declarations (key: ${{ ... }}) are in env section
		// Skip lines that are just references to the variables (like __GH_AW_GITHUB_ACTOR__)
		if strings.Contains(line, ": ${{") && runSectionStarted {
			t.Errorf("Found environment variable declaration after run section started: %s", line)
		}
	}
	assert.True(t, envSectionStarted, "Should have env section")
	assert.True(t, runSectionStarted, "Should have run section")
}

func TestGenerateUnifiedPromptStep_MinimalSections(t *testing.T) {
	// Test that only temp folder is included when no other features are enabled
	compiler := &Compiler{
		trialMode:            false,
		trialLogicalRepoSlug: "",
	}

	data := &WorkflowData{
		ParsedTools:       NewTools(map[string]any{}),
		CacheMemoryConfig: nil,
		RepoMemoryConfig:  nil,
		SafeOutputs:       nil,
		Permissions:       "",
		On:                "push",
	}

	var yaml strings.Builder
	compiler.generateUnifiedPromptStep(&yaml, data)

	output := yaml.String()

	// Verify single step is created
	assert.Contains(t, output, "- name: Append context instructions to prompt")

	// Verify only temp folder is included
	assert.Contains(t, output, "temp_folder_prompt.md", "Should include temp folder instructions")

	// Verify other sections are NOT included
	assert.NotContains(t, output, "playwright_prompt.md", "Should not include playwright without tool")
	assert.NotContains(t, output, "Cache Folder Available", "Should not include cache memory without config")
	assert.NotContains(t, output, "Repo Memory Available", "Should not include repo memory without config")
	assert.NotContains(t, output, "<safe-outputs>", "Should not include safe outputs without config")
	assert.NotContains(t, output, "<github-context>", "Should not include GitHub context without tool")
}

func TestGenerateUnifiedPromptStep_TrialMode(t *testing.T) {
	// Test that trial mode note is included
	compiler := &Compiler{
		trialMode:            true,
		trialLogicalRepoSlug: "owner/repo",
	}

	data := &WorkflowData{
		ParsedTools:       NewTools(map[string]any{}),
		CacheMemoryConfig: nil,
		RepoMemoryConfig:  nil,
		SafeOutputs:       nil,
		Permissions:       "",
		On:                "push",
	}

	var yaml strings.Builder
	compiler.generateUnifiedPromptStep(&yaml, data)

	output := yaml.String()

	// Verify trial mode note is included
	assert.Contains(t, output, "## Note")
	assert.Contains(t, output, "owner/repo")
}

func TestGenerateUnifiedPromptStep_PRContext(t *testing.T) {
	// Test that PR context is included with proper condition
	compiler := &Compiler{
		trialMode:            false,
		trialLogicalRepoSlug: "",
	}

	data := &WorkflowData{
		ParsedTools:       NewTools(map[string]any{}),
		CacheMemoryConfig: nil,
		RepoMemoryConfig:  nil,
		SafeOutputs:       nil,
		Permissions:       "contents: read",
		On:                "issue_comment",
	}

	var yaml strings.Builder
	compiler.generateUnifiedPromptStep(&yaml, data)

	output := yaml.String()

	// Verify PR context is included with condition
	assert.Contains(t, output, "pr_context_prompt.md", "Should include PR context file")
	assert.Contains(t, output, "if [", "Should have shell conditional for PR context")
	assert.Contains(t, output, "GITHUB_EVENT_NAME", "Should check event name")
}

func TestCollectPromptSections_Order(t *testing.T) {
	// Test that sections are collected in the correct order
	compiler := &Compiler{
		trialMode:            true,
		trialLogicalRepoSlug: "owner/repo",
	}

	data := &WorkflowData{
		ParsedTools: NewTools(map[string]any{
			"playwright": true,
			"github":     true,
		}),
		CacheMemoryConfig: &CacheMemoryConfig{
			Caches: []CacheMemoryEntry{{ID: "default"}},
		},
		RepoMemoryConfig: &RepoMemoryConfig{
			Memories: []RepoMemoryEntry{{ID: "default", BranchName: "memory"}},
		},
		SafeOutputs: &SafeOutputsConfig{
			CreateIssues: &CreateIssuesConfig{},
		},
		Permissions: "contents: read",
		On:          "issue_comment",
	}

	sections := compiler.collectPromptSections(data)

	// Verify we have sections
	require.NotEmpty(t, sections, "Should collect sections")

	// Verify order:
	// 1. Temp folder
	// 2. Playwright
	// 3. Trial mode note
	// 4. Cache memory
	// 5. Repo memory
	// 6. Safe outputs
	// 7. GitHub context
	// 8. PR context

	var sectionTypes []string
	for _, section := range sections {
		if section.IsFile {
			if strings.Contains(section.Content, "temp_folder") {
				sectionTypes = append(sectionTypes, "temp")
			} else if strings.Contains(section.Content, "playwright") {
				sectionTypes = append(sectionTypes, "playwright")
			} else if strings.Contains(section.Content, "pr_context") {
				sectionTypes = append(sectionTypes, "pr-context")
			}
		} else {
			if strings.Contains(section.Content, "## Note") {
				sectionTypes = append(sectionTypes, "trial")
			} else if strings.Contains(section.Content, "Cache Folder") {
				sectionTypes = append(sectionTypes, "cache")
			} else if strings.Contains(section.Content, "Repo Memory") {
				sectionTypes = append(sectionTypes, "repo")
			} else if strings.Contains(section.Content, "safe-outputs") {
				sectionTypes = append(sectionTypes, "safe-outputs")
			} else if strings.Contains(section.Content, "github-context") {
				sectionTypes = append(sectionTypes, "github")
			}
		}
	}

	// Verify expected order (not all may be present, but order should be maintained)
	expectedOrder := []string{"temp", "playwright", "trial", "cache", "repo", "safe-outputs", "github", "pr-context"}

	// Check that the sections we found appear in the expected order
	lastIndex := -1
	for _, sectionType := range sectionTypes {
		currentIndex := -1
		for i, expected := range expectedOrder {
			if expected == sectionType {
				currentIndex = i
				break
			}
		}
		assert.Greater(t, currentIndex, lastIndex, "Section %s should appear after previous section", sectionType)
		lastIndex = currentIndex
	}
}

func TestWriteSectionContent_FileReference(t *testing.T) {
	compiler := &Compiler{}
	section := PromptSection{
		Content: "test_prompt.md",
		IsFile:  true,
	}

	var yaml strings.Builder
	compiler.writeSectionContent(&yaml, section, "  ")

	output := yaml.String()
	assert.Contains(t, output, `cat "/opt/gh-aw/prompts/test_prompt.md" >> "$GH_AW_PROMPT"`)
}

func TestWriteSectionContent_InlineContent(t *testing.T) {
	compiler := &Compiler{}
	section := PromptSection{
		Content: "Line 1\nLine 2\nLine 3",
		IsFile:  false,
	}

	var yaml strings.Builder
	compiler.writeSectionContent(&yaml, section, "  ")

	output := yaml.String()
	assert.Contains(t, output, "cat << 'PROMPT_EOF' >> \"$GH_AW_PROMPT\"")
	assert.Contains(t, output, "Line 1")
	assert.Contains(t, output, "Line 2")
	assert.Contains(t, output, "Line 3")
	assert.Contains(t, output, "PROMPT_EOF")
}

func TestGenerateUnifiedPromptStep_NoSections(t *testing.T) {
	// This should never happen in practice, but test the edge case
	compiler := &Compiler{
		trialMode: false,
	}

	// Create minimal data that would result in at least temp folder
	data := &WorkflowData{
		ParsedTools: NewTools(map[string]any{}),
	}

	var yaml strings.Builder
	compiler.generateUnifiedPromptStep(&yaml, data)

	output := yaml.String()

	// Should still generate step with at least temp folder
	assert.Contains(t, output, "- name: Append context instructions to prompt")
	assert.Contains(t, output, "temp_folder_prompt.md")
}
