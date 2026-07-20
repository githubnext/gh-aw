//go:build !integration

package cli

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
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

func TestIsRetryableBootstrapGitHubAppInstallationError(t *testing.T) {
	if !isRetryableBootstrapGitHubAppInstallationError(errors.New("gh: Not Found (HTTP 404)")) {
		t.Fatal("expected HTTP 404 to be retryable")
	}
	if isRetryableBootstrapGitHubAppInstallationError(errors.New("gh: Forbidden (HTTP 403)")) {
		t.Fatal("expected HTTP 403 to be non-retryable")
	}
}

func TestBootstrapGitHubAppInstalled_UsesUserInstallationsForAllRepositories(t *testing.T) {
	originalRunGH := runBootstrapGHContext
	t.Cleanup(func() {
		runBootstrapGHContext = originalRunGH
	})

	var calls []string
	runBootstrapGHContext = func(_ context.Context, _ string, args ...string) ([]byte, error) {
		calls = append(calls, strings.Join(args, " "))
		if len(args) >= 2 && args[0] == "api" && args[1] == "/user/installations?per_page=100" {
			return []byte("123\tIv1.client\t987\tmy-mona-org-agenticops\tall\n"), nil
		}
		return nil, errors.New("unexpected gh api call")
	}

	installed, err := bootstrapGitHubAppInstalled(context.Background(), "my-mona-org/agenticops", &bootstrapCreatedGitHubApp{
		ClientID: "Iv1.client",
		AppID:    "987",
		Slug:     "my-mona-org-agenticops",
	})
	if err != nil {
		t.Fatalf("bootstrapGitHubAppInstalled returned error: %v", err)
	}
	if !installed {
		t.Fatal("expected installation to be detected")
	}
	if len(calls) != 1 {
		t.Fatalf("expected 1 gh api call, got %d", len(calls))
	}
	if strings.Contains(calls[0], "/repos/my-mona-org/agenticops/installation") {
		t.Fatalf("expected user installations endpoint, got %q", calls[0])
	}
}

func TestBootstrapGitHubAppInstalled_SelectedInstallationChecksRepositoryMembership(t *testing.T) {
	originalRunGH := runBootstrapGHContext
	t.Cleanup(func() {
		runBootstrapGHContext = originalRunGH
	})

	var calls []string
	runBootstrapGHContext = func(_ context.Context, _ string, args ...string) ([]byte, error) {
		calls = append(calls, strings.Join(args, " "))
		if len(args) >= 2 && args[0] == "api" && args[1] == "/user/installations?per_page=100" {
			return []byte("123\t\t987\tmy-mona-org-agenticops\tselected\n"), nil
		}
		if len(args) >= 2 && args[0] == "api" && args[1] == "/user/installations/123/repositories?per_page=100" {
			return []byte("my-mona-org/agenticops\n"), nil
		}
		return nil, errors.New("unexpected gh api call")
	}

	installed, err := bootstrapGitHubAppInstalled(context.Background(), "my-mona-org/agenticops", &bootstrapCreatedGitHubApp{
		AppID: "987",
		Slug:  "my-mona-org-agenticops",
	})
	if err != nil {
		t.Fatalf("bootstrapGitHubAppInstalled returned error: %v", err)
	}
	if !installed {
		t.Fatal("expected selected installation to include target repository")
	}
	if len(calls) != 2 {
		t.Fatalf("expected 2 gh api calls, got %d", len(calls))
	}
}

func TestHandleBootstrapGitHubAppExistingFlow(t *testing.T) {
	t.Run("no existing credentials", func(t *testing.T) {
		handled, err := handleBootstrapGitHubAppExistingFlow(context.Background(), "octo/platform-ops", repositoryPackageBootstrapAction{
			AppIDVariable:    "APP_ID",
			PrivateKeySecret: "APP_PRIVATE_KEY",
		}, bootstrapGitHubAppOverrides{}, "", "")
		if err != nil {
			t.Fatalf("handleBootstrapGitHubAppExistingFlow returned error: %v", err)
		}
		if handled {
			t.Fatal("expected flow to continue when no existing credentials are supplied")
		}
	})

	t.Run("applies supplied credentials", func(t *testing.T) {
		originalUpsertVariable := bootstrapUpsertVariable
		originalSetSecret := bootstrapSetSecret
		t.Cleanup(func() {
			bootstrapUpsertVariable = originalUpsertVariable
			bootstrapSetSecret = originalSetSecret
		})

		var writes []string
		bootstrapUpsertVariable = func(_ context.Context, _ string, name, value string) error {
			writes = append(writes, "var:"+name+"="+value)
			return nil
		}
		bootstrapSetSecret = func(_ context.Context, _ string, name, value string) error {
			writes = append(writes, "secret:"+name+"="+value)
			return nil
		}

		handled, err := handleBootstrapGitHubAppExistingFlow(context.Background(), "octo/platform-ops", repositoryPackageBootstrapAction{
			AppIDVariable:    "APP_ID",
			PrivateKeySecret: "APP_PRIVATE_KEY",
		}, bootstrapGitHubAppOverrides{}, "Iv1.client", "pem-value")
		if err != nil {
			t.Fatalf("handleBootstrapGitHubAppExistingFlow returned error: %v", err)
		}
		if !handled {
			t.Fatal("expected existing flow to be handled")
		}
		expected := []string{"var:APP_ID=Iv1.client", "secret:APP_PRIVATE_KEY=pem-value"}
		if !slices.Equal(writes, expected) {
			t.Fatalf("unexpected writes: %#v", writes)
		}
	})
}

func TestHandleBootstrapGitHubAppCreateOrExistingChoice_UsesExistingOverride(t *testing.T) {
	originalUpsertVariable := bootstrapUpsertVariable
	originalSetSecret := bootstrapSetSecret
	t.Cleanup(func() {
		bootstrapUpsertVariable = originalUpsertVariable
		bootstrapSetSecret = originalSetSecret
	})

	t.Setenv(bootstrapGitHubAppClientIDEnv, "Iv1.client")
	t.Setenv(bootstrapGitHubAppPrivateKeyEnv, "pem-value")

	var writes []string
	bootstrapUpsertVariable = func(_ context.Context, _ string, name, value string) error {
		writes = append(writes, "var:"+name+"="+value)
		return nil
	}
	bootstrapSetSecret = func(_ context.Context, _ string, name, value string) error {
		writes = append(writes, "secret:"+name+"="+value)
		return nil
	}

	handled, err := handleBootstrapGitHubAppCreateOrExistingChoice(context.Background(), "octo/platform-ops", repositoryPackageBootstrapAction{
		Mode:             "create-or-existing",
		AppIDVariable:    "APP_ID",
		PrivateKeySecret: "APP_PRIVATE_KEY",
	}, bootstrapGitHubAppOverrides{Mode: "existing"})
	if err != nil {
		t.Fatalf("handleBootstrapGitHubAppCreateOrExistingChoice returned error: %v", err)
	}
	if !handled {
		t.Fatal("expected existing override to be handled")
	}
	expected := []string{"var:APP_ID=Iv1.client", "secret:APP_PRIVATE_KEY=pem-value"}
	if !slices.Equal(writes, expected) {
		t.Fatalf("unexpected writes: %#v", writes)
	}
}

func TestChooseBootstrapGitHubAppMode_NonInteractiveError(t *testing.T) {
	originalInteractive := bootstrapIsInteractive
	t.Cleanup(func() {
		bootstrapIsInteractive = originalInteractive
	})
	bootstrapIsInteractive = func() bool { return false }

	if _, err := chooseBootstrapGitHubAppMode(); err == nil {
		t.Fatal("expected non-interactive mode selection error")
	}
}

func TestCompleteExistingGitHubAppCredentials_UsesProvidedValues(t *testing.T) {
	clientID, privateKey, err := completeExistingGitHubAppCredentials("Iv1.client", "pem-value", repositoryPackageBootstrapAction{
		AppIDVariable: "APP_ID",
	}, "octo/platform-ops")
	if err != nil {
		t.Fatalf("completeExistingGitHubAppCredentials returned error: %v", err)
	}
	if clientID != "Iv1.client" || privateKey != "pem-value" {
		t.Fatalf("unexpected credentials: %q %q", clientID, privateKey)
	}
}

func TestSetupBootstrapGitHubAppDetails(t *testing.T) {
	originalCheckOwnerType := bootstrapCheckOwnerType
	t.Cleanup(func() {
		bootstrapCheckOwnerType = originalCheckOwnerType
	})
	bootstrapCheckOwnerType = func(context.Context, string) (string, error) { return "Organization", nil }

	t.Run("defaults", func(t *testing.T) {
		appOwner, appOwnerType, appName, homepageURL, description, err := setupBootstrapGitHubAppDetails(context.Background(), "octo/platform-ops", "octo", "Organization", repositoryPackageBootstrapAction{}, bootstrapGitHubAppOverrides{})
		if err != nil {
			t.Fatalf("setupBootstrapGitHubAppDetails returned error: %v", err)
		}
		if appOwner != "octo" || appOwnerType != "Organization" || appName != "octo-platform-ops" {
			t.Fatalf("unexpected app details: %q %q %q", appOwner, appOwnerType, appName)
		}
		if homepageURL != "https://github.com/octo/platform-ops" || description != "Bootstrap app for octo/platform-ops" {
			t.Fatalf("unexpected defaults: %q %q", homepageURL, description)
		}
	})

	t.Run("owner override", func(t *testing.T) {
		appOwner, appOwnerType, appName, homepageURL, description, err := setupBootstrapGitHubAppDetails(context.Background(), "octo/platform-ops", "octo", "User", repositoryPackageBootstrapAction{
			AppName:     "configured-name",
			HomepageURL: "https://example.com",
			Description: "Configured description",
		}, bootstrapGitHubAppOverrides{Owner: "platform-eng"})
		if err != nil {
			t.Fatalf("setupBootstrapGitHubAppDetails returned error: %v", err)
		}
		if appOwner != "platform-eng" || appOwnerType != "Organization" || appName != "configured-name" {
			t.Fatalf("unexpected override details: %q %q %q", appOwner, appOwnerType, appName)
		}
		if homepageURL != "https://example.com" || description != "Configured description" {
			t.Fatalf("unexpected override values: %q %q", homepageURL, description)
		}
	})
}

func TestBuildBootstrapGitHubAppMux(t *testing.T) {
	originalExchange := bootstrapExchangeAppCode
	t.Cleanup(func() {
		bootstrapExchangeAppCode = originalExchange
	})

	t.Run("register page", func(t *testing.T) {
		mux := buildBootstrapGitHubAppMux(context.Background(), "state", "octo", "Organization", "app", "description", "<html>register</html>", bootstrapGitHubAppFlowChannels{
			resultCh: make(chan *bootstrapCreatedGitHubApp, 1),
			errCh:    make(chan error, 1),
		})
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/register", nil))
		if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "register") {
			t.Fatalf("unexpected register response: %d %q", rec.Code, rec.Body.String())
		}
	})

	t.Run("callback missing code", func(t *testing.T) {
		errCh := make(chan error, 1)
		mux := buildBootstrapGitHubAppMux(context.Background(), "state", "octo", "Organization", "app", "description", "<html>register</html>", bootstrapGitHubAppFlowChannels{
			resultCh: make(chan *bootstrapCreatedGitHubApp, 1),
			errCh:    errCh,
		})
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/callback?state=state", nil))
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected bad request, got %d", rec.Code)
		}
		if err := <-errCh; err == nil || !strings.Contains(err.Error(), "manifest code") {
			t.Fatalf("unexpected callback error: %v", err)
		}
	})

	t.Run("callback state mismatch", func(t *testing.T) {
		errCh := make(chan error, 1)
		mux := buildBootstrapGitHubAppMux(context.Background(), "state", "octo", "Organization", "app", "description", "<html>register</html>", bootstrapGitHubAppFlowChannels{
			resultCh: make(chan *bootstrapCreatedGitHubApp, 1),
			errCh:    errCh,
		})
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/callback?state=wrong&code=abc", nil))
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected bad request, got %d", rec.Code)
		}
		if err := <-errCh; err == nil || !strings.Contains(err.Error(), "state mismatch") {
			t.Fatalf("unexpected callback error: %v", err)
		}
	})

	t.Run("callback success", func(t *testing.T) {
		bootstrapExchangeAppCode = func(context.Context, string, string, string, string, string) (*bootstrapCreatedGitHubApp, error) {
			return &bootstrapCreatedGitHubApp{InstallURL: "https://example.com/install", ClientID: "Iv1.client"}, nil
		}
		resultCh := make(chan *bootstrapCreatedGitHubApp, 1)
		mux := buildBootstrapGitHubAppMux(context.Background(), "state", "octo", "Organization", "app", "description", "<html>register</html>", bootstrapGitHubAppFlowChannels{
			resultCh: resultCh,
			errCh:    make(chan error, 1),
		})
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/callback?state=state&code=abc", nil))
		if rec.Code != http.StatusFound {
			t.Fatalf("expected redirect, got %d", rec.Code)
		}
		select {
		case created := <-resultCh:
			if created.ClientID != "Iv1.client" {
				t.Fatalf("unexpected created app: %#v", created)
			}
		default:
			t.Fatal("expected created app result")
		}
	})
}

func TestCreateBootstrapGitHubApp_CanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := createBootstrapGitHubApp(ctx, "octo/platform-ops", "octo", "platform-ops", "Organization", repositoryPackageBootstrapAction{}, bootstrapGitHubAppOverrides{OpenBrowser: false})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context canceled, got %v", err)
	}
}

func TestWaitForBootstrapGitHubAppInstallation(t *testing.T) {
	t.Run("nil app", func(t *testing.T) {
		if err := waitForBootstrapGitHubAppInstallation(context.Background(), "octo/platform-ops", nil); err != nil {
			t.Fatalf("waitForBootstrapGitHubAppInstallation returned error: %v", err)
		}
	})

	t.Run("canceled context after retryable error", func(t *testing.T) {
		originalRunGH := runBootstrapGHContext
		t.Cleanup(func() {
			runBootstrapGHContext = originalRunGH
		})
		runBootstrapGHContext = func(_ context.Context, _ string, _ ...string) ([]byte, error) {
			return nil, errors.New("gh: Not Found (HTTP 404)")
		}
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		err := waitForBootstrapGitHubAppInstallation(ctx, "octo/platform-ops", &bootstrapCreatedGitHubApp{
			InstallURL: "https://example.com/install",
			Slug:       "agentic-ops",
		})
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("expected context canceled, got %v", err)
		}
	})
}

func TestBootstrapGitHubAppInstallationMatches(t *testing.T) {
	created := &bootstrapCreatedGitHubApp{ClientID: "Iv1.client", AppID: "42", Slug: "agentic-ops"}
	if !bootstrapGitHubAppInstallationMatches(bootstrapGitHubAppUserInstallation{ClientID: "Iv1.client"}, created) {
		t.Fatal("expected client ID match")
	}
	if !bootstrapGitHubAppInstallationMatches(bootstrapGitHubAppUserInstallation{AppSlug: "agentic-ops"}, created) {
		t.Fatal("expected slug match")
	}
	if !bootstrapGitHubAppInstallationMatches(bootstrapGitHubAppUserInstallation{AppID: "42"}, created) {
		t.Fatal("expected app ID match")
	}
	if bootstrapGitHubAppInstallationMatches(bootstrapGitHubAppUserInstallation{AppID: "99"}, created) {
		t.Fatal("expected mismatched installation to be false")
	}
}
