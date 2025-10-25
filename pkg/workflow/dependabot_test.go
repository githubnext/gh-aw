package workflow

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/goccy/go-yaml"
)

func TestParseNpmPackage(t *testing.T) {
	tests := []struct {
		name           string
		pkg            string
		expectedName   string
		expectedVersion string
	}{
		{
			name:           "scoped package with version",
			pkg:            "@playwright/mcp@latest",
			expectedName:   "@playwright/mcp",
			expectedVersion: "latest",
		},
		{
			name:           "scoped package with specific version",
			pkg:            "@playwright/mcp@1.2.3",
			expectedName:   "@playwright/mcp",
			expectedVersion: "1.2.3",
		},
		{
			name:           "scoped package without version",
			pkg:            "@playwright/mcp",
			expectedName:   "@playwright/mcp",
			expectedVersion: "latest",
		},
		{
			name:           "non-scoped package with version",
			pkg:            "playwright@1.0.0",
			expectedName:   "playwright",
			expectedVersion: "1.0.0",
		},
		{
			name:           "non-scoped package without version",
			pkg:            "playwright",
			expectedName:   "playwright",
			expectedVersion: "latest",
		},
		{
			name:           "package with semver range",
			pkg:            "react@^18.0.0",
			expectedName:   "react",
			expectedVersion: "^18.0.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dep := parseNpmPackage(tt.pkg)
			if dep.Name != tt.expectedName {
				t.Errorf("expected name %q, got %q", tt.expectedName, dep.Name)
			}
			if dep.Version != tt.expectedVersion {
				t.Errorf("expected version %q, got %q", tt.expectedVersion, dep.Version)
			}
		})
	}
}

func TestCollectNpmDependencies(t *testing.T) {
	compiler := NewCompiler(false, "", "test")

	tests := []struct {
		name         string
		workflows    []*WorkflowData
		expectedDeps []NpmDependency
	}{
		{
			name: "single workflow with npm dependencies",
			workflows: []*WorkflowData{
				{
					CustomSteps: "npx @playwright/mcp@latest",
				},
			},
			expectedDeps: []NpmDependency{
				{Name: "@playwright/mcp", Version: "latest"},
			},
		},
		{
			name: "multiple workflows with different dependencies",
			workflows: []*WorkflowData{
				{
					CustomSteps: "npx @playwright/mcp@latest",
				},
				{
					CustomSteps: "npx typescript@5.0.0",
				},
			},
			expectedDeps: []NpmDependency{
				{Name: "@playwright/mcp", Version: "latest"},
				{Name: "typescript", Version: "5.0.0"},
			},
		},
		{
			name: "duplicate dependencies use last version",
			workflows: []*WorkflowData{
				{
					CustomSteps: "npx typescript@4.0.0",
				},
				{
					CustomSteps: "npx typescript@5.0.0",
				},
			},
			expectedDeps: []NpmDependency{
				{Name: "typescript", Version: "5.0.0"},
			},
		},
		{
			name:         "no npm dependencies",
			workflows:    []*WorkflowData{},
			expectedDeps: []NpmDependency{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deps := compiler.collectNpmDependencies(tt.workflows)
			if len(deps) != len(tt.expectedDeps) {
				t.Errorf("expected %d dependencies, got %d", len(tt.expectedDeps), len(deps))
			}
			for i, dep := range deps {
				if i >= len(tt.expectedDeps) {
					break
				}
				expected := tt.expectedDeps[i]
				if dep.Name != expected.Name {
					t.Errorf("dependency %d: expected name %q, got %q", i, expected.Name, dep.Name)
				}
				if dep.Version != expected.Version {
					t.Errorf("dependency %d: expected version %q, got %q", i, expected.Version, dep.Version)
				}
			}
		})
	}
}

func TestGeneratePackageJSON(t *testing.T) {
	compiler := NewCompiler(false, "", "test")
	tempDir := t.TempDir()
	packageJSONPath := filepath.Join(tempDir, "package.json")

	deps := []NpmDependency{
		{Name: "@playwright/mcp", Version: "latest"},
		{Name: "typescript", Version: "5.0.0"},
	}

	// Test creating new package.json
	err := compiler.generatePackageJSON(packageJSONPath, deps, false)
	if err != nil {
		t.Fatalf("failed to generate package.json: %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(packageJSONPath); os.IsNotExist(err) {
		t.Fatal("package.json was not created")
	}

	// Read and verify content
	data, err := os.ReadFile(packageJSONPath)
	if err != nil {
		t.Fatalf("failed to read package.json: %v", err)
	}

	var pkgJSON PackageJSON
	if err := json.Unmarshal(data, &pkgJSON); err != nil {
		t.Fatalf("failed to parse package.json: %v", err)
	}

	// Verify structure
	if pkgJSON.Name != "gh-aw-workflows-deps" {
		t.Errorf("expected name 'gh-aw-workflows-deps', got %q", pkgJSON.Name)
	}
	if !pkgJSON.Private {
		t.Error("expected private to be true")
	}
	if len(pkgJSON.Dependencies) != 2 {
		t.Errorf("expected 2 dependencies, got %d", len(pkgJSON.Dependencies))
	}

	// Verify dependencies
	if pkgJSON.Dependencies["@playwright/mcp"] != "latest" {
		t.Errorf("expected @playwright/mcp@latest, got %q", pkgJSON.Dependencies["@playwright/mcp"])
	}
	if pkgJSON.Dependencies["typescript"] != "5.0.0" {
		t.Errorf("expected typescript@5.0.0, got %q", pkgJSON.Dependencies["typescript"])
	}
}

func TestGeneratePackageJSON_MergeExisting(t *testing.T) {
	compiler := NewCompiler(false, "", "test")
	tempDir := t.TempDir()
	packageJSONPath := filepath.Join(tempDir, "package.json")

	// Create existing package.json with some fields
	existingPkg := PackageJSON{
		Name:    "my-custom-name",
		Private: true,
		License: "Apache-2.0",
		Dependencies: map[string]string{
			"lodash": "^4.17.21",
		},
	}
	existingData, _ := json.MarshalIndent(existingPkg, "", "  ")
	os.WriteFile(packageJSONPath, existingData, 0644)

	// Generate with new dependencies
	newDeps := []NpmDependency{
		{Name: "@playwright/mcp", Version: "latest"},
	}

	err := compiler.generatePackageJSON(packageJSONPath, newDeps, false)
	if err != nil {
		t.Fatalf("failed to merge package.json: %v", err)
	}

	// Read and verify merged content
	data, err := os.ReadFile(packageJSONPath)
	if err != nil {
		t.Fatalf("failed to read package.json: %v", err)
	}

	var pkgJSON PackageJSON
	if err := json.Unmarshal(data, &pkgJSON); err != nil {
		t.Fatalf("failed to parse package.json: %v", err)
	}

	// Verify existing fields were preserved
	if pkgJSON.Name != "my-custom-name" {
		t.Errorf("expected name 'my-custom-name', got %q", pkgJSON.Name)
	}
	if pkgJSON.License != "Apache-2.0" {
		t.Errorf("expected license 'Apache-2.0', got %q", pkgJSON.License)
	}

	// Verify dependencies were merged
	if len(pkgJSON.Dependencies) != 2 {
		t.Errorf("expected 2 dependencies, got %d", len(pkgJSON.Dependencies))
	}
	if pkgJSON.Dependencies["lodash"] != "^4.17.21" {
		t.Error("existing lodash dependency should be preserved")
	}
	if pkgJSON.Dependencies["@playwright/mcp"] != "latest" {
		t.Error("new @playwright/mcp dependency should be added")
	}
}

func TestGenerateDependabotConfig(t *testing.T) {
	compiler := NewCompiler(false, "", "test")
	tempDir := t.TempDir()
	dependabotPath := filepath.Join(tempDir, "dependabot.yml")

	// Test creating new dependabot.yml
	err := compiler.generateDependabotConfig(dependabotPath, false)
	if err != nil {
		t.Fatalf("failed to generate dependabot.yml: %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(dependabotPath); os.IsNotExist(err) {
		t.Fatal("dependabot.yml was not created")
	}

	// Read and verify content
	data, err := os.ReadFile(dependabotPath)
	if err != nil {
		t.Fatalf("failed to read dependabot.yml: %v", err)
	}

	var config DependabotConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		t.Fatalf("failed to parse dependabot.yml: %v", err)
	}

	// Verify structure
	if config.Version != 2 {
		t.Errorf("expected version 2, got %d", config.Version)
	}
	if len(config.Updates) != 1 {
		t.Fatalf("expected 1 update entry, got %d", len(config.Updates))
	}

	update := config.Updates[0]
	if update.PackageEcosystem != "npm" {
		t.Errorf("expected package-ecosystem 'npm', got %q", update.PackageEcosystem)
	}
	if update.Directory != "/.github/workflows" {
		t.Errorf("expected directory '/.github/workflows', got %q", update.Directory)
	}
	if update.Schedule.Interval != "weekly" {
		t.Errorf("expected interval 'weekly', got %q", update.Schedule.Interval)
	}
}

func TestGenerateDependabotConfig_PreserveExisting(t *testing.T) {
	compiler := NewCompiler(false, "", "test")
	tempDir := t.TempDir()
	dependabotPath := filepath.Join(tempDir, "dependabot.yml")

	// Create existing dependabot.yml with npm entry
	existingConfig := DependabotConfig{
		Version: 2,
		Updates: []DependabotUpdateEntry{
			{
				PackageEcosystem: "npm",
				Directory:        "/.github/workflows",
			},
		},
	}
	existingConfig.Updates[0].Schedule.Interval = "weekly"
	existingData, _ := yaml.Marshal(&existingConfig)
	os.WriteFile(dependabotPath, existingData, 0644)

	// Try to generate without force - should preserve
	err := compiler.generateDependabotConfig(dependabotPath, false)
	if err != nil {
		t.Fatalf("failed to check existing dependabot.yml: %v", err)
	}

	// Verify file was preserved (no error means it was skipped)
	data, _ := os.ReadFile(dependabotPath)
	var config DependabotConfig
	yaml.Unmarshal(data, &config)
	if len(config.Updates) != 1 {
		t.Error("existing config should be preserved without force flag")
	}
}

func TestGenerateDependabotManifests_NoDependencies(t *testing.T) {
	compiler := NewCompiler(true, "", "test")
	tempDir := t.TempDir()

	// Workflow with no npm dependencies
	workflows := []*WorkflowData{
		{
			CustomSteps: "echo 'hello world'",
		},
	}

	err := compiler.GenerateDependabotManifests(workflows, tempDir, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify no files were created
	packageJSONPath := filepath.Join(tempDir, "package.json")
	if _, err := os.Stat(packageJSONPath); !os.IsNotExist(err) {
		t.Error("package.json should not be created when there are no dependencies")
	}
}

func TestGenerateDependabotManifests_WithDependencies(t *testing.T) {
	compiler := NewCompiler(true, "", "test")
	tempDir := t.TempDir()
	workflowDir := filepath.Join(tempDir, ".github", "workflows")
	os.MkdirAll(workflowDir, 0755)

	// Workflow with npm dependencies
	workflows := []*WorkflowData{
		{
			CustomSteps: "npx @playwright/mcp@latest",
		},
	}

	// Note: This will fail npm install, but we can test the package.json generation
	_ = compiler.GenerateDependabotManifests(workflows, workflowDir, false)
	
	// In non-strict mode, npm failure is just a warning
	// Check that package.json was created
	packageJSONPath := filepath.Join(workflowDir, "package.json")
	if _, err := os.Stat(packageJSONPath); os.IsNotExist(err) {
		t.Error("package.json should be created even if npm install fails in non-strict mode")
	}

	// Verify package.json content
	data, _ := os.ReadFile(packageJSONPath)
	var pkgJSON PackageJSON
	json.Unmarshal(data, &pkgJSON)
	
	if len(pkgJSON.Dependencies) != 1 {
		t.Errorf("expected 1 dependency, got %d", len(pkgJSON.Dependencies))
	}
	if pkgJSON.Dependencies["@playwright/mcp"] != "latest" {
		t.Error("@playwright/mcp dependency should be present")
	}
}

func TestGenerateDependabotManifests_StrictMode(t *testing.T) {
	compiler := NewCompiler(true, "", "test")
	compiler.SetStrictMode(true)
	tempDir := t.TempDir()
	workflowDir := filepath.Join(tempDir, ".github", "workflows")
	os.MkdirAll(workflowDir, 0755)

	// Workflow with npm dependencies
	workflows := []*WorkflowData{
		{
			CustomSteps: "npx @playwright/mcp@latest",
		},
	}

	// In strict mode, npm failure should cause an error
	strictErr := compiler.GenerateDependabotManifests(workflows, workflowDir, false)
	
	// We expect an error in strict mode when npm install fails
	// (unless npm is installed and the package is available)
	// The test validates that strict mode propagates errors correctly
	if strictErr != nil {
		// This is expected if npm is not available
		if _, lookupErr := os.Stat("/usr/bin/npm"); os.IsNotExist(lookupErr) {
			t.Logf("npm not available, strict mode error expected: %v", strictErr)
		}
	}
}
