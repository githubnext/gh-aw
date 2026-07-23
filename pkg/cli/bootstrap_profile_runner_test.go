//go:build !integration

package cli

import (
	"context"
	"testing"
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

func TestBootstrapProfileAddWizardPhases(t *testing.T) {
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
			},
		},
	}

	preInstall := bootstrapProfileAddWizardPreInstall(profile)
	if preInstall == nil || preInstall.Profile == nil {
		t.Fatal("expected pre-install bootstrap profile")
	}
	if got := len(preInstall.Profile.Config); got != 4 {
		t.Fatalf("pre-install profile should contain 4 actions, got %d", got)
	}
	if preInstall.Profile.Config[0].Type != "require-owner-type" || preInstall.Profile.Config[3].Type != "repo-secret" {
		t.Fatalf("unexpected pre-install action ordering: %+v", preInstall.Profile.Config)
	}

	postInstall := bootstrapProfileAddWizardPostInstall(profile)
	if postInstall == nil || postInstall.Profile == nil {
		t.Fatal("expected post-install bootstrap profile")
	}
	if got := len(postInstall.Profile.Config); got != 3 {
		t.Fatalf("post-install profile should contain 3 actions, got %d", got)
	}
	if postInstall.Profile.Config[0].Type != "copilot-auth" || postInstall.Profile.Config[2].Type != "handoff" {
		t.Fatalf("unexpected post-install action ordering: %+v", postInstall.Profile.Config)
	}

	if got := len(profile.Profile.Config); got != 7 {
		t.Fatalf("original profile should remain unchanged, got %d actions", got)
	}
}
