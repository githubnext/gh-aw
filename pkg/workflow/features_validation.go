// Package workflow provides validation for feature flags.
//
// # Features Validation
//
// This file validates feature flag values to ensure they meet requirements
// before being used in workflow compilation. It ensures that:
//   - action-tag uses full 40-character SHA when specified
//   - Other feature-specific constraints are met
//
// # Validation Functions
//
//   - validateFeatures() - Validates all feature flags in WorkflowData
//   - validateActionTag() - Validates action-tag is a full SHA
//   - isValidFullSHA() - Checks if a string is a valid 40-character SHA
//
// # When to Add Validation Here
//
// Add validation to this file when:
//   - Adding new feature flags that require specific value formats
//   - Feature flags need cross-validation with other workflow settings
//   - Feature flag values need format or constraint checking
package workflow

import (
	"fmt"
	"regexp"

	"github.com/githubnext/gh-aw/pkg/logger"
)

var featuresValidationLog = logger.New("workflow:features_validation")

var shaRegex = regexp.MustCompile("^[0-9a-f]{40}$")

// validateFeatures validates all feature flags in the workflow data
func validateFeatures(data *WorkflowData) error {
	if data == nil || data.Features == nil {
		featuresValidationLog.Print("No features to validate")
		return nil
	}

	featuresValidationLog.Printf("Validating features: count=%d", len(data.Features))

	// Validate action-tag if present
	if actionTagVal, exists := data.Features["action-tag"]; exists {
		featuresValidationLog.Print("Validating action-tag feature")
		if err := validateActionTag(actionTagVal); err != nil {
			featuresValidationLog.Printf("Action-tag validation failed: %v", err)
			return err
		}
		featuresValidationLog.Print("Action-tag validation passed")
	}

	featuresValidationLog.Print("Features validation completed successfully")
	return nil
}

// validateActionTag validates that action-tag is a full 40-character SHA when specified
func validateActionTag(value any) error {
	// Allow empty or nil values
	if value == nil {
		return nil
	}

	// Convert to string
	strVal, ok := value.(string)
	if !ok {
		return fmt.Errorf("💡 The action-tag feature needs to be a string (got %T).\n\nExample:\n  features:\n    action-tag: \"abc123...\" # full 40-char SHA", value)
	}

	// Allow empty string (falls back to version)
	if strVal == "" {
		return nil
	}

	// Validate it's a full SHA (40 hex characters)
	if !isValidFullSHA(strVal) {
		return fmt.Errorf("🔒 The action-tag must be a full 40-character commit SHA (got %q with length %d).\n\nWhy? Short SHAs can be ambiguous and pose security risks.\n\nTo get the full SHA:\n  git rev-parse <ref>\n\nExample:\n  features:\n    action-tag: \"1234567890abcdef1234567890abcdef12345678\"\n\nLearn more: https://githubnext.github.io/gh-aw/reference/features/#action-tag", strVal, len(strVal))
	}

	return nil
}

// isValidFullSHA checks if a string is a valid 40-character hexadecimal SHA
func isValidFullSHA(s string) bool {
	if len(s) != 40 {
		return false
	}
	return shaRegex.MatchString(s)
}
