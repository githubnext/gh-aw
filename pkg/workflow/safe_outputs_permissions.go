package workflow

import (
	"slices"
	"strings"

	"github.com/github/gh-aw/pkg/logger"
)

var safeOutputsPermissionsLog = logger.New("workflow:safe_outputs_permissions")

// oidcVaultActions is the list of known GitHub Actions that require id-token: write
// to authenticate with secret vaults or cloud providers via OIDC (OpenID Connect).
// Inclusion criteria: actions that use the GitHub OIDC token to authenticate to
// external cloud providers or secret management systems. Add new entries when
// a well-known action is identified that exchanges an OIDC JWT for cloud credentials.
var oidcVaultActions = []string{
	"aws-actions/configure-aws-credentials", // AWS OIDC / Secrets Manager
	"azure/login",                           // Azure Key Vault / OIDC
	"google-github-actions/auth",            // GCP Secret Manager / OIDC
	"hashicorp/vault-action",                // HashiCorp Vault
	"cyberark/conjur-action",                // CyberArk Conjur
}

// stepsRequireIDToken returns true if any of the provided steps use a known
// OIDC/secret-vault action that requires the id-token: write permission.
func stepsRequireIDToken(steps []any) bool {
	for _, step := range steps {
		stepMap, ok := step.(map[string]any)
		if !ok {
			continue
		}
		uses, ok := stepMap["uses"].(string)
		if !ok || uses == "" {
			continue
		}
		// Strip the @version suffix before matching
		actionRef, _, _ := strings.Cut(uses, "@")
		if slices.Contains(oidcVaultActions, actionRef) {
			return true
		}
	}
	return false
}

// isHandlerStaged returns true when a safe output handler is effectively staged
// (i.e., it will only emit preview output, not make real API calls). A handler is
// staged when either the global safe-outputs staged flag is true, or the
// per-handler staged flag is true. Staged handlers do not require write permissions.
func isHandlerStaged(globalStaged, handlerStaged bool) bool {
	return globalStaged || handlerStaged
}

// getPushFallbackAsPullRequest returns the effective fallback-as-pull-request setting (defaults to true).
func getPushFallbackAsPullRequest(config *PushToPullRequestBranchConfig) bool {
	if config == nil || config.FallbackAsPullRequest == nil {
		return true // Default
	}
	return *config.FallbackAsPullRequest
}

// getCheckBranchProtection returns the effective check-branch-protection setting (defaults to true).
func getCheckBranchProtection(config *PushToPullRequestBranchConfig) bool {
	if config == nil || config.CheckBranchProtection == nil {
		return true // Default: check is enabled
	}
	return *config.CheckBranchProtection
}

// ComputePermissionsForSafeOutputs computes the minimal required permissions
// based on the configured safe-outputs. This function is used by both the
// consolidated safe outputs job and the conclusion job to ensure they only
// request the permissions they actually need.
//
// This implements the principle of least privilege by only including
// permissions that are required by the configured safe outputs.
// Handlers that are staged (globally or per-handler) are skipped because
// staged mode only emits preview output and does not make any API calls.
func ComputePermissionsForSafeOutputs(safeOutputs *SafeOutputsConfig) *Permissions {
	if safeOutputs == nil {
		safeOutputsPermissionsLog.Print("No safe outputs configured, returning empty permissions")
		return NewPermissions()
	}

	permissions := NewPermissions()

	for _, handler := range safeOutputHandlers {
		if handler.PermissionBuilder == nil {
			continue
		}
		handlerPermissions := handler.PermissionBuilder(safeOutputs)
		if handlerPermissions == nil {
			continue
		}
		if handler.Key != "" {
			safeOutputsPermissionsLog.Printf("Adding permissions for %s", handler.Key)
		}
		permissions.Merge(handlerPermissions)
	}

	// NoOp and MissingTool don't require write permissions beyond what's already included
	// They only need to comment if add-comment is already configured

	// Handle id-token permission for OIDC/secret vault actions in user-provided steps.
	// Explicit "none" disables auto-detection; explicit "write" always adds it;
	// otherwise auto-detect from the steps list.
	if safeOutputs.IDToken != nil && *safeOutputs.IDToken == "none" {
		safeOutputsPermissionsLog.Print("id-token permission explicitly disabled (none)")
	} else if safeOutputs.IDToken != nil && *safeOutputs.IDToken == "write" {
		safeOutputsPermissionsLog.Print("id-token: write explicitly requested")
		permissions.Set(PermissionIdToken, PermissionWrite)
	} else if stepsRequireIDToken(safeOutputs.Steps) {
		safeOutputsPermissionsLog.Print("Auto-detected OIDC/vault action in steps; adding id-token: write")
		permissions.Set(PermissionIdToken, PermissionWrite)
	}

	// If safeOutputs is configured but no permissions were accumulated (all handlers staged),
	// return explicit empty permissions so the compiled safe_outputs job renders
	// "permissions: {}" rather than omitting the block and inheriting workflow-level permissions.
	// This makes the security posture self-documenting in the generated YAML.
	if len(permissions.permissions) == 0 {
		safeOutputsPermissionsLog.Print("All handlers staged; returning explicit empty permissions (permissions: {})")
		return NewPermissionsEmpty()
	}

	safeOutputsPermissionsLog.Printf("Computed permissions with %d scopes", len(permissions.permissions))
	return permissions
}

// SafeOutputsConfigFromKeys builds a minimal SafeOutputsConfig from a list of safe-output
// key names (e.g. "create-issue", "add-comment"). Only the fields needed for permission
// computation are populated. This is used by external callers (e.g. the interactive wizard)
// that want to call ComputePermissionsForSafeOutputs without constructing a full config.
func SafeOutputsConfigFromKeys(keys []string) *SafeOutputsConfig {
	config := &SafeOutputsConfig{}
	for _, key := range keys {
		handler, ok := getSafeOutputHandlerByKey(key)
		if !ok || handler.NewConfig == nil {
			continue
		}
		if hasSafeOutputFieldSet(config, handler.StructField) {
			continue
		}
		if !setSafeOutputField(config, handler.StructField, handler.NewConfig()) {
			safeOutputsPermissionsLog.Printf(
				"Warning: failed to set safe-output field %q for key %q from descriptor constructor",
				handler.StructField,
				key,
			)
		}
	}
	return config
}
