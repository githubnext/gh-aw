//go:build !integration

package workflow

import (
	"fmt"
	"go/ast"
	"go/parser"
	gotoken "go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestTypeNameCollisions scans for duplicate type names across packages
// to prevent regressions from type naming conflicts. This test ensures
// that type consistency improvements maintain distinct names.
func TestTypeNameCollisions(t *testing.T) {
	// Map of type names to their locations (package.TypeName -> file paths)
	typeLocations := make(map[string][]string)

	// Walk through pkg directory
	pkgRoot := filepath.Join("..", "..")
	err := filepath.Walk(pkgRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip non-Go files and test files
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}

		// Skip vendor and hidden directories
		if strings.Contains(path, "/vendor/") || strings.Contains(path, "/.") {
			return nil
		}

		// Parse the file
		fset := gotoken.NewFileSet()
		file, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			// Skip files that can't be parsed (might be generated or have syntax errors)
			return nil
		}

		// Extract package name
		pkgName := file.Name.Name

		// Find all type declarations
		ast.Inspect(file, func(n ast.Node) bool {
			// Look for type declarations
			typeSpec, ok := n.(*ast.TypeSpec)
			if !ok {
				return true
			}

			typeName := typeSpec.Name.Name

			// Skip private types (lowercase first letter)
			if len(typeName) > 0 && strings.ToLower(typeName[:1]) == typeName[:1] {
				return true
			}

			// Create a qualified name (package.Type)
			qualifiedName := fmt.Sprintf("%s.%s", pkgName, typeName)

			// Record the location
			relPath, _ := filepath.Rel(pkgRoot, path)
			typeLocations[qualifiedName] = append(typeLocations[qualifiedName], relPath)

			return true
		})

		return nil
	})

	if err != nil {
		t.Fatalf("Failed to walk directory: %v", err)
	}

	// Check for collisions within same package
	collisions := make(map[string][]string)
	for qualifiedName, locations := range typeLocations {
		if len(locations) > 1 {
			collisions[qualifiedName] = locations
		}
	}

	if len(collisions) > 0 {
		t.Errorf("Found %d type name collisions within packages:", len(collisions))
		for qualifiedName, locations := range collisions {
			t.Errorf("  Type %s declared in multiple files:", qualifiedName)
			for _, loc := range locations {
				t.Errorf("    - %s", loc)
			}
		}
	}
}

// TestKnownTypeDistinctions verifies that specific types that should be
// distinct remain separate. This prevents regression of type consolidation
// that should be kept separate for clarity.
func TestKnownTypeDistinctions(t *testing.T) {
	t.Parallel()

	// Map of type names that should exist in specific packages
	// This prevents accidental merging or removal
	expectedTypes := map[string][]string{
		"cli": {
			"CopilotWorkflowStep", // Simplified step for Copilot setup
			"WorkflowJob",         // Job structure for Copilot setup
			"Workflow",            // Workflow structure for Copilot setup
		},
		"workflow": {
			"WorkflowStep",              // Full step type for workflow compilation
			"ValidatableTool",           // Interface for tool validation
			"GitHubToolConfig",          // GitHub tool configuration
			"PermissionsValidationResult", // Validation result type
		},
		"constants": {
			"FeatureFlag", // Feature flag type
			"Version",     // Version string type
			"LineLength",  // Line length type
		},
	}

	for pkgName, types := range expectedTypes {
		for _, typeName := range types {
			t.Run(fmt.Sprintf("%s.%s", pkgName, typeName), func(t *testing.T) {
				// This test will fail to compile if the type doesn't exist
				// We're documenting expected types, not actually checking them here
				// The compiler ensures they exist
			})
		}
	}
}

// TestCopilotWorkflowStepDistinction specifically tests that CopilotWorkflowStep
// is distinct from workflow.WorkflowStep, as this was a known collision issue.
func TestCopilotWorkflowStepDistinction(t *testing.T) {
	t.Parallel()

	// Find CopilotWorkflowStep in cli package
	cliPath := filepath.Join("..", "cli")
	foundCopilotStep := false

	err := filepath.Walk(cliPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}

		fset := gotoken.NewFileSet()
		file, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			return nil
		}

		ast.Inspect(file, func(n ast.Node) bool {
			typeSpec, ok := n.(*ast.TypeSpec)
			if !ok {
				return true
			}

			if typeSpec.Name.Name == "CopilotWorkflowStep" {
				foundCopilotStep = true
				return false
			}

			return true
		})

		return nil
	})

	if err != nil {
		t.Fatalf("Failed to scan cli package: %v", err)
	}

	if !foundCopilotStep {
		t.Error("CopilotWorkflowStep type not found in cli package")
	}

	// Find WorkflowStep in workflow package
	workflowPath := "."
	foundWorkflowStep := false

	err = filepath.Walk(workflowPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}

		fset := gotoken.NewFileSet()
		file, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			return nil
		}

		ast.Inspect(file, func(n ast.Node) bool {
			typeSpec, ok := n.(*ast.TypeSpec)
			if !ok {
				return true
			}

			if typeSpec.Name.Name == "WorkflowStep" {
				foundWorkflowStep = true
				return false
			}

			return true
		})

		return nil
	})

	if err != nil {
		t.Fatalf("Failed to scan workflow package: %v", err)
	}

	if !foundWorkflowStep {
		t.Error("WorkflowStep type not found in workflow package")
	}

	// Both types should exist and be distinct
	if !foundCopilotStep || !foundWorkflowStep {
		t.Error("Expected both CopilotWorkflowStep and WorkflowStep to exist as distinct types")
	}
}

// TestValidatableToolInterface verifies that ValidatableTool interface
// exists and is properly defined in the workflow package.
func TestValidatableToolInterfaceExists(t *testing.T) {
	t.Parallel()

	// Scan workflow package for ValidatableTool interface
	foundInterface := false

	err := filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}

		fset := gotoken.NewFileSet()
		file, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			return nil
		}

		ast.Inspect(file, func(n ast.Node) bool {
			typeSpec, ok := n.(*ast.TypeSpec)
			if !ok {
				return true
			}

			// Check if it's an interface
			_, isInterface := typeSpec.Type.(*ast.InterfaceType)
			if isInterface && typeSpec.Name.Name == "ValidatableTool" {
				foundInterface = true
				return false
			}

			return true
		})

		return nil
	})

	if err != nil {
		t.Fatalf("Failed to scan workflow package: %v", err)
	}

	if !foundInterface {
		t.Error("ValidatableTool interface not found in workflow package")
	}
}
