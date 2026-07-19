package workflow

import (
	"encoding/json"
	"fmt"
	"strconv"
)

// ParseFrontmatterConfig creates a FrontmatterConfig from a raw frontmatter map
// This provides a single entry point for converting untyped frontmatter into
// a structured configuration with better error handling.
func ParseFrontmatterConfig(frontmatter map[string]any) (*FrontmatterConfig, error) {
	frontmatterTypesLog.Printf("Parsing frontmatter config with %d fields", len(frontmatter))
	var config FrontmatterConfig

	jsonBytes, err := json.Marshal(frontmatter)
	if err != nil {
		frontmatterTypesLog.Printf("Failed to marshal frontmatter: %v", err)
		return nil, fmt.Errorf("failed to marshal frontmatter to JSON: %w", err)
	}

	if err := json.Unmarshal(jsonBytes, &config); err != nil {
		frontmatterTypesLog.Printf("Failed to unmarshal frontmatter: %v", err)
		return nil, fmt.Errorf("failed to unmarshal frontmatter into config: %w", err)
	}

	if err := validateFrontmatterRunsOnFields(frontmatter, &config); err != nil {
		return nil, err
	}
	populateTypedFrontmatterConfig(frontmatter, &config)

	frontmatterTypesLog.Printf("Successfully parsed frontmatter config: name=%s, engine=%v", config.Name, config.Engine)
	return &config, nil
}

func validateFrontmatterRunsOnFields(frontmatter map[string]any, config *FrontmatterConfig) error {
	if err := validateRunsOnValue(config.RunsOn); err != nil {
		return err
	}
	if err := validateRunsOnValue(config.RunsOnSlim); err != nil {
		return err
	}
	if safeOutputsRaw, ok := frontmatter["safe-outputs"].(map[string]any); ok {
		if err := validateRunsOnValue(safeOutputsRaw["runs-on"]); err != nil {
			return err
		}
		if threatRaw, ok := safeOutputsRaw["threat-detection"].(map[string]any); ok {
			return validateRunsOnValue(threatRaw["runs-on"])
		}
	}
	return nil
}

func populateTypedFrontmatterConfig(frontmatter map[string]any, config *FrontmatterConfig) {
	populateTypedRuntimes(config)
	populateTypedPermissions(config)
	populateCheckoutConfigs(config)
	populateOnNeedsConfig(config)

	config.ExperimentConfigs = extractExperimentConfigsFromFrontmatter(frontmatter)
	config.ModelPolicyAllowed, config.ModelPolicyBlocked = extractModelPolicyFromFrontmatter(frontmatter)
	if rawSkills, ok := frontmatter["skills"].([]any); ok {
		config.SkillReferences = parseRawSkillReferences(rawSkills)
	}
}

func populateTypedRuntimes(config *FrontmatterConfig) {
	if len(config.Runtimes) == 0 {
		return
	}
	runtimesTyped, err := parseRuntimesConfig(config.Runtimes)
	if err == nil {
		config.RuntimesTyped = runtimesTyped
		frontmatterTypesLog.Printf("Parsed typed runtimes config with %d runtimes", countRuntimes(runtimesTyped))
	}
}

func populateTypedPermissions(config *FrontmatterConfig) {
	if len(config.Permissions) == 0 {
		return
	}
	permissionsTyped, err := parsePermissionsConfig(config.Permissions)
	if err == nil {
		config.PermissionsTyped = permissionsTyped
		frontmatterTypesLog.Print("Parsed typed permissions config")
	}
}

func populateCheckoutConfigs(config *FrontmatterConfig) {
	if config.Checkout == nil {
		return
	}
	if checkoutValue, ok := config.Checkout.(bool); ok && !checkoutValue {
		config.CheckoutDisabled = true
		frontmatterTypesLog.Print("Checkout disabled via checkout: false")
		return
	}
	checkoutConfigs, err := ParseCheckoutConfigs(config.Checkout)
	if err == nil {
		config.CheckoutConfigs = checkoutConfigs
		frontmatterTypesLog.Printf("Parsed checkout config: %d entries", len(checkoutConfigs))
	}
}

func populateOnNeedsConfig(config *FrontmatterConfig) {
	if len(config.On) == 0 {
		return
	}
	onNeeds, err := parseOnNeedsConfig(config.On)
	if err == nil {
		config.OnNeeds = onNeeds
		frontmatterTypesLog.Printf("Parsed typed on.needs config with %d entries", len(onNeeds))
	}
}

func extractModelPolicyFromFrontmatter(frontmatter map[string]any) ([]string, []string) {
	modelsRaw, ok := frontmatter["models"].(map[string]any)
	if !ok {
		return nil, nil
	}
	return parseModelPolicyList(modelsRaw["allowed"]), parseModelPolicyList(modelsRaw["blocked"])
}

func parseModelPolicyList(value any) []string {
	values, ok := value.([]any)
	if !ok {
		if value != nil {
			frontmatterTypesLog.Printf("Skipping model policy list: expected array, got %T", value)
		}
		return nil
	}
	result := make([]string, 0, len(values))
	for _, v := range values {
		s, ok := v.(string)
		if !ok {
			frontmatterTypesLog.Printf("Skipping model policy entry: expected string, got %T", v)
			continue
		}
		if s == "" {
			frontmatterTypesLog.Printf("Skipping model policy entry: empty string")
			continue
		}
		result = append(result, s)
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

func parseOnNeedsConfig(on map[string]any) ([]string, error) {
	return parseOnNeedsValues(on)
}

// parseRuntimesConfig converts a map[string]any to RuntimesConfig
func parseRuntimesConfig(runtimes map[string]any) (*RuntimesConfig, error) {
	config := &RuntimesConfig{}

	for runtimeID, configAny := range runtimes {
		configMap, ok := configAny.(map[string]any)
		if !ok {
			frontmatterTypesLog.Printf("Skipping runtime '%s': expected map, got %T", runtimeID, configAny)
			continue
		}

		runtimeConfig, ok := parseRuntimeConfigMap(configMap)
		if !ok {
			continue
		}
		assignRuntimeConfig(config, runtimeID, runtimeConfig)
	}

	return config, nil
}

func parseRuntimeConfigMap(configMap map[string]any) (*RuntimeConfig, bool) {
	var version string
	if versionAny, hasVersion := configMap["version"]; hasVersion {
		var ok bool
		version, ok = parseRuntimeVersion(versionAny)
		if !ok {
			return nil, false
		}
	}

	ifCondition, _ := configMap["if"].(string)
	actionRepo, _ := configMap["action-repo"].(string)
	actionVersion, _ := configMap["action-version"].(string)
	runInstallScripts := parseOptionalBool(configMap["run-install-scripts"])
	cooldown := parseOptionalBool(configMap["cooldown"])

	return &RuntimeConfig{
		Version:           version,
		If:                ifCondition,
		ActionRepo:        actionRepo,
		ActionVersion:     actionVersion,
		Cooldown:          cooldown,
		RunInstallScripts: runInstallScripts,
	}, true
}

func parseRuntimeVersion(versionAny any) (string, bool) {
	switch v := versionAny.(type) {
	case string:
		return v, true
	case int:
		return strconv.Itoa(v), true
	case float64:
		if v == float64(int(v)) {
			return strconv.Itoa(int(v)), true
		}
		return fmt.Sprintf("%g", v), true
	default:
		return "", false
	}
}

func parseOptionalBool(value any) *bool {
	if boolValue, ok := value.(bool); ok {
		return &boolValue
	}
	return nil
}

func assignRuntimeConfig(config *RuntimesConfig, runtimeID string, runtimeConfig *RuntimeConfig) {
	switch runtimeID {
	case "node":
		config.Node = runtimeConfig
	case "python":
		config.Python = runtimeConfig
	case "go":
		config.Go = runtimeConfig
	case "uv":
		config.UV = runtimeConfig
	case "bun":
		config.Bun = runtimeConfig
	case "deno":
		config.Deno = runtimeConfig
	case "dotnet":
		config.Dotnet = runtimeConfig
	case "elixir":
		config.Elixir = runtimeConfig
	case "gh-aw":
		config.GhAw = runtimeConfig
	case "haskell":
		config.Haskell = runtimeConfig
	case "java":
		config.Java = runtimeConfig
	case "ruby":
		config.Ruby = runtimeConfig
	}
}

// parsePermissionsConfig converts a map[string]any to PermissionsConfig
func parsePermissionsConfig(permissions map[string]any) (*PermissionsConfig, error) {
	config := &PermissionsConfig{}

	if shorthand, ok := parsePermissionShorthand(permissions); ok {
		config.Shorthand = shorthand
		return config, nil
	}

	for scope, level := range permissions {
		if levelStr, ok := level.(string); ok {
			if setter, exists := permissionScopeSetters[scope]; exists {
				setter(config, levelStr)
			}
		}
	}

	return config, nil
}

func parsePermissionShorthand(permissions map[string]any) (string, bool) {
	if len(permissions) != 1 {
		return "", false
	}
	for key, value := range permissions {
		strValue, ok := value.(string)
		if !ok {
			return "", false
		}
		for _, shorthand := range []string{"read-all", "write-all", "read", "write", "none"} {
			if key == shorthand || strValue == shorthand {
				return shorthand, true
			}
		}
	}
	return "", false
}

var permissionScopeSetters = map[string]func(*PermissionsConfig, string){
	"actions":                              func(c *PermissionsConfig, v string) { c.Actions = v },
	"checks":                               func(c *PermissionsConfig, v string) { c.Checks = v },
	"contents":                             func(c *PermissionsConfig, v string) { c.Contents = v },
	"deployments":                          func(c *PermissionsConfig, v string) { c.Deployments = v },
	"id-token":                             func(c *PermissionsConfig, v string) { c.IDToken = v },
	"issues":                               func(c *PermissionsConfig, v string) { c.Issues = v },
	"discussions":                          func(c *PermissionsConfig, v string) { c.Discussions = v },
	"packages":                             func(c *PermissionsConfig, v string) { c.Packages = v },
	"pages":                                func(c *PermissionsConfig, v string) { c.Pages = v },
	"pull-requests":                        func(c *PermissionsConfig, v string) { c.PullRequests = v },
	"repository-projects":                  func(c *PermissionsConfig, v string) { c.RepositoryProjects = v },
	"security-events":                      func(c *PermissionsConfig, v string) { c.SecurityEvents = v },
	"statuses":                             func(c *PermissionsConfig, v string) { c.Statuses = v },
	"vulnerability-alerts":                 func(c *PermissionsConfig, v string) { c.VulnerabilityAlerts = v },
	"organization-projects":                func(c *PermissionsConfig, v string) { c.OrganizationProjects = v },
	"administration":                       func(c *PermissionsConfig, v string) { c.Administration = v },
	"environments":                         func(c *PermissionsConfig, v string) { c.Environments = v },
	"git-signing":                          func(c *PermissionsConfig, v string) { c.GitSigning = v },
	"workflows":                            func(c *PermissionsConfig, v string) { c.Workflows = v },
	"repository-hooks":                     func(c *PermissionsConfig, v string) { c.RepositoryHooks = v },
	"single-file":                          func(c *PermissionsConfig, v string) { c.SingleFile = v },
	"codespaces":                           func(c *PermissionsConfig, v string) { c.Codespaces = v },
	"repository-custom-properties":         func(c *PermissionsConfig, v string) { c.RepositoryCustomProperties = v },
	"members":                              func(c *PermissionsConfig, v string) { c.Members = v },
	"organization-administration":          func(c *PermissionsConfig, v string) { c.OrganizationAdministration = v },
	"team-discussions":                     func(c *PermissionsConfig, v string) { c.TeamDiscussions = v },
	"organization-hooks":                   func(c *PermissionsConfig, v string) { c.OrganizationHooks = v },
	"organization-members":                 func(c *PermissionsConfig, v string) { c.OrganizationMembers = v },
	"organization-packages":                func(c *PermissionsConfig, v string) { c.OrganizationPackages = v },
	"organization-self-hosted-runners":     func(c *PermissionsConfig, v string) { c.OrganizationSelfHostedRunners = v },
	"organization-custom-org-roles":        func(c *PermissionsConfig, v string) { c.OrganizationCustomOrgRoles = v },
	"organization-custom-properties":       func(c *PermissionsConfig, v string) { c.OrganizationCustomProperties = v },
	"organization-custom-repository-roles": func(c *PermissionsConfig, v string) { c.OrganizationCustomRepositoryRoles = v },
	"organization-announcement-banners":    func(c *PermissionsConfig, v string) { c.OrganizationAnnouncementBanners = v },
	"organization-events":                  func(c *PermissionsConfig, v string) { c.OrganizationEvents = v },
	"organization-plan":                    func(c *PermissionsConfig, v string) { c.OrganizationPlan = v },
	"organization-user-blocking":           func(c *PermissionsConfig, v string) { c.OrganizationUserBlocking = v },
	"organization-personal-access-token-requests": func(c *PermissionsConfig, v string) { c.OrganizationPersonalAccessTokenReqs = v },
	"organization-personal-access-tokens":         func(c *PermissionsConfig, v string) { c.OrganizationPersonalAccessTokens = v },
	"organization-copilot":                        func(c *PermissionsConfig, v string) { c.OrganizationCopilot = v },
	"organization-codespaces":                     func(c *PermissionsConfig, v string) { c.OrganizationCodespaces = v },
	"email-addresses":                             func(c *PermissionsConfig, v string) { c.EmailAddresses = v },
	"codespaces-lifecycle-admin":                  func(c *PermissionsConfig, v string) { c.CodespacesLifecycleAdmin = v },
	"codespaces-metadata":                         func(c *PermissionsConfig, v string) { c.CodespacesMetadata = v },
}
