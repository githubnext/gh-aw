package workflow

import (
	"fmt"
)

// DiscussionConfig holds the configuration for discussion tracking
type DiscussionConfig struct {
	Enabled      bool
	CategoryName string
}

// extractDiscussionConfig extracts discussion configuration from frontmatter
func (c *Compiler) extractDiscussionConfig(frontmatter map[string]any) *DiscussionConfig {
	discussionValue, exists := frontmatter["discussion"]
	if !exists {
		// Default: enabled with empty category (let JavaScript resolve it)
		return &DiscussionConfig{
			Enabled:      true,
			CategoryName: "",
		}
	}

	config := &DiscussionConfig{}

	// Handle nil value (simple enable with defaults)
	if discussionValue == nil {
		config.Enabled = true
		config.CategoryName = ""
		return config
	}

	// Handle boolean value (explicit enable/disable)
	if boolValue, ok := discussionValue.(bool); ok {
		config.Enabled = boolValue
		if config.Enabled {
			config.CategoryName = ""
		}
		return config
	}

	// Handle string value (custom category name)
	if categoryName, ok := discussionValue.(string); ok {
		config.Enabled = true
		config.CategoryName = categoryName
		return config
	}

	// Invalid type, default to enabled with empty category (let JavaScript resolve it)
	config.Enabled = true
	config.CategoryName = ""
	return config
}

// addDiscussionEnvVars adds discussion environment variables to a safe output job
// This helper is used by all safe-output jobs to pass discussion tracking info
func (c *Compiler) addDiscussionEnvVars(steps *[]string, data *WorkflowData) {
	if data.DiscussionConfig != nil && data.DiscussionConfig.Enabled {
		*steps = append(*steps, "          GITHUB_AW_DISCUSSION_NUMBER: ${{ needs.activation.outputs.discussion-number }}\n")
		*steps = append(*steps, "          GITHUB_AW_DISCUSSION_URL: ${{ needs.activation.outputs.discussion-url }}\n")
	}
}

// addDiscussionCreationStep adds the discussion creation step to the activation job
func (c *Compiler) addDiscussionCreationStep(steps *[]string, outputs map[string]string, data *WorkflowData) {
	if data.DiscussionConfig != nil && data.DiscussionConfig.Enabled {
		*steps = append(*steps, "      - name: Create discussion to track workflow run\n")
		*steps = append(*steps, "        id: create-discussion\n")
		*steps = append(*steps, "        uses: actions/github-script@v8\n")
		*steps = append(*steps, "        env:\n")
		*steps = append(*steps, fmt.Sprintf("          GITHUB_AW_WORKFLOW_NAME: %q\n", data.Name))
		*steps = append(*steps, fmt.Sprintf("          GITHUB_AW_DISCUSSION_CATEGORY: %q\n", data.DiscussionConfig.CategoryName))
		*steps = append(*steps, "        with:\n")
		*steps = append(*steps, "          script: |\n")

		// Inline the JavaScript directly
		*steps = append(*steps, FormatJavaScriptForYAML(createActivationDiscussionScript)...)

		// Add discussion outputs
		outputs["discussion-id"] = "${{ steps.create-discussion.outputs.discussion-id }}"
		outputs["discussion-number"] = "${{ steps.create-discussion.outputs.discussion-number }}"
		outputs["discussion-url"] = "${{ steps.create-discussion.outputs.discussion-url }}"
	}
}
