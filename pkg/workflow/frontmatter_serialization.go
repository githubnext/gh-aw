package workflow

import "reflect"

// countRuntimes counts the number of non-nil runtimes in RuntimesConfig
func countRuntimes(config *RuntimesConfig) int {
	if config == nil {
		return 0
	}
	count := 0
	if config.Node != nil {
		count++
	}
	if config.Python != nil {
		count++
	}
	if config.Go != nil {
		count++
	}
	if config.UV != nil {
		count++
	}
	if config.Bun != nil {
		count++
	}
	if config.Deno != nil {
		count++
	}
	if config.GhAw != nil {
		count++
	}
	return count
}

// ExtractMapField is a convenience wrapper for extracting map[string]any fields
// from frontmatter. This maintains backward compatibility with existing extraction
// patterns while preserving original types (avoiding JSON conversion which would
// convert all numbers to float64).
//
// Returns an empty map if the key doesn't exist (for backward compatibility).
func ExtractMapField(frontmatter map[string]any, key string) map[string]any {
	// Check if key exists and value is not nil
	value, exists := frontmatter[key]
	if !exists || value == nil {
		frontmatterTypesLog.Printf("Field '%s' not found in frontmatter, returning empty map", key)
		return make(map[string]any)
	}

	// Direct type assertion to preserve original types (especially integers)
	// This avoids JSON marshaling which would convert integers to float64
	if valueMap, ok := value.(map[string]any); ok {
		frontmatterTypesLog.Printf("Extracted map field '%s' with %d entries", key, len(valueMap))
		return valueMap
	}

	// For backward compatibility, return empty map if not a map
	frontmatterTypesLog.Printf("Field '%s' is not a map type, returning empty map", key)
	return make(map[string]any)
}

// ToMap converts FrontmatterConfig back to map[string]any for backward compatibility
// This allows gradual migration from map[string]any to strongly-typed config
func (fc *FrontmatterConfig) ToMap() map[string]any {
	frontmatterTypesLog.Printf("Converting FrontmatterConfig to map: name=%s", fc.Name)
	result := make(map[string]any)

	fc.addCoreFieldsToMap(result)
	fc.addConfigurationSectionsToMap(result)
	fc.addEventAndTriggerFieldsToMap(result)
	fc.addNetworkAndSandboxFieldsToMap(result)
	fc.addFeatureAndEnvironmentFieldsToMap(result)
	fc.addExecutionSettingsToMap(result)
	fc.addImportAndMetadataFieldsToMap(result)
	return result
}

func (fc *FrontmatterConfig) addCoreFieldsToMap(result map[string]any) {
	if fc.Name != "" {
		result["name"] = fc.Name
	}
	if fc.Description != "" {
		result["description"] = fc.Description
	}
	if fc.Engine != nil {
		result["engine"] = fc.Engine
	}
	if fc.Source != "" {
		result["source"] = fc.Source
	}
	if fc.Redirect != "" {
		result["redirect"] = fc.Redirect
	}
	if fc.TrackerID != "" {
		result["tracker-id"] = fc.TrackerID
	}
	if fc.Version != "" {
		result["version"] = fc.Version
	}
	if fc.TimeoutMinutes != nil {
		result["timeout-minutes"] = fc.TimeoutMinutes.ToValue()
	}
	if fc.Strict != nil {
		result["strict"] = *fc.Strict
	}
	if len(fc.Labels) > 0 {
		result["labels"] = fc.Labels
	}
}

func (fc *FrontmatterConfig) addConfigurationSectionsToMap(result map[string]any) {
	if fc.Tools != nil {
		result["tools"] = fc.Tools.ToMap()
	}
	if fc.MCPServers != nil {
		result["mcp-servers"] = fc.MCPServers
	}
	if fc.RuntimesTyped != nil {
		result["runtimes"] = runtimesConfigToMap(fc.RuntimesTyped)
	} else if fc.Runtimes != nil {
		result["runtimes"] = fc.Runtimes
	}
	if fc.Jobs != nil {
		result["jobs"] = fc.Jobs
	}
	if fc.SafeOutputs != nil {
		// Convert SafeOutputsConfig to map - would need a ToMap method
		result["safe-outputs"] = fc.SafeOutputs
	}
	if fc.MCPScripts != nil {
		// Convert MCPScriptsConfig to map - would need a ToMap method
		result["mcp-scripts"] = fc.MCPScripts
	}
}

func (fc *FrontmatterConfig) addEventAndTriggerFieldsToMap(result map[string]any) {
	if fc.On != nil {
		result["on"] = fc.On
	}
	if fc.PermissionsTyped != nil {
		result["permissions"] = permissionsConfigToMap(fc.PermissionsTyped)
	} else if fc.Permissions != nil {
		result["permissions"] = fc.Permissions
	}
	if fc.Concurrency != nil {
		result["concurrency"] = fc.Concurrency
	}
	if fc.If != "" {
		result["if"] = fc.If
	}
}

func (fc *FrontmatterConfig) addNetworkAndSandboxFieldsToMap(result map[string]any) {
	if fc.Network != nil {
		if networkValue, ok := networkPermissionsToMapValue(fc.Network); ok {
			result["network"] = networkValue
		}
	}
	if fc.Sandbox != nil {
		result["sandbox"] = fc.Sandbox
	}
}

func networkPermissionsToMapValue(network *NetworkPermissions) (any, bool) {
	if len(network.Allowed) == 1 && network.Allowed[0] == "defaults" &&
		!network.AllowedInput && network.Firewall == nil && len(network.Blocked) == 0 {
		return "defaults", true
	}
	networkMap := make(map[string]any)
	if len(network.Allowed) > 0 {
		networkMap["allowed"] = network.Allowed
	}
	if network.AllowedInput {
		networkMap["allowed-input"] = true
	}
	if len(network.Blocked) > 0 {
		networkMap["blocked"] = network.Blocked
	}
	if network.Firewall != nil {
		networkMap["firewall"] = network.Firewall
	}
	return networkMap, len(networkMap) > 0
}

func (fc *FrontmatterConfig) addFeatureAndEnvironmentFieldsToMap(result map[string]any) {
	if fc.Features != nil {
		result["features"] = fc.Features
	}
	if fc.Env != nil {
		result["env"] = fc.Env
	}
	if fc.Secrets != nil {
		result["secrets"] = fc.Secrets
	}
}

func (fc *FrontmatterConfig) addExecutionSettingsToMap(result map[string]any) {
	if !isNilValue(fc.RunsOn) {
		result["runs-on"] = fc.RunsOn
	}
	if !isEmptyRunsOnValue(fc.RunsOnSlim) {
		result["runs-on-slim"] = fc.RunsOnSlim
	}
	if fc.RunName != "" {
		result["run-name"] = fc.RunName
	}
	if fc.PreSteps != nil {
		result["pre-steps"] = fc.PreSteps
	}
	if fc.Steps != nil {
		result["steps"] = fc.Steps
	}
	if fc.PreAgentSteps != nil {
		result["pre-agent-steps"] = fc.PreAgentSteps
	}
	if fc.PostSteps != nil {
		result["post-steps"] = fc.PostSteps
	}
	if fc.Environment != nil {
		result["environment"] = fc.Environment
	}
	if fc.Container != nil {
		result["container"] = fc.Container
	}
	if fc.Services != nil {
		result["services"] = fc.Services
	}
	if fc.Cache != nil {
		result["cache"] = fc.Cache
	}
}

func (fc *FrontmatterConfig) addImportAndMetadataFieldsToMap(result map[string]any) {
	if fc.Imports != nil {
		result["imports"] = fc.Imports
	}
	if fc.Include != nil {
		result["include"] = fc.Include
	}

	// Metadata
	if fc.Metadata != nil {
		result["metadata"] = fc.Metadata
	}
	if fc.SecretMasking != nil {
		result["secret-masking"] = fc.SecretMasking
	}
}

func isNilValue(v any) bool {
	if v == nil {
		return true
	}

	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return rv.IsNil()
	default:
		return false
	}
}

// runtimeConfigToMap converts a single RuntimeConfig to map[string]any
func runtimeConfigToMap(rc *RuntimeConfig) map[string]any {
	m := map[string]any{}
	if rc.Version != "" {
		m["version"] = rc.Version
	}
	if rc.If != "" {
		m["if"] = rc.If
	}
	if rc.ActionRepo != "" {
		m["action-repo"] = rc.ActionRepo
	}
	if rc.ActionVersion != "" {
		m["action-version"] = rc.ActionVersion
	}
	if rc.Cooldown != nil {
		m["cooldown"] = *rc.Cooldown
	}
	if rc.RunInstallScripts != nil {
		m["run-install-scripts"] = *rc.RunInstallScripts
	}
	return m
}

// runtimesConfigToMap converts RuntimesConfig back to map[string]any
func runtimesConfigToMap(config *RuntimesConfig) map[string]any {
	if config == nil {
		return nil
	}
	frontmatterTypesLog.Printf("Converting RuntimesConfig to map: %d runtime(s) configured", countRuntimes(config))

	result := make(map[string]any)

	runtimes := []struct {
		key string
		rc  *RuntimeConfig
	}{
		{"node", config.Node},
		{"python", config.Python},
		{"go", config.Go},
		{"uv", config.UV},
		{"bun", config.Bun},
		{"deno", config.Deno},
		{"dotnet", config.Dotnet},
		{"elixir", config.Elixir},
		{"gh-aw", config.GhAw},
		{"haskell", config.Haskell},
		{"java", config.Java},
		{"ruby", config.Ruby},
	}
	for _, r := range runtimes {
		if r.rc != nil {
			if m := runtimeConfigToMap(r.rc); len(m) > 0 {
				result[r.key] = m
			}
		}
	}

	if len(result) == 0 {
		return nil
	}

	return result
}

// permissionsConfigToMap converts PermissionsConfig back to map[string]any
func permissionsConfigToMap(config *PermissionsConfig) map[string]any {
	if config == nil {
		return nil
	}

	// If shorthand is set, return it directly
	if config.Shorthand != "" {
		frontmatterTypesLog.Printf("Converting PermissionsConfig to map via shorthand: %s", config.Shorthand)
		return map[string]any{config.Shorthand: config.Shorthand}
	}

	frontmatterTypesLog.Print("Converting detailed PermissionsConfig to map")
	result := make(map[string]any)

	addPermissionScopeValues(result, permissionConfigScopeValues(config))

	if len(result) == 0 {
		return nil
	}

	return result
}

func addPermissionScopeValues(result map[string]any, values []struct {
	key   string
	value string
}) {
	for _, entry := range values {
		if entry.value != "" {
			result[entry.key] = entry.value
		}
	}
}

func permissionConfigScopeValues(config *PermissionsConfig) []struct {
	key   string
	value string
} {
	return []struct {
		key   string
		value string
	}{
		{"actions", config.Actions},
		{"checks", config.Checks},
		{"contents", config.Contents},
		{"deployments", config.Deployments},
		{"id-token", config.IDToken},
		{"issues", config.Issues},
		{"discussions", config.Discussions},
		{"packages", config.Packages},
		{"pages", config.Pages},
		{"pull-requests", config.PullRequests},
		{"repository-projects", config.RepositoryProjects},
		{"security-events", config.SecurityEvents},
		{"statuses", config.Statuses},
		{"vulnerability-alerts", config.VulnerabilityAlerts},
		{"organization-projects", config.OrganizationProjects},
		{"administration", config.Administration},
		{"environments", config.Environments},
		{"git-signing", config.GitSigning},
		{"workflows", config.Workflows},
		{"repository-hooks", config.RepositoryHooks},
		{"single-file", config.SingleFile},
		{"codespaces", config.Codespaces},
		{"repository-custom-properties", config.RepositoryCustomProperties},
		{"members", config.Members},
		{"organization-administration", config.OrganizationAdministration},
		{"team-discussions", config.TeamDiscussions},
		{"organization-hooks", config.OrganizationHooks},
		{"organization-members", config.OrganizationMembers},
		{"organization-packages", config.OrganizationPackages},
		{"organization-self-hosted-runners", config.OrganizationSelfHostedRunners},
		{"organization-custom-org-roles", config.OrganizationCustomOrgRoles},
		{"organization-custom-properties", config.OrganizationCustomProperties},
		{"organization-custom-repository-roles", config.OrganizationCustomRepositoryRoles},
		{"organization-announcement-banners", config.OrganizationAnnouncementBanners},
		{"organization-events", config.OrganizationEvents},
		{"organization-plan", config.OrganizationPlan},
		{"organization-user-blocking", config.OrganizationUserBlocking},
		{"organization-personal-access-token-requests", config.OrganizationPersonalAccessTokenReqs},
		{"organization-personal-access-tokens", config.OrganizationPersonalAccessTokens},
		{"organization-copilot", config.OrganizationCopilot},
		{"organization-codespaces", config.OrganizationCodespaces},
		{"email-addresses", config.EmailAddresses},
		{"codespaces-lifecycle-admin", config.CodespacesLifecycleAdmin},
		{"codespaces-metadata", config.CodespacesMetadata},
	}
}
