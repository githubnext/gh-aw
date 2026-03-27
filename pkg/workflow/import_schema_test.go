//go:build !integration

package workflow_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/stringutil"
	"github.com/github/gh-aw/pkg/testutil"
	"github.com/github/gh-aw/pkg/workflow"
)

// TestImportWithUsesAndWith tests that imports can use 'uses'/'with' syntax as an
// alias for 'path'/'inputs'.
func TestImportWithUsesAndWith(t *testing.T) {
	tempDir := testutil.TempDir(t, "test-import-uses-with-*")

	sharedPath := filepath.Join(tempDir, "shared", "worker.md")
	if err := os.MkdirAll(filepath.Dir(sharedPath), 0755); err != nil {
		t.Fatalf("Failed to create shared directory: %v", err)
	}

	sharedContent := `---
import-schema:
  region:
    description: AWS region to target
    type: string
    required: true
  count:
    description: Number of items
    type: number
    default: 10
---

# Worker Instructions

Deploy ${{ github.aw.import-inputs.count }} items to ${{ github.aw.import-inputs.region }}.
`
	if err := os.WriteFile(sharedPath, []byte(sharedContent), 0644); err != nil {
		t.Fatalf("Failed to write shared file: %v", err)
	}

	workflowPath := filepath.Join(tempDir, "main.md")
	workflowContent := `---
on: issues
permissions:
  contents: read
  issues: read
engine: copilot
imports:
  - uses: shared/worker.md
    with:
      region: us-east-1
      count: 5
---

# Main Workflow

Runs the worker.
`
	if err := os.WriteFile(workflowPath, []byte(workflowContent), 0644); err != nil {
		t.Fatalf("Failed to write workflow file: %v", err)
	}

	compiler := workflow.NewCompiler()
	if err := compiler.CompileWorkflow(workflowPath); err != nil {
		t.Fatalf("CompileWorkflow failed: %v", err)
	}

	lockFilePath := stringutil.MarkdownToLockFile(workflowPath)
	lockFileContent, err := os.ReadFile(lockFilePath)
	if err != nil {
		t.Fatalf("Failed to read lock file: %v", err)
	}
	lockContent := string(lockFileContent)

	if !strings.Contains(lockContent, "5 items") {
		t.Errorf("Expected lock file to contain substituted count '5 items', got:\n%s", lockContent)
	}
	if !strings.Contains(lockContent, "us-east-1") {
		t.Errorf("Expected lock file to contain substituted region 'us-east-1'")
	}

	if strings.Contains(lockContent, "github.aw.import-inputs.region") {
		t.Error("Expected github.aw.import-inputs.region to be substituted in lock file")
	}
	if strings.Contains(lockContent, "github.aw.import-inputs.count") {
		t.Error("Expected github.aw.import-inputs.count to be substituted in lock file")
	}
}

// TestImportSchemaValidationMissingRequired tests that the compiler rejects imports
// that are missing a required 'with' value declared in import-schema.
func TestImportSchemaValidationMissingRequired(t *testing.T) {
	tempDir := testutil.TempDir(t, "test-import-schema-missing-*")

	sharedPath := filepath.Join(tempDir, "shared", "required.md")
	if err := os.MkdirAll(filepath.Dir(sharedPath), 0755); err != nil {
		t.Fatalf("Failed to create shared directory: %v", err)
	}

	sharedContent := `---
import-schema:
  region:
    description: AWS region
    type: string
    required: true
---

# Shared Instructions

Region: ${{ github.aw.import-inputs.region }}.
`
	if err := os.WriteFile(sharedPath, []byte(sharedContent), 0644); err != nil {
		t.Fatalf("Failed to write shared file: %v", err)
	}

	workflowPath := filepath.Join(tempDir, "main.md")
	workflowContent := `---
on: issues
permissions:
  contents: read
  issues: read
engine: copilot
imports:
  - uses: shared/required.md
    with: {}
---

# Main Workflow

Missing required input.
`
	if err := os.WriteFile(workflowPath, []byte(workflowContent), 0644); err != nil {
		t.Fatalf("Failed to write workflow file: %v", err)
	}

	compiler := workflow.NewCompiler()
	err := compiler.CompileWorkflow(workflowPath)
	if err == nil {
		t.Fatal("Expected compilation to fail due to missing required 'with' input, but it succeeded")
	}
	if !strings.Contains(err.Error(), "region") {
		t.Errorf("Expected error to mention 'region', got: %v", err)
	}
}

// TestImportSchemaValidationUnknownKey tests that the compiler rejects imports
// that provide an unknown 'with' key not declared in import-schema.
func TestImportSchemaValidationUnknownKey(t *testing.T) {
	tempDir := testutil.TempDir(t, "test-import-schema-unknown-*")

	sharedPath := filepath.Join(tempDir, "shared", "typed.md")
	if err := os.MkdirAll(filepath.Dir(sharedPath), 0755); err != nil {
		t.Fatalf("Failed to create shared directory: %v", err)
	}

	sharedContent := `---
import-schema:
  region:
    type: string
---

# Shared Instructions

Region: ${{ github.aw.import-inputs.region }}.
`
	if err := os.WriteFile(sharedPath, []byte(sharedContent), 0644); err != nil {
		t.Fatalf("Failed to write shared file: %v", err)
	}

	workflowPath := filepath.Join(tempDir, "main.md")
	workflowContent := `---
on: issues
permissions:
  contents: read
  issues: read
engine: copilot
imports:
  - uses: shared/typed.md
    with:
      region: us-east-1
      unknown_param: foo
---

# Main Workflow

Has unknown key.
`
	if err := os.WriteFile(workflowPath, []byte(workflowContent), 0644); err != nil {
		t.Fatalf("Failed to write workflow file: %v", err)
	}

	compiler := workflow.NewCompiler()
	err := compiler.CompileWorkflow(workflowPath)
	if err == nil {
		t.Fatal("Expected compilation to fail due to unknown 'with' key, but it succeeded")
	}
	if !strings.Contains(err.Error(), "unknown_param") {
		t.Errorf("Expected error to mention 'unknown_param', got: %v", err)
	}
}

// TestImportSchemaChoiceValidation tests that choice type validation works.
func TestImportSchemaChoiceValidation(t *testing.T) {
	tempDir := testutil.TempDir(t, "test-import-schema-choice-*")

	sharedPath := filepath.Join(tempDir, "shared", "env.md")
	if err := os.MkdirAll(filepath.Dir(sharedPath), 0755); err != nil {
		t.Fatalf("Failed to create shared directory: %v", err)
	}

	sharedContent := `---
import-schema:
  environment:
    type: choice
    options:
      - staging
      - production
    required: true
---

# Environment Instructions

Deploy to ${{ github.aw.import-inputs.environment }}.
`
	if err := os.WriteFile(sharedPath, []byte(sharedContent), 0644); err != nil {
		t.Fatalf("Failed to write shared file: %v", err)
	}

	t.Run("valid choice value", func(t *testing.T) {
		workflowPath := filepath.Join(tempDir, "valid.md")
		workflowContent := `---
on: issues
permissions:
  contents: read
  issues: read
engine: copilot
imports:
  - uses: shared/env.md
    with:
      environment: staging
---

# Valid Choice
`
		if err := os.WriteFile(workflowPath, []byte(workflowContent), 0644); err != nil {
			t.Fatalf("Failed to write workflow file: %v", err)
		}
		compiler := workflow.NewCompiler()
		if err := compiler.CompileWorkflow(workflowPath); err != nil {
			t.Fatalf("Expected compilation to succeed with valid choice, got: %v", err)
		}
	})

	t.Run("invalid choice value", func(t *testing.T) {
		workflowPath := filepath.Join(tempDir, "invalid.md")
		workflowContent := `---
on: issues
permissions:
  contents: read
  issues: read
engine: copilot
imports:
  - uses: shared/env.md
    with:
      environment: development
---

# Invalid Choice
`
		if err := os.WriteFile(workflowPath, []byte(workflowContent), 0644); err != nil {
			t.Fatalf("Failed to write workflow file: %v", err)
		}
		compiler := workflow.NewCompiler()
		err := compiler.CompileWorkflow(workflowPath)
		if err == nil {
			t.Fatal("Expected compilation to fail for invalid choice value")
		}
		if !strings.Contains(err.Error(), "development") {
			t.Errorf("Expected error to mention 'development', got: %v", err)
		}
	})
}

// TestImportSchemaNoSchemaBackwardCompat tests that imports without import-schema
// still work (backward compatibility).
func TestImportSchemaNoSchemaBackwardCompat(t *testing.T) {
	tempDir := testutil.TempDir(t, "test-import-no-schema-*")

	sharedPath := filepath.Join(tempDir, "shared", "noschema.md")
	if err := os.MkdirAll(filepath.Dir(sharedPath), 0755); err != nil {
		t.Fatalf("Failed to create shared directory: %v", err)
	}

	// Shared workflow uses old-style 'inputs' field (no import-schema)
	sharedContent := `---
inputs:
  count:
    type: number
    default: 10
---

# No Schema Instructions

Count: ${{ github.aw.inputs.count }}.
`
	if err := os.WriteFile(sharedPath, []byte(sharedContent), 0644); err != nil {
		t.Fatalf("Failed to write shared file: %v", err)
	}

	workflowPath := filepath.Join(tempDir, "main.md")
	workflowContent := `---
on: issues
permissions:
  contents: read
  issues: read
engine: copilot
imports:
  - uses: shared/noschema.md
    with:
      count: 42
---

# Main Workflow
`
	if err := os.WriteFile(workflowPath, []byte(workflowContent), 0644); err != nil {
		t.Fatalf("Failed to write workflow file: %v", err)
	}

	compiler := workflow.NewCompiler()
	if err := compiler.CompileWorkflow(workflowPath); err != nil {
		t.Fatalf("CompileWorkflow failed (backward compat): %v", err)
	}

	lockFilePath := stringutil.MarkdownToLockFile(workflowPath)
	lockFileContent, err := os.ReadFile(lockFilePath)
	if err != nil {
		t.Fatalf("Failed to read lock file: %v", err)
	}
	lockContent := string(lockFileContent)

	if !strings.Contains(lockContent, "Count: 42") {
		t.Errorf("Expected lock file to contain 'Count: 42'")
	}
}

// TestImportSchemaObjectType tests that object type inputs with one-level deep
// properties are validated and that sub-fields are accessible via dotted expressions.
func TestImportSchemaObjectType(t *testing.T) {
	tempDir := testutil.TempDir(t, "test-import-schema-object-*")

	sharedPath := filepath.Join(tempDir, "shared", "config-worker.md")
	if err := os.MkdirAll(filepath.Dir(sharedPath), 0755); err != nil {
		t.Fatalf("Failed to create shared directory: %v", err)
	}

	sharedContent := `---
import-schema:
  config:
    type: object
    description: Configuration object
    properties:
      apiKey:
        type: string
        required: true
      timeout:
        type: number
        default: 30
  region:
    type: string
    required: true
---

# Config Worker Instructions

API key: ${{ github.aw.import-inputs.config.apiKey }}.
Timeout: ${{ github.aw.import-inputs.config.timeout }}.
Region: ${{ github.aw.import-inputs.region }}.
`
	if err := os.WriteFile(sharedPath, []byte(sharedContent), 0644); err != nil {
		t.Fatalf("Failed to write shared file: %v", err)
	}

	t.Run("valid object input substitution", func(t *testing.T) {
		workflowPath := filepath.Join(tempDir, "valid.md")
		workflowContent := `---
on: issues
permissions:
  contents: read
  issues: read
engine: copilot
imports:
  - uses: shared/config-worker.md
    with:
      config:
        apiKey: my-secret-key
        timeout: 60
      region: eu-west-1
---

# Valid Object
`
		if err := os.WriteFile(workflowPath, []byte(workflowContent), 0644); err != nil {
			t.Fatalf("Failed to write workflow file: %v", err)
		}
		compiler := workflow.NewCompiler()
		if err := compiler.CompileWorkflow(workflowPath); err != nil {
			t.Fatalf("Expected compilation to succeed with valid object input, got: %v", err)
		}

		lockFilePath := stringutil.MarkdownToLockFile(workflowPath)
		lockContent, err := os.ReadFile(lockFilePath)
		if err != nil {
			t.Fatalf("Failed to read lock file: %v", err)
		}
		content := string(lockContent)

		if !strings.Contains(content, "my-secret-key") {
			t.Errorf("Expected lock file to contain substituted apiKey 'my-secret-key'")
		}
		if !strings.Contains(content, "60") {
			t.Errorf("Expected lock file to contain substituted timeout '60'")
		}
		if !strings.Contains(content, "eu-west-1") {
			t.Errorf("Expected lock file to contain substituted region 'eu-west-1'")
		}
		if strings.Contains(content, "github.aw.import-inputs.config.apiKey") {
			t.Error("Expected expression to be substituted in lock file")
		}
	})

	t.Run("missing required sub-property", func(t *testing.T) {
		workflowPath := filepath.Join(tempDir, "missing-sub.md")
		workflowContent := `---
on: issues
permissions:
  contents: read
  issues: read
engine: copilot
imports:
  - uses: shared/config-worker.md
    with:
      config:
        timeout: 60
      region: eu-west-1
---

# Missing required sub-prop
`
		if err := os.WriteFile(workflowPath, []byte(workflowContent), 0644); err != nil {
			t.Fatalf("Failed to write workflow file: %v", err)
		}
		compiler := workflow.NewCompiler()
		err := compiler.CompileWorkflow(workflowPath)
		if err == nil {
			t.Fatal("Expected compilation to fail due to missing required 'apiKey'")
		}
		if !strings.Contains(err.Error(), "apiKey") {
			t.Errorf("Expected error to mention 'apiKey', got: %v", err)
		}
	})

	t.Run("unknown sub-property", func(t *testing.T) {
		workflowPath := filepath.Join(tempDir, "unknown-sub.md")
		workflowContent := `---
on: issues
permissions:
  contents: read
  issues: read
engine: copilot
imports:
  - uses: shared/config-worker.md
    with:
      config:
        apiKey: key
        unknownProp: value
      region: eu-west-1
---

# Unknown sub-prop
`
		if err := os.WriteFile(workflowPath, []byte(workflowContent), 0644); err != nil {
			t.Fatalf("Failed to write workflow file: %v", err)
		}
		compiler := workflow.NewCompiler()
		err := compiler.CompileWorkflow(workflowPath)
		if err == nil {
			t.Fatal("Expected compilation to fail due to unknown sub-property")
		}
		if !strings.Contains(err.Error(), "unknownProp") {
			t.Errorf("Expected error to mention 'unknownProp', got: %v", err)
		}
	})
}

// TestImportSchemaArrayType tests that array type inputs are validated and substituted
// correctly, including as a YAML inline array in the imported workflow's mcp-servers.serena field.
func TestImportSchemaArrayType(t *testing.T) {
	tempDir := testutil.TempDir(t, "test-import-schema-array-*")

	sharedPath := filepath.Join(tempDir, "shared", "mcp", "serena.md")
	if err := os.MkdirAll(filepath.Dir(sharedPath), 0755); err != nil {
		t.Fatalf("Failed to create shared directory: %v", err)
	}

	// Shared workflow with mcp-servers.serena parameterised via import-schema
	sharedContent := `---
import-schema:
  languages:
    type: array
    items:
      type: string
    required: true
    description: Languages to enable for Serena analysis

mcp-servers:
  serena: ${{ github.aw.import-inputs.languages }}
---

## Serena Analysis

Configured for languages: ${{ github.aw.import-inputs.languages }}.
`
	if err := os.WriteFile(sharedPath, []byte(sharedContent), 0644); err != nil {
		t.Fatalf("Failed to write shared file: %v", err)
	}

	t.Run("valid array input configures serena tools", func(t *testing.T) {
		workflowPath := filepath.Join(tempDir, "valid.md")
		workflowContent := `---
on: issues
permissions:
  contents: read
  issues: read
engine: copilot
imports:
  - uses: shared/mcp/serena.md
    with:
      languages:
        - go
        - typescript
---

# Valid Array Input
`
		if err := os.WriteFile(workflowPath, []byte(workflowContent), 0644); err != nil {
			t.Fatalf("Failed to write workflow file: %v", err)
		}
		compiler := workflow.NewCompiler()
		if err := compiler.CompileWorkflow(workflowPath); err != nil {
			t.Fatalf("Expected compilation to succeed, got: %v", err)
		}

		lockFilePath := stringutil.MarkdownToLockFile(workflowPath)
		lockContent, err := os.ReadFile(lockFilePath)
		if err != nil {
			t.Fatalf("Failed to read lock file: %v", err)
		}
		content := string(lockContent)

		// The serena tool should be configured with both languages
		if !strings.Contains(content, "go") {
			t.Errorf("Expected lock file to contain 'go' in serena config")
		}
		if !strings.Contains(content, "typescript") {
			t.Errorf("Expected lock file to contain 'typescript' in serena config")
		}
		// The markdown body expression should be substituted too
		if strings.Contains(content, "github.aw.import-inputs.languages") {
			t.Error("Expected import-inputs expression to be substituted in lock file")
		}
	})

	t.Run("wrong type for array input is rejected", func(t *testing.T) {
		workflowPath := filepath.Join(tempDir, "wrong-type.md")
		workflowContent := `---
on: issues
permissions:
  contents: read
  issues: read
engine: copilot
imports:
  - uses: shared/mcp/serena.md
    with:
      languages: "go"
---

# Wrong type
`
		if err := os.WriteFile(workflowPath, []byte(workflowContent), 0644); err != nil {
			t.Fatalf("Failed to write workflow file: %v", err)
		}
		compiler := workflow.NewCompiler()
		err := compiler.CompileWorkflow(workflowPath)
		if err == nil {
			t.Fatal("Expected compilation to fail because 'languages' should be an array, not a string")
		}
		if !strings.Contains(err.Error(), "languages") {
			t.Errorf("Expected error to mention 'languages', got: %v", err)
		}
	})

	t.Run("array items type validated", func(t *testing.T) {
		workflowPath := filepath.Join(tempDir, "wrong-item-type.md")
		workflowContent := `---
on: issues
permissions:
  contents: read
  issues: read
engine: copilot
imports:
  - uses: shared/mcp/serena.md
    with:
      languages:
        - go
        - 42
---

# Wrong item type
`
		if err := os.WriteFile(workflowPath, []byte(workflowContent), 0644); err != nil {
			t.Fatalf("Failed to write workflow file: %v", err)
		}
		compiler := workflow.NewCompiler()
		err := compiler.CompileWorkflow(workflowPath)
		if err == nil {
			t.Fatal("Expected compilation to fail because array items should be strings, not numbers")
		}
	})

	t.Run("missing required array input", func(t *testing.T) {
		workflowPath := filepath.Join(tempDir, "missing-required.md")
		workflowContent := `---
on: issues
permissions:
  contents: read
  issues: read
engine: copilot
imports:
  - uses: shared/mcp/serena.md
    with: {}
---

# Missing required
`
		if err := os.WriteFile(workflowPath, []byte(workflowContent), 0644); err != nil {
			t.Fatalf("Failed to write workflow file: %v", err)
		}
		compiler := workflow.NewCompiler()
		err := compiler.CompileWorkflow(workflowPath)
		if err == nil {
			t.Fatal("Expected compilation to fail because 'languages' is required")
		}
		if !strings.Contains(err.Error(), "languages") {
			t.Errorf("Expected error to mention 'languages', got: %v", err)
		}
	})
}
