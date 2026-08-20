//go:build !integration

package workflow

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResolveCacheStepName(t *testing.T) {
	tests := []struct {
		name  string
		cache map[string]any
		idx   int
		total int
		want  string
	}{
		{
			name:  "name field present takes priority over everything",
			cache: map[string]any{"name": "My Cache", "key": "some-key"},
			idx:   0,
			total: 1,
			want:  "My Cache",
		},
		{
			name:  "empty name string falls through to key",
			cache: map[string]any{"name": "", "key": "some-key"},
			idx:   0,
			total: 1,
			want:  "Cache (some-key)",
		},
		{
			name:  "non-string name falls through to key",
			cache: map[string]any{"name": 42, "key": "some-key"},
			idx:   0,
			total: 1,
			want:  "Cache (some-key)",
		},
		{
			name:  "no name, key present, single total returns key-based name",
			cache: map[string]any{"key": "npm-cache"},
			idx:   0,
			total: 1,
			want:  "Cache (npm-cache)",
		},
		{
			name:  "empty key string falls through to default stepName",
			cache: map[string]any{"key": ""},
			idx:   0,
			total: 1,
			want:  "Cache",
		},
		{
			name:  "non-string key falls through to default stepName",
			cache: map[string]any{"key": 123},
			idx:   0,
			total: 1,
			want:  "Cache",
		},
		{
			name:  "no name or key, single total returns default Cache",
			cache: map[string]any{},
			idx:   0,
			total: 1,
			want:  "Cache",
		},
		{
			name:  "no name or key, multiple total returns indexed default",
			cache: map[string]any{},
			idx:   0,
			total: 3,
			want:  "Cache 1",
		},
		{
			name:  "no name or key, multiple total uses 1-based idx",
			cache: map[string]any{},
			idx:   2,
			total: 3,
			want:  "Cache 3",
		},
		{
			name:  "key present with multiple total still prefers key over indexed default",
			cache: map[string]any{"key": "build-cache"},
			idx:   1,
			total: 2,
			want:  "Cache (build-cache)",
		},
		{
			name:  "name present with multiple total still prefers name",
			cache: map[string]any{"name": "Custom"},
			idx:   1,
			total: 2,
			want:  "Custom",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveCacheStepName(tt.cache, tt.idx, tt.total)
			require.Equal(t, tt.want, got)
		})
	}
}
