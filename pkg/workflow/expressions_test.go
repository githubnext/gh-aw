package workflow

import (
	"strings"
	"testing"
)

func TestExpressionNode_Render(t *testing.T) {
	expr := &ExpressionNode{Expression: "github.event_name == 'issues'"}
	expected := "github.event_name == 'issues'"
	if result := expr.Render(); result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

func TestAndNode_Render(t *testing.T) {
	left := &ExpressionNode{Expression: "condition1"}
	right := &ExpressionNode{Expression: "condition2"}
	andNode := &AndNode{Left: left, Right: right}

	expected := "(condition1) && (condition2)"
	if result := andNode.Render(); result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

func TestOrNode_Render(t *testing.T) {
	left := &ExpressionNode{Expression: "condition1"}
	right := &ExpressionNode{Expression: "condition2"}
	orNode := &OrNode{Left: left, Right: right}

	expected := "(condition1) || (condition2)"
	if result := orNode.Render(); result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

func TestNotNode_Render(t *testing.T) {
	child := &ExpressionNode{Expression: "github.event_name == 'issues'"}
	notNode := &NotNode{Child: child}

	expected := "!(github.event_name == 'issues')"
	if result := notNode.Render(); result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

func TestDisjunctionNode_Render(t *testing.T) {
	tests := []struct {
		name     string
		terms    []ConditionNode
		expected string
	}{
		{
			name:     "empty terms",
			terms:    []ConditionNode{},
			expected: "",
		},
		{
			name: "single term",
			terms: []ConditionNode{
				&ExpressionNode{Expression: "condition1"},
			},
			expected: "condition1",
		},
		{
			name: "two terms",
			terms: []ConditionNode{
				&ExpressionNode{Expression: "condition1"},
				&ExpressionNode{Expression: "condition2"},
			},
			expected: "condition1 || condition2",
		},
		{
			name: "multiple terms",
			terms: []ConditionNode{
				&ExpressionNode{Expression: "github.event_name == 'issues'"},
				&ExpressionNode{Expression: "github.event_name == 'pull_request'"},
				&ExpressionNode{Expression: "github.event_name == 'issue_comment'"},
			},
			expected: "github.event_name == 'issues' || github.event_name == 'pull_request' || github.event_name == 'issue_comment'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			disjunctionNode := &DisjunctionNode{Terms: tt.terms}
			if result := disjunctionNode.Render(); result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestComplexExpressionTree(t *testing.T) {
	// Test: (condition1 && condition2) || !(condition3)
	condition1 := &ExpressionNode{Expression: "github.event_name == 'issues'"}
	condition2 := &ExpressionNode{Expression: "github.event.action == 'opened'"}
	condition3 := &ExpressionNode{Expression: "github.event.pull_request.draft == true"}

	andNode := &AndNode{Left: condition1, Right: condition2}
	notNode := &NotNode{Child: condition3}
	orNode := &OrNode{Left: andNode, Right: notNode}

	expected := "((github.event_name == 'issues') && (github.event.action == 'opened')) || (!(github.event.pull_request.draft == true))"
	if result := orNode.Render(); result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

func TestBuildConditionTree(t *testing.T) {
	tests := []struct {
		name              string
		existingCondition string
		draftCondition    string
		expectedPattern   string
	}{
		{
			name:              "empty existing condition",
			existingCondition: "",
			draftCondition:    "draft_condition",
			expectedPattern:   "draft_condition",
		},
		{
			name:              "both conditions present",
			existingCondition: "existing_condition",
			draftCondition:    "draft_condition",
			expectedPattern:   "(existing_condition) && (draft_condition)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildConditionTree(tt.existingCondition, tt.draftCondition)
			if rendered := result.Render(); rendered != tt.expectedPattern {
				t.Errorf("Expected '%s', got '%s'", tt.expectedPattern, rendered)
			}
		})
	}
}

func TestBuildReactionCondition(t *testing.T) {
	result := buildReactionCondition()
	rendered := result.Render()

	// The result should be a flat OR chain without deep nesting
	expectedSubstrings := []string{
		"github.event_name == 'issues'",
		"github.event_name == 'issue_comment'",
		"github.event_name == 'pull_request_comment'",
		"github.event_name == 'pull_request_review_comment'",
		"github.event_name == 'pull_request'",
		"github.event.pull_request.head.repo.full_name == github.repository",
		"&&",
		"||",
	}

	for _, substr := range expectedSubstrings {
		if !strings.Contains(rendered, substr) {
			t.Errorf("Expected rendered condition to contain '%s', but got: %s", substr, rendered)
		}
	}

	// With the fork check, the pull_request condition should be more complex
	// It should contain both the event name check and the not-from-fork check
	if !strings.Contains(rendered, "(github.event_name == 'pull_request') && (github.event.pull_request.head.repo.full_name == github.repository)") {
		t.Errorf("Expected pull_request condition to include fork check, but got: %s", rendered)
	}
}

func TestFunctionCallNode_Render(t *testing.T) {
	tests := []struct {
		name     string
		function string
		args     []ConditionNode
		expected string
	}{
		{
			name:     "contains function with two arguments",
			function: "contains",
			args: []ConditionNode{
				&PropertyAccessNode{PropertyPath: "github.event.issue.labels"},
				&StringLiteralNode{Value: "bug"},
			},
			expected: "contains(github.event.issue.labels, 'bug')",
		},
		{
			name:     "startsWith function",
			function: "startsWith",
			args: []ConditionNode{
				&PropertyAccessNode{PropertyPath: "github.ref"},
				&StringLiteralNode{Value: "refs/heads/"},
			},
			expected: "startsWith(github.ref, 'refs/heads/')",
		},
		{
			name:     "function with no arguments",
			function: "always",
			args:     []ConditionNode{},
			expected: "always()",
		},
		{
			name:     "function with multiple arguments",
			function: "format",
			args: []ConditionNode{
				&StringLiteralNode{Value: "Hello {0}"},
				&PropertyAccessNode{PropertyPath: "github.actor"},
			},
			expected: "format('Hello {0}', github.actor)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node := &FunctionCallNode{
				FunctionName: tt.function,
				Arguments:    tt.args,
			}
			if result := node.Render(); result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestPropertyAccessNode_Render(t *testing.T) {
	tests := []struct {
		name     string
		property string
		expected string
	}{
		{
			name:     "simple property",
			property: "github.actor",
			expected: "github.actor",
		},
		{
			name:     "nested property",
			property: "github.event.issue.number",
			expected: "github.event.issue.number",
		},
		{
			name:     "deep nested property",
			property: "github.event.pull_request.head.sha",
			expected: "github.event.pull_request.head.sha",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node := &PropertyAccessNode{PropertyPath: tt.property}
			if result := node.Render(); result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestStringLiteralNode_Render(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		expected string
	}{
		{
			name:     "simple string",
			value:    "hello",
			expected: "'hello'",
		},
		{
			name:     "string with spaces",
			value:    "hello world",
			expected: "'hello world'",
		},
		{
			name:     "empty string",
			value:    "",
			expected: "''",
		},
		{
			name:     "string with special characters",
			value:    "issue-123",
			expected: "'issue-123'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node := &StringLiteralNode{Value: tt.value}
			if result := node.Render(); result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestBooleanLiteralNode_Render(t *testing.T) {
	tests := []struct {
		name     string
		value    bool
		expected string
	}{
		{
			name:     "true value",
			value:    true,
			expected: "true",
		},
		{
			name:     "false value",
			value:    false,
			expected: "false",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node := &BooleanLiteralNode{Value: tt.value}
			if result := node.Render(); result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestNumberLiteralNode_Render(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		expected string
	}{
		{
			name:     "integer",
			value:    "42",
			expected: "42",
		},
		{
			name:     "decimal",
			value:    "3.14",
			expected: "3.14",
		},
		{
			name:     "negative number",
			value:    "-10",
			expected: "-10",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node := &NumberLiteralNode{Value: tt.value}
			if result := node.Render(); result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestComparisonNode_Render(t *testing.T) {
	tests := []struct {
		name     string
		left     ConditionNode
		operator string
		right    ConditionNode
		expected string
	}{
		{
			name:     "equality comparison",
			left:     &PropertyAccessNode{PropertyPath: "github.event.action"},
			operator: "==",
			right:    &StringLiteralNode{Value: "opened"},
			expected: "github.event.action == 'opened'",
		},
		{
			name:     "inequality comparison",
			left:     &PropertyAccessNode{PropertyPath: "github.event.issue.number"},
			operator: "!=",
			right:    &NumberLiteralNode{Value: "0"},
			expected: "github.event.issue.number != 0",
		},
		{
			name:     "greater than comparison",
			left:     &PropertyAccessNode{PropertyPath: "github.event.issue.comments"},
			operator: ">",
			right:    &NumberLiteralNode{Value: "5"},
			expected: "github.event.issue.comments > 5",
		},
		{
			name:     "less than or equal comparison",
			left:     &PropertyAccessNode{PropertyPath: "github.run_number"},
			operator: "<=",
			right:    &NumberLiteralNode{Value: "100"},
			expected: "github.run_number <= 100",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node := &ComparisonNode{
				Left:     tt.left,
				Operator: tt.operator,
				Right:    tt.right,
			}
			if result := node.Render(); result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestTernaryNode_Render(t *testing.T) {
	tests := []struct {
		name       string
		condition  ConditionNode
		trueValue  ConditionNode
		falseValue ConditionNode
		expected   string
	}{
		{
			name: "simple ternary",
			condition: &ComparisonNode{
				Left:     &PropertyAccessNode{PropertyPath: "github.event.action"},
				Operator: "==",
				Right:    &StringLiteralNode{Value: "opened"},
			},
			trueValue:  &StringLiteralNode{Value: "new"},
			falseValue: &StringLiteralNode{Value: "existing"},
			expected:   "github.event.action == 'opened' ? 'new' : 'existing'",
		},
		{
			name:       "ternary with boolean literals",
			condition:  &PropertyAccessNode{PropertyPath: "github.event.pull_request.draft"},
			trueValue:  &StringLiteralNode{Value: "draft"},
			falseValue: &StringLiteralNode{Value: "ready"},
			expected:   "github.event.pull_request.draft ? 'draft' : 'ready'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node := &TernaryNode{
				Condition:  tt.condition,
				TrueValue:  tt.trueValue,
				FalseValue: tt.falseValue,
			}
			if result := node.Render(); result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestContainsNode_Render(t *testing.T) {
	tests := []struct {
		name     string
		array    ConditionNode
		value    ConditionNode
		expected string
	}{
		{
			name:     "contains with property and string",
			array:    &PropertyAccessNode{PropertyPath: "github.event.issue.labels"},
			value:    &StringLiteralNode{Value: "bug"},
			expected: "contains(github.event.issue.labels, 'bug')",
		},
		{
			name:     "contains with nested property",
			array:    &PropertyAccessNode{PropertyPath: "github.event.pull_request.requested_reviewers"},
			value:    &PropertyAccessNode{PropertyPath: "github.actor"},
			expected: "contains(github.event.pull_request.requested_reviewers, github.actor)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node := &ContainsNode{
				Array: tt.array,
				Value: tt.value,
			}
			if result := node.Render(); result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

// TestGitHubActionsArrayMatching tests the specific array matching technique mentioned in the issue
func TestGitHubActionsArrayMatching(t *testing.T) {
	// Test the array matching pattern from GitHub Actions docs
	// Example: contains(github.event.issue.labels.*.name, 'bug')
	tests := []struct {
		name     string
		pattern  ConditionNode
		expected string
	}{
		{
			name: "label matching with contains",
			pattern: &ContainsNode{
				Array: &PropertyAccessNode{PropertyPath: "github.event.issue.labels.*.name"},
				Value: &StringLiteralNode{Value: "bug"},
			},
			expected: "contains(github.event.issue.labels.*.name, 'bug')",
		},
		{
			name: "multiple label matching with OR",
			pattern: &OrNode{
				Left: &ContainsNode{
					Array: &PropertyAccessNode{PropertyPath: "github.event.issue.labels.*.name"},
					Value: &StringLiteralNode{Value: "bug"},
				},
				Right: &ContainsNode{
					Array: &PropertyAccessNode{PropertyPath: "github.event.issue.labels.*.name"},
					Value: &StringLiteralNode{Value: "enhancement"},
				},
			},
			expected: "(contains(github.event.issue.labels.*.name, 'bug')) || (contains(github.event.issue.labels.*.name, 'enhancement'))",
		},
		{
			name: "complex array matching with conditions",
			pattern: &AndNode{
				Left: &ContainsNode{
					Array: &PropertyAccessNode{PropertyPath: "github.event.issue.labels.*.name"},
					Value: &StringLiteralNode{Value: "priority-high"},
				},
				Right: &ComparisonNode{
					Left:     &PropertyAccessNode{PropertyPath: "github.event.action"},
					Operator: "==",
					Right:    &StringLiteralNode{Value: "opened"},
				},
			},
			expected: "(contains(github.event.issue.labels.*.name, 'priority-high')) && (github.event.action == 'opened')",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if result := tt.pattern.Render(); result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

// TestComplexGitHubActionsExpressions tests complex real-world GitHub Actions expressions
func TestComplexGitHubActionsExpressions(t *testing.T) {
	tests := []struct {
		name       string
		expression ConditionNode
		expected   string
	}{
		{
			name: "conditional workflow run based on labels and action",
			expression: &AndNode{
				Left: &OrNode{
					Left: &ComparisonNode{
						Left:     &PropertyAccessNode{PropertyPath: "github.event.action"},
						Operator: "==",
						Right:    &StringLiteralNode{Value: "opened"},
					},
					Right: &ComparisonNode{
						Left:     &PropertyAccessNode{PropertyPath: "github.event.action"},
						Operator: "==",
						Right:    &StringLiteralNode{Value: "synchronize"},
					},
				},
				Right: &ContainsNode{
					Array: &PropertyAccessNode{PropertyPath: "github.event.pull_request.labels.*.name"},
					Value: &StringLiteralNode{Value: "auto-deploy"},
				},
			},
			expected: "((github.event.action == 'opened') || (github.event.action == 'synchronize')) && (contains(github.event.pull_request.labels.*.name, 'auto-deploy'))",
		},
		{
			name: "ternary expression for environment selection",
			expression: &TernaryNode{
				Condition: &FunctionCallNode{
					FunctionName: "startsWith",
					Arguments: []ConditionNode{
						&PropertyAccessNode{PropertyPath: "github.ref"},
						&StringLiteralNode{Value: "refs/heads/main"},
					},
				},
				TrueValue:  &StringLiteralNode{Value: "production"},
				FalseValue: &StringLiteralNode{Value: "staging"},
			},
			expected: "startsWith(github.ref, 'refs/heads/main') ? 'production' : 'staging'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if result := tt.expression.Render(); result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

// TestHelperFunctions tests the helper functions for building expressions
func TestHelperFunctions(t *testing.T) {
	t.Run("BuildPropertyAccess", func(t *testing.T) {
		node := BuildPropertyAccess("github.event.action")
		expected := "github.event.action"
		if result := node.Render(); result != expected {
			t.Errorf("Expected '%s', got '%s'", expected, result)
		}
	})

	t.Run("BuildStringLiteral", func(t *testing.T) {
		node := BuildStringLiteral("opened")
		expected := "'opened'"
		if result := node.Render(); result != expected {
			t.Errorf("Expected '%s', got '%s'", expected, result)
		}
	})

	t.Run("BuildBooleanLiteral", func(t *testing.T) {
		node := BuildBooleanLiteral(true)
		expected := "true"
		if result := node.Render(); result != expected {
			t.Errorf("Expected '%s', got '%s'", expected, result)
		}
	})

	t.Run("BuildNumberLiteral", func(t *testing.T) {
		node := BuildNumberLiteral("42")
		expected := "42"
		if result := node.Render(); result != expected {
			t.Errorf("Expected '%s', got '%s'", expected, result)
		}
	})

	t.Run("BuildEquals", func(t *testing.T) {
		node := BuildEquals(
			BuildPropertyAccess("github.event.action"),
			BuildStringLiteral("opened"),
		)
		expected := "github.event.action == 'opened'"
		if result := node.Render(); result != expected {
			t.Errorf("Expected '%s', got '%s'", expected, result)
		}
	})

	t.Run("BuildNotEquals", func(t *testing.T) {
		node := BuildNotEquals(
			BuildPropertyAccess("github.event.issue.number"),
			BuildNumberLiteral("0"),
		)
		expected := "github.event.issue.number != 0"
		if result := node.Render(); result != expected {
			t.Errorf("Expected '%s', got '%s'", expected, result)
		}
	})

	t.Run("BuildContains", func(t *testing.T) {
		node := BuildContains(
			BuildPropertyAccess("github.event.issue.labels.*.name"),
			BuildStringLiteral("bug"),
		)
		expected := "contains(github.event.issue.labels.*.name, 'bug')"
		if result := node.Render(); result != expected {
			t.Errorf("Expected '%s', got '%s'", expected, result)
		}
	})

	t.Run("BuildFunctionCall", func(t *testing.T) {
		node := BuildFunctionCall("startsWith",
			BuildPropertyAccess("github.ref"),
			BuildStringLiteral("refs/heads/"),
		)
		expected := "startsWith(github.ref, 'refs/heads/')"
		if result := node.Render(); result != expected {
			t.Errorf("Expected '%s', got '%s'", expected, result)
		}
	})

	t.Run("BuildTernary", func(t *testing.T) {
		node := BuildTernary(
			BuildEquals(BuildPropertyAccess("github.event.action"), BuildStringLiteral("opened")),
			BuildStringLiteral("new"),
			BuildStringLiteral("existing"),
		)
		expected := "github.event.action == 'opened' ? 'new' : 'existing'"
		if result := node.Render(); result != expected {
			t.Errorf("Expected '%s', got '%s'", expected, result)
		}
	})
}

// TestConvenienceHelpers tests the convenience helper functions
func TestConvenienceHelpers(t *testing.T) {
	t.Run("BuildLabelContains", func(t *testing.T) {
		node := BuildLabelContains("bug")
		expected := "contains(github.event.issue.labels.*.name, 'bug')"
		if result := node.Render(); result != expected {
			t.Errorf("Expected '%s', got '%s'", expected, result)
		}
	})

	t.Run("BuildActionEquals", func(t *testing.T) {
		node := BuildActionEquals("opened")
		expected := "github.event.action == 'opened'"
		if result := node.Render(); result != expected {
			t.Errorf("Expected '%s', got '%s'", expected, result)
		}
	})

	t.Run("BuildEventTypeEquals", func(t *testing.T) {
		node := BuildEventTypeEquals("push")
		expected := "github.event_name == 'push'"
		if result := node.Render(); result != expected {
			t.Errorf("Expected '%s', got '%s'", expected, result)
		}
	})

	t.Run("BuildRefStartsWith", func(t *testing.T) {
		node := BuildRefStartsWith("refs/heads/main")
		expected := "startsWith(github.ref, 'refs/heads/main')"
		if result := node.Render(); result != expected {
			t.Errorf("Expected '%s', got '%s'", expected, result)
		}
	})
}

// TestRealWorldExpressionPatterns tests common expression patterns used in GitHub Actions
func TestRealWorldExpressionPatterns(t *testing.T) {
	tests := []struct {
		name       string
		expression ConditionNode
		expected   string
	}{
		{
			name: "run on main branch only",
			expression: BuildEquals(
				BuildPropertyAccess("github.ref"),
				BuildStringLiteral("refs/heads/main"),
			),
			expected: "github.ref == 'refs/heads/main'",
		},
		{
			name: "run on PR with specific label",
			expression: &AndNode{
				Left:  BuildEventTypeEquals("pull_request"),
				Right: BuildLabelContains("deploy"),
			},
			expected: "(github.event_name == 'pull_request') && (contains(github.event.issue.labels.*.name, 'deploy'))",
		},
		{
			name: "skip draft PRs",
			expression: &AndNode{
				Left: BuildEventTypeEquals("pull_request"),
				Right: &NotNode{
					Child: BuildPropertyAccess("github.event.pull_request.draft"),
				},
			},
			expected: "(github.event_name == 'pull_request') && (!(github.event.pull_request.draft))",
		},
		{
			name: "conditional deployment environment",
			expression: BuildTernary(
				BuildRefStartsWith("refs/heads/main"),
				BuildStringLiteral("production"),
				BuildStringLiteral("staging"),
			),
			expected: "startsWith(github.ref, 'refs/heads/main') ? 'production' : 'staging'",
		},
		{
			name: "run on multiple event actions",
			expression: &DisjunctionNode{
				Terms: []ConditionNode{
					BuildActionEquals("opened"),
					BuildActionEquals("synchronize"),
					BuildActionEquals("reopened"),
				},
			},
			expected: "github.event.action == 'opened' || github.event.action == 'synchronize' || github.event.action == 'reopened'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if result := tt.expression.Render(); result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

// TestExpressionNodeWithDescription tests ExpressionNode with description field
func TestExpressionNodeWithDescription(t *testing.T) {
	tests := []struct {
		name        string
		expression  string
		description string
		expected    string
	}{
		{
			name:        "expression without description",
			expression:  "github.event_name == 'issues'",
			description: "",
			expected:    "github.event_name == 'issues'",
		},
		{
			name:        "expression with description",
			expression:  "github.event_name == 'issues'",
			description: "Check if this is an issue event",
			expected:    "github.event_name == 'issues'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expr := &ExpressionNode{
				Expression:  tt.expression,
				Description: tt.description,
			}
			if result := expr.Render(); result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

// TestDisjunctionNodeMultiline tests multiline rendering functionality
func TestDisjunctionNodeMultiline(t *testing.T) {
	tests := []struct {
		name      string
		terms     []ConditionNode
		multiline bool
		expected  string
	}{
		{
			name: "single line rendering (default)",
			terms: []ConditionNode{
				&ExpressionNode{Expression: "github.event_name == 'issues'", Description: "Check if this is an issue event"},
				&ExpressionNode{Expression: "github.event_name == 'pull_request'", Description: "Check if this is a pull request event"},
			},
			multiline: false,
			expected:  "github.event_name == 'issues' || github.event_name == 'pull_request'",
		},
		{
			name: "multiline rendering with comments",
			terms: []ConditionNode{
				&ExpressionNode{Expression: "github.event_name == 'issues'", Description: "Check if this is an issue event"},
				&ExpressionNode{Expression: "github.event_name == 'pull_request'", Description: "Check if this is a pull request event"},
			},
			multiline: true,
			expected:  "# Check if this is an issue event\ngithub.event_name == 'issues' ||\n# Check if this is a pull request event\ngithub.event_name == 'pull_request'",
		},
		{
			name: "multiline rendering without comments",
			terms: []ConditionNode{
				&ExpressionNode{Expression: "github.event_name == 'issues'"},
				&ExpressionNode{Expression: "github.event_name == 'pull_request'"},
			},
			multiline: true,
			expected:  "github.event_name == 'issues' ||\ngithub.event_name == 'pull_request'",
		},
		{
			name: "multiline rendering with mixed comment presence",
			terms: []ConditionNode{
				&ExpressionNode{Expression: "github.event_name == 'issues'", Description: "Check if this is an issue event"},
				&ExpressionNode{Expression: "github.event_name == 'pull_request'"},
				&ExpressionNode{Expression: "github.event_name == 'issue_comment'", Description: "Check if this is an issue comment event"},
			},
			multiline: true,
			expected:  "# Check if this is an issue event\ngithub.event_name == 'issues' ||\ngithub.event_name == 'pull_request' ||\n# Check if this is an issue comment event\ngithub.event_name == 'issue_comment'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			disjunctionNode := &DisjunctionNode{
				Terms:     tt.terms,
				Multiline: tt.multiline,
			}
			if result := disjunctionNode.Render(); result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

// TestRenderMultilineMethod tests the RenderMultiline method directly
func TestRenderMultilineMethod(t *testing.T) {
	tests := []struct {
		name     string
		terms    []ConditionNode
		expected string
	}{
		{
			name:     "empty terms",
			terms:    []ConditionNode{},
			expected: "",
		},
		{
			name: "single term",
			terms: []ConditionNode{
				&ExpressionNode{Expression: "condition1", Description: "First condition"},
			},
			expected: "condition1",
		},
		{
			name: "multiple terms with comments",
			terms: []ConditionNode{
				&ExpressionNode{Expression: "github.event_name == 'issues'", Description: "Handle issue events"},
				&ExpressionNode{Expression: "github.event_name == 'pull_request'", Description: "Handle PR events"},
				&ExpressionNode{Expression: "github.event_name == 'issue_comment'", Description: "Handle comment events"},
			},
			expected: "# Handle issue events\ngithub.event_name == 'issues' ||\n# Handle PR events\ngithub.event_name == 'pull_request' ||\n# Handle comment events\ngithub.event_name == 'issue_comment'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			disjunctionNode := &DisjunctionNode{Terms: tt.terms}
			if result := disjunctionNode.RenderMultiline(); result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

// TestHelperFunctionsForMultiline tests the new helper functions
func TestHelperFunctionsForMultiline(t *testing.T) {
	t.Run("BuildExpressionWithDescription", func(t *testing.T) {
		expr := BuildExpressionWithDescription("github.event_name == 'issues'", "Check if this is an issue event")

		expected := "github.event_name == 'issues'"
		if result := expr.Render(); result != expected {
			t.Errorf("Expected '%s', got '%s'", expected, result)
		}

		if expr.Description != "Check if this is an issue event" {
			t.Errorf("Expected description 'Check if this is an issue event', got '%s'", expr.Description)
		}
	})

	t.Run("BuildMultilineDisjunction", func(t *testing.T) {
		term1 := BuildExpressionWithDescription("github.event_name == 'issues'", "Handle issue events")
		term2 := BuildExpressionWithDescription("github.event_name == 'pull_request'", "Handle PR events")

		disjunction := BuildMultilineDisjunction(term1, term2)

		if !disjunction.Multiline {
			t.Error("Expected Multiline to be true")
		}

		expected := "# Handle issue events\ngithub.event_name == 'issues' ||\n# Handle PR events\ngithub.event_name == 'pull_request'"
		if result := disjunction.Render(); result != expected {
			t.Errorf("Expected '%s', got '%s'", expected, result)
		}
	})
}

func TestBuildNotFromFork(t *testing.T) {
	result := BuildNotFromFork()
	rendered := result.Render()

	expected := "github.event.pull_request.head.repo.full_name == github.repository"
	if rendered != expected {
		t.Errorf("Expected '%s', got '%s'", expected, rendered)
	}
}
