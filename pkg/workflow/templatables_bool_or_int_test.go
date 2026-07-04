package workflow

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTemplatableBoolOrIntUnmarshalJSONRejectsFloat(t *testing.T) {
	var value TemplatableBoolOrInt
	err := json.Unmarshal([]byte("1.5"), &value)
	require.Error(t, err, "float input should be rejected")
}
