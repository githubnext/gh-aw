//go:build !integration

package workflow

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateBlockScalarExpressionSizes(t *testing.T) {
	const smallMax = 100 // tiny limit to keep tests readable

	t.Run("empty YAML passes", func(t *testing.T) {
		err := validateBlockScalarExpressionSizes([]string{}, smallMax)
		assert.NoError(t, err, "empty lines should pass")
	})

	t.Run("block scalar under limit without expression passes", func(t *testing.T) {
		lines := strings.Split(`jobs:
  test:
    steps:
      - name: Small block
        run: |
          echo hello
          echo world
`, "\n")
		err := validateBlockScalarExpressionSizes(lines, smallMax)
		assert.NoError(t, err, "small block without expression should pass")
	})

	t.Run("block scalar under limit with expression passes", func(t *testing.T) {
		lines := strings.Split(`jobs:
  test:
    steps:
      - name: Small block with expression
        run: |
          echo ${{ github.actor }}
`, "\n")
		err := validateBlockScalarExpressionSizes(lines, smallMax)
		assert.NoError(t, err, "small block with expression should pass (under limit)")
	})

	t.Run("large block scalar without expression passes", func(t *testing.T) {
		var sb strings.Builder
		sb.WriteString("jobs:\n  test:\n    steps:\n      - name: Large block\n        run: |\n")
		for range 50 {
			sb.WriteString("          echo " + strings.Repeat("x", 10) + "\n")
		}
		lines := strings.Split(sb.String(), "\n")
		err := validateBlockScalarExpressionSizes(lines, smallMax)
		assert.NoError(t, err, "large block without any expression should pass")
	})

	t.Run("large block scalar with expression fails", func(t *testing.T) {
		var sb strings.Builder
		sb.WriteString("jobs:\n  test:\n    steps:\n      - name: Large block with expression\n        run: |\n")
		// Fill content to exceed smallMax
		for range 50 {
			sb.WriteString("          echo " + strings.Repeat("x", 10) + "\n")
		}
		// Add a template expression
		sb.WriteString("          echo ${{ github.actor }}\n")
		lines := strings.Split(sb.String(), "\n")
		err := validateBlockScalarExpressionSizes(lines, smallMax)
		require.Error(t, err, "large block with expression should fail")
		assert.Contains(t, err.Error(), "exceeds maximum allowed size", "error should describe the size issue")
		assert.Contains(t, err.Error(), "run", "error should identify the block key")
	})

	t.Run("expression at beginning of large block fails", func(t *testing.T) {
		var sb strings.Builder
		sb.WriteString("jobs:\n  test:\n    steps:\n      - name: Block\n        run: |\n")
		sb.WriteString("          echo ${{ github.ref_name }}\n")
		// Fill to exceed limit after the expression
		for range 50 {
			sb.WriteString("          echo " + strings.Repeat("z", 10) + "\n")
		}
		lines := strings.Split(sb.String(), "\n")
		err := validateBlockScalarExpressionSizes(lines, smallMax)
		require.Error(t, err, "block with expression at start should fail when total exceeds limit")
	})

	t.Run("multiple blocks: only large expression block fails", func(t *testing.T) {
		var sb strings.Builder
		// First block: small with expression - should pass
		sb.WriteString("jobs:\n  test:\n    steps:\n      - name: Small\n        run: |\n")
		sb.WriteString("          echo ${{ github.actor }}\n")
		// Second block: large without expression - should pass
		sb.WriteString("      - name: Large no expr\n        run: |\n")
		for range 50 {
			sb.WriteString("          echo " + strings.Repeat("a", 10) + "\n")
		}
		lines := strings.Split(sb.String(), "\n")
		err := validateBlockScalarExpressionSizes(lines, smallMax)
		assert.NoError(t, err, "only large-with-expression blocks should fail; small-with-expression and large-without-expression are both fine")
	})

	t.Run("folded block scalar (>) with expression also checked", func(t *testing.T) {
		var sb strings.Builder
		sb.WriteString("jobs:\n  test:\n    steps:\n      - name: Folded\n        run: >\n")
		for range 50 {
			sb.WriteString("          echo " + strings.Repeat("b", 10) + "\n")
		}
		sb.WriteString("          echo ${{ github.actor }}\n")
		lines := strings.Split(sb.String(), "\n")
		err := validateBlockScalarExpressionSizes(lines, smallMax)
		require.Error(t, err, "folded block (>) with expression exceeding limit should fail")
		assert.Contains(t, err.Error(), "exceeds maximum allowed size", "error message should describe the issue")
	})

	t.Run("MaxExpressionSize used by compiler validation", func(t *testing.T) {
		// Ensure the constant is exactly 21000 as documented
		assert.Equal(t, 21000, MaxExpressionSize, "MaxExpressionSize must be 21000 to match GitHub Actions limit")
	})
}
