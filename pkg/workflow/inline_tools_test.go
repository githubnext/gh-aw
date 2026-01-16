package workflow

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseInlineTools(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected []InlineToolConfig
	}{
		{
			name:     "nil input",
			input:    nil,
			expected: nil,
		},
		{
			name:     "empty array",
			input:    []any{},
			expected: nil,
		},
		{
			name: "single tool with all fields",
			input: []any{
				map[string]any{
					"name":        "create_deployment",
					"description": "Create deployment to specified environment",
					"parameters": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"environment": map[string]any{
								"type": "string",
								"enum": []any{"staging", "production"},
							},
						},
						"required": []any{"environment"},
					},
					"implementation": "const { environment } = params; return { status: 'ok' };",
				},
			},
			expected: []InlineToolConfig{
				{
					Name:        "create_deployment",
					Description: "Create deployment to specified environment",
					Parameters: map[string]any{
						"type": "object",
						"properties": map[string]any{
							"environment": map[string]any{
								"type": "string",
								"enum": []any{"staging", "production"},
							},
						},
						"required": []any{"environment"},
					},
					Implementation: "const { environment } = params; return { status: 'ok' };",
				},
			},
		},
		{
			name: "tool without implementation",
			input: []any{
				map[string]any{
					"name":        "read_context",
					"description": "Read workflow context",
					"parameters": map[string]any{
						"type": "object",
					},
				},
			},
			expected: []InlineToolConfig{
				{
					Name:        "read_context",
					Description: "Read workflow context",
					Parameters: map[string]any{
						"type": "object",
					},
					Implementation: "",
				},
			},
		},
		{
			name: "multiple tools",
			input: []any{
				map[string]any{
					"name":        "tool1",
					"description": "First tool",
				},
				map[string]any{
					"name":        "tool2",
					"description": "Second tool",
				},
			},
			expected: []InlineToolConfig{
				{
					Name:        "tool1",
					Description: "First tool",
				},
				{
					Name:        "tool2",
					Description: "Second tool",
				},
			},
		},
		{
			name:     "invalid input type (not array)",
			input:    "not an array",
			expected: nil,
		},
		{
			name: "skip invalid tool (not object)",
			input: []any{
				"invalid",
				map[string]any{
					"name":        "valid_tool",
					"description": "Valid tool",
				},
			},
			expected: []InlineToolConfig{
				{
					Name:        "valid_tool",
					Description: "Valid tool",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseInlineTools(tt.input)

			if tt.expected == nil {
				assert.Nil(t, result)
			} else {
				require.NotNil(t, result)
				assert.Len(t, result, len(tt.expected))

				for i, expectedTool := range tt.expected {
					assert.Equal(t, expectedTool.Name, result[i].Name)
					assert.Equal(t, expectedTool.Description, result[i].Description)
					assert.Equal(t, expectedTool.Implementation, result[i].Implementation)

					if expectedTool.Parameters != nil {
						assert.NotNil(t, result[i].Parameters)
					}
				}
			}
		})
	}
}

func TestValidateInlineToolDefinition(t *testing.T) {
	tests := []struct {
		name      string
		tool      InlineToolConfig
		index     int
		shouldErr bool
		errMsg    string
	}{
		{
			name: "valid tool with all fields",
			tool: InlineToolConfig{
				Name:        "create_deployment",
				Description: "Create deployment to specified environment",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"environment": map[string]any{
							"type": "string",
						},
					},
				},
				Implementation: "return { status: 'ok' };",
			},
			index:     0,
			shouldErr: false,
		},
		{
			name: "valid tool without parameters",
			tool: InlineToolConfig{
				Name:           "simple_tool",
				Description:    "A simple tool without parameters",
				Implementation: "return { result: true };",
			},
			index:     0,
			shouldErr: false,
		},
		{
			name: "valid tool without implementation",
			tool: InlineToolConfig{
				Name:        "runtime_tool",
				Description: "Tool with runtime-provided implementation",
			},
			index:     0,
			shouldErr: false,
		},
		{
			name: "missing name",
			tool: InlineToolConfig{
				Description: "Tool without name",
			},
			index:     0,
			shouldErr: true,
			errMsg:    "missing required field 'name'",
		},
		{
			name: "empty name",
			tool: InlineToolConfig{
				Name:        "",
				Description: "Tool with empty name",
			},
			index:     0,
			shouldErr: true,
			errMsg:    "missing required field 'name'",
		},
		{
			name: "invalid name (starts with number)",
			tool: InlineToolConfig{
				Name:        "123tool",
				Description: "Tool with invalid name",
			},
			index:     0,
			shouldErr: true,
			errMsg:    "invalid name",
		},
		{
			name: "invalid name (special characters)",
			tool: InlineToolConfig{
				Name:        "tool@name",
				Description: "Tool with invalid name",
			},
			index:     0,
			shouldErr: true,
			errMsg:    "invalid name",
		},
		{
			name: "valid name with underscore and hyphen",
			tool: InlineToolConfig{
				Name:        "my_tool-name",
				Description: "Tool with valid name containing underscore and hyphen",
			},
			index:     0,
			shouldErr: false,
		},
		{
			name: "missing description",
			tool: InlineToolConfig{
				Name: "tool_name",
			},
			index:     0,
			shouldErr: true,
			errMsg:    "missing required field 'description'",
		},
		{
			name: "too short description",
			tool: InlineToolConfig{
				Name:        "tool_name",
				Description: "Short",
			},
			index:     0,
			shouldErr: true,
			errMsg:    "description must be at least 10 characters",
		},
		{
			name: "invalid parameters schema (missing type)",
			tool: InlineToolConfig{
				Name:        "tool_name",
				Description: "Tool with invalid parameters",
				Parameters: map[string]any{
					"properties": map[string]any{},
				},
			},
			index:     0,
			shouldErr: true,
			errMsg:    "must have a 'type' field",
		},
		{
			name: "invalid parameters schema (invalid type)",
			tool: InlineToolConfig{
				Name:        "tool_name",
				Description: "Tool with invalid parameters type",
				Parameters: map[string]any{
					"type": "invalid_type",
				},
			},
			index:     0,
			shouldErr: true,
			errMsg:    "must be one of",
		},
		{
			name: "empty implementation string",
			tool: InlineToolConfig{
				Name:           "tool_name",
				Description:    "Tool with empty implementation",
				Implementation: "   ",
			},
			index:     0,
			shouldErr: true,
			errMsg:    "empty implementation",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateInlineToolDefinition(tt.tool, tt.index)

			if tt.shouldErr {
				require.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestIsValidToolName(t *testing.T) {
	tests := []struct {
		name  string
		input string
		valid bool
	}{
		{"valid lowercase", "toolname", true},
		{"valid with uppercase", "ToolName", true},
		{"valid with underscore", "tool_name", true},
		{"valid with hyphen", "tool-name", true},
		{"valid with numbers", "tool123", true},
		{"valid mixed", "my_tool-123", true},
		{"empty string", "", false},
		{"starts with number", "123tool", false},
		{"starts with underscore", "_tool", false},
		{"starts with hyphen", "-tool", false},
		{"contains special char", "tool@name", false},
		{"contains space", "tool name", false},
		{"contains dot", "tool.name", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidToolName(tt.input)
			assert.Equal(t, tt.valid, result)
		})
	}
}

func TestCheckInlineToolNameUniqueness(t *testing.T) {
	tests := []struct {
		name      string
		tools     []InlineToolConfig
		allTools  *ToolsConfig
		shouldErr bool
		errMsg    string
	}{
		{
			name:      "no tools",
			tools:     []InlineToolConfig{},
			allTools:  &ToolsConfig{},
			shouldErr: false,
		},
		{
			name: "unique names",
			tools: []InlineToolConfig{
				{Name: "tool1", Description: "First tool"},
				{Name: "tool2", Description: "Second tool"},
			},
			allTools:  &ToolsConfig{},
			shouldErr: false,
		},
		{
			name: "duplicate names in inline tools",
			tools: []InlineToolConfig{
				{Name: "tool1", Description: "First tool"},
				{Name: "tool1", Description: "Duplicate tool"},
			},
			allTools:  &ToolsConfig{},
			shouldErr: true,
			errMsg:    "duplicate inline tool name 'tool1'",
		},
		{
			name: "conflicts with built-in tool",
			tools: []InlineToolConfig{
				{Name: "github", Description: "Custom github tool"},
			},
			allTools:  &ToolsConfig{},
			shouldErr: true,
			errMsg:    "conflicts with built-in tool",
		},
		{
			name: "conflicts with bash built-in",
			tools: []InlineToolConfig{
				{Name: "bash", Description: "Custom bash tool"},
			},
			allTools:  &ToolsConfig{},
			shouldErr: true,
			errMsg:    "conflicts with built-in tool",
		},
		{
			name: "conflicts with custom MCP tool",
			tools: []InlineToolConfig{
				{Name: "my_mcp", Description: "Inline tool"},
			},
			allTools: &ToolsConfig{
				Custom: map[string]MCPServerConfig{
					"my_mcp": {},
				},
			},
			shouldErr: true,
			errMsg:    "conflicts with custom MCP tool",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := checkInlineToolNameUniqueness(tt.tools, tt.allTools)

			if tt.shouldErr {
				require.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateParametersSchema(t *testing.T) {
	tests := []struct {
		name      string
		params    map[string]any
		toolName  string
		index     int
		shouldErr bool
		errMsg    string
	}{
		{
			name: "valid object schema",
			params: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"field1": map[string]any{"type": "string"},
				},
			},
			toolName:  "test_tool",
			index:     0,
			shouldErr: false,
		},
		{
			name: "valid string schema",
			params: map[string]any{
				"type": "string",
			},
			toolName:  "test_tool",
			index:     0,
			shouldErr: false,
		},
		{
			name: "missing type field",
			params: map[string]any{
				"properties": map[string]any{},
			},
			toolName:  "test_tool",
			index:     0,
			shouldErr: true,
			errMsg:    "must have a 'type' field",
		},
		{
			name: "invalid type value",
			params: map[string]any{
				"type": "invalid",
			},
			toolName:  "test_tool",
			index:     0,
			shouldErr: true,
			errMsg:    "must be one of",
		},
		{
			name: "type is not string",
			params: map[string]any{
				"type": 123,
			},
			toolName:  "test_tool",
			index:     0,
			shouldErr: true,
			errMsg:    "'type' field must be a string",
		},
		{
			name: "object with invalid properties",
			params: map[string]any{
				"type":       "object",
				"properties": "not a map",
			},
			toolName:  "test_tool",
			index:     0,
			shouldErr: true,
			errMsg:    "'properties' field must be an object",
		},
		{
			name: "object with invalid required",
			params: map[string]any{
				"type":     "object",
				"required": "not an array",
			},
			toolName:  "test_tool",
			index:     0,
			shouldErr: true,
			errMsg:    "'required' field must be an array",
		},
		{
			name: "object with valid required array of strings",
			params: map[string]any{
				"type":     "object",
				"required": []string{"field1", "field2"},
			},
			toolName:  "test_tool",
			index:     0,
			shouldErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateParametersSchema(tt.params, tt.toolName, tt.index)

			if tt.shouldErr {
				require.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateInlineTools(t *testing.T) {
	tests := []struct {
		name      string
		data      *WorkflowData
		shouldErr bool
		errMsg    string
	}{
		{
			name:      "nil workflow data",
			data:      nil,
			shouldErr: false,
		},
		{
			name: "no inline tools",
			data: &WorkflowData{
				EngineConfig: &EngineConfig{ID: "copilot"},
				ParsedTools:  &ToolsConfig{},
			},
			shouldErr: false,
		},
		{
			name: "inline tools present (not yet supported)",
			data: &WorkflowData{
				EngineConfig: &EngineConfig{ID: "copilot"},
				ParsedTools: &ToolsConfig{
					Inline: []InlineToolConfig{
						{
							Name:        "test_tool",
							Description: "Test tool",
						},
					},
				},
			},
			shouldErr: true,
			errMsg:    "inline tools are not yet supported",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateInlineTools(tt.data)

			if tt.shouldErr {
				require.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestToolsConfigWithInlineTools(t *testing.T) {
	// Test that inline tools are properly integrated with ToolsConfig
	toolsMap := map[string]any{
		"github": map[string]any{
			"allowed": []any{"issue_read"},
		},
		"inline": []any{
			map[string]any{
				"name":        "custom_tool",
				"description": "A custom inline tool",
				"parameters": map[string]any{
					"type": "object",
				},
				"implementation": "return { success: true };",
			},
		},
	}

	tools := NewTools(toolsMap)

	require.NotNil(t, tools)
	assert.NotNil(t, tools.GitHub)
	assert.Len(t, tools.Inline, 1)
	assert.Equal(t, "custom_tool", tools.Inline[0].Name)
	assert.Equal(t, "A custom inline tool", tools.Inline[0].Description)

	// Test ToMap conversion
	result := tools.ToMap()
	require.NotNil(t, result)

	// inline should be present in the result
	inlineField, hasInline := result["inline"]
	require.True(t, hasInline, "inline field should be present in ToMap result")

	// Check if it's a slice ([]any is the actual type)
	inlineSlice, ok := inlineField.([]map[string]any)
	if !ok {
		// It might be []any, so we need to convert
		inlineAny, ok := inlineField.([]any)
		require.True(t, ok, "inline should be convertible to []any, got %T", inlineField)
		require.Len(t, inlineAny, 1)

		// Convert first element to map
		toolMap, ok := inlineAny[0].(map[string]any)
		require.True(t, ok, "inline tool should be map[string]any, got %T", inlineAny[0])
		assert.Equal(t, "custom_tool", toolMap["name"])
	} else {
		assert.Len(t, inlineSlice, 1)
		assert.Equal(t, "custom_tool", inlineSlice[0]["name"])
	}
}
