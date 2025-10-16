package workflow

import (
	"regexp"
	"strings"
)

// normalizeBranchName converts a branch name to be valid for git
// Valid characters: alphanumeric, -, _, /, .
// Max length: 128 characters
// The function removes/replaces invalid characters and truncates to max length
func normalizeBranchName(branchName string) string {
	// Replace any sequence of invalid characters with a single dash
	// Valid characters are: a-z, A-Z, 0-9, -, _, /, .
	invalidCharsRegex := regexp.MustCompile(`[^a-zA-Z0-9\-_/.]+`)
	normalized := invalidCharsRegex.ReplaceAllString(branchName, "-")

	// Remove leading and trailing dashes
	normalized = strings.Trim(normalized, "-")

	// Truncate to max 128 characters
	if len(normalized) > 128 {
		normalized = normalized[:128]
	}

	// Ensure it doesn't end with a dash after truncation
	normalized = strings.TrimRight(normalized, "-")

	return normalized
}

// GenerateBranchNormalizationScript generates a bash script that normalizes the GITHUB_AW_ASSETS_BRANCH env var
// This script is added as a step in the agent job and the upload_assets job
func GenerateBranchNormalizationScript() string {
	return `# Normalize GITHUB_AW_ASSETS_BRANCH to be a valid git branch name
# Valid characters: alphanumeric, -, _, /, .
# Max length: 128 characters
if [ -n "${GITHUB_AW_ASSETS_BRANCH}" ]; then
  # Remove invalid characters (replace with dash)
  NORMALIZED=$(echo "${GITHUB_AW_ASSETS_BRANCH}" | sed 's/[^a-zA-Z0-9\/_.,-]/-/g')
  # Remove leading and trailing dashes
  NORMALIZED=$(echo "${NORMALIZED}" | sed 's/^-*//;s/-*$//')
  # Truncate to 128 characters
  NORMALIZED=$(echo "${NORMALIZED}" | cut -c1-128)
  # Remove trailing dashes after truncation
  NORMALIZED=$(echo "${NORMALIZED}" | sed 's/-*$//')
  # Export the normalized value
  export GITHUB_AW_ASSETS_BRANCH="${NORMALIZED}"
  echo "GITHUB_AW_ASSETS_BRANCH=${NORMALIZED}" >> $GITHUB_ENV
  echo "✓ Normalized GITHUB_AW_ASSETS_BRANCH: ${NORMALIZED}"
else
  echo "⚠ GITHUB_AW_ASSETS_BRANCH not set, skipping normalization"
fi`
}
