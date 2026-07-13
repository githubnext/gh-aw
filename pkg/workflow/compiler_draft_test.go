//go:build !integration

package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/stringutil"

	"github.com/github/gh-aw/pkg/testutil"
	"github.com/stretchr/testify/assert"
)

func TestPullRequestDraftFilter(t *testing.T) {
	// Create temporary directory for test files
	tmpDir := testutil.TempDir(t, "draft-filter-test")

	compiler := NewCompiler()

	tests := []struct {
		name         string
		frontmatter  string
		expectedIf   string // Expected if condition in the generated lock file
		shouldHaveIf bool   // Whether an if condition should be present
	}{
		{
			name: "pull_request with draft: false",
			frontmatter: `---
on:
  pull_request:
    types: [opened, edited]
    draft: false

permissions:
  contents: read
  issues: read
  pull-requests: read

strict: false
tools:
  github:
    allowed: [issue_read]
---`,
			expectedIf:   "github.event_name != 'pull_request' || github.event.pull_request.draft == false",
			shouldHaveIf: true,
		},
		{
			name: "pull_request with draft: true (include only drafts)",
			frontmatter: `---
on:
  pull_request:
    types: [opened, edited]
    draft: true

permissions:
  contents: read
  issues: read
  pull-requests: read

strict: false
tools:
  github:
    allowed: [issue_read]
---`,
			expectedIf:   "github.event_name != 'pull_request' || github.event.pull_request.draft == true",
			shouldHaveIf: true,
		},
		{
			name: "pull_request without draft field (no filter)",
			frontmatter: `---
on:
  pull_request:
    types: [opened, edited]

permissions:
  contents: read
  issues: read
  pull-requests: read

strict: false
tools:
  github:
    allowed: [issue_read]
---`,
			shouldHaveIf: false,
		},
		{
			name: "pull_request with draft: false and existing if condition",
			frontmatter: `---
on:
  pull_request:
    types: [opened, edited]
    draft: false

if: github.actor != 'dependabot[bot]'

permissions:
  contents: read
  issues: read
  pull-requests: read

strict: false
tools:
  github:
    allowed: [issue_read]
---`,
			expectedIf:   "(github.actor != 'dependabot[bot]') && (github.event_name != 'pull_request' || github.event.pull_request.draft == false)",
			shouldHaveIf: true,
		},
		{
			name: "pull_request with draft: true and existing if condition",
			frontmatter: `---
on:
  pull_request:
    types: [opened, edited]
    draft: true

if: github.actor != 'dependabot[bot]'

permissions:
  contents: read
  issues: read
  pull-requests: read

strict: false
tools:
  github:
    allowed: [issue_read]
---`,
			expectedIf:   "(github.actor != 'dependabot[bot]') && (github.event_name != 'pull_request' || github.event.pull_request.draft == true)",
			shouldHaveIf: true,
		},
		{
			name: "non-pull_request trigger (no filter applied)",
			frontmatter: `---
on:
  issues:
    types: [opened]

permissions:
  contents: read
  issues: read
  pull-requests: read

strict: false
tools:
  github:
    allowed: [issue_read]
---`,
			shouldHaveIf: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testContent := tt.frontmatter + `

# Test Draft Filter Workflow

This is a test workflow for draft filtering.
`

			testFile := filepath.Join(tmpDir, tt.name+"-workflow.md")
			if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
				t.Fatal(err)
			}

			// Compile the workflow
			err := compiler.CompileWorkflow(testFile)
			if err != nil {
				t.Fatalf("Unexpected error compiling workflow: %v", err)
			}

			// Read the generated lock file
			lockFile := stringutil.MarkdownToLockFile(testFile)
			content, err := os.ReadFile(lockFile)
			if err != nil {
				t.Fatalf("Failed to read lock file: %v", err)
			}

			lockContent := string(content)

			if tt.shouldHaveIf {
				// Check that the expected if condition is present (normalize for multiline comparison)
				normalizedLockContent := strings.Join(strings.Fields(lockContent), " ")
				normalizedExpectedIf := strings.Join(strings.Fields(tt.expectedIf), " ")
				if !strings.Contains(normalizedLockContent, normalizedExpectedIf) {
					t.Errorf("Expected lock file to contain '%s' but it didn't.\nExpected (normalized): %s\nActual (normalized): %s\nOriginal Content:\n%s",
						tt.expectedIf, normalizedExpectedIf, normalizedLockContent, lockContent)
				}
			} else {
				// Check that no draft-related if condition is present in the main job
				if strings.Contains(lockContent, "github.event.pull_request.draft == false") {
					t.Errorf("Expected no draft filter condition but found one in lock file.\nContent:\n%s", lockContent)
				}
			}
		})
	}
}

// TestDraftFieldCommentingInOnSection specifically tests that the draft field is commented out in the on section
func TestCommentOutProcessedFieldsInOnSection(t *testing.T) {
	compiler := NewCompiler()

	tests := []struct {
		name        string
		input       string
		expected    string
		description string
	}{
		{
			name: "pull_request with draft and paths",
			input: `on:
    pull_request:
        draft: false
        paths:
            - go.mod
            - go.sum
    workflow_dispatch:`,
			expected: `on:
    pull_request:
        # draft: false # Draft filtering applied via job conditions
        paths:
            - go.mod
            - go.sum
    workflow_dispatch:`,
			description: "Should comment out draft but keep paths",
		},
		{
			name: "pull_request with two-space indentation",
			input: `on:
  pull_request:
    draft: false
  workflow_dispatch:`,
			expected: `on:
  pull_request:
    # draft: false # Draft filtering applied via job conditions
  workflow_dispatch:`,
			description: "Should comment out draft with two-space indentation style",
		},
		{
			name: "pull_request with draft and types",
			input: `on:
    pull_request:
        draft: true
        types:
            - opened
            - edited`,
			expected: `on:
    pull_request:
        # draft: true # Draft filtering applied via job conditions
        types:
            - opened
            - edited`,
			description: "Should comment out draft but keep types",
		},
		{
			name: "pull_request with only draft field",
			input: `on:
    pull_request:
        draft: false
    workflow_dispatch:`,
			expected: `on:
    pull_request:
        # draft: false # Draft filtering applied via job conditions
    workflow_dispatch:`,
			description: "Should comment out draft even when it's the only field",
		},
		{
			name: "multiple pull_request sections",
			input: `on:
    pull_request:
        draft: false
        paths:
            - "*.go"
    schedule:
        - cron: "0 9 * * 1"`,
			expected: `on:
    pull_request:
        # draft: false # Draft filtering applied via job conditions
        paths:
            - "*.go"
    schedule:
        - cron: "0 9 * * 1"`,
			description: "Should comment out draft in pull_request while leaving other sections unchanged",
		},
		{
			name: "no pull_request section",
			input: `on:
    workflow_dispatch:
    push:
        branches:
            - main`,
			expected: `on:
    workflow_dispatch:
    push:
        branches:
            - main`,
			description: "Should leave unchanged when no pull_request section",
		},
		{
			name: "pull_request without draft field",
			input: `on:
    pull_request:
        types:
            - opened`,
			expected: `on:
    pull_request:
        types:
            - opened`,
			description: "Should leave unchanged when no draft field in pull_request",
		},
		{
			name: "issues names after pull_request forks array",
			input: `on:
  pull_request:
    forks:
      - trusted/*
  issues:
    names:
      - bug
  workflow_dispatch:`,
			expected: `on:
  pull_request:
    # forks: # Fork filtering applied via job conditions
      # - trusted/* # Fork filtering applied via job conditions
  issues:
    # names: # Label filtering applied via job conditions
    # - bug # Label filtering applied via job conditions
  workflow_dispatch:`,
			description: "Should reset forks array tracker when entering a new event section",
		},
		{
			name: "issues names after workflow_run conclusion array",
			input: `on:
  workflow_run:
    conclusion:
      - failure
  issues:
    names:
      - bug
  workflow_dispatch:`,
			expected: `on:
  workflow_run:
    # conclusion: # Conclusion filtering compiled into if condition
    # - failure # Conclusion filtering compiled into if condition
  issues:
    # names: # Label filtering applied via job conditions
    # - bug # Label filtering applied via job conditions
  workflow_dispatch:`,
			description: "Should reset workflow_run conclusion tracker when entering a new event section",
		},
		{
			name: "workflow_run followed by bots keeps workflows and types uncommented",
			input: `on:
  workflow_run:
    workflows: ["CI"]
    types: [completed]
  bots:
    - dependabot
  workflow_dispatch:`,
			expected: `on:
  workflow_run:
    workflows: ["CI"]
    types: [completed]
  # bots: # Bots processed as bot check in pre-activation job
  # - dependabot # Bots processed as bot check in pre-activation job
  workflow_dispatch:`,
			description: "Should not let bots array state leak into workflow_run fields",
		},
		{
			name: "bots before workflow_run do not comment workflow_run list items",
			input: `on:
  bots:
    - dependabot
  workflow_run:
    workflows: ["CI"]
    types: [completed]
  workflow_dispatch:`,
			expected: `on:
  # bots: # Bots processed as bot check in pre-activation job
  # - dependabot # Bots processed as bot check in pre-activation job
  workflow_run:
    workflows: ["CI"]
    types: [completed]
  workflow_dispatch:`,
			description: "Should reset bots array tracker before entering workflow_run section",
		},
		{
			name: "bots before workflow_run with multi-line arrays do not comment workflow_run list items",
			input: `on:
  bots:
    - dependabot
  workflow_run:
    workflows:
      - CI
    types:
      - completed
  workflow_dispatch:`,
			expected: `on:
  # bots: # Bots processed as bot check in pre-activation job
  # - dependabot # Bots processed as bot check in pre-activation job
  workflow_run:
    workflows:
      - CI
    types:
      - completed
  workflow_dispatch:`,
			description: "Should not comment out multi-line workflow_run.workflows/types items when bots precedes workflow_run",
		},
		{
			name: "skip-if-check-failing before workflow_run does not corrupt workflow_run list items",
			input: `on:
  skip-if-check-failing:
    - build
  workflow_run:
    workflows:
      - CI
    types:
      - completed
  workflow_dispatch:`,
			expected: `on:
  # skip-if-check-failing: # Skip-if-check-failing processed as check status gate in pre-activation job
  # - build
  workflow_run:
    workflows:
      - CI
    types:
      - completed
  workflow_dispatch:`,
			description: "Should reset inSkipIfCheckFailing before entering workflow_run to prevent list-item corruption",
		},
		{
			name: "roles before workflow_run do not comment workflow_run list items",
			input: `on:
  roles:
    - write
  workflow_run:
    workflows: ["CI"]
    types: [completed]
  workflow_dispatch:`,
			expected: `on:
  # roles: # Roles processed as role check in pre-activation job
  # - write # Roles processed as role check in pre-activation job
  workflow_run:
    workflows: ["CI"]
    types: [completed]
  workflow_dispatch:`,
			description: "Should reset roles array tracker before entering workflow_run section",
		},
		{
			name: "roles all before workflow_run keeps workflow_run intact",
			input: `on:
  roles: all
  workflow_run:
    workflows: ["CI"]
    types: [completed]
  workflow_dispatch:`,
			expected: `on:
  # roles: all # Roles processed as role check in pre-activation job
  workflow_run:
    workflows: ["CI"]
    types: [completed]
  workflow_dispatch:`,
			description: "Should handle inline roles value without affecting workflow_run fields",
		},
		{
			name: "top-level on needs array",
			input: `on:
  needs:
    - study_repo
  schedule:
    - cron: "23 * * * *"
  workflow_dispatch:`,
			expected: `on:
  # needs: # Needs processed as dependency in pre-activation job
  # - study_repo # Needs processed as dependency in pre-activation job
  schedule:
    - cron: "23 * * * *"
  workflow_dispatch:`,
			description: "Should comment out needs in on section after compiler processing",
		},
		{
			name: "top-level on needs array with compact list indentation",
			input: `on:
  needs:
  - study_repo
  schedule:
  - cron: "23 * * * *"
  workflow_dispatch:`,
			expected: `on:
  # needs: # Needs processed as dependency in pre-activation job
  # - study_repo # Needs processed as dependency in pre-activation job
  schedule:
  - cron: "23 * * * *"
  workflow_dispatch:`,
			description: "Should comment out needs list items when emitted with compact indentation",
		},
		{
			name: "top-level on needs inline array",
			input: `on:
  needs: [study_repo, setup]
  schedule:
    - cron: "23 * * * *"
  workflow_dispatch:`,
			expected: `on:
  # needs: [study_repo, setup] # Needs processed as dependency in pre-activation job
  schedule:
    - cron: "23 * * * *"
  workflow_dispatch:`,
			description: "Should comment out needs when emitted as an inline YAML array",
		},
		{
			name: "issues with two-space indentation names",
			input: `on:
  issues:
    names:
      - bug
  workflow_dispatch:`,
			expected: `on:
  issues:
    # names: # Label filtering applied via job conditions
    # - bug # Label filtering applied via job conditions
  workflow_dispatch:`,
			description: "Should comment out names in issues section with two-space indentation style",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := compiler.commentOutProcessedFieldsInOnSection(tt.input, map[string]any{})

			if result != tt.expected {
				t.Errorf("%s\nExpected:\n%s\nGot:\n%s", tt.description, tt.expected, result)
			}
		})
	}
}

func TestCommentOutProcessedFieldsInOnSectionBlankLineInBlock(t *testing.T) {
	compiler := NewCompiler()

	result := compiler.commentOutProcessedFieldsInOnSection(`on:
  steps: |
    echo hello

    echo world
  workflow_dispatch:`, map[string]any{})

	// The blank line inside the commented block is emitted at the block's base
	// indentation (matching "# steps:") with no trailing space, keeping the block in
	// a single comment group for yamllint's comments-indentation rule.
	assert.Contains(t, result, "\n  #\n")
	assert.NotContains(t, result, "#  \n")
	assert.NotContains(t, result, "# \n")
}

func TestCommentOutProcessedFieldsInOnSectionTrailingSpaceOnNonBlankLine(t *testing.T) {
	compiler := NewCompiler()

	// The "echo hello" line intentionally includes trailing spaces.
	result := compiler.commentOutProcessedFieldsInOnSection(`on:
  steps: |
    echo hello   
    echo world
  workflow_dispatch:`, map[string]any{})

	assert.NotContains(t, result, "# echo hello   ")
	assert.Contains(t, result, "# echo hello")
}

// containsInNonCommentLines checks if a string appears in any non-comment lines
// A comment line is one that starts with '#' (after trimming leading whitespace)
