//go:build !integration

package workflow

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnforceSafeUpdate(t *testing.T) {
	tests := []struct {
		name        string
		manifest    *GHAWManifest
		secretNames []string
		wantErr     bool
		wantErrMsgs []string
	}{
		{
			name:        "nil manifest skips enforcement (first compile)",
			manifest:    nil,
			secretNames: []string{"MY_SECRET"},
			wantErr:     false,
		},
		{
			name:        "empty secrets with existing manifest passes",
			manifest:    &GHAWManifest{Version: 1, Secrets: []string{}},
			secretNames: []string{},
			wantErr:     false,
		},
		{
			name:        "GITHUB_TOKEN always allowed even when not in manifest",
			manifest:    &GHAWManifest{Version: 1, Secrets: []string{}},
			secretNames: []string{"GITHUB_TOKEN"},
			wantErr:     false,
		},
		{
			name:        "GITHUB_TOKEN with secrets. prefix always allowed",
			manifest:    &GHAWManifest{Version: 1, Secrets: []string{}},
			secretNames: []string{"secrets.GITHUB_TOKEN"},
			wantErr:     false,
		},
		{
			name: "known secret passes",
			manifest: &GHAWManifest{
				Version: 1,
				Secrets: []string{"secrets.MY_SECRET"},
			},
			secretNames: []string{"MY_SECRET"},
			wantErr:     false,
		},
		{
			name: "new restricted secret causes failure",
			manifest: &GHAWManifest{
				Version: 1,
				Secrets: []string{"secrets.EXISTING_SECRET"},
			},
			secretNames: []string{"EXISTING_SECRET", "NEW_SECRET"},
			wantErr:     true,
			wantErrMsgs: []string{"secrets.NEW_SECRET", "safe update mode"},
		},
		{
			name: "multiple new secrets listed in error",
			manifest: &GHAWManifest{
				Version: 1,
				Secrets: []string{},
			},
			secretNames: []string{"SECRET_A", "SECRET_B"},
			wantErr:     true,
			wantErrMsgs: []string{"secrets.SECRET_A", "secrets.SECRET_B"},
		},
		{
			name: "GITHUB_TOKEN plus known secret passes",
			manifest: &GHAWManifest{
				Version: 1,
				Secrets: []string{"secrets.MY_SECRET"},
			},
			secretNames: []string{"GITHUB_TOKEN", "MY_SECRET"},
			wantErr:     false,
		},
		{
			name: "empty manifest blocks any new secret except GITHUB_TOKEN",
			manifest: &GHAWManifest{
				Version: 1,
				Secrets: []string{},
			},
			secretNames: []string{"SOME_SECRET"},
			wantErr:     true,
			wantErrMsgs: []string{"secrets.SOME_SECRET"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := EnforceSafeUpdate(tt.manifest, tt.secretNames)
			if tt.wantErr {
				require.Error(t, err, "expected safe update enforcement error")
				for _, msg := range tt.wantErrMsgs {
					assert.Contains(t, err.Error(), msg, "error message should contain %q", msg)
				}
			} else {
				assert.NoError(t, err, "unexpected safe update enforcement error")
			}
		})
	}
}

func TestBuildSafeUpdateError(t *testing.T) {
	violations := []string{"secrets.NEW_SECRET", "secrets.ANOTHER_SECRET"}
	err := buildSafeUpdateError(violations)
	require.Error(t, err, "should return an error")

	msg := err.Error()
	assert.Contains(t, msg, "safe update mode", "error message")
	assert.Contains(t, msg, "secrets.NEW_SECRET", "violation in message")
	assert.Contains(t, msg, "secrets.ANOTHER_SECRET", "violation in message")
	assert.Contains(t, msg, "interactive agentic flow", "remediation guidance")
}

func TestEffectiveSafeUpdate(t *testing.T) {
	tests := []struct {
		name           string
		compilerFlag   bool
		rawFrontmatter map[string]any
		want           bool
	}{
		{
			name:         "compiler flag off, no frontmatter => false",
			compilerFlag: false,
			want:         false,
		},
		{
			name:         "compiler flag on => true",
			compilerFlag: true,
			want:         true,
		},
		{
			name:           "frontmatter safe-update true is ignored, compiler flag off => false",
			compilerFlag:   false,
			rawFrontmatter: map[string]any{"safe-update": true},
			want:           false,
		},
		{
			name:           "frontmatter safe-update true is ignored, compiler flag on => true",
			compilerFlag:   true,
			rawFrontmatter: map[string]any{"safe-update": true},
			want:           true,
		},
		{
			name:           "frontmatter safe-update false, compiler flag off => false",
			compilerFlag:   false,
			rawFrontmatter: map[string]any{"safe-update": false},
			want:           false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Compiler{safeUpdate: tt.compilerFlag}
			data := &WorkflowData{RawFrontmatter: tt.rawFrontmatter}
			got := c.effectiveSafeUpdate(data)
			assert.Equal(t, tt.want, got, "effectiveSafeUpdate result")
		})
	}
}
