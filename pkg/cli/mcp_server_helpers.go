package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/github/gh-aw/pkg/sliceutil"
	"github.com/github/gh-aw/pkg/workflow"
	"github.com/modelcontextprotocol/go-sdk/jsonrpc"
)

// mcpErrorData marshals data to JSON for use in jsonrpc.Error.Data field.
// Returns nil if marshaling fails to avoid errors in error handling.
func mcpErrorData(v any) json.RawMessage {
	if v == nil {
		return nil
	}
	data, err := json.Marshal(v)
	if err != nil {
		// Log the error but return nil to avoid breaking error handling
		mcpLog.Printf("Failed to marshal error data: %v", err)
		return nil
	}
	return data
}

// boolPtr returns a pointer to the given bool value, used for optional *bool fields.
func boolPtr(b bool) *bool { return &b }

// getRepository retrieves the current repository name (owner/repo format).
// Results are cached for 1 hour to avoid repeated queries.
// Checks GITHUB_REPOSITORY environment variable first, then falls back to gh repo view.
func getRepository() (string, error) {
	// Check cache first
	if repo, ok := mcpCache.GetRepo(); ok {
		mcpLog.Printf("Using cached repository: %s", repo)
		return repo, nil
	}

	// Try GITHUB_REPOSITORY environment variable first
	repo := os.Getenv("GITHUB_REPOSITORY")
	if repo != "" {
		mcpLog.Printf("Got repository from GITHUB_REPOSITORY: %s", repo)
		mcpCache.SetRepo(repo)
		return repo, nil
	}

	// Fall back to gh repo view
	mcpLog.Print("Querying repository using gh repo view")
	cmd := workflow.ExecGH("repo", "view", "--json", "nameWithOwner", "--jq", ".nameWithOwner")
	output, err := cmd.Output()
	if err != nil {
		mcpLog.Printf("Failed to get repository: %v", err)
		return "", fmt.Errorf("failed to get repository: %w", err)
	}

	repo = strings.TrimSpace(string(output))
	if repo == "" {
		return "", fmt.Errorf("repository not found")
	}

	mcpLog.Printf("Got repository from gh repo view: %s", repo)
	mcpCache.SetRepo(repo)
	return repo, nil
}

// queryActorRole queries the GitHub API to determine the actor's role in the repository.
// Returns the permission level (admin, maintain, write, triage, read) or an error.
// Results are cached for 1 hour to avoid excessive API calls.
func queryActorRole(ctx context.Context, actor string, repo string) (string, error) {
	if actor == "" {
		return "", fmt.Errorf("actor not specified")
	}
	if repo == "" {
		return "", fmt.Errorf("repository not specified")
	}

	// Check cache first
	if perm, ok := mcpCache.GetPermission(actor, repo); ok {
		mcpLog.Printf("Using cached permission for %s in %s: %s", actor, repo, perm)
		return perm, nil
	}

	// Query GitHub API for user's permission level
	// GET /repos/{owner}/{repo}/collaborators/{username}/permission
	apiPath := fmt.Sprintf("/repos/%s/collaborators/%s/permission", repo, actor)
	mcpLog.Printf("Querying GitHub API for %s's permission in %s", actor, repo)

	cmd := workflow.ExecGHContext(ctx, "api", apiPath, "--jq", ".permission")
	output, err := cmd.Output()
	if err != nil {
		mcpLog.Printf("Failed to query actor permission: %v", err)
		return "", fmt.Errorf("failed to query actor permission: %w", err)
	}

	permission := strings.TrimSpace(string(output))
	if permission == "" {
		return "", fmt.Errorf("no permission found for actor %s in repository %s", actor, repo)
	}

	mcpCache.SetPermission(actor, repo, permission)
	mcpLog.Printf("Cached permission for %s in %s: %s", actor, repo, permission)

	return permission, nil
}

// hasWriteAccess checks if the given permission level is write or higher.
// Permission levels from highest to lowest: admin, maintain, write, triage, read
func hasWriteAccess(permission string) bool {
	switch permission {
	case "admin", "maintain", "write":
		return true
	default:
		return false
	}
}

// validateWorkflowName validates that a workflow name exists.
// Returns nil if the workflow exists, or an error with suggestions if not.
// Empty workflow names are considered valid (means all workflows).
func validateWorkflowName(workflowName string) error {
	// Empty workflow name means "all workflows" - this is valid
	if workflowName == "" {
		return nil
	}

	mcpLog.Printf("Validating workflow name: %s", workflowName)

	// Try to resolve as workflow ID first
	resolvedName, err := workflow.ResolveWorkflowName(workflowName)
	if err == nil {
		mcpLog.Printf("Workflow name resolved successfully: %s -> %s", workflowName, resolvedName)
		return nil
	}

	// Check if it's a valid GitHub Actions workflow name
	agenticWorkflowNames, nameErr := getAgenticWorkflowNames(false)
	if nameErr == nil && sliceutil.Contains(agenticWorkflowNames, workflowName) {
		mcpLog.Printf("Workflow name is valid GitHub Actions workflow name: %s", workflowName)
		return nil
	}

	// Workflow not found - build error with suggestions
	mcpLog.Printf("Workflow name not found: %s", workflowName)

	suggestions := []string{
		"Use the 'status' tool to see all available workflows",
		"Check for typos in the workflow name",
		"Use the workflow ID (e.g., 'test-claude') or GitHub Actions workflow name (e.g., 'Test Claude')",
	}

	// Add fuzzy match suggestions
	similarNames := suggestWorkflowNames(workflowName)
	if len(similarNames) > 0 {
		suggestions = append([]string{fmt.Sprintf("Did you mean: %s?", strings.Join(similarNames, ", "))}, suggestions...)
	}

	return fmt.Errorf("workflow '%s' not found. %s", workflowName, strings.Join(suggestions, " "))
}

// checkActorPermission validates if the actor has sufficient permissions for restricted tools.
// Returns nil if access is allowed, or a jsonrpc.Error if access is denied.
// Uses GitHub API to query the actor's actual repository role with 1-hour caching.
func checkActorPermission(actor string, validateActor bool, toolName string) error {
	// If validation is disabled, always allow access
	if !validateActor {
		mcpLog.Printf("Tool %s: access allowed (validation disabled)", toolName)
		return nil
	}

	// If validation is enabled but no actor is specified, deny access
	if actor == "" {
		mcpLog.Printf("Tool %s: access denied (no actor specified, validation enabled)", toolName)
		return &jsonrpc.Error{
			Code:    jsonrpc.CodeInvalidRequest,
			Message: "permission denied: insufficient role",
			Data: mcpErrorData(map[string]any{
				"error":  "GITHUB_ACTOR environment variable not set",
				"tool":   toolName,
				"reason": "This tool requires at least write access to the repository. Set GITHUB_ACTOR environment variable to enable access.",
			}),
		}
	}

	// Get repository using cached lookup
	repo, err := getRepository()
	if err != nil {
		mcpLog.Printf("Tool %s: failed to get repository context, allowing access: %v", toolName, err)
		// If we can't determine the repository, allow access (fail open)
		return nil
	}

	if repo == "" {
		mcpLog.Printf("Tool %s: no repository context, allowing access", toolName)
		// No repository context, allow access
		return nil
	}

	// Query actor's role in the repository with caching
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	permission, err := queryActorRole(ctx, actor, repo)
	if err != nil {
		mcpLog.Printf("Tool %s: failed to query actor role, denying access: %v", toolName, err)
		return &jsonrpc.Error{
			Code:    jsonrpc.CodeInternalError,
			Message: "permission denied: unable to verify repository access",
			Data: mcpErrorData(map[string]any{
				"error":      err.Error(),
				"tool":       toolName,
				"actor":      actor,
				"repository": repo,
				"reason":     "Failed to query actor's repository permissions from GitHub API.",
			}),
		}
	}

	// Check if the actor has write+ access
	if !hasWriteAccess(permission) {
		mcpLog.Printf("Tool %s: access denied for actor %s (permission: %s, requires: write+)", toolName, actor, permission)
		return &jsonrpc.Error{
			Code:    jsonrpc.CodeInvalidRequest,
			Message: "permission denied: insufficient role",
			Data: mcpErrorData(map[string]any{
				"error":      "insufficient repository permissions",
				"tool":       toolName,
				"actor":      actor,
				"repository": repo,
				"role":       permission,
				"required":   "write, maintain, or admin",
				"reason":     fmt.Sprintf("Actor %s has %s access to %s. This tool requires at least write access.", actor, permission, repo),
			}),
		}
	}

	mcpLog.Printf("Tool %s: access allowed for actor %s (permission: %s)", toolName, actor, permission)
	return nil
}
