package cli

import (
"fmt"
"os"
"os/exec"
"path/filepath"
"strings"

"github.com/githubnext/gh-aw/pkg/console"
"github.com/spf13/cobra"
)

// NewDevDepsCommand creates the dev-deps command
func NewDevDepsCommand() *cobra.Command {
cmd := &cobra.Command{
Use:   "dev-deps",
Short: "Validate development dependencies (Node.js, npm, nvm)",
Long: `Validate that all required development dependencies are installed and configured correctly.

This command checks for:
  - Node.js installation and version
  - npm installation and version
  - nvm installation (optional but recommended)
  - Local dev dependencies (prettier, etc.)

Examples:
  gh aw dev-deps              # Check all development dependencies
  gh aw dev-deps --verbose    # Show detailed version information`,
Run: func(cmd *cobra.Command, args []string) {
verbose, _ := cmd.Flags().GetBool("verbose")
if err := validateDevDeps(verbose); err != nil {
fmt.Fprintln(os.Stderr, console.FormatErrorMessage(err.Error()))
os.Exit(1)
}
},
}

return cmd
}

// validateDevDeps checks for required development dependencies
func validateDevDeps(verbose bool) error {
hasErrors := false
warnings := []string{}

// Check Node.js
fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Checking Node.js installation..."))
nodeVersion, err := getCommandVersion("node", "--version")
if err != nil {
fmt.Fprintln(os.Stderr, console.FormatErrorMessage("✗ Node.js is not installed or not in PATH"))
hasErrors = true
} else {
fmt.Fprintf(os.Stderr, "%s Node.js %s\n", console.FormatSuccessMessage("✓"), nodeVersion)
if verbose {
nodePath, _ := exec.LookPath("node")
fmt.Fprintf(os.Stderr, "  Path: %s\n", nodePath)
}
}

// Check npm
fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Checking npm installation..."))
npmVersion, err := getCommandVersion("npm", "--version")
if err != nil {
fmt.Fprintln(os.Stderr, console.FormatErrorMessage("✗ npm is not installed or not in PATH"))
hasErrors = true
} else {
fmt.Fprintf(os.Stderr, "%s npm %s\n", console.FormatSuccessMessage("✓"), npmVersion)
if verbose {
npmPath, _ := exec.LookPath("npm")
fmt.Fprintf(os.Stderr, "  Path: %s\n", npmPath)
}
}

// Check nvm (optional)
fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Checking nvm installation..."))
nvmVersion, err := getNvmVersion()
if err != nil {
fmt.Fprintln(os.Stderr, console.FormatWarningMessage("⚠ nvm is not installed (optional but recommended)"))
warnings = append(warnings, "nvm is not installed. Consider installing nvm for better Node.js version management.")
} else {
fmt.Fprintf(os.Stderr, "%s nvm %s\n", console.FormatSuccessMessage("✓"), nvmVersion)
if verbose {
// Check .nvmrc
nvmrcPath := filepath.Join(".", ".nvmrc")
if _, err := os.Stat(nvmrcPath); err == nil {
content, _ := os.ReadFile(nvmrcPath)
fmt.Fprintf(os.Stderr, "  .nvmrc specifies Node.js version: %s\n", strings.TrimSpace(string(content)))
}
}
}

// Check local dev dependencies
fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Checking local dev dependencies..."))
devDir := filepath.Join(".", "dev")
devPackageJSON := filepath.Join(devDir, "package.json")
devNodeModules := filepath.Join(devDir, "node_modules")

if _, err := os.Stat(devPackageJSON); os.IsNotExist(err) {
fmt.Fprintln(os.Stderr, console.FormatErrorMessage("✗ dev/package.json not found"))
hasErrors = true
} else if _, err := os.Stat(devNodeModules); os.IsNotExist(err) {
fmt.Fprintln(os.Stderr, console.FormatErrorMessage("✗ dev/node_modules not found"))
fmt.Fprintln(os.Stderr, console.FormatInfoMessage("  Run 'make deps' or 'cd dev && npm install' to install dependencies"))
hasErrors = true
} else {
fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("✓ Local dev dependencies installed"))

// Check for prettier specifically
prettierBin := filepath.Join(devNodeModules, ".bin", "prettier")
if _, err := os.Stat(prettierBin); os.IsNotExist(err) {
fmt.Fprintln(os.Stderr, console.FormatWarningMessage("⚠ prettier not found in dev/node_modules/.bin"))
warnings = append(warnings, "prettier is missing from local dev dependencies")
} else {
if verbose {
fmt.Fprintf(os.Stderr, "  prettier: %s\n", prettierBin)
}
}
}

// Display summary
fmt.Fprintln(os.Stderr, "")
if hasErrors {
fmt.Fprintln(os.Stderr, console.FormatErrorMessage("✗ Development environment validation failed"))
fmt.Fprintln(os.Stderr, "")
fmt.Fprintln(os.Stderr, console.FormatInfoMessage("To fix issues:"))
fmt.Fprintln(os.Stderr, "  1. Install Node.js: https://nodejs.org/")
fmt.Fprintln(os.Stderr, "  2. Install nvm (optional): https://github.com/nvm-sh/nvm")
fmt.Fprintln(os.Stderr, "  3. Run 'make deps' to install local dependencies")
return fmt.Errorf("development environment validation failed")
}

if len(warnings) > 0 {
fmt.Fprintln(os.Stderr, console.FormatWarningMessage("⚠ Warnings:"))
for _, warning := range warnings {
fmt.Fprintf(os.Stderr, "  - %s\n", warning)
}
fmt.Fprintln(os.Stderr, "")
}

fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("✓ All required development dependencies are installed"))
return nil
}

// getCommandVersion runs a command with version flag and returns the version string
func getCommandVersion(command string, versionFlag string) (string, error) {
cmd := exec.Command(command, versionFlag)
output, err := cmd.CombinedOutput()
if err != nil {
return "", err
}
return strings.TrimSpace(string(output)), nil
}

// getNvmVersion checks for nvm installation and returns version
func getNvmVersion() (string, error) {
// nvm is typically a shell function, not a binary, so we need to check differently
// Try to find nvm.sh or detect it via NVM_DIR environment variable
nvmDir := os.Getenv("NVM_DIR")
if nvmDir == "" {
// Try common locations
homeDir, _ := os.UserHomeDir()
possiblePaths := []string{
filepath.Join(homeDir, ".nvm", "nvm.sh"),
"/usr/local/opt/nvm/nvm.sh",
}

for _, path := range possiblePaths {
if _, err := os.Stat(path); err == nil {
nvmDir = filepath.Dir(path)
break
}
}
}

if nvmDir == "" {
return "", fmt.Errorf("nvm not found")
}

// Try to get version from nvm.sh
nvmScript := filepath.Join(nvmDir, "nvm.sh")
if _, err := os.Stat(nvmScript); err != nil {
return "", fmt.Errorf("nvm.sh not found")
}

// Execute nvm --version through bash
cmd := exec.Command("bash", "-c", fmt.Sprintf("source %s && nvm --version", nvmScript))
output, err := cmd.CombinedOutput()
if err != nil {
// If version check fails, just return that nvm is installed
return "installed", nil
}

return strings.TrimSpace(string(output)), nil
}
