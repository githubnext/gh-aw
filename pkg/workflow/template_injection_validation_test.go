package workflow

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateNoTemplateInjection(t *testing.T) {
	tests := []struct {
		name        string
		yaml        string
		shouldError bool
		errorString string
	}{
		{
			name: "safe pattern - expression in env variable",
			yaml: `jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - name: Safe usage
        env:
          ISSUE_TITLE: ${{ github.event.issue.title }}
        run: |
          echo "Title: $ISSUE_TITLE"`,
			shouldError: false,
		},
		{
			name: "safe pattern - no expressions in run block",
			yaml: `jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - name: Safe command
        run: |
          echo "Hello world"
          bash script.sh`,
			shouldError: false,
		},
		{
			name: "safe pattern - safe context expressions",
			yaml: `jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - name: Safe contexts
        run: |
          echo "Actor: ${{ github.actor }}"
          echo "Repository: ${{ github.repository }}"
          echo "SHA: ${{ github.sha }}"`,
			shouldError: false,
		},
		{
			name: "unsafe pattern - github.event in run block",
			yaml: `jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - name: Unsafe usage
        run: |
          echo "Issue: ${{ github.event.issue.title }}"`,
			shouldError: true,
			errorString: "template injection",
		},
		{
			name: "unsafe pattern - steps.outputs in run block",
			yaml: `jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - name: Unsafe usage
        run: |
          bash /opt/gh-aw/actions/stop_mcp_gateway.sh ${{ steps.start-mcp-gateway.outputs.gateway-pid }}`,
			shouldError: true,
			errorString: "steps.*.outputs",
		},
		{
			name: "unsafe pattern - inputs in run block",
			yaml: `jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - name: Unsafe usage
        run: |
          echo "Input: ${{ inputs.user_data }}"`,
			shouldError: true,
			errorString: "workflow inputs",
		},
		{
			name: "unsafe pattern - multiple violations",
			yaml: `jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - name: Multiple unsafe patterns
        run: |
          echo "Title: ${{ github.event.issue.title }}"
          echo "Body: ${{ github.event.issue.body }}"
          bash script.sh ${{ steps.foo.outputs.bar }}`,
			shouldError: true,
			errorString: "template injection",
		},
		{
			name: "unsafe pattern - single line run command",
			yaml: `jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - name: Single line unsafe
        run: echo "PR title: ${{ github.event.pull_request.title }}"`,
			shouldError: true,
			errorString: "github.event",
		},
		{
			name: "safe pattern - expression in condition",
			yaml: `jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - name: Conditional step
        if: github.event.issue.title == 'test'
        run: |
          echo "Running conditional step"`,
			shouldError: false,
		},
		{
			name: "unsafe pattern - github.event.comment",
			yaml: `jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - name: Process comment
        run: |
          comment="${{ github.event.comment.body }}"
          echo "$comment"`,
			shouldError: true,
			errorString: "github.event",
		},
		{
			name: "unsafe pattern - github.event.pull_request",
			yaml: `jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - name: Process PR
        run: |
          title="${{ github.event.pull_request.title }}"
          body="${{ github.event.pull_request.body }}"`,
			shouldError: true,
			errorString: "github.event",
		},
		{
			name: "safe pattern - mixed safe and env usage",
			yaml: `jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - name: Mixed safe usage
        env:
          TITLE: ${{ github.event.issue.title }}
          ACTOR: ${{ github.actor }}
        run: |
          echo "Title: $TITLE"
          echo "Actor: $ACTOR"
          echo "SHA: ${{ github.sha }}"`,
			shouldError: false,
		},
		{
			name: "unsafe pattern - github.head_ref in run",
			yaml: `jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - name: Branch name
        run: |
          echo "Branch: ${{ github.head_ref }}"`,
			shouldError: false, // head_ref is not in our unsafe list (it's in env vars already in real workflows)
		},
		{
			name: "complex unsafe pattern - nested in script",
			yaml: `jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - name: Complex unsafe
        run: |
          if [ -n "${{ github.event.issue.number }}" ]; then
            curl -X POST "https://api.github.com/repos/owner/repo/issues/${{ github.event.issue.number }}/comments"
          fi`,
			shouldError: true,
			errorString: "github.event",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateNoTemplateInjection(tt.yaml)

			if tt.shouldError {
				require.Error(t, err, "Expected validation to fail but it passed")
				if tt.errorString != "" {
					assert.Contains(t, err.Error(), tt.errorString,
						"Error message should contain expected string")
				}
				// Verify error message quality
				assert.Contains(t, err.Error(), "template injection",
					"Error should mention template injection")
				assert.Contains(t, err.Error(), "Safe Pattern",
					"Error should provide safe pattern example")
			} else {
				assert.NoError(t, err, "Expected validation to pass but got error: %v", err)
			}
		})
	}
}

func TestTemplateInjectionErrorMessageQuality(t *testing.T) {
	// Test that error messages are helpful and actionable
	yaml := `jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - name: Test step
        run: echo "${{ github.event.issue.title }}"
      - name: Another step
        run: bash script.sh ${{ steps.foo.outputs.bar }}`

	err := validateNoTemplateInjection(yaml)
	require.Error(t, err, "Should detect template injection")

	errMsg := err.Error()

	// Check for key components of a good error message
	t.Run("mentions security risk", func(t *testing.T) {
		assert.Contains(t, errMsg, "Security Risk",
			"Error should explain the security implications")
	})

	t.Run("shows safe pattern", func(t *testing.T) {
		assert.Contains(t, errMsg, "Safe Pattern",
			"Error should show the correct way to do it")
		assert.Contains(t, errMsg, "env:",
			"Safe pattern should mention env variables")
	})

	t.Run("shows unsafe pattern", func(t *testing.T) {
		assert.Contains(t, errMsg, "Unsafe Pattern",
			"Error should show what NOT to do")
	})

	t.Run("provides references", func(t *testing.T) {
		assert.Contains(t, errMsg, "References",
			"Error should link to documentation")
		assert.Contains(t, errMsg, "security-hardening-for-github-actions",
			"Should link to GitHub security docs")
		assert.Contains(t, errMsg, "zizmor",
			"Should reference zizmor tool")
	})

	t.Run("groups by context", func(t *testing.T) {
		assert.Contains(t, errMsg, "github.event",
			"Should identify github.event context")
		assert.Contains(t, errMsg, "steps.*.outputs",
			"Should identify steps outputs context")
	})
}

func TestExtractRunSnippet(t *testing.T) {
	tests := []struct {
		name       string
		runContent string
		expression string
		want       string
	}{
		{
			name: "simple one-line",
			runContent: `  echo "Title: ${{ github.event.issue.title }}"
  echo "Done"`,
			expression: "${{ github.event.issue.title }}",
			want:       `echo "Title: ${{ github.event.issue.title }}"`,
		},
		{
			name: "multiline with indentation",
			runContent: `  if [ -n "${{ github.event.issue.number }}" ]; then
    echo "Processing"
  fi`,
			expression: "${{ github.event.issue.number }}",
			want:       `if [ -n "${{ github.event.issue.number }}" ]; then`,
		},
		{
			name:       "long line truncation",
			runContent: "  " + strings.Repeat("x", 120) + " ${{ github.event.issue.title }}",
			expression: "${{ github.event.issue.title }}",
			want:       strings.Repeat("x", 97) + "...",
		},
		{
			name:       "expression not found",
			runContent: `  echo "Hello"`,
			expression: "${{ github.event.issue.title }}",
			want:       "${{ github.event.issue.title }}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractRunSnippet(tt.runContent, tt.expression)
			assert.Equal(t, tt.want, got,
				"Snippet extraction should match expected output")
		})
	}
}

func TestDetectExpressionContext(t *testing.T) {
	tests := []struct {
		expression string
		want       string
	}{
		{
			expression: "${{ github.event.issue.title }}",
			want:       "github.event",
		},
		{
			expression: "${{ github.event.pull_request.body }}",
			want:       "github.event",
		},
		{
			expression: "${{ steps.foo.outputs.bar }}",
			want:       "steps.*.outputs",
		},
		{
			expression: "${{ steps.start-mcp-gateway.outputs.gateway-pid }}",
			want:       "steps.*.outputs",
		},
		{
			expression: "${{ inputs.user_data }}",
			want:       "workflow inputs",
		},
		{
			expression: "${{ github.actor }}",
			want:       "unknown context",
		},
	}

	for _, tt := range tests {
		t.Run(tt.expression, func(t *testing.T) {
			got := detectExpressionContext(tt.expression)
			assert.Equal(t, tt.want, got,
				"Context detection should correctly identify expression type")
		})
	}
}

func TestTemplateInjectionRealWorldPatterns(t *testing.T) {
	// Test patterns found in real workflows from the problem statement
	t.Run("stop_mcp_gateway pattern", func(t *testing.T) {
		yaml := `jobs:
  agent:
    steps:
      - name: Stop MCP gateway
        if: always()
        continue-on-error: true
        env:
          MCP_GATEWAY_PORT: ${{ steps.start-mcp-gateway.outputs.gateway-port }}
          MCP_GATEWAY_API_KEY: ${{ steps.start-mcp-gateway.outputs.gateway-api-key }}
        run: |
          bash /opt/gh-aw/actions/stop_mcp_gateway.sh ${{ steps.start-mcp-gateway.outputs.gateway-pid }}`

		err := validateNoTemplateInjection(yaml)
		require.Error(t, err, "Should detect unsafe gateway-pid usage in run command")
		assert.Contains(t, err.Error(), "steps.*.outputs",
			"Should identify as steps.outputs context")
		assert.Contains(t, err.Error(), "gateway-pid",
			"Error should mention the specific expression")
	})

	t.Run("safe version of stop_mcp_gateway", func(t *testing.T) {
		yaml := `jobs:
  agent:
    steps:
      - name: Stop MCP gateway
        if: always()
        continue-on-error: true
        env:
          MCP_GATEWAY_PORT: ${{ steps.start-mcp-gateway.outputs.gateway-port }}
          MCP_GATEWAY_API_KEY: ${{ steps.start-mcp-gateway.outputs.gateway-api-key }}
          GATEWAY_PID: ${{ steps.start-mcp-gateway.outputs.gateway-pid }}
        run: |
          bash /opt/gh-aw/actions/stop_mcp_gateway.sh "$GATEWAY_PID"`

		err := validateNoTemplateInjection(yaml)
		assert.NoError(t, err, "Should pass with gateway-pid in env variable")
	})
}
