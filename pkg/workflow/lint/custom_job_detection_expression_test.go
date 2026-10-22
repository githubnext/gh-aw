package lint

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCustomJobDetectionConditionExpressionContract(t *testing.T) {
	for _, condition := range []string{
		"always() && needs.detection.result == 'success'",
		"${{always() && needs.detection.result == 'success'}}",
		" \n${{ \n always() && needs.detection.result == 'success' \t }} \t",
		"contains('${{ text }}', '}}')", "${{ contains('${{ text }}', '}}') }}",
	} {
		t.Run(condition, func(t *testing.T) {
			safe, err := customJobDetectionCondition(condition)
			require.NoError(t, err)
			assert.True(t, safe)
		})
	}
	for _, condition := range []string{
		"${{ }}", "${{}}", "${{ always()", "always() }}",
		"${{ always() }} trailing", "${{ false }}${{ true }}",
		"${{ always() }} && needs.detection.result == 'success' }}",
	} {
		t.Run(condition, func(t *testing.T) {
			_, err := customJobDetectionCondition(condition)
			require.Error(t, err)
		})
	}
}
