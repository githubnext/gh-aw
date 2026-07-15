//go:build !integration

package cli

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestLoadBootstrapGitHubAppOverrides(t *testing.T) {
	t.Setenv(bootstrapGitHubAppModeEnv, "create")
	t.Setenv(bootstrapGitHubAppOwnerEnv, "octo-platform")
	t.Setenv(bootstrapGitHubAppNameEnv, "octo-control-plane")
	t.Setenv(bootstrapGitHubAppURLEnv, "https://github.com/octo/platform-ops")
	t.Setenv(bootstrapGitHubAppDescriptionEnv, "Bootstrap app")
	t.Setenv(bootstrapNoOpenBrowserEnv, "true")

	overrides, err := loadBootstrapGitHubAppOverrides()
	if err != nil {
		t.Fatalf("loadBootstrapGitHubAppOverrides returned error: %v", err)
	}
	if overrides.Mode != "create" {
		t.Fatalf("expected create mode, got %q", overrides.Mode)
	}
	if overrides.Owner != "octo-platform" {
		t.Fatalf("expected owner override, got %q", overrides.Owner)
	}
	if overrides.Name != "octo-control-plane" {
		t.Fatalf("expected name override, got %q", overrides.Name)
	}
	if overrides.HomepageURL != "https://github.com/octo/platform-ops" {
		t.Fatalf("expected homepage override, got %q", overrides.HomepageURL)
	}
	if overrides.Description != "Bootstrap app" {
		t.Fatalf("expected description override, got %q", overrides.Description)
	}
	if overrides.OpenBrowser {
		t.Fatal("expected browser opening to be disabled")
	}
}

func TestLoadBootstrapGitHubAppOverrides_RejectsInvalidMode(t *testing.T) {
	t.Setenv(bootstrapGitHubAppModeEnv, "later")

	_, err := loadBootstrapGitHubAppOverrides()
	if err == nil {
		t.Fatal("expected invalid mode error")
	}
}

func TestParseBootstrapBool(t *testing.T) {
	t.Run("truthy", func(t *testing.T) {
		truthy := []string{"1", "true", "yes", "on"}
		for _, raw := range truthy {
			got, err := parseBootstrapBool(raw)
			if err != nil {
				t.Fatalf("parseBootstrapBool(%q) returned error: %v", raw, err)
			}
			if !got {
				t.Fatalf("expected %q to parse as true", raw)
			}
		}
	})

	t.Run("falsy", func(t *testing.T) {
		falsy := []string{"0", "false", "no", "off"}
		for _, raw := range falsy {
			got, err := parseBootstrapBool(raw)
			if err != nil {
				t.Fatalf("parseBootstrapBool(%q) returned error: %v", raw, err)
			}
			if got {
				t.Fatalf("expected %q to parse as false", raw)
			}
		}
	})

	t.Run("invalid", func(t *testing.T) {
		if _, err := parseBootstrapBool("maybe"); err == nil {
			t.Fatal("expected invalid boolean error")
		}
	})
}

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

func TestWorkflowGrantsCopilotRequestsWrite_UsesFrontmatterPermissions(t *testing.T) {
	t.Run("requires structural permission", func(t *testing.T) {
		content := []byte("---\nengine: copilot\npermissions:\n  contents: read\n---\n\ncopilot-requests: write\n")
		if workflowGrantsCopilotRequestsWrite(content) {
			t.Fatal("expected prompt body text to be ignored")
		}
	})

	t.Run("accepts explicit permission", func(t *testing.T) {
		content := []byte("---\nengine: copilot\npermissions:\n  copilot-requests: write\n---\n")
		if !workflowGrantsCopilotRequestsWrite(content) {
			t.Fatal("expected explicit copilot-requests: write permission")
		}
	})
}

func TestRunBootstrapGitHubAppAction_NonInteractiveCreateRequiresExplicitOverride(t *testing.T) {
	originalInteractive := bootstrapIsInteractive
	originalCheckOwnerType := bootstrapCheckOwnerType
	originalCreateGitHubApp := bootstrapCreateGitHubApp
	t.Cleanup(func() {
		bootstrapIsInteractive = originalInteractive
		bootstrapCheckOwnerType = originalCheckOwnerType
		bootstrapCreateGitHubApp = originalCreateGitHubApp
	})

	bootstrapIsInteractive = func() bool { return false }
	bootstrapCheckOwnerType = func(context.Context, string) (string, error) { return "Organization", nil }
	bootstrapCreateGitHubApp = func(context.Context, string, string, string, string, repositoryPackageBootstrapAction, bootstrapGitHubAppOverrides) (*bootstrapCreatedGitHubApp, error) {
		t.Fatal("createBootstrapGitHubApp should not be called")
		return nil, nil
	}

	_, err := runBootstrapGitHubAppAction(context.Background(), "octo/platform-ops", repositoryPackageBootstrapAction{
		Type:             "github-app",
		AppIDVariable:    "APP_ID",
		PrivateKeySecret: "APP_PRIVATE_KEY",
	}, &bootstrapProfileExistingState{
		variables: map[string]struct{}{},
		secrets:   map[string]struct{}{},
	})
	if err == nil {
		t.Fatal("expected non-interactive create error")
	}
	if !strings.Contains(err.Error(), bootstrapGitHubAppClientIDEnv) || !strings.Contains(err.Error(), bootstrapGitHubAppPrivateKeyEnv) {
		t.Fatalf("expected error to reference bootstrap GitHub App env vars, got %v", err)
	}
}

func TestRunBootstrapGitHubAppAction_RepairsExistingCredentialPairAtomically(t *testing.T) {
	originalCheckOwnerType := bootstrapCheckOwnerType
	originalUpsertVariable := bootstrapUpsertVariable
	originalSetSecret := bootstrapSetSecret
	t.Cleanup(func() {
		bootstrapCheckOwnerType = originalCheckOwnerType
		bootstrapUpsertVariable = originalUpsertVariable
		bootstrapSetSecret = originalSetSecret
	})

	t.Setenv(bootstrapGitHubAppClientIDEnv, "Iv1.client")
	t.Setenv(bootstrapGitHubAppPrivateKeyEnv, "-----BEGIN PRIVATE KEY-----\nkey\n-----END PRIVATE KEY-----")
	bootstrapCheckOwnerType = func(context.Context, string) (string, error) { return "Organization", nil }

	var writes []string
	bootstrapUpsertVariable = func(_ context.Context, _ string, name, value string) error {
		writes = append(writes, "var:"+name+"="+value)
		return nil
	}
	bootstrapSetSecret = func(_ context.Context, _ string, name, value string) error {
		writes = append(writes, "secret:"+name+"="+value)
		return nil
	}

	_, err := runBootstrapGitHubAppAction(context.Background(), "octo/platform-ops", repositoryPackageBootstrapAction{
		Type:             "github-app",
		AppIDVariable:    "APP_ID",
		PrivateKeySecret: "APP_PRIVATE_KEY",
		Mode:             "existing",
	}, &bootstrapProfileExistingState{
		variables: map[string]struct{}{"APP_ID": {}},
		secrets:   map[string]struct{}{},
	})
	if err != nil {
		t.Fatalf("runBootstrapGitHubAppAction returned error: %v", err)
	}
	expected := []string{
		"var:APP_ID=Iv1.client",
		"secret:APP_PRIVATE_KEY=-----BEGIN PRIVATE KEY-----\nkey\n-----END PRIVATE KEY-----",
	}
	if !slices.Equal(writes, expected) {
		t.Fatalf("unexpected writes: %#v", writes)
	}
}

func TestRunBootstrapGitHubAppAction_CreateOverwritesPartialCredentialPair(t *testing.T) {
	originalInteractive := bootstrapIsInteractive
	originalCheckOwnerType := bootstrapCheckOwnerType
	originalUpsertVariable := bootstrapUpsertVariable
	originalSetSecret := bootstrapSetSecret
	originalCreateGitHubApp := bootstrapCreateGitHubApp
	t.Cleanup(func() {
		bootstrapIsInteractive = originalInteractive
		bootstrapCheckOwnerType = originalCheckOwnerType
		bootstrapUpsertVariable = originalUpsertVariable
		bootstrapSetSecret = originalSetSecret
		bootstrapCreateGitHubApp = originalCreateGitHubApp
	})

	t.Setenv(bootstrapGitHubAppModeEnv, "create")
	bootstrapIsInteractive = func() bool { return false }
	bootstrapCheckOwnerType = func(context.Context, string) (string, error) { return "Organization", nil }

	var writes []string
	bootstrapUpsertVariable = func(_ context.Context, _ string, name, value string) error {
		writes = append(writes, "var:"+name+"="+value)
		return nil
	}
	bootstrapSetSecret = func(_ context.Context, _ string, name, value string) error {
		writes = append(writes, "secret:"+name+"="+value)
		return nil
	}
	bootstrapCreateGitHubApp = func(context.Context, string, string, string, string, repositoryPackageBootstrapAction, bootstrapGitHubAppOverrides) (*bootstrapCreatedGitHubApp, error) {
		return &bootstrapCreatedGitHubApp{
			ClientID: "Iv1.created",
			PEM:      "pem-value",
		}, nil
	}

	_, err := runBootstrapGitHubAppAction(context.Background(), "octo/platform-ops", repositoryPackageBootstrapAction{
		Type:             "github-app",
		AppIDVariable:    "APP_ID",
		PrivateKeySecret: "APP_PRIVATE_KEY",
	}, &bootstrapProfileExistingState{
		variables: map[string]struct{}{"APP_ID": {}},
		secrets:   map[string]struct{}{},
	})
	if err != nil {
		t.Fatalf("runBootstrapGitHubAppAction returned error: %v", err)
	}
	expected := []string{
		"var:APP_ID=Iv1.created",
		"secret:APP_PRIVATE_KEY=pem-value",
	}
	if !slices.Equal(writes, expected) {
		t.Fatalf("unexpected writes: %#v", writes)
	}
}

func TestBootstrapRepositoryInputEnvNames(t *testing.T) {
	if got := bootstrapRepositoryVariableEnvName("CENTRAL_AGENTIC_OPS_MODE"); got != "GH_AW_BOOTSTRAP_VAR_CENTRAL_AGENTIC_OPS_MODE" {
		t.Fatalf("unexpected variable env name: %s", got)
	}
	if got := bootstrapRepositorySecretEnvName("copilot-token.pem"); got != "GH_AW_BOOTSTRAP_SECRET_COPILOT_TOKEN_PEM" {
		t.Fatalf("unexpected secret env name: %s", got)
	}
}

func TestProfileSourcesUseActionsTokenCopilotAuth(t *testing.T) {
	workflowDir := t.TempDir()
	workflowPath := filepath.Join(workflowDir, "copilot.md")

	t.Run("rejects prompt-body false positive", func(t *testing.T) {
		content := "---\nengine: copilot\npermissions:\n  contents: read\n---\n\ncopilot-requests: write\n"
		if err := os.WriteFile(workflowPath, []byte(content), 0o644); err != nil {
			t.Fatalf("write workflow: %v", err)
		}
		ok, err := profileSourcesUseActionsTokenCopilotAuth(context.Background(), []string{workflowPath})
		if err != nil {
			t.Fatalf("profileSourcesUseActionsTokenCopilotAuth returned error: %v", err)
		}
		if ok {
			t.Fatal("expected missing frontmatter permission to disable actions-token auth")
		}
	})

	t.Run("accepts explicit frontmatter permission", func(t *testing.T) {
		content := "---\nengine: copilot\npermissions:\n  copilot-requests: write\n---\n"
		if err := os.WriteFile(workflowPath, []byte(content), 0o644); err != nil {
			t.Fatalf("write workflow: %v", err)
		}
		ok, err := profileSourcesUseActionsTokenCopilotAuth(context.Background(), []string{workflowPath})
		if err != nil {
			t.Fatalf("profileSourcesUseActionsTokenCopilotAuth returned error: %v", err)
		}
		if !ok {
			t.Fatal("expected explicit frontmatter permission to enable actions-token auth")
		}
	})
}

func TestIsRetryableBootstrapGitHubAppInstallationError(t *testing.T) {
	if !isRetryableBootstrapGitHubAppInstallationError(errors.New("gh: Not Found (HTTP 404)")) {
		t.Fatal("expected HTTP 404 to be retryable")
	}
	if isRetryableBootstrapGitHubAppInstallationError(errors.New("gh: Forbidden (HTTP 403)")) {
		t.Fatal("expected HTTP 403 to be non-retryable")
	}
}
