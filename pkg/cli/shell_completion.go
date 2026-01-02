package cli

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/githubnext/gh-aw/pkg/console"
	"github.com/githubnext/gh-aw/pkg/logger"
	"github.com/spf13/cobra"
)

var shellCompletionLog = logger.New("cli:shell_completion")

// FixCompletionScript post-processes the generated completion script to use "gh aw" instead of "gh".
// This is necessary because Cobra generates completion scripts based on the binary name,
// but GitHub CLI extensions are invoked as "gh <extension-name>".
func FixCompletionScript(script, shellType string) string {
	shellCompletionLog.Printf("Fixing completion script for shell: %s", shellType)

	// For bash and zsh, we need to replace function names and completion directives
	// For fish, we replace the command name in completion definitions
	switch shellType {
	case "bash":
		// Replace function prefixes: __gh_ -> __gh_aw_
		script = strings.ReplaceAll(script, "__gh_", "__gh_aw_")
		// Replace completion function names: _gh( -> _gh_aw(
		script = strings.ReplaceAll(script, "_gh(", "_gh_aw(")
		// Replace completion registration: complete -o default -F __gh_completion gh
		script = strings.ReplaceAll(script, " __gh_completion gh", " __gh_aw_completion gh")
		// Update completion comment header
		script = strings.ReplaceAll(script, "# bash completion V2 for gh ", "# bash completion V2 for gh aw ")
		script = strings.ReplaceAll(script, "# bash completion for gh ", "# bash completion for gh aw ")
		// Fix requestComp variable for bash v2: requestComp="${words[0]} __complete
		script = strings.ReplaceAll(script, `requestComp="${words[0]} __complete`, `requestComp="gh aw __complete`)
		// Fix requestComp variable for bash v1: requestComp="GH_ACTIVE_HELP=0 ${words[0]} __completeNoDesc
		script = strings.ReplaceAll(script, `requestComp="GH_ACTIVE_HELP=0 ${words[0]} __completeNoDesc`, `requestComp="GH_ACTIVE_HELP=0 gh aw __completeNoDesc`)

	case "zsh":
		// Replace function names
		script = strings.ReplaceAll(script, "__gh_", "__gh_aw_")
		script = strings.ReplaceAll(script, "_gh(", "_gh_aw(")
		// Update compdef to register for both "gh aw" and as a subcommand
		script = strings.ReplaceAll(script, "#compdef gh\ncompdef _gh gh", "#compdef gh\n# Register completion for 'gh aw' as a two-word command\ncompdef _gh_aw gh")
		// Update completion comment header
		script = strings.ReplaceAll(script, "# zsh completion for gh ", "# zsh completion for gh aw ")
		// Fix the requestComp to use "gh aw" instead of just the first word
		// In zsh, words[1] is the command name (after gh), so we replace it entirely with "gh aw"
		script = strings.ReplaceAll(script, `requestComp="${words[1]} __complete ${words[2,-1]}"`, `requestComp="gh aw __complete ${words[2,-1]}"`)

	case "fish":
		// For fish, replace completion command names
		// Fish uses: complete -c gh ...
		script = strings.ReplaceAll(script, "complete -c gh ", "complete -c gh -n '__fish_seen_subcommand_from aw' ")
		// Also add completion for the aw subcommand itself
		script = "# Fish completion for gh aw\n" + script
	}

	shellCompletionLog.Printf("Completion script fixed for %s", shellType)
	return script
}

// ShellType represents the detected shell type
type ShellType string

const (
	ShellBash       ShellType = "bash"
	ShellZsh        ShellType = "zsh"
	ShellFish       ShellType = "fish"
	ShellPowerShell ShellType = "powershell"
	ShellUnknown    ShellType = "unknown"
)

// DetectShell detects the current shell from environment variables
func DetectShell() ShellType {
	shellCompletionLog.Print("Detecting current shell")

	// Check shell-specific version variables first (most reliable)
	if os.Getenv("ZSH_VERSION") != "" {
		shellCompletionLog.Print("Detected zsh from ZSH_VERSION")
		return ShellZsh
	}
	if os.Getenv("BASH_VERSION") != "" {
		shellCompletionLog.Print("Detected bash from BASH_VERSION")
		return ShellBash
	}
	if os.Getenv("FISH_VERSION") != "" {
		shellCompletionLog.Print("Detected fish from FISH_VERSION")
		return ShellFish
	}

	// Fall back to $SHELL environment variable
	shell := os.Getenv("SHELL")
	if shell == "" {
		shellCompletionLog.Print("SHELL environment variable not set, checking platform")
		// On Windows, check for PowerShell
		if runtime.GOOS == "windows" {
			shellCompletionLog.Print("Detected Windows, assuming PowerShell")
			return ShellPowerShell
		}
		shellCompletionLog.Print("Could not detect shell")
		return ShellUnknown
	}

	shellCompletionLog.Printf("SHELL environment variable: %s", shell)

	// Extract shell name from path
	shellName := filepath.Base(shell)
	shellCompletionLog.Printf("Shell base name: %s", shellName)

	switch {
	case strings.Contains(shellName, "bash"):
		shellCompletionLog.Print("Detected bash from SHELL")
		return ShellBash
	case strings.Contains(shellName, "zsh"):
		shellCompletionLog.Print("Detected zsh from SHELL")
		return ShellZsh
	case strings.Contains(shellName, "fish"):
		shellCompletionLog.Print("Detected fish from SHELL")
		return ShellFish
	case strings.Contains(shellName, "pwsh") || strings.Contains(shellName, "powershell"):
		shellCompletionLog.Print("Detected PowerShell from SHELL")
		return ShellPowerShell
	default:
		shellCompletionLog.Printf("Unknown shell: %s", shellName)
		return ShellUnknown
	}
}

// InstallShellCompletion installs shell completion for the detected shell
func InstallShellCompletion(verbose bool, rootCmd CommandProvider) error {
	shellCompletionLog.Print("Starting shell completion installation")

	// Type assert rootCmd to *cobra.Command to access additional methods if needed
	// For now, we only use the CommandProvider interface methods
	cmd, ok := rootCmd.(*cobra.Command)
	if !ok {
		return fmt.Errorf("rootCmd must be a *cobra.Command")
	}

	shellType := DetectShell()
	shellCompletionLog.Printf("Detected shell type: %s", shellType)

	if shellType == ShellUnknown {
		return fmt.Errorf("could not detect shell type. Please install completions manually using: gh aw completion <shell>")
	}

	fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Detected shell: %s", shellType)))

	switch shellType {
	case ShellBash:
		return installBashCompletion(verbose, cmd)
	case ShellZsh:
		return installZshCompletion(verbose, cmd)
	case ShellFish:
		return installFishCompletion(verbose, cmd)
	case ShellPowerShell:
		return installPowerShellCompletion(verbose, cmd)
	default:
		return fmt.Errorf("shell completion not supported for: %s", shellType)
	}
}

// installBashCompletion installs bash completion
func installBashCompletion(verbose bool, cmd *cobra.Command) error {
	shellCompletionLog.Print("Installing bash completion")

	// Generate completion script using Cobra
	var buf bytes.Buffer
	if err := cmd.GenBashCompletion(&buf); err != nil {
		return fmt.Errorf("failed to generate bash completion: %w", err)
	}

	// Post-process the completion script to use "gh aw" instead of "gh"
	completionScript := FixCompletionScript(buf.String(), "bash")

	// Determine installation path
	var completionPath string
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	// Try to determine the best location for bash completions
	if runtime.GOOS == "darwin" {
		// macOS with Homebrew
		brewPrefix := os.Getenv("HOMEBREW_PREFIX")
		if brewPrefix == "" {
			// Try common locations
			for _, prefix := range []string{"/opt/homebrew", "/usr/local"} {
				if _, err := os.Stat(filepath.Join(prefix, "etc", "bash_completion.d")); err == nil {
					brewPrefix = prefix
					break
				}
			}
		}
		if brewPrefix != "" {
			completionPath = filepath.Join(brewPrefix, "etc", "bash_completion.d", "gh-aw")
		} else {
			completionPath = filepath.Join(homeDir, ".bash_completion.d", "gh-aw")
		}
	} else {
		// Linux
		if _, err := os.Stat("/etc/bash_completion.d"); err == nil {
			completionPath = "/etc/bash_completion.d/gh-aw"
		} else {
			completionPath = filepath.Join(homeDir, ".bash_completion.d", "gh-aw")
		}
	}

	// Create directory if needed (for user-level installations)
	completionDir := filepath.Dir(completionPath)
	if strings.HasPrefix(completionDir, homeDir) {
		// Use restrictive permissions (0750) following principle of least privilege
		if err := os.MkdirAll(completionDir, 0750); err != nil {
			return fmt.Errorf("failed to create completion directory: %w", err)
		}
	}

	// Try to write completion file
	// Use restrictive permissions (0600) following principle of least privilege
	err = os.WriteFile(completionPath, []byte(completionScript), 0600)
	if err != nil && strings.HasPrefix(completionPath, "/etc") {
		// If system-wide installation fails, fall back to user directory
		shellCompletionLog.Printf("Failed to install system-wide, falling back to user directory: %v", err)
		completionPath = filepath.Join(homeDir, ".bash_completion.d", "gh-aw")
		// Use restrictive permissions (0750) following principle of least privilege
		if err := os.MkdirAll(filepath.Dir(completionPath), 0750); err != nil {
			return fmt.Errorf("failed to create user completion directory: %w", err)
		}
		// Use restrictive permissions (0600) following principle of least privilege
		if err := os.WriteFile(completionPath, []byte(completionScript), 0600); err != nil {
			return fmt.Errorf("failed to write completion file: %w", err)
		}
	} else if err != nil {
		return fmt.Errorf("failed to write completion file: %w", err)
	}

	fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("Installed bash completion to: %s", completionPath)))

	// Check if .bashrc sources completions
	bashrcPath := filepath.Join(homeDir, ".bashrc")
	if strings.HasPrefix(completionPath, homeDir) {
		// For user-level installations, check if .bashrc sources the completion directory
		bashrcContent, err := os.ReadFile(bashrcPath)
		needsSourceLine := true
		if err == nil {
			if strings.Contains(string(bashrcContent), ".bash_completion.d") ||
				strings.Contains(string(bashrcContent), completionPath) {
				needsSourceLine = false
			}
		}

		if needsSourceLine {
			fmt.Fprintln(os.Stderr, "")
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage("To enable completions, add the following to your ~/.bashrc:"))
			fmt.Fprintln(os.Stderr, "")
			fmt.Fprintf(os.Stderr, "  for f in ~/.bash_completion.d/*; do [ -f \"$f\" ] && source \"$f\"; done\n")
			fmt.Fprintln(os.Stderr, "")
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Then restart your shell or run: source ~/.bashrc"))
		} else {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Please restart your shell for completions to take effect"))
		}
	} else {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Please restart your shell for completions to take effect"))
	}

	return nil
}

// installZshCompletion installs zsh completion
func installZshCompletion(verbose bool, cmd *cobra.Command) error {
	shellCompletionLog.Print("Installing zsh completion")

	// Generate completion script using Cobra
	var buf bytes.Buffer
	if err := cmd.GenZshCompletion(&buf); err != nil {
		return fmt.Errorf("failed to generate zsh completion: %w", err)
	}

	// Post-process the completion script to use "gh aw" instead of "gh"
	completionScript := FixCompletionScript(buf.String(), "zsh")

	// Determine installation path
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	// Check for fpath directories
	var completionPath string

	// Try user's local completion directory first
	userCompletionDir := filepath.Join(homeDir, ".zsh", "completions")
	// Use restrictive permissions (0750) following principle of least privilege
	if err := os.MkdirAll(userCompletionDir, 0750); err != nil {
		return fmt.Errorf("failed to create completion directory: %w", err)
	}
	completionPath = filepath.Join(userCompletionDir, "_gh-aw")

	// Write completion file
	// Use restrictive permissions (0600) following principle of least privilege
	if err := os.WriteFile(completionPath, []byte(completionScript), 0600); err != nil {
		return fmt.Errorf("failed to write completion file: %w", err)
	}

	fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("Installed zsh completion to: %s", completionPath)))

	// Check if .zshrc configures fpath
	zshrcPath := filepath.Join(homeDir, ".zshrc")
	zshrcContent, err := os.ReadFile(zshrcPath)
	needsFpath := true
	if err == nil {
		if strings.Contains(string(zshrcContent), userCompletionDir) {
			needsFpath = false
		}
	}

	if needsFpath {
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("To enable completions, add the following to your ~/.zshrc:"))
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintf(os.Stderr, "  fpath=(~/.zsh/completions $fpath)\n")
		fmt.Fprintf(os.Stderr, "  autoload -Uz compinit && compinit\n")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Then restart your shell or run: source ~/.zshrc"))
	} else {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Please restart your shell for completions to take effect"))
	}

	return nil
}

// installFishCompletion installs fish completion
func installFishCompletion(verbose bool, cmd *cobra.Command) error {
	shellCompletionLog.Print("Installing fish completion")

	// Generate completion script using Cobra
	var buf bytes.Buffer
	if err := cmd.GenFishCompletion(&buf, true); err != nil {
		return fmt.Errorf("failed to generate fish completion: %w", err)
	}

	// Post-process the completion script to use "gh aw" instead of "gh"
	completionScript := FixCompletionScript(buf.String(), "fish")

	// Determine installation path
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	// Fish completion directory
	completionDir := filepath.Join(homeDir, ".config", "fish", "completions")
	// Use restrictive permissions (0750) following principle of least privilege
	if err := os.MkdirAll(completionDir, 0750); err != nil {
		return fmt.Errorf("failed to create completion directory: %w", err)
	}

	completionPath := filepath.Join(completionDir, "gh-aw.fish")

	// Write completion file
	// Use restrictive permissions (0600) following principle of least privilege
	if err := os.WriteFile(completionPath, []byte(completionScript), 0600); err != nil {
		return fmt.Errorf("failed to write completion file: %w", err)
	}

	fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("Installed fish completion to: %s", completionPath)))
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Fish will automatically load completions on next shell start"))

	return nil
}

// installPowerShellCompletion installs PowerShell completion
func installPowerShellCompletion(verbose bool, cmd *cobra.Command) error {
	shellCompletionLog.Print("Installing PowerShell completion")

	// Determine PowerShell profile path
	var profileCmd *exec.Cmd
	if runtime.GOOS == "windows" {
		profileCmd = exec.Command("powershell", "-NoProfile", "-Command", "echo $PROFILE")
	} else {
		profileCmd = exec.Command("pwsh", "-NoProfile", "-Command", "echo $PROFILE")
	}

	var profileBuf bytes.Buffer
	profileCmd.Stdout = &profileBuf
	if err := profileCmd.Run(); err != nil {
		return fmt.Errorf("failed to get PowerShell profile path: %w", err)
	}

	profilePath := strings.TrimSpace(profileBuf.String())

	fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("PowerShell profile path: %s", profilePath)))
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage("To enable completions, add the following to your PowerShell profile:"))
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "  gh aw completion powershell | Out-String | Invoke-Expression")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Or run the following command to append it automatically:"))
	fmt.Fprintln(os.Stderr, "")
	if runtime.GOOS == "windows" {
		fmt.Fprintln(os.Stderr, "  gh aw completion powershell >> $PROFILE")
	} else {
		fmt.Fprintln(os.Stderr, "  echo 'gh aw completion powershell | Out-String | Invoke-Expression' >> $PROFILE")
	}
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Then restart your shell or run: . $PROFILE"))

	return nil
}
