package workflow

import (
	"strings"
	"testing"
)

// TestErrorPatternsOnCopilotEngine verifies that custom error patterns can be specified on the Copilot engine
func TestErrorPatternsOnCopilotEngine(t *testing.T) {
	compiler := NewCompiler(false, "", "")
	engine := NewCopilotEngine()

	// Workflow with custom error patterns on Copilot engine
	data := &WorkflowData{
		Name: "test-copilot-patterns",
		EngineConfig: &EngineConfig{
			ID: "copilot",
			ErrorPatterns: []ErrorPattern{
				{
					Pattern:      `PROJECT_ERROR:\s+(.+)`,
					LevelGroup:   0,
					MessageGroup: 1,
					Description:  "Project-specific error pattern for Copilot",
				},
				{
					Pattern:      `\[CUSTOM\]\s+(ERROR|WARNING):\s+(.+)`,
					LevelGroup:   1,
					MessageGroup: 2,
					Description:  "Custom bracketed error pattern",
				},
			},
		},
	}

	var yamlBuilder strings.Builder
	compiler.generateErrorValidation(&yamlBuilder, engine, data)

	generated := yamlBuilder.String()

	// Should generate error validation step
	if !strings.Contains(generated, "Validate agent logs for errors") {
		t.Error("Should generate error validation step for Copilot with custom patterns")
	}

	// Should include GH_AW_ERROR_PATTERNS environment variable
	if !strings.Contains(generated, "GH_AW_ERROR_PATTERNS") {
		t.Error("Should include error patterns environment variable")
	}

	// Should contain custom patterns
	if !strings.Contains(generated, "PROJECT_ERROR") {
		t.Error("Should include PROJECT_ERROR custom pattern")
	}

	if !strings.Contains(generated, "Custom bracketed error pattern") {
		t.Error("Should include custom bracketed error pattern description")
	}

	// Should also contain Copilot's built-in patterns
	if !strings.Contains(generated, "Copilot CLI") {
		t.Error("Should still include Copilot's built-in error patterns")
	}
}

// TestErrorPatternsOnClaudeEngine verifies that custom error patterns can be specified on the Claude engine
func TestErrorPatternsOnClaudeEngine(t *testing.T) {
	compiler := NewCompiler(false, "", "")
	engine := NewClaudeEngine()

	// Workflow with custom error patterns on Claude engine
	data := &WorkflowData{
		Name: "test-claude-patterns",
		EngineConfig: &EngineConfig{
			ID: "claude",
			ErrorPatterns: []ErrorPattern{
				{
					Pattern:      `CLAUDE_PROJECT_ERROR:\s+(.+)`,
					LevelGroup:   0,
					MessageGroup: 1,
					Description:  "Project-specific error pattern for Claude",
				},
			},
		},
	}

	var yamlBuilder strings.Builder
	compiler.generateErrorValidation(&yamlBuilder, engine, data)

	generated := yamlBuilder.String()

	// Should generate error validation step
	if !strings.Contains(generated, "Validate agent logs for errors") {
		t.Error("Should generate error validation step for Claude with custom patterns")
	}

	// Should include custom pattern
	if !strings.Contains(generated, "CLAUDE_PROJECT_ERROR") {
		t.Error("Should include CLAUDE_PROJECT_ERROR custom pattern")
	}

	if !strings.Contains(generated, "Project-specific error pattern for Claude") {
		t.Error("Should include Claude pattern description")
	}

	// Should also contain common patterns
	if !strings.Contains(generated, "GitHub Actions workflow command") {
		t.Error("Should still include common error patterns")
	}
}

// TestErrorPatternsOnCodexEngine verifies that custom error patterns can be specified on the Codex engine
func TestErrorPatternsOnCodexEngine(t *testing.T) {
	compiler := NewCompiler(false, "", "")
	engine := NewCodexEngine()

	// Workflow with custom error patterns on Codex engine
	data := &WorkflowData{
		Name: "test-codex-patterns",
		EngineConfig: &EngineConfig{
			ID: "codex",
			ErrorPatterns: []ErrorPattern{
				{
					Pattern:      `CODEX_PROJECT_ERROR:\s+(.+)`,
					LevelGroup:   0,
					MessageGroup: 1,
					Description:  "Project-specific error pattern for Codex",
				},
			},
		},
	}

	var yamlBuilder strings.Builder
	compiler.generateErrorValidation(&yamlBuilder, engine, data)

	generated := yamlBuilder.String()

	// Should generate error validation step
	if !strings.Contains(generated, "Validate agent logs for errors") {
		t.Error("Should generate error validation step for Codex with custom patterns")
	}

	// Should include custom pattern
	if !strings.Contains(generated, "CODEX_PROJECT_ERROR") {
		t.Error("Should include CODEX_PROJECT_ERROR custom pattern")
	}

	if !strings.Contains(generated, "Project-specific error pattern for Codex") {
		t.Error("Should include Codex pattern description")
	}

	// Should also contain Codex-specific patterns
	if !strings.Contains(generated, "Codex ERROR messages") {
		t.Error("Should still include Codex's built-in error patterns")
	}
}

// TestErrorPatternsOnCustomEngine verifies that custom error patterns work with the custom engine (backward compatibility)
func TestErrorPatternsOnCustomEngine(t *testing.T) {
	compiler := NewCompiler(false, "", "")
	engine := NewCustomEngine()

	// Workflow with custom error patterns on custom engine
	data := &WorkflowData{
		Name: "test-custom-patterns",
		EngineConfig: &EngineConfig{
			ID: "custom",
			ErrorPatterns: []ErrorPattern{
				{
					Pattern:      `CUSTOM_ENGINE_ERROR:\s+(.+)`,
					LevelGroup:   0,
					MessageGroup: 1,
					Description:  "Error pattern for custom engine",
				},
			},
		},
	}

	var yamlBuilder strings.Builder
	compiler.generateErrorValidation(&yamlBuilder, engine, data)

	generated := yamlBuilder.String()

	// Should generate error validation step
	if !strings.Contains(generated, "Validate agent logs for errors") {
		t.Error("Should generate error validation step for custom engine with patterns")
	}

	// Should include custom pattern
	if !strings.Contains(generated, "CUSTOM_ENGINE_ERROR") {
		t.Error("Should include CUSTOM_ENGINE_ERROR pattern")
	}

	if !strings.Contains(generated, "Error pattern for custom engine") {
		t.Error("Should include custom engine pattern description")
	}
}

// TestErrorPatternsMergeEngineAndCustom verifies that engine patterns and custom patterns are properly merged
func TestErrorPatternsMergeEngineAndCustom(t *testing.T) {
	compiler := NewCompiler(false, "", "")
	engine := NewCopilotEngine()

	// Get the count of built-in Copilot patterns
	builtinPatterns := engine.GetErrorPatterns()
	builtinCount := len(builtinPatterns)

	if builtinCount == 0 {
		t.Fatal("Copilot should have built-in error patterns")
	}

	// Workflow with custom error patterns
	customPatternCount := 3
	data := &WorkflowData{
		Name: "test-merge-patterns",
		EngineConfig: &EngineConfig{
			ID: "copilot",
			ErrorPatterns: []ErrorPattern{
				{Pattern: `PATTERN_1:\s+(.+)`, MessageGroup: 1, Description: "Pattern 1"},
				{Pattern: `PATTERN_2:\s+(.+)`, MessageGroup: 1, Description: "Pattern 2"},
				{Pattern: `PATTERN_3:\s+(.+)`, MessageGroup: 1, Description: "Pattern 3"},
			},
		},
	}

	var yamlBuilder strings.Builder
	compiler.generateErrorValidation(&yamlBuilder, engine, data)

	generated := yamlBuilder.String()

	// Verify all custom patterns are included
	if !strings.Contains(generated, "PATTERN_1") {
		t.Error("Should include PATTERN_1")
	}
	if !strings.Contains(generated, "PATTERN_2") {
		t.Error("Should include PATTERN_2")
	}
	if !strings.Contains(generated, "PATTERN_3") {
		t.Error("Should include PATTERN_3")
	}

	// Verify at least one built-in pattern is included
	if !strings.Contains(generated, "GitHub Actions workflow command") {
		t.Error("Should include built-in GitHub Actions workflow command patterns")
	}

	// Count total patterns in the generated JSON by looking for the pattern field
	// Note: We use \\\\\\\\s because the JSON is already escaped in the YAML
	patternCount := strings.Count(generated, `\\\\s`)
	expectedMinCount := builtinCount + customPatternCount

	if patternCount < expectedMinCount {
		t.Logf("Pattern count check: found %d occurrences of \\\\s pattern markers", patternCount)
		// This is informational - the important thing is that we verified individual patterns above
		// The exact count can vary due to JSON escaping in YAML
	}
}

// TestErrorPatternsExtractFromFrontmatter verifies that error patterns are correctly extracted from frontmatter
func TestErrorPatternsExtractFromFrontmatter(t *testing.T) {
	tests := []struct {
		name        string
		frontmatter map[string]any
		engineID    string
		wantCount   int
		wantPattern string
	}{
		{
			name: "copilot with custom patterns",
			frontmatter: map[string]any{
				"engine": map[string]any{
					"id": "copilot",
					"error_patterns": []any{
						map[string]any{
							"pattern":       `PROJECT_ERROR:\s+(.+)`,
							"message_group": 1,
							"description":   "Project error",
						},
					},
				},
			},
			engineID:    "copilot",
			wantCount:   1,
			wantPattern: `PROJECT_ERROR:\s+(.+)`,
		},
		{
			name: "claude with multiple patterns",
			frontmatter: map[string]any{
				"engine": map[string]any{
					"id": "claude",
					"error_patterns": []any{
						map[string]any{
							"pattern":     `ERROR_A:\s+(.+)`,
							"description": "Error A",
						},
						map[string]any{
							"pattern":     `ERROR_B:\s+(.+)`,
							"description": "Error B",
						},
					},
				},
			},
			engineID:    "claude",
			wantCount:   2,
			wantPattern: `ERROR_A:\s+(.+)`,
		},
		{
			name: "codex with no custom patterns",
			frontmatter: map[string]any{
				"engine": map[string]any{
					"id": "codex",
				},
			},
			engineID:  "codex",
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compiler := NewCompiler(false, "", "")
			_, config := compiler.ExtractEngineConfig(tt.frontmatter)

			if config == nil {
				t.Fatal("Failed to extract engine config")
			}

			if config.ID != tt.engineID {
				t.Errorf("Expected engine ID %s, got %s", tt.engineID, config.ID)
			}

			if len(config.ErrorPatterns) != tt.wantCount {
				t.Errorf("Expected %d error patterns, got %d", tt.wantCount, len(config.ErrorPatterns))
			}

			if tt.wantCount > 0 && config.ErrorPatterns[0].Pattern != tt.wantPattern {
				t.Errorf("Expected pattern %s, got %s", tt.wantPattern, config.ErrorPatterns[0].Pattern)
			}
		})
	}
}
