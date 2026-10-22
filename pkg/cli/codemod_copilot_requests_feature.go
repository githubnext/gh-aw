package cli

import (
	"strings"

	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/workflow/codemod"
)

var copilotRequestsFeatureCodemodLog = logger.New("cli:codemod_copilot_requests_feature")

// getCopilotRequestsFeatureToPermissionsCodemod migrates features.copilot-requests to permissions.copilot-requests: write.
func getCopilotRequestsFeatureToPermissionsCodemod() Codemod {
	return Codemod{
		ID:           "features-copilot-requests-to-permissions",
		Name:         "Migrate features.copilot-requests to permissions",
		Description:  "Removes deprecated features.copilot-requests and adds permissions.copilot-requests: write when enabled.",
		IntroducedIn: "1.0.0",
		Apply: func(content string, frontmatter map[string]any) (string, bool, error) {
			featuresAny, hasFeatures := frontmatter["features"]
			if !hasFeatures {
				return content, false, nil
			}
			featuresMap, ok := featuresAny.(map[string]any)
			if !ok {
				return content, false, nil
			}

			featureValue, hasFeature := featuresMap["copilot-requests"]
			if !hasFeature {
				return content, false, nil
			}

			enabled, isBool := featureValue.(bool)
			if !isBool {
				return content, false, nil
			}

			if enabled && !canSafelyAddCopilotRequestsPermission(frontmatter) {
				copilotRequestsFeatureCodemodLog.Print("Skipping migration: unable to safely add permissions.copilot-requests")
				return content, false, nil
			}

			addPermission := enabled && !frontmatterHasCopilotRequestsPermission(frontmatter)
			newContent, applied, err := codemod.Apply(content, func(f *codemod.Frontmatter) error {
				if err := f.Remove("features", "copilot-requests"); err != nil {
					return err
				}
				if err := f.PruneEmpty("features"); err != nil {
					return err
				}
				if addPermission {
					if value, exists := f.Lookup("permissions"); exists {
						if _, mapping := value.(map[string]any); !mapping {
							return f.Set([]string{"permissions"}, map[string]any{"copilot-requests": "write"})
						}
					}
					return f.Set([]string{"permissions", "copilot-requests"}, "write")
				}
				return nil
			})
			if applied {
				if addPermission {
					copilotRequestsFeatureCodemodLog.Print("Migrated features.copilot-requests to permissions.copilot-requests: write")
				} else {
					copilotRequestsFeatureCodemodLog.Print("Removed deprecated features.copilot-requests")
				}
			}
			return newContent, applied, err
		},
	}
}

func frontmatterHasCopilotRequestsPermission(frontmatter map[string]any) bool {
	permissionsAny, hasPermissions := frontmatter["permissions"]
	if !hasPermissions {
		return false
	}
	permissionsMap, ok := permissionsAny.(map[string]any)
	if !ok {
		return false
	}
	_, hasCopilotRequests := permissionsMap["copilot-requests"]
	return hasCopilotRequests
}

func canSafelyAddCopilotRequestsPermission(frontmatter map[string]any) bool {
	permissionsAny, hasPermissions := frontmatter["permissions"]
	if !hasPermissions {
		return true
	}
	if _, ok := permissionsAny.(map[string]any); ok {
		return true
	}

	strValue, ok := permissionsAny.(string)
	if !ok {
		return false
	}
	trimmed := strings.TrimSpace(strValue)
	return trimmed == "" || trimmed == "{}"
}
