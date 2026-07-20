//go:build !integration

package cli

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestBuildBootstrapGitHubAppMux(t *testing.T) {
	originalExchange := bootstrapExchangeGitHubAppCode
	t.Cleanup(func() {
		bootstrapExchangeGitHubAppCode = originalExchange
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

	t.Run("callback rejects invalid code", func(t *testing.T) {
		errCh := make(chan error, 1)
		mux := buildBootstrapGitHubAppMux(context.Background(), "state", "octo", "Organization", "app", "description", "<html>register</html>", bootstrapGitHubAppFlowChannels{
			resultCh: make(chan *bootstrapCreatedGitHubApp, 1),
			errCh:    errCh,
		})
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/callback?state=state&code=../bad", nil))
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected bad request, got %d", rec.Code)
		}
		if err := <-errCh; err == nil || !strings.Contains(err.Error(), "invalid GitHub App manifest code format") {
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
		bootstrapExchangeGitHubAppCode = func(context.Context, string, string, string, string, string) (*bootstrapCreatedGitHubApp, error) {
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
