//go:build !integration

package cli

import (
	"context"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBootstrapActionNeedsMutation(t *testing.T) {
	state := &bootstrapProfileExistingState{
		variables: map[string]struct{}{"EXISTING_VAR": {}, "APP_ID": {}},
		secrets:   map[string]struct{}{"EXISTING_SECRET": {}},
	}

	tests := []struct {
		name             string
		action           repositoryPackageBootstrapAction
		usesActionsToken bool
		want             bool
	}{
		{name: "repo variable missing", action: repositoryPackageBootstrapAction{Type: "repo-variable", Name: "NEW_VAR"}, want: true},
		{name: "repo variable existing", action: repositoryPackageBootstrapAction{Type: "repo-variable", Name: "EXISTING_VAR"}, want: false},
		{name: "repo secret missing", action: repositoryPackageBootstrapAction{Type: "repo-secret", Name: "NEW_SECRET"}, want: true},
		{name: "repo secret existing", action: repositoryPackageBootstrapAction{Type: "repo-secret", Name: "EXISTING_SECRET"}, want: false},
		{name: "github app partial", action: repositoryPackageBootstrapAction{Type: "github-app", AppIDVariable: "APP_ID", PrivateKeySecret: "APP_PRIVATE_KEY"}, want: true},
		{name: "copilot auth with actions token", action: repositoryPackageBootstrapAction{Type: "copilot-auth", Secret: "COPILOT_TOKEN"}, usesActionsToken: true, want: false},
		{name: "commit push always pending", action: repositoryPackageBootstrapAction{Type: "commit-and-push"}, want: true},
		{name: "handoff never pending", action: repositoryPackageBootstrapAction{Type: "handoff"}, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := bootstrapActionNeedsMutation(context.Background(), "octo/platform-ops", tt.action, state, tt.usesActionsToken)
			if err != nil {
				t.Fatalf("bootstrapActionNeedsMutation returned error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("bootstrapActionNeedsMutation returned %t, want %t", got, tt.want)
			}
		})
	}
}

func TestBootstrapProfileState(t *testing.T) {
	originalRunGH := runBootstrapGHContext
	t.Cleanup(func() {
		runBootstrapGHContext = originalRunGH
	})

	runBootstrapGHContext = func(_ context.Context, _ string, args ...string) ([]byte, error) {
		if len(args) > 1 && args[1] == "/repos/octo/platform-ops/actions/variables?per_page=100" {
			return []byte("BETA\nALPHA\n"), nil
		}
		return []byte("SECRET_ONE\n"), nil
	}

	state, err := bootstrapProfileState(context.Background(), "octo/platform-ops")
	if err != nil {
		t.Fatalf("bootstrapProfileState returned error: %v", err)
	}
	if _, ok := state.variables["ALPHA"]; !ok {
		t.Fatal("expected ALPHA variable")
	}
	if _, ok := state.variables["BETA"]; !ok {
		t.Fatal("expected BETA variable")
	}
	if _, ok := state.secrets["SECRET_ONE"]; !ok {
		t.Fatal("expected SECRET_ONE secret")
	}
}

func TestBootstrapProfileFilterActions(t *testing.T) {
	// Verify that filterBootstrapProfileActions correctly filters actions and
	// preserves the original profile unchanged.
	allTypes := []string{"require-owner-type", "github-app", "repo-variable", "repo-secret", "copilot-auth", "commit-and-push", "handoff"}

	profile := &resolvedBootstrapProfile{
		PackageID: "owner/repo",
		Profile: &repositoryPackageBootstrap{
			Config: []repositoryPackageBootstrapAction{
				{Type: "require-owner-type"},
				{Type: "github-app"},
				{Type: "repo-variable"},
				{Type: "repo-secret"},
				{Type: "copilot-auth"},
				{Type: "commit-and-push"},
				{Type: "handoff"},
				{Type: "unsupported"},
			},
		},
	}

	// Filter to only known action types (excludes "unsupported")
	filtered := filterBootstrapProfileActions(profile, func(action repositoryPackageBootstrapAction) bool {
		return slices.Contains(allTypes, action.Type)
	})
	require.NotNil(t, filtered)
	assert.Equal(t, allTypes, bootstrapActionTypes(filtered.Profile.Config))

	// Original profile should remain unchanged
	assert.Len(t, profile.Profile.Config, 8)

	// nil / empty profile returns nil
	assert.Nil(t, filterBootstrapProfileActions(nil, func(_ repositoryPackageBootstrapAction) bool { return true }))

	unsupportedOnlyProfile := &resolvedBootstrapProfile{
		PackageID: "owner/repo",
		Profile:   &repositoryPackageBootstrap{Config: []repositoryPackageBootstrapAction{{Type: "unsupported"}}},
	}
	assert.Nil(t, filterBootstrapProfileActions(unsupportedOnlyProfile, func(_ repositoryPackageBootstrapAction) bool { return false }),
		"all-filtered-out profile should return nil")
}

func bootstrapActionTypes(actions []repositoryPackageBootstrapAction) []string {
	types := make([]string, 0, len(actions))
	for _, action := range actions {
		types = append(types, action.Type)
	}
	return types
}
