package workflow

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSafeOutputsImport tests that safe-output types can be imported from shared workflows
func TestSafeOutputsImport(t *testing.T) {
	compiler := NewCompiler(false, "", "1.0.0")

	// Create a temporary directory for test files
	tmpDir := t.TempDir()
	workflowsDir := filepath.Join(tmpDir, ".github", "workflows")
	err := os.MkdirAll(workflowsDir, 0755)
	require.NoError(t, err, "Failed to create workflows directory")

	// Create a shared workflow with create-issue configuration
	sharedWorkflow := `---
safe-outputs:
  create-issue:
    title-prefix: "[shared] "
    labels:
      - imported
      - automation
---

# Shared Create Issue Configuration

This shared workflow provides create-issue configuration.
`

	sharedFile := filepath.Join(workflowsDir, "shared-create-issue.md")
	err = os.WriteFile(sharedFile, []byte(sharedWorkflow), 0644)
	require.NoError(t, err, "Failed to write shared file")

	// Create main workflow that imports the create-issue configuration
	mainWorkflow := `---
on: issues
permissions:
  contents: read
imports:
  - ./shared-create-issue.md
---

# Main Workflow

This workflow uses the imported create-issue configuration.
`

	mainFile := filepath.Join(workflowsDir, "main.md")
	err = os.WriteFile(mainFile, []byte(mainWorkflow), 0644)
	require.NoError(t, err, "Failed to write main file")

	// Change to the workflows directory for relative path resolution
	oldDir, err := os.Getwd()
	require.NoError(t, err, "Failed to get current directory")
	err = os.Chdir(workflowsDir)
	require.NoError(t, err, "Failed to change directory")
	defer func() { _ = os.Chdir(oldDir) }()

	// Parse the main workflow
	workflowData, err := compiler.ParseWorkflowFile("main.md")
	require.NoError(t, err, "Failed to parse workflow")
	require.NotNil(t, workflowData.SafeOutputs, "SafeOutputs should not be nil")
	require.NotNil(t, workflowData.SafeOutputs.CreateIssues, "CreateIssues configuration should be imported")

	// Verify create-issue configuration was imported correctly
	assert.Equal(t, "[shared] ", workflowData.SafeOutputs.CreateIssues.TitlePrefix)
	assert.Equal(t, []string{"imported", "automation"}, workflowData.SafeOutputs.CreateIssues.Labels)
}

// TestSafeOutputsImportMultipleTypes tests importing multiple safe-output types from a shared workflow
func TestSafeOutputsImportMultipleTypes(t *testing.T) {
	compiler := NewCompiler(false, "", "1.0.0")

	// Create a temporary directory for test files
	tmpDir := t.TempDir()
	workflowsDir := filepath.Join(tmpDir, ".github", "workflows")
	err := os.MkdirAll(workflowsDir, 0755)
	require.NoError(t, err, "Failed to create workflows directory")

	// Create a shared workflow with multiple safe-output types
	sharedWorkflow := `---
safe-outputs:
  create-issue:
    title-prefix: "[bug] "
    labels:
      - bug
  add-comment:
    max: 3
---

# Shared Safe Outputs

This shared workflow provides multiple safe-output types.
`

	sharedFile := filepath.Join(workflowsDir, "shared-outputs.md")
	err = os.WriteFile(sharedFile, []byte(sharedWorkflow), 0644)
	require.NoError(t, err, "Failed to write shared file")

	// Create main workflow that imports the safe-outputs
	mainWorkflow := `---
on: issues
permissions:
  contents: read
imports:
  - ./shared-outputs.md
---

# Main Workflow
`

	mainFile := filepath.Join(workflowsDir, "main.md")
	err = os.WriteFile(mainFile, []byte(mainWorkflow), 0644)
	require.NoError(t, err, "Failed to write main file")

	// Change to the workflows directory for relative path resolution
	oldDir, err := os.Getwd()
	require.NoError(t, err, "Failed to get current directory")
	err = os.Chdir(workflowsDir)
	require.NoError(t, err, "Failed to change directory")
	defer func() { _ = os.Chdir(oldDir) }()

	// Parse the main workflow
	workflowData, err := compiler.ParseWorkflowFile("main.md")
	require.NoError(t, err, "Failed to parse workflow")
	require.NotNil(t, workflowData.SafeOutputs, "SafeOutputs should not be nil")

	// Verify both types were imported
	require.NotNil(t, workflowData.SafeOutputs.CreateIssues, "CreateIssues should be imported")
	assert.Equal(t, "[bug] ", workflowData.SafeOutputs.CreateIssues.TitlePrefix)
	assert.Equal(t, []string{"bug"}, workflowData.SafeOutputs.CreateIssues.Labels)

	require.NotNil(t, workflowData.SafeOutputs.AddComments, "AddComments should be imported")
	assert.Equal(t, 3, workflowData.SafeOutputs.AddComments.Max)
}

// TestSafeOutputsImportConflict tests that a conflict error is returned when the same safe-output type is defined in both main and imported workflow
func TestSafeOutputsImportConflict(t *testing.T) {
	compiler := NewCompiler(false, "", "1.0.0")

	// Create a temporary directory for test files
	tmpDir := t.TempDir()
	workflowsDir := filepath.Join(tmpDir, ".github", "workflows")
	err := os.MkdirAll(workflowsDir, 0755)
	require.NoError(t, err, "Failed to create workflows directory")

	// Create a shared workflow with create-issue configuration
	sharedWorkflow := `---
safe-outputs:
  create-issue:
    title-prefix: "[shared] "
---

# Shared Create Issue Configuration
`

	sharedFile := filepath.Join(workflowsDir, "shared-create-issue.md")
	err = os.WriteFile(sharedFile, []byte(sharedWorkflow), 0644)
	require.NoError(t, err, "Failed to write shared file")

	// Create main workflow that also defines create-issue (conflict)
	mainWorkflow := `---
on: issues
permissions:
  contents: read
imports:
  - ./shared-create-issue.md
safe-outputs:
  create-issue:
    title-prefix: "[main] "
---

# Main Workflow with Conflict
`

	mainFile := filepath.Join(workflowsDir, "main.md")
	err = os.WriteFile(mainFile, []byte(mainWorkflow), 0644)
	require.NoError(t, err, "Failed to write main file")

	// Change to the workflows directory for relative path resolution
	oldDir, err := os.Getwd()
	require.NoError(t, err, "Failed to get current directory")
	err = os.Chdir(workflowsDir)
	require.NoError(t, err, "Failed to change directory")
	defer func() { _ = os.Chdir(oldDir) }()

	// Parse the main workflow - should fail with conflict error
	_, err = compiler.ParseWorkflowFile("main.md")
	require.Error(t, err, "Expected conflict error")
	assert.Contains(t, err.Error(), "safe-outputs conflict")
	assert.Contains(t, err.Error(), "create-issue")
}

// TestSafeOutputsImportConflictBetweenImports tests that a conflict error is returned when the same safe-output type is defined in multiple imported workflows
func TestSafeOutputsImportConflictBetweenImports(t *testing.T) {
	compiler := NewCompiler(false, "", "1.0.0")

	// Create a temporary directory for test files
	tmpDir := t.TempDir()
	workflowsDir := filepath.Join(tmpDir, ".github", "workflows")
	err := os.MkdirAll(workflowsDir, 0755)
	require.NoError(t, err, "Failed to create workflows directory")

	// Create first shared workflow with create-issue
	sharedWorkflow1 := `---
safe-outputs:
  create-issue:
    title-prefix: "[shared1] "
---

# Shared Create Issue 1
`

	sharedFile1 := filepath.Join(workflowsDir, "shared-create-issue1.md")
	err = os.WriteFile(sharedFile1, []byte(sharedWorkflow1), 0644)
	require.NoError(t, err, "Failed to write shared file 1")

	// Create second shared workflow with create-issue (conflict)
	sharedWorkflow2 := `---
safe-outputs:
  create-issue:
    title-prefix: "[shared2] "
---

# Shared Create Issue 2
`

	sharedFile2 := filepath.Join(workflowsDir, "shared-create-issue2.md")
	err = os.WriteFile(sharedFile2, []byte(sharedWorkflow2), 0644)
	require.NoError(t, err, "Failed to write shared file 2")

	// Create main workflow that imports both (conflict between imports)
	mainWorkflow := `---
on: issues
permissions:
  contents: read
imports:
  - ./shared-create-issue1.md
  - ./shared-create-issue2.md
---

# Main Workflow with Import Conflict
`

	mainFile := filepath.Join(workflowsDir, "main.md")
	err = os.WriteFile(mainFile, []byte(mainWorkflow), 0644)
	require.NoError(t, err, "Failed to write main file")

	// Change to the workflows directory for relative path resolution
	oldDir, err := os.Getwd()
	require.NoError(t, err, "Failed to get current directory")
	err = os.Chdir(workflowsDir)
	require.NoError(t, err, "Failed to change directory")
	defer func() { _ = os.Chdir(oldDir) }()

	// Parse the main workflow - should fail with conflict error
	_, err = compiler.ParseWorkflowFile("main.md")
	require.Error(t, err, "Expected conflict error")
	assert.Contains(t, err.Error(), "safe-outputs conflict")
	assert.Contains(t, err.Error(), "create-issue")
}

// TestSafeOutputsImportNoConflictDifferentTypes tests that importing different safe-output types does not cause a conflict
func TestSafeOutputsImportNoConflictDifferentTypes(t *testing.T) {
	compiler := NewCompiler(false, "", "1.0.0")

	// Create a temporary directory for test files
	tmpDir := t.TempDir()
	workflowsDir := filepath.Join(tmpDir, ".github", "workflows")
	err := os.MkdirAll(workflowsDir, 0755)
	require.NoError(t, err, "Failed to create workflows directory")

	// Create a shared workflow with create-discussion configuration
	sharedWorkflow := `---
safe-outputs:
  create-discussion:
    title-prefix: "[shared] "
    category: "General"
---

# Shared Create Discussion Configuration
`

	sharedFile := filepath.Join(workflowsDir, "shared-create-discussion.md")
	err = os.WriteFile(sharedFile, []byte(sharedWorkflow), 0644)
	require.NoError(t, err, "Failed to write shared file")

	// Create main workflow with create-issue (different type, no conflict)
	mainWorkflow := `---
on: issues
permissions:
  contents: read
imports:
  - ./shared-create-discussion.md
safe-outputs:
  create-issue:
    title-prefix: "[main] "
---

# Main Workflow with Different Types
`

	mainFile := filepath.Join(workflowsDir, "main.md")
	err = os.WriteFile(mainFile, []byte(mainWorkflow), 0644)
	require.NoError(t, err, "Failed to write main file")

	// Change to the workflows directory for relative path resolution
	oldDir, err := os.Getwd()
	require.NoError(t, err, "Failed to get current directory")
	err = os.Chdir(workflowsDir)
	require.NoError(t, err, "Failed to change directory")
	defer func() { _ = os.Chdir(oldDir) }()

	// Parse the main workflow - should succeed
	workflowData, err := compiler.ParseWorkflowFile("main.md")
	require.NoError(t, err, "Failed to parse workflow")
	require.NotNil(t, workflowData.SafeOutputs, "SafeOutputs should not be nil")

	// Verify both types are present
	require.NotNil(t, workflowData.SafeOutputs.CreateIssues, "CreateIssues should be present from main")
	assert.Equal(t, "[main] ", workflowData.SafeOutputs.CreateIssues.TitlePrefix)

	require.NotNil(t, workflowData.SafeOutputs.CreateDiscussions, "CreateDiscussions should be imported")
	assert.Equal(t, "[shared] ", workflowData.SafeOutputs.CreateDiscussions.TitlePrefix)
	assert.Equal(t, "General", workflowData.SafeOutputs.CreateDiscussions.Category)
}

// TestSafeOutputsImportFromMultipleWorkflows tests importing different safe-output types from multiple workflows
func TestSafeOutputsImportFromMultipleWorkflows(t *testing.T) {
	compiler := NewCompiler(false, "", "1.0.0")

	// Create a temporary directory for test files
	tmpDir := t.TempDir()
	workflowsDir := filepath.Join(tmpDir, ".github", "workflows")
	err := os.MkdirAll(workflowsDir, 0755)
	require.NoError(t, err, "Failed to create workflows directory")

	// Create first shared workflow with create-issue
	sharedWorkflow1 := `---
safe-outputs:
  create-issue:
    title-prefix: "[issue] "
---

# Shared Create Issue
`

	sharedFile1 := filepath.Join(workflowsDir, "shared-issue.md")
	err = os.WriteFile(sharedFile1, []byte(sharedWorkflow1), 0644)
	require.NoError(t, err, "Failed to write shared file 1")

	// Create second shared workflow with add-comment
	sharedWorkflow2 := `---
safe-outputs:
  add-comment:
    max: 5
---

# Shared Add Comment
`

	sharedFile2 := filepath.Join(workflowsDir, "shared-comment.md")
	err = os.WriteFile(sharedFile2, []byte(sharedWorkflow2), 0644)
	require.NoError(t, err, "Failed to write shared file 2")

	// Create main workflow that imports both
	mainWorkflow := `---
on: issues
permissions:
  contents: read
imports:
  - ./shared-issue.md
  - ./shared-comment.md
---

# Main Workflow
`

	mainFile := filepath.Join(workflowsDir, "main.md")
	err = os.WriteFile(mainFile, []byte(mainWorkflow), 0644)
	require.NoError(t, err, "Failed to write main file")

	// Change to the workflows directory for relative path resolution
	oldDir, err := os.Getwd()
	require.NoError(t, err, "Failed to get current directory")
	err = os.Chdir(workflowsDir)
	require.NoError(t, err, "Failed to change directory")
	defer func() { _ = os.Chdir(oldDir) }()

	// Parse the main workflow
	workflowData, err := compiler.ParseWorkflowFile("main.md")
	require.NoError(t, err, "Failed to parse workflow")
	require.NotNil(t, workflowData.SafeOutputs, "SafeOutputs should not be nil")

	// Verify both types are present
	require.NotNil(t, workflowData.SafeOutputs.CreateIssues, "CreateIssues should be imported from first shared workflow")
	assert.Equal(t, "[issue] ", workflowData.SafeOutputs.CreateIssues.TitlePrefix)

	require.NotNil(t, workflowData.SafeOutputs.AddComments, "AddComments should be imported from second shared workflow")
	assert.Equal(t, 5, workflowData.SafeOutputs.AddComments.Max)
}

// TestMergeSafeOutputsUnit tests the MergeSafeOutputs function directly
func TestMergeSafeOutputsUnit(t *testing.T) {
	compiler := NewCompiler(false, "", "1.0.0")

	tests := []struct {
		name          string
		topConfig     *SafeOutputsConfig
		importedJSON  []string
		expectError   bool
		errorContains string
		expectedTypes []string // Types that should be present after merge
	}{
		{
			name:          "empty imports",
			topConfig:     nil,
			importedJSON:  []string{},
			expectError:   false,
			expectedTypes: []string{},
		},
		{
			name:      "import create-issue to empty config",
			topConfig: nil,
			importedJSON: []string{
				`{"create-issue":{"title-prefix":"[test] "}}`,
			},
			expectError:   false,
			expectedTypes: []string{"create-issue"},
		},
		{
			name: "conflict: create-issue in both",
			topConfig: &SafeOutputsConfig{
				CreateIssues: &CreateIssuesConfig{TitlePrefix: "[top] "},
			},
			importedJSON: []string{
				`{"create-issue":{"title-prefix":"[imported] "}}`,
			},
			expectError:   true,
			errorContains: "safe-outputs conflict",
		},
		{
			name:      "conflict: same type in multiple imports",
			topConfig: nil,
			importedJSON: []string{
				`{"create-issue":{"title-prefix":"[import1] "}}`,
				`{"create-issue":{"title-prefix":"[import2] "}}`,
			},
			expectError:   true,
			errorContains: "safe-outputs conflict",
		},
		{
			name: "no conflict: different types",
			topConfig: &SafeOutputsConfig{
				CreateIssues: &CreateIssuesConfig{TitlePrefix: "[top] "},
			},
			importedJSON: []string{
				`{"add-comment":{"max":3}}`,
			},
			expectError:   false,
			expectedTypes: []string{"create-issue", "add-comment"},
		},
		{
			name:      "import multiple types from single config",
			topConfig: nil,
			importedJSON: []string{
				`{"create-issue":{"title-prefix":"[test] "},"add-comment":{"max":5}}`,
			},
			expectError:   false,
			expectedTypes: []string{"create-issue", "add-comment"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := compiler.MergeSafeOutputs(tt.topConfig, tt.importedJSON)

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorContains)
				return
			}

			require.NoError(t, err)

			// Verify expected types are present
			for _, expectedType := range tt.expectedTypes {
				assert.True(t, hasSafeOutputType(result, expectedType), "Expected %s to be present", expectedType)
			}
		})
	}
}

// TestSafeOutputsImportMetaFields tests that safe-output meta fields can be imported from shared workflows
func TestSafeOutputsImportMetaFields(t *testing.T) {
	compiler := NewCompiler(false, "", "1.0.0")

	// Create a temporary directory for test files
	tmpDir := t.TempDir()
	workflowsDir := filepath.Join(tmpDir, ".github", "workflows")
	err := os.MkdirAll(workflowsDir, 0755)
	require.NoError(t, err, "Failed to create workflows directory")

	// Create a shared workflow with meta fields
	sharedWorkflow := `---
safe-outputs:
  allowed-domains:
    - "example.com"
    - "api.example.com"
  staged: true
  env:
    TEST_VAR: "test_value"
  github-token: "${{ secrets.CUSTOM_TOKEN }}"
  max-patch-size: 2048
  runs-on: "ubuntu-latest"
---

# Shared Meta Fields Configuration

This shared workflow provides meta configuration fields.
`

	sharedFile := filepath.Join(workflowsDir, "shared-meta.md")
	err = os.WriteFile(sharedFile, []byte(sharedWorkflow), 0644)
	require.NoError(t, err, "Failed to write shared file")

	// Create main workflow that imports the meta configuration
	mainWorkflow := `---
on: issues
permissions:
  contents: read
imports:
  - ./shared-meta.md
safe-outputs:
  create-issue:
    title-prefix: "[test] "
---

# Main Workflow

This workflow uses the imported meta configuration.
`

	mainFile := filepath.Join(workflowsDir, "main.md")
	err = os.WriteFile(mainFile, []byte(mainWorkflow), 0644)
	require.NoError(t, err, "Failed to write main file")

	// Change to the workflows directory for relative path resolution
	oldDir, err := os.Getwd()
	require.NoError(t, err, "Failed to get current directory")
	err = os.Chdir(workflowsDir)
	require.NoError(t, err, "Failed to change directory")
	defer func() { _ = os.Chdir(oldDir) }()

	// Parse the main workflow
	workflowData, err := compiler.ParseWorkflowFile("main.md")
	require.NoError(t, err, "Failed to parse workflow")
	require.NotNil(t, workflowData.SafeOutputs, "SafeOutputs should not be nil")

	// Verify create-issue from main workflow
	require.NotNil(t, workflowData.SafeOutputs.CreateIssues, "CreateIssues should be present from main")
	assert.Equal(t, "[test] ", workflowData.SafeOutputs.CreateIssues.TitlePrefix)

	// Verify imported meta fields
	assert.Equal(t, []string{"example.com", "api.example.com"}, workflowData.SafeOutputs.AllowedDomains, "AllowedDomains should be imported")
	assert.True(t, workflowData.SafeOutputs.Staged, "Staged should be imported and true")
	assert.Equal(t, map[string]string{"TEST_VAR": "test_value"}, workflowData.SafeOutputs.Env, "Env should be imported")
	assert.Equal(t, "${{ secrets.CUSTOM_TOKEN }}", workflowData.SafeOutputs.GitHubToken, "GitHubToken should be imported")
	// Note: When main workflow has safe-outputs section, extractSafeOutputsConfig sets MaximumPatchSize default (1024)
	// before merge happens, so imported value is not used. User should specify max-patch-size in main workflow.
	assert.Equal(t, 1024, workflowData.SafeOutputs.MaximumPatchSize, "MaximumPatchSize defaults to 1024 when main has safe-outputs")
	assert.Equal(t, "ubuntu-latest", workflowData.SafeOutputs.RunsOn, "RunsOn should be imported")
}

// TestSafeOutputsImportMetaFieldsMainTakesPrecedence tests that main workflow meta fields take precedence over imports
func TestSafeOutputsImportMetaFieldsMainTakesPrecedence(t *testing.T) {
	compiler := NewCompiler(false, "", "1.0.0")

	// Create a temporary directory for test files
	tmpDir := t.TempDir()
	workflowsDir := filepath.Join(tmpDir, ".github", "workflows")
	err := os.MkdirAll(workflowsDir, 0755)
	require.NoError(t, err, "Failed to create workflows directory")

	// Create a shared workflow with meta fields
	sharedWorkflow := `---
safe-outputs:
  allowed-domains:
    - "shared.example.com"
  github-token: "${{ secrets.SHARED_TOKEN }}"
  max-patch-size: 1024
---

# Shared Meta Fields Configuration
`

	sharedFile := filepath.Join(workflowsDir, "shared-meta.md")
	err = os.WriteFile(sharedFile, []byte(sharedWorkflow), 0644)
	require.NoError(t, err, "Failed to write shared file")

	// Create main workflow that has its own meta fields
	mainWorkflow := `---
on: issues
permissions:
  contents: read
imports:
  - ./shared-meta.md
safe-outputs:
  allowed-domains:
    - "main.example.com"
  github-token: "${{ secrets.MAIN_TOKEN }}"
  max-patch-size: 2048
  create-issue:
    title-prefix: "[test] "
---

# Main Workflow

This workflow has its own meta configuration that should take precedence.
`

	mainFile := filepath.Join(workflowsDir, "main.md")
	err = os.WriteFile(mainFile, []byte(mainWorkflow), 0644)
	require.NoError(t, err, "Failed to write main file")

	// Change to the workflows directory for relative path resolution
	oldDir, err := os.Getwd()
	require.NoError(t, err, "Failed to get current directory")
	err = os.Chdir(workflowsDir)
	require.NoError(t, err, "Failed to change directory")
	defer func() { _ = os.Chdir(oldDir) }()

	// Parse the main workflow
	workflowData, err := compiler.ParseWorkflowFile("main.md")
	require.NoError(t, err, "Failed to parse workflow")
	require.NotNil(t, workflowData.SafeOutputs, "SafeOutputs should not be nil")

	// Verify main workflow meta fields take precedence
	assert.Equal(t, []string{"main.example.com"}, workflowData.SafeOutputs.AllowedDomains, "AllowedDomains from main should take precedence")
	assert.Equal(t, "${{ secrets.MAIN_TOKEN }}", workflowData.SafeOutputs.GitHubToken, "GitHubToken from main should take precedence")
	assert.Equal(t, 2048, workflowData.SafeOutputs.MaximumPatchSize, "MaximumPatchSize from main should take precedence")
}

// TestSafeOutputsImportMetaFieldsFromOnlyImport tests that meta fields are correctly imported when main has no safe-outputs section
func TestSafeOutputsImportMetaFieldsFromOnlyImport(t *testing.T) {
	compiler := NewCompiler(false, "", "1.0.0")

	// Create a temporary directory for test files
	tmpDir := t.TempDir()
	workflowsDir := filepath.Join(tmpDir, ".github", "workflows")
	err := os.MkdirAll(workflowsDir, 0755)
	require.NoError(t, err, "Failed to create workflows directory")

	// Create a shared workflow with meta fields and create-issue
	sharedWorkflow := `---
safe-outputs:
  create-issue:
    title-prefix: "[imported] "
  allowed-domains:
    - "import.example.com"
  github-token: "${{ secrets.IMPORT_TOKEN }}"
  max-patch-size: 4096
  staged: true
  runs-on: "ubuntu-22.04"
---

# Shared Safe Outputs Configuration
`

	sharedFile := filepath.Join(workflowsDir, "shared-full.md")
	err = os.WriteFile(sharedFile, []byte(sharedWorkflow), 0644)
	require.NoError(t, err, "Failed to write shared file")

	// Create main workflow that has NO safe-outputs section (only imports)
	mainWorkflow := `---
on: issues
permissions:
  contents: read
imports:
  - ./shared-full.md
---

# Main Workflow

This workflow uses only imported safe-outputs configuration.
`

	mainFile := filepath.Join(workflowsDir, "main.md")
	err = os.WriteFile(mainFile, []byte(mainWorkflow), 0644)
	require.NoError(t, err, "Failed to write main file")

	// Change to the workflows directory for relative path resolution
	oldDir, err := os.Getwd()
	require.NoError(t, err, "Failed to get current directory")
	err = os.Chdir(workflowsDir)
	require.NoError(t, err, "Failed to change directory")
	defer func() { _ = os.Chdir(oldDir) }()

	// Parse the main workflow
	workflowData, err := compiler.ParseWorkflowFile("main.md")
	require.NoError(t, err, "Failed to parse workflow")
	require.NotNil(t, workflowData.SafeOutputs, "SafeOutputs should not be nil")

	// Verify safe output type from import
	require.NotNil(t, workflowData.SafeOutputs.CreateIssues, "CreateIssues should be imported")
	assert.Equal(t, "[imported] ", workflowData.SafeOutputs.CreateIssues.TitlePrefix)

	// Verify all meta fields from import (no defaults from main since main has no safe-outputs)
	assert.Equal(t, []string{"import.example.com"}, workflowData.SafeOutputs.AllowedDomains, "AllowedDomains should be imported")
	assert.Equal(t, "${{ secrets.IMPORT_TOKEN }}", workflowData.SafeOutputs.GitHubToken, "GitHubToken should be imported")
	assert.Equal(t, 4096, workflowData.SafeOutputs.MaximumPatchSize, "MaximumPatchSize should be imported")
	assert.True(t, workflowData.SafeOutputs.Staged, "Staged should be imported")
	assert.Equal(t, "ubuntu-22.04", workflowData.SafeOutputs.RunsOn, "RunsOn should be imported")
}

// TestSafeOutputsImportJobsFromSharedWorkflow tests that safe-outputs.jobs can be imported from shared workflows
func TestSafeOutputsImportJobsFromSharedWorkflow(t *testing.T) {
	compiler := NewCompiler(false, "", "1.0.0")

	// Create a temporary directory for test files
	tmpDir := t.TempDir()
	workflowsDir := filepath.Join(tmpDir, ".github", "workflows")
	err := os.MkdirAll(workflowsDir, 0755)
	require.NoError(t, err, "Failed to create workflows directory")

	// Create a shared workflow with safe-outputs.jobs configuration
	sharedWorkflow := `---
safe-outputs:
  jobs:
    my-custom-job:
      name: "My Custom Job"
      runs-on: ubuntu-latest
      permissions:
        contents: read
        issues: write
      steps:
        - name: Run custom action
          run: echo "Hello from custom job"
---

# Shared Safe Jobs Configuration

This shared workflow provides custom safe-job definitions.
`

	sharedFile := filepath.Join(workflowsDir, "shared-safe-jobs.md")
	err = os.WriteFile(sharedFile, []byte(sharedWorkflow), 0644)
	require.NoError(t, err, "Failed to write shared file")

	// Create main workflow that imports the safe-jobs configuration
	mainWorkflow := `---
on: issues
permissions:
  contents: read
imports:
  - ./shared-safe-jobs.md
---

# Main Workflow

This workflow imports safe-jobs from a shared workflow.
`

	mainFile := filepath.Join(workflowsDir, "main.md")
	err = os.WriteFile(mainFile, []byte(mainWorkflow), 0644)
	require.NoError(t, err, "Failed to write main file")

	// Change to the workflows directory for relative path resolution
	oldDir, err := os.Getwd()
	require.NoError(t, err, "Failed to get current directory")
	err = os.Chdir(workflowsDir)
	require.NoError(t, err, "Failed to change directory")
	defer func() { _ = os.Chdir(oldDir) }()

	// Parse the main workflow
	workflowData, err := compiler.ParseWorkflowFile("main.md")
	require.NoError(t, err, "Failed to parse workflow")
	require.NotNil(t, workflowData.SafeOutputs, "SafeOutputs should not be nil")

	// Verify that jobs were imported
	require.NotNil(t, workflowData.SafeOutputs.Jobs, "Jobs should be imported")
	require.Contains(t, workflowData.SafeOutputs.Jobs, "my-custom-job", "my-custom-job should be present")

	// Verify job configuration
	job := workflowData.SafeOutputs.Jobs["my-custom-job"]
	assert.Equal(t, "My Custom Job", job.Name, "Job name should match")
	assert.Equal(t, "ubuntu-latest", job.RunsOn, "Job runs-on should match")
	assert.Len(t, job.Steps, 1, "Job should have 1 step")
	assert.Contains(t, job.Permissions, "contents", "Job should have contents permission")
	assert.Contains(t, job.Permissions, "issues", "Job should have issues permission")
}

// TestSafeOutputsImportJobsWithMainWorkflowJobs tests importing jobs when main workflow also has jobs
func TestSafeOutputsImportJobsWithMainWorkflowJobs(t *testing.T) {
	compiler := NewCompiler(false, "", "1.0.0")

	// Create a temporary directory for test files
	tmpDir := t.TempDir()
	workflowsDir := filepath.Join(tmpDir, ".github", "workflows")
	err := os.MkdirAll(workflowsDir, 0755)
	require.NoError(t, err, "Failed to create workflows directory")

	// Create a shared workflow with safe-outputs.jobs configuration
	sharedWorkflow := `---
safe-outputs:
  jobs:
    imported-job:
      name: "Imported Job"
      runs-on: ubuntu-latest
      steps:
        - name: Imported step
          run: echo "Imported"
---

# Shared Safe Jobs Configuration
`

	sharedFile := filepath.Join(workflowsDir, "shared-jobs.md")
	err = os.WriteFile(sharedFile, []byte(sharedWorkflow), 0644)
	require.NoError(t, err, "Failed to write shared file")

	// Create main workflow that has its own jobs AND imports jobs
	mainWorkflow := `---
on: issues
permissions:
  contents: read
imports:
  - ./shared-jobs.md
safe-outputs:
  jobs:
    main-job:
      name: "Main Job"
      runs-on: ubuntu-latest
      steps:
        - name: Main step
          run: echo "Main"
---

# Main Workflow with Jobs

This workflow has its own jobs and imports more jobs.
`

	mainFile := filepath.Join(workflowsDir, "main.md")
	err = os.WriteFile(mainFile, []byte(mainWorkflow), 0644)
	require.NoError(t, err, "Failed to write main file")

	// Change to the workflows directory for relative path resolution
	oldDir, err := os.Getwd()
	require.NoError(t, err, "Failed to get current directory")
	err = os.Chdir(workflowsDir)
	require.NoError(t, err, "Failed to change directory")
	defer func() { _ = os.Chdir(oldDir) }()

	// Parse the main workflow
	workflowData, err := compiler.ParseWorkflowFile("main.md")
	require.NoError(t, err, "Failed to parse workflow")
	require.NotNil(t, workflowData.SafeOutputs, "SafeOutputs should not be nil")

	// Verify that both main and imported jobs are present
	require.NotNil(t, workflowData.SafeOutputs.Jobs, "Jobs should not be nil")
	require.Contains(t, workflowData.SafeOutputs.Jobs, "main-job", "main-job should be present")
	require.Contains(t, workflowData.SafeOutputs.Jobs, "imported-job", "imported-job should be imported")

	// Verify both job configurations
	mainJob := workflowData.SafeOutputs.Jobs["main-job"]
	assert.Equal(t, "Main Job", mainJob.Name, "Main job name should match")

	importedJob := workflowData.SafeOutputs.Jobs["imported-job"]
	assert.Equal(t, "Imported Job", importedJob.Name, "Imported job name should match")
}

// TestSafeOutputsImportJobsConflict tests that a conflict error is returned when the same job name is defined in both main and imported workflow
func TestSafeOutputsImportJobsConflict(t *testing.T) {
	compiler := NewCompiler(false, "", "1.0.0")

	// Create a temporary directory for test files
	tmpDir := t.TempDir()
	workflowsDir := filepath.Join(tmpDir, ".github", "workflows")
	err := os.MkdirAll(workflowsDir, 0755)
	require.NoError(t, err, "Failed to create workflows directory")

	// Create a shared workflow with safe-outputs.jobs configuration
	sharedWorkflow := `---
safe-outputs:
  jobs:
    duplicate-job:
      name: "Shared Duplicate Job"
      runs-on: ubuntu-latest
      steps:
        - name: Shared step
          run: echo "Shared"
---

# Shared Safe Jobs Configuration with Duplicate
`

	sharedFile := filepath.Join(workflowsDir, "shared-duplicate.md")
	err = os.WriteFile(sharedFile, []byte(sharedWorkflow), 0644)
	require.NoError(t, err, "Failed to write shared file")

	// Create main workflow that has the same job name (conflict)
	mainWorkflow := `---
on: issues
permissions:
  contents: read
imports:
  - ./shared-duplicate.md
safe-outputs:
  jobs:
    duplicate-job:
      name: "Main Duplicate Job"
      runs-on: ubuntu-latest
      steps:
        - name: Main step
          run: echo "Main"
---

# Main Workflow with Duplicate Job Name
`

	mainFile := filepath.Join(workflowsDir, "main.md")
	err = os.WriteFile(mainFile, []byte(mainWorkflow), 0644)
	require.NoError(t, err, "Failed to write main file")

	// Change to the workflows directory for relative path resolution
	oldDir, err := os.Getwd()
	require.NoError(t, err, "Failed to get current directory")
	err = os.Chdir(workflowsDir)
	require.NoError(t, err, "Failed to change directory")
	defer func() { _ = os.Chdir(oldDir) }()

	// Parse the main workflow - should fail with conflict error
	_, err = compiler.ParseWorkflowFile("main.md")
	require.Error(t, err, "Expected conflict error")
	assert.Contains(t, err.Error(), "duplicate-job", "Error should mention the conflicting job name")
	assert.Contains(t, err.Error(), "conflict", "Error should mention conflict")
}
