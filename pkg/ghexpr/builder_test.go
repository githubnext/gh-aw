//go:build !integration

package ghexpr_test

import (
	"testing"

	"github.com/github/gh-aw/pkg/ghexpr"
	"github.com/stretchr/testify/assert"
)

// TestBuildBasicNodes verifies that all primitive builders produce the expected Render output.
func TestBuildBasicNodes(t *testing.T) {
	assert.Equal(t, "a.b.c", ghexpr.BuildPropertyAccess("a.b.c").Render())
	assert.Equal(t, "'hello'", ghexpr.BuildStringLiteral("hello").Render())
	assert.Equal(t, "'it''s'", ghexpr.BuildStringLiteral("it's").Render(), "embedded single quote must be doubled")
	assert.Equal(t, "true", ghexpr.BuildBooleanLiteral(true).Render())
	assert.Equal(t, "false", ghexpr.BuildBooleanLiteral(false).Render())
	assert.Equal(t, "null", ghexpr.BuildNullLiteral().Render())
}

// TestBuildComparison verifies comparison builders.
func TestBuildComparison(t *testing.T) {
	left := ghexpr.BuildPropertyAccess("github.event_name")
	right := ghexpr.BuildStringLiteral("push")

	assert.Equal(t, "github.event_name == 'push'", ghexpr.BuildEquals(left, right).Render())
	assert.Equal(t, "github.event_name != 'push'", ghexpr.BuildNotEquals(left, right).Render())
	assert.Equal(t, "github.event_name < 'push'", ghexpr.BuildComparison(left, "<", right).Render())
}

// TestBuildFunctionCall verifies function-call builder.
func TestBuildFunctionCall(t *testing.T) {
	node := ghexpr.BuildFunctionCall("contains",
		ghexpr.BuildPropertyAccess("github.event.label.name"),
		ghexpr.BuildStringLiteral("deploy"),
	)
	assert.Equal(t, "contains(github.event.label.name, 'deploy')", node.Render())
}

// TestBuildLogical verifies AND / OR builders.
func TestBuildLogical(t *testing.T) {
	a := ghexpr.BuildPropertyAccess("a")
	b := ghexpr.BuildPropertyAccess("b")

	// PropertyAccessNode operands of AND don't need parentheses.
	assert.Equal(t, "a && b", ghexpr.BuildAnd(a, b).Render())
	assert.Equal(t, "a || b", ghexpr.BuildOr(a, b).Render())
}

// TestBuildConditionTree verifies condition tree composition.
func TestBuildConditionTree(t *testing.T) {
	assert.Equal(t, "draft",
		ghexpr.BuildConditionTree("", "draft").Render(),
		"empty existing condition returns draft-only node")

	tree := ghexpr.BuildConditionTree("existing", "draft")
	assert.Equal(t, "(existing) && (draft)", tree.Render())
}

// TestBuildDisjunction verifies multi-term OR builder.
func TestBuildDisjunction(t *testing.T) {
	node := ghexpr.BuildDisjunction(false,
		ghexpr.BuildPropertyAccess("a"),
		ghexpr.BuildPropertyAccess("b"),
		ghexpr.BuildPropertyAccess("c"),
	)
	assert.Equal(t, "a || b || c", node.Render())
}

// TestRenderCondition verifies RenderCondition optimises and renders.
func TestRenderCondition(t *testing.T) {
	// A && true should optimise to just A.
	node := ghexpr.BuildAnd(
		ghexpr.BuildPropertyAccess("github.event_name"),
		ghexpr.BuildBooleanLiteral(true),
	)
	assert.Equal(t, "github.event_name", ghexpr.RenderCondition(node))
}
