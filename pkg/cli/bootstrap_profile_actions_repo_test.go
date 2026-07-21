//go:build !integration

package cli

import (
	"context"
	"slices"
	"strings"
	"testing"
)

func TestListBootstrapRepoNamesPaginate(t *testing.T) {
	originalRunGH := runBootstrapGHContext
	t.Cleanup(func() {
		runBootstrapGHContext = originalRunGH
	})

	calls := []string{}
	runBootstrapGHContext = func(_ context.Context, _ string, args ...string) ([]byte, error) {
		calls = append(calls, strings.Join(args, " "))
		if strings.Contains(args[1], "/variables") {
			return []byte("ALPHA\nOMEGA\n"), nil
		}
		return []byte("FIRST\nSECOND\n"), nil
	}

	variables, err := listBootstrapRepoVariableNames(context.Background(), "octo/platform-ops")
	if err != nil {
		t.Fatalf("listBootstrapRepoVariableNames returned error: %v", err)
	}
	if !slices.Equal(variables, []string{"ALPHA", "OMEGA"}) {
		t.Fatalf("unexpected variables: %#v", variables)
	}

	secrets, err := listBootstrapRepoSecretNames(context.Background(), "octo/platform-ops")
	if err != nil {
		t.Fatalf("listBootstrapRepoSecretNames returned error: %v", err)
	}
	if !slices.Equal(secrets, []string{"FIRST", "SECOND"}) {
		t.Fatalf("unexpected secrets: %#v", secrets)
	}

	if len(calls) != 2 {
		t.Fatalf("expected 2 gh api calls, got %d", len(calls))
	}
	for _, call := range calls {
		if !strings.Contains(call, "--paginate") {
			t.Fatalf("expected paginated gh api call, got %q", call)
		}
	}
}

func TestRunBootstrapRepoVariableAction(t *testing.T) {
	originalUpsertVariable := bootstrapUpsertVariable
	t.Cleanup(func() {
		bootstrapUpsertVariable = originalUpsertVariable
	})

	t.Setenv(bootstrapRepositoryVariableEnvName("MY_VAR"), "configured")
	var gotName, gotValue string
	bootstrapUpsertVariable = func(_ context.Context, _ string, name, value string) error {
		gotName = name
		gotValue = value
		return nil
	}

	applied, err := runBootstrapRepoVariableAction(context.Background(), "octo/platform-ops", repositoryPackageBootstrapAction{
		Name: "MY_VAR",
	}, &bootstrapProfileExistingState{variables: map[string]struct{}{}, secrets: map[string]struct{}{}})
	if err != nil {
		t.Fatalf("runBootstrapRepoVariableAction returned error: %v", err)
	}
	if !applied {
		t.Fatal("expected variable action to apply")
	}
	if gotName != "MY_VAR" || gotValue != "configured" {
		t.Fatalf("unexpected variable write: %s=%s", gotName, gotValue)
	}
}

func TestRunBootstrapRepoSecretAction(t *testing.T) {
	originalSetSecret := bootstrapSetSecret
	t.Cleanup(func() {
		bootstrapSetSecret = originalSetSecret
	})

	t.Setenv(bootstrapRepositorySecretEnvName("MY_SECRET"), "top-secret")
	var gotName, gotValue string
	bootstrapSetSecret = func(_ context.Context, _ string, name, value string) error {
		gotName = name
		gotValue = value
		return nil
	}

	applied, err := runBootstrapRepoSecretAction(context.Background(), "octo/platform-ops", repositoryPackageBootstrapAction{
		Name: "MY_SECRET",
	}, &bootstrapProfileExistingState{variables: map[string]struct{}{}, secrets: map[string]struct{}{}})
	if err != nil {
		t.Fatalf("runBootstrapRepoSecretAction returned error: %v", err)
	}
	if !applied {
		t.Fatal("expected secret action to apply")
	}
	if gotName != "MY_SECRET" || gotValue != "top-secret" {
		t.Fatalf("unexpected secret write: %s=%s", gotName, gotValue)
	}
}

func TestRunBootstrapCopilotAuthAction(t *testing.T) {
	t.Run("skips actions token auth", func(t *testing.T) {
		applied, err := runBootstrapCopilotAuthAction(context.Background(), "octo/platform-ops", repositoryPackageBootstrapAction{
			Secret: "COPILOT_TOKEN",
		}, &bootstrapProfileExistingState{variables: map[string]struct{}{}, secrets: map[string]struct{}{}}, true)
		if err != nil {
			t.Fatalf("runBootstrapCopilotAuthAction returned error: %v", err)
		}
		if applied {
			t.Fatal("expected action to skip when Actions token auth is enabled")
		}
	})

	t.Run("stores valid pat", func(t *testing.T) {
		originalSetSecret := bootstrapSetSecret
		t.Cleanup(func() {
			bootstrapSetSecret = originalSetSecret
		})

		t.Setenv("COPILOT_TOKEN", "github_pat_abc123xyz")
		var wrote string
		bootstrapSetSecret = func(_ context.Context, _ string, name, value string) error {
			wrote = name + "=" + value
			return nil
		}

		applied, err := runBootstrapCopilotAuthAction(context.Background(), "octo/platform-ops", repositoryPackageBootstrapAction{
			Secret: "COPILOT_TOKEN",
		}, &bootstrapProfileExistingState{variables: map[string]struct{}{}, secrets: map[string]struct{}{}}, false)
		if err != nil {
			t.Fatalf("runBootstrapCopilotAuthAction returned error: %v", err)
		}
		if !applied {
			t.Fatal("expected action to apply")
		}
		if wrote != "COPILOT_TOKEN=github_pat_abc123xyz" {
			t.Fatalf("unexpected secret write: %s", wrote)
		}
	})
}

func TestBootstrapRepoMutationHelpers_RejectInvalidRepo(t *testing.T) {
	if err := upsertBootstrapRepoVariable("not-a-repo", "NAME", "value"); err == nil {
		t.Fatal("expected invalid repo slug error for variable upsert")
	}
	if err := setBootstrapRepoSecret("not-a-repo", "NAME", "value"); err == nil {
		t.Fatal("expected invalid repo slug error for secret set")
	}
}
