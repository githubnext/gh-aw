package workflow

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDetectActionMode_WithInjectedGetter(t *testing.T) {
	origRelease := IsRelease()
	defer SetIsRelease(origRelease)
	SetIsRelease(false)

	tests := []struct {
		name string
		env  map[string]string
		want ActionMode
	}{
		{
			name: "explicit override wins",
			env: map[string]string{
				"GH_AW_ACTION_MODE": "release",
				"GITHUB_REF":        "refs/heads/main",
			},
			want: ActionModeRelease,
		},
		{
			name: "release tag uses action mode",
			env: map[string]string{
				"GITHUB_REF": "refs/tags/v1.0.0",
			},
			want: ActionModeAction,
		},
		{
			name: "release event uses action mode",
			env: map[string]string{
				"GITHUB_EVENT_NAME": "release",
			},
			want: ActionModeAction,
		},
		{
			name: "invalid override falls through",
			env: map[string]string{
				"GH_AW_ACTION_MODE": "bogus",
			},
			want: ActionModeDev,
		},
		{
			name: "default is dev mode",
			env:  map[string]string{},
			want: ActionModeDev,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			getenv := func(key string) string { return tc.env[key] }
			assert.Equal(t, tc.want, detectActionMode("ignored-version", getenv))
		})
	}
}

func TestDetectActionMode_ReleaseFlagAndOverride(t *testing.T) {
	origRelease := IsRelease()
	defer SetIsRelease(origRelease)

	t.Run("release flag uses action mode", func(t *testing.T) {
		SetIsRelease(true)
		assert.Equal(t, ActionModeAction, detectActionMode("ignored-version", func(string) string { return "" }))
	})

	t.Run("env override beats release flag", func(t *testing.T) {
		SetIsRelease(true)
		getenv := func(key string) string {
			if key == "GH_AW_ACTION_MODE" {
				return "dev"
			}
			return ""
		}
		assert.Equal(t, ActionModeDev, detectActionMode("ignored-version", getenv))
	})
}
