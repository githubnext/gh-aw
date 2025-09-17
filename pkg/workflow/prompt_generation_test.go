package workflow

import (
	"strings"
	"testing"
)

func TestGeneratePromptUsesJavaScriptAction(t *testing.T) {
	compiler := NewCompiler(false, "", "test")

	data := &WorkflowData{
		MarkdownContent: "Test workflow content",
	}

	var yaml strings.Builder
	compiler.generatePrompt(&yaml, data)

	output := yaml.String()

	// Check that it uses actions/github-script@v8
	if !strings.Contains(output, "uses: actions/github-script@v8") {
		t.Error("Expected 'uses: actions/github-script@v8' in prompt generation step")
	}

	// Check that GITHUB_AW_PROMPT environment variable is included
	if !strings.Contains(output, "GITHUB_AW_PROMPT: /tmp/aw-prompts/prompt.txt") {
		t.Error("Expected 'GITHUB_AW_PROMPT: /tmp/aw-prompts/prompt.txt' in prompt generation step")
	}

	// Check that GITHUB_AW_PROMPT_CONTENT environment variable is included
	if !strings.Contains(output, "GITHUB_AW_PROMPT_CONTENT:") {
		t.Error("Expected 'GITHUB_AW_PROMPT_CONTENT:' environment variable in prompt generation step")
	}

	// Check that it's a single step named "Create and print prompt"
	if !strings.Contains(output, "name: Create and print prompt") {
		t.Error("Expected 'name: Create and print prompt' in prompt generation step")
	}

	// Check that there's no separate "Print prompt to step summary" step
	if strings.Contains(output, "name: Print prompt to step summary") {
		t.Error("Should not contain separate 'Print prompt to step summary' step")
	}

	// Check that there's no shell commands for cat/echo
	if strings.Contains(output, "cat > $GITHUB_AW_PROMPT") {
		t.Error("Should not contain shell commands for creating prompt file")
	}

	if strings.Contains(output, "echo \"## Generated Prompt\" >> $GITHUB_STEP_SUMMARY") {
		t.Error("Should not contain shell commands for step summary")
	}
}

func TestGeneratePromptWithSafeOutputs(t *testing.T) {
	compiler := NewCompiler(false, "", "test")

	data := &WorkflowData{
		MarkdownContent: "Test workflow content",
		SafeOutputs: &SafeOutputsConfig{
			MissingTool: &MissingToolConfig{Max: 10},
		},
	}

	var yaml strings.Builder
	compiler.generatePrompt(&yaml, data)

	output := yaml.String()

	// Check that GITHUB_AW_SAFE_OUTPUTS environment variable is included when SafeOutputs is configured
	if !strings.Contains(output, "GITHUB_AW_SAFE_OUTPUTS: ${{ env.GITHUB_AW_SAFE_OUTPUTS }}") {
		t.Error("Expected 'GITHUB_AW_SAFE_OUTPUTS' environment variable when SafeOutputs is configured")
	}

	// Check that the prompt content includes safe outputs instructions
	if !strings.Contains(output, "GITHUB_AW_PROMPT_CONTENT:") {
		t.Error("Expected prompt content to be passed as environment variable")
	}
}

func TestGeneratePromptWithCacheMemory(t *testing.T) {
	compiler := NewCompiler(false, "", "test")

	data := &WorkflowData{
		MarkdownContent: "Test workflow content",
		CacheMemoryConfig: &CacheMemoryConfig{
			Enabled: true,
		},
	}

	var yaml strings.Builder
	compiler.generatePrompt(&yaml, data)

	output := yaml.String()

	// Check that the step uses JavaScript action
	if !strings.Contains(output, "uses: actions/github-script@v8") {
		t.Error("Expected JavaScript action to be used")
	}

	// The cache memory instructions should be included in the prompt content
	// Since we pass it as GITHUB_AW_PROMPT_CONTENT, we can't easily check the content here
	// but we can ensure the structure is correct
	if !strings.Contains(output, "GITHUB_AW_PROMPT_CONTENT:") {
		t.Error("Expected prompt content environment variable")
	}
}

func TestBuildPromptContentBasic(t *testing.T) {
	compiler := NewCompiler(false, "", "test")

	data := &WorkflowData{
		MarkdownContent: "This is test content\nWith multiple lines",
	}

	content := compiler.buildPromptContent(data)

	if !strings.Contains(content, "This is test content") {
		t.Error("Expected basic markdown content in prompt")
	}

	if !strings.Contains(content, "With multiple lines") {
		t.Error("Expected multi-line content to be preserved")
	}
}

func TestBuildPromptContentWithCacheMemory(t *testing.T) {
	compiler := NewCompiler(false, "", "test")

	data := &WorkflowData{
		MarkdownContent: "Test content",
		CacheMemoryConfig: &CacheMemoryConfig{
			Enabled: true,
		},
	}

	content := compiler.buildPromptContent(data)

	if !strings.Contains(content, "## Cache Folder Available") {
		t.Error("Expected cache folder section in prompt content")
	}

	if !strings.Contains(content, "/tmp/cache-memory/") {
		t.Error("Expected cache folder path in prompt content")
	}

	if !strings.Contains(content, "persistent cache folder") {
		t.Error("Expected cache folder description in prompt content")
	}
}

func TestBuildPromptContentWithSafeOutputs(t *testing.T) {
	compiler := NewCompiler(false, "", "test")

	data := &WorkflowData{
		MarkdownContent: "Test content",
		SafeOutputs: &SafeOutputsConfig{
			CreateIssues: &CreateIssuesConfig{
				TitlePrefix: "[Test]",
				Max:         5,
			},
			MissingTool: &MissingToolConfig{Max: 10},
		},
	}

	content := compiler.buildPromptContent(data)

	if !strings.Contains(content, "Creating an Issue") {
		t.Error("Expected 'Creating an Issue' in prompt content")
	}

	if !strings.Contains(content, "Reporting Missing Tools or Functionality") {
		t.Error("Expected 'Reporting Missing Tools or Functionality' in prompt content")
	}

	if !strings.Contains(content, "**IMPORTANT**: To do the actions mentioned") {
		t.Error("Expected important note about safe-outputs tools")
	}

	if !strings.Contains(content, "use the create-issue tool from the safe-outputs MCP") {
		t.Error("Expected instructions for creating issues")
	}
}

func TestBuildPromptContentRemovesXMLComments(t *testing.T) {
	compiler := NewCompiler(false, "", "test")

	data := &WorkflowData{
		MarkdownContent: "Test content\n<!-- This is a comment -->\nMore content",
	}

	content := compiler.buildPromptContent(data)

	if strings.Contains(content, "<!-- This is a comment -->") {
		t.Error("Expected XML comments to be removed from prompt content")
	}

	if !strings.Contains(content, "Test content") {
		t.Error("Expected non-comment content to be preserved")
	}

	if !strings.Contains(content, "More content") {
		t.Error("Expected content after comments to be preserved")
	}
}

func TestGeneratePromptEscapesSpecialCharacters(t *testing.T) {
	compiler := NewCompiler(false, "", "test")

	data := &WorkflowData{
		MarkdownContent: "Content with \"quotes\" and \nnewlines",
	}

	var yaml strings.Builder
	compiler.generatePrompt(&yaml, data)

	output := yaml.String()

	// Check that quotes are escaped in YAML
	if !strings.Contains(output, "\\\"") {
		t.Error("Expected quotes to be escaped in YAML environment variable")
	}

	// Check that newlines are escaped as \\n
	if !strings.Contains(output, "\\n") {
		t.Error("Expected newlines to be escaped in YAML environment variable")
	}
}
