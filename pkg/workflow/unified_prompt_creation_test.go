package workflow

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGenerateUnifiedPromptCreationStep_OrderingBuiltinFirst tests that built-in prompts
// are prepended (written first) before user prompt content
func TestGenerateUnifiedPromptCreationStep_OrderingBuiltinFirst(t *testing.T) {
	compiler := &Compiler{
		trialMode:            false,
		trialLogicalRepoSlug: "",
	}

	// Create data with multiple built-in sections
	data := &WorkflowData{
		ParsedTools: NewTools(map[string]any{
			"playwright": true,
		}),
		SafeOutputs: &SafeOutputsConfig{
			CreateIssues: &CreateIssuesConfig{},
		},
	}

	// Collect built-in sections
	builtinSections := compiler.collectPromptSections(data)

	// Create a simple user prompt
	userPromptChunks := []string{"# User Prompt\n\nThis is the user's task."}

	var yaml strings.Builder
	compiler.generateUnifiedPromptCreationStep(&yaml, builtinSections, userPromptChunks, nil, data)

	output := yaml.String()

	// Find positions of different prompt sections in the output
	tempFolderPos := strings.Index(output, "temp_folder_prompt.md")
	playwrightPos := strings.Index(output, "playwright_prompt.md")
	safeOutputsPos := strings.Index(output, "<safe-outputs>")
	userPromptPos := strings.Index(output, "# User Prompt")

	// Verify all sections are present
	require.NotEqual(t, -1, tempFolderPos, "Temp folder prompt should be present")
	require.NotEqual(t, -1, playwrightPos, "Playwright prompt should be present")
	require.NotEqual(t, -1, safeOutputsPos, "Safe outputs prompt should be present")
	require.NotEqual(t, -1, userPromptPos, "User prompt should be present")

	// Verify ordering: built-in prompts come before user prompt
	assert.Less(t, tempFolderPos, userPromptPos, "Temp folder prompt should come before user prompt")
	assert.Less(t, playwrightPos, userPromptPos, "Playwright prompt should come before user prompt")
	assert.Less(t, safeOutputsPos, userPromptPos, "Safe outputs prompt should come before user prompt")
}

// TestGenerateUnifiedPromptCreationStep_SubstitutionWithBuiltinExpressions tests that
// expressions in built-in prompts (like GitHub context) are properly extracted and substituted
func TestGenerateUnifiedPromptCreationStep_SubstitutionWithBuiltinExpressions(t *testing.T) {
	compiler := &Compiler{
		trialMode:            false,
		trialLogicalRepoSlug: "",
	}

	// Create data with GitHub tool enabled (which includes GitHub context prompt with expressions)
	data := &WorkflowData{
		ParsedTools: NewTools(map[string]any{
			"github": true,
		}),
	}

	// Collect built-in sections (should include GitHub context with expressions)
	builtinSections := compiler.collectPromptSections(data)

	// Create a simple user prompt
	userPromptChunks := []string{"# User Prompt"}

	var yaml strings.Builder
	compiler.generateUnifiedPromptCreationStep(&yaml, builtinSections, userPromptChunks, nil, data)

	output := yaml.String()

	// Verify environment variables from GitHub context prompt are declared
	assert.Contains(t, output, "GH_AW_GITHUB_REPOSITORY:", "Should have GH_AW_GITHUB_REPOSITORY env var")
	assert.Contains(t, output, "${{ github.repository }}", "Should have github.repository expression")

	// Verify environment variables section comes before run section
	envPos := strings.Index(output, "env:")
	runPos := strings.Index(output, "run: |")
	assert.Less(t, envPos, runPos, "env section should come before run section")
}

// TestGenerateUnifiedPromptCreationStep_SubstitutionWithUserExpressions tests that
// expressions in user prompt are properly handled alongside built-in prompt expressions
func TestGenerateUnifiedPromptCreationStep_SubstitutionWithUserExpressions(t *testing.T) {
	compiler := &Compiler{
		trialMode:            false,
		trialLogicalRepoSlug: "",
	}

	// Create data with a built-in section
	data := &WorkflowData{
		ParsedTools: NewTools(map[string]any{}),
	}

	// Collect built-in sections (minimal - just temp folder)
	builtinSections := compiler.collectPromptSections(data)

	// Create user prompt with expressions
	userMarkdown := "Repository: ${{ github.repository }}\nActor: ${{ github.actor }}"

	// Extract expressions from user prompt
	extractor := NewExpressionExtractor()
	expressionMappings, err := extractor.ExtractExpressions(userMarkdown)
	require.NoError(t, err)
	require.Len(t, expressionMappings, 2, "Should extract 2 expressions from user prompt")

	// Replace expressions with placeholders
	userPromptWithPlaceholders := extractor.ReplaceExpressionsWithEnvVars(userMarkdown)
	userPromptChunks := []string{userPromptWithPlaceholders}

	var yaml strings.Builder
	compiler.generateUnifiedPromptCreationStep(&yaml, builtinSections, userPromptChunks, expressionMappings, data)

	output := yaml.String()

	// Verify environment variables from user expressions are declared
	assert.Contains(t, output, "GH_AW_GITHUB_REPOSITORY:", "Should have GH_AW_GITHUB_REPOSITORY env var")
	assert.Contains(t, output, "GH_AW_GITHUB_ACTOR:", "Should have GH_AW_GITHUB_ACTOR env var")
	assert.Contains(t, output, "${{ github.repository }}", "Should have github.repository expression value")
	assert.Contains(t, output, "${{ github.actor }}", "Should have github.actor expression value")

	// Verify substitution step is generated
	assert.Contains(t, output, "Substitute placeholders", "Should have placeholder substitution step")
}

// TestGenerateUnifiedPromptCreationStep_MultipleUserChunks tests that multiple
// user prompt chunks are properly appended after built-in prompts
func TestGenerateUnifiedPromptCreationStep_MultipleUserChunks(t *testing.T) {
	compiler := &Compiler{
		trialMode:            false,
		trialLogicalRepoSlug: "",
	}

	// Create data with minimal built-in sections
	data := &WorkflowData{
		ParsedTools: NewTools(map[string]any{}),
	}

	// Collect built-in sections
	builtinSections := compiler.collectPromptSections(data)

	// Create multiple user prompt chunks
	userPromptChunks := []string{
		"# Part 1\n\nFirst chunk of user prompt.",
		"# Part 2\n\nSecond chunk of user prompt.",
		"# Part 3\n\nThird chunk of user prompt.",
	}

	var yaml strings.Builder
	compiler.generateUnifiedPromptCreationStep(&yaml, builtinSections, userPromptChunks, nil, data)

	output := yaml.String()

	// Count PROMPT_EOF markers
	// With system tags:
	// - 2 for opening <system> tag
	// - 2 for closing </system> tag
	// - 2 per user chunk
	eofCount := strings.Count(output, "PROMPT_EOF")
	expectedEOFCount := 4 + (len(userPromptChunks) * 2) // 4 for system tags, 2 per user chunk
	assert.Equal(t, expectedEOFCount, eofCount, "Should have correct number of PROMPT_EOF markers")

	// Verify all user chunks are present and in order
	part1Pos := strings.Index(output, "# Part 1")
	part2Pos := strings.Index(output, "# Part 2")
	part3Pos := strings.Index(output, "# Part 3")

	require.NotEqual(t, -1, part1Pos, "Part 1 should be present")
	require.NotEqual(t, -1, part2Pos, "Part 2 should be present")
	require.NotEqual(t, -1, part3Pos, "Part 3 should be present")

	assert.Less(t, part1Pos, part2Pos, "Part 1 should come before Part 2")
	assert.Less(t, part2Pos, part3Pos, "Part 2 should come before Part 3")

	// Verify built-in prompt comes before all user chunks
	tempFolderPos := strings.Index(output, "temp_folder_prompt.md")
	require.NotEqual(t, -1, tempFolderPos, "Temp folder prompt should be present")
	assert.Less(t, tempFolderPos, part1Pos, "Built-in prompt should come before user prompt chunks")

	// Verify system tags wrap built-in prompts
	systemOpenPos := strings.Index(output, "<system>")
	systemClosePos := strings.Index(output, "</system>")
	require.NotEqual(t, -1, systemOpenPos, "Opening system tag should be present")
	require.NotEqual(t, -1, systemClosePos, "Closing system tag should be present")
	assert.Less(t, systemOpenPos, tempFolderPos, "System tag should open before built-in prompts")
	assert.Less(t, tempFolderPos, systemClosePos, "System tag should close after built-in prompts")
	assert.Less(t, systemClosePos, part1Pos, "System tag should close before user prompt")
}

// TestGenerateUnifiedPromptCreationStep_CombinedExpressions tests that expressions
// from both built-in prompts and user prompts are properly combined and substituted
func TestGenerateUnifiedPromptCreationStep_CombinedExpressions(t *testing.T) {
	compiler := &Compiler{
		trialMode:            false,
		trialLogicalRepoSlug: "",
	}

	// Create data with GitHub tool enabled (has built-in expressions)
	data := &WorkflowData{
		ParsedTools: NewTools(map[string]any{
			"github": true,
		}),
	}

	// Collect built-in sections (includes GitHub context with expressions)
	builtinSections := compiler.collectPromptSections(data)

	// Create user prompt with different expressions
	userMarkdown := "Run ID: ${{ github.run_id }}\nWorkspace: ${{ github.workspace }}"

	// Extract expressions from user prompt
	extractor := NewExpressionExtractor()
	expressionMappings, err := extractor.ExtractExpressions(userMarkdown)
	require.NoError(t, err)

	// Replace expressions with placeholders
	userPromptWithPlaceholders := extractor.ReplaceExpressionsWithEnvVars(userMarkdown)
	userPromptChunks := []string{userPromptWithPlaceholders}

	var yaml strings.Builder
	compiler.generateUnifiedPromptCreationStep(&yaml, builtinSections, userPromptChunks, expressionMappings, data)

	output := yaml.String()

	// Verify environment variables from both built-in and user prompts are present
	// From built-in GitHub context prompt
	assert.Contains(t, output, "GH_AW_GITHUB_REPOSITORY:", "Should have built-in env var")
	assert.Contains(t, output, "GH_AW_GITHUB_ACTOR:", "Should have built-in env var")

	// From user prompt
	assert.Contains(t, output, "GH_AW_GITHUB_RUN_ID:", "Should have user prompt env var")
	assert.Contains(t, output, "GH_AW_GITHUB_WORKSPACE:", "Should have user prompt env var")

	// Verify all environment variables are sorted (after GH_AW_PROMPT)
	envSection := output[strings.Index(output, "env:"):strings.Index(output, "run: |")]
	lines := strings.Split(envSection, "\n")

	var envVarNames []string
	for _, line := range lines {
		if strings.Contains(line, "GH_AW_") && !strings.Contains(line, "GH_AW_PROMPT:") && !strings.Contains(line, "GH_AW_SAFE_OUTPUTS:") {
			// Extract variable name
			parts := strings.SplitN(strings.TrimSpace(line), ":", 2)
			if len(parts) == 2 {
				envVarNames = append(envVarNames, parts[0])
			}
		}
	}

	// Check that variables are sorted
	for i := 1; i < len(envVarNames); i++ {
		assert.LessOrEqual(t, envVarNames[i-1], envVarNames[i],
			"Environment variables should be sorted: %s should come before or equal to %s",
			envVarNames[i-1], envVarNames[i])
	}
}

// TestGenerateUnifiedPromptCreationStep_NoAppendSteps tests that the old
// "Append context instructions" step is not generated
func TestGenerateUnifiedPromptCreationStep_NoAppendSteps(t *testing.T) {
	compiler := &Compiler{
		trialMode:            false,
		trialLogicalRepoSlug: "",
	}

	data := &WorkflowData{
		ParsedTools: NewTools(map[string]any{
			"playwright": true,
			"github":     true,
		}),
		SafeOutputs: &SafeOutputsConfig{
			CreateIssues: &CreateIssuesConfig{},
		},
	}

	builtinSections := compiler.collectPromptSections(data)

	// Create user prompt with expressions to ensure substitution step is generated
	userMarkdown := "Run ID: ${{ github.run_id }}"
	extractor := NewExpressionExtractor()
	expressionMappings, _ := extractor.ExtractExpressions(userMarkdown)
	userPromptWithPlaceholders := extractor.ReplaceExpressionsWithEnvVars(userMarkdown)
	userPromptChunks := []string{userPromptWithPlaceholders}

	var yaml strings.Builder
	compiler.generateUnifiedPromptCreationStep(&yaml, builtinSections, userPromptChunks, expressionMappings, data)

	output := yaml.String()

	// Verify there's only the unified step and substitution step (not old separate steps)
	stepNameCount := strings.Count(output, "- name:")
	assert.Equal(t, 2, stepNameCount, "Should have exactly 2 steps: Create prompt and Substitute placeholders")

	// Verify the old append step name is not present
	assert.NotContains(t, output, "Append context instructions to prompt",
		"Should not have old 'Append context instructions' step")
	assert.NotContains(t, output, "Append prompt (part",
		"Should not have old 'Append prompt (part N)' steps")
}

// TestGenerateUnifiedPromptCreationStep_FirstContentUsesCreate tests that
// the first content uses ">" (create/overwrite) and subsequent content uses ">>" (append)
func TestGenerateUnifiedPromptCreationStep_FirstContentUsesCreate(t *testing.T) {
	compiler := &Compiler{
		trialMode:            false,
		trialLogicalRepoSlug: "",
	}

	data := &WorkflowData{
		ParsedTools: NewTools(map[string]any{}),
	}

	builtinSections := compiler.collectPromptSections(data)
	userPromptChunks := []string{"# User Prompt"}

	var yaml strings.Builder
	compiler.generateUnifiedPromptCreationStep(&yaml, builtinSections, userPromptChunks, nil, data)

	output := yaml.String()

	// Find the first cat command (should use > for create)
	firstCatPos := strings.Index(output, `cat "`)
	require.NotEqual(t, -1, firstCatPos, "Should have cat command")

	// Extract the line containing the first cat command
	firstCatLine := output[firstCatPos : firstCatPos+strings.Index(output[firstCatPos:], "\n")]

	// Verify it uses > (create mode)
	assert.Contains(t, firstCatLine, `> "$GH_AW_PROMPT"`,
		"First content should use > (create mode): %s", firstCatLine)

	// Find subsequent cat commands (should use >> for append)
	remainingOutput := output[firstCatPos+len(firstCatLine):]
	if strings.Contains(remainingOutput, `cat "`) || strings.Contains(remainingOutput, "cat << 'PROMPT_EOF'") {
		// Verify subsequent operations use >> (append mode)
		assert.Contains(t, remainingOutput, `>> "$GH_AW_PROMPT"`,
			"Subsequent content should use >> (append mode)")
	}
}

// TestGenerateUnifiedPromptCreationStep_SystemTags tests that built-in prompts
// are wrapped in <system> XML tags
func TestGenerateUnifiedPromptCreationStep_SystemTags(t *testing.T) {
	compiler := &Compiler{
		trialMode:            false,
		trialLogicalRepoSlug: "",
	}

	// Create data with multiple built-in sections
	data := &WorkflowData{
		ParsedTools: NewTools(map[string]any{
			"playwright": true,
		}),
		SafeOutputs: &SafeOutputsConfig{
			CreateIssues: &CreateIssuesConfig{},
		},
	}

	// Collect built-in sections
	builtinSections := compiler.collectPromptSections(data)

	// Create user prompt
	userPromptChunks := []string{"# User Task\n\nThis is the user's task."}

	var yaml strings.Builder
	compiler.generateUnifiedPromptCreationStep(&yaml, builtinSections, userPromptChunks, nil, data)

	output := yaml.String()

	// Verify system tags are present
	assert.Contains(t, output, "<system>", "Should have opening system tag")
	assert.Contains(t, output, "</system>", "Should have closing system tag")

	// Verify system tags wrap built-in content
	systemOpenPos := strings.Index(output, "<system>")
	systemClosePos := strings.Index(output, "</system>")

	// Find positions of built-in content
	tempFolderPos := strings.Index(output, "temp_folder_prompt.md")
	playwrightPos := strings.Index(output, "playwright_prompt.md")
	safeOutputsPos := strings.Index(output, "<safe-outputs>")

	// Find position of user content
	userTaskPos := strings.Index(output, "# User Task")

	// Verify ordering: <system> -> built-in content -> </system> -> user content
	require.NotEqual(t, -1, systemOpenPos, "Opening system tag should be present")
	require.NotEqual(t, -1, systemClosePos, "Closing system tag should be present")
	require.NotEqual(t, -1, tempFolderPos, "Temp folder should be present")
	require.NotEqual(t, -1, userTaskPos, "User task should be present")

	assert.Less(t, systemOpenPos, tempFolderPos, "System tag should open before temp folder")
	assert.Less(t, tempFolderPos, playwrightPos, "Temp folder should come before playwright")
	assert.Less(t, playwrightPos, safeOutputsPos, "Playwright should come before safe outputs")
	assert.Less(t, safeOutputsPos, systemClosePos, "Safe outputs should come before system close tag")
	assert.Less(t, systemClosePos, userTaskPos, "System tag should close before user content")
}
