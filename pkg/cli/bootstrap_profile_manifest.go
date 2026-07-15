package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const bootstrapActionTypeExample = "require-owner-type, repo-variable, repo-secret, github-app, copilot-auth, or handoff"

type repositoryPackageBootstrap struct {
	Actions []repositoryPackageBootstrapAction
}

type repositoryPackageBootstrapAction struct {
	Type             string
	Owner            string
	Value            string
	Name             string
	Prompt           string
	Description      string
	Default          string
	Optional         bool
	Enum             []string
	When             *repositoryPackageBootstrapCondition
	Secret           string
	Strategy         string
	Message          string
	Mode             string
	AppIDVariable    string
	PrivateKeySecret string
	AppName          string
	HomepageURL      string
	Permissions      map[string]string
	Events           []string
	ExistingOnly     bool
}

type repositoryPackageBootstrapCondition struct {
	Variable string
	Equals   string
}

type resolvedBootstrapProfile struct {
	PackageID string
	Source    string
	Profile   *repositoryPackageBootstrap
}

func resolveBootstrapProfileFromSources(ctx context.Context, sources []string) (*resolvedBootstrapProfile, error) {
	profiles := make([]*resolvedBootstrapProfile, 0, 1)
	seenPackageIDs := make(map[string]struct{})

	for _, source := range sources {
		if localProfile, err := resolveLocalBootstrapProfileFromSource(source); err != nil {
			return nil, err
		} else if localProfile != nil {
			if _, exists := seenPackageIDs[localProfile.PackageID]; exists {
				continue
			}
			seenPackageIDs[localProfile.PackageID] = struct{}{}
			profiles = append(profiles, localProfile)
			continue
		}

		repoSpec, ok, repoErr := parseRepositoryPackageSpec(source)
		if !ok {
			continue
		}
		if repoErr != nil {
			return nil, repoErr
		}

		pkg, err := resolveRepositoryPackage(ctx, repoSpec, explicitHostForRepo(repoSpec.RepoSlug))
		if err != nil {
			if repoSpec.PackagePath == "" || !isRepositoryPackageManifestNotFound(err) {
				return nil, err
			}
			continue
		}
		if pkg.Bootstrap == nil {
			continue
		}

		packageID := repositoryPackageIdentifier(repoSpec.RepoSlug, repoSpec.PackagePath)
		if _, exists := seenPackageIDs[packageID]; exists {
			continue
		}
		seenPackageIDs[packageID] = struct{}{}

		profiles = append(profiles, &resolvedBootstrapProfile{
			PackageID: packageID,
			Source:    source,
			Profile:   pkg.Bootstrap,
		})
	}

	if len(profiles) == 0 {
		return nil, nil
	}
	if len(profiles) == 1 {
		return profiles[0], nil
	}

	packageIDs := make([]string, 0, len(profiles))
	for _, profile := range profiles {
		packageIDs = append(packageIDs, profile.PackageID)
	}
	sort.Strings(packageIDs)

	return nil, fmt.Errorf("multiple bootstrap profiles matched the selected sources: %s; select a single package profile or split the bootstrap into separate runs", strings.Join(packageIDs, ", "))
}

func resolveLocalBootstrapProfileFromSource(source string) (*resolvedBootstrapProfile, error) {
	if !isLocalWorkflowPath(source) {
		return nil, nil
	}

	resolvedPath, err := filepath.Abs(source)
	if err != nil {
		return nil, fmt.Errorf("resolve local bootstrap profile %q: %w", source, err)
	}

	manifestPath, packageID, err := localBootstrapManifestPath(resolvedPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	if manifestPath == "" {
		return nil, nil
	}

	content, err := os.ReadFile(manifestPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("read local bootstrap manifest %q: %w", manifestPath, err)
	}

	manifest, _, err := parseRepositoryPackageManifest(manifestPath, content)
	if err != nil {
		return nil, err
	}
	if manifest.Bootstrap == nil {
		return nil, nil
	}

	return &resolvedBootstrapProfile{
		PackageID: packageID,
		Source:    source,
		Profile:   manifest.Bootstrap,
	}, nil
}

func localBootstrapManifestPath(resolvedPath string) (string, string, error) {
	info, err := os.Stat(resolvedPath)
	if err != nil {
		return "", "", err
	}

	if info.IsDir() {
		manifestPath := filepath.Join(resolvedPath, repositoryPackageManifestFileName)
		if _, err := os.Stat(manifestPath); err != nil {
			return "", "", err
		}
		return manifestPath, filepath.Clean(resolvedPath), nil
	}

	if filepath.Base(resolvedPath) != repositoryPackageManifestFileName {
		return "", "", nil
	}

	return resolvedPath, filepath.Clean(filepath.Dir(resolvedPath)), nil
}

func extractManifestBootstrap(value any, manifestPath string) (*repositoryPackageBootstrap, error) {
	root, ok := value.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("invalid Agentic Workflow manifest %q: bootstrap must be a mapping. Example: bootstrap: { actions: [{ type: repo-variable, name: EXAMPLE, prompt: Enter a value }] }", manifestPath)
	}

	actionsValue, ok := root["actions"]
	if !ok {
		return nil, fmt.Errorf("invalid Agentic Workflow manifest %q: bootstrap.actions is required. Example: bootstrap: { actions: [{ type: repo-variable, name: EXAMPLE, prompt: Enter a value }] }", manifestPath)
	}

	actionItems, ok := actionsValue.([]any)
	if !ok {
		return nil, fmt.Errorf("invalid Agentic Workflow manifest %q: bootstrap.actions must be a list. Example: bootstrap: { actions: [{ type: repo-variable, name: EXAMPLE, prompt: Enter a value }] }", manifestPath)
	}
	if len(actionItems) == 0 {
		return nil, fmt.Errorf("invalid Agentic Workflow manifest %q: bootstrap.actions must not be empty. Example: bootstrap: { actions: [{ type: repo-variable, name: EXAMPLE, prompt: Enter a value }] }", manifestPath)
	}

	bootstrap := &repositoryPackageBootstrap{}
	for index, item := range actionItems {
		actionMap, ok := item.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("invalid Agentic Workflow manifest %q: bootstrap.actions[%d] must be a mapping. Example: { type: repo-variable, name: EXAMPLE, prompt: Enter a value }", manifestPath, index)
		}

		actionType, ok := stringValue(actionMap["type"])
		if !ok || strings.TrimSpace(actionType) == "" {
			return nil, fmt.Errorf("invalid Agentic Workflow manifest %q: bootstrap.actions[%d].type must be a non-empty string. Example: type: repo-variable", manifestPath, index)
		}

		action, err := parseManifestBootstrapAction(strings.TrimSpace(actionType), actionMap, manifestPath, index)
		if err != nil {
			return nil, err
		}
		bootstrap.Actions = append(bootstrap.Actions, action)
	}

	return bootstrap, nil
}

func parseManifestBootstrapAction(actionType string, actionMap map[string]any, manifestPath string, index int) (repositoryPackageBootstrapAction, error) {
	action := repositoryPackageBootstrapAction{Type: actionType}

	if owner, ok := stringValue(actionMap["owner"]); ok {
		action.Owner = strings.TrimSpace(owner)
	}
	if value, ok := stringValue(actionMap["value"]); ok {
		action.Value = strings.TrimSpace(value)
	}
	if name, ok := stringValue(actionMap["name"]); ok {
		action.Name = strings.TrimSpace(name)
	}
	if prompt, ok := stringValue(actionMap["prompt"]); ok {
		action.Prompt = strings.TrimSpace(prompt)
	}
	if description, ok := stringValue(actionMap["description"]); ok {
		action.Description = strings.TrimSpace(description)
	}
	if defaultValue, ok := stringValue(actionMap["default"]); ok {
		action.Default = defaultValue
	}
	if secret, ok := stringValue(actionMap["secret"]); ok {
		action.Secret = strings.TrimSpace(secret)
	}
	if strategy, ok := stringValue(actionMap["strategy"]); ok {
		action.Strategy = strings.TrimSpace(strategy)
	}
	if message, ok := stringValue(actionMap["message"]); ok {
		action.Message = strings.TrimSpace(message)
	}
	if mode, ok := stringValue(actionMap["mode"]); ok {
		action.Mode = strings.TrimSpace(mode)
	}
	if appIDVariable, ok := stringValue(actionMap["app-id-variable"]); ok {
		action.AppIDVariable = strings.TrimSpace(appIDVariable)
	}
	if privateKeySecret, ok := stringValue(actionMap["private-key-secret"]); ok {
		action.PrivateKeySecret = strings.TrimSpace(privateKeySecret)
	}
	if appName, ok := stringValue(actionMap["app-name"]); ok {
		action.AppName = strings.TrimSpace(appName)
	}
	if homepageURL, ok := stringValue(actionMap["homepage-url"]); ok {
		action.HomepageURL = strings.TrimSpace(homepageURL)
	}
	if optional, ok := actionMap["optional"].(bool); ok {
		action.Optional = optional
	}
	if existingOnly, ok := actionMap["existing-only"].(bool); ok {
		action.ExistingOnly = existingOnly
	}
	if enumValues, ok, err := stringListValue(actionMap["enum"]); err != nil {
		return repositoryPackageBootstrapAction{}, manifestBootstrapFieldError(manifestPath, index, "enum", err)
	} else if ok {
		action.Enum = enumValues
	}
	if events, ok, err := stringListValue(actionMap["events"]); err != nil {
		return repositoryPackageBootstrapAction{}, manifestBootstrapFieldError(manifestPath, index, "events", err)
	} else if ok {
		action.Events = events
	}
	if _, exists := actionMap["when"]; exists {
		return repositoryPackageBootstrapAction{}, fmt.Errorf("invalid Agentic Workflow manifest %q: bootstrap.actions[%d].when is not supported yet. Example: remove the when field and keep only supported keys such as type, name, and prompt", manifestPath, index)
	}
	if permissionsValue, exists := actionMap["permissions"]; exists {
		permissions, err := stringMapValue(permissionsValue)
		if err != nil {
			return repositoryPackageBootstrapAction{}, manifestBootstrapFieldError(manifestPath, index, "permissions", err)
		}
		action.Permissions = permissions
	}

	switch actionType {
	case "require-owner-type":
		if action.Owner != "" && action.Owner != "repo" {
			return repositoryPackageBootstrapAction{}, fmt.Errorf("invalid Agentic Workflow manifest %q: bootstrap.actions[%d].owner must be 'repo' when type=require-owner-type. Example: { type: require-owner-type, owner: repo, value: org }", manifestPath, index)
		}
		if action.Value != "any" && action.Value != "org" && action.Value != "user" {
			return repositoryPackageBootstrapAction{}, fmt.Errorf("invalid Agentic Workflow manifest %q: bootstrap.actions[%d].value must be one of: any, org, user. Example: { type: require-owner-type, value: org }", manifestPath, index)
		}
	case "repo-variable":
		if action.Name == "" {
			return repositoryPackageBootstrapAction{}, fmt.Errorf("invalid Agentic Workflow manifest %q: bootstrap.actions[%d].name is required when type=repo-variable. Example: { type: repo-variable, name: EXAMPLE, prompt: Enter a value }", manifestPath, index)
		}
		if action.Prompt == "" {
			return repositoryPackageBootstrapAction{}, fmt.Errorf("invalid Agentic Workflow manifest %q: bootstrap.actions[%d].prompt is required when type=repo-variable. Example: { type: repo-variable, name: EXAMPLE, prompt: Enter a value }", manifestPath, index)
		}
	case "repo-secret":
		if action.Name == "" {
			return repositoryPackageBootstrapAction{}, fmt.Errorf("invalid Agentic Workflow manifest %q: bootstrap.actions[%d].name is required when type=repo-secret. Example: { type: repo-secret, name: EXAMPLE_SECRET, prompt: Enter a secret }", manifestPath, index)
		}
		if action.Prompt == "" {
			return repositoryPackageBootstrapAction{}, fmt.Errorf("invalid Agentic Workflow manifest %q: bootstrap.actions[%d].prompt is required when type=repo-secret. Example: { type: repo-secret, name: EXAMPLE_SECRET, prompt: Enter a secret }", manifestPath, index)
		}
	case "github-app":
		if action.AppName == "" && action.Name != "" {
			action.AppName = action.Name
		}
		if action.ExistingOnly && action.Mode != "" && action.Mode != "existing" {
			return repositoryPackageBootstrapAction{}, fmt.Errorf("invalid Agentic Workflow manifest %q: bootstrap.actions[%d].existing-only requires mode to be 'existing' or unset. Remove mode=%q or set it to 'existing'", manifestPath, index, action.Mode)
		}
		if action.ExistingOnly && action.Mode == "" {
			action.Mode = "existing"
		}
		if action.Mode == "" {
			action.Mode = "create-or-existing"
		}
		if action.Mode != "create-or-existing" && action.Mode != "existing" {
			return repositoryPackageBootstrapAction{}, fmt.Errorf("invalid Agentic Workflow manifest %q: bootstrap.actions[%d].mode must be one of: create-or-existing, existing. Example: { type: github-app, mode: existing, app-id-variable: APP_ID, private-key-secret: APP_PRIVATE_KEY }", manifestPath, index)
		}
		if action.Owner != "" && action.Owner != "repo" {
			return repositoryPackageBootstrapAction{}, fmt.Errorf("invalid Agentic Workflow manifest %q: bootstrap.actions[%d].owner must be 'repo' when type=github-app. Example: { type: github-app, owner: repo, app-id-variable: APP_ID, private-key-secret: APP_PRIVATE_KEY }", manifestPath, index)
		}
		if action.AppIDVariable == "" {
			return repositoryPackageBootstrapAction{}, fmt.Errorf("invalid Agentic Workflow manifest %q: bootstrap.actions[%d].app-id-variable is required when type=github-app. Example: { type: github-app, app-id-variable: APP_ID, private-key-secret: APP_PRIVATE_KEY }", manifestPath, index)
		}
		if action.PrivateKeySecret == "" {
			return repositoryPackageBootstrapAction{}, fmt.Errorf("invalid Agentic Workflow manifest %q: bootstrap.actions[%d].private-key-secret is required when type=github-app. Example: { type: github-app, app-id-variable: APP_ID, private-key-secret: APP_PRIVATE_KEY }", manifestPath, index)
		}
	case "copilot-auth":
		if action.Secret == "" {
			action.Secret = "COPILOT_GITHUB_TOKEN"
		}
		if action.Strategy == "" {
			action.Strategy = "prompt-if-actions-auth-unavailable"
		}
		if action.Strategy != "prompt-if-actions-auth-unavailable" {
			return repositoryPackageBootstrapAction{}, fmt.Errorf("invalid Agentic Workflow manifest %q: bootstrap.actions[%d].strategy must be 'prompt-if-actions-auth-unavailable'. Example: { type: copilot-auth, strategy: prompt-if-actions-auth-unavailable }", manifestPath, index)
		}
	case "handoff":
		if action.Message == "" {
			return repositoryPackageBootstrapAction{}, fmt.Errorf("invalid Agentic Workflow manifest %q: bootstrap.actions[%d].message is required when type=handoff. Example: { type: handoff, message: Continue with repository-specific setup. }", manifestPath, index)
		}
	default:
		return repositoryPackageBootstrapAction{}, fmt.Errorf("invalid Agentic Workflow manifest %q: bootstrap.actions[%d].type %q is not supported. Example: use one of %s", manifestPath, index, actionType, bootstrapActionTypeExample)
	}

	return action, nil
}

func stringListValue(value any) ([]string, bool, error) {
	if value == nil {
		return nil, false, nil
	}
	items, ok := value.([]any)
	if !ok {
		if direct, ok := value.([]string); ok {
			return direct, true, nil
		}
		return nil, false, errors.New("must be a list of strings. Example: [\"issues\", \"pull_request\"]")
	}
	result := make([]string, 0, len(items))
	for _, item := range items {
		stringItem, ok := stringValue(item)
		if !ok {
			return nil, false, errors.New("must be a list of strings. Example: [\"issues\", \"pull_request\"]")
		}
		result = append(result, stringItem)
	}
	return result, true, nil
}

func stringMapValue(value any) (map[string]string, error) {
	root, ok := value.(map[string]any)
	if !ok {
		return nil, errors.New("must be a string map. Example: { contents: \"read\", issues: \"write\" }")
	}
	result := make(map[string]string, len(root))
	for key, rawValue := range root {
		stringValue, ok := stringValue(rawValue)
		if !ok {
			return nil, errors.New("must be a string map. Example: { contents: \"read\", issues: \"write\" }")
		}
		result[key] = stringValue
	}
	return result, nil
}

func manifestBootstrapFieldError(manifestPath string, index int, field string, err error) error {
	if example, ok := manifestBootstrapFieldExample(field); ok {
		return fmt.Errorf("invalid Agentic Workflow manifest %q: bootstrap.actions[%d].%s %s. Example: bootstrap.actions[%d].%s: %s", manifestPath, index, field, err.Error(), index, field, example)
	}
	return fmt.Errorf("invalid Agentic Workflow manifest %q: bootstrap.actions[%d].%s %s", manifestPath, index, field, err.Error())
}

func manifestBootstrapFieldExample(field string) (string, bool) {
	switch field {
	case "enum", "events":
		return `["issues", "pull_request"]`, true
	case "permissions":
		return `{ contents: "read", issues: "write" }`, true
	default:
		return "", false
	}
}
