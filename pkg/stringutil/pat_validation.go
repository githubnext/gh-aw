package stringutil

import (
	"errors"
	"strings"

	"github.com/github/gh-aw/pkg/logger"
)

var patLog = logger.New("stringutil:pat_validation")

// PATType represents the type of a GitHub Personal Access Token
type PATType string

const (
	// PATTypeFineGrained is a fine-grained personal access token (starts with "github_pat_")
	PATTypeFineGrained PATType = "fine-grained"
	// PATTypeClassic is a classic personal access token (starts with "ghp_")
	PATTypeClassic PATType = "classic"
	// PATTypeOAuth is an OAuth token (starts with "gho_")
	PATTypeOAuth PATType = "oauth"
	// PATTypeUnknown is an unknown token type
	PATTypeUnknown PATType = "unknown"
)

// String returns the string representation of a PATType
func (p PATType) String() string {
	return string(p)
}

// IsFineGrained returns true if the token is a fine-grained PAT
func (p PATType) IsFineGrained() bool {
	return p == PATTypeFineGrained
}

// IsValid returns true if the token type is known (not unknown)
func (p PATType) IsValid() bool {
	return p != PATTypeUnknown
}

// ClassifyPAT determines the type of a GitHub Personal Access Token based on its prefix.
//
// Token prefixes:
//   - "github_pat_" = Fine-grained PAT (required for Copilot)
//   - "ghp_" = Classic PAT (not supported for Copilot)
//   - "gho_" = OAuth token (not a PAT at all)
//
// Parameters:
//   - token: The token string to classify
//
// Returns:
//   - PATType: The type of the token
func ClassifyPAT(token string) PATType {
	var patType PATType
	switch {
	case strings.HasPrefix(token, "github_pat_"):
		patType = PATTypeFineGrained
	case strings.HasPrefix(token, "ghp_"):
		patType = PATTypeClassic
	case strings.HasPrefix(token, "gho_"):
		patType = PATTypeOAuth
	default:
		patType = PATTypeUnknown
	}
	patLog.Printf("Classified PAT type: %s", patType)
	return patType
}

// IsFineGrainedPAT returns true if the token is a fine-grained personal access token
func IsFineGrainedPAT(token string) bool {
	return strings.HasPrefix(token, "github_pat_")
}

// IsClassicPAT returns true if the token is a classic personal access token
func IsClassicPAT(token string) bool {
	return strings.HasPrefix(token, "ghp_")
}

// IsOAuthToken returns true if the token is an OAuth token (not a PAT)
func IsOAuthToken(token string) bool {
	return strings.HasPrefix(token, "gho_")
}

// ValidateCopilotPAT validates that a token is a valid fine-grained PAT for Copilot.
// Returns an error if the token is not a fine-grained PAT with a descriptive error message.
//
// Parameters:
//   - token: The token string to validate
//
// Returns:
//   - error: An error with a descriptive message if the token is not valid, nil otherwise
func ValidateCopilotPAT(token string) error {
	patType := ClassifyPAT(token)
	patLog.Printf("Validating Copilot PAT: type=%s", patType)

	switch patType {
	case PATTypeFineGrained:
		return nil
	case PATTypeClassic:
		return errors.New("classic personal access tokens (ghp_...) are not supported for Copilot. Please create a fine-grained PAT at https://github.com/settings/personal-access-tokens/new")
	case PATTypeOAuth:
		return errors.New("OAuth tokens (gho_...) are not supported for Copilot. Please create a fine-grained PAT at https://github.com/settings/personal-access-tokens/new")
	default:
		return errors.New("unrecognized token format. Please create a fine-grained PAT (starting with 'github_pat_') at https://github.com/settings/personal-access-tokens/new")
	}
}

// GetPATTypeDescription returns a human-readable description of the PAT type
func GetPATTypeDescription(token string) string {
	patType := ClassifyPAT(token)

	switch patType {
	case PATTypeFineGrained:
		return "fine-grained personal access token"
	case PATTypeClassic:
		return "classic personal access token"
	case PATTypeOAuth:
		return "OAuth token"
	default:
		return "unknown token type"
	}
}
