package workflow

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/githubnext/gh-aw/pkg/console"
)

// writeSharedAction writes a shared action file, creating directories as needed and updating only if content differs
func (c *Compiler) writeSharedAction(markdownPath string, actionPath string, content string, actionName string) error {
	// Get git root to write action relative to it, fallback to markdown directory for tests
	gitRoot := filepath.Dir(markdownPath)
	for {
		if _, err := os.Stat(filepath.Join(gitRoot, ".git")); err == nil {
			break
		}
		parent := filepath.Dir(gitRoot)
		if parent == gitRoot {
			// Reached filesystem root without finding .git - use markdown directory as fallback
			gitRoot = filepath.Dir(markdownPath)
			break
		}
		gitRoot = parent
	}

	actionsDir := filepath.Join(gitRoot, ".github", "actions", actionPath)
	actionFile := filepath.Join(actionsDir, "action.yml")

	// Create directory if it doesn't exist
	if err := os.MkdirAll(actionsDir, 0755); err != nil {
		return fmt.Errorf("failed to create actions directory: %w", err)
	}

	// Write the action file if it doesn't exist or is different
	if _, err := os.Stat(actionFile); os.IsNotExist(err) {
		if c.verbose {
			fmt.Println(console.FormatInfoMessage(fmt.Sprintf("Creating shared %s action: %s", actionName, actionFile)))
		}
		if err := os.WriteFile(actionFile, []byte(content), 0644); err != nil {
			return fmt.Errorf("failed to write %s action: %w", actionName, err)
		}
		// Track the created file
		if c.fileTracker != nil {
			c.fileTracker.TrackCreated(actionFile)
		}
	} else {
		// Check if the content is different and update if needed
		existing, err := os.ReadFile(actionFile)
		if err != nil {
			return fmt.Errorf("failed to read existing action file: %w", err)
		}
		if string(existing) != content {
			if c.verbose {
				fmt.Println(console.FormatInfoMessage(fmt.Sprintf("Updating shared %s action: %s", actionName, actionFile)))
			}
			if err := os.WriteFile(actionFile, []byte(content), 0644); err != nil {
				return fmt.Errorf("failed to update %s action: %w", actionName, err)
			}
			// Track the updated file
			if c.fileTracker != nil {
				c.fileTracker.TrackCreated(actionFile)
			}
		}
	}

	return nil
}

// writeReactionAction writes the shared reaction action if ai-reaction is used
func (c *Compiler) writeReactionAction(markdownPath string) error {
	return c.writeSharedAction(markdownPath, "reaction", reactionActionTemplate, "reaction")
}

// writeComputeTextAction writes the shared compute-text action
func (c *Compiler) writeComputeTextAction(markdownPath string) error {
	return c.writeSharedAction(markdownPath, "compute-text", computeTextActionTemplate, "compute-text")
}

// writeCheckTeamMemberAction writes the shared check-team-member action
func (c *Compiler) writeCheckTeamMemberAction(markdownPath string) error {
	return c.writeSharedAction(markdownPath, "check-team-member", checkTeamMemberTemplate, "check-team-member")
}
