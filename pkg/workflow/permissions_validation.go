package workflow

import (
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"slices"
	"sort"
	"strings"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/setutil"
	"github.com/github/gh-aw/pkg/sliceutil"
	"github.com/github/gh-aw/pkg/stringutil"
	"github.com/goccy/go-yaml"
)

// PermissionsValidationResult contains the result of permissions validation
type PermissionsValidationResult struct {
	MissingPermissions    map[PermissionScope]PermissionLevel // Permissions required but not granted
	ReadOnlyMode          bool                                // Whether the GitHub MCP is in read-only mode
	HasValidationIssues   bool                                // Whether there are any validation issues
	MissingToolsetDetails map[string][]PermissionScope        // Maps toolset name to missing permissions
}

// ValidatePermissions validates that workflow permissions match the required GitHub MCP toolsets
// This is the general-purpose permission validator used during workflow compilation to check
// that the declared permissions are sufficient for the GitHub MCP toolsets being used.
//
// Parameters:
//   - permissions: The workflow's declared permissions
//   - githubTool: The GitHub tool configuration implementing ValidatableTool interface
//   - parsedToolsets: optional pre-parsed toolsets slice; when provided it is used directly
//     instead of calling ParseGitHubToolsets(githubTool.GetToolsets()). Pass nil or omit to
//     let ValidatePermissions derive the toolsets from the tool configuration.
//
// Returns:
//   - A validation result indicating any missing permissions and which toolsets require them
//
// Use ValidatePermissions (this function) for general permission validation against GitHub MCP toolsets.
// Use ValidateIncludedPermissions (in imports.go) when validating permissions from included/imported workflow files.
func ValidatePermissions(permissions *Permissions, githubTool ValidatableTool, parsedToolsets ...[]string) *PermissionsValidationResult {
	if permissionsValidationLog.Enabled() {
		permissionsValidationLog.Print("Starting permissions validation")
	}

	// MissingPermissions and MissingToolsetDetails are lazily initialized by
	// checkMissingPermissions to avoid heap allocations on the happy path
	// (no missing permissions). Callers that read these fields get nil maps,
	// which behave like empty maps for reads (len, range, index) but must not
	// be written to outside of checkMissingPermissions.
	result := &PermissionsValidationResult{}

	// If GitHub tool is not configured, no validation needed
	// Check both for nil interface and nil concrete type
	if githubTool == nil {
		if permissionsValidationLog.Enabled() {
			permissionsValidationLog.Print("No GitHub tool configured (nil interface), skipping validation")
		}
		return result
	}

	// Check if concrete type is nil (interface wrapping nil pointer)
	if config, ok := githubTool.(*GitHubToolConfig); ok && config == nil {
		if permissionsValidationLog.Enabled() {
			permissionsValidationLog.Print("No GitHub tool configured (nil concrete type), skipping validation")
		}
		return result
	}

	readOnly := githubTool.IsReadOnly()
	result.ReadOnlyMode = readOnly

	// Use pre-parsed toolsets when provided (avoids redundant ParseGitHubToolsets calls in hot paths).
	toolsets := resolveValidationToolsets(githubTool, readOnly, parsedToolsets...)
	if len(toolsets) == 0 {
		if permissionsValidationLog.Enabled() {
			permissionsValidationLog.Print("No toolsets to validate")
		}
		return result
	}

	// Collect required permissions for all toolsets
	requiredPermissions := collectRequiredPermissions(toolsets, readOnly)
	if permissionsValidationLog.Enabled() {
		permissionsValidationLog.Printf("Required permissions: %v", requiredPermissions)
	}

	// Check for missing permissions
	checkMissingPermissions(permissions, requiredPermissions, toolsets, result)

	result.HasValidationIssues = len(result.MissingPermissions) > 0
	if permissionsValidationLog.Enabled() {
		permissionsValidationLog.Printf("Validation complete: hasIssues=%v, missingCount=%d", result.HasValidationIssues, len(result.MissingPermissions))
	}

	return result
}

func resolveValidationToolsets(githubTool ValidatableTool, readOnly bool, parsedToolsets ...[]string) []string {
	if len(parsedToolsets) > 0 && parsedToolsets[0] != nil {
		toolsets := parsedToolsets[0]
		if permissionsValidationLog.Enabled() {
			permissionsValidationLog.Printf("Validating with pre-parsed toolsets: %v, read-only: %v", toolsets, readOnly)
		}
		return toolsets
	}
	toolsetsStr := githubTool.GetToolsets()
	if permissionsValidationLog.Enabled() {
		permissionsValidationLog.Printf("Validating toolsets: %s, read-only: %v", toolsetsStr, readOnly)
	}
	return ParseGitHubToolsets(toolsetsStr)
}

// checkMissingPermissions checks if all required permissions are granted
func checkMissingPermissions(permissions *Permissions, required map[PermissionScope]PermissionLevel, toolsets []string, result *PermissionsValidationResult) {
	if permissionsValidationLog.Enabled() {
		permissionsValidationLog.Printf("Checking missing permissions: required_count=%d, toolsets=%v", len(required), toolsets)
	}
	// toolsetPerms is cached on first missing permission to avoid the accessor
	// overhead on the happy path (all permissions granted).
	var toolsetPerms map[string]GitHubToolsetPermissions
	for scope, requiredLevel := range required {
		grantedLevel, granted := permissions.Get(scope)

		missing := false
		if !granted {
			missing = true
		} else if requiredLevel == PermissionWrite && grantedLevel != PermissionWrite {
			missing = true
		}

		if missing {
			// Lazily initialize maps on the first missing permission to avoid
			// heap allocations on the happy path (all permissions granted).
			if result.MissingPermissions == nil {
				result.MissingPermissions = make(map[PermissionScope]PermissionLevel)
			}
			result.MissingPermissions[scope] = requiredLevel

			if result.MissingToolsetDetails == nil {
				result.MissingToolsetDetails = make(map[string][]PermissionScope)
			}
			if toolsetPerms == nil {
				toolsetPerms = getToolsetPermissionsMap()
			}
			// Track which toolsets require this permission
			for _, toolset := range toolsets {
				perms, exists := toolsetPerms[toolset]
				if !exists {
					continue
				}

				requiresScope := slices.Contains(perms.ReadPermissions, scope)
				if !requiresScope {
					if slices.Contains(perms.WritePermissions, scope) {
						requiresScope = true
					}
				}

				if requiresScope {
					result.MissingToolsetDetails[toolset] = append(result.MissingToolsetDetails[toolset], scope)
				}
			}
		}
	}
}

// FormatValidationMessage formats the validation result into a human-readable message
func FormatValidationMessage(result *PermissionsValidationResult, strict bool) string {
	if !result.HasValidationIssues {
		return ""
	}

	// Format missing permissions
	if len(result.MissingPermissions) > 0 {
		return formatMissingPermissionsMessage(result)
	}

	return ""
}

// formatMissingPermissionsMessage formats the missing permissions error message
func formatMissingPermissionsMessage(result *PermissionsValidationResult) string {
	var scopes []string
	for scope := range result.MissingPermissions {
		scopes = append(scopes, string(scope))
	}
	sort.Strings(scopes)

	var lines []string

	// Build permission list with toolset details inline
	permLines := missingPermissionLines(result, scopes)

	lines = append(lines, "Missing required permissions for GitHub toolsets:")
	lines = append(lines, permLines...)
	lines = append(lines, "")
	lines = append(lines, "To fix this, you can either:")
	lines = append(lines, "")
	lines = append(lines, "Option 1: Add missing permissions to your workflow frontmatter:")
	lines = append(lines, "permissions:")
	for _, scopeStr := range scopes {
		scope := PermissionScope(scopeStr)
		level := result.MissingPermissions[scope]
		lines = append(lines, fmt.Sprintf("  %s: %s", scope, level))
	}
	lines = append(lines, "")
	lines = append(lines, fmt.Sprintf("See: %s", constants.DocsPermissionsURL))

	// Add suggestion to reduce toolsets if we have toolset details
	lines = appendMissingToolsetReductionSuggestion(lines, result)

	return strings.Join(lines, "\n")
}

func missingPermissionLines(result *PermissionsValidationResult, scopes []string) []string {
	var permLines []string
	for _, scopeStr := range scopes {
		scope := PermissionScope(scopeStr)
		level := result.MissingPermissions[scope]
		requiredBy := missingPermissionRequiredBy(result, scope)
		if len(requiredBy) > 0 {
			sort.Strings(requiredBy)
			permLines = append(permLines, fmt.Sprintf("  - %s: %s (required by %s)", scope, level, strings.Join(requiredBy, ", ")))
		} else {
			permLines = append(permLines, fmt.Sprintf("  - %s: %s", scope, level))
		}
	}
	return permLines
}

func missingPermissionRequiredBy(result *PermissionsValidationResult, scope PermissionScope) []string {
	var requiredBy []string
	if len(result.MissingToolsetDetails) == 0 {
		return requiredBy
	}
	for toolset, toolsetScopes := range result.MissingToolsetDetails {
		if slices.Contains(toolsetScopes, scope) {
			requiredBy = append(requiredBy, toolset)
		}
	}
	return requiredBy
}

func appendMissingToolsetReductionSuggestion(lines []string, result *PermissionsValidationResult) []string {
	if len(result.MissingToolsetDetails) == 0 {
		return lines
	}
	lines = append(lines, "")
	lines = append(lines, "Option 2: Reduce the required toolsets in your workflow:")
	lines = append(lines, "Remove or adjust toolsets that require these permissions:")
	toolsetsMap := make(map[string]struct{})
	for toolset := range result.MissingToolsetDetails {
		toolsetsMap[toolset] = struct{}{}
	}
	for _, toolset := range sliceutil.SortedKeys(toolsetsMap) {
		lines = append(lines, "  - "+toolset)
	}
	return lines
}

// ValidateIncludedPermissions validates that the main workflow permissions satisfy the imported
// workflow requirements. This function is specifically used when merging included/imported workflow
// files to ensure the main workflow has sufficient permissions to support all imported files.
//
// Use ValidatePermissions (in permissions_validator.go) for general permission validation against
// GitHub MCP toolsets. Use ValidateIncludedPermissions (this function) when validating permissions
// from included/imported workflow files.
func (c *Compiler) ValidateIncludedPermissions(topPermissionsYAML string, importedPermissionsJSON string) error {
	permissionsValidationLog.Print("Validating included workflow permissions")

	// If no imported permissions, no validation needed
	if importedPermissionsJSON == "" || importedPermissionsJSON == "{}" {
		permissionsValidationLog.Print("No included workflow permissions to validate")
		return nil
	}

	// Parse top-level permissions
	topPerms := parseTopPermissionsForInclude(topPermissionsYAML)

	// Track missing permissions
	missingPermissions := make(map[PermissionScope]PermissionLevel)
	insufficientPermissions := make(map[PermissionScope]includedPermissionGap)

	// Split by newlines to handle multiple JSON objects from different imports
	lines := strings.Split(importedPermissionsJSON, "\n")
	permissionsValidationLog.Printf("Processing %d permission definition lines", len(lines))

	collectIncludedPermissionIssues(lines, topPerms, missingPermissions, insufficientPermissions)

	// If there are missing or insufficient permissions, return an error
	if len(missingPermissions) > 0 || len(insufficientPermissions) > 0 {
		return errors.New(buildIncludedPermissionsError(missingPermissions, insufficientPermissions))
	}

	permissionsValidationLog.Print("All included workflow permissions are satisfied by main workflow")
	return nil
}

type includedPermissionGap struct {
	required PermissionLevel
	current  PermissionLevel
}

func parseTopPermissionsForInclude(topPermissionsYAML string) *Permissions {
	if topPermissionsYAML != "" {
		return NewPermissionsParser(topPermissionsYAML).ToPermissions()
	}
	return NewPermissions()
}

func collectIncludedPermissionIssues(lines []string, topPerms *Permissions, missing map[PermissionScope]PermissionLevel, insufficient map[PermissionScope]includedPermissionGap) {
	for _, line := range lines {
		importedPermsMap, ok := parseIncludedPermissionLine(line)
		if !ok {
			continue
		}
		for scopeStr, levelValue := range importedPermsMap {
			requiredLevel, ok := includedRequiredLevel(levelValue)
			if !ok {
				continue
			}
			recordIncludedPermissionIssue(topPerms, PermissionScope(scopeStr), requiredLevel, missing, insufficient)
		}
	}
}

func parseIncludedPermissionLine(line string) (map[string]any, bool) {
	line = strings.TrimSpace(line)
	if line == "" || line == "{}" {
		return nil, false
	}
	var importedPermsMap map[string]any
	if err := json.Unmarshal([]byte(line), &importedPermsMap); err != nil {
		permissionsValidationLog.Printf("Skipping malformed permission entry: %q (error: %v)", line, err)
		return nil, false
	}
	return importedPermsMap, true
}

func includedRequiredLevel(levelValue any) (PermissionLevel, bool) {
	levelStr, ok := levelValue.(string)
	if !ok {
		return "", false
	}
	return PermissionLevel(levelStr), true
}

func recordIncludedPermissionIssue(topPerms *Permissions, scope PermissionScope, required PermissionLevel, missing map[PermissionScope]PermissionLevel, insufficient map[PermissionScope]includedPermissionGap) {
	currentLevel, exists := topPerms.Get(scope)
	if !exists || currentLevel == PermissionNone {
		missing[scope] = required
		permissionsValidationLog.Printf("Missing permission: %s: %s", scope, required)
		return
	}
	if !isPermissionSufficient(currentLevel, required) {
		insufficient[scope] = includedPermissionGap{required: required, current: currentLevel}
		permissionsValidationLog.Printf("Insufficient permission: %s: has %s, needs %s", scope, currentLevel, required)
	}
}

func buildIncludedPermissionsError(missing map[PermissionScope]PermissionLevel, insufficient map[PermissionScope]includedPermissionGap) string {
	var errorMsg strings.Builder
	errorMsg.WriteString("ERROR: Imported workflows require permissions that are not granted in the main workflow.\n\n")
	errorMsg.WriteString("The permission set must be explicitly declared in the main workflow.\n\n")
	appendIncludedMissingPermissions(&errorMsg, missing)
	appendIncludedInsufficientPermissions(&errorMsg, insufficient)
	appendIncludedPermissionsSuggestion(&errorMsg, missing, insufficient)
	return errorMsg.String()
}

func appendIncludedMissingPermissions(errorMsg *strings.Builder, missing map[PermissionScope]PermissionLevel) {
	if len(missing) == 0 {
		return
	}
	errorMsg.WriteString("Missing permissions:\n")
	scopes := sortedPermissionScopeKeys(missing)
	for _, scope := range scopes {
		fmt.Fprintf(errorMsg, "  - %s: %s\n", scope, missing[scope])
	}
	errorMsg.WriteString("\n")
}

func appendIncludedInsufficientPermissions(errorMsg *strings.Builder, insufficient map[PermissionScope]includedPermissionGap) {
	if len(insufficient) == 0 {
		return
	}
	errorMsg.WriteString("Insufficient permissions:\n")
	scopes := sortedPermissionScopeKeys(insufficient)
	for _, scope := range scopes {
		info := insufficient[scope]
		fmt.Fprintf(errorMsg, "  - %s: has %s, requires %s\n", scope, info.current, info.required)
	}
	errorMsg.WriteString("\n")
}

func appendIncludedPermissionsSuggestion(errorMsg *strings.Builder, missing map[PermissionScope]PermissionLevel, insufficient map[PermissionScope]includedPermissionGap) {
	errorMsg.WriteString("Suggested fix: Add the required permissions to your main workflow frontmatter:\n")
	errorMsg.WriteString("permissions:\n")
	allRequired := make(map[PermissionScope]PermissionLevel)
	maps.Copy(allRequired, missing)
	for scope, info := range insufficient {
		allRequired[scope] = info.required
	}
	for _, scope := range sortedPermissionScopeKeys(allRequired) {
		fmt.Fprintf(errorMsg, "  %s: %s\n", scope, allRequired[scope])
	}
}

func sortedPermissionScopeKeys[V any](values map[PermissionScope]V) []PermissionScope {
	var scopes []PermissionScope
	for scope := range values {
		scopes = append(scopes, scope)
	}
	SortPermissionScopes(scopes)
	return scopes
}

// ValidatePermissionScopeNames validates that all permission scope names in the
// permissions YAML are recognized GitHub Actions permission scopes. When an
// unrecognized scope that closely resembles a valid scope is found, a "Did you
// mean?" suggestion is returned so users can quickly fix typos.
//
// Example: "contnts: read" → suggests "contents"
func ValidatePermissionScopeNames(permissionsYAML string) error {
	if permissionsYAML == "" {
		return nil
	}

	permissionsValidationLog.Print("Validating permission scope names")

	// Collect all valid scope names for fuzzy matching
	allScopes := validPermissionScopeNames()
	// "all" is a meta-key that is always valid in shorthand contexts
	validMeta := map[string]struct {
	}{
		"all":       {},
		"read-all":  {},
		"write-all": {},
		"none":      {},
	}

	// Strip optional "permissions:" prefix so we can parse just the map content
	content, ok := stripPermissionsYAMLPrefix(permissionsYAML)
	if !ok {
		return nil
	}

	// Try to parse the content as a YAML map of scope → level
	var permsMap map[string]any
	if err := yaml.Unmarshal([]byte(content), &permsMap); err != nil {
		// Not a map (e.g., a shorthand like "read-all"); nothing to validate
		return nil
	}

	for scopeKey := range permsMap {
		if setutil.Contains(validMeta, scopeKey) {
			continue
		}
		if _, ok := validPermissionScopes[scopeKey]; ok {
			continue
		}

		if err := unknownPermissionScopeError(scopeKey, allScopes); err != nil {
			return err
		}
		if len(stringutil.FindClosestMatches(scopeKey, allScopes, 3)) == 0 &&
			!hasCaseOnlyPermissionScopeMatch(scopeKey) {
			continue // too different to be a typo, ignore silently
		}
	}

	return nil
}

func validPermissionScopeNames() []string {
	ghTokenScopes := GetAllPermissionScopes()
	appOnlyScopes := GetAllGitHubAppOnlyScopes()
	allScopes := make([]string, 0, safeAllocationCapacity(len(ghTokenScopes), len(appOnlyScopes), 1))
	for _, scope := range ghTokenScopes {
		allScopes = append(allScopes, string(scope))
	}
	for _, scope := range appOnlyScopes {
		allScopes = append(allScopes, string(scope))
	}
	return append(allScopes, string(PermissionCopilotRequests))
}

func stripPermissionsYAMLPrefix(permissionsYAML string) (string, bool) {
	content := strings.TrimSpace(permissionsYAML)
	if strings.HasPrefix(content, "permissions:") {
		lines := strings.SplitN(content, "\n", 2)
		if len(lines) != 2 {
			return "", false
		}
		content = lines[1]
	}
	return content, true
}

func hasCaseOnlyPermissionScopeMatch(scopeKey string) bool {
	lowerScopeKey := strings.ToLower(scopeKey)
	if lowerScopeKey == scopeKey {
		return false
	}
	_, ok := validPermissionScopes[lowerScopeKey]
	return ok
}

func unknownPermissionScopeError(scopeKey string, allScopes []string) error {
	lowerScopeKey := strings.ToLower(scopeKey)
	if lowerScopeKey != scopeKey {
		if _, ok := validPermissionScopes[lowerScopeKey]; ok {
			return formatUnknownPermissionScopeError(scopeKey, lowerScopeKey, allScopes)
		}
	}
	permissionsValidationLog.Printf("Unknown permission scope key: %q", scopeKey)
	suggestions := stringutil.FindClosestMatches(scopeKey, allScopes, 3)
	if len(suggestions) == 0 {
		return nil
	}
	return formatUnknownPermissionScopeError(scopeKey, strings.Join(suggestions, ", "), allScopes)
}

func formatUnknownPermissionScopeError(scopeKey, suggestion string, allScopes []string) error {
	return fmt.Errorf(
		"unknown permission scope %q.\n\nDid you mean: %s?\n\nValid permission scopes include: %s\n\nSee: %s",
		scopeKey,
		suggestion,
		strings.Join(allScopes[:min(10, len(allScopes))], ", ")+"...",
		constants.DocsPermissionsURL,
	)
}
