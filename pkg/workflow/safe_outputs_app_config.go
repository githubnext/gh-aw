package workflow

import (
	"encoding/json"
	"fmt"
	"maps"
	"strings"

	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/sliceutil"
)

var safeOutputsAppLog = logger.New("workflow:safe_outputs_app")
var githubExpressionWhitespaceReplacer = strings.NewReplacer("\r\n", " ", "\n", " ", "\r", " ", "\t", " ")

// ========================================
// GitHub App Configuration
// ========================================

// GitHubAppConfig holds configuration for GitHub App-based token minting
type GitHubAppConfig struct {
	AppID           string            `yaml:"client-id,omitempty"`         // GitHub App client ID (or legacy app ID) (e.g., "${{ vars.APP_ID }}")
	PrivateKey      string            `yaml:"private-key,omitempty"`       // GitHub App private key (e.g., "${{ secrets.APP_PRIVATE_KEY }}")
	IgnoreIfMissing bool              `yaml:"ignore-if-missing,omitempty"` // If true, skip token minting when client-id/private-key resolve empty
	Owner           string            `yaml:"owner,omitempty"`             // Optional: owner of the GitHub App installation (defaults to checkout.repository owner when derivable, otherwise current repository owner)
	Repositories    []string          `yaml:"repositories,omitempty"`      // Optional: comma or newline-separated list of repositories to grant access to
	Permissions     map[string]string `yaml:"permissions,omitempty"`       // Optional: extra permission-* fields to merge into the minted token (nested wins over job-level)
}

// ========================================
// App Configuration Parsing
// ========================================

// parseAppConfig parses the app configuration from a map
func parseAppConfig(appMap map[string]any) *GitHubAppConfig {
	safeOutputsAppLog.Print("Parsing GitHub App configuration")
	appConfig := &GitHubAppConfig{}
	appConfig.AppID = parseGitHubAppID(appMap)
	appConfig.PrivateKey = parseGitHubAppString(appMap, "private-key")
	appConfig.IgnoreIfMissing = parseGitHubAppIgnoreIfMissing(appMap)
	appConfig.Owner = parseGitHubAppString(appMap, "owner")
	appConfig.Repositories = parseGitHubAppRepositories(appMap)
	appConfig.Permissions = parseGitHubAppPermissions(appMap)
	return appConfig
}

func parseGitHubAppID(appMap map[string]any) string {
	if clientID, exists := appMap["client-id"]; exists {
		clientIDStr, _ := clientID.(string)
		return clientIDStr
	}
	return parseGitHubAppString(appMap, "app-id")
}

func parseGitHubAppString(appMap map[string]any, key string) string {
	value, exists := appMap[key]
	if !exists {
		return ""
	}
	valueStr, _ := value.(string)
	return valueStr
}

func parseGitHubAppIgnoreIfMissing(appMap map[string]any) bool {
	ignoreIfMissing, exists := appMap["ignore-if-missing"]
	if !exists {
		return false
	}
	ignore, ok := ignoreIfMissing.(bool)
	if !ok {
		safeOutputsAppLog.Printf("Ignoring github-app.ignore-if-missing: expected boolean, got %T", ignoreIfMissing)
		return false
	}
	return ignore
}

func parseGitHubAppRepositories(appMap map[string]any) []string {
	repos, exists := appMap["repositories"]
	if !exists {
		return nil
	}
	reposArray, ok := repos.([]any)
	if !ok {
		return nil
	}
	repoStrings := make([]string, 0, len(reposArray))
	for _, repo := range reposArray {
		if repoStr, ok := repo.(string); ok {
			repoStrings = append(repoStrings, repoStr)
		}
	}
	return repoStrings
}

func parseGitHubAppPermissions(appMap map[string]any) map[string]string {
	perms, exists := appMap["permissions"]
	if !exists {
		return nil
	}
	permsMap, ok := perms.(map[string]any)
	if !ok {
		safeOutputsAppLog.Printf("Ignoring github-app.permissions: expected object, got %T", perms)
		return nil
	}
	permissions := make(map[string]string, len(permsMap))
	for key, val := range permsMap {
		if valStr, ok := val.(string); ok {
			permissions[key] = valStr
		} else {
			safeOutputsAppLog.Printf("Ignoring github-app.permissions[%q]: expected string value, got %T", key, val)
		}
	}
	return permissions
}

func (app *GitHubAppConfig) shouldIgnoreMissingKey() bool {
	if app == nil {
		return false
	}
	return app.IgnoreIfMissing
}

func (app *GitHubAppConfig) hasRequiredCredentials() bool {
	if app == nil {
		return false
	}
	return strings.TrimSpace(app.AppID) != "" && strings.TrimSpace(app.PrivateKey) != ""
}

// extractWrappedGitHubExpression returns the inner text for values wrapped as
// `${{ ... }}` (for example, `${{ secrets.APP_ID }}` -> `secrets.APP_ID`).
// It returns false for literals and malformed/empty wrappers.
func extractWrappedGitHubExpression(value string) (string, bool) {
	trimmed := strings.TrimSpace(value)
	if !strings.HasPrefix(trimmed, "${{") || !strings.HasSuffix(trimmed, "}}") {
		return "", false
	}
	inner := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(trimmed, "${{"), "}}"))
	// Reject wrappers with no usable expression body (e.g. `${{ }}`).
	if inner == "" {
		return "", false
	}
	return inner, true
}

// buildGitHubExpressionNonEmptyCheck renders a non-empty check node from wrapped
// expressions (`${{ secrets.KEY }}` -> `secrets.KEY != ”`) or literals
// (`plain-value` -> `'plain-value' != ”`).
func buildGitHubExpressionNonEmptyCheck(value string) ConditionNode {
	trimmed := strings.TrimSpace(value)
	if inner, ok := extractWrappedGitHubExpression(trimmed); ok {
		return BuildNotEquals(&ExpressionNode{Expression: inner}, BuildStringLiteral(""))
	}
	return BuildNotEquals(BuildStringLiteral(strings.TrimSpace(githubExpressionWhitespaceReplacer.Replace(trimmed))), BuildStringLiteral(""))
}

// ifInvalidContextNames lists the GitHub Actions expression contexts that are not
// available in step-level 'if:' conditions. GitHub Actions allows matrix in this
// position, but still rejects secrets and jobs references.
// See: https://docs.github.com/en/actions/writing-workflows/choosing-what-your-workflow-does/evaluate-expressions-in-workflows-and-actions#contexts
var ifInvalidContextNames = map[string]struct{}{
	"jobs":    {},
	"secrets": {},
}

const (
	ignoreIfMissingAppIDEnvVar      = "GH_AW_IGNORE_IF_MISSING_APP_ID"
	ignoreIfMissingPrivateKeyEnvVar = "GH_AW_IGNORE_IF_MISSING_PRIVATE_KEY"
)

type stepEnvAssignment struct {
	Name  string
	Value string
}

type ignoreIfMissingGuard struct {
	Condition      string
	EnvAssignments []stepEnvAssignment
}

// isGitHubExpressionIdentifierChar reports whether a byte can appear in a GitHub
// Actions expression identifier token (ASCII letter, digit, or underscore).
func isGitHubExpressionIdentifierChar(ch byte) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') || ch == '_'
}

// isGitHubExpressionIdentifierStart reports whether inner[i] begins an identifier
// token rather than landing in the middle of one.
func isGitHubExpressionIdentifierStart(inner string, i int) bool {
	if i >= len(inner) {
		return false
	}
	return isGitHubExpressionIdentifierChar(inner[i]) && (i == 0 || !isGitHubExpressionIdentifierChar(inner[i-1]))
}

// consumeSingleQuotedGitHubExpressionString skips over a single-quoted GitHub
// expression string literal, honoring doubled single quotes as escapes. It returns
// the first byte position after the closing quote, or len(inner) if unterminated.
func consumeSingleQuotedGitHubExpressionString(inner string, start int) int {
	i := start + 1
	for i < len(inner) {
		if inner[i] != '\'' {
			i++
			continue
		}
		if i+1 < len(inner) && inner[i+1] == '\'' {
			i += 2
			continue
		}
		return i + 1
	}
	return i
}

// containsInvalidIfContextReference returns true when the inner expression body
// contains a jobs or secrets context token anywhere outside single-quoted string
// literals, including bracket notation such as secrets['TOKEN'].
func containsInvalidIfContextReference(inner string) bool {
	for i := 0; i < len(inner); {
		if inner[i] == '\'' {
			i = consumeSingleQuotedGitHubExpressionString(inner, i)
			continue
		}

		if !isGitHubExpressionIdentifierStart(inner, i) {
			i++
			continue
		}

		start := i
		for i < len(inner) && isGitHubExpressionIdentifierChar(inner[i]) {
			i++
		}
		name := inner[start:i]
		if _, ok := ifInvalidContextNames[name]; !ok {
			continue
		}

		j := i
		for j < len(inner) && (inner[j] == ' ' || inner[j] == '\t' || inner[j] == '\n' || inner[j] == '\r') {
			j++
		}
		if j < len(inner) && (inner[j] == '.' || inner[j] == '[') {
			return true
		}
	}
	return false
}

// combineGitHubIfExpressions accepts either wrapped `${{ ... }}` conditions or raw
// inner expression fragments and normalizes them into one wrapped if-expression.
func combineGitHubIfExpressions(expressions ...string) string {
	var parts []string
	for _, expression := range expressions {
		trimmed := strings.TrimSpace(expression)
		if trimmed == "" {
			continue
		}
		if inner, ok := extractWrappedGitHubExpression(trimmed); ok {
			parts = append(parts, inner)
			continue
		}
		parts = append(parts, trimmed)
	}
	if len(parts) == 0 {
		return ""
	}
	return wrapGitHubExpression(strings.Join(parts, " && "))
}

func appendStepEnvAssignments(steps []string, assignments []stepEnvAssignment) []string {
	if len(assignments) == 0 {
		return steps
	}
	steps = append(steps, "        env:\n")
	for _, assignment := range assignments {
		steps = append(steps, fmt.Sprintf("          %s: %s\n", assignment.Name, assignment.Value))
	}
	return steps
}

// buildIgnoreIfMissingCondition returns a GitHub Actions if-expression that requires
// all GitHub App credential inputs that can be checked in an if: condition to be non-empty.
// Values referencing the secrets or jobs contexts are routed through step-local env
// aliases so the guard can still check them through the supported env context.
func buildIgnoreIfMissingCondition(app *GitHubAppConfig) ignoreIfMissingGuard {
	var checks []ConditionNode
	guard := ignoreIfMissingGuard{}
	for _, credential := range []struct {
		value   string
		envName string
	}{
		{value: app.AppID, envName: ignoreIfMissingAppIDEnvVar},
		{value: app.PrivateKey, envName: ignoreIfMissingPrivateKeyEnvVar},
	} {
		trimmed := strings.TrimSpace(credential.value)
		if inner, ok := extractWrappedGitHubExpression(trimmed); ok {
			if containsInvalidIfContextReference(inner) {
				safeOutputsAppLog.Printf("Rewriting %q in ignore-if-missing condition through env.%s: context not valid in if: expressions", inner, credential.envName)
				guard.EnvAssignments = append(guard.EnvAssignments, stepEnvAssignment{
					Name:  credential.envName,
					Value: trimmed,
				})
				checks = append(checks, BuildNotEquals(&ExpressionNode{Expression: "env." + credential.envName}, BuildStringLiteral("")))
				continue
			}
		}
		checks = append(checks, buildGitHubExpressionNonEmptyCheck(credential.value))
	}
	if len(checks) == 0 {
		return guard
	}
	condition := checks[0]
	for i := 1; i < len(checks); i++ {
		condition = BuildAnd(condition, checks[i])
	}
	guard.Condition = wrapGitHubExpression(RenderCondition(condition))
	return guard
}

// ========================================
// App Configuration Merging
// ========================================

// mergeAppFromIncludedConfigs merges app configuration from included safe-outputs configurations
// If the top-level workflow has an app configured, it takes precedence
// Otherwise, the first app configuration found in included configs is used
func (c *Compiler) mergeAppFromIncludedConfigs(topSafeOutputs *SafeOutputsConfig, includedConfigs []string) (*GitHubAppConfig, error) {
	safeOutputsAppLog.Printf("Merging app configuration: included_configs=%d", len(includedConfigs))
	// If top-level workflow already has app configured, use it (no merge needed)
	if topSafeOutputs != nil && topSafeOutputs.GitHubApp != nil {
		safeOutputsAppLog.Print("Using top-level app configuration")
		return topSafeOutputs.GitHubApp, nil
	}

	// Otherwise, find the first app configuration in included configs
	for _, configJSON := range includedConfigs {
		if configJSON == "" || configJSON == "{}" {
			continue
		}

		// Parse the safe-outputs configuration
		var safeOutputsConfig map[string]any
		if err := json.Unmarshal([]byte(configJSON), &safeOutputsConfig); err != nil {
			continue // Skip invalid JSON
		}

		// Extract app from the safe-outputs.github-app field
		if appData, exists := safeOutputsConfig["github-app"]; exists {
			if appMap, ok := appData.(map[string]any); ok {
				appConfig := parseAppConfig(appMap)

				// Return first valid app configuration found
				if appConfig.AppID != "" && appConfig.PrivateKey != "" {
					safeOutputsAppLog.Print("Found valid app configuration in included config")
					return appConfig, nil
				}
			}
		}
	}

	safeOutputsAppLog.Print("No app configuration found in included configs")
	return nil, nil
}

// ========================================
// GitHub App Token Steps Generation
// ========================================

// buildGitHubAppTokenMintStep generates the step to mint a GitHub App installation access token
// Permissions are automatically computed from the safe output job requirements.
// fallbackRepoExpr overrides the default ${{ github.event.repository.name }} fallback when
// no explicit repositories are configured (e.g. pass needs.activation.outputs.target_repo_name for
// workflow_call relay workflows so the token is scoped to the platform repo's NAME, not the full
// owner/repo slug — actions/create-github-app-token expects repo names only when owner is also set).
func (c *Compiler) buildGitHubAppTokenMintStep(app *GitHubAppConfig, permissions *Permissions, fallbackRepoExpr string) []string {
	return c.buildGitHubAppTokenMintStepWithMeta(app, permissions, fallbackRepoExpr, "", "Generate GitHub App token", "safe-outputs-app-token")
}

func (c *Compiler) buildGitHubAppTokenMintStepForRepository(app *GitHubAppConfig, permissions *Permissions, fallbackRepoExpr string, ownerSourceRepository string) []string {
	return c.buildGitHubAppTokenMintStepWithMeta(app, permissions, fallbackRepoExpr, ownerSourceRepository, "Generate GitHub App token", "safe-outputs-app-token")
}

func (c *Compiler) buildGitHubAppTokenMintStepWithMeta(app *GitHubAppConfig, permissions *Permissions, fallbackRepoExpr string, ownerSourceRepository string, stepName string, stepID string) []string {
	safeOutputsAppLog.Printf("Building GitHub App token mint step: owner=%s, repos=%d", app.Owner, len(app.Repositories))
	owner, ownerSteps := resolveGitHubAppOwner(app, ownerSourceRepository, stepName, stepID)
	steps := append([]string{}, ownerSteps...)
	steps = append(steps, fmt.Sprintf("      - name: %s\n", stepName))
	steps = append(steps, fmt.Sprintf("        id: %s\n", stepID))
	steps = appendGitHubAppTokenGuard(steps, app)
	steps = appendGitHubAppTokenInputs(steps, app, owner, fallbackRepoExpr)
	steps = appendGitHubAppPermissionInputs(steps, app, permissions)
	return steps
}

func appendGitHubAppTokenGuard(steps []string, app *GitHubAppConfig) []string {
	if !app.shouldIgnoreMissingKey() {
		return steps
	}
	guard := buildIgnoreIfMissingCondition(app)
	steps = appendStepEnvAssignments(steps, guard.EnvAssignments)
	if guard.Condition != "" {
		steps = append(steps, fmt.Sprintf("        if: %s\n", guard.Condition))
	}
	return steps
}

func appendGitHubAppTokenInputs(steps []string, app *GitHubAppConfig, owner, fallbackRepoExpr string) []string {
	steps = append(steps, fmt.Sprintf("        uses: %s\n", getActionPin("actions/create-github-app-token")))
	steps = append(steps, "        with:\n")
	steps = append(steps, fmt.Sprintf("          client-id: %s\n", app.AppID))
	steps = append(steps, fmt.Sprintf("          private-key: %s\n", app.PrivateKey))
	steps = append(steps, fmt.Sprintf("          owner: %s\n", owner))
	steps = appendGitHubAppRepositoryInputs(steps, app.Repositories, fallbackRepoExpr)
	steps = append(steps, "          github-api-url: ${{ github.api_url }}\n")
	return steps
}

func appendGitHubAppRepositoryInputs(steps []string, repositories []string, fallbackRepoExpr string) []string {
	if len(repositories) == 1 && repositories[0] == "*" {
		safeOutputsAppLog.Print("Using org-wide GitHub App token (repositories: *)")
		return steps
	}
	if len(repositories) == 1 {
		return append(steps, fmt.Sprintf("          repositories: %s\n", repositories[0]))
	}
	if len(repositories) > 1 {
		steps = append(steps, "          repositories: |-\n")
		for _, repo := range repositories {
			steps = append(steps, fmt.Sprintf("            %s\n", repo))
		}
		return steps
	}
	repoExpr := fallbackRepoExpr
	if repoExpr == "" {
		repoExpr = "${{ github.event.repository.name }}"
	}
	return append(steps, fmt.Sprintf("          repositories: %s\n", repoExpr))
}

func appendGitHubAppPermissionInputs(steps []string, app *GitHubAppConfig, permissions *Permissions) []string {
	if permissions == nil {
		return steps
	}
	permissionFields := convertPermissionsToAppTokenFields(permissions)
	mergeGitHubAppPermissionOverrides(permissionFields, app.Permissions)
	for _, key := range sliceutil.SortedKeys(permissionFields) {
		steps = append(steps, fmt.Sprintf("          %s: %s\n", key, permissionFields[key]))
	}
	return steps
}

func mergeGitHubAppPermissionOverrides(permissionFields map[string]string, overrides map[string]string) {
	for key, val := range overrides {
		scope := convertStringToPermissionScope(key)
		if scope == "" {
			safeOutputsAppLog.Printf("Skipping unknown permission scope %q in github-app.permissions", key)
			continue
		}
		level := strings.ToLower(strings.TrimSpace(val))
		tempPerms := NewPermissionsFromMap(map[PermissionScope]PermissionLevel{scope: PermissionLevel(level)})
		maps.Copy(permissionFields, convertPermissionsToAppTokenFields(tempPerms))
	}
}

// convertPermissionsToAppTokenFields converts job Permissions to permission-* action inputs
// This follows GitHub's recommendation for explicit permission control.
func convertPermissionsToAppTokenFields(permissions *Permissions) map[string]string {
	fields := make(map[string]string)
	addActionPermissionFields(fields, permissions)
	addRepositoryAppPermissionFields(fields, permissions)
	addOrganizationAppPermissionFields(fields, permissions)
	addUserAppPermissionFields(fields, permissions)
	return fields
}

func addActionPermissionFields(fields map[string]string, permissions *Permissions) {
	mappings := []struct {
		scope PermissionScope
		field string
	}{
		{PermissionActions, "permission-actions"},
		{PermissionChecks, "permission-checks"},
		{PermissionContents, "permission-contents"},
		{PermissionDeployments, "permission-deployments"},
		{PermissionIssues, "permission-issues"},
		{PermissionPackages, "permission-packages"},
		{PermissionPages, "permission-pages"},
		{PermissionPullRequests, "permission-pull-requests"},
		{PermissionSecurityEvents, "permission-security-events"},
		{PermissionStatuses, "permission-statuses"},
		{PermissionVulnerabilityAlerts, "permission-vulnerability-alerts"},
		{PermissionDiscussions, "permission-discussions"},
	}
	for _, mapping := range mappings {
		if level, ok := permissions.Get(mapping.scope); ok {
			fields[mapping.field] = string(level)
		}
	}
}

func addRepositoryAppPermissionFields(fields map[string]string, permissions *Permissions) {
	addExplicitPermissionFields(fields, permissions, []appTokenPermissionMapping{
		{PermissionAdministration, "permission-administration"},
		{PermissionEnvironments, "permission-environments"},
		{PermissionGitSigning, "permission-git-signing"},
		{PermissionWorkflows, "permission-workflows"},
		{PermissionRepositoryHooks, "permission-repository-hooks"},
		{PermissionSingleFile, "permission-single-file"},
		{PermissionCodespaces, "permission-codespaces"},
		{PermissionRepositoryCustomProperties, "permission-repository-custom-properties"},
	})
}

type appTokenPermissionMapping struct {
	scope PermissionScope
	field string
}

func addExplicitPermissionFields(fields map[string]string, permissions *Permissions, mappings []appTokenPermissionMapping) {
	for _, mapping := range mappings {
		if level, ok := permissions.GetExplicit(mapping.scope); ok {
			fields[mapping.field] = string(level)
		}
	}
}

func addOrganizationAppPermissionFields(fields map[string]string, permissions *Permissions) {
	addExplicitPermissionFields(fields, permissions, []appTokenPermissionMapping{
		{PermissionOrganizationProj, "permission-organization-projects"},
		{PermissionMembers, "permission-members"},
		{PermissionOrganizationAdministration, "permission-organization-administration"},
		{PermissionTeamDiscussions, "permission-team-discussions"},
		{PermissionOrganizationHooks, "permission-organization-hooks"},
		{PermissionOrganizationMembers, "permission-organization-members"},
		{PermissionOrganizationPackages, "permission-organization-packages"},
		{PermissionOrganizationSelfHostedRunners, "permission-organization-self-hosted-runners"},
		{PermissionOrganizationCustomOrgRoles, "permission-organization-custom-org-roles"},
		{PermissionOrganizationCustomProperties, "permission-organization-custom-properties"},
		{PermissionOrganizationCustomRepositoryRoles, "permission-organization-custom-repository-roles"},
		{PermissionOrganizationAnnouncementBanners, "permission-organization-announcement-banners"},
		{PermissionOrganizationEvents, "permission-organization-events"},
		{PermissionOrganizationPlan, "permission-organization-plan"},
		{PermissionOrganizationUserBlocking, "permission-organization-user-blocking"},
		{PermissionOrganizationPersonalAccessTokenReqs, "permission-organization-personal-access-token-requests"},
		{PermissionOrganizationPersonalAccessTokens, "permission-organization-personal-access-tokens"},
		{PermissionOrganizationCopilot, "permission-organization-copilot"},
		{PermissionOrganizationCodespaces, "permission-organization-codespaces"},
	})
}

func addUserAppPermissionFields(fields map[string]string, permissions *Permissions) {
	addExplicitPermissionFields(fields, permissions, []appTokenPermissionMapping{
		{PermissionEmailAddresses, "permission-email-addresses"},
		{PermissionCodespacesLifecycleAdmin, "permission-codespaces-lifecycle-admin"},
		{PermissionCodespacesMetadata, "permission-codespaces-metadata"},
	})
}

// ========================================
// Activation Token Steps Generation
// ========================================

// buildActivationAppTokenMintStep generates the step to mint a GitHub App installation access token
// for use in the pre-activation (reaction) and activation (status comment) jobs.
func (c *Compiler) buildActivationAppTokenMintStep(app *GitHubAppConfig, permissions *Permissions) []string {
	safeOutputsAppLog.Printf("Building activation GitHub App token mint step: owner=%s", app.Owner)
	var steps []string

	steps = append(steps, "      - name: Generate GitHub App token for activation\n")
	steps = append(steps, "        id: activation-app-token\n")
	if app.shouldIgnoreMissingKey() {
		guard := buildIgnoreIfMissingCondition(app)
		steps = appendStepEnvAssignments(steps, guard.EnvAssignments)
		if guard.Condition != "" {
			steps = append(steps, fmt.Sprintf("        if: %s\n", guard.Condition))
		}
	}
	steps = append(steps, fmt.Sprintf("        uses: %s\n", getActionPin("actions/create-github-app-token")))
	steps = append(steps, "        with:\n")
	steps = append(steps, fmt.Sprintf("          client-id: %s\n", app.AppID))
	steps = append(steps, fmt.Sprintf("          private-key: %s\n", app.PrivateKey))

	// Add owner - default to current repository owner if not specified
	owner := app.Owner
	if owner == "" {
		owner = "${{ github.repository_owner }}"
	}
	steps = append(steps, fmt.Sprintf("          owner: %s\n", owner))

	// Add repositories - behavior depends on configuration:
	// - If repositories is ["*"], omit the field to allow org-wide access
	// - If repositories is a single value, use inline format
	// - If repositories has multiple values, use block scalar format (newline-separated)
	// - If repositories is empty/not specified, default to the current repository
	if len(app.Repositories) == 1 && app.Repositories[0] == "*" {
		// Org-wide access: omit repositories field entirely
		safeOutputsAppLog.Print("Using org-wide GitHub App token for activation (repositories: *)")
	} else if len(app.Repositories) == 1 {
		steps = append(steps, fmt.Sprintf("          repositories: %s\n", app.Repositories[0]))
	} else if len(app.Repositories) > 1 {
		steps = append(steps, "          repositories: |-\n")
		for _, repo := range app.Repositories {
			steps = append(steps, fmt.Sprintf("            %s\n", repo))
		}
	} else {
		steps = append(steps, "          repositories: ${{ github.event.repository.name }}\n")
	}

	// Always add github-api-url from environment variable
	steps = append(steps, "          github-api-url: ${{ github.api_url }}\n")

	// Add permission-* fields automatically computed from job permissions
	if permissions != nil {
		permissionFields := convertPermissionsToAppTokenFields(permissions)

		keys := sliceutil.SortedKeys(permissionFields)

		for _, key := range keys {
			steps = append(steps, fmt.Sprintf("          %s: %s\n", key, permissionFields[key]))
		}
	}

	return steps
}

// resolveActivationToken returns the GitHub token to use for activation steps (reactions, status comments).
// Priority: GitHub App minted token > custom github-token > GITHUB_TOKEN (default)
//
// When returning the app token reference, callers MUST ensure that buildActivationAppTokenMintStep
// has already been called to generate the 'activation-app-token' step, since this function returns
// a reference to that step's output (${{ steps.activation-app-token.outputs.token }}).
func (c *Compiler) resolveActivationToken(data *WorkflowData) string {
	if data.ActivationGitHubApp != nil {
		if data.ActivationGitHubApp.shouldIgnoreMissingKey() {
			return combineTokenExpressions("${{ steps.activation-app-token.outputs.token }}", "${{ secrets.GITHUB_TOKEN }}")
		}
		return "${{ steps.activation-app-token.outputs.token }}"
	}
	if data.ActivationGitHubToken != "" {
		return data.ActivationGitHubToken
	}
	return "${{ secrets.GITHUB_TOKEN }}"
}
