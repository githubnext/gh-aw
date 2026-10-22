package aw

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAuthoredJobDependencies(t *testing.T) {
	graph := AuthoredJobDependencies(map[string]any{
		"prepare":   map[string]any{"needs": "activation"},
		"scan":      map[string]any{"needs": []any{"agent", "detection"}},
		"report":    map[string]any{"needs": []string{"scan"}},
		"publish":   map[string]any{"needs": "safe_outputs"},
		"loop":      map[string]any{"needs": "loop"},
		"malformed": map[string]any{"needs": []any{123}},
		"invalid":   false,
	})
	assert.True(t, graph.DependsOn("report", "agent"))
	assert.True(t, graph.DependsOn("report", "detection"))
	assert.True(t, graph.DependsOn("publish", "agent"))
	assert.False(t, graph.DependsOn("prepare", "agent"))
	assert.False(t, graph.DependsOn("loop", "agent"))
	assert.False(t, graph.DependsOn("malformed", "agent"))
	assert.False(t, graph.DependsOn("invalid", "agent"))
	assert.False(t, graph.DependsOn("missing", "agent"))
}
