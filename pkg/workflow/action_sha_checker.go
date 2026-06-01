package workflow

import (
	"context"
	"fmt"
	"os"
	"regexp"

	"github.com/github/gh-aw/pkg/logger"
	"github.com/goccy/go-yaml"
)

var actionSHACheckerLog = logger.New("workflow:action_sha_checker")

// actionUsesPattern matches action references in lock files:
// owner/repo@40-char-hex-sha with optional version comment
var actionUsesPattern = regexp.MustCompile(`([a-zA-Z0-9_.-]+/[a-zA-Z0-9_.-]+(?:/[a-zA-Z0-9_.-]+)*)@([0-9a-f]{40})(?:\s*#\s*([^\s]+))?`)

// ActionUsage represents an action used in a workflow with its SHA
type ActionUsage struct {
	Repo    string // e.g., "actions/checkout"
	SHA     string // The SHA currently used
	Version string // The version tag if available (e.g., "v5")
}

// ActionUpdateCheck represents the result of checking if an action needs updating
type ActionUpdateCheck struct {
	Action      ActionUsage
	NeedsUpdate bool
	LatestSHA   string
	Message     string
}

// ExtractActionsFromLockFile parses a lock.yml file and extracts all action usages
func ExtractActionsFromLockFile(lockFilePath string) ([]ActionUsage, error) {
	actionSHACheckerLog.Printf("Extracting actions from lock file: %s", lockFilePath)

	content, err := os.ReadFile(lockFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read lock file: %w", err)
	}

	// Validate lock file schema compatibility before parsing
	if err := ValidateLockSchemaCompatibility(string(content), lockFilePath); err != nil {
		return nil, err
	}
	if err := validateLockYAML(content); err != nil {
		return nil, err
	}
	return extractActionsFromContent(string(content)), nil
}

func validateLockYAML(content []byte) error {
	var workflowData map[string]any
	if err := yaml.Unmarshal(content, &workflowData); err != nil {
		return fmt.Errorf("failed to parse lock file YAML: %w", err)
	}
	return nil
}

func extractActionsFromContent(content string) []ActionUsage {
	actions := make(map[string]ActionUsage) // Use map to deduplicate
	matches := actionUsesPattern.FindAllStringSubmatch(content, -1)

	for _, match := range matches {
		action, ok := parseActionUsageMatch(match)
		if !ok {
			continue
		}
		actionKey := action.Repo + "@" + action.SHA
		if _, exists := actions[actionKey]; exists {
			continue
		}
		actions[actionKey] = action
	}

	result := make([]ActionUsage, 0, len(actions))
	for _, action := range actions {
		result = append(result, action)
	}

	actionSHACheckerLog.Printf("Extracted %d unique actions", len(result))
	return result
}

func parseActionUsageMatch(match []string) (ActionUsage, bool) {
	if len(match) < 3 {
		return ActionUsage{}, false
	}
	repo := match[1]
	sha := match[2]
	version := resolveActionUsageVersion(repo, sha, match)
	return ActionUsage{Repo: repo, SHA: sha, Version: version}, true
}

func resolveActionUsageVersion(repo, sha string, match []string) string {
	if len(match) >= 4 && match[3] != "" {
		actionSHACheckerLog.Printf("Found action: %s@%s (version: %s)", repo, sha, match[3])
		return match[3]
	}
	if pin, found := getLatestActionPinByRepo(repo); found {
		actionSHACheckerLog.Printf("Found action: %s@%s (version from pins: %s)", repo, sha, pin.Version)
		return pin.Version
	}
	actionSHACheckerLog.Printf("Found action: %s@%s (no version)", repo, sha)
	return ""
}

// CheckActionSHAUpdates checks if actions need updating by comparing with latest SHAs
func CheckActionSHAUpdates(ctx context.Context, actions []ActionUsage, resolver *ActionResolver) []ActionUpdateCheck {
	actionSHACheckerLog.Printf("Checking %d actions for updates", len(actions))

	results := make([]ActionUpdateCheck, 0, len(actions))

	for _, action := range actions {
		check := ActionUpdateCheck{
			Action:      action,
			NeedsUpdate: false,
		}

		// Skip if we don't have a version to check against
		if action.Version == "" {
			actionSHACheckerLog.Printf("Skipping %s: no version tag available", action.Repo)
			continue
		}

		// Resolve the latest SHA for this version
		latestSHA, err := resolver.ResolveSHA(ctx, action.Repo, action.Version)
		if err != nil {
			actionSHACheckerLog.Printf("Failed to resolve %s@%s: %v", action.Repo, action.Version, err)
			check.Message = fmt.Sprintf("Unable to check for updates: %v", err)
			results = append(results, check)
			continue
		}

		check.LatestSHA = latestSHA

		// Compare SHAs
		if action.SHA != latestSHA {
			check.NeedsUpdate = true
			check.Message = fmt.Sprintf("Action %s@%s is using SHA %s but latest is %s",
				action.Repo, action.Version, action.SHA[:7], latestSHA[:7])
			actionSHACheckerLog.Printf("UPDATE NEEDED: %s", check.Message)
		} else {
			actionSHACheckerLog.Printf("Action %s@%s is up to date", action.Repo, action.Version)
		}

		results = append(results, check)
	}

	return results
}
