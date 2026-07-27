//go:build !integration

package cli

import (
	"bytes"
	"context"
	"strings"
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

// TestPrintBootstrapConfigTODO_PreservesManifestOrder verifies that all declared config
// actions are included in the TODO output in their exact manifest order. This guards
// against regressions that split actions into pre-install and post-install phases and
// thereby reorder or omit steps.
func TestPrintBootstrapConfigTODO_PreservesManifestOrder(t *testing.T) {
	// Declare a profile with an interleaved mix of action types to ensure ordering
	// cannot be accidentally satisfied by any implicit categorisation.
	profile := &resolvedBootstrapProfile{
		PackageID: "owner/repo",
		Profile: &repositoryPackageBootstrap{
			Config: []repositoryPackageBootstrapAction{
				{Type: "require-owner-type", Value: "Organization"},
				{Type: "copilot-auth", Secret: "COPILOT_TOKEN"},
				{Type: "repo-variable", Name: "MY_VAR"},
				{Type: "github-app", AppIDVariable: "APP_ID", PrivateKeySecret: "APP_KEY"},
				{Type: "repo-secret", Name: "MY_SECRET"},
				{Type: "commit-and-push", Message: "commit changes"},
				{Type: "handoff", Message: "all done"},
			},
		},
	}

	var buf bytes.Buffer
	printBootstrapConfigTODO(&buf, profile)
	output := buf.String()

	// Every action type must appear in the output.
	for _, want := range []string{
		"Organization",   // require-owner-type
		"COPILOT_TOKEN",  // copilot-auth
		"MY_VAR",         // repo-variable
		"APP_ID",         // github-app
		"MY_SECRET",      // repo-secret
		"commit changes", // commit-and-push
		"all done",       // handoff
	} {
		if !strings.Contains(output, want) {
			t.Errorf("expected %q in output, got:\n%s", want, output)
		}
	}

	// Verify declared order is preserved in the output.
	positions := []int{
		strings.Index(output, "Organization"),
		strings.Index(output, "COPILOT_TOKEN"),
		strings.Index(output, "MY_VAR"),
		strings.Index(output, "APP_ID"),
		strings.Index(output, "MY_SECRET"),
		strings.Index(output, "commit changes"),
		strings.Index(output, "all done"),
	}
	for i := 1; i < len(positions); i++ {
		if positions[i-1] < 0 || positions[i] < 0 {
			continue // already reported above
		}
		if positions[i-1] >= positions[i] {
			t.Errorf("action at position %d appears after action at position %d (ordering regression)", i-1, i)
		}
	}
}
