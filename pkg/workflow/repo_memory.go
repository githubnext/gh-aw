package workflow

import (
	"fmt"
	"strings"

	"github.com/githubnext/gh-aw/pkg/logger"
)

var repoMemoryLog = logger.New("workflow:repo_memory")

// RepoMemoryConfig holds configuration for repo-memory functionality
type RepoMemoryConfig struct {
	Memories []RepoMemoryEntry `yaml:"memories,omitempty"` // repo-memory configurations
}

// RepoMemoryEntry represents a single repo-memory configuration
type RepoMemoryEntry struct {
	ID            string   `yaml:"id"`                       // memory identifier (required for array notation)
	TargetRepo    string   `yaml:"target-repo,omitempty"`    // target repository (default: current repo)
	BranchName    string   `yaml:"branch-name,omitempty"`    // branch name (default: memory/{memory-id})
	FileGlob      []string `yaml:"file-glob,omitempty"`      // file glob patterns for allowed files
	MaxFileSize   int      `yaml:"max-file-size,omitempty"`  // maximum size per file in bytes (default: 1MB)
	MaxFileCount  int      `yaml:"max-file-count,omitempty"` // maximum file count per commit (default: 100)
	Description   string   `yaml:"description,omitempty"`    // optional description for this memory
	CreateOrphan  bool     `yaml:"create-orphan,omitempty"`  // create orphaned branch if missing (default: true)
}

// RepoMemoryToolConfig represents the configuration for repo-memory in tools
type RepoMemoryToolConfig struct {
	// Can be boolean, object, or array - handled by this file
	Raw any `yaml:"-"`
}

// generateDefaultBranchName generates a default branch name for a given memory ID
func generateDefaultBranchName(memoryID string) string {
	if memoryID == "default" {
		return "memory/default"
	}
	return fmt.Sprintf("memory/%s", memoryID)
}

// extractRepoMemoryConfig extracts repo-memory configuration from tools section
func (c *Compiler) extractRepoMemoryConfig(toolsConfig *ToolsConfig) (*RepoMemoryConfig, error) {
	// Check if repo-memory tool is configured
	if toolsConfig == nil || toolsConfig.RepoMemory == nil {
		return nil, nil
	}

	repoMemoryLog.Print("Extracting repo-memory configuration from ToolsConfig")

	config := &RepoMemoryConfig{}
	repoMemoryValue := toolsConfig.RepoMemory.Raw

	// Handle nil value (simple enable with defaults) - same as true
	if repoMemoryValue == nil {
		config.Memories = []RepoMemoryEntry{
			{
				ID:           "default",
				BranchName:   generateDefaultBranchName("default"),
				MaxFileSize:  1048576, // 1MB
				MaxFileCount: 100,
				CreateOrphan: true,
			},
		}
		return config, nil
	}

	// Handle boolean value (simple enable/disable)
	if boolValue, ok := repoMemoryValue.(bool); ok {
		if boolValue {
			// Create a single default memory entry
			config.Memories = []RepoMemoryEntry{
				{
					ID:           "default",
					BranchName:   generateDefaultBranchName("default"),
					MaxFileSize:  1048576, // 1MB
					MaxFileCount: 100,
					CreateOrphan: true,
				},
			}
		}
		// If false, return empty config (empty array means disabled)
		return config, nil
	}

	// Handle array of memory configurations
	if memoryArray, ok := repoMemoryValue.([]any); ok {
		repoMemoryLog.Printf("Processing memory array with %d entries", len(memoryArray))
		config.Memories = make([]RepoMemoryEntry, 0, len(memoryArray))
		for _, item := range memoryArray {
			if memoryMap, ok := item.(map[string]any); ok {
				entry := RepoMemoryEntry{
					MaxFileSize:  1048576, // 1MB default
					MaxFileCount: 100,      // 100 files default
					CreateOrphan: true,     // create orphan by default
				}

				// ID is required for array notation
				if id, exists := memoryMap["id"]; exists {
					if idStr, ok := id.(string); ok {
						entry.ID = idStr
					}
				}
				// Use "default" if no ID specified
				if entry.ID == "" {
					entry.ID = "default"
				}

				// Parse target-repo
				if targetRepo, exists := memoryMap["target-repo"]; exists {
					if repoStr, ok := targetRepo.(string); ok {
						entry.TargetRepo = repoStr
					}
				}

				// Parse branch-name
				if branchName, exists := memoryMap["branch-name"]; exists {
					if branchStr, ok := branchName.(string); ok {
						entry.BranchName = branchStr
					}
				}
				// Set default branch name if not specified
				if entry.BranchName == "" {
					entry.BranchName = generateDefaultBranchName(entry.ID)
				}

				// Parse file-glob
				if fileGlob, exists := memoryMap["file-glob"]; exists {
					if globArray, ok := fileGlob.([]any); ok {
						entry.FileGlob = make([]string, 0, len(globArray))
						for _, item := range globArray {
							if str, ok := item.(string); ok {
								entry.FileGlob = append(entry.FileGlob, str)
							}
						}
					} else if globStr, ok := fileGlob.(string); ok {
						// Allow single string to be treated as array of one
						entry.FileGlob = []string{globStr}
					}
				}

				// Parse max-file-size
				if maxFileSize, exists := memoryMap["max-file-size"]; exists {
					if sizeInt, ok := maxFileSize.(int); ok {
						entry.MaxFileSize = sizeInt
					} else if sizeFloat, ok := maxFileSize.(float64); ok {
						entry.MaxFileSize = int(sizeFloat)
					} else if sizeUint64, ok := maxFileSize.(uint64); ok {
						entry.MaxFileSize = int(sizeUint64)
					}
				}

				// Parse max-file-count
				if maxFileCount, exists := memoryMap["max-file-count"]; exists {
					if countInt, ok := maxFileCount.(int); ok {
						entry.MaxFileCount = countInt
					} else if countFloat, ok := maxFileCount.(float64); ok {
						entry.MaxFileCount = int(countFloat)
					} else if countUint64, ok := maxFileCount.(uint64); ok {
						entry.MaxFileCount = int(countUint64)
					}
				}

				// Parse description
				if description, exists := memoryMap["description"]; exists {
					if descStr, ok := description.(string); ok {
						entry.Description = descStr
					}
				}

				// Parse create-orphan
				if createOrphan, exists := memoryMap["create-orphan"]; exists {
					if orphanBool, ok := createOrphan.(bool); ok {
						entry.CreateOrphan = orphanBool
					}
				}

				config.Memories = append(config.Memories, entry)
			}
		}

		// Check for duplicate memory IDs
		if err := validateNoDuplicateMemoryIDs(config.Memories); err != nil {
			return nil, err
		}

		return config, nil
	}

	// Handle object configuration (single memory, backward compatible)
	// Convert to array with single entry
	if configMap, ok := repoMemoryValue.(map[string]any); ok {
		entry := RepoMemoryEntry{
			ID:           "default",
			BranchName:   generateDefaultBranchName("default"),
			MaxFileSize:  1048576, // 1MB default
			MaxFileCount: 100,      // 100 files default
			CreateOrphan: true,     // create orphan by default
		}

		// Parse target-repo
		if targetRepo, exists := configMap["target-repo"]; exists {
			if repoStr, ok := targetRepo.(string); ok {
				entry.TargetRepo = repoStr
			}
		}

		// Parse branch-name
		if branchName, exists := configMap["branch-name"]; exists {
			if branchStr, ok := branchName.(string); ok {
				entry.BranchName = branchStr
			}
		}

		// Parse file-glob
		if fileGlob, exists := configMap["file-glob"]; exists {
			if globArray, ok := fileGlob.([]any); ok {
				entry.FileGlob = make([]string, 0, len(globArray))
				for _, item := range globArray {
					if str, ok := item.(string); ok {
						entry.FileGlob = append(entry.FileGlob, str)
					}
				}
			} else if globStr, ok := fileGlob.(string); ok {
				// Allow single string to be treated as array of one
				entry.FileGlob = []string{globStr}
			}
		}

		// Parse max-file-size
		if maxFileSize, exists := configMap["max-file-size"]; exists {
			if sizeInt, ok := maxFileSize.(int); ok {
				entry.MaxFileSize = sizeInt
			} else if sizeFloat, ok := maxFileSize.(float64); ok {
				entry.MaxFileSize = int(sizeFloat)
			} else if sizeUint64, ok := maxFileSize.(uint64); ok {
				entry.MaxFileSize = int(sizeUint64)
			}
		}

		// Parse max-file-count
		if maxFileCount, exists := configMap["max-file-count"]; exists {
			if countInt, ok := maxFileCount.(int); ok {
				entry.MaxFileCount = countInt
			} else if countFloat, ok := maxFileCount.(float64); ok {
				entry.MaxFileCount = int(countFloat)
			} else if countUint64, ok := maxFileCount.(uint64); ok {
				entry.MaxFileCount = int(countUint64)
			}
		}

		// Parse description
		if description, exists := configMap["description"]; exists {
			if descStr, ok := description.(string); ok {
				entry.Description = descStr
			}
		}

		// Parse create-orphan
		if createOrphan, exists := configMap["create-orphan"]; exists {
			if orphanBool, ok := createOrphan.(bool); ok {
				entry.CreateOrphan = orphanBool
			}
		}

		config.Memories = []RepoMemoryEntry{entry}
		return config, nil
	}

	return nil, nil
}

// validateNoDuplicateMemoryIDs checks for duplicate memory IDs and returns an error if found
func validateNoDuplicateMemoryIDs(memories []RepoMemoryEntry) error {
	seen := make(map[string]bool)
	for _, memory := range memories {
		if seen[memory.ID] {
			return fmt.Errorf("duplicate memory ID found: '%s'. Each memory must have a unique ID", memory.ID)
		}
		seen[memory.ID] = true
	}
	return nil
}

// generateRepoMemoryPushSteps generates steps to push changes back to the repo-memory branches
// This runs at the end of the workflow (always condition) to persist any changes made
func generateRepoMemoryPushSteps(builder *strings.Builder, data *WorkflowData) {
	if data.RepoMemoryConfig == nil || len(data.RepoMemoryConfig.Memories) == 0 {
		return
	}

	repoMemoryLog.Printf("Generating repo-memory push steps for %d memories", len(data.RepoMemoryConfig.Memories))

	builder.WriteString("      # Push repo memory changes back to git branches\n")

	for _, memory := range data.RepoMemoryConfig.Memories {
		// Determine the target repository
		targetRepo := memory.TargetRepo
		if targetRepo == "" {
			targetRepo = "${{ github.repository }}"
		}

		// Determine the memory directory
		memoryDir := fmt.Sprintf("/tmp/gh-aw/repo-memory-%s", memory.ID)

		// Step: Push changes to repo-memory branch
		builder.WriteString(fmt.Sprintf("      - name: Push repo-memory changes (%s)\n", memory.ID))
		builder.WriteString("        if: always()\n")
		builder.WriteString("        env:\n")
		builder.WriteString("          GH_TOKEN: ${{ github.token }}\n")
		builder.WriteString("        run: |\n")
		builder.WriteString("          set -e\n")
		builder.WriteString(fmt.Sprintf("          cd \"%s\" || exit 0\n", memoryDir))
		builder.WriteString("          \n")
		builder.WriteString("          # Check if we have any changes to commit\n")
		builder.WriteString("          if [ -n \"$(git status --porcelain)\" ]; then\n")
		builder.WriteString("            echo \"Changes detected in repo memory, committing and pushing...\"\n")
		builder.WriteString("            \n")
		
		// Add file validation if constraints are specified
		if len(memory.FileGlob) > 0 || memory.MaxFileSize > 0 || memory.MaxFileCount > 0 {
			builder.WriteString("            # Validate files before committing\n")
			
			if memory.MaxFileSize > 0 {
				builder.WriteString(fmt.Sprintf("            # Check file sizes (max: %d bytes)\n", memory.MaxFileSize))
				builder.WriteString(fmt.Sprintf("            if find . -type f -size +%dc | grep -q .; then\n", memory.MaxFileSize))
				builder.WriteString("              echo \"Error: Files exceed maximum size limit\"\n")
				builder.WriteString(fmt.Sprintf("              find . -type f -size +%dc -exec ls -lh {} \\;\n", memory.MaxFileSize))
				builder.WriteString("              exit 1\n")
				builder.WriteString("            fi\n")
				builder.WriteString("            \n")
			}
			
			if memory.MaxFileCount > 0 {
				builder.WriteString(fmt.Sprintf("            # Check file count (max: %d files)\n", memory.MaxFileCount))
				builder.WriteString("            FILE_COUNT=$(git status --porcelain | wc -l)\n")
				builder.WriteString(fmt.Sprintf("            if [ \"$FILE_COUNT\" -gt %d ]; then\n", memory.MaxFileCount))
				builder.WriteString(fmt.Sprintf("              echo \"Error: Too many files to commit ($FILE_COUNT > %d)\"\n", memory.MaxFileCount))
				builder.WriteString("              exit 1\n")
				builder.WriteString("            fi\n")
				builder.WriteString("            \n")
			}
		}
		
		builder.WriteString("            # Add all changes\n")
		builder.WriteString("            git add -A\n")
		builder.WriteString("            \n")
		builder.WriteString("            # Commit changes\n")
		builder.WriteString("            git commit -m \"Update memory from workflow run ${{ github.run_id }}\"\n")
		builder.WriteString("            \n")
		builder.WriteString("            # Pull with ours merge strategy (our changes win in conflicts)\n")
		builder.WriteString("            set +e\n")
		builder.WriteString(fmt.Sprintf("            git pull --no-rebase -s recursive -X ours \"https://x-access-token:${GH_TOKEN}@github.com/%s.git\" \"%s\" 2>&1\n",
			targetRepo, memory.BranchName))
		builder.WriteString("            PULL_EXIT_CODE=$?\n")
		builder.WriteString("            set -e\n")
		builder.WriteString("            \n")
		builder.WriteString("            # Push changes (force push if needed due to conflict resolution)\n")
		builder.WriteString(fmt.Sprintf("            git push \"https://x-access-token:${GH_TOKEN}@github.com/%s.git\" \"HEAD:%s\"\n",
			targetRepo, memory.BranchName))
		builder.WriteString("            \n")
		builder.WriteString("            echo \"Successfully pushed changes to repo memory\"\n")
		builder.WriteString("          else\n")
		builder.WriteString("            echo \"No changes in repo memory, skipping push\"\n")
		builder.WriteString("          fi\n")
	}
}

// generateRepoMemorySteps generates git steps for the repo-memory configuration
func generateRepoMemorySteps(builder *strings.Builder, data *WorkflowData) {
	if data.RepoMemoryConfig == nil || len(data.RepoMemoryConfig.Memories) == 0 {
		return
	}

	repoMemoryLog.Printf("Generating repo-memory steps for %d memories", len(data.RepoMemoryConfig.Memories))

	builder.WriteString("      # Repo memory git-based storage configuration from frontmatter processed below\n")

	for _, memory := range data.RepoMemoryConfig.Memories {
		// Determine the target repository
		targetRepo := memory.TargetRepo
		if targetRepo == "" {
			targetRepo = "${{ github.repository }}"
		}

		// Determine the memory directory
		memoryDir := fmt.Sprintf("/tmp/gh-aw/repo-memory-%s", memory.ID)

		// Step 1: Clone the repo-memory branch
		builder.WriteString(fmt.Sprintf("      - name: Clone repo-memory branch (%s)\n", memory.ID))
		builder.WriteString("        env:\n")
		builder.WriteString("          GH_TOKEN: ${{ github.token }}\n")
		builder.WriteString(fmt.Sprintf("          BRANCH_NAME: %s\n", memory.BranchName))
		builder.WriteString("        run: |\n")
		builder.WriteString("          set +e  # Don't fail if branch doesn't exist\n")
		builder.WriteString(fmt.Sprintf("          git clone --depth 1 --single-branch --branch \"%s\" \"https://x-access-token:${GH_TOKEN}@github.com/%s.git\" \"%s\" 2>/dev/null\n",
			memory.BranchName, targetRepo, memoryDir))
		builder.WriteString("          CLONE_EXIT_CODE=$?\n")
		builder.WriteString("          set -e\n")
		builder.WriteString("          \n")
		builder.WriteString("          if [ $CLONE_EXIT_CODE -ne 0 ]; then\n")
		
		if memory.CreateOrphan {
			builder.WriteString(fmt.Sprintf("            echo \"Branch %s does not exist, creating orphan branch\"\n", memory.BranchName))
			builder.WriteString(fmt.Sprintf("            mkdir -p \"%s\"\n", memoryDir))
			builder.WriteString(fmt.Sprintf("            cd \"%s\"\n", memoryDir))
			builder.WriteString("            git init\n")
			builder.WriteString("            git checkout --orphan \"$BRANCH_NAME\"\n")
			builder.WriteString("            git config user.name \"github-actions[bot]\"\n")
			builder.WriteString("            git config user.email \"github-actions[bot]@users.noreply.github.com\"\n")
			builder.WriteString(fmt.Sprintf("            git remote add origin \"https://x-access-token:${GH_TOKEN}@github.com/%s.git\"\n", targetRepo))
		} else {
			builder.WriteString(fmt.Sprintf("            echo \"Branch %s does not exist and create-orphan is false, skipping\"\n", memory.BranchName))
			builder.WriteString(fmt.Sprintf("            mkdir -p \"%s\"\n", memoryDir))
		}
		
		builder.WriteString("          else\n")
		builder.WriteString(fmt.Sprintf("            echo \"Successfully cloned %s branch\"\n", memory.BranchName))
		builder.WriteString(fmt.Sprintf("            cd \"%s\"\n", memoryDir))
		builder.WriteString("            git config user.name \"github-actions[bot]\"\n")
		builder.WriteString("            git config user.email \"github-actions[bot]@users.noreply.github.com\"\n")
		builder.WriteString("          fi\n")
		builder.WriteString("          \n")
		
		// Create the memory subdirectory
		builder.WriteString(fmt.Sprintf("          mkdir -p \"%s/memory/%s\"\n", memoryDir, memory.ID))
		builder.WriteString(fmt.Sprintf("          echo \"Repo memory directory ready at %s/memory/%s\"\n", memoryDir, memory.ID))
	}
}
