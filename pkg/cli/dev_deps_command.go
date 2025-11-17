package cli

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/githubnext/gh-aw/pkg/console"
	"github.com/githubnext/gh-aw/pkg/logger"
	"github.com/spf13/cobra"
)

var devDepsCommandLog = logger.New("cli:dev_deps_command")

// NewDevDepsCommand creates the dev-deps command
func NewDevDepsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dev-deps",
		Short: "Install development dependencies and validate environment",
		Long: `Install all development dependencies and validate that Node.js, npm, and nvm are properly configured.

This command:
- Validates Node.js installation and version
- Validates npm installation and version
- Validates nvm installation (optional but recommended)
- Installs Go dependencies
- Installs golangci-lint for linting
- Installs JavaScript dependencies in pkg/workflow/js
- Downloads GitHub Actions workflow schema

Examples:
  gh aw dev-deps
  gh aw dev-deps -v`,
		Run: func(cmd *cobra.Command, args []string) {
			verbose, _ := cmd.Flags().GetBool("verbose")
			devDepsCommandLog.Printf("Executing dev-deps command: verbose=%v", verbose)
			if err := InstallDevDeps(verbose); err != nil {
				devDepsCommandLog.Printf("Dev-deps command failed: %v", err)
				fmt.Fprintln(os.Stderr, console.FormatErrorMessage(err.Error()))
				os.Exit(1)
			}
			devDepsCommandLog.Print("Dev-deps command completed successfully")
		},
	}

	return cmd
}

// InstallDevDeps installs all development dependencies and validates the environment
func InstallDevDeps(verbose bool) error {
	// Validate Node.js environment first
	if err := ValidateNodeEnvironment(verbose); err != nil {
		return err
	}

	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Installing Go dependencies..."))
	}

	// Run go mod download
	cmd := exec.Command("go", "mod", "download")
	if verbose {
		cmd.Stdout = os.Stderr
		cmd.Stderr = os.Stderr
	}
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to download Go modules: %w", err)
	}

	// Run go mod tidy
	cmd = exec.Command("go", "mod", "tidy")
	if verbose {
		cmd.Stdout = os.Stderr
		cmd.Stderr = os.Stderr
	}
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to tidy Go modules: %w", err)
	}

	// Install gopls
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Installing gopls..."))
	}
	cmd = exec.Command("go", "install", "golang.org/x/tools/gopls@latest")
	if verbose {
		cmd.Stdout = os.Stderr
		cmd.Stderr = os.Stderr
	}
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to install gopls: %w", err)
	}

	// Install actionlint
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Installing actionlint..."))
	}
	cmd = exec.Command("go", "install", "github.com/rhysd/actionlint/cmd/actionlint@latest")
	if verbose {
		cmd.Stdout = os.Stderr
		cmd.Stderr = os.Stderr
	}
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to install actionlint: %w", err)
	}

	// Install golangci-lint
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Installing golangci-lint..."))
	}
	cmd = exec.Command("go", "install", "github.com/golangci/golangci-lint/cmd/golangci-lint@latest")
	if verbose {
		cmd.Stdout = os.Stderr
		cmd.Stderr = os.Stderr
	}
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to install golangci-lint: %w", err)
	}

	// Install JavaScript dependencies
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Installing JavaScript dependencies in pkg/workflow/js..."))
	}
	cmd = exec.Command("npm", "ci")
	cmd.Dir = "pkg/workflow/js"
	if verbose {
		cmd.Stdout = os.Stderr
		cmd.Stderr = os.Stderr
	}
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to install JavaScript dependencies: %w", err)
	}

	// Download GitHub Actions schema
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Downloading GitHub Actions workflow schema..."))
	}
	if err := DownloadGitHubActionsSchema(verbose); err != nil {
		return fmt.Errorf("failed to download GitHub Actions schema: %w", err)
	}

	fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("✓ All development dependencies installed successfully"))
	return nil
}

// ValidateNodeEnvironment validates that Node.js, npm, and optionally nvm are installed
func ValidateNodeEnvironment(verbose bool) error {
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Validating Node.js environment..."))
	}

	// Check Node.js
	nodeVersion, err := exec.Command("node", "--version").Output()
	if err != nil {
		return fmt.Errorf("Node.js is not installed or not in PATH. Please install Node.js from https://nodejs.org/")
	}
	nodeVersionStr := strings.TrimSpace(string(nodeVersion))
	if verbose {
		fmt.Fprintf(os.Stderr, "%s Node.js version: %s\n", console.FormatInfoMessage("✓"), nodeVersionStr)
	}

	// Check npm
	npmVersion, err := exec.Command("npm", "--version").Output()
	if err != nil {
		return fmt.Errorf("npm is not installed or not in PATH. Please install npm (typically comes with Node.js)")
	}
	npmVersionStr := strings.TrimSpace(string(npmVersion))
	if verbose {
		fmt.Fprintf(os.Stderr, "%s npm version: %s\n", console.FormatInfoMessage("✓"), npmVersionStr)
	}

	// Check nvm (optional)
	nvmPath, err := exec.LookPath("nvm")
	if err != nil {
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage("⚠ nvm is not installed (optional but recommended). Install from https://github.com/nvm-sh/nvm"))
		}
	} else if verbose {
		fmt.Fprintf(os.Stderr, "%s nvm is installed at: %s\n", console.FormatInfoMessage("✓"), nvmPath)
	}

	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("✓ Node.js environment validated"))
	}

	return nil
}

// DownloadGitHubActionsSchema downloads the GitHub Actions workflow schema
func DownloadGitHubActionsSchema(verbose bool) error {
	// Create directory if it doesn't exist
	if err := os.MkdirAll("pkg/workflow/schemas", 0755); err != nil {
		return fmt.Errorf("failed to create schemas directory: %w", err)
	}

	// Download schema using curl
	cmd := exec.Command("curl", "-s", "-o", "pkg/workflow/schemas/github-workflow.json",
		"https://raw.githubusercontent.com/SchemaStore/schemastore/master/src/schemas/json/github-workflow.json")
	if verbose {
		cmd.Stdout = os.Stderr
		cmd.Stderr = os.Stderr
	}
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to download schema: %w", err)
	}

	// Format schema with prettier from local installation
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Formatting schema with prettier..."))
	}
	cmd = exec.Command("npx", "prettier", "--write", "../../workflow/schemas/github-workflow.json", "--ignore-path", "/dev/null")
	cmd.Dir = "pkg/workflow/js"
	if err := cmd.Run(); err != nil {
		// Don't fail if prettier formatting fails
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage("Warning: Failed to format schema with prettier"))
		}
	}

	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("✓ Downloaded and formatted GitHub Actions schema"))
	}

	return nil
}
