package cli

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	"charm.land/huh/v2"
	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/parser"
	"github.com/github/gh-aw/pkg/repoutil"
	"github.com/github/gh-aw/pkg/stringutil"
	"github.com/github/gh-aw/pkg/tty"
	"github.com/github/gh-aw/pkg/workflow"
)

const (
	bootstrapProfileManifestTimeout  = 10 * time.Minute
	bootstrapProfileInstallPollDelay = 5 * time.Second
	bootstrapGitHubAppModeEnv        = "GH_AW_BOOTSTRAP_GITHUB_APP_MODE"
	bootstrapGitHubAppOwnerEnv       = "GH_AW_BOOTSTRAP_GITHUB_APP_OWNER"
	bootstrapGitHubAppNameEnv        = "GH_AW_BOOTSTRAP_GITHUB_APP_NAME"
	bootstrapGitHubAppURLEnv         = "GH_AW_BOOTSTRAP_GITHUB_APP_URL"
	bootstrapGitHubAppDescriptionEnv = "GH_AW_BOOTSTRAP_GITHUB_APP_DESCRIPTION"
	bootstrapGitHubAppClientIDEnv    = "GH_AW_BOOTSTRAP_GITHUB_APP_CLIENT_ID"
	bootstrapGitHubAppPrivateKeyEnv  = "GH_AW_BOOTSTRAP_GITHUB_APP_PRIVATE_KEY"
	bootstrapNoOpenBrowserEnv        = "GH_AW_BOOTSTRAP_NO_OPEN_BROWSER"
)

var (
	runBootstrapGHContext    = workflow.RunGHContext
	bootstrapIsInteractive   = tty.IsStderrTerminal
	bootstrapUpsertVariable  = upsertBootstrapRepoVariable
	bootstrapSetSecret       = setBootstrapRepoSecret
	bootstrapCreateGitHubApp = createBootstrapGitHubApp
	bootstrapCheckOwnerType  = checkSetupRepositoryOwnerType
)

type bootstrapProfileRunConfig struct {
	Repo     string
	RepoDir  string
	Sources  []string
	Profile  *resolvedBootstrapProfile
	Yes      bool
	PlanOnly bool
	Verbose  bool
	Force    bool
}

type bootstrapProfileExistingState struct {
	variables map[string]struct{}
	secrets   map[string]struct{}
}

type bootstrapGitHubAppOverrides struct {
	Mode        string
	Owner       string
	Name        string
	HomepageURL string
	Description string
	OpenBrowser bool
}

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

type bootstrapGitHubAppExchangeResponse struct {
	HTMLURL  string `json:"html_url"`
	ClientID string `json:"client_id"`
	ID       int64  `json:"id"`
	PEM      string `json:"pem"`
	Slug     string `json:"slug"`
	Name     string `json:"name"`
}

type bootstrapGitHubAppRepositoryInstallation struct {
	ClientID string `json:"client_id"`
	AppID    int64  `json:"app_id"`
	AppSlug  string `json:"app_slug"`
	ID       int64  `json:"id"`
}

func buildBootstrapProfilePlan(ctx context.Context, repo string, profile *resolvedBootstrapProfile, sources []string, repoReady bool) (bool, []string, error) {
	if profile == nil || profile.Profile == nil {
		return false, nil, nil
	}

	lines := make([]string, 0, len(profile.Profile.Actions))
	if !repoReady {
		for _, action := range profile.Profile.Actions {
			if err := validateBootstrapActionPreRepo(ctx, repo, action); err != nil {
				return false, nil, err
			}
			if bootstrapActionCanMutate(action, sources) {
				lines = append(lines, "- bootstrap profile will configure "+bootstrapActionPlanLabel(action))
			}
		}
		return len(lines) > 0, lines, nil
	}

	state, err := bootstrapProfileState(ctx, repo)
	if err != nil {
		return false, nil, err
	}
	usesActionsToken, err := profileSourcesUseActionsTokenCopilotAuth(ctx, sources)
	if err != nil {
		return false, nil, err
	}

	needsMutation := false
	for _, action := range profile.Profile.Actions {
		pending, err := bootstrapActionNeedsMutation(ctx, repo, action, state, usesActionsToken)
		if err != nil {
			return false, nil, err
		}
		if pending {
			needsMutation = true
			lines = append(lines, "- bootstrap profile will configure "+bootstrapActionPlanLabel(action))
		}
	}

	return needsMutation, lines, nil
}

func executeBootstrapProfile(ctx context.Context, config bootstrapProfileRunConfig) error {
	if config.Profile == nil || config.Profile.Profile == nil {
		return nil
	}

	state, err := bootstrapProfileState(ctx, config.Repo)
	if err != nil {
		return err
	}
	usesActionsToken, err := profileSourcesUseActionsTokenCopilotAuth(ctx, config.Sources)
	if err != nil {
		return err
	}

	for _, action := range config.Profile.Profile.Actions {
		pending, err := bootstrapActionNeedsMutation(ctx, config.Repo, action, state, usesActionsToken)
		if err != nil {
			return err
		}
		if !pending && action.Type != "handoff" {
			continue
		}

		switch action.Type {
		case "require-owner-type":
			if err := runBootstrapRequireOwnerType(ctx, config.Repo, action); err != nil {
				return err
			}
		case "repo-variable":
			applied, err := runBootstrapRepoVariableAction(ctx, config.Repo, action, state)
			if err != nil {
				return err
			}
			if applied {
				state.variables[action.Name] = struct{}{}
			}
		case "repo-secret":
			applied, err := runBootstrapRepoSecretAction(ctx, config.Repo, action, state)
			if err != nil {
				return err
			}
			if applied {
				state.secrets[action.Name] = struct{}{}
			}
		case "github-app":
			_, err := runBootstrapGitHubAppAction(ctx, config.Repo, action, state)
			if err != nil {
				return err
			}
			state.variables[action.AppIDVariable] = struct{}{}
			state.secrets[action.PrivateKeySecret] = struct{}{}
		case "copilot-auth":
			applied, err := runBootstrapCopilotAuthAction(ctx, config.Repo, action, state, usesActionsToken)
			if err != nil {
				return err
			}
			if applied {
				state.secrets[action.Secret] = struct{}{}
			}
		case "handoff":
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage(action.Message))
		default:
			return fmt.Errorf("unsupported bootstrap action type %q. Example: use one of %s", action.Type, bootstrapActionTypeExample)
		}
	}

	return nil
}

func bootstrapProfileState(ctx context.Context, repo string) (*bootstrapProfileExistingState, error) {
	variableNames, err := listBootstrapRepoVariableNames(ctx, repo)
	if err != nil {
		return nil, err
	}
	secretNames, err := listBootstrapRepoSecretNames(ctx, repo)
	if err != nil {
		return nil, err
	}

	state := &bootstrapProfileExistingState{
		variables: make(map[string]struct{}, len(variableNames)),
		secrets:   make(map[string]struct{}, len(secretNames)),
	}
	for _, name := range variableNames {
		state.variables[name] = struct{}{}
	}
	for _, name := range secretNames {
		state.secrets[name] = struct{}{}
	}
	return state, nil
}

func bootstrapActionNeedsMutation(ctx context.Context, repo string, action repositoryPackageBootstrapAction, state *bootstrapProfileExistingState, usesActionsToken bool) (bool, error) {
	switch action.Type {
	case "require-owner-type":
		return false, runBootstrapRequireOwnerType(ctx, repo, action)
	case "repo-variable":
		_, exists := state.variables[action.Name]
		return !exists, nil
	case "repo-secret":
		_, exists := state.secrets[action.Name]
		return !exists, nil
	case "github-app":
		_, hasVar := state.variables[action.AppIDVariable]
		_, hasSecret := state.secrets[action.PrivateKeySecret]
		return !hasVar || !hasSecret, nil
	case "copilot-auth":
		_, hasSecret := state.secrets[action.Secret]
		return !hasSecret && !usesActionsToken, nil
	case "handoff":
		return false, nil
	default:
		return false, fmt.Errorf("unsupported bootstrap action type %q. Example: use one of %s", action.Type, bootstrapActionTypeExample)
	}
}

func validateBootstrapActionPreRepo(ctx context.Context, repo string, action repositoryPackageBootstrapAction) error {
	if action.Type == "require-owner-type" {
		return runBootstrapRequireOwnerType(ctx, repo, action)
	}
	return nil
}

func bootstrapActionCanMutate(action repositoryPackageBootstrapAction, sources []string) bool {
	switch action.Type {
	case "repo-variable", "repo-secret", "github-app":
		return true
	case "copilot-auth":
		return true
	default:
		return false
	}
}

func bootstrapActionPlanLabel(action repositoryPackageBootstrapAction) string {
	switch action.Type {
	case "repo-variable":
		return "repository variable " + action.Name
	case "repo-secret":
		return "repository secret " + action.Name
	case "github-app":
		return fmt.Sprintf("GitHub App credentials (%s, %s)", action.AppIDVariable, action.PrivateKeySecret)
	case "copilot-auth":
		return "Copilot secret " + action.Secret
	default:
		return action.Type
	}
}

func runBootstrapRequireOwnerType(ctx context.Context, repo string, action repositoryPackageBootstrapAction) error {
	owner, _, err := repoutil.SplitRepoSlug(repo)
	if err != nil {
		return err
	}
	ownerType, err := bootstrapCheckOwnerType(ctx, owner)
	if err != nil {
		return err
	}
	normalized := normalizeSetupOwnerType(ownerType)
	if action.Value != "" && action.Value != "any" && normalized != action.Value {
		return fmt.Errorf("owner %s is %s, but bootstrap profile requires %s. Example: set bootstrap.actions[].value to %s or use a repository owned by a matching account type", owner, normalized, action.Value, normalized)
	}
	return nil
}

func runBootstrapRepoVariableAction(ctx context.Context, repo string, action repositoryPackageBootstrapAction, state *bootstrapProfileExistingState) (bool, error) {
	if _, exists := state.variables[action.Name]; exists {
		return false, nil
	}
	value, ok, err := resolveBootstrapTextValue(bootstrapRepositoryVariableEnvName(action.Name), action.Prompt, action.Description, action.Default, action.Enum, action.Optional)
	if err != nil {
		return false, err
	}
	if !ok {
		return false, nil
	}
	if err := bootstrapUpsertVariable(ctx, repo, action.Name, value); err != nil {
		return false, err
	}
	fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Set repository variable "+action.Name))
	return true, nil
}

func runBootstrapRepoSecretAction(ctx context.Context, repo string, action repositoryPackageBootstrapAction, state *bootstrapProfileExistingState) (bool, error) {
	if _, exists := state.secrets[action.Name]; exists {
		return false, nil
	}
	value, ok, err := resolveBootstrapSecretValue(bootstrapRepositorySecretEnvName(action.Name), action.Prompt, action.Description, action.Optional)
	if err != nil {
		return false, err
	}
	if !ok {
		return false, nil
	}
	if err := bootstrapSetSecret(ctx, repo, action.Name, value); err != nil {
		return false, err
	}
	fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Set repository secret "+action.Name))
	return true, nil
}

func runBootstrapCopilotAuthAction(ctx context.Context, repo string, action repositoryPackageBootstrapAction, state *bootstrapProfileExistingState, usesActionsToken bool) (bool, error) {
	if usesActionsToken {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Skipping Copilot PAT setup because selected workflows already support GitHub Actions token auth."))
		return false, nil
	}
	if _, exists := state.secrets[action.Secret]; exists {
		return false, nil
	}
	value, ok, err := resolveBootstrapSecretValue(action.Secret, "Copilot fine-grained PAT", "Enter a fine-grained personal access token starting with github_pat_.", false)
	if err != nil {
		return false, err
	}
	if !ok {
		return false, nil
	}
	if err := stringutil.ValidateCopilotPAT(value); err != nil {
		return false, err
	}
	if err := bootstrapSetSecret(ctx, repo, action.Secret, value); err != nil {
		return false, err
	}
	fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Set repository secret "+action.Secret))
	return true, nil
}

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

func parseBootstrapBool(raw string) (bool, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "1", "true", "yes", "on":
		return true, nil
	case "0", "false", "no", "off":
		return false, nil
	default:
		return false, errors.New("expected one of: 1, true, yes, on, 0, false, no, off. Example: GH_AW_BOOTSTRAP_NO_OPEN_BROWSER=true")
	}
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

func listBootstrapRepoVariableNames(ctx context.Context, repo string) ([]string, error) {
	output, err := runBootstrapGHContext(ctx, "Checking repository variables...", "api", fmt.Sprintf("/repos/%s/actions/variables?per_page=100", repo), "--paginate", "--jq", ".variables[].name")
	if err != nil {
		return nil, fmt.Errorf("failed to list repository variables for %s: %w", repo, err)
	}
	return parseBootstrapNames(output), nil
}

func listBootstrapRepoSecretNames(ctx context.Context, repo string) ([]string, error) {
	output, err := runBootstrapGHContext(ctx, "Checking repository secrets...", "api", fmt.Sprintf("/repos/%s/actions/secrets?per_page=100", repo), "--paginate", "--jq", ".secrets[].name")
	if err != nil {
		return nil, fmt.Errorf("failed to list repository secrets for %s: %w", repo, err)
	}
	return parseBootstrapNames(output), nil
}

func upsertBootstrapRepoVariable(ctx context.Context, repo, name, value string) error {
	target := defaultsTarget{}
	owner, repoName, err := repoutil.SplitRepoSlug(repo)
	if err != nil {
		return err
	}
	target.scope = defaultsScopeRepo
	target.repoOwner = owner
	target.repoName = repoName
	return upsertDefaultsVariable(target, name, value)
}

func setBootstrapRepoSecret(ctx context.Context, repo, name, value string) error {
	owner, repoName, err := repoutil.SplitRepoSlug(repo)
	if err != nil {
		return err
	}
	client, err := api.NewRESTClient(secretSetClientOptions(""))
	if err != nil {
		return err
	}
	return setRepoSecret(client, owner, repoName, name, value)
}

func parseBootstrapNames(output []byte) []string {
	trimmed := strings.TrimSpace(string(output))
	if trimmed == "" {
		return nil
	}
	lines := strings.Split(trimmed, "\n")
	result := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			result = append(result, line)
		}
	}
	sort.Strings(result)
	return result
}

func resolveBootstrapTextValue(envName, title, description, defaultValue string, allowed []string, optional bool) (string, bool, error) {
	if envValue := strings.TrimSpace(os.Getenv(envName)); envValue != "" {
		if err := validateBootstrapEnumValue(envValue, allowed, optional); err != nil {
			return "", false, err
		}
		return envValue, true, nil
	}
	if !tty.IsStderrTerminal() {
		if defaultValue != "" {
			if err := validateBootstrapEnumValue(defaultValue, allowed, optional); err != nil {
				return "", false, err
			}
			return defaultValue, true, nil
		}
		if optional {
			return "", false, nil
		}
		return "", false, fmt.Errorf("%s is required; set environment variable %s or rerun interactively. Example: export %s='example-value'", title, envName, envName)
	}

	var value string
	input := huh.NewInput().Title(title).Description(description).Value(&value)
	if defaultValue != "" {
		input = input.Placeholder(defaultValue)
	}
	input = input.Validate(func(v string) error {
		trimmed := strings.TrimSpace(v)
		if trimmed == "" && defaultValue != "" {
			trimmed = defaultValue
		}
		if trimmed == "" && optional {
			return nil
		}
		if trimmed == "" {
			return errors.New("value cannot be empty. Example: enter a non-empty value such as example-value")
		}
		return validateBootstrapEnumValue(trimmed, allowed, optional)
	})
	if err := console.NewInputForm(input).Run(); err != nil {
		return "", false, err
	}
	value = strings.TrimSpace(value)
	if value == "" {
		value = defaultValue
	}
	if value == "" && optional {
		return "", false, nil
	}
	return value, true, nil
}

func resolveBootstrapSecretValue(envName, title, description string, optional bool) (string, bool, error) {
	if envValue := strings.TrimRight(os.Getenv(envName), "\r\n"); envValue != "" {
		return envValue, true, nil
	}
	if !tty.IsStderrTerminal() {
		if optional {
			return "", false, nil
		}
		return "", false, fmt.Errorf("%s is required; set environment variable %s or rerun interactively. Example: export %s='example-secret'", title, envName, envName)
	}
	value, err := console.PromptSecretInput(title, description)
	if err != nil {
		return "", false, err
	}
	value = strings.TrimRight(value, "\r\n")
	if value == "" && optional {
		return "", false, nil
	}
	return value, true, nil
}

func validateBootstrapEnumValue(value string, allowed []string, optional bool) error {
	if value == "" && optional {
		return nil
	}
	if len(allowed) == 0 {
		return nil
	}
	if slices.Contains(allowed, value) {
		return nil
	}
	return fmt.Errorf("value must be one of: %s. Example: %s", strings.Join(allowed, ", "), allowed[0])
}

func profileSourcesUseActionsTokenCopilotAuth(ctx context.Context, sources []string) (bool, error) {
	if len(sources) == 0 {
		return false, nil
	}
	resolved, err := ResolveWorkflows(ctx, sources, false)
	if err != nil {
		return false, err
	}
	hasCopilot := false
	for _, candidate := range resolved.Workflows {
		if candidate == nil || candidate.IsActionWorkflow || candidate.IsPackageSkillFile || candidate.IsPackageAgentFile {
			continue
		}
		engine := strings.TrimSpace(candidate.Engine)
		if engine != "" && engine != "copilot" {
			continue
		}
		hasCopilot = true
		if !workflowGrantsCopilotRequestsWrite(candidate.Content) {
			return false, nil
		}
	}
	return hasCopilot, nil
}

func deriveBootstrapAppName(repo, explicitName string) string {
	candidate := strings.TrimSpace(explicitName)
	if candidate == "" {
		candidate = repo
	}
	candidate = strings.ReplaceAll(candidate, "/", "-")
	clean := strings.Builder{}
	previousDash := false
	for _, ch := range candidate {
		allowed := ch >= 'a' && ch <= 'z' || ch >= 'A' && ch <= 'Z' || ch >= '0' && ch <= '9'
		if allowed {
			clean.WriteRune(ch)
			previousDash = false
			continue
		}
		if !previousDash {
			clean.WriteRune('-')
			previousDash = true
		}
	}
	result := strings.Trim(clean.String(), "-")
	if len(result) <= 34 {
		return result
	}
	suffix := strings.TrimLeft(result[len(result)-15:], "-")
	prefixLength := max(1, 34-len(suffix)-1)
	prefix := strings.TrimRight(result[:prefixLength], "-")
	return strings.Trim(prefix+"-"+suffix, "-")
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

func bootstrapRandomHex(size int) (string, error) {
	buf := make([]byte, size)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func htmlEscape(value string) string {
	replacer := strings.NewReplacer("&", "&amp;", "\"", "&quot;", "<", "&lt;", ">", "&gt;")
	return replacer.Replace(value)
}

func openBootstrapBrowser(url string) bool {
	commands := [][]string{{"gh", "browse", url}}
	switch runtime.GOOS {
	case "darwin":
		commands = append([][]string{{"open", url}}, commands...)
	case "windows":
		commands = append([][]string{{"cmd", "/c", "start", "", url}}, commands...)
	default:
		commands = append([][]string{{"xdg-open", url}}, commands...)
	}
	for _, args := range commands {
		cmd := exec.Command(args[0], args[1:]...)
		if err := cmd.Start(); err == nil {
			return true
		}
	}
	return false
}

func netListener() (net.Listener, error) {
	return net.Listen("tcp", "127.0.0.1:0")
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func bootstrapRepositoryVariableEnvName(name string) string {
	return bootstrapInputEnvName("VAR", name)
}

func bootstrapRepositorySecretEnvName(name string) string {
	return bootstrapInputEnvName("SECRET", name)
}

func bootstrapInputEnvName(kind, name string) string {
	suffix := strings.ToUpper(strings.TrimSpace(name))
	if suffix == "" {
		suffix = "VALUE"
	}
	var builder strings.Builder
	lastUnderscore := false
	for _, ch := range suffix {
		switch {
		case ch >= 'A' && ch <= 'Z', ch >= '0' && ch <= '9':
			builder.WriteRune(ch)
			lastUnderscore = false
		default:
			if !lastUnderscore {
				builder.WriteByte('_')
				lastUnderscore = true
			}
		}
	}
	suffix = strings.Trim(builder.String(), "_")
	if suffix == "" {
		suffix = "VALUE"
	}
	return "GH_AW_BOOTSTRAP_" + kind + "_" + suffix
}

func workflowGrantsCopilotRequestsWrite(content []byte) bool {
	frontmatter, err := parser.ExtractFrontmatterFromContent(string(content))
	if err != nil || frontmatter == nil {
		return false
	}
	permissions, ok := frontmatter.Frontmatter["permissions"].(map[string]any)
	if !ok {
		return false
	}
	level, ok := permissions[string(workflow.PermissionCopilotRequests)].(string)
	return ok && strings.TrimSpace(level) == "write"
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
