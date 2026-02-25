//go:build integration

package workflow

import (
	"strings"
	"testing"
)

// TestExtractExpressionsFromPlaywrightArgs tests the helper function
func TestExtractExpressionsFromPlaywrightArgs(t *testing.T) {
	tests := []struct {
		name                string
		customArgs          []string
		expectedExpressions int // Number of unique expressions expected
	}{
		{
			name:                "Single expression in custom args",
			customArgs:          []string{"--arg", "${{ secrets.TEST_DOMAIN }}"},
			expectedExpressions: 1,
		},
		{
			name:                "Multiple expressions",
			customArgs:          []string{"--arg", "${{ github.event.issue.number }}", "--secret", "${{ secrets.API_KEY }}"},
			expectedExpressions: 2,
		},
		{
			name:                "No expressions",
			customArgs:          []string{"--static-arg"},
			expectedExpressions: 0,
		},
		{
			name:                "Empty args",
			customArgs:          nil,
			expectedExpressions: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expressions := extractExpressionsFromPlaywrightArgs(tt.customArgs)

			if len(expressions) != tt.expectedExpressions {
				t.Errorf("Expected %d expressions, got %d", tt.expectedExpressions, len(expressions))
			}

			// Verify all expressions are in the map with proper GH_AW_ prefix
			for envVar, originalExpr := range expressions {
				if !strings.HasPrefix(envVar, "GH_AW_") {
					t.Errorf("Expected env var to start with GH_AW_, got %s", envVar)
				}
				if !strings.HasPrefix(originalExpr, "${{") || !strings.HasSuffix(originalExpr, "}}") {
					t.Errorf("Expected original expression to be wrapped in ${{ }}, got %s", originalExpr)
				}
			}
		})
	}
}

// TestReplaceExpressionsInPlaywrightArgs tests the helper function
func TestReplaceExpressionsInPlaywrightArgs(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		expressions map[string]string
		validate    func(t *testing.T, result []string)
	}{
		{
			name: "Replace single expression",
			args: []string{
				"${{ secrets.TEST_DOMAIN }}",
			},
			expressions: map[string]string{
				"GH_AW_SECRETS_TEST_DOMAIN": "${{ secrets.TEST_DOMAIN }}",
			},
			validate: func(t *testing.T, result []string) {
				if len(result) != 1 {
					t.Errorf("Expected 1 result, got %d", len(result))
				}
				if !strings.Contains(result[0], "__GH_AW_") {
					t.Errorf("Expected result to contain __GH_AW_, got %s", result[0])
				}
				if strings.Contains(result[0], "${{ secrets.") {
					t.Errorf("Result should not contain secret expressions, got %s", result[0])
				}
			},
		},
		{
			name: "Replace multiple expressions",
			args: []string{
				"${{ secrets.API_KEY }}",
				"example.com",
				"${{ secrets.ANOTHER_SECRET }}",
			},
			expressions: map[string]string{
				"GH_AW_SECRETS_API_KEY":        "${{ secrets.API_KEY }}",
				"GH_AW_SECRETS_ANOTHER_SECRET": "${{ secrets.ANOTHER_SECRET }}",
			},
			validate: func(t *testing.T, result []string) {
				if len(result) != 3 {
					t.Errorf("Expected 3 results, got %d", len(result))
				}
				// Second element should be unchanged
				if result[1] != "example.com" {
					t.Errorf("Expected example.com to be unchanged, got %s", result[1])
				}
				// First and third should be replaced
				for i, r := range []int{0, 2} {
					if strings.Contains(result[r], "${{ secrets.") {
						t.Errorf("Result[%d] should not contain secret expressions, got %s", i, result[r])
					}
				}
			},
		},
		{
			name: "No expressions to replace",
			args: []string{
				"example.com",
				"test.org",
			},
			expressions: map[string]string{},
			validate: func(t *testing.T, result []string) {
				if len(result) != 2 {
					t.Errorf("Expected 2 results, got %d", len(result))
				}
				if result[0] != "example.com" || result[1] != "test.org" {
					t.Errorf("Expected unchanged results, got %v", result)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := replaceExpressionsInPlaywrightArgs(tt.args, tt.expressions)
			tt.validate(t, result)
		})
	}
}
