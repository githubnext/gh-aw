package workflow

import (
	"fmt"
	"strings"
)

// generateRepoMemoryPromptStep generates a separate step for repo memory instructions
// when repo-memory is enabled, informing the agent about git-based persistent storage capabilities
func (c *Compiler) generateRepoMemoryPromptStep(yaml *strings.Builder, config *RepoMemoryConfig) {
	if config == nil || len(config.Memories) == 0 {
		return
	}

	appendPromptStepWithHeredoc(yaml,
		"Append repo memory instructions to prompt",
		func(y *strings.Builder) {
			generateRepoMemoryPromptSection(y, config)
		})
}

// generateRepoMemoryPromptSection generates the repo memory notification section for prompts
// when repo-memory is enabled, informing the agent about git-based persistent storage capabilities
func generateRepoMemoryPromptSection(yaml *strings.Builder, config *RepoMemoryConfig) {
	if config == nil || len(config.Memories) == 0 {
		return
	}

	yaml.WriteString("          \n")
	yaml.WriteString("          ---\n")
	yaml.WriteString("          \n")

	// Check if there's only one memory with ID "default" to use singular form
	if len(config.Memories) == 1 && config.Memories[0].ID == "default" {
		yaml.WriteString("          ## Repo Memory Available\n")
		yaml.WriteString("          \n")
		memory := config.Memories[0]
		memoryDir := fmt.Sprintf("/tmp/gh-aw/repo-memory-%s/memory/%s/", memory.ID, memory.ID)
		
		if memory.Description != "" {
			yaml.WriteString(fmt.Sprintf("          You have access to a persistent repo memory folder at `%s` where you can read and write files that are stored in a git branch. %s\n", memoryDir, memory.Description))
		} else {
			yaml.WriteString(fmt.Sprintf("          You have access to a persistent repo memory folder at `%s` where you can read and write files that are stored in a git branch.\n", memoryDir))
		}
		yaml.WriteString("          \n")
		yaml.WriteString("          - **Read/Write Access**: You can freely read from and write to any files in this folder\n")
		yaml.WriteString(fmt.Sprintf("          - **Git Branch Storage**: Files are stored in the `%s` branch", memory.BranchName))
		if memory.TargetRepo != "" {
			yaml.WriteString(fmt.Sprintf(" of repository `%s`\n", memory.TargetRepo))
		} else {
			yaml.WriteString(" of the current repository\n")
		}
		yaml.WriteString("          - **Automatic Push**: Changes are automatically committed and pushed after the workflow completes\n")
		yaml.WriteString("          - **Merge Strategy**: In case of conflicts, your changes (current version) win\n")
		yaml.WriteString("          - **Persistence**: Files persist across workflow runs via git branch storage\n")
		
		// Add file constraints if specified
		if len(memory.FileGlob) > 0 || memory.MaxFileSize > 0 || memory.MaxFileCount > 0 {
			yaml.WriteString("          \n")
			yaml.WriteString("          **Constraints:**\n")
			if len(memory.FileGlob) > 0 {
				yaml.WriteString(fmt.Sprintf("          - **Allowed Files**: Only files matching patterns: %s\n", strings.Join(memory.FileGlob, ", ")))
			}
			if memory.MaxFileSize > 0 {
				yaml.WriteString(fmt.Sprintf("          - **Max File Size**: %d bytes (%.2f MB) per file\n", memory.MaxFileSize, float64(memory.MaxFileSize)/1048576.0))
			}
			if memory.MaxFileCount > 0 {
				yaml.WriteString(fmt.Sprintf("          - **Max File Count**: %d files per commit\n", memory.MaxFileCount))
			}
		}
		
		yaml.WriteString("          \n")
		yaml.WriteString("          Examples of what you can store:\n")
		yaml.WriteString(fmt.Sprintf("          - `%snotes.md` - general notes and observations\n", memoryDir))
		yaml.WriteString(fmt.Sprintf("          - `%sstate.json` - structured state data\n", memoryDir))
		yaml.WriteString(fmt.Sprintf("          - `%shistory/` - organized history files in subdirectories\n", memoryDir))
		yaml.WriteString("          \n")
		yaml.WriteString("          Feel free to create, read, update, and organize files in this folder as needed for your tasks.\n")
	} else {
		// Multiple memories or non-default single memory
		yaml.WriteString("          ## Repo Memory Locations Available\n")
		yaml.WriteString("          \n")
		yaml.WriteString("          You have access to persistent repo memory folders where you can read and write files that are stored in git branches:\n")
		yaml.WriteString("          \n")
		for _, memory := range config.Memories {
			memoryDir := fmt.Sprintf("/tmp/gh-aw/repo-memory-%s/memory/%s/", memory.ID, memory.ID)
			yaml.WriteString(fmt.Sprintf("          - **%s**: `%s`", memory.ID, memoryDir))
			if memory.Description != "" {
				yaml.WriteString(fmt.Sprintf(" - %s", memory.Description))
			}
			yaml.WriteString(fmt.Sprintf(" (branch: `%s`", memory.BranchName))
			if memory.TargetRepo != "" {
				yaml.WriteString(fmt.Sprintf(" in `%s`", memory.TargetRepo))
			}
			yaml.WriteString(")\n")
		}
		yaml.WriteString("          \n")
		yaml.WriteString("          - **Read/Write Access**: You can freely read from and write to any files in these folders\n")
		yaml.WriteString("          - **Git Branch Storage**: Each memory is stored in its own git branch\n")
		yaml.WriteString("          - **Automatic Push**: Changes are automatically committed and pushed after the workflow completes\n")
		yaml.WriteString("          - **Merge Strategy**: In case of conflicts, your changes (current version) win\n")
		yaml.WriteString("          - **Persistence**: Files persist across workflow runs via git branch storage\n")
		yaml.WriteString("          \n")
		yaml.WriteString("          Examples of what you can store:\n")
		for _, memory := range config.Memories {
			memoryDir := fmt.Sprintf("/tmp/gh-aw/repo-memory-%s/memory/%s", memory.ID, memory.ID)
			yaml.WriteString(fmt.Sprintf("          - `%s/notes.md` - general notes and observations\n", memoryDir))
			yaml.WriteString(fmt.Sprintf("          - `%s/state.json` - structured state data\n", memoryDir))
			yaml.WriteString(fmt.Sprintf("          - `%s/history/` - organized history files\n", memoryDir))
		}
		yaml.WriteString("          \n")
		yaml.WriteString("          Feel free to create, read, update, and organize files in these folders as needed for your tasks.\n")
	}
}
