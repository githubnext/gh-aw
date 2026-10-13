//go:build !integration

package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPostStepsSafeOutputsGuidedErrorCodemod_Metadata(t *testing.T) {
	t.Parallel()
	codemod := getPostStepsSafeOutputsGuidedErrorCodemod()

	assert.Equal(t, "post-steps-safe-outputs-guided-error", codemod.ID)
	assert.NotEmpty(t, codemod.Name)
	assert.NotEmpty(t, codemod.Description)
	assert.NotEmpty(t, codemod.IntroducedIn)
	assert.True(t, codemod.Guided, "codemod must be guided since manual remediation is required")
	assert.NotNil(t, codemod.Apply)
	assert.NotNil(t, codemod.ApplyWithContext, "codemod must expose ApplyWithContext to resolve post-steps from imports")
}

func TestPostStepsSafeOutputsGuidedErrorCodemod_Apply(t *testing.T) {
	t.Parallel()
	codemod := getPostStepsSafeOutputsGuidedErrorCodemod()

	tests := []struct {
		name        string
		content     string
		frontmatter map[string]any
		wantErr     bool
		errContains []string
	}{
		{
			name: "post-step reading GH_AW_SAFE_OUTPUTS returns guided error",
			content: `---
on: workflow_dispatch
post-steps:
  - name: Check outputs
    run: cat $GH_AW_SAFE_OUTPUTS
---

# Agent
`,
			frontmatter: map[string]any{
				"on": "workflow_dispatch",
				"post-steps": []any{
					map[string]any{
						"name": "Check outputs",
						"run":  "cat $GH_AW_SAFE_OUTPUTS",
					},
				},
			},
			wantErr: true,
			errContains: []string{
				`post-steps step 1 ("Check outputs") accesses the safe outputs output file directly`,
				"use safe-outputs instead",
				"https://github.github.com/gh-aw/reference/safe-outputs/",
			},
		},
		{
			name: "post-step reading jq with GH_AW_SAFE_OUTPUTS returns guided error",
			content: `---
on: workflow_dispatch
post-steps:
  - run: jq . $GH_AW_SAFE_OUTPUTS
---
`,
			frontmatter: map[string]any{
				"on": "workflow_dispatch",
				"post-steps": []any{
					map[string]any{
						"run": "jq . $GH_AW_SAFE_OUTPUTS",
					},
				},
			},
			wantErr: true,
			errContains: []string{
				`post-steps step 1 accesses the safe outputs output file directly`,
				"use safe-outputs instead",
				"https://github.github.com/gh-aw/reference/safe-outputs/",
			},
		},
		{
			name: "post-step reading literal agent_output.json path returns guided error",
			content: `---
on: workflow_dispatch
post-steps:
  - name: Verify output exists
    run: test -s /tmp/gh-aw/agent_output.json
---
`,
			frontmatter: map[string]any{
				"on": "workflow_dispatch",
				"post-steps": []any{
					map[string]any{
						"name": "Verify output exists",
						"run":  "test -s /tmp/gh-aw/agent_output.json",
					},
				},
			},
			wantErr: true,
			errContains: []string{
				`post-steps step 1 ("Verify output exists") accesses the safe outputs output file directly`,
				"use safe-outputs instead",
			},
		},
		{
			name: "strict: false still detects safe outputs access in post-steps",
			content: `---
on: workflow_dispatch
strict: false
post-steps:
  - name: Check outputs
    run: cat $GH_AW_SAFE_OUTPUTS
---
`,
			frontmatter: map[string]any{
				"on":     "workflow_dispatch",
				"strict": false,
				"post-steps": []any{
					map[string]any{
						"name": "Check outputs",
						"run":  "cat $GH_AW_SAFE_OUTPUTS",
					},
				},
			},
			wantErr: true,
			errContains: []string{
				`post-steps step 1 ("Check outputs") accesses the safe outputs output file directly`,
			},
		},
		{
			name: "benign post-step is a no-op",
			content: `---
on: workflow_dispatch
post-steps:
  - name: Echo status
    run: echo "Done"
  - uses: actions/upload-artifact@v4
    with:
      name: report
      path: ./report.txt
---
`,
			frontmatter: map[string]any{
				"on": "workflow_dispatch",
				"post-steps": []any{
					map[string]any{
						"name": "Echo status",
						"run":  `echo "Done"`,
					},
					map[string]any{
						"uses": "actions/upload-artifact@v4",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "no post-steps is a no-op",
			content: `---
on: workflow_dispatch
---
`,
			frontmatter: map[string]any{
				"on": "workflow_dispatch",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			newContent, applied, err := codemod.Apply(tt.content, tt.frontmatter)
			assert.False(t, applied, "guided codemod never modifies the workflow content directly")
			assert.Equal(t, tt.content, newContent, "content must be preserved")
			if !tt.wantErr {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			for _, expected := range tt.errContains {
				assert.Contains(t, err.Error(), expected)
			}
		})
	}
}

func TestPostStepsSafeOutputsGuidedErrorCodemod_ApplyWithContext_Imported(t *testing.T) {
	t.Parallel()
	codemod := getPostStepsSafeOutputsGuidedErrorCodemod()

	dir := t.TempDir()

	// Shared import file with post-step that accesses $GH_AW_SAFE_OUTPUTS
	importContent := `---
post-steps:
  - name: Imported check
    run: cat $GH_AW_SAFE_OUTPUTS
---
`
	importPath := filepath.Join(dir, "shared-post-steps.md")
	require.NoError(t, os.WriteFile(importPath, []byte(importContent), 0o644))

	// Main workflow file importing shared-post-steps.md
	mainContent := `---
on: workflow_dispatch
imports:
  - shared-post-steps.md
---

# Agent
`
	mainPath := filepath.Join(dir, "workflow.md")
	require.NoError(t, os.WriteFile(mainPath, []byte(mainContent), 0o644))

	frontmatter := map[string]any{
		"on":      "workflow_dispatch",
		"imports": []any{"shared-post-steps.md"},
	}

	newContent, applied, err := codemod.ApplyWithContext(mainContent, frontmatter, mainPath)
	assert.False(t, applied, "guided codemod never modifies the workflow")
	assert.Equal(t, mainContent, newContent, "content must be preserved")
	require.Error(t, err, "should detect safe outputs access from imported post-steps")
	assert.Contains(t, err.Error(), `post-steps step 1 ("Imported check") accesses the safe outputs output file directly`)
	assert.Contains(t, err.Error(), "https://github.github.com/gh-aw/reference/safe-outputs/")
}

func TestPostStepsSafeOutputsGuidedErrorCodemod_ApplyWithContext_NoImportedViolation(t *testing.T) {
	t.Parallel()
	codemod := getPostStepsSafeOutputsGuidedErrorCodemod()

	dir := t.TempDir()

	importContent := `---
post-steps:
  - name: Benign check
    run: echo "All good"
---
`
	importPath := filepath.Join(dir, "shared-benign.md")
	require.NoError(t, os.WriteFile(importPath, []byte(importContent), 0o644))

	mainContent := `---
on: workflow_dispatch
imports:
  - shared-benign.md
---

# Agent
`
	mainPath := filepath.Join(dir, "workflow.md")
	require.NoError(t, os.WriteFile(mainPath, []byte(mainContent), 0o644))

	frontmatter := map[string]any{
		"on":      "workflow_dispatch",
		"imports": []any{"shared-benign.md"},
	}

	newContent, applied, err := codemod.ApplyWithContext(mainContent, frontmatter, mainPath)
	assert.False(t, applied)
	assert.Equal(t, mainContent, newContent)
	require.NoError(t, err)
}

func TestPostStepsSafeOutputsGuidedErrorCodemod_ProcessWorkflowFile_GuidedErrorWrapped(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	workflowFile := filepath.Join(tmpDir, "check-requirements.md")

	content := `---
on: workflow_dispatch
post-steps:
  - name: Check requirements
    run: cat $GH_AW_SAFE_OUTPUTS
---

# Check
`
	require.NoError(t, os.WriteFile(workflowFile, []byte(content), 0o644))

	codemod := getPostStepsSafeOutputsGuidedErrorCodemod()
	_, _, err := processWorkflowFileWithInfo(workflowFile, []Codemod{codemod}, false, false)

	require.Error(t, err)
	var guidedErr *GuidedError
	require.ErrorAs(t, err, &guidedErr, "error should be wrapped as GuidedError")
	assert.Contains(t, err.Error(), `post-steps step 1 ("Check requirements") accesses the safe outputs output file directly`)
}
