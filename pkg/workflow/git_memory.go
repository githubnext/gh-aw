package workflow

import (
	"fmt"
	"strings"

	"github.com/githubnext/gh-aw/pkg/logger"
)

var gitMemoryLog = logger.New("workflow:git_memory")

// GitMemoryConfig holds configuration for git-memory functionality
type GitMemoryConfig struct {
	Branches []GitMemoryEntry `yaml:"branches,omitempty"` // git memory branch configurations
}

// GitMemoryEntry represents a single git-memory configuration
type GitMemoryEntry struct {
	ID          string `yaml:"id"`                    // branch identifier (required for array notation)
	Branch      string `yaml:"branch,omitempty"`      // branch name (must start with "memory/")
	Description string `yaml:"description,omitempty"` // optional description for this branch
}

// generateDefaultBranchName generates a default branch name for a given branch ID
func generateDefaultBranchName(branchID string) string {
	if branchID == "default" {
		return "memory/default"
	}
	return fmt.Sprintf("memory/%s", branchID)
}

// validateBranchName validates that a branch name starts with "memory/"
func validateBranchName(branchName string) error {
	if !strings.HasPrefix(branchName, "memory/") {
		return fmt.Errorf("git-memory branch name must start with 'memory/', got: %s", branchName)
	}
	return nil
}

// extractGitMemoryConfig extracts git-memory configuration from tools section
func (c *Compiler) extractGitMemoryConfig(toolsConfig *ToolsConfig) (*GitMemoryConfig, error) {
	// Check if git-memory tool is configured
	if toolsConfig == nil || toolsConfig.GitMemory == nil {
		return nil, nil
	}

	gitMemoryLog.Print("Extracting git-memory configuration from ToolsConfig")

	config := &GitMemoryConfig{}
	gitMemoryValue := toolsConfig.GitMemory.Raw

	// Handle nil value (simple enable with defaults) - same as true
	if gitMemoryValue == nil {
		config.Branches = []GitMemoryEntry{
			{
				ID:     "default",
				Branch: generateDefaultBranchName("default"),
			},
		}
		return config, nil
	}

	// Handle boolean value (simple enable/disable)
	if boolValue, ok := gitMemoryValue.(bool); ok {
		if boolValue {
			// Create a single default branch entry
			config.Branches = []GitMemoryEntry{
				{
					ID:     "default",
					Branch: generateDefaultBranchName("default"),
				},
			}
		}
		// If false, return empty config (empty array means disabled)
		return config, nil
	}

	// Handle array of branch configurations
	if branchArray, ok := gitMemoryValue.([]any); ok {
		gitMemoryLog.Printf("Processing git-memory array with %d entries", len(branchArray))
		config.Branches = make([]GitMemoryEntry, 0, len(branchArray))
		for _, item := range branchArray {
			if branchMap, ok := item.(map[string]any); ok {
				entry := GitMemoryEntry{}

				// ID is required for array notation
				if id, exists := branchMap["id"]; exists {
					if idStr, ok := id.(string); ok {
						entry.ID = idStr
					}
				}
				// Use "default" if no ID specified
				if entry.ID == "" {
					entry.ID = "default"
				}

				// Parse custom branch name
				if branch, exists := branchMap["branch"]; exists {
					if branchStr, ok := branch.(string); ok {
						entry.Branch = branchStr
					}
				}
				// Set default branch if not specified
				if entry.Branch == "" {
					entry.Branch = generateDefaultBranchName(entry.ID)
				}

				// Validate branch name starts with "memory/"
				if err := validateBranchName(entry.Branch); err != nil {
					return nil, err
				}

				// Parse description
				if description, exists := branchMap["description"]; exists {
					if descStr, ok := description.(string); ok {
						entry.Description = descStr
					}
				}

				config.Branches = append(config.Branches, entry)
			}
		}

		// Check for duplicate branch IDs
		if err := validateNoDuplicateGitMemoryIDs(config.Branches); err != nil {
			return nil, err
		}

		return config, nil
	}

	// Handle object configuration (single branch, backward compatible)
	if configMap, ok := gitMemoryValue.(map[string]any); ok {
		entry := GitMemoryEntry{
			ID:     "default",
			Branch: generateDefaultBranchName("default"),
		}

		// Parse custom branch name
		if branch, exists := configMap["branch"]; exists {
			if branchStr, ok := branch.(string); ok {
				entry.Branch = branchStr
			}
		}

		// Validate branch name starts with "memory/"
		if err := validateBranchName(entry.Branch); err != nil {
			return nil, err
		}

		// Parse description
		if description, exists := configMap["description"]; exists {
			if descStr, ok := description.(string); ok {
				entry.Description = descStr
			}
		}

		config.Branches = []GitMemoryEntry{entry}
		return config, nil
	}

	return nil, nil
}

// validateNoDuplicateGitMemoryIDs checks for duplicate branch IDs
func validateNoDuplicateGitMemoryIDs(branches []GitMemoryEntry) error {
	seen := make(map[string]bool)
	for _, branch := range branches {
		if seen[branch.ID] {
			return fmt.Errorf("duplicate git-memory branch ID: %s", branch.ID)
		}
		seen[branch.ID] = true
	}
	return nil
}

// generateGitMemoryCheckoutSteps generates checkout steps for git-memory branches
func generateGitMemoryCheckoutSteps(builder *strings.Builder, config *GitMemoryConfig) {
	if config == nil || len(config.Branches) == 0 {
		return
	}

	gitMemoryLog.Printf("Generating git-memory checkout steps for %d branches", len(config.Branches))

	builder.WriteString("      # Git memory branch checkout steps\n")

	for _, branch := range config.Branches {
		stepName := fmt.Sprintf("Checkout or create git-memory branch (%s)", branch.ID)
		builder.WriteString(fmt.Sprintf("      - name: %s\n", stepName))
		builder.WriteString("        run: |\n")
		builder.WriteString("          set -e\n")
		builder.WriteString(fmt.Sprintf("          BRANCH=\"%s\"\n", branch.Branch))
		builder.WriteString("          \n")
		builder.WriteString("          # Fetch the branch if it exists remotely\n")
		builder.WriteString("          if git ls-remote --exit-code --heads origin \"$BRANCH\" > /dev/null 2>&1; then\n")
		builder.WriteString("            echo \"Branch $BRANCH exists remotely, checking out...\"\n")
		builder.WriteString("            git fetch origin \"$BRANCH\"\n")
		builder.WriteString("            git checkout \"$BRANCH\"\n")
		builder.WriteString("            git pull origin \"$BRANCH\" || true\n")
		builder.WriteString("          else\n")
		builder.WriteString("            echo \"Branch $BRANCH does not exist, creating orphaned branch...\"\n")
		builder.WriteString("            git checkout --orphan \"$BRANCH\"\n")
		builder.WriteString("            git rm -rf . || true\n")
		builder.WriteString("            echo \"# Git Memory Branch: $BRANCH\" > README.md\n")
		builder.WriteString("            git add README.md\n")
		builder.WriteString("            git commit -m \"Initialize git-memory branch: $BRANCH\"\n")
		builder.WriteString("          fi\n")
		builder.WriteString("          \n")
		builder.WriteString("          # Return to original branch for workflow execution\n")
		builder.WriteString("          git checkout -\n")
	}
}

// generateGitMemoryCommitPushSteps generates commit and push steps for git-memory branches
// These steps run after the AI execution in the post-steps section
func generateGitMemoryCommitPushSteps(builder *strings.Builder, config *GitMemoryConfig) {
	if config == nil || len(config.Branches) == 0 {
		return
	}

	gitMemoryLog.Printf("Generating git-memory commit/push steps for %d branches", len(config.Branches))

	builder.WriteString("      # Git memory branch commit and push steps\n")

	for _, branch := range config.Branches {
		stepName := fmt.Sprintf("Commit and push git-memory branch (%s)", branch.ID)
		builder.WriteString(fmt.Sprintf("      - name: %s\n", stepName))
		builder.WriteString("        if: always()\n")
		builder.WriteString("        run: |\n")
		builder.WriteString("          set -e\n")
		builder.WriteString(fmt.Sprintf("          BRANCH=\"%s\"\n", branch.Branch))
		builder.WriteString("          \n")
		builder.WriteString("          # Checkout the memory branch\n")
		builder.WriteString("          git checkout \"$BRANCH\"\n")
		builder.WriteString("          \n")
		builder.WriteString("          # Stage all changes\n")
		builder.WriteString("          git add -A\n")
		builder.WriteString("          \n")
		builder.WriteString("          # Commit if there are changes\n")
		builder.WriteString("          if ! git diff --cached --quiet; then\n")
		builder.WriteString("            git commit -m \"Update git-memory: ${{ github.workflow }} run ${{ github.run_id }}\"\n")
		builder.WriteString("            \n")
		builder.WriteString("            # Push with fast-forward merge strategy (ours - current version wins)\n")
		builder.WriteString("            git push origin \"$BRANCH\" || {\n")
		builder.WriteString("              # If push fails due to conflicts, use our version\n")
		builder.WriteString("              git fetch origin \"$BRANCH\"\n")
		builder.WriteString("              git merge -X ours origin/\"$BRANCH\" -m \"Merge with ours strategy\"\n")
		builder.WriteString("              git push origin \"$BRANCH\"\n")
		builder.WriteString("            }\n")
		builder.WriteString("            echo \"Changes pushed to $BRANCH\"\n")
		builder.WriteString("          else\n")
		builder.WriteString("            echo \"No changes to commit for $BRANCH\"\n")
		builder.WriteString("          fi\n")
	}
}
