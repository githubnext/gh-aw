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

	t.Run("keep chomping (|+) block with expression fails", func(t *testing.T) {
		var sb strings.Builder
		sb.WriteString("jobs:\n  test:\n    steps:\n      - name: Keep\n        run: |+\n")
		for range 50 {
			sb.WriteString("          echo " + strings.Repeat("c", 10) + "\n")
		}
		sb.WriteString("          echo ${{ github.actor }}\n")
		lines := strings.Split(sb.String(), "\n")
		err := validateBlockScalarExpressionSizes(lines, smallMax)
		require.Error(t, err, "keep-chomping block (|+) with expression exceeding limit should fail")
		assert.Contains(t, err.Error(), "exceeds maximum allowed size")
	})

	t.Run("indentation indicator block (|2) with expression fails", func(t *testing.T) {
		var sb strings.Builder
		sb.WriteString("jobs:\n  test:\n    steps:\n      - name: Indent\n        run: |2\n")
		for range 50 {
			sb.WriteString("          echo " + strings.Repeat("d", 10) + "\n")
		}
		sb.WriteString("          echo ${{ github.actor }}\n")
		lines := strings.Split(sb.String(), "\n")
		err := validateBlockScalarExpressionSizes(lines, smallMax)
		require.Error(t, err, "explicit-indent block (|2) with expression exceeding limit should fail")
		assert.Contains(t, err.Error(), "exceeds maximum allowed size")
	})

	t.Run("combined indicator block (|2-) with expression fails", func(t *testing.T) {
		var sb strings.Builder
		sb.WriteString("jobs:\n  test:\n    steps:\n      - name: Combined\n        run: |2-\n")
		for range 50 {
			sb.WriteString("          echo " + strings.Repeat("e", 10) + "\n")
		}
		sb.WriteString("          echo ${{ github.actor }}\n")
		lines := strings.Split(sb.String(), "\n")
		err := validateBlockScalarExpressionSizes(lines, smallMax)
		require.Error(t, err, "combined indicator block (|2-) with expression exceeding limit should fail")
		assert.Contains(t, err.Error(), "exceeds maximum allowed size")
	})
}

func TestValidateExpressionSizesSinglePass(t *testing.T) {
	const smallMax = 100 // tiny limit to keep tests readable

	t.Run("empty YAML passes", func(t *testing.T) {
		err := validateExpressionSizesSinglePass("", smallMax)
		assert.NoError(t, err, "empty content should pass")
	})

	t.Run("block scalar under limit without expression passes", func(t *testing.T) {
		yaml := `jobs:
  test:
    steps:
      - name: Small block
        run: |
          echo hello
          echo world
`
		err := validateExpressionSizesSinglePass(yaml, smallMax)
		assert.NoError(t, err, "small block without expression should pass")
	})

	t.Run("block scalar under limit with expression passes", func(t *testing.T) {
		yaml := `jobs:
  test:
    steps:
      - name: Small block with expression
        run: |
          echo ${{ github.actor }}
`
		err := validateExpressionSizesSinglePass(yaml, smallMax)
		assert.NoError(t, err, "small block with expression should pass (under limit)")
	})

	t.Run("large block scalar without expression passes", func(t *testing.T) {
		var sb strings.Builder
		sb.WriteString("jobs:\n  test:\n    steps:\n      - name: Large block\n        run: |\n")
		for range 50 {
			sb.WriteString("          echo " + strings.Repeat("x", 10) + "\n")
		}
		err := validateExpressionSizesSinglePass(sb.String(), smallMax)
		assert.NoError(t, err, "large block without any expression should pass")
	})

	t.Run("large block scalar with expression fails", func(t *testing.T) {
		var sb strings.Builder
		sb.WriteString("jobs:\n  test:\n    steps:\n      - name: Large block with expression\n        run: |\n")
		for range 50 {
			sb.WriteString("          echo " + strings.Repeat("x", 10) + "\n")
		}
		sb.WriteString("          echo ${{ github.actor }}\n")
		err := validateExpressionSizesSinglePass(sb.String(), smallMax)
		require.Error(t, err, "large block with expression should fail")
		assert.Contains(t, err.Error(), "exceeds maximum allowed size")
		assert.Contains(t, err.Error(), "run")
	})

	t.Run("single line exceeding limit fails", func(t *testing.T) {
		// A single YAML line longer than maxSize should be caught by the per-line check.
		line := "        value: " + strings.Repeat("x", smallMax+1) + "\n"
		err := validateExpressionSizesSinglePass(line, smallMax)
		require.Error(t, err, "single line over limit should fail")
		assert.Contains(t, err.Error(), "exceeds maximum allowed")
	})

	t.Run("expression at beginning of large block fails", func(t *testing.T) {
		var sb strings.Builder
		sb.WriteString("jobs:\n  test:\n    steps:\n      - name: Block\n        run: |\n")
		sb.WriteString("          echo ${{ github.ref_name }}\n")
		for range 50 {
			sb.WriteString("          echo " + strings.Repeat("z", 10) + "\n")
		}
		err := validateExpressionSizesSinglePass(sb.String(), smallMax)
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
		err := validateExpressionSizesSinglePass(sb.String(), smallMax)
		assert.NoError(t, err, "only large-with-expression blocks should fail")
	})

	t.Run("folded block (>) with expression fails", func(t *testing.T) {
		var sb strings.Builder
		sb.WriteString("jobs:\n  test:\n    steps:\n      - name: Folded\n        run: >\n")
		for range 50 {
			sb.WriteString("          echo " + strings.Repeat("b", 10) + "\n")
		}
		sb.WriteString("          echo ${{ github.actor }}\n")
		err := validateExpressionSizesSinglePass(sb.String(), smallMax)
		require.Error(t, err, "folded block (>) with expression exceeding limit should fail")
		assert.Contains(t, err.Error(), "exceeds maximum allowed size")
	})

	t.Run("keep chomping (|+) with expression fails", func(t *testing.T) {
		var sb strings.Builder
		sb.WriteString("jobs:\n  test:\n    steps:\n      - name: Keep\n        run: |+\n")
		for range 50 {
			sb.WriteString("          echo " + strings.Repeat("c", 10) + "\n")
		}
		sb.WriteString("          echo ${{ github.actor }}\n")
		err := validateExpressionSizesSinglePass(sb.String(), smallMax)
		require.Error(t, err, "keep-chomping block (|+) with expression exceeding limit should fail")
		assert.Contains(t, err.Error(), "exceeds maximum allowed size")
	})

	t.Run("indentation indicator (|2) with expression fails", func(t *testing.T) {
		var sb strings.Builder
		sb.WriteString("jobs:\n  test:\n    steps:\n      - name: Indent\n        run: |2\n")
		for range 50 {
			sb.WriteString("          echo " + strings.Repeat("d", 10) + "\n")
		}
		sb.WriteString("          echo ${{ github.actor }}\n")
		err := validateExpressionSizesSinglePass(sb.String(), smallMax)
		require.Error(t, err, "explicit-indent block (|2) with expression exceeding limit should fail")
		assert.Contains(t, err.Error(), "exceeds maximum allowed size")
	})

	t.Run("combined indicator (|2-) with expression fails", func(t *testing.T) {
		var sb strings.Builder
		sb.WriteString("jobs:\n  test:\n    steps:\n      - name: Combined\n        run: |2-\n")
		for range 50 {
			sb.WriteString("          echo " + strings.Repeat("e", 10) + "\n")
		}
		sb.WriteString("          echo ${{ github.actor }}\n")
		err := validateExpressionSizesSinglePass(sb.String(), smallMax)
		require.Error(t, err, "combined indicator block (|2-) with expression exceeding limit should fail")
		assert.Contains(t, err.Error(), "exceeds maximum allowed size")
	})

	t.Run("block open at EOF is checked", func(t *testing.T) {
		var sb strings.Builder
		sb.WriteString("jobs:\n  test:\n    steps:\n      - name: EOF block\n        run: |\n")
		for range 50 {
			sb.WriteString("          echo " + strings.Repeat("f", 10) + "\n")
		}
		sb.WriteString("          echo ${{ github.actor }}")
		// No trailing newline — block is still open at EOF
		err := validateExpressionSizesSinglePass(sb.String(), smallMax)
		require.Error(t, err, "block still open at EOF with expression exceeding limit should fail")
		assert.Contains(t, err.Error(), "exceeds maximum allowed size")
	})
}
