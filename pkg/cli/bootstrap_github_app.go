package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"charm.land/huh/v2"
	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/repoutil"
	"github.com/github/gh-aw/pkg/workflow"
)

// GitHub App bootstrap environment variable names.
const (
	bootstrapGitHubAppModeEnv        = "GH_AW_BOOTSTRAP_GITHUB_APP_MODE"
	bootstrapGitHubAppOwnerEnv       = "GH_AW_BOOTSTRAP_GITHUB_APP_OWNER"
	bootstrapGitHubAppNameEnv        = "GH_AW_BOOTSTRAP_GITHUB_APP_NAME"
	bootstrapGitHubAppURLEnv         = "GH_AW_BOOTSTRAP_GITHUB_APP_URL"
	bootstrapGitHubAppDescriptionEnv = "GH_AW_BOOTSTRAP_GITHUB_APP_DESCRIPTION"
	bootstrapGitHubAppClientIDEnv    = "GH_AW_BOOTSTRAP_GITHUB_APP_CLIENT_ID"
	bootstrapGitHubAppPrivateKeyEnv  = "GH_AW_BOOTSTRAP_GITHUB_APP_PRIVATE_KEY"
	bootstrapNoOpenBrowserEnv        = "GH_AW_BOOTSTRAP_NO_OPEN_BROWSER"
)

// bootstrapGitHubAppOverrides holds environment-variable overrides for the
// GitHub App creation flow.
type bootstrapGitHubAppOverrides struct {
	Mode        string
	Owner       string
	Name        string
	HomepageURL string
	Description string
	OpenBrowser bool
}

// bootstrapCreatedGitHubApp carries the credentials and metadata returned
// after a GitHub App is created or retrieved.
type bootstrapCreatedGitHubApp struct {
	Owner       string
	OwnerType   string
	Name        string
	SettingsURL string
	InstallURL  string
	ClientID    string
	AppID       string
	PEM         string
	Slug        string
}

// bootstrapGitHubAppExchangeResponse is the API payload returned by the
// GitHub App manifest code exchange endpoint.
type bootstrapGitHubAppExchangeResponse struct {
	HTMLURL  string `json:"html_url"`
	ClientID string `json:"client_id"`
	ID       int64  `json:"id"`
	PEM      string `json:"pem"`
	Slug     string `json:"slug"`
	Name     string `json:"name"`
}

// bootstrapGitHubAppRepositoryInstallation is the API payload returned by the
// repository installation endpoint.
type bootstrapGitHubAppRepositoryInstallation struct {
	ClientID string `json:"client_id"`
	AppID    int64  `json:"app_id"`
	AppSlug  string `json:"app_slug"`
	ID       int64  `json:"id"`
}

// runBootstrapGitHubAppAction creates or retrieves GitHub App credentials and
// stores them as a repository variable (client ID) and secret (private key).
func runBootstrapGitHubAppAction(ctx context.Context, repo string, action repositoryPackageBootstrapAction, state *bootstrapProfileExistingState) (*bootstrapCreatedGitHubApp, error) {
	_, hasVar := state.variables[action.AppIDVariable]
	_, hasSecret := state.secrets[action.PrivateKeySecret]
	if hasVar && hasSecret {
		return nil, nil
	}

	overrides, err := loadBootstrapGitHubAppOverrides()
	if err != nil {
		return nil, err
	}

	owner, repoName, err := repoutil.SplitRepoSlug(repo)
	if err != nil {
		return nil, err
	}
	ownerType, err := bootstrapCheckOwnerType(ctx, owner)
	if err != nil {
		return nil, err
	}

	var clientID string
	var privateKey string
	clientID = strings.TrimSpace(os.Getenv(bootstrapGitHubAppClientIDEnv))
	privateKey = strings.TrimRight(os.Getenv(bootstrapGitHubAppPrivateKeyEnv), "\r\n")
	if clientID != "" || privateKey != "" || action.Mode == "existing" || overrides.Mode == "existing" {
		resolvedClientID, resolvedPrivateKey, err := completeExistingGitHubAppCredentials(clientID, privateKey, action, repo)
		if err != nil {
			return nil, err
		}
		if err := bootstrapUpsertVariable(ctx, repo, action.AppIDVariable, resolvedClientID); err != nil {
			return nil, err
		}
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Set repository variable "+action.AppIDVariable))
		if err := bootstrapSetSecret(ctx, repo, action.PrivateKeySecret, resolvedPrivateKey); err != nil {
			return nil, err
		}
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Set repository secret "+action.PrivateKeySecret))
		return nil, nil
	}

	createNew := action.Mode == "create-or-existing" || overrides.Mode == "create"
	if createNew {
		choice := overrides.Mode
		if choice == "" {
			choice, err = chooseBootstrapGitHubAppMode()
			if err != nil {
				return nil, err
			}
		}
		if choice == "existing" {
			resolvedClientID, resolvedPrivateKey, err := completeExistingGitHubAppCredentials(clientID, privateKey, action, repo)
			if err != nil {
				return nil, err
			}
			if err := bootstrapUpsertVariable(ctx, repo, action.AppIDVariable, resolvedClientID); err != nil {
				return nil, err
			}
			if err := bootstrapSetSecret(ctx, repo, action.PrivateKeySecret, resolvedPrivateKey); err != nil {
				return nil, err
			}
			fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Configured existing GitHub App credentials"))
			return nil, nil
		}
	}

	if !bootstrapIsInteractive() && overrides.Mode != "create" {
		return nil, fmt.Errorf("creating a new GitHub App requires an interactive browser flow; provide existing credentials via %s and %s, or set %s=create to force browser-based creation. Example: export %s=Iv23example and %s='-----BEGIN PRIVATE KEY-----...'", bootstrapGitHubAppClientIDEnv, bootstrapGitHubAppPrivateKeyEnv, bootstrapGitHubAppModeEnv, bootstrapGitHubAppClientIDEnv, bootstrapGitHubAppPrivateKeyEnv)
	}
	createdApp, err := bootstrapCreateGitHubApp(ctx, repo, owner, repoName, ownerType, action, overrides)
	if err != nil {
		return nil, err
	}
	if err := bootstrapUpsertVariable(ctx, repo, action.AppIDVariable, createdApp.ClientID); err != nil {
		return nil, err
	}
	fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Set repository variable "+action.AppIDVariable))
	if err := bootstrapSetSecret(ctx, repo, action.PrivateKeySecret, createdApp.PEM); err != nil {
		return nil, err
	}
	fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Set repository secret "+action.PrivateKeySecret))
	if createdApp.InstallURL != "" {
		if err := waitForBootstrapGitHubAppInstallation(ctx, repo, createdApp); err != nil {
			return nil, err
		}
	}
	return createdApp, nil
}

func chooseBootstrapGitHubAppMode() (string, error) {
	if !bootstrapIsInteractive() {
		return "", fmt.Errorf("choose an existing GitHub App or set %s=create to allow browser-based creation in non-interactive environments. Example: export %s=existing", bootstrapGitHubAppModeEnv, bootstrapGitHubAppModeEnv)
	}
	var choice string
	form := console.NewSelectForm(huh.NewSelect[string]().
		Title("How should gh aw configure the GitHub App?").
		Description("Create a new GitHub App in the browser or provide credentials for an existing app.").
		Options(
			huh.NewOption("Create a new GitHub App", "create"),
			huh.NewOption("Use existing GitHub App credentials", "existing"),
		).
		Value(&choice))
	if err := form.Run(); err != nil {
		return "", err
	}
	if choice == "" {
		choice = "create"
	}
	return choice, nil
}

func completeExistingGitHubAppCredentials(existingClientID string, existingPrivateKey string, action repositoryPackageBootstrapAction, repo string) (string, string, error) {
	clientID := strings.TrimSpace(existingClientID)
	privateKey := strings.TrimSpace(existingPrivateKey)
	var err error
	if clientID == "" {
		clientID, _, err = resolveBootstrapTextValue(bootstrapGitHubAppClientIDEnv, "GitHub App client ID", "Enter the GitHub App client ID to store in "+action.AppIDVariable+".", "", nil, false)
		if err != nil {
			return "", "", err
		}
	}
	if privateKey == "" {
		privateKey, _, err = resolveBootstrapSecretValue(bootstrapGitHubAppPrivateKeyEnv, "GitHub App private key", "Paste the PEM private key for the GitHub App used by "+repo+".", false)
		if err != nil {
			return "", "", err
		}
	}
	return clientID, privateKey, nil
}

// createBootstrapGitHubApp drives the GitHub App manifest browser flow: it
// spins up a local HTTP server, opens the registration URL in the browser,
// waits for the OAuth callback, and exchanges the code for app credentials.
func createBootstrapGitHubApp(ctx context.Context, repo, owner, repoName, ownerType string, action repositoryPackageBootstrapAction, overrides bootstrapGitHubAppOverrides) (*bootstrapCreatedGitHubApp, error) {
	state, err := bootstrapRandomHex(16)
	if err != nil {
		return nil, err
	}

	listener, err := netListener()
	if err != nil {
		return nil, err
	}
	defer listener.Close()

	appOwner := owner
	appOwnerType := ownerType
	if overrides.Owner != "" {
		appOwner = overrides.Owner
		appOwnerType, err = bootstrapCheckOwnerType(ctx, appOwner)
		if err != nil {
			return nil, err
		}
	}

	appName := deriveBootstrapAppName(repo, firstNonEmpty(overrides.Name, action.AppName))
	homepageURL := strings.TrimSpace(firstNonEmpty(overrides.HomepageURL, action.HomepageURL))
	if homepageURL == "" {
		homepageURL = "https://github.com/" + repo
	}
	description := strings.TrimSpace(firstNonEmpty(overrides.Description, action.Description))
	if description == "" {
		description = "Bootstrap app for " + repo
	}

	resultCh := make(chan *bootstrapCreatedGitHubApp, 1)
	errCh := make(chan error, 1)
	server := &http.Server{}
	redirectURL := fmt.Sprintf("http://%s/callback", listener.Addr().String())
	manifest := buildBootstrapGitHubAppManifest(action, appName, homepageURL, redirectURL, description)
	bootstrapLog.Printf("Creating GitHub App via browser manifest flow: appOwner=%s, appName=%s, redirectURL=%s", appOwner, appName, redirectURL)
	registrationURL := buildBootstrapGitHubAppRegistrationURL(appOwner, appOwnerType, state)
	registrationPage, err := renderBootstrapGitHubAppRegistrationPage(registrationURL, manifest)
	if err != nil {
		return nil, fmt.Errorf("failed to encode GitHub App registration manifest for browser handoff; report this issue if it persists: %w", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/register", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(registrationPage))
	})
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		returnedState := r.URL.Query().Get("state")
		code := r.URL.Query().Get("code")
		if code == "" {
			http.Error(w, "Missing GitHub App manifest code.", http.StatusBadRequest)
			select {
			case errCh <- errors.New("GitHub did not return an app manifest code"):
			default:
			}
			return
		}
		if returnedState != state {
			http.Error(w, "State mismatch while creating the GitHub App.", http.StatusBadRequest)
			select {
			case errCh <- errors.New("state mismatch while creating the GitHub App"):
			default:
			}
			return
		}
		createdApp, exchangeErr := exchangeBootstrapGitHubAppCode(ctx, code, owner, ownerType, appName, description)
		if exchangeErr != nil {
			http.Error(w, "GitHub App creation completed, but gh aw could not exchange the manifest code.", http.StatusInternalServerError)
			select {
			case errCh <- exchangeErr:
			default:
			}
			return
		}
		if createdApp.InstallURL != "" {
			http.Redirect(w, r, createdApp.InstallURL, http.StatusFound)
		} else {
			w.WriteHeader(http.StatusNoContent)
		}
		select {
		case resultCh <- createdApp:
		default:
		}
	})
	server.Handler = mux

	go func() {
		_ = server.Serve(listener)
	}()
	defer func() {
		_ = server.Shutdown(context.Background())
	}()

	printBootstrapGitHubAppManifestReview(appOwner, manifest)
	openURL := fmt.Sprintf("http://%s/register", listener.Addr().String())
	opened := false
	if overrides.OpenBrowser {
		opened = openBootstrapBrowser(openURL)
	}
	if !opened {
		fmt.Fprintln(os.Stderr, console.FormatCommandMessage(openURL))
	}

	timeout := time.NewTimer(bootstrapProfileManifestTimeout)
	defer timeout.Stop()

	select {
	case createdApp := <-resultCh:
		return createdApp, nil
	case err := <-errCh:
		return nil, err
	case <-timeout.C:
		return nil, errors.New("timed out waiting for GitHub App creation to complete in the browser")
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// loadBootstrapGitHubAppOverrides reads GitHub App creation overrides from
// well-known environment variables.
func loadBootstrapGitHubAppOverrides() (bootstrapGitHubAppOverrides, error) {
	overrides := bootstrapGitHubAppOverrides{
		Mode:        "",
		Owner:       strings.TrimSpace(os.Getenv(bootstrapGitHubAppOwnerEnv)),
		Name:        strings.TrimSpace(os.Getenv(bootstrapGitHubAppNameEnv)),
		HomepageURL: strings.TrimSpace(os.Getenv(bootstrapGitHubAppURLEnv)),
		Description: strings.TrimSpace(os.Getenv(bootstrapGitHubAppDescriptionEnv)),
		OpenBrowser: true,
	}

	switch mode := strings.ToLower(strings.TrimSpace(os.Getenv(bootstrapGitHubAppModeEnv))); mode {
	case "", "auto":
	case "create", "existing":
		overrides.Mode = mode
	default:
		return bootstrapGitHubAppOverrides{}, fmt.Errorf("%s must be one of: auto, create, existing. Example: export %s=create", bootstrapGitHubAppModeEnv, bootstrapGitHubAppModeEnv)
	}

	if raw := strings.TrimSpace(os.Getenv(bootstrapNoOpenBrowserEnv)); raw != "" {
		disabled, err := parseBootstrapBool(raw)
		if err != nil {
			return bootstrapGitHubAppOverrides{}, fmt.Errorf("%s: %w", bootstrapNoOpenBrowserEnv, err)
		}
		overrides.OpenBrowser = !disabled
	}

	return overrides, nil
}

func exchangeBootstrapGitHubAppCode(ctx context.Context, code, owner, ownerType, appName, description string) (*bootstrapCreatedGitHubApp, error) {
	output, err := workflow.RunGHContext(ctx, "Exchanging GitHub App manifest code...", "api", "-X", "POST", "-H", "Accept: application/vnd.github+json", "/app-manifests/"+code+"/conversions")
	if err != nil {
		return nil, err
	}
	var payload bootstrapGitHubAppExchangeResponse
	if err := json.Unmarshal(output, &payload); err != nil {
		return nil, fmt.Errorf("failed to parse GitHub App manifest exchange response: %w", err)
	}
	return &bootstrapCreatedGitHubApp{
		Owner:       owner,
		OwnerType:   ownerType,
		Name:        firstNonEmpty(payload.Name, appName),
		SettingsURL: payload.HTMLURL,
		InstallURL:  buildBootstrapGitHubAppInstallURL(payload.Slug),
		ClientID:    payload.ClientID,
		AppID:       strconv.FormatInt(payload.ID, 10),
		PEM:         payload.PEM,
		Slug:        payload.Slug,
	}, nil
}

func waitForBootstrapGitHubAppInstallation(ctx context.Context, repo string, createdApp *bootstrapCreatedGitHubApp) error {
	if createdApp == nil || createdApp.InstallURL == "" || createdApp.Slug == "" {
		return nil
	}
	bootstrapLog.Printf("Polling for GitHub App installation: repo=%s, slug=%s", repo, createdApp.Slug)
	deadlineTimer := time.NewTimer(bootstrapProfileManifestTimeout)
	defer deadlineTimer.Stop()
	pollTicker := time.NewTicker(bootstrapProfileInstallPollDelay)
	defer pollTicker.Stop()
	var lastErr error
	for {
		installed, err := bootstrapGitHubAppInstalled(ctx, repo, createdApp)
		if err == nil && installed {
			fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("GitHub App installation detected for "+repo))
			return nil
		}
		if err != nil {
			if !isRetryableBootstrapGitHubAppInstallationError(err) {
				return fmt.Errorf("failed to check GitHub App installation for %s: %w", repo, err)
			}
			lastErr = err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-deadlineTimer.C:
			if lastErr != nil {
				return fmt.Errorf("timed out waiting for the GitHub App installation to complete for %s: %w", repo, lastErr)
			}
			return fmt.Errorf("timed out waiting for the GitHub App installation to complete for %s", repo)
		case <-pollTicker.C:
		}
	}
}

func bootstrapGitHubAppInstalled(ctx context.Context, repo string, createdApp *bootstrapCreatedGitHubApp) (bool, error) {
	output, err := runBootstrapGHContext(ctx, "Checking GitHub App installation...", "api", "/repos/"+repo+"/installation")
	if err != nil {
		return false, err
	}
	var payload bootstrapGitHubAppRepositoryInstallation
	if err := json.Unmarshal(output, &payload); err != nil {
		return false, err
	}
	if payload.ClientID != "" && payload.ClientID == createdApp.ClientID {
		return payload.ID > 0, nil
	}
	if payload.AppSlug == createdApp.Slug || strconv.FormatInt(payload.AppID, 10) == createdApp.AppID {
		return payload.ID > 0, nil
	}
	return false, nil
}

func buildBootstrapGitHubAppManifest(action repositoryPackageBootstrapAction, appName, homepageURL, redirectURL, description string) map[string]any {
	permissions := action.Permissions
	if len(permissions) == 0 {
		permissions = map[string]string{
			"metadata": "read",
		}
	}
	events := action.Events
	if events == nil {
		events = []string{}
	}
	return map[string]any{
		"name":                     appName,
		"url":                      homepageURL,
		"hook_attributes":          map[string]any{"url": homepageURL, "active": false},
		"redirect_url":             redirectURL,
		"public":                   false,
		"request_oauth_on_install": false,
		"description":              description,
		"default_permissions":      permissions,
		"default_events":           events,
	}
}

func buildBootstrapGitHubAppRegistrationURL(owner, ownerType, state string) string {
	if strings.EqualFold(ownerType, "Organization") {
		return fmt.Sprintf("https://github.com/organizations/%s/settings/apps/new?state=%s", owner, state)
	}
	return "https://github.com/settings/apps/new?state=" + state
}

func renderBootstrapGitHubAppRegistrationPage(registrationURL string, manifest map[string]any) (string, error) {
	encoded, err := json.Marshal(manifest)
	if err != nil {
		return "", err
	}
	return "<!doctype html><html lang=\"en\"><head><meta charset=\"utf-8\"><title>Redirecting To GitHub App Creation</title></head><body><p>Redirecting to GitHub App creation...</p><form id=\"manifest-form\" action=\"" + htmlEscape(registrationURL) + "\" method=\"post\"><input type=\"hidden\" name=\"manifest\" value=\"" + htmlEscape(string(encoded)) + "\"><noscript><button type=\"submit\">Continue To GitHub App Creation</button></noscript></form><script>document.getElementById('manifest-form').submit();</script></body></html>", nil
}

func printBootstrapGitHubAppManifestReview(owner string, manifest map[string]any) {
	permissions := map[string]string{}
	switch raw := manifest["default_permissions"].(type) {
	case map[string]string:
		permissions = raw
	case map[string]any:
		for name, value := range raw {
			text, ok := value.(string)
			if ok {
				permissions[name] = text
				continue
			}
			permissions[name] = "<non-string-value>"
		}
	}
	permissionNames := make([]string, 0, len(permissions))
	for name := range permissions {
		permissionNames = append(permissionNames, name)
	}
	sort.Strings(permissionNames)
	getManifestStringOrDefault := func(key string) string {
		value, ok := manifest[key].(string)
		if !ok {
			return "<unavailable>"
		}
		if strings.TrimSpace(value) == "" {
			return "<unavailable>"
		}
		return value
	}
	lines := []string{
		"GitHub App manifest for " + owner + ":",
		"- name: " + getManifestStringOrDefault("name"),
		"- homepage: " + getManifestStringOrDefault("url"),
		"- redirect URL: " + getManifestStringOrDefault("redirect_url"),
		"- description: " + getManifestStringOrDefault("description"),
		"- permissions:",
	}
	for _, name := range permissionNames {
		lines = append(lines, fmt.Sprintf("  - %s: %s", name, permissions[name]))
	}
	lines = append(lines, "")
	for _, line := range lines {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(line))
	}
}

func buildBootstrapGitHubAppInstallURL(slug string) string {
	if strings.TrimSpace(slug) == "" {
		return ""
	}
	return "https://github.com/apps/" + slug + "/installations/new"
}

func isRetryableBootstrapGitHubAppInstallationError(err error) bool {
	if err == nil {
		return false
	}
	message := err.Error()
	return strings.Contains(message, "HTTP 404") ||
		strings.Contains(message, "HTTP 500") ||
		strings.Contains(message, "HTTP 502") ||
		strings.Contains(message, "HTTP 503") ||
		strings.Contains(message, "HTTP 504")
}
