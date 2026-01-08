package workflow

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMapToolConfig_GetString(t *testing.T) {
	tests := []struct {
		name      string
		config    MapToolConfig
		key       string
		wantValue string
		wantOk    bool
	}{
		{
			name: "existing string key",
			config: MapToolConfig{
				"name": "test-server",
			},
			key:       "name",
			wantValue: "test-server",
			wantOk:    true,
		},
		{
			name: "non-string value",
			config: MapToolConfig{
				"port": 8080,
			},
			key:       "port",
			wantValue: "",
			wantOk:    false,
		},
		{
			name: "non-existent key",
			config: MapToolConfig{
				"foo": "bar",
			},
			key:       "missing",
			wantValue: "",
			wantOk:    false,
		},
		{
			name:      "empty config",
			config:    MapToolConfig{},
			key:       "anything",
			wantValue: "",
			wantOk:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotValue, gotOk := tt.config.GetString(tt.key)

			assert.Equal(t, tt.wantOk, gotOk, "GetString() ok")
			assert.Equal(t, tt.wantValue, gotValue, "GetString() value")
		})
	}
}

func TestMapToolConfig_GetStringArray(t *testing.T) {
	tests := []struct {
		name      string
		config    MapToolConfig
		key       string
		wantValue []string
		wantOk    bool
	}{
		{
			name: "array of any with strings",
			config: MapToolConfig{
				"items": []any{"a", "b", "c"},
			},
			key:       "items",
			wantValue: []string{"a", "b", "c"},
			wantOk:    true,
		},
		{
			name: "array of strings",
			config: MapToolConfig{
				"items": []string{"x", "y", "z"},
			},
			key:       "items",
			wantValue: []string{"x", "y", "z"},
			wantOk:    true,
		},
		{
			name: "array of any with mixed types (filters non-strings)",
			config: MapToolConfig{
				"items": []any{"string", 123, true, "another"},
			},
			key:       "items",
			wantValue: []string{"string", "another"},
			wantOk:    true,
		},
		{
			name: "empty array",
			config: MapToolConfig{
				"items": []string{},
			},
			key:       "items",
			wantValue: []string{},
			wantOk:    true,
		},
		{
			name: "non-array value",
			config: MapToolConfig{
				"value": "not-an-array",
			},
			key:       "value",
			wantValue: nil,
			wantOk:    false,
		},
		{
			name: "non-existent key",
			config: MapToolConfig{
				"foo": []string{"bar"},
			},
			key:       "missing",
			wantValue: nil,
			wantOk:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotValue, gotOk := tt.config.GetStringArray(tt.key)

			assert.Equal(t, tt.wantOk, gotOk, "GetStringArray() ok")
			assert.Equal(t, tt.wantValue, gotValue, "GetStringArray() value")
		})
	}
}

func TestMapToolConfig_GetStringMap(t *testing.T) {
	tests := []struct {
		name      string
		config    MapToolConfig
		key       string
		wantValue map[string]string
		wantOk    bool
	}{
		{
			name: "map[string]any with strings",
			config: MapToolConfig{
				"env": map[string]any{
					"VAR1": "value1",
					"VAR2": "value2",
				},
			},
			key: "env",
			wantValue: map[string]string{
				"VAR1": "value1",
				"VAR2": "value2",
			},
			wantOk: true,
		},
		{
			name: "map[string]string",
			config: MapToolConfig{
				"headers": map[string]string{
					"Authorization": "Bearer token",
					"Content-Type":  "application/json",
				},
			},
			key: "headers",
			wantValue: map[string]string{
				"Authorization": "Bearer token",
				"Content-Type":  "application/json",
			},
			wantOk: true,
		},
		{
			name: "map[string]any with mixed types (filters non-strings)",
			config: MapToolConfig{
				"mixed": map[string]any{
					"string": "value",
					"number": 123,
					"bool":   true,
				},
			},
			key: "mixed",
			wantValue: map[string]string{
				"string": "value",
			},
			wantOk: true,
		},
		{
			name: "empty map",
			config: MapToolConfig{
				"empty": map[string]string{},
			},
			key:       "empty",
			wantValue: map[string]string{},
			wantOk:    true,
		},
		{
			name: "non-map value",
			config: MapToolConfig{
				"value": "not-a-map",
			},
			key:       "value",
			wantValue: nil,
			wantOk:    false,
		},
		{
			name: "non-existent key",
			config: MapToolConfig{
				"foo": map[string]string{"bar": "baz"},
			},
			key:       "missing",
			wantValue: nil,
			wantOk:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotValue, gotOk := tt.config.GetStringMap(tt.key)

			assert.Equal(t, tt.wantOk, gotOk, "GetStringMap() ok")
			assert.Equal(t, tt.wantValue, gotValue, "GetStringMap() value")
		})
	}
}

func TestMapToolConfig_GetAny(t *testing.T) {
	tests := []struct {
		name      string
		config    MapToolConfig
		key       string
		wantValue any
		wantOk    bool
	}{
		{
			name: "string value",
			config: MapToolConfig{
				"name": "test-server",
			},
			key:       "name",
			wantValue: "test-server",
			wantOk:    true,
		},
		{
			name: "number value",
			config: MapToolConfig{
				"port": 8080,
			},
			key:       "port",
			wantValue: 8080,
			wantOk:    true,
		},
		{
			name: "boolean value",
			config: MapToolConfig{
				"enabled": true,
			},
			key:       "enabled",
			wantValue: true,
			wantOk:    true,
		},
		{
			name: "array value",
			config: MapToolConfig{
				"items": []string{"a", "b", "c"},
			},
			key:       "items",
			wantValue: []string{"a", "b", "c"},
			wantOk:    true,
		},
		{
			name: "object value",
			config: MapToolConfig{
				"nested": map[string]any{"key": "value"},
			},
			key:       "nested",
			wantValue: map[string]any{"key": "value"},
			wantOk:    true,
		},
		{
			name: "nil value",
			config: MapToolConfig{
				"nullable": nil,
			},
			key:       "nullable",
			wantValue: nil,
			wantOk:    true,
		},
		{
			name: "non-existent key",
			config: MapToolConfig{
				"foo": "bar",
			},
			key:       "missing",
			wantValue: nil,
			wantOk:    false,
		},
		{
			name:      "empty config",
			config:    MapToolConfig{},
			key:       "anything",
			wantValue: nil,
			wantOk:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotValue, gotOk := tt.config.GetAny(tt.key)

			assert.Equal(t, tt.wantOk, gotOk, "GetAny() ok")

			// For non-existent keys, value should be nil
			if !tt.wantOk {
				assert.Nil(t, gotValue, "GetAny() value should be nil for non-existent key")
				return
			}

			// For existing keys, compare values based on type
			switch wantVal := tt.wantValue.(type) {
			case string:
				assert.Equal(t, wantVal, gotValue, "GetAny() value")
			case int:
				assert.Equal(t, wantVal, gotValue, "GetAny() value")
			case bool:
				assert.Equal(t, wantVal, gotValue, "GetAny() value")
			case nil:
				assert.Nil(t, gotValue, "GetAny() value")
			default:
				// For complex types like arrays and objects, just verify they exist
				assert.NotNil(t, gotValue, "GetAny() value should not be nil")
			}
		})
	}
}

func TestMCPConfigRenderer_FieldDefaults(t *testing.T) {
	// Test that MCPConfigRenderer can be created with zero values
	renderer := MCPConfigRenderer{}

	assert.Empty(t, renderer.IndentLevel, "Default IndentLevel should be empty")
	assert.Empty(t, renderer.Format, "Default Format should be empty")
	assert.False(t, renderer.RequiresCopilotFields, "Default RequiresCopilotFields should be false")
	assert.False(t, renderer.RewriteLocalhostToDocker, "Default RewriteLocalhostToDocker should be false")
}

func TestMCPConfigRenderer_FieldAssignment(t *testing.T) {
	// Test that MCPConfigRenderer fields can be set
	renderer := MCPConfigRenderer{
		IndentLevel:              "    ",
		Format:                   "json",
		RequiresCopilotFields:    true,
		RewriteLocalhostToDocker: true,
	}

	assert.Equal(t, "    ", renderer.IndentLevel, "IndentLevel")
	assert.Equal(t, "json", renderer.Format, "Format")
	assert.True(t, renderer.RequiresCopilotFields, "RequiresCopilotFields")
	assert.True(t, renderer.RewriteLocalhostToDocker, "RewriteLocalhostToDocker")
}
