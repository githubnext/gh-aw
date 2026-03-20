package workflow

import (
	"fmt"
	"os"
	"strings"

	"github.com/github/gh-aw/pkg/logger"
)

var discussionLog = logger.New("workflow:create_discussion")

// CreateDiscussionsConfig holds configuration for creating GitHub discussions from agent output
type CreateDiscussionsConfig struct {
	BaseSafeOutputConfig  `yaml:",inline"`
	TitlePrefix           string   `yaml:"title-prefix,omitempty"`
	Category              string   `yaml:"category,omitempty"`                // Discussion category ID or name
	Labels                []string `yaml:"labels,omitempty"`                  // Labels to attach to discussions and match when closing older ones
	AllowedLabels         []string `yaml:"allowed-labels,omitempty"`          // Optional list of allowed labels. If omitted, any labels are allowed (including creating new ones).
	TargetRepoSlug        string   `yaml:"target-repo,omitempty"`             // Target repository in format "owner/repo" for cross-repository discussions
	AllowedRepos          []string `yaml:"allowed-repos,omitempty"`           // List of additional repositories that discussions can be created in
	CloseOlderDiscussions *string  `yaml:"close-older-discussions,omitempty"` // When true, close older discussions with same title prefix or labels as outdated
	CloseOlderKey         string   `yaml:"close-older-key,omitempty"`         // Optional explicit deduplication key for close-older matching. When set, uses gh-aw-close-key marker instead of workflow-id markers.
	RequiredCategory      string   `yaml:"required-category,omitempty"`       // Required category for matching when close-older-discussions is enabled
	Expires               int      `yaml:"expires,omitempty"`                 // Hours until the discussion expires and should be automatically closed
	FallbackToIssue       *bool    `yaml:"fallback-to-issue,omitempty"`       // When true (default), fallback to create-issue if discussion creation fails due to permissions.
	Footer                *string  `yaml:"footer,omitempty"`                  // Controls whether AI-generated footer is added. When false, visible footer is omitted but XML markers are kept.
}

// parseDiscussionsConfig handles create-discussion configuration
func (c *Compiler) parseDiscussionsConfig(outputMap map[string]any) *CreateDiscussionsConfig {
	// Check if the key exists
	if _, exists := outputMap["create-discussion"]; !exists {
		return nil
	}

	discussionLog.Print("Parsing create-discussion configuration")

	// Get the config data to check for special cases before unmarshaling
	configData, _ := outputMap["create-discussion"].(map[string]any)

	// Pre-process the expires field (convert to hours before unmarshaling)
	expiresDisabled := preprocessExpiresField(configData, discussionLog)

	// Pre-process templatable bool fields
	for _, field := range []string{"close-older-discussions", "footer"} {
		if err := preprocessBoolFieldAsString(configData, field, discussionLog); err != nil {
			discussionLog.Printf("Invalid %s value: %v", field, err)
			return nil
		}
	}

	// Pre-process templatable int fields
	if err := preprocessIntFieldAsString(configData, "max", discussionLog); err != nil {
		discussionLog.Printf("Invalid max value: %v", err)
		return nil
	}

	// Unmarshal into typed config struct
	var config CreateDiscussionsConfig
	if err := unmarshalConfig(outputMap, "create-discussion", &config, discussionLog); err != nil {
		discussionLog.Printf("Failed to unmarshal config: %v", err)
		// For backward compatibility, handle nil/empty config
		config = CreateDiscussionsConfig{}
	}

	// Set default max if not specified
	if config.Max == nil {
		config.Max = defaultIntStr(1)
	}

	// Set default expires to 7 days (168 hours) if not specified and not explicitly disabled
	if config.Expires == 0 && !expiresDisabled {
		config.Expires = 168 // 7 days = 168 hours
		discussionLog.Print("Using default expiration: 7 days (168 hours)")
	} else if expiresDisabled {
		config.Expires = 0
		discussionLog.Print("Expiration explicitly disabled")
	}

	// Set default fallback-to-issue to true if not specified
	if config.FallbackToIssue == nil {
		trueVal := true
		config.FallbackToIssue = &trueVal
		discussionLog.Print("Using default fallback-to-issue: true")
	}

	// Normalize and validate category naming convention
	config.Category = normalizeDiscussionCategory(config.Category, discussionLog, c.markdownPath)

	// Log configured values
	if config.TitlePrefix != "" {
		discussionLog.Printf("Title prefix configured: %q", config.TitlePrefix)
	}
	if config.Category != "" {
		discussionLog.Printf("Discussion category configured: %q", config.Category)
	}
	if len(config.Labels) > 0 {
		discussionLog.Printf("Labels configured: %v", config.Labels)
	}
	if len(config.AllowedLabels) > 0 {
		discussionLog.Printf("Allowed labels configured: %v", config.AllowedLabels)
	}
	if config.TargetRepoSlug != "" {
		discussionLog.Printf("Target repository configured: %s", config.TargetRepoSlug)
	}
	if len(config.AllowedRepos) > 0 {
		discussionLog.Printf("Allowed repos configured: %v", config.AllowedRepos)
	}
	if config.CloseOlderDiscussions != nil {
		discussionLog.Print("Close older discussions flag set")
		if config.RequiredCategory != "" {
			discussionLog.Printf("Required category for close older discussions: %q", config.RequiredCategory)
		}
	}
	if config.Expires > 0 {
		discussionLog.Printf("Discussion expiration configured: %d hours", config.Expires)
	}
	if config.FallbackToIssue != nil {
		discussionLog.Printf("Fallback to issue configured: %t", *config.FallbackToIssue)
	}

	return &config
}

// Returns normalized category (or original if it's a category ID)
func normalizeDiscussionCategory(category string, log *logger.Logger, markdownPath string) string {
	// Empty category is allowed (GitHub Discussions will use default)
	if category == "" {
		return category
	}

	// GitHub Discussion category IDs start with "DIC_" - don't normalize these
	if strings.HasPrefix(category, "DIC_") {
		return category
	}

	// List of known category naming issues and their corrections
	categoryCorrections := map[string]string{
		"Audits":   "audits",
		"General":  "general",
		"Reports":  "reports",
		"Research": "research",
	}

	// Check if category has uppercase letters and normalize
	normalizedCategory := strings.ToLower(category)
	if category != normalizedCategory {
		var message string
		// Check if we have a known correction
		if corrected, exists := categoryCorrections[category]; exists {
			message = fmt.Sprintf("Discussion category %q normalized to lowercase: %q", category, corrected)
			if log != nil {
				log.Printf("Normalized discussion category %q to lowercase: %q", category, corrected)
			}
		} else {
			message = fmt.Sprintf("Discussion category %q normalized to lowercase: %q", category, normalizedCategory)
			if log != nil {
				log.Printf("Normalized discussion category %q to lowercase: %q", category, normalizedCategory)
			}
		}

		// Print formatted info message to stderr
		fmt.Fprintln(os.Stderr, formatCompilerMessage(markdownPath, "info", message))
	}

	// Warn about singular forms of common categories
	singularToPlural := map[string]string{
		"audit":  "audits",
		"report": "reports",
	}

	if plural, isSingular := singularToPlural[normalizedCategory]; isSingular {
		if log != nil {
			log.Printf("⚠ Discussion category %q is singular; consider using plural form %q for consistency", normalizedCategory, plural)
		}
	}

	return normalizedCategory
}
