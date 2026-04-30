package cli

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/logger"
	"github.com/spf13/cobra"
)

var skillsLog = logger.New("cli:skills")

// AgentType represents a supported coding agent target
type AgentType string

const (
	AgentClaudeCode AgentType = "claude-code"
	AgentCopilot    AgentType = "copilot"
	AgentCodex      AgentType = "codex"
)

// supportedAgents is the list of valid --agent flag values
var supportedAgents = []AgentType{AgentClaudeCode, AgentCopilot, AgentCodex}

//go:embed user-skills
var userSkillsFS embed.FS

// userSkillsRootDir is the root directory name inside the embedded FS
const userSkillsRootDir = "user-skills"

// AgentSkillsDir returns the target directory where skills should be installed for
// the given agent. Returns an error when the agent is unsupported or the home
// directory cannot be resolved.
func AgentSkillsDir(agent AgentType) (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	switch agent {
	case AgentClaudeCode:
		// Claude Code loads sub-agents from ~/.claude/agents/
		return filepath.Join(homeDir, ".claude", "agents"), nil
	case AgentCopilot:
		// Copilot CLI reads custom agent files from ~/.config/gh-copilot/agents/
		return filepath.Join(homeDir, ".config", "gh-copilot", "agents"), nil
	case AgentCodex:
		// Codex reads custom agent files from ~/.codex/agents/
		return filepath.Join(homeDir, ".codex", "agents"), nil
	default:
		return "", fmt.Errorf("unsupported agent %q; must be one of: %s", agent, formatAgentList(supportedAgents))
	}
}

// formatAgentList formats a slice of agent types as a comma-separated string.
func formatAgentList(agents []AgentType) string {
	names := make([]string, len(agents))
	for i, a := range agents {
		names[i] = string(a)
	}
	return strings.Join(names, ", ")
}

// listUserSkills returns the skill names available in the embedded user-skills FS.
func listUserSkills() ([]string, error) {
	entries, err := fs.ReadDir(userSkillsFS, userSkillsRootDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read embedded user-skills: %w", err)
	}

	var names []string
	for _, entry := range entries {
		if entry.IsDir() {
			names = append(names, entry.Name())
		}
	}
	sort.Strings(names)
	return names, nil
}

// skillDescription reads the description from a skill's SKILL.md frontmatter.
func skillDescription(skillName string) string {
	skillPath := filepath.Join(userSkillsRootDir, skillName, "SKILL.md")
	data, err := userSkillsFS.ReadFile(skillPath)
	if err != nil {
		return ""
	}

	// Extract description from YAML frontmatter (between --- delimiters)
	content := string(data)
	const sep = "---"
	_, afterFirst, found := strings.Cut(content, sep)
	if !found {
		return ""
	}
	frontmatter, _, found := strings.Cut(afterFirst, sep)
	if !found {
		return ""
	}

	for line := range strings.SplitSeq(frontmatter, "\n") {
		line = strings.TrimSpace(line)
		if desc, ok := strings.CutPrefix(line, "description:"); ok {
			return strings.TrimSpace(desc)
		}
	}
	return ""
}

// installSkill copies a single skill's SKILL.md to the target directory.
func installSkill(skillName string, targetDir string, force bool) error {
	srcPath := filepath.Join(userSkillsRootDir, skillName, "SKILL.md")
	data, err := userSkillsFS.ReadFile(srcPath)
	if err != nil {
		return fmt.Errorf("failed to read embedded skill %q: %w", skillName, err)
	}

	destPath := filepath.Join(targetDir, skillName+".md")

	if _, statErr := os.Stat(destPath); statErr == nil && !force {
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(
			fmt.Sprintf("Skill %q already exists at %s (use --force to overwrite)", skillName, destPath)))
		return nil
	}

	if err := os.WriteFile(destPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write skill %q to %s: %w", skillName, destPath, err)
	}

	fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(
		fmt.Sprintf("Installed skill %q → %s", skillName, destPath)))
	return nil
}

// NewSkillsCommand creates the skills command with install, list, and path subcommands.
func NewSkillsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "skills",
		Short: "Manage gh-aw skills for coding agents (Claude Code, Copilot, Codex)",
		Long: `Manage official gh-aw skills that wrap the gh aw CLI for use inside coding agents.

Skills are Markdown files that teach coding agents (Claude Code, GitHub Copilot CLI,
Codex) how to drive gh aw commands conversationally. Installing a skill gives your
agent native, first-party guidance for common gh-aw verbs.

Available skills:
  discover-workflows  Browse the workflow catalog and suggest installs
  install-workflow    Install a workflow, wire secrets, verify compilation
  compile-workflows   Compile .md → .lock.yml, surface and fix errors
  audit-workflows     Audit a completed run and summarise findings
  debug-workflow-run  Fetch logs for a failed run and diagnose the cause

Examples:
  gh aw skills list                          # List available skills
  gh aw skills install                       # Install all skills (default: claude-code)
  gh aw skills install --agent claude-code   # Install for Claude Code
  gh aw skills path                          # Show the install directory
  gh aw skills path --agent copilot          # Show Copilot's install directory`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	cmd.AddCommand(newSkillsListSubcommand())
	cmd.AddCommand(newSkillsInstallSubcommand())
	cmd.AddCommand(newSkillsPathSubcommand())

	return cmd
}

// newSkillsListSubcommand creates the `skills list` subcommand.
func newSkillsListSubcommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List available gh-aw skills",
		Long: `List all gh-aw skills available for installation.

Each skill is a conversational adapter that teaches a coding agent (Claude Code,
GitHub Copilot CLI, Codex) how to drive a specific gh aw CLI verb.

Examples:
  gh aw skills list   # Show all available skills`,
		RunE: func(cmd *cobra.Command, args []string) error {
			skills, err := listUserSkills()
			if err != nil {
				return err
			}

			fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Available gh-aw skills:"))
			fmt.Fprintln(os.Stderr, "")

			for _, name := range skills {
				desc := skillDescription(name)
				if desc == "" {
					fmt.Fprintf(os.Stderr, "  %-26s\n", name)
				} else {
					// Truncate long descriptions for terminal display
					if len(desc) > 80 {
						desc = desc[:77] + "..."
					}
					fmt.Fprintf(os.Stderr, "  %-26s %s\n", name, desc)
				}
			}

			fmt.Fprintln(os.Stderr, "")
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Install all skills with: gh aw skills install"))
			return nil
		},
	}
}

// newSkillsInstallSubcommand creates the `skills install` subcommand.
func newSkillsInstallSubcommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "install [skill]...",
		Short: "Install gh-aw skills into the coding agent's skill directory",
		Long: `Install one or more gh-aw skills into the target coding agent's skill directory.

If no skill names are provided, all available skills are installed.

The --agent flag selects the coding agent target. Supported agents:
  claude-code  Claude Code (~/.claude/agents/)
  copilot      GitHub Copilot CLI (~/.config/gh-copilot/agents/)
  codex        Codex (~/.codex/agents/)

Examples:
  gh aw skills install                              # Install all skills for Claude Code
  gh aw skills install install-workflow             # Install a specific skill
  gh aw skills install --agent copilot              # Install for GitHub Copilot CLI
  gh aw skills install compile-workflows --force    # Overwrite existing skill`,
		RunE: func(cmd *cobra.Command, args []string) error {
			agentName, _ := cmd.Flags().GetString("agent")
			force, _ := cmd.Flags().GetBool("force")

			agent := AgentType(agentName)
			skillsLog.Printf("Installing skills: agent=%s, skills=%v, force=%v", agent, args, force)

			targetDir, err := AgentSkillsDir(agent)
			if err != nil {
				return err
			}

			// Ensure the target directory exists
			if err := os.MkdirAll(targetDir, 0750); err != nil {
				return fmt.Errorf("failed to create skills directory %s: %w", targetDir, err)
			}

			// Determine which skills to install
			skillNames := args
			if len(skillNames) == 0 {
				skillNames, err = listUserSkills()
				if err != nil {
					return err
				}
			}

			// Validate skill names before installing any of them
			available, err := listUserSkills()
			if err != nil {
				return err
			}
			availableSet := make(map[string]bool, len(available))
			for _, s := range available {
				availableSet[s] = true
			}
			for _, name := range skillNames {
				if !availableSet[name] {
					return fmt.Errorf("unknown skill %q; run 'gh aw skills list' to see available skills", name)
				}
			}

			fmt.Fprintln(os.Stderr, console.FormatInfoMessage(
				fmt.Sprintf("Installing gh-aw skills for %s into %s", agent, targetDir)))
			fmt.Fprintln(os.Stderr, "")

			var installed int
			for _, name := range skillNames {
				if err := installSkill(name, targetDir, force); err != nil {
					return err
				}
				installed++
			}

			fmt.Fprintln(os.Stderr, "")
			fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(
				fmt.Sprintf("Installed %d skill(s) to %s", installed, targetDir)))
			return nil
		},
	}

	cmd.Flags().String("agent", string(AgentClaudeCode), "Target coding agent (claude-code, copilot, codex)")
	cmd.Flags().Bool("force", false, "Overwrite existing skill files")
	// Register shell completion for the --agent flag
	if err := cmd.RegisterFlagCompletionFunc("agent", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		var names []string
		for _, a := range supportedAgents {
			names = append(names, string(a))
		}
		return names, cobra.ShellCompDirectiveNoFileComp
	}); err != nil {
		skillsLog.Printf("Failed to register agent flag completion: %v", err)
	}

	return cmd
}

// newSkillsPathSubcommand creates the `skills path` subcommand.
func newSkillsPathSubcommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "path",
		Short: "Print the skill install directory for the given coding agent",
		Long: `Print the path to the directory where skills are installed for the given coding agent.

The --agent flag selects the coding agent target. Supported agents:
  claude-code  Claude Code (~/.claude/agents/)
  copilot      GitHub Copilot CLI (~/.config/gh-copilot/agents/)
  codex        Codex (~/.codex/agents/)

Examples:
  gh aw skills path                    # Print path for Claude Code (default)
  gh aw skills path --agent copilot    # Print path for GitHub Copilot CLI`,
		RunE: func(cmd *cobra.Command, args []string) error {
			agentName, _ := cmd.Flags().GetString("agent")
			agent := AgentType(agentName)
			skillsLog.Printf("Getting skills path: agent=%s", agent)

			dir, err := AgentSkillsDir(agent)
			if err != nil {
				return err
			}

			// Print to stdout — this is structured data suitable for piping/scripting
			fmt.Println(dir)
			return nil
		},
	}

	cmd.Flags().String("agent", string(AgentClaudeCode), "Target coding agent (claude-code, copilot, codex)")
	if err := cmd.RegisterFlagCompletionFunc("agent", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		var names []string
		for _, a := range supportedAgents {
			names = append(names, string(a))
		}
		return names, cobra.ShellCompDirectiveNoFileComp
	}); err != nil {
		skillsLog.Printf("Failed to register agent flag completion: %v", err)
	}

	return cmd
}
