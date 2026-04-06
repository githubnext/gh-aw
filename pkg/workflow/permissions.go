package workflow

import (
	"slices"

	"github.com/github/gh-aw/pkg/logger"
)

var permissionsLog = logger.New("workflow:permissions")

// convertStringToPermissionScope converts a string key to a PermissionScope
func convertStringToPermissionScope(key string) PermissionScope {
	switch key {
	// GitHub Actions permission scopes (supported by GITHUB_TOKEN)
	case "actions":
		return PermissionActions
	case "attestations":
		return PermissionAttestations
	case "checks":
		return PermissionChecks
	case "contents":
		return PermissionContents
	case "deployments":
		return PermissionDeployments
	case "discussions":
		return PermissionDiscussions
	case "id-token":
		return PermissionIdToken
	case "issues":
		return PermissionIssues
	case "metadata":
		return PermissionMetadata
	case "models":
		return PermissionModels
	case "packages":
		return PermissionPackages
	case "pages":
		return PermissionPages
	case "pull-requests":
		return PermissionPullRequests
	case "repository-projects":
		return PermissionRepositoryProj
	case "security-events":
		return PermissionSecurityEvents
	case "statuses":
		return PermissionStatuses
	case "copilot-requests":
		return PermissionCopilotRequests
	// GitHub App-only permission scopes (not supported by GITHUB_TOKEN, require a GitHub App)
	// organization-projects is included here because it is a GitHub App-only scope
	// (it is excluded from GetAllPermissionScopes() and skipped in YAML rendering).
	case "organization-projects":
		return PermissionOrganizationProj
	case "administration":
		return PermissionAdministration
	case "members":
		return PermissionMembers
	case "organization-administration":
		return PermissionOrganizationAdministration
	case "environments":
		return PermissionEnvironments
	case "git-signing":
		return PermissionGitSigning
	case "team-discussions":
		return PermissionTeamDiscussions
	case "vulnerability-alerts":
		return PermissionVulnerabilityAlerts
	case "workflows":
		return PermissionWorkflows
	case "repository-hooks":
		return PermissionRepositoryHooks
	case "organization-hooks":
		return PermissionOrganizationHooks
	case "organization-members":
		return PermissionOrganizationMembers
	case "organization-packages":
		return PermissionOrganizationPackages
	case "organization-self-hosted-runners":
		return PermissionOrganizationSelfHostedRunners
	case "single-file":
		return PermissionSingleFile
	case "codespaces":
		return PermissionCodespaces
	case "email-addresses":
		return PermissionEmailAddresses
	case "repository-custom-properties":
		return PermissionRepositoryCustomProperties
	case "organization-custom-org-roles":
		return PermissionOrganizationCustomOrgRoles
	case "organization-custom-properties":
		return PermissionOrganizationCustomProperties
	case "organization-custom-repository-roles":
		return PermissionOrganizationCustomRepositoryRoles
	case "organization-announcement-banners":
		return PermissionOrganizationAnnouncementBanners
	case "organization-events":
		return PermissionOrganizationEvents
	case "organization-plan":
		return PermissionOrganizationPlan
	case "organization-user-blocking":
		return PermissionOrganizationUserBlocking
	case "organization-personal-access-token-requests":
		return PermissionOrganizationPersonalAccessTokenReqs
	case "organization-personal-access-tokens":
		return PermissionOrganizationPersonalAccessTokens
	case "organization-copilot":
		return PermissionOrganizationCopilot
	case "organization-codespaces":
		return PermissionOrganizationCodespaces
	case "codespaces-lifecycle-admin":
		return PermissionCodespacesLifecycleAdmin
	case "codespaces-metadata":
		return PermissionCodespacesMetadata
	case "all":
		// "all" is a meta-key handled at the parser level; it is not a real scope
		return ""
	default:
		permissionsLog.Printf("Unknown permission scope key: %s", key)
		return ""
	}
}

// PermissionLevel represents the level of access (read, write, none)
type PermissionLevel string

const (
	PermissionRead  PermissionLevel = "read"
	PermissionWrite PermissionLevel = "write"
	PermissionNone  PermissionLevel = "none"
)

// PermissionScope represents a GitHub Actions permission scope
type PermissionScope string

const (
	// GitHub Actions permission scopes (supported by GITHUB_TOKEN)
	PermissionActions          PermissionScope = "actions"
	PermissionAttestations     PermissionScope = "attestations"
	PermissionChecks           PermissionScope = "checks"
	PermissionContents         PermissionScope = "contents"
	PermissionDeployments      PermissionScope = "deployments"
	PermissionDiscussions      PermissionScope = "discussions"
	PermissionIdToken          PermissionScope = "id-token"
	PermissionIssues           PermissionScope = "issues"
	PermissionMetadata         PermissionScope = "metadata"
	PermissionModels           PermissionScope = "models"
	PermissionPackages         PermissionScope = "packages"
	PermissionPages            PermissionScope = "pages"
	PermissionPullRequests     PermissionScope = "pull-requests"
	PermissionRepositoryProj   PermissionScope = "repository-projects"
	PermissionOrganizationProj PermissionScope = "organization-projects"
	PermissionSecurityEvents   PermissionScope = "security-events"
	PermissionStatuses         PermissionScope = "statuses"
	// PermissionCopilotRequests is a GitHub Actions permission scope used with the copilot-requests feature.
	// It enables use of the GitHub Actions token as the Copilot authentication token.
	PermissionCopilotRequests PermissionScope = "copilot-requests"

	// GitHub App-only permission scopes (not supported by GITHUB_TOKEN, require a GitHub App token).
	// When any of these are specified in the workflow permissions, a GitHub App must be configured.
	// These permissions are skipped when rendering GitHub Actions workflow YAML, but are passed
	// as permission-* inputs when minting GitHub App installation access tokens.

	// Repository-level GitHub App permissions
	PermissionAdministration             PermissionScope = "administration"
	PermissionEnvironments               PermissionScope = "environments"
	PermissionGitSigning                 PermissionScope = "git-signing"
	PermissionVulnerabilityAlerts        PermissionScope = "vulnerability-alerts"
	PermissionWorkflows                  PermissionScope = "workflows"
	PermissionRepositoryHooks            PermissionScope = "repository-hooks"
	PermissionSingleFile                 PermissionScope = "single-file"
	PermissionCodespaces                 PermissionScope = "codespaces"
	PermissionRepositoryCustomProperties PermissionScope = "repository-custom-properties"

	// Organization-level GitHub App permissions
	PermissionMembers                             PermissionScope = "members"
	PermissionOrganizationAdministration          PermissionScope = "organization-administration"
	PermissionTeamDiscussions                     PermissionScope = "team-discussions"
	PermissionOrganizationHooks                   PermissionScope = "organization-hooks"
	PermissionOrganizationMembers                 PermissionScope = "organization-members"
	PermissionOrganizationPackages                PermissionScope = "organization-packages"
	PermissionOrganizationSelfHostedRunners       PermissionScope = "organization-self-hosted-runners"
	PermissionOrganizationCustomOrgRoles          PermissionScope = "organization-custom-org-roles"
	PermissionOrganizationCustomProperties        PermissionScope = "organization-custom-properties"
	PermissionOrganizationCustomRepositoryRoles   PermissionScope = "organization-custom-repository-roles"
	PermissionOrganizationAnnouncementBanners     PermissionScope = "organization-announcement-banners"
	PermissionOrganizationEvents                  PermissionScope = "organization-events"
	PermissionOrganizationPlan                    PermissionScope = "organization-plan"
	PermissionOrganizationUserBlocking            PermissionScope = "organization-user-blocking"
	PermissionOrganizationPersonalAccessTokenReqs PermissionScope = "organization-personal-access-token-requests"
	PermissionOrganizationPersonalAccessTokens    PermissionScope = "organization-personal-access-tokens"
	PermissionOrganizationCopilot                 PermissionScope = "organization-copilot"
	PermissionOrganizationCodespaces              PermissionScope = "organization-codespaces"

	// User-level GitHub App permissions
	PermissionEmailAddresses           PermissionScope = "email-addresses"
	PermissionCodespacesLifecycleAdmin PermissionScope = "codespaces-lifecycle-admin"
	PermissionCodespacesMetadata       PermissionScope = "codespaces-metadata"
)

// GetAllPermissionScopes returns all GitHub Actions permission scopes (supported by GITHUB_TOKEN).
// These are the scopes that can be set on the workflow's GITHUB_TOKEN.
// For GitHub App-only scopes, see GetAllGitHubAppOnlyScopes.
func GetAllPermissionScopes() []PermissionScope {
	return []PermissionScope{
		PermissionActions,
		PermissionAttestations,
		PermissionChecks,
		PermissionContents,
		PermissionDeployments,
		PermissionDiscussions,
		PermissionIdToken,
		PermissionIssues,
		PermissionMetadata,
		PermissionModels,
		PermissionPackages,
		PermissionPages,
		PermissionPullRequests,
		PermissionRepositoryProj,
		PermissionSecurityEvents,
		PermissionStatuses,
	}
}

// GetAllGitHubAppOnlyScopes returns all GitHub App-only permission scopes.
// These scopes are not supported by GITHUB_TOKEN and require a GitHub App installation token.
// When any of these scopes are used in a workflow, a GitHub App must be configured.
func GetAllGitHubAppOnlyScopes() []PermissionScope {
	return []PermissionScope{
		// Repository-level GitHub App permissions
		PermissionAdministration,
		PermissionEnvironments,
		PermissionGitSigning,
		PermissionVulnerabilityAlerts,
		PermissionWorkflows,
		PermissionRepositoryHooks,
		PermissionSingleFile,
		PermissionCodespaces,
		PermissionRepositoryCustomProperties,
		// Organization-level GitHub App permissions
		PermissionOrganizationProj,
		PermissionMembers,
		PermissionOrganizationAdministration,
		PermissionTeamDiscussions,
		PermissionOrganizationHooks,
		PermissionOrganizationMembers,
		PermissionOrganizationPackages,
		PermissionOrganizationSelfHostedRunners,
		PermissionOrganizationCustomOrgRoles,
		PermissionOrganizationCustomProperties,
		PermissionOrganizationCustomRepositoryRoles,
		PermissionOrganizationAnnouncementBanners,
		PermissionOrganizationEvents,
		PermissionOrganizationPlan,
		PermissionOrganizationUserBlocking,
		PermissionOrganizationPersonalAccessTokenReqs,
		PermissionOrganizationPersonalAccessTokens,
		PermissionOrganizationCopilot,
		PermissionOrganizationCodespaces,
		// User-level GitHub App permissions
		PermissionEmailAddresses,
		PermissionCodespacesLifecycleAdmin,
		PermissionCodespacesMetadata,
	}
}

// IsGitHubAppOnlyScope returns true if the scope is a GitHub App-only permission
// (not supported by GITHUB_TOKEN). These scopes require a GitHub App to exercise.
func IsGitHubAppOnlyScope(scope PermissionScope) bool {
	isAppOnly := slices.Contains(GetAllGitHubAppOnlyScopes(), scope)
	if isAppOnly {
		permissionsLog.Printf("Scope %q requires GitHub App (not supported by GITHUB_TOKEN)", scope)
	}
	return isAppOnly
}

// Permissions represents GitHub Actions permissions
// It can be a shorthand (read-all, write-all, read, write, none) or a map of scopes to levels
// It can also have an "all" permission that expands to all scopes
type Permissions struct {
	shorthand     string
	permissions   map[PermissionScope]PermissionLevel
	hasAll        bool
	allLevel      PermissionLevel
	explicitEmpty bool // When true, renders "permissions: {}" even if no permissions are set
}
