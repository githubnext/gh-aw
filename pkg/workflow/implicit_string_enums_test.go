//go:build !integration

package workflow

import (
	"reflect"
	"testing"
)

func TestImplicitStringEnumFieldTypes(t *testing.T) {
	tests := []struct {
		name     string
		value    any
		typeName string
	}{
		{"safe outputs URLs", SafeOutputsConfig{}.URLs, "SafeOutputsURLsPolicy"},
		{"AI reaction", WorkflowData{}.AIReaction, "ReactionType"},
		{"MCP script parameter", MCPScriptParam{}.Type, "MCPParamType"},
		{"runner topology", RunnerConfig{}.Topology, "RunnerTopology"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if actual := reflect.TypeOf(tt.value).Name(); actual != tt.typeName {
				t.Errorf("field type = %q, want %q", actual, tt.typeName)
			}
		})
	}
}
