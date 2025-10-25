package workflow

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/githubnext/gh-aw/pkg/console"
	"github.com/githubnext/gh-aw/pkg/logger"
	"github.com/goccy/go-yaml"
)

var dependabotLog = logger.New("workflow:dependabot")

// PackageJSON represents the structure of a package.json file
type PackageJSON struct {
	Name            string            `json:"name"`
	Private         bool              `json:"private"`
	License         string            `json:"license,omitempty"`
	Dependencies    map[string]string `json:"dependencies,omitempty"`
	DevDependencies map[string]string `json:"devDependencies,omitempty"`
}

// DependabotConfig represents the structure of .github/dependabot.yml
type DependabotConfig struct {
	Version int                     `yaml:"version"`
	Updates []DependabotUpdateEntry `yaml:"updates"`
}

// DependabotUpdateEntry represents a single update configuration in dependabot.yml
type DependabotUpdateEntry struct {
	PackageEcosystem string `yaml:"package-ecosystem"`
	Directory        string `yaml:"directory"`
	Schedule         struct {
		Interval string `yaml:"interval"`
	} `yaml:"schedule"`
}

// NpmDependency represents a parsed npm package with version
type NpmDependency struct {
	Name    string
	Version string // semver range or specific version
}

// GenerateDependabotManifests generates package.json and dependabot.yml if npm dependencies are found
func (c *Compiler) GenerateDependabotManifests(workflowDataList []*WorkflowData, workflowDir string, forceOverwrite bool) error {
	dependabotLog.Print("Starting Dependabot manifest generation")

	// Collect all npm dependencies from all workflows
	allDeps := c.collectNpmDependencies(workflowDataList)
	if len(allDeps) == 0 {
		dependabotLog.Print("No npm dependencies found, skipping manifest generation")
		if c.verbose {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage("No npm dependencies detected in workflows, skipping Dependabot manifest generation"))
		}
		return nil
	}

	dependabotLog.Printf("Found %d unique npm dependencies", len(allDeps))
	if c.verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Found %d npm dependencies in workflows", len(allDeps))))
	}

	// Generate package.json
	packageJSONPath := filepath.Join(workflowDir, "package.json")
	if err := c.generatePackageJSON(packageJSONPath, allDeps, forceOverwrite); err != nil {
		if c.strictMode {
			return fmt.Errorf("failed to generate package.json: %w", err)
		}
		c.IncrementWarningCount()
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to generate package.json: %v", err)))
		return nil
	}

	// Generate package-lock.json
	if err := c.generatePackageLock(workflowDir); err != nil {
		if c.strictMode {
			return fmt.Errorf("failed to generate package-lock.json: %w", err)
		}
		c.IncrementWarningCount()
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to generate package-lock.json: %v", err)))
	}

	// Generate dependabot.yml
	dependabotPath := filepath.Join(filepath.Dir(workflowDir), "dependabot.yml")
	if err := c.generateDependabotConfig(dependabotPath, forceOverwrite); err != nil {
		if c.strictMode {
			return fmt.Errorf("failed to generate dependabot.yml: %w", err)
		}
		c.IncrementWarningCount()
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to generate dependabot.yml: %v", err)))
	}

	if c.verbose {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Successfully generated Dependabot manifests"))
	}

	return nil
}

// collectNpmDependencies collects all npm dependencies from workflow data
func (c *Compiler) collectNpmDependencies(workflowDataList []*WorkflowData) []NpmDependency {
	dependabotLog.Print("Collecting npm dependencies from workflows")

	depMap := make(map[string]string) // package name -> version (last seen)

	for _, workflowData := range workflowDataList {
		packages := extractNpxPackages(workflowData)
		for _, pkg := range packages {
			dep := parseNpmPackage(pkg)
			depMap[dep.Name] = dep.Version
		}
	}

	// Convert map to sorted slice
	var deps []NpmDependency
	for name, version := range depMap {
		deps = append(deps, NpmDependency{
			Name:    name,
			Version: version,
		})
	}

	// Sort by name for deterministic output
	sort.Slice(deps, func(i, j int) bool {
		return deps[i].Name < deps[j].Name
	})

	dependabotLog.Printf("Collected %d unique dependencies", len(deps))
	return deps
}

// parseNpmPackage parses a package string like "@playwright/mcp@latest" into name and version
func parseNpmPackage(pkg string) NpmDependency {
	// Handle scoped packages (@org/package@version)
	if strings.HasPrefix(pkg, "@") {
		// Find the second @ for version separator
		parts := strings.Split(pkg, "@")
		if len(parts) >= 3 {
			// @org/package@version
			return NpmDependency{
				Name:    "@" + parts[1],
				Version: parts[2],
			}
		} else if len(parts) == 2 {
			// @org/package (no version)
			return NpmDependency{
				Name:    pkg,
				Version: "latest",
			}
		}
	}

	// Handle non-scoped packages (package@version)
	parts := strings.SplitN(pkg, "@", 2)
	if len(parts) == 2 {
		return NpmDependency{
			Name:    parts[0],
			Version: parts[1],
		}
	}

	// No version specified
	return NpmDependency{
		Name:    pkg,
		Version: "latest",
	}
}

// generatePackageJSON creates or updates package.json with dependencies
func (c *Compiler) generatePackageJSON(path string, deps []NpmDependency, forceOverwrite bool) error {
	dependabotLog.Printf("Generating package.json at %s", path)

	var pkgJSON PackageJSON

	// Check if package.json already exists
	if _, err := os.Stat(path); err == nil {
		// File exists - merge dependencies
		dependabotLog.Print("Existing package.json found, merging dependencies")

		existingData, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read existing package.json: %w", err)
		}

		if err := json.Unmarshal(existingData, &pkgJSON); err != nil {
			return fmt.Errorf("failed to parse existing package.json: %w", err)
		}

		if c.verbose {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Merging with existing package.json"))
		}
	} else {
		// New package.json
		dependabotLog.Print("Creating new package.json")
		pkgJSON = PackageJSON{
			Name:    "gh-aw-workflows-deps",
			Private: true,
			License: "MIT",
		}
	}

	// Initialize dependencies map if nil
	if pkgJSON.Dependencies == nil {
		pkgJSON.Dependencies = make(map[string]string)
	}

	// Add/update dependencies
	for _, dep := range deps {
		pkgJSON.Dependencies[dep.Name] = dep.Version
	}

	// Write package.json with nice formatting
	jsonData, err := json.MarshalIndent(pkgJSON, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal package.json: %w", err)
	}

	// Add newline at end for POSIX compliance
	jsonData = append(jsonData, '\n')

	if err := os.WriteFile(path, jsonData, 0644); err != nil {
		return fmt.Errorf("failed to write package.json: %w", err)
	}

	dependabotLog.Printf("Successfully wrote package.json with %d dependencies", len(pkgJSON.Dependencies))
	if c.verbose {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("Generated package.json with %d dependencies", len(pkgJSON.Dependencies))))
	}

	// Track the created file
	if c.fileTracker != nil {
		c.fileTracker.TrackCreated(path)
	}

	return nil
}

// generatePackageLock runs npm install --package-lock-only to create package-lock.json
func (c *Compiler) generatePackageLock(workflowDir string) error {
	dependabotLog.Printf("Generating package-lock.json in %s", workflowDir)

	// Check if npm is available
	npmPath, err := exec.LookPath("npm")
	if err != nil {
		return fmt.Errorf("npm command not found - cannot generate package-lock.json. Install Node.js/npm to enable this feature")
	}

	if c.verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Running npm install --package-lock-only..."))
	}

	// Run npm install --package-lock-only
	cmd := exec.Command(npmPath, "install", "--package-lock-only")
	cmd.Dir = workflowDir

	// Capture output for error reporting
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("npm install --package-lock-only failed: %w\nOutput: %s", err, string(output))
	}

	lockfilePath := filepath.Join(workflowDir, "package-lock.json")
	if _, err := os.Stat(lockfilePath); err != nil {
		return fmt.Errorf("package-lock.json was not created")
	}

	dependabotLog.Print("Successfully generated package-lock.json")
	if c.verbose {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Generated package-lock.json"))
	}

	// Track the created file
	if c.fileTracker != nil {
		c.fileTracker.TrackCreated(lockfilePath)
	}

	return nil
}

// generateDependabotConfig creates or updates .github/dependabot.yml
func (c *Compiler) generateDependabotConfig(path string, forceOverwrite bool) error {
	dependabotLog.Printf("Generating dependabot.yml at %s", path)

	var config DependabotConfig

	// Check if dependabot.yml already exists
	if _, err := os.Stat(path); err == nil {
		if !forceOverwrite {
			// File exists and we're not forcing - preserve it
			dependabotLog.Print("Existing dependabot.yml found, preserving (use --force to overwrite)")
			if c.verbose {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage("Existing dependabot.yml preserved (use compile with --force flag to overwrite)"))
			}
			return nil
		}

		// Force overwrite - read existing config to potentially merge
		dependabotLog.Print("Existing dependabot.yml found, will overwrite")
		existingData, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read existing dependabot.yml: %w", err)
		}

		if err := yaml.Unmarshal(existingData, &config); err != nil {
			// If we can't parse it, start fresh
			dependabotLog.Print("Could not parse existing dependabot.yml, creating new one")
			config = DependabotConfig{Version: 2}
		}
	} else {
		// New dependabot.yml
		dependabotLog.Print("Creating new dependabot.yml")
		config = DependabotConfig{Version: 2}
	}

	// Check if npm ecosystem already exists for .github/workflows
	npmExists := false
	for _, update := range config.Updates {
		if update.PackageEcosystem == "npm" && update.Directory == "/.github/workflows" {
			npmExists = true
			break
		}
	}

	// Add npm ecosystem if it doesn't exist
	if !npmExists {
		npmUpdate := DependabotUpdateEntry{
			PackageEcosystem: "npm",
			Directory:        "/.github/workflows",
		}
		npmUpdate.Schedule.Interval = "weekly"
		config.Updates = append(config.Updates, npmUpdate)
	}

	// Write dependabot.yml
	yamlData, err := yaml.Marshal(&config)
	if err != nil {
		return fmt.Errorf("failed to marshal dependabot.yml: %w", err)
	}

	if err := os.WriteFile(path, yamlData, 0644); err != nil {
		return fmt.Errorf("failed to write dependabot.yml: %w", err)
	}

	dependabotLog.Print("Successfully wrote dependabot.yml")
	if c.verbose {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Generated .github/dependabot.yml"))
	}

	// Track the created file
	if c.fileTracker != nil {
		c.fileTracker.TrackCreated(path)
	}

	return nil
}
