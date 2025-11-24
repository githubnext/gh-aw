package workflow

import (
	"fmt"
	"strings"
)

// generateGitMemoryPromptStep generates a separate step for git memory instructions
// when git-memory is enabled, informing the agent about persistent git storage capabilities
func (c *Compiler) generateGitMemoryPromptStep(yaml *strings.Builder, config *GitMemoryConfig) {
	if config == nil || len(config.Branches) == 0 {
		return
	}

	appendPromptStepWithHeredoc(yaml,
		"Append git memory instructions to prompt",
		func(y *strings.Builder) {
			generateGitMemoryPromptSection(y, config)
		})
}

// generateGitMemoryPromptSection generates the git memory notification section for prompts
// when git-memory is enabled, informing the agent about persistent git storage capabilities
func generateGitMemoryPromptSection(yaml *strings.Builder, config *GitMemoryConfig) {
	if config == nil || len(config.Branches) == 0 {
		return
	}

	yaml.WriteString("          \n")
	yaml.WriteString("          ---\n")
	yaml.WriteString("          \n")

	// Check if there's only one branch with ID "default" to use singular form
	if len(config.Branches) == 1 && config.Branches[0].ID == "default" {
		yaml.WriteString("          ## Git Memory Branch Available\n")
		yaml.WriteString("          \n")
		branch := config.Branches[0]
		if branch.Description != "" {
			yaml.WriteString(fmt.Sprintf("          You have access to a persistent git memory branch `%s` where you can read and write files to create memories and store information. %s\n", branch.Branch, branch.Description))
		} else {
			yaml.WriteString(fmt.Sprintf("          You have access to a persistent git memory branch `%s` where you can read and write files to create memories and store information.\n", branch.Branch))
		}
		yaml.WriteString("          \n")
		yaml.WriteString("          - **Git Storage**: Files are stored in a git branch and persist across workflow runs\n")
		yaml.WriteString("          - **Read/Write Access**: You can freely read from and write to any files in the working directory\n")
		yaml.WriteString("          - **Orphaned Branch**: This is an orphaned git branch separate from the main codebase\n")
		yaml.WriteString("          - **Fast-Forward Merge**: Changes use fast-forward merge with 'ours' strategy (current version wins on conflicts)\n")
		yaml.WriteString("          - **Automatic Commit**: All changes are automatically committed and pushed after workflow execution\n")
		yaml.WriteString("          \n")
		yaml.WriteString("          Examples of what you can store:\n")
		yaml.WriteString("          - `notes.txt` - general notes and observations\n")
		yaml.WriteString("          - `preferences.json` - user preferences and settings\n")
		yaml.WriteString("          - `history.log` - activity history and logs\n")
		yaml.WriteString("          - `state/` - organized state files in subdirectories\n")
		yaml.WriteString("          \n")
		yaml.WriteString("          Feel free to create, read, update, and organize files as needed for your tasks.\n")
	} else {
		// Multiple branches or non-default single branch
		yaml.WriteString("          ## Git Memory Branches Available\n")
		yaml.WriteString("          \n")
		yaml.WriteString("          You have access to persistent git memory branches where you can read and write files to create memories and store information:\n")
		yaml.WriteString("          \n")
		for _, branch := range config.Branches {
			if branch.Description != "" {
				yaml.WriteString(fmt.Sprintf("          - **%s**: `%s` - %s\n", branch.ID, branch.Branch, branch.Description))
			} else {
				yaml.WriteString(fmt.Sprintf("          - **%s**: `%s`\n", branch.ID, branch.Branch))
			}
		}
		yaml.WriteString("          \n")
		yaml.WriteString("          - **Git Storage**: Files are stored in git branches and persist across workflow runs\n")
		yaml.WriteString("          - **Read/Write Access**: You can freely read from and write to any files in the working directory\n")
		yaml.WriteString("          - **Orphaned Branches**: These are orphaned git branches separate from the main codebase\n")
		yaml.WriteString("          - **Fast-Forward Merge**: Changes use fast-forward merge with 'ours' strategy (current version wins on conflicts)\n")
		yaml.WriteString("          - **Automatic Commit**: All changes are automatically committed and pushed after workflow execution\n")
		yaml.WriteString("          \n")
		yaml.WriteString("          Examples of what you can store:\n")
		yaml.WriteString("          - `notes.txt` - general notes and observations\n")
		yaml.WriteString("          - `preferences.json` - user preferences and settings\n")
		yaml.WriteString("          - `state/` - organized state files in subdirectories\n")
		yaml.WriteString("          \n")
		yaml.WriteString("          Feel free to create, read, update, and organize files as needed for your tasks.\n")
	}
}
