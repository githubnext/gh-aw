package parser

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCustomWebhookSchemaRejected(t *testing.T) {
	for _, custom := range []any{
		nil, []any{}, map[string]any{"type": "example.event"},
		[]any{map[string]any{"type": "example.event"}},
		[]any{map[string]any{"type": "example.first"}, map[string]any{"type": "example.second"}},
	} {
		for _, mixed := range []bool{false, true} {
			gateway := map[string]any{"webhooks": custom}
			if mixed {
				gateway["dependabot-alerts"] = map[string]any{}
			}
			err := ValidateMainWorkflowFrontmatterWithSchemaAndLocation(map[string]any{"on": map[string]any{"gateway-dispatch": gateway}}, "custom.md")
			require.Error(t, err, "custom webhooks must reject even alongside supported triggers")
		}
	}
}
