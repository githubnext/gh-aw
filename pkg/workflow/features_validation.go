// This file provides validation for feature flags.
//
// # Features Validation
//
// This file validates feature flag values to ensure they meet requirements
// before being used in workflow compilation. It ensures that:
//   - action-tag uses a full 40-character SHA or a version tag when specified
//   - disable-xpia-prompt is not combined with bash tool access (supply-chain attack vector)
//   - Other feature-specific constraints are met
//
// # Validation Functions
//
//   - validateFeatures() - Validates all feature flags in WorkflowData
//   - validateActionTag() - Validates action-tag is a full SHA or version tag
//   - validateDisableXPIAWithBash() - Rejects disable-xpia-prompt combined with bash tools
//   - isValidFullSHA() - Checks if a string is a valid 40-character SHA
//   - semverutil.IsActionVersionTag() - Checks if a string is a valid version tag (in pkg/semverutil)
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

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/gitutil"
	"github.com/github/gh-aw/pkg/semverutil"
)

var featuresValidationLog = newValidationLogger("features")

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

	// Validate that disable-xpia-prompt is not combined with bash tool access
	if isFeatureEnabled(constants.DisableXPIAPromptFeatureFlag, data) {
		featuresValidationLog.Print("Validating disable-xpia-prompt combination")
		if err := validateDisableXPIAWithBash(data); err != nil {
			featuresValidationLog.Printf("disable-xpia-prompt combination validation failed: %v", err)
			return err
		}
		featuresValidationLog.Print("disable-xpia-prompt combination validation passed")
	}

	featuresValidationLog.Print("Features validation completed successfully")
	return nil
}

// validateDisableXPIAWithBash rejects the dangerous combination of disable-xpia-prompt: true
// and bash tool access. When XPIA protection is disabled, the agent has no framing to
// distinguish adversarial instructions from legitimate ones. Combined with bash tool access,
// a prompt-injection payload can trivially escalate to arbitrary shell command execution,
// such as npm install of a malicious package with lifecycle scripts.
func validateDisableXPIAWithBash(data *WorkflowData) error {
	if data == nil || data.ParsedTools == nil {
		return nil
	}

	if data.ParsedTools.Bash == nil {
		return nil
	}

	return NewValidationError(
		"features.disable-xpia-prompt",
		"true",
		"disable-xpia-prompt cannot be combined with bash tool access. "+
			"Disabling XPIA protection removes the primary defense against prompt-injection attacks. "+
			"When combined with bash tool access, a prompt-injection payload can escalate to "+
			"arbitrary shell command execution (e.g. npm install of an attacker-controlled package).",
		"Either re-enable XPIA protection by removing the disable-xpia-prompt feature flag, "+
			"or remove the bash tool from the workflow's tool configuration.\n"+
			"If shell access is required, keep XPIA protection enabled (omit disable-xpia-prompt or set it to false).",
	)
}

// validateActionTag validates that action-tag is a full 40-character SHA or a version tag when specified
func validateActionTag(value any) error {
	// Allow empty or nil values
	if value == nil {
		return nil
	}

	// Convert to string
	strVal, ok := value.(string)
	if !ok {
		return NewValidationError(
			"features.action-tag",
			fmt.Sprintf("%T", value),
			fmt.Sprintf("action-tag must be a string, got %T", value),
			"Provide a string value for action-tag. Example:\nfeatures:\n  action-tag: \"v0\"",
		)
	}

	// Allow empty string (falls back to version)
	if strVal == "" {
		return nil
	}

	// Accept full 40-character commit SHA
	if isValidFullSHA(strVal) {
		return nil
	}

	// Accept version tags like "v0", "v1", "v1.0.0"
	if semverutil.IsActionVersionTag(strVal) {
		return nil
	}

	return NewValidationError(
		"features.action-tag",
		strVal,
		fmt.Sprintf("action-tag must be a full 40-character commit SHA or a version tag (e.g. v0, v1.0.0). Got: %q", strVal),
		"Use a version tag or a full commit SHA. Examples:\nfeatures:\n  action-tag: \"v0\"\n\nOr with a full SHA:\nfeatures:\n  action-tag: \"a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6q7r8s9t0\"",
	)
}

// isValidFullSHA checks if a string is a valid 40-character hexadecimal SHA
func isValidFullSHA(s string) bool {
	return gitutil.IsValidFullSHA(s)
}
